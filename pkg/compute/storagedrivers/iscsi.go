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

package storagedrivers

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SIscsiStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SIscsiStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (self *SIscsiStorageDriver) GetStorageType() string {
	return api.STORAGE_ISCSI
}

func (self *SIscsiStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.StorageCreateInput) error {
	input.StorageConf = jsonutils.NewDict()

	// Validate required iSCSI Target address
	if len(input.IscsiTarget) == 0 {
		return httperrors.NewMissingParameterError("iscsi_target")
	}
	if err := self.validateTargetAddress(input.IscsiTarget); err != nil {
		return httperrors.NewInputParameterError("invalid iscsi_target: %v", err)
	}

	// Validate required iSCSI IQN
	if len(input.IscsiIqn) == 0 {
		return httperrors.NewMissingParameterError("iscsi_iqn")
	}
	if err := self.validateIQN(input.IscsiIqn); err != nil {
		return httperrors.NewInputParameterError("invalid iscsi_iqn: %v", err)
	}

	// Validate required iSCSI Portal address
	if len(input.IscsiPortal) == 0 {
		return httperrors.NewMissingParameterError("iscsi_portal")
	}
	if err := self.validatePortalAddress(input.IscsiPortal); err != nil {
		return httperrors.NewInputParameterError("invalid iscsi_portal: %v", err)
	}

	// Validate authentication parameters if provided
	if err := self.validateAuthParams(input.IscsiUsername, input.IscsiPassword); err != nil {
		return httperrors.NewInputParameterError("invalid authentication parameters: %v", err)
	}

	// Set default LUN ID if not provided
	if input.IscsiLunId < 0 {
		input.IscsiLunId = 0
	}

	// Check for duplicate iSCSI storage configuration
	if err := self.checkDuplicateStorage(input); err != nil {
		return err
	}

	// Test iSCSI connection availability
	if err := self.testIscsiConnection(input); err != nil {
		return httperrors.NewBadRequestError("iSCSI connection test failed: %v", err)
	}

	// Store iSCSI configuration
	iscsiConf := api.IscsiStorageConf{
		Target:   strings.TrimSpace(input.IscsiTarget),
		Iqn:      strings.TrimSpace(input.IscsiIqn),
		Portal:   strings.TrimSpace(input.IscsiPortal),
		Username: strings.TrimSpace(input.IscsiUsername),
		Password: strings.TrimSpace(input.IscsiPassword),
		LunId:    input.IscsiLunId,
	}

	input.StorageConf.Update(jsonutils.Marshal(iscsiConf))

	return nil
}

func (self *SIscsiStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
	// Check if storage already has a cache assigned
	if len(storage.StoragecacheId) > 0 {
		// Update storage status to online
		_, err := db.Update(storage, func() error {
			storage.Status = api.STORAGE_ONLINE
			return nil
		})
		if err != nil {
			log.Errorf("update storage %s status to online error: %v", storage.Name, err)
		}
		return
	}

	// Check if there's an existing storage cache for the same iSCSI configuration
	storages := []models.SStorage{}
	q := models.StorageManager.Query().Equals("storage_type", api.STORAGE_ISCSI)
	if err := db.FetchModelObjects(models.StorageManager, q, &storages); err != nil {
		log.Errorf("fetch iSCSI storages error: %v", err)
		return
	}

	// Get current storage configuration
	currentTarget, _ := storage.StorageConf.GetString("target")
	currentIqn, _ := storage.StorageConf.GetString("iqn")
	currentPortal, _ := storage.StorageConf.GetString("portal")

	// Look for existing storage with same configuration to share cache
	for i := 0; i < len(storages); i++ {
		if storages[i].Id == storage.Id {
			continue // Skip current storage
		}

		existingTarget, _ := storages[i].StorageConf.GetString("target")
		existingIqn, _ := storages[i].StorageConf.GetString("iqn")
		existingPortal, _ := storages[i].StorageConf.GetString("portal")

		// If same target, IQN, and portal, share the storage cache
		if currentTarget == existingTarget &&
			currentIqn == existingIqn &&
			currentPortal == existingPortal &&
			len(storages[i].StoragecacheId) > 0 {

			_, err := db.Update(storage, func() error {
				storage.StoragecacheId = storages[i].StoragecacheId
				storage.Status = api.STORAGE_ONLINE
				return nil
			})
			if err != nil {
				log.Errorf("update storagecache_id for storage %s error: %v", storage.Name, err)
			}
			return
		}
	}

	// Create new storage cache if no existing one found
	sc := &models.SStoragecache{}
	sc.SetModelManager(models.StoragecacheManager, sc)
	sc.Name = fmt.Sprintf("iscsi-cache-%s", storage.Id)
	sc.Path = options.Options.DefaultImageCacheDir
	sc.ExternalId = storage.Id

	if err := models.StoragecacheManager.TableSpec().Insert(ctx, sc); err != nil {
		log.Errorf("insert storagecache for storage %s error: %v", storage.Name, err)
		return
	}

	// Update storage with cache ID and set status to online
	_, err := db.Update(storage, func() error {
		storage.StoragecacheId = sc.Id
		storage.Status = api.STORAGE_ONLINE
		return nil
	})
	if err != nil {
		log.Errorf("update storagecache info for storage %s error: %v", storage.Name, err)
	}

	log.Infof("Successfully created storage cache %s for iSCSI storage %s", sc.Id, storage.Name)
}

