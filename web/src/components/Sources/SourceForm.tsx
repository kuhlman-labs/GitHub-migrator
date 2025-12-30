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
import { CheckCircleIcon, XCircleIcon, ChevronDownIcon, ChevronRightIcon } from '@primer/octicons-react';
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
  const [showOAuthConfig, setShowOAuthConfig] = useState(false);

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
    setValidationState('idle');
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
        organization: formData.organization || undefined,
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validate()) return;

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
          ...(formData.type === 'azuredevops' && { organization: formData.organization }),
          ...oauthFields,
        }
      : {
          name: formData.name,
          type: formData.type,
          base_url: formData.base_url,
          token: formData.token,
          ...(formData.type === 'azuredevops' && { organization: formData.organization }),
          ...oauthFields,
        };

    await onSubmit(data);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
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
            ? 'API endpoint (use https://api.github.com for github.com)'
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

      {/* Organization (ADO only) */}
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
        </FormControl>
      )}

      {/* OAuth Configuration (Optional) */}
      <div
        className="rounded-lg border overflow-hidden mt-4"
        style={{ borderColor: 'var(--borderColor-default)' }}
      >
        <button
          type="button"
          onClick={() => setShowOAuthConfig(!showOAuthConfig)}
          className="w-full flex justify-between items-center p-3 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          style={{ backgroundColor: 'var(--bgColor-muted)' }}
        >
          <div className="flex items-center gap-2">
            {showOAuthConfig ? <ChevronDownIcon /> : <ChevronRightIcon />}
            <Text className="font-semibold">OAuth Configuration (Optional)</Text>
          </div>
          {source?.has_oauth && (
            <Text className="text-xs" style={{ color: 'var(--fgColor-success)' }}>✓ Configured</Text>
          )}
        </button>

        {showOAuthConfig && (
          <div 
            className="p-3 border-t"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <Text as="p" className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
              Configure OAuth to enable user self-service authentication. Users will be able to log in via this source 
              and prove their repository access permissions.
            </Text>

            {formData.type === 'github' ? (
              <>
                <FormControl className="mb-3">
                  <FormControl.Label>OAuth Client ID</FormControl.Label>
                  <TextInput
                    value={formData.oauth_client_id}
                    onChange={(e) => handleChange('oauth_client_id', e.target.value)}
                    placeholder="GitHub OAuth App Client ID"
                    block
                  />
                  <FormControl.Caption>
                    Create an OAuth App at: Settings → Developer settings → OAuth Apps
                  </FormControl.Caption>
                </FormControl>

                <FormControl>
                  <FormControl.Label>
                    OAuth Client Secret {isEditing && '(leave blank to keep existing)'}
                  </FormControl.Label>
                  <TextInput
                    type="password"
                    value={formData.oauth_client_secret}
                    onChange={(e) => handleChange('oauth_client_secret', e.target.value)}
                    placeholder={isEditing ? '••••••••' : 'GitHub OAuth App Client Secret'}
                    block
                  />
                </FormControl>
              </>
            ) : (
              <>
                <FormControl className="mb-3">
                  <FormControl.Label>Entra ID Tenant ID</FormControl.Label>
                  <TextInput
                    value={formData.entra_tenant_id}
                    onChange={(e) => handleChange('entra_tenant_id', e.target.value)}
                    placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                    block
                  />
                  <FormControl.Caption>
                    Your Microsoft Entra ID (Azure AD) tenant ID
                  </FormControl.Caption>
                </FormControl>

                <FormControl className="mb-3">
                  <FormControl.Label>Entra ID Client ID</FormControl.Label>
                  <TextInput
                    value={formData.entra_client_id}
                    onChange={(e) => handleChange('entra_client_id', e.target.value)}
                    placeholder="App registration client ID"
                    block
                  />
                  <FormControl.Caption>
                    Register an app in Microsoft Entra ID with Azure DevOps API permissions
                  </FormControl.Caption>
                </FormControl>

                <FormControl>
                  <FormControl.Label>
                    Entra ID Client Secret {isEditing && '(leave blank to keep existing)'}
                  </FormControl.Label>
                  <TextInput
                    type="password"
                    value={formData.entra_client_secret}
                    onChange={(e) => handleChange('entra_client_secret', e.target.value)}
                    placeholder={isEditing ? '••••••••' : 'App registration client secret'}
                    block
                  />
                </FormControl>
              </>
            )}
          </div>
        )}
      </div>

      {/* Test Connection */}
      <div className="pt-2">
        <Button
          type="button"
          onClick={handleTestConnection}
          disabled={validationState === 'validating'}
          variant="invisible"
        >
          {validationState === 'validating' ? (
            <>
              <Spinner size="small" /> Testing...
            </>
          ) : (
            'Test Connection'
          )}
        </Button>
        
        {validationState === 'success' && (
          <Flash variant="success" className="mt-2">
            <CheckCircleIcon /> {validationMessage}
          </Flash>
        )}
        {validationState === 'error' && (
          <Flash variant="danger" className="mt-2">
            <XCircleIcon /> {validationMessage}
          </Flash>
        )}
      </div>

      {/* Form Actions */}
      <div className="flex justify-end gap-3 pt-4 border-t" style={{ borderColor: 'var(--borderColor-muted)' }}>
        <Button type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" disabled={isSubmitting}>
          {isSubmitting ? (
            <>
              <Spinner size="small" /> Saving...
            </>
          ) : (
            isEditing ? 'Save Changes' : 'Create Source'
          )}
        </Button>
      </div>
    </form>
  );
}

