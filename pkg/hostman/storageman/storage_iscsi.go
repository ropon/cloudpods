// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storageman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SIscsiStorageConf struct {
	Target   string `json:"target"`
	Iqn      string `json:"iqn"`
	Portal   string `json:"portal"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	LunId    int    `json:"lun_id"`
}

type SIscsiStorage struct {
	SBaseStorage
	SIscsiStorageConf

	// iSCSI connection management
	devicePath   string
	isConnected  bool
	connectionMu sync.RWMutex
}

func NewIscsiStorage(manager *SStorageManager, path string) *SIscsiStorage {
	ret := &SIscsiStorage{
		SBaseStorage: *NewBaseStorage(manager, path),
	}
	return ret
}

type SIscsiStorageFactory struct{}

func (factory *SIscsiStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewIscsiStorage(manager, mountPoint)
}

func (factory *SIscsiStorageFactory) StorageType() string {
	return api.STORAGE_ISCSI
}

func init() {
	registerStorageFactory(&SIscsiStorageFactory{})
}

func (s *SIscsiStorage) StorageType() string {
	return api.STORAGE_ISCSI
}

func (s *SIscsiStorage) IsLocal() bool {
	return false
}

func (s *SIscsiStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if conf != nil {
		if dconf, ok := conf.(*jsonutils.JSONDict); ok {
			s.StorageConf = dconf
		}
		if err := conf.Unmarshal(&s.SIscsiStorageConf); err != nil {
			return errors.Wrapf(err, "unmarshal iSCSI storage config")
		}
	}
	return nil
}

// MountStorage connects to the iSCSI target and discovers the device
func (s *SIscsiStorage) MountStorage() error {
	s.connectionMu.Lock()
	defer s.connectionMu.Unlock()

	if s.isConnected {
		log.Infof("iSCSI storage %s already connected", s.StorageName)
		return nil
	}

	log.Infof("Mounting iSCSI storage %s (target: %s, iqn: %s, portal: %s)",
		s.StorageName, s.Target, s.Iqn, s.Portal)

	// Step 1: Discover iSCSI targets
	if err := s.discoverTarget(); err != nil {
		return errors.Wrapf(err, "discover iSCSI target")
	}

	// Step 2: Login to iSCSI target
	if err := s.loginTarget(); err != nil {
		return errors.Wrapf(err, "login to iSCSI target")
	}

	// Step 3: Wait for device to appear and get device path
	devicePath, err := s.waitForDevice()
	if err != nil {
		// Cleanup on failure
		s.logoutTarget()
		return errors.Wrapf(err, "wait for iSCSI device")
	}

	s.devicePath = devicePath
	s.isConnected = true

	log.Infof("Successfully mounted iSCSI storage %s at device %s", s.StorageName, s.devicePath)
	return nil
}

// UnmountStorage disconnects from the iSCSI target and cleans up resources
func (s *SIscsiStorage) UnmountStorage() error {
	s.connectionMu.Lock()
	defer s.connectionMu.Unlock()

	if !s.isConnected {
		log.Infof("iSCSI storage %s already disconnected", s.StorageName)
		return nil
	}

	log.Infof("Unmounting iSCSI storage %s", s.StorageName)

	// Step 1: Logout from iSCSI target
	if err := s.logoutTarget(); err != nil {
		log.Errorf("Failed to logout from iSCSI target: %v", err)
		// Continue with cleanup even if logout fails
	}

	// Step 2: Clean up discovery records
	if err := s.cleanupDiscovery(); err != nil {
		log.Errorf("Failed to cleanup iSCSI discovery: %v", err)
	}

	s.devicePath = ""
	s.isConnected = false

	log.Infof("Successfully unmounted iSCSI storage %s", s.StorageName)
	return nil
}

// discoverTarget discovers the iSCSI target using iscsiadm
func (s *SIscsiStorage) discoverTarget() error {
	args := []string{
		"-m", "discovery",
		"-t", "sendtargets",
		"-p", s.Portal,
	}

	// Add authentication if provided
	if s.Username != "" && s.Password != "" {
		args = append(args, "--username", s.Username, "--password", s.Password)
	}

	cmd := procutils.NewCommand("iscsiadm", args...)
	output, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "iscsiadm discovery failed: %s", string(output))
	}

	log.Infof("iSCSI discovery output: %s", string(output))

	// Verify that our target IQN is in the discovery results
	if !strings.Contains(string(output), s.Iqn) {
		return errors.Errorf("target IQN %s not found in discovery results", s.Iqn)
	}

	return nil
}

// loginTarget logs into the iSCSI target
func (s *SIscsiStorage) loginTarget() error {
	// Set authentication parameters if provided
	if s.Username != "" && s.Password != "" {
		if err := s.setAuthParams(); err != nil {
			return errors.Wrapf(err, "set authentication parameters")
		}
	}

	// Login to target
	args := []string{
		"-m", "node",
		"-T", s.Iqn,
		"-p", s.Portal,
		"--login",
	}

	cmd := procutils.NewCommand("iscsiadm", args...)
	output, err := cmd.Output()
	if err != nil {
		// Check if already logged in
		if strings.Contains(string(output), "already exists") {
			log.Infof("Already logged into iSCSI target %s", s.Iqn)
			return nil
		}
		return errors.Wrapf(err, "iscsiadm login failed: %s", string(output))
	}

	log.Infof("Successfully logged into iSCSI target %s", s.Iqn)
	return nil
}

// logoutTarget logs out from the iSCSI target
func (s *SIscsiStorage) logoutTarget() error {
	args := []string{
		"-m", "node",
		"-T", s.Iqn,
		"-p", s.Portal,
		"--logout",
	}

	cmd := procutils.NewCommand("iscsiadm", args...)
	output, err := cmd.Output()
	if err != nil {
		// Check if already logged out
		if strings.Contains(string(output), "not found") {
			log.Infof("Already logged out from iSCSI target %s", s.Iqn)
			return nil
		}
		return errors.Wrapf(err, "iscsiadm logout failed: %s", string(output))
	}

	log.Infof("Successfully logged out from iSCSI target %s", s.Iqn)
	return nil
}

// setAuthParams sets authentication parameters for the iSCSI session
func (s *SIscsiStorage) setAuthParams() error {
	// Set authentication method
	if err := s.setNodeParam("node.session.auth.authmethod", "CHAP"); err != nil {
		return err
	}

	// Set username
	if err := s.setNodeParam("node.session.auth.username", s.Username); err != nil {
		return err
	}

	// Set password
	if err := s.setNodeParam("node.session.auth.password", s.Password); err != nil {
		return err
	}

	return nil
}

// setNodeParam sets a node parameter using iscsiadm
func (s *SIscsiStorage) setNodeParam(param, value string) error {
	args := []string{
		"-m", "node",
		"-T", s.Iqn,
		"-p", s.Portal,
		"-o", "update",
		"-n", param,
		"-v", value,
	}

	cmd := procutils.NewCommand("iscsiadm", args...)
	output, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "failed to set node parameter %s: %s", param, string(output))
	}

	return nil
}

// waitForDevice waits for the iSCSI device to appear and returns its path
func (s *SIscsiStorage) waitForDevice() (string, error) {
	// Wait up to 30 seconds for device to appear
	timeout := 30 * time.Second
	interval := 1 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		devicePath, err := s.findDevicePath()
		if err == nil && devicePath != "" {
			return devicePath, nil
		}

		time.Sleep(interval)
	}

	return "", errors.Errorf("timeout waiting for iSCSI device to appear")
}

// findDevicePath finds the device path for the iSCSI LUN
func (s *SIscsiStorage) findDevicePath() (string, error) {
	// Look for device in /dev/disk/by-path/ that matches our iSCSI target
	byPathDir := "/dev/disk/by-path"
	entries, err := os.ReadDir(byPathDir)
	if err != nil {
		return "", errors.Wrapf(err, "read %s", byPathDir)
	}

	// Pattern to match iSCSI device paths
	// Example: ip-192.168.1.100:3260-iscsi-iqn.2023-01.com.example:target01-lun-0
	targetIP := strings.Split(s.Portal, ":")[0]
	pattern := fmt.Sprintf("ip-%s.*-iscsi-%s-lun-%d", regexp.QuoteMeta(targetIP), regexp.QuoteMeta(s.Iqn), s.LunId)
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return "", errors.Wrapf(err, "compile device path pattern")
	}

	for _, entry := range entries {
		if regex.MatchString(entry.Name()) {
			devicePath := filepath.Join(byPathDir, entry.Name())
			// Resolve symlink to get actual device path
			realPath, err := filepath.EvalSymlinks(devicePath)
			if err != nil {
				log.Warningf("Failed to resolve symlink %s: %v", devicePath, err)
				continue
			}
			return realPath, nil
		}
	}

	return "", errors.Errorf("device not found for target %s lun %d", s.Iqn, s.LunId)
}

// cleanupDiscovery removes discovery records for the target
func (s *SIscsiStorage) cleanupDiscovery() error {
	args := []string{
		"-m", "node",
		"-T", s.Iqn,
		"-p", s.Portal,
		"-o", "delete",
	}

	cmd := procutils.NewCommand("iscsiadm", args...)
	output, err := cmd.Output()
	if err != nil {
		// Ignore errors if node doesn't exist
		if strings.Contains(string(output), "not found") {
			return nil
		}
		return errors.Wrapf(err, "cleanup discovery failed: %s", string(output))
	}

	return nil
}

// GetDevicePath returns the current device path
func (s *SIscsiStorage) GetDevicePath() string {
	s.connectionMu.RLock()
	defer s.connectionMu.RUnlock()
	return s.devicePath
}

// IsConnected returns whether the iSCSI storage is currently connected
func (s *SIscsiStorage) IsConnected() bool {
	s.connectionMu.RLock()
	defer s.connectionMu.RUnlock()
	return s.isConnected
}

// Accessible checks if the iSCSI storage is accessible
func (s *SIscsiStorage) Accessible() error {
	if !s.IsConnected() {
		if err := s.MountStorage(); err != nil {
			return errors.Wrapf(err, "mount iSCSI storage")
		}
	}

	// Check if device path exists and is accessible
	devicePath := s.GetDevicePath()
	if devicePath == "" {
		return errors.Errorf("no device path available")
	}

	if _, err := os.Stat(devicePath); err != nil {
		return errors.Wrapf(err, "device %s not accessible", devicePath)
	}

	return nil
}

// Detach disconnects from the iSCSI storage
func (s *SIscsiStorage) Detach() error {
	return s.UnmountStorage()
}

// SyncStorageInfo synchronizes storage information with the management system
func (s *SIscsiStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := map[string]interface{}{
		"name":   s.StorageName,
		"status": api.STORAGE_ONLINE,
		"zone":   s.GetZoneId(),
	}

	if len(s.StorageId) > 0 {
		return modules.Storages.Put(
			hostutils.GetComputeSession(context.Background()),
			s.StorageId,
			jsonutils.Marshal(content),
		)
	}

	return modules.Storages.Get(
		hostutils.GetComputeSession(context.Background()),
		s.StorageName,
		jsonutils.Marshal(content),
	)
}

// getDeviceSize gets the size of a block device in MB
func (s *SIscsiStorage) getDeviceSize(devicePath string) (int64, error) {
	cmd := procutils.NewCommand("blockdev", "--getsize64", devicePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, errors.Wrapf(err, "get device size for %s", devicePath)
	}

	sizeBytes, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "parse device size")
	}

	return sizeBytes / 1024 / 1024, nil
}

// GetDiskById finds a disk by ID
func (s *SIscsiStorage) GetDiskById(diskId string) (IDisk, error) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()

	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			if err := s.Disks[i].Probe(); err != nil {
				return nil, errors.Wrapf(err, "probe disk %s", diskId)
			}
			return s.Disks[i], nil
		}
	}

	// Create new disk if not found
	disk := s.CreateDisk(diskId)
	if disk.Probe() == nil {
		return disk, nil
	}

	return nil, errors.ErrNotFound
}

// CreateDisk creates a new disk instance
func (s *SIscsiStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()

	// For iSCSI, we create a local disk that points to the iSCSI device
	disk := NewLocalDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

// RemoveDisk removes a disk from the storage
func (s *SIscsiStorage) RemoveDisk(disk IDisk) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()

	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == disk.GetId() {
			s.Disks = append(s.Disks[:i], s.Disks[i+1:]...)
			break
		}
	}
}

// GetDisksPath returns paths of all disks in the storage
func (s *SIscsiStorage) GetDisksPath() ([]string, error) {
	// For iSCSI storage, disks are typically managed differently
	// Return empty slice as iSCSI doesn't have traditional disk files
	return []string{}, nil
}

// Placeholder implementations for required interface methods
func (s *SIscsiStorage) GetSnapshotDir() string {
	return ""
}

func (s *SIscsiStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return ""
}

func (s *SIscsiStorage) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) IsSnapshotExist(diskId, snapshotId string) (bool, error) {
	return false, errors.ErrNotSupported
}

func (s *SIscsiStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return errors.ErrNotSupported
}

func (s *SIscsiStorage) CreateDiskFromSnapshot(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) CreateDiskFromExistingPath(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) error {
	return errors.ErrNotSupported
}

func (s *SIscsiStorage) CleanRecycleDiskfiles(ctx context.Context) {
	log.Infof("iSCSI storage %s: CleanRecycleDiskfiles - no action needed", s.StorageName)
}

// Additional required interface methods

func (s *SIscsiStorage) Lvmlockd() bool {
	return false
}

func (s *SIscsiStorage) GetBackupDir() string {
	return ""
}

func (s *SIscsiStorage) StorageBackup(ctx context.Context, params *SStorageBackup) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) StorageBackupRecovery(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) GetFreeSizeMb() int {
	// For iSCSI, we don't track free space at storage level
	return -1
}

func (s *SIscsiStorage) GetCapacityMb() int {
	// For iSCSI, we don't track capacity at storage level
	return -1
}

func (s *SIscsiStorage) DeleteDiskfile(diskPath string, skipRecycle bool) error {
	return errors.ErrNotSupported
}

func (s *SIscsiStorage) GetFuseTmpPath() string {
	return ""
}

func (s *SIscsiStorage) GetFuseMountPath() string {
	return ""
}

func (s *SIscsiStorage) GetImgsaveBackupPath() string {
	return ""
}

func (s *SIscsiStorage) CreateDiskByDiskinfo(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return s.SBaseStorage.CreateDiskByDiskinfo(ctx, params)
}

func (s *SIscsiStorage) SaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) CreateDiskFromBackup(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) error {
	return errors.ErrNotSupported
}

func (s *SIscsiStorage) DiskMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) GetCloneTargetDiskPath(ctx context.Context, targetDiskId string) string {
	return ""
}

func (s *SIscsiStorage) CloneDiskFromStorage(ctx context.Context, srcStorage IStorage, srcDisk IDisk, targetDiskId string, fullCopy bool, encInfo apis.SEncryptInfo) (*hostapi.ServerCloneDiskFromStorageResponse, error) {
	return nil, errors.ErrNotSupported
}

func (s *SIscsiStorage) DestinationPrepareMigrate(ctx context.Context, liveMigrate bool, disksUri string, snapshotsUri string, disksBackingFile, diskSnapsChain, outChainSnaps jsonutils.JSONObject, rebaseDisks bool, diskDesc *desc.SGuestDisk, serverId string, idx, totalDiskCount int, encInfo *apis.SEncryptInfo, sysDiskHasTemplate bool) error {
	return errors.ErrNotSupported
}