// validateIQN validates iSCSI Qualified Name according to RFC 3720
func (self *SIscsiStorageDriver) validateIQN(iqn string) error {
	if len(iqn) == 0 {
		return fmt.Errorf("IQN cannot be empty")
	}

	// IQN format: iqn.yyyy-mm.naming-authority:unique-name
	// Example: iqn.2023-01.com.example:storage.target01
	iqnPattern := `^iqn\.\d{4}-\d{2}\.([a-zA-Z0-9\-\.]+):([a-zA-Z0-9\-\._:]+)$`
	matched, err := regexp.MatchString(iqnPattern, iqn)
	if err != nil {
		return fmt.Errorf("failed to validate IQN pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("IQN format must be 'iqn.yyyy-mm.naming-authority:unique-name'")
	}

	// Additional length check (RFC 3720 specifies max 223 characters)
	if len(iqn) > 223 {
		return fmt.Errorf("IQN length cannot exceed 223 characters")
	}

	return nil
}

// validateTargetAddress validates iSCSI target IP address
func (self *SIscsiStorageDriver) validateTargetAddress(target string) error {
	if len(target) == 0 {
		return fmt.Errorf("target address cannot be empty")
	}

	// Parse IP address
	ip := net.ParseIP(target)
	if ip == nil {
		return fmt.Errorf("invalid IP address format")
	}

	return nil
}

// validatePortalAddress validates iSCSI portal address (IP:port format)
func (self *SIscsiStorageDriver) validatePortalAddress(portal string) error {
	if len(portal) == 0 {
		return fmt.Errorf("portal address cannot be empty")
	}

	// Split host and port
	host, portStr, err := net.SplitHostPort(portal)
	if err != nil {
		return fmt.Errorf("invalid portal format, expected 'IP:port': %v", err)
	}

	// Validate IP address
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("invalid IP address in portal")
	}

	// Validate port number
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %v", err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number must be between 1 and 65535")
	}

	return nil
}

// validateAuthParams validates iSCSI authentication parameters
func (self *SIscsiStorageDriver) validateAuthParams(username, password string) error {
	// If username is provided, password must also be provided
	if len(username) > 0 && len(password) == 0 {
		return fmt.Errorf("password is required when username is provided")
	}

	// If password is provided, username must also be provided
	if len(password) > 0 && len(username) == 0 {
		return fmt.Errorf("username is required when password is provided")
	}

	// Validate username format (basic validation)
	if len(username) > 0 {
		if len(username) > 255 {
			return fmt.Errorf("username length cannot exceed 255 characters")
		}
		// Username should not contain special characters that might cause issues
		if strings.ContainsAny(username, " \t\n\r") {
			return fmt.Errorf("username cannot contain whitespace characters")
		}
	}

	// Validate password format (basic validation)
	if len(password) > 0 {
		if len(password) > 255 {
			return fmt.Errorf("password length cannot exceed 255 characters")
		}
	}

	return nil
}

