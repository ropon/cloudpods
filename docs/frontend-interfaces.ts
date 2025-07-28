/**
 * Frontend TypeScript interfaces for iSCSI storage support
 * This file provides type definitions for frontend developers implementing iSCSI storage UI
 */

// Storage type enumeration
export enum StorageType {
  LOCAL = 'local',
  RBD = 'rbd',
  NFS = 'nfs',
  GPFS = 'gpfs',
  ISCSI = 'iscsi',
  CLVM = 'clvm',
  SLVM = 'slvm'
}

// Medium type enumeration
export enum MediumType {
  SSD = 'ssd',
  ROTATE = 'rotate',
  HYBRID = 'hybrid'
}

// Base storage creation interface
export interface BaseStorageCreateInput {
  name: string;
  zone: string;
  storage_type: StorageType;
  medium_type: MediumType;
  capacity?: number;
}

// iSCSI specific configuration
export interface IscsiStorageConfig {
  iscsi_target: string;      // Required: iSCSI Target server address
  iscsi_iqn: string;         // Required: iSCSI Qualified Name
  iscsi_portal: string;      // Required: iSCSI Portal address (IP:Port)
  iscsi_username?: string;   // Optional: Authentication username
  iscsi_password?: string;   // Optional: Authentication password
  iscsi_lun_id?: number;     // Optional: LUN ID (default: 0)
}

// Complete iSCSI storage creation input
export interface IscsiStorageCreateInput extends BaseStorageCreateInput, IscsiStorageConfig {
  storage_type: StorageType.ISCSI;
}

// Form validation interface
export interface ValidationResult {
  isValid: boolean;
  errors: string[];
}

// Form field validation functions
export class IscsiValidator {
  /**
   * Validate IQN format according to RFC 3720
   * Format: iqn.yyyy-mm.naming-authority:unique-name
   */
  static validateIQN(iqn: string): ValidationResult {
    const iqnPattern = /^iqn\.\d{4}-\d{2}\.([a-zA-Z0-9\-\.]+):(.+)$/;
    
    if (!iqn) {
      return { isValid: false, errors: ['IQN is required'] };
    }
    
    if (!iqnPattern.test(iqn)) {
      return { 
        isValid: false, 
        errors: ['Invalid IQN format. Expected: iqn.yyyy-mm.naming-authority:unique-name'] 
      };
    }
    
    return { isValid: true, errors: [] };
  }

  /**
   * Validate Portal address format (IP:Port)
   */
  static validatePortal(portal: string): ValidationResult {
    if (!portal) {
      return { isValid: false, errors: ['Portal address is required'] };
    }

    const portalPattern = /^(\d{1,3}\.){3}\d{1,3}:\d{1,5}$/;
    
    if (!portalPattern.test(portal)) {
      return { 
        isValid: false, 
        errors: ['Invalid portal format. Expected: IP:Port (e.g., 192.168.1.100:3260)'] 
      };
    }

    const [ip, portStr] = portal.split(':');
    const port = parseInt(portStr);
    
    // Validate IP address octets
    const octets = ip.split('.');
    for (const octet of octets) {
      const num = parseInt(octet);
      if (num < 0 || num > 255) {
        return { 
          isValid: false, 
          errors: ['Invalid IP address in portal'] 
        };
      }
    }
    
    // Validate port range
    if (port < 1 || port > 65535) {
      return { 
        isValid: false, 
        errors: ['Port number must be between 1 and 65535'] 
      };
    }
    
    return { isValid: true, errors: [] };
  }

  /**
   * Validate Target IP address
   */
  static validateTarget(target: string): ValidationResult {
    if (!target) {
      return { isValid: false, errors: ['Target address is required'] };
    }

    const ipPattern = /^(\d{1,3}\.){3}\d{1,3}$/;
    
    if (!ipPattern.test(target)) {
      return { 
        isValid: false, 
        errors: ['Invalid IP address format'] 
      };
    }

    const octets = target.split('.');
    for (const octet of octets) {
      const num = parseInt(octet);
      if (num < 0 || num > 255) {
        return { 
          isValid: false, 
          errors: ['Invalid IP address'] 
        };
      }
    }
    
    return { isValid: true, errors: [] };
  }

