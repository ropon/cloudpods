# iSCSI Storage Manager Implementation

This document describes the implementation of the iSCSI storage manager for the OneCloud hostman service.

## Overview

The `SIscsiStorage` struct implements the `IStorage` interface to provide iSCSI storage support. It manages connections to iSCSI targets and provides disk management capabilities.

## Key Features

### 1. iSCSI Connection Management

- **MountStorage()**: Connects to iSCSI target using `iscsiadm` tool
  - Discovers iSCSI targets
  - Logs into the target with optional CHAP authentication
  - Waits for device to appear and discovers device path
- **UnmountStorage()**: Disconnects from iSCSI target
  - Logs out from the target
  - Cleans up discovery records
  - Resets connection state

### 2. Device Discovery and Path Management

- **findDevicePath()**: Locates the iSCSI device in `/dev/disk/by-path/`
- **waitForDevice()**: Waits up to 30 seconds for device to appear after login
- **getDeviceSize()**: Gets device size using `blockdev` command

### 3. Authentication Support

- Supports CHAP authentication with username/password
- **setAuthParams()**: Configures authentication parameters using `iscsiadm`
- **setNodeParam()**: Sets individual node parameters

### 4. Storage Interface Implementation

- Implements all required `IStorage` interface methods
- Uses `NewLocalDisk` for disk instances (iSCSI devices appear as local block devices)
- Provides appropriate error responses for unsupported operations (snapshots, backups, etc.)

## Configuration Structure

```go
type SIscsiStorageConf struct {
    Target   string `json:"target"`     // iSCSI Target IP address
    Iqn      string `json:"iqn"`        // iSCSI Qualified Name
    Portal   string `json:"portal"`     // iSCSI Portal (IP:port)
    Username string `json:"username,omitempty"` // CHAP username (optional)
    Password string `json:"password,omitempty"` // CHAP password (optional)
    LunId    int    `json:"lun_id"`     // LUN ID
}
```

## Usage Flow

1. **Storage Creation**: Factory creates `SIscsiStorage` instance
2. **Configuration**: `SetStorageInfo()` configures iSCSI parameters
3. **Connection**: `MountStorage()` establishes iSCSI connection
4. **Device Access**: Storage provides access to iSCSI device as local disk
5. **Cleanup**: `UnmountStorage()` disconnects when no longer needed

## Thread Safety

- Uses `sync.RWMutex` for connection state management
- Inherits disk management locking from `SBaseStorage`

## Error Handling

- Comprehensive error handling for all iSCSI operations
- Graceful handling of already connected/disconnected states
- Cleanup on connection failures

## Limitations

- Snapshots not supported (iSCSI is block-level storage)
- Backup operations not supported at storage level
- Migration operations not supported
- Clone operations not supported

## Dependencies

- `iscsiadm` tool must be available on the host system
- `blockdev` tool for device size queries
- Proper network connectivity to iSCSI target

## Testing

The implementation includes comprehensive unit tests covering:

- Factory functionality
- Basic properties and configuration
- Connection state management
- Unsupported operations
- Disk management
- Capacity reporting