// checkDuplicateStorage checks if an iSCSI storage with the same configuration already exists
func (self *SIscsiStorageDriver) checkDuplicateStorage(input *api.StorageCreateInput) error {
	storages := []models.SStorage{}
	q := models.StorageManager.Query().Equals("storage_type", api.STORAGE_ISCSI)
	err := db.FetchModelObjects(models.StorageManager, q, &storages)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	for i := 0; i < len(storages); i++ {
		target, _ := storages[i].StorageConf.GetString("target")
		iqn, _ := storages[i].StorageConf.GetString("iqn")
		portal, _ := storages[i].StorageConf.GetString("portal")
		lunId, _ := storages[i].StorageConf.Int("lun_id")

		// Check if the same target, IQN, portal, and LUN ID combination exists
		if input.IscsiTarget == target &&
			input.IscsiIqn == iqn &&
			input.IscsiPortal == portal &&
			int64(input.IscsiLunId) == lunId {
			return httperrors.NewDuplicateResourceError("iSCSI storage with target=%s, iqn=%s, portal=%s, lun_id=%d already exists",
				target, iqn, portal, lunId)
		}
	}

	return nil
}

// testIscsiConnection tests the availability of iSCSI connection
func (self *SIscsiStorageDriver) testIscsiConnection(input *api.StorageCreateInput) error {
	// Extract host and port from portal
	host, portStr, err := net.SplitHostPort(input.IscsiPortal)
	if err != nil {
		return fmt.Errorf("failed to parse portal address: %v", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port in portal: %v", err)
	}

	// Test TCP connection to the iSCSI portal
	timeout := 10 * time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to iSCSI portal %s: %v", input.IscsiPortal, err)
	}
	defer conn.Close()

	log.Infof("Successfully connected to iSCSI portal %s", input.IscsiPortal)
	return nil
}

// testIscsiConnectionUpdate tests the availability of iSCSI connection with updated configuration
func (self *SIscsiStorageDriver) testIscsiConnectionUpdate(input api.StorageUpdateInput) error {
	// Get existing configuration from StorageConf
	target, _ := input.StorageConf.GetString("target")
	iqn, _ := input.StorageConf.GetString("iqn")
	portal, _ := input.StorageConf.GetString("portal")

	if len(target) == 0 || len(iqn) == 0 || len(portal) == 0 {
		return fmt.Errorf("missing required iSCSI configuration parameters")
	}

	// Extract host and port from portal
	host, portStr, err := net.SplitHostPort(portal)
	if err != nil {
		return fmt.Errorf("failed to parse portal address: %v", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port in portal: %v", err)
	}

	// Test TCP connection to the iSCSI portal
	timeout := 10 * time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to iSCSI portal %s: %v", portal, err)
	}
	defer conn.Close()

	log.Infof("Successfully connected to iSCSI portal %s with updated configuration", portal)
	return nil
}

func (self *SIscsiStorageDriver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, input api.StorageUpdateInput) (api.StorageUpdateInput, error) {
	// Validate authentication parameters if provided for update
	if len(input.IscsiUsername) > 0 || len(input.IscsiPassword) > 0 {
		if err := self.validateAuthParams(input.IscsiUsername, input.IscsiPassword); err != nil {
			return input, httperrors.NewInputParameterError("invalid authentication parameters: %v", err)
		}

		// Update authentication information in storage configuration
		if len(input.IscsiUsername) > 0 {
			input.StorageConf.Set("username", jsonutils.NewString(strings.TrimSpace(input.IscsiUsername)))
			input.UpdateStorageConf = true
		}
		if len(input.IscsiPassword) > 0 {
			input.StorageConf.Set("password", jsonutils.NewString(strings.TrimSpace(input.IscsiPassword)))
			input.UpdateStorageConf = true
		}

		// Test iSCSI connection with updated authentication parameters after configuration change
		if err := self.testIscsiConnectionUpdate(input); err != nil {
			return input, httperrors.NewBadRequestError("iSCSI connection test failed with updated configuration: %v", err)
		}
	}

	return self.SBaseStorageDriver.ValidateUpdateData(ctx, userCred, input)
}