  /**
   * Validate LUN ID
   */
  static validateLunId(lunId: number): ValidationResult {
    if (lunId < 0 || lunId > 255) {
      return { 
        isValid: false, 
        errors: ['LUN ID must be between 0 and 255'] 
      };
    }
    
    return { isValid: true, errors: [] };
  }

  /**
   * Validate complete iSCSI configuration
   */
  static validateIscsiConfig(config: IscsiStorageConfig): ValidationResult {
    const errors: string[] = [];
    
    const targetResult = this.validateTarget(config.iscsi_target);
    if (!targetResult.isValid) {
      errors.push(...targetResult.errors);
    }
    
    const iqnResult = this.validateIQN(config.iscsi_iqn);
    if (!iqnResult.isValid) {
      errors.push(...iqnResult.errors);
    }
    
    const portalResult = this.validatePortal(config.iscsi_portal);
    if (!portalResult.isValid) {
      errors.push(...portalResult.errors);
    }
    
    if (config.iscsi_lun_id !== undefined) {
      const lunResult = this.validateLunId(config.iscsi_lun_id);
      if (!lunResult.isValid) {
        errors.push(...lunResult.errors);
      }
    }
    
    return {
      isValid: errors.length === 0,
      errors
    };
  }
}

// Form state interface for React/Vue components
export interface IscsiStorageFormState {
  name: string;
  zone: string;
  storage_type: StorageType;
  medium_type: MediumType;
  iscsi_target: string;
  iscsi_iqn: string;
  iscsi_portal: string;
  iscsi_username: string;
  iscsi_password: string;
  iscsi_lun_id: number;
  showAdvanced: boolean;
  isSubmitting: boolean;
  errors: Record<string, string[]>;
}

// API response interfaces
export interface StorageCreateResponse {
  id: string;
  name: string;
  status: string;
  storage_type: string;
  created_at: string;
}

export interface ApiError {
  code: number;
  message: string;
  details?: Record<string, string[]>;
}

// Form component props interface
export interface IscsiStorageFormProps {
  zones: Array<{ id: string; name: string }>;
  onSubmit: (data: IscsiStorageCreateInput) => Promise<void>;
  onCancel: () => void;
  initialValues?: Partial<IscsiStorageFormState>;
  isLoading?: boolean;
}

// Example usage in React component
/*
import React, { useState } from 'react';

const IscsiStorageForm: React.FC<IscsiStorageFormProps> = ({ 
  zones, 
  onSubmit, 
  onCancel 
}) => {
  const [formState, setFormState] = useState<IscsiStorageFormState>({
    name: '',
    zone: '',
    storage_type: StorageType.ISCSI,
    medium_type: MediumType.SSD,
    iscsi_target: '',
    iscsi_iqn: '',
    iscsi_portal: '',
    iscsi_username: '',
    iscsi_password: '',
    iscsi_lun_id: 0,
    showAdvanced: false,
    isSubmitting: false,
    errors: {}
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    const validation = IscsiValidator.validateIscsiConfig({
      iscsi_target: formState.iscsi_target,
      iscsi_iqn: formState.iscsi_iqn,
      iscsi_portal: formState.iscsi_portal,
      iscsi_username: formState.iscsi_username,
      iscsi_password: formState.iscsi_password,
      iscsi_lun_id: formState.iscsi_lun_id
    });

    if (!validation.isValid) {
      setFormState(prev => ({ ...prev, errors: { general: validation.errors } }));
      return;
    }

    setFormState(prev => ({ ...prev, isSubmitting: true }));
    
    try {
      await onSubmit({
        name: formState.name,
        zone: formState.zone,
        storage_type: StorageType.ISCSI,
        medium_type: formState.medium_type,
        iscsi_target: formState.iscsi_target,
        iscsi_iqn: formState.iscsi_iqn,
        iscsi_portal: formState.iscsi_portal,
        iscsi_username: formState.iscsi_username || undefined,
        iscsi_password: formState.iscsi_password || undefined,
        iscsi_lun_id: formState.iscsi_lun_id
      });
    } catch (error) {
      // Handle error
    } finally {
      setFormState(prev => ({ ...prev, isSubmitting: false }));
    }
  };

  // Form JSX would go here...
};
*/