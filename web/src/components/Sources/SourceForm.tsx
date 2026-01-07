import { useState } from 'react';
import {
  FormControl,
  TextInput,
  Select,
  Button,
  Flash,
  Spinner,
  Text,
} from '@primer/react';
import { CheckCircleIcon, XCircleIcon, ChevronDownIcon, ChevronRightIcon, SyncIcon, CheckIcon, XIcon } from '@primer/octicons-react';
import { SuccessButton, SecondaryButton, PrimaryButton } from '../common/buttons';
import type { Source, CreateSourceRequest, UpdateSourceRequest, SourceType } from '../../types';
import { sourcesApi } from '../../services/api/sources';

interface SourceFormProps {
  source?: Source | null;
  onSubmit: (data: CreateSourceRequest | UpdateSourceRequest) => Promise<void>;
  onCancel: () => void;
  isSubmitting?: boolean;
}

// Get default base URL based on source type
function getDefaultBaseUrl(type: SourceType): string {
  return type === 'github' ? 'https://api.github.com' : 'https://dev.azure.com/';
}

/**
 * Reusable form for creating or editing sources.
 */
export function SourceForm({ source, onSubmit, onCancel, isSubmitting }: SourceFormProps) {
  const isEditing = !!source;
  const initialType = (source?.type || 'github') as SourceType;
  
  const [formData, setFormData] = useState({
    name: source?.name || '',
    type: initialType,
    base_url: source?.base_url || getDefaultBaseUrl(initialType),
    token: '',
    organization: source?.organization || '',
    enterprise_slug: source?.enterprise_slug || '',
    // GitHub App fields for discovery operations
    app_id: source?.app_id?.toString() || '',
    app_private_key: '',
    app_installation_id: '',
    // OAuth fields for user self-service (GitHub/GHES)
    oauth_client_id: '',
    oauth_client_secret: '',
    // OAuth fields for user self-service (Azure DevOps / Entra ID)
    entra_tenant_id: '',
    entra_client_id: '',
    entra_client_secret: '',
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [validationState, setValidationState] = useState<'idle' | 'validating' | 'success' | 'error'>('idle');
  const [validationMessage, setValidationMessage] = useState('');
  const [showAppConfig, setShowAppConfig] = useState(source?.has_app_auth || false);
  // Track if connection-critical fields were modified during edit (token, base_url, or organization for ADO)
  const [connectionFieldsChanged, setConnectionFieldsChanged] = useState(false);

  const handleChange = (field: keyof typeof formData, value: string) => {
    setFormData(prev => {
      const updated = { ...prev, [field]: value };
      // When type changes and base_url is still the default, update to new default
      if (field === 'type') {
        const oldDefaultUrl = getDefaultBaseUrl(prev.type);
        const newDefaultUrl = getDefaultBaseUrl(value as SourceType);
        if (prev.base_url === oldDefaultUrl || !prev.base_url) {
          updated.base_url = newDefaultUrl;
        }
      }
      return updated;
    });
    setErrors(prev => ({ ...prev, [field]: '' }));
    
    // Track connection-critical field changes during edit
    // These fields affect connectivity and require re-testing
    const connectionCriticalFields = ['token', 'base_url', 'organization'];
    if (isEditing && connectionCriticalFields.includes(field) && value.trim() !== '') {
      setConnectionFieldsChanged(true);
      setValidationState('idle'); // Reset validation since connection params changed
    } else if (!connectionCriticalFields.includes(field)) {
      // For non-critical fields (name, enterprise_slug), just reset validation state
      setValidationState('idle');
    }
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (!formData.base_url.trim()) {
      newErrors.base_url = 'Base URL is required';
    }
    if (!isEditing && !formData.token.trim()) {
      newErrors.token = 'Token is required';
    }
    if (formData.type === 'azuredevops' && !formData.organization.trim()) {
      newErrors.organization = 'Organization is required for Azure DevOps';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleTestConnection = async () => {
    if (!formData.base_url || (!isEditing && !formData.token)) {
      setValidationState('error');
      setValidationMessage('Please fill in the required fields first');
      return;
    }

    setValidationState('validating');
    setValidationMessage('');

    try {
      const result = await sourcesApi.validate({
        type: formData.type,
        base_url: formData.base_url,
        token: isEditing ? (formData.token || source?.masked_token || '') : formData.token,
        // Only include organization for ADO sources
        ...(formData.type === 'azuredevops' && { organization: formData.organization || undefined }),
        // Only include enterprise_slug for GitHub sources
        ...(formData.type === 'github' && { enterprise_slug: formData.enterprise_slug || undefined }),
      });

      if (result.valid) {
        setValidationState('success');
        setValidationMessage('Connection successful!');
      } else {
        setValidationState('error');
        setValidationMessage(result.error || 'Connection failed');
      }
    } catch (err) {
      setValidationState('error');
      setValidationMessage(err instanceof Error ? err.message : 'Connection test failed');
    }
  };

  const handleSubmit = async () => {
    if (!validate()) return;

    // Build GitHub App fields (only for GitHub sources)
    const appFields = formData.type === 'github'
      ? {
          ...(formData.app_id && { app_id: parseInt(formData.app_id) }),
          ...(formData.app_private_key && { app_private_key: formData.app_private_key }),
          ...(formData.app_installation_id && { app_installation_id: parseInt(formData.app_installation_id) }),
        }
      : {};

    // Build OAuth fields based on type
    const oauthFields = formData.type === 'github'
      ? {
          ...(formData.oauth_client_id && { oauth_client_id: formData.oauth_client_id }),
          ...(formData.oauth_client_secret && { oauth_client_secret: formData.oauth_client_secret }),
        }
      : {
          ...(formData.entra_tenant_id && { entra_tenant_id: formData.entra_tenant_id }),
          ...(formData.entra_client_id && { entra_client_id: formData.entra_client_id }),
          ...(formData.entra_client_secret && { entra_client_secret: formData.entra_client_secret }),
        };

    const data: CreateSourceRequest | UpdateSourceRequest = isEditing
      ? {
          name: formData.name,
          base_url: formData.base_url,
          ...(formData.token && { token: formData.token }),
          // For ADO, organization is required
          ...(formData.type === 'azuredevops' && { organization: formData.organization }),
          // For GitHub, only include enterprise_slug (organization not needed)
          ...(formData.type === 'github' && formData.enterprise_slug && { enterprise_slug: formData.enterprise_slug }),
          ...appFields,
          ...oauthFields,
        }
      : {
          name: formData.name,
          type: formData.type,
          base_url: formData.base_url,
          token: formData.token,
          // For ADO, organization is required
          ...(formData.type === 'azuredevops' && { organization: formData.organization }),
          // For GitHub, only include enterprise_slug (organization not needed)
          ...(formData.type === 'github' && formData.enterprise_slug && { enterprise_slug: formData.enterprise_slug }),
          ...appFields,
          ...oauthFields,
        };

    await onSubmit(data);
  };

  const handleFormSubmit = () => {
    handleSubmit();
  };

  return (
    <div className="space-y-4">
      {/* Name */}
      <FormControl>
        <FormControl.Label>Name</FormControl.Label>
        <TextInput
          value={formData.name}
          onChange={(e) => handleChange('name', e.target.value)}
          placeholder="e.g., GHES Production, ADO Main"
          block
        />
        {errors.name && (
          <FormControl.Validation variant="error">{errors.name}</FormControl.Validation>
        )}
        <FormControl.Caption>
          A friendly name to identify this source
        </FormControl.Caption>
      </FormControl>

      {/* Type (only for new sources) */}
      {!isEditing && (
        <FormControl>
          <FormControl.Label>Type</FormControl.Label>
          <Select
            value={formData.type}
            onChange={(e) => handleChange('type', e.target.value)}
          >
            <Select.Option value="github">GitHub</Select.Option>
            <Select.Option value="azuredevops">Azure DevOps</Select.Option>
          </Select>
        </FormControl>
      )}

      {/* Base URL */}
      <FormControl>
        <FormControl.Label>Base URL</FormControl.Label>
        <TextInput
          value={formData.base_url}
          onChange={(e) => handleChange('base_url', e.target.value)}
          placeholder={formData.type === 'github' ? 'https://api.github.com' : 'https://dev.azure.com/your-org'}
          block
        />
        {errors.base_url && (
          <FormControl.Validation variant="error">{errors.base_url}</FormControl.Validation>
        )}
        <FormControl.Caption>
          {formData.type === 'github' 
            ? 'API endpoint (e.g., https://api.github.com for github.com or https://ghes.example.com/api/v3 for GHES)'
            : 'Azure DevOps organization URL'}
        </FormControl.Caption>
      </FormControl>

      {/* Token */}
      <FormControl>
        <FormControl.Label>
          {isEditing ? 'Token (leave blank to keep existing)' : 'Personal Access Token'}
        </FormControl.Label>
        <TextInput
          type="password"
          value={formData.token}
          onChange={(e) => handleChange('token', e.target.value)}
          placeholder={isEditing ? '••••••••' : 'Enter your PAT'}
          block
        />
        {errors.token && (
          <FormControl.Validation variant="error">{errors.token}</FormControl.Validation>
        )}
        {isEditing && source?.masked_token && (
          <FormControl.Caption>
            Current token: {source.masked_token}
          </FormControl.Caption>
        )}
      </FormControl>

      {/* Enterprise Slug (GitHub only, optional) */}
      {formData.type === 'github' && (
        <FormControl>
          <FormControl.Label>Enterprise Slug (Optional)</FormControl.Label>
          <TextInput
            value={formData.enterprise_slug}
            onChange={(e) => handleChange('enterprise_slug', e.target.value)}
            placeholder="e.g., your-enterprise-slug"
            block
          />
          <FormControl.Caption>
            Pre-populate enterprise slug for enterprise-wide discovery. If specified, enterprise discovery will use this as the default.
          </FormControl.Caption>
        </FormControl>
      )}

      {/* Organization (ADO only - required) */}
      {formData.type === 'azuredevops' && (
        <FormControl>
          <FormControl.Label>Organization</FormControl.Label>
          <TextInput
            value={formData.organization}
            onChange={(e) => handleChange('organization', e.target.value)}
            placeholder="Your Azure DevOps organization"
            block
          />
          {errors.organization && (
            <FormControl.Validation variant="error">{errors.organization}</FormControl.Validation>
          )}
          <FormControl.Caption>
            Required for Azure DevOps. Will be used as the default for discovery.
          </FormControl.Caption>
        </FormControl>
      )}

      {/* GitHub App Configuration (Optional, GitHub sources only) */}
      {formData.type === 'github' && (
        <div
          className="rounded-lg border overflow-hidden"
          style={{ borderColor: 'var(--borderColor-default)' }}
        >
          <button
            type="button"
            onClick={() => setShowAppConfig(!showAppConfig)}
            className="w-full flex justify-between items-center p-3 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            style={{ backgroundColor: 'var(--bgColor-muted)' }}
          >
            <div className="flex items-center gap-2">
              {showAppConfig ? <ChevronDownIcon /> : <ChevronRightIcon />}
              <Text className="font-semibold">GitHub App Configuration (Optional)</Text>
            </div>
            {source?.has_app_auth && (
              <Text className="text-xs" style={{ color: 'var(--fgColor-success)' }}>✓ Configured</Text>
            )}
          </button>

          {showAppConfig && (
            <div 
              className="p-3 border-t"
              style={{ borderColor: 'var(--borderColor-default)' }}
            >
              <Text as="p" className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
                Configure a GitHub App for enhanced rate limits and security during discovery operations.
                The PAT above is still required for certain operations.
              </Text>

              <FormControl className="mb-3">
                <FormControl.Label>App ID</FormControl.Label>
                <TextInput
                  type="number"
                  value={formData.app_id}
                  onChange={(e) => handleChange('app_id', e.target.value)}
                  placeholder="123456"
                  block
                />
                <FormControl.Caption>
                  Find this in your GitHub App settings page.
                </FormControl.Caption>
              </FormControl>

              <FormControl className="mb-3">
                <FormControl.Label>
                  Private Key {isEditing && '(leave blank to keep existing)'}
                </FormControl.Label>
                <textarea
                  value={formData.app_private_key}
                  onChange={(e) => handleChange('app_private_key', e.target.value)}
                  placeholder={isEditing && source?.has_app_auth
                    ? 'Private key is configured. Enter a new value to update.'
                    : '-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----'}
                  className="w-full p-2 rounded border font-mono text-sm"
                  style={{ 
                    backgroundColor: 'var(--bgColor-default)',
                    borderColor: 'var(--borderColor-default)',
                    color: 'var(--fgColor-default)',
                    minHeight: '100px',
                  }}
                />
                <Text as="p" className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                  Generate a private key from your GitHub App settings and paste it here.
                </Text>
              </FormControl>

              <FormControl>
                <FormControl.Label>Installation ID</FormControl.Label>
                <TextInput
                  type="number"
                  value={formData.app_installation_id}
                  onChange={(e) => handleChange('app_installation_id', e.target.value)}
                  placeholder="12345678"
                  block
                />
                <FormControl.Caption>
                  The installation ID from when the app was installed to your organization.
                </FormControl.Caption>
              </FormControl>
            </div>
          )}
        </div>
      )}

      {/* Validation Result */}
      {validationState === 'success' && (
        <Flash variant="success">
          <CheckCircleIcon /> {validationMessage}
        </Flash>
      )}
      {validationState === 'error' && (
        <Flash variant="danger">
          <XCircleIcon /> {validationMessage}
        </Flash>
      )}

      {/* Form Actions */}
      <div className="flex gap-3 pt-4 border-t" style={{ borderColor: 'var(--borderColor-muted)' }}>
        {/* Test Connection Button - styled like DestinationSettings */}
        {validationState === 'success' ? (
          <SuccessButton
            type="button"
            onClick={handleTestConnection}
            leadingVisual={CheckIcon}
          >
            Connected
          </SuccessButton>
        ) : validationState === 'error' ? (
          <Button
            type="button"
            onClick={handleTestConnection}
            variant="danger"
            leadingVisual={XIcon}
          >
            Retry Test
          </Button>
        ) : (
          <SecondaryButton
            type="button"
            onClick={handleTestConnection}
            disabled={validationState === 'validating'}
            leadingVisual={validationState === 'validating' ? Spinner : SyncIcon}
          >
            {validationState === 'validating' ? 'Testing...' : 'Test Connection'}
          </SecondaryButton>
        )}
        
        {/* Show helpful message about connection test requirement */}
        {isEditing && !connectionFieldsChanged && validationState === 'idle' && (
          <div className="flex items-center text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            Connection test optional (no connection settings changed)
          </div>
        )}
        
        <div className="flex-1" />
        
        <Button type="button" onClick={onCancel}>
          Cancel
        </Button>
        <PrimaryButton 
          type="button" 
          onClick={handleFormSubmit} 
          disabled={
            isSubmitting || 
            // Require connection test only for new sources or when connection fields changed during edit
            ((!isEditing || connectionFieldsChanged) && validationState !== 'success')
          }
          title={
            (!isEditing || connectionFieldsChanged) && validationState !== 'success' 
              ? 'Test connection before saving' 
              : undefined
          }
        >
          {isSubmitting ? (
            <>
              <Spinner size="small" /> Saving...
            </>
          ) : (
            isEditing ? 'Save Changes' : 'Create Source'
          )}
        </PrimaryButton>
      </div>
    </div>
  );
}

