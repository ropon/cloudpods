# Frontend UI Support for iSCSI Storage Type

## Overview

This document describes the frontend UI implementation requirements for supporting iSCSI storage type in the CloudPods storage creation interface.

## Storage Creation Form Updates

### 1. Storage Type Selection

The storage type dropdown should include "iSCSI" as an option alongside existing types:
- local
- rbd
- nfs
- gpfs
- iscsi (NEW)

### 2. iSCSI Configuration Fields

When "iscsi" is selected as the storage type, the following fields should be displayed:

#### Required Fields

1. **iSCSI Target** (`iscsi_target`)
   - Type: Text input
   - Validation: IP address format
   - Example: `192.168.1.100`
   - Help text: "iSCSI Target server IP address"

2. **iSCSI IQN** (`iscsi_iqn`)
   - Type: Text input
   - Validation: IQN format (RFC 3720)
   - Pattern: `^iqn\.\d{4}-\d{2}\.([a-zA-Z0-9\-\.]+):(.+)$`
   - Example: `iqn.2023-01.com.example:storage.target01`
   - Help text: "iSCSI Qualified Name (IQN) following RFC 3720 format"

3. **iSCSI Portal** (`iscsi_portal`)
   - Type: Text input
   - Validation: IP:Port format
   - Pattern: `^(\d{1,3}\.){3}\d{1,3}:\d{1,5}$`
   - Example: `192.168.1.100:3260`
   - Help text: "iSCSI Portal address with port (default: 3260)"

#### Optional Fields

4. **Authentication Username** (`iscsi_username`)
   - Type: Text input
   - Optional: true
   - Help text: "Username for CHAP authentication (optional)"

5. **Authentication Password** (`iscsi_password`)
   - Type: Password input
   - Optional: true
   - Help text: "Password for CHAP authentication (optional)"

6. **LUN ID** (`iscsi_lun_id`)
   - Type: Number input
   - Default: 0
   - Min: 0
   - Max: 255
   - Help text: "Logical Unit Number (default: 0)"

### 3. Form Validation

#### Client-side Validation

1. **IQN Format Validation**
   ```javascript
   function validateIQN(iqn) {
     const iqnPattern = /^iqn\.\d{4}-\d{2}\.([a-zA-Z0-9\-\.]+):(.+)$/;
     return iqnPattern.test(iqn);
   }
   ```

2. **Portal Address Validation**
   ```javascript
   function validatePortal(portal) {
     const portalPattern = /^(\d{1,3}\.){3}\d{1,3}:\d{1,5}$/;
     if (!portalPattern.test(portal)) return false;
     
     const [ip, port] = portal.split(':');
     const portNum = parseInt(port);
     return portNum >= 1 && portNum <= 65535;
   }
   ```

3. **IP Address Validation**
   ```javascript
   function validateIP(ip) {
     const ipPattern = /^(\d{1,3}\.){3}\d{1,3}$/;
     if (!ipPattern.test(ip)) return false;
     
     return ip.split('.').every(octet => {
       const num = parseInt(octet);
       return num >= 0 && num <= 255;
     });
   }
   ```

#### Server-side Validation

The backend will perform additional validation including:
- iSCSI target reachability test
- Authentication credential verification
- Duplicate storage configuration check

### 4. Form Behavior

#### Field Visibility
- iSCSI fields should only be visible when "iscsi" is selected as storage type
- Other storage type fields should be hidden when iSCSI is selected

#### Form Submission
```javascript
// Example form data structure for iSCSI storage
const formData = {
  name: "iscsi-storage-01",
  zone: "zone-id",
  storage_type: "iscsi",
  medium_type: "ssd",
  iscsi_target: "192.168.1.100",
  iscsi_iqn: "iqn.2023-01.com.example:storage.target01",
  iscsi_portal: "192.168.1.100:3260",
  iscsi_username: "iscsi_user", // optional
  iscsi_password: "iscsi_pass", // optional
  iscsi_lun_id: 0
};
```

### 5. Error Handling

#### Validation Errors
- Display field-specific validation errors inline
- Highlight invalid fields with red border
- Show helpful error messages

#### API Errors
- Handle connection test failures
- Display authentication errors
- Show duplicate storage warnings

### 6. User Experience

#### Progressive Disclosure
- Show basic fields first (Target, IQN, Portal)
- Provide "Advanced Options" section for authentication and LUN ID
- Use collapsible sections to reduce form complexity

#### Help and Documentation
- Provide tooltips for technical fields
- Link to iSCSI configuration documentation
- Include examples for each field

#### Real-time Validation
- Validate fields on blur
- Show validation status with icons
- Provide immediate feedback

## API Integration

### Endpoint
```
POST /api/v1/storages
```

### Request Body
```json
{
  "name": "iscsi-storage-01",
  "zone": "zone-id",
  "storage_type": "iscsi",
  "medium_type": "ssd",
  "iscsi_target": "192.168.1.100",
  "iscsi_iqn": "iqn.2023-01.com.example:storage.target01",
  "iscsi_portal": "192.168.1.100:3260",
  "iscsi_username": "iscsi_user",
  "iscsi_password": "iscsi_pass",
  "iscsi_lun_id": 0
}
```

### Response Handling
- Success: Redirect to storage list or show success message
- Validation Error: Display field-specific errors
- Server Error: Show generic error message with retry option

## Implementation Checklist

- [ ] Add "iscsi" option to storage type dropdown
- [ ] Implement iSCSI configuration form fields
- [ ] Add client-side validation for IQN and Portal formats
- [ ] Implement conditional field visibility
- [ ] Add form submission handling for iSCSI data
- [ ] Implement error handling and display
- [ ] Add help text and tooltips
- [ ] Test form with various input scenarios
- [ ] Verify API integration
- [ ] Add unit tests for validation functions

## Testing Scenarios

1. **Valid iSCSI Configuration**
   - All required fields filled correctly
   - Optional authentication provided
   - Form submits successfully

2. **Invalid IQN Format**
   - Malformed IQN string
   - Validation error displayed
   - Form submission blocked

3. **Invalid Portal Address**
   - Incorrect IP format
   - Invalid port number
   - Validation error displayed

4. **Connection Test Failure**
   - Unreachable target
   - Authentication failure
   - Appropriate error message shown

5. **Duplicate Storage**
   - Same configuration already exists
   - Warning message displayed
   - User can choose to continue or modify