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
	"testing"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestIscsiStorageFactory(t *testing.T) {
	factory := &SIscsiStorageFactory{}

	// Test storage type
	if factory.StorageType() != api.STORAGE_ISCSI {
		t.Errorf("Expected storage type %s, got %s", api.STORAGE_ISCSI, factory.StorageType())
	}

	// Test creating storage instance
	manager := &SStorageManager{}
	storage := factory.NewStorage(manager, "/test/path")

	if storage == nil {
		t.Error("Expected non-nil storage instance")
	}

	if storage.StorageType() != api.STORAGE_ISCSI {
		t.Errorf("Expected storage type %s, got %s", api.STORAGE_ISCSI, storage.StorageType())
	}
}

func TestIscsiStorageBasicProperties(t *testing.T) {
	manager := &SStorageManager{}
	storage := NewIscsiStorage(manager, "/test/path")

	// Test basic properties
	if storage.IsLocal() {
		t.Error("iSCSI storage should not be local")
	}

	if storage.Lvmlockd() {
		t.Error("iSCSI storage should not use lvmlockd")
	}

	if storage.StorageType() != api.STORAGE_ISCSI {
		t.Errorf("Expected storage type %s, got %s", api.STORAGE_ISCSI, storage.StorageType())
	}
}

func TestIscsiStorageSetStorageInfo(t *testing.T) {
	manager := &SStorageManager{}
	storage := NewIscsiStorage(manager, "/test/path")

	// Test configuration
	conf := jsonutils.NewDict()
	conf.Set("target", jsonutils.NewString("192.168.1.100"))
	conf.Set("iqn", jsonutils.NewString("iqn.2023-01.com.example:target01"))
	conf.Set("portal", jsonutils.NewString("192.168.1.100:3260"))
	conf.Set("username", jsonutils.NewString("testuser"))
	conf.Set("password", jsonutils.NewString("testpass"))
	conf.Set("lun_id", jsonutils.NewInt(0))

	err := storage.SetStorageInfo("test-storage-id", "test-storage", conf)
	if err != nil {
		t.Errorf("SetStorageInfo failed: %v", err)
	}

	// Verify configuration was set
	if storage.StorageId != "test-storage-id" {
		t.Errorf("Expected storage ID 'test-storage-id', got '%s'", storage.StorageId)
	}

	if storage.StorageName != "test-storage" {
		t.Errorf("Expected storage name 'test-storage', got '%s'", storage.StorageName)
	}

	if storage.Target != "192.168.1.100" {
		t.Errorf("Expected target '192.168.1.100', got '%s'", storage.Target)
	}

	if storage.Iqn != "iqn.2023-01.com.example:target01" {
		t.Errorf("Expected IQN 'iqn.2023-01.com.example:target01', got '%s'", storage.Iqn)
	}

	if storage.Portal != "192.168.1.100:3260" {
		t.Errorf("Expected portal '192.168.1.100:3260', got '%s'", storage.Portal)
	}

	if storage.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", storage.Username)
	}

	if storage.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", storage.Password)
	}

	if storage.LunId != 0 {
		t.Errorf("Expected LUN ID 0, got %d", storage.LunId)
	}
}

func TestIscsiStorageConnectionState(t *testing.T) {
	manager := &SStorageManager{}
	storage := NewIscsiStorage(manager, "/test/path")

	// Initially should not be connected
	if storage.IsConnected() {
		t.Error("Storage should not be connected initially")
	}

	if storage.GetDevicePath() != "" {
		t.Error("Device path should be empty initially")
	}
}

func TestIscsiStorageUnsupportedOperations(t *testing.T) {
	manager := &SStorageManager{}
	storage := NewIscsiStorage(manager, "/test/path")

	// Test unsupported snapshot operations
	if exists, err := storage.IsSnapshotExist("disk1", "snap1"); exists || err == nil {
		t.Error("Snapshot operations should not be supported")
	}

	// Test unsupported backup operations
	if storage.GetBackupDir() != "" {
		t.Error("Backup directory should be empty for iSCSI storage")
	}

	// Test unsupported paths
	if storage.GetSnapshotDir() != "" {
		t.Error("Snapshot directory should be empty for iSCSI storage")
	}

	if storage.GetFuseTmpPath() != "" {
		t.Error("FUSE tmp path should be empty for iSCSI storage")
	}

	if storage.GetFuseMountPath() != "" {
		t.Error("FUSE mount path should be empty for iSCSI storage")
	}

	if storage.GetImgsaveBackupPath() != "" {
		t.Error("Image save backup path should be empty for iSCSI storage")
	}
}

func TestIscsiStorageCapacityReporting(t *testing.T) {
	manager := &SStorageManager{}
	storage := NewIscsiStorage(manager, "/test/path")

	// For iSCSI storage, capacity reporting should return -1 when not connected
	if storage.GetFreeSizeMb() != -1 {
		t.Error("Free size should be -1 for iSCSI storage")
	}

	if storage.GetCapacityMb() != -1 {
		t.Error("Capacity should be -1 for iSCSI storage")
	}
}

func TestIscsiStorageDiskManagement(t *testing.T) {
	manager := &SStorageManager{}
	storage := NewIscsiStorage(manager, "/test/path")

	// Test creating disk
	disk := storage.CreateDisk("test-disk-id")
	if disk == nil {
		t.Error("CreateDisk should return a disk instance")
	}

	if disk.GetId() != "test-disk-id" {
		t.Errorf("Expected disk ID 'test-disk-id', got '%s'", disk.GetId())
	}

	// Test disk list
	if len(storage.Disks) != 1 {
		t.Errorf("Expected 1 disk, got %d", len(storage.Disks))
	}

	// Test removing disk
	storage.RemoveDisk(disk)
	if len(storage.Disks) != 0 {
		t.Errorf("Expected 0 disks after removal, got %d", len(storage.Disks))
	}

	// Test GetDisksPath
	paths, err := storage.GetDisksPath()
	if err != nil {
		t.Errorf("GetDisksPath failed: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("Expected empty paths for iSCSI storage, got %d paths", len(paths))
	}
}
