import { useState } from 'react';
import { FormControl, TextInput, Button, Flash, Text, Heading } from '@primer/react';
import { CheckCircleIcon, XCircleIcon, SyncIcon, CheckIcon, XIcon } from '@primer/octicons-react';
import { SuccessButton, SecondaryButton } from '../common/buttons';
import { useMutation } from '@tanstack/react-query';
import { settingsApi, type SettingsResponse, type UpdateSettingsRequest, type ValidationResponse } from '../../services/api/settings';

interface DestinationSettingsProps {
  settings: SettingsResponse;
  onSave: (updates: UpdateSettingsRequest) => void;
  isSaving: boolean;
}

export function DestinationSettings({ settings, onSave, isSaving }: DestinationSettingsProps) {
  const [baseURL, setBaseURL] = useState(settings.destination_base_url);
  const [token, setToken] = useState('');
  const [appId, setAppId] = useState(settings.destination_app_id?.toString() || '');
  const [appPrivateKey, setAppPrivateKey] = useState('');
  const [appInstallationId, setAppInstallationId] = useState(
    settings.destination_app_installation_id?.toString() || ''
  );
  const [enterpriseSlug, setEnterpriseSlug] = useState(settings.destination_enterprise_slug || '');
  const [showAppConfig, setShowAppConfig] = useState(
    settings.destination_app_id !== undefined && settings.destination_app_id > 0
  );

  // Validation mutation
  const validateMutation = useMutation<ValidationResponse, Error>({
    mutationFn: () => settingsApi.validateDestination({
      base_url: baseURL,
      token: token || 'existing', // Use placeholder if token not changed
      app_id: appId ? parseInt(appId) : undefined,
      app_private_key: appPrivateKey || undefined,
      app_installation_id: appInstallationId ? parseInt(appInstallationId) : undefined,
    }),
  });

  const handleSave = () => {
    const updates: UpdateSettingsRequest = {
      destination_base_url: baseURL,
    };

    // Only include token if changed
    if (token) {
      updates.destination_token = token;
    }

    // App credentials
    if (showAppConfig) {
      if (appId) updates.destination_app_id = parseInt(appId);
      if (appPrivateKey) updates.destination_app_private_key = appPrivateKey;
      if (appInstallationId) updates.destination_app_installation_id = parseInt(appInstallationId);
    }

    // Enterprise slug (empty string clears it)
    updates.destination_enterprise_slug = enterpriseSlug || undefined;

    onSave(updates);
  };

  const hasChanges = baseURL !== settings.destination_base_url || 
    token !== '' || 
    appPrivateKey !== '' ||
    appId !== (settings.destination_app_id?.toString() || '') ||
    appInstallationId !== (settings.destination_app_installation_id?.toString() || '') ||
    enterpriseSlug !== (settings.destination_enterprise_slug || '');

  return (
    <div className="max-w-2xl">
      <Heading as="h2" className="text-lg mb-2">Destination GitHub</Heading>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure the destination GitHub instance where repositories will be migrated to.
        Supported destinations are GitHub.com (GHEC) and GHEC with data residency.
      </Text>

      {/* Status indicator */}
      {settings.destination_configured ? (
        <Flash variant="success" className="mb-4">
          <div className="flex items-center gap-2">
            <CheckCircleIcon size={16} />
            <Text weight="medium">Destination is configured and ready for migrations.</Text>
          </div>
        </Flash>
      ) : (
        <Flash variant="warning" className="mb-4">
          <div className="flex items-center gap-2">
            <XCircleIcon size={16} />
            <Text weight="medium">Destination is not configured. Please add your GitHub credentials.</Text>
          </div>
        </Flash>
      )}

      <div className="space-y-4">
        <FormControl>
          <FormControl.Label>Base URL</FormControl.Label>
          <TextInput
            value={baseURL}
            onChange={(e) => setBaseURL(e.target.value)}
            placeholder="https://api.github.com"
            block
            monospace
          />
          <FormControl.Caption>
            Use https://api.github.com for GitHub.com, or a GHEC data residency URL (e.g., https://api.fabrikam.ghe.com)
          </FormControl.Caption>
        </FormControl>

        <FormControl>
          <FormControl.Label>Personal Access Token</FormControl.Label>
          <TextInput
            type="password"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder={settings.destination_token_configured ? '••••••••••••••••' : 'ghp_...'}
            block
            monospace
          />
          <FormControl.Caption>
            {settings.destination_token_configured 
              ? 'Leave blank to keep the existing token, or enter a new value to update.'
              : 'A personal access token with repo and admin:org scopes.'}
          </FormControl.Caption>
        </FormControl>

        {/* GitHub App Configuration (Collapsible) */}
        <div 
          className="border rounded-lg p-4"
          style={{ borderColor: 'var(--borderColor-default)' }}
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <Text className="font-semibold block">GitHub App Authentication (Optional)</Text>
              <Text className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                Use a GitHub App for enhanced rate limits and security.
              </Text>
            </div>
            <Button
              variant="invisible"
              size="small"
              onClick={() => setShowAppConfig(!showAppConfig)}
            >
              {showAppConfig ? 'Hide' : 'Configure'}
            </Button>
          </div>

          {showAppConfig && (
            <div className="space-y-4 mt-4 pt-4 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
              <FormControl>
                <FormControl.Label>App ID</FormControl.Label>
                <TextInput
                  type="number"
                  value={appId}
                  onChange={(e) => setAppId(e.target.value)}
                  placeholder="123456"
                  block
                />
              </FormControl>

              <FormControl>
                <FormControl.Label>Private Key</FormControl.Label>
                <textarea
                  value={appPrivateKey}
                  onChange={(e) => setAppPrivateKey(e.target.value)}
                  placeholder={settings.destination_app_key_configured 
                    ? 'Private key is configured. Enter a new value to update.'
                    : '-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----'}
                  className="w-full p-2 rounded border font-mono text-sm"
                  style={{ 
                    backgroundColor: 'var(--bgColor-default)',
                    borderColor: 'var(--borderColor-default)',
                    color: 'var(--fgColor-default)',
                    minHeight: '120px',
                  }}
                />
              </FormControl>

              <FormControl>
                <FormControl.Label>Installation ID</FormControl.Label>
                <TextInput
                  type="number"
                  value={appInstallationId}
                  onChange={(e) => setAppInstallationId(e.target.value)}
                  placeholder="12345678"
                  block
                />
                <FormControl.Caption>
                  The installation ID of the GitHub App in your organization.
                </FormControl.Caption>
              </FormControl>
            </div>
          )}
        </div>

        {/* Enterprise Slug (Optional) */}
        <FormControl>
          <FormControl.Label>Enterprise Slug (Optional)</FormControl.Label>
          <TextInput
            value={enterpriseSlug}
            onChange={(e) => setEnterpriseSlug(e.target.value)}
            placeholder="e.g., my-enterprise"
            block
          />
          <FormControl.Caption>
            The slug of your GitHub Enterprise. Required for enterprise admin authorization checks.
            Find this in your GitHub Enterprise settings URL.
          </FormControl.Caption>
        </FormControl>

        {/* Validation Result */}
        {validateMutation.data && (
          <Flash variant={validateMutation.data.valid ? 'success' : 'danger'}>
            {validateMutation.data.valid ? (
              <>
                <CheckCircleIcon /> Connection successful! 
                {validateMutation.data.details?.username && (
                  <> Authenticated as <strong>{String(validateMutation.data.details.username)}</strong></>
                )}
              </>
            ) : (
              <>
                <XCircleIcon /> {validateMutation.data.error}
              </>
            )}
          </Flash>
        )}

        {/* Actions */}
        <div className="flex gap-3 pt-4">
          {validateMutation.data?.valid ? (
            <SuccessButton
              onClick={() => validateMutation.mutate()}
              disabled={!baseURL || validateMutation.isPending}
              leadingVisual={CheckIcon}
            >
              Connected
            </SuccessButton>
          ) : validateMutation.data && !validateMutation.data.valid ? (
            <Button
              onClick={() => validateMutation.mutate()}
              disabled={!baseURL || validateMutation.isPending}
              leadingVisual={XIcon}
              variant="danger"
            >
              {validateMutation.isPending ? 'Testing...' : 'Retry Test'}
            </Button>
          ) : (
            <SecondaryButton
              onClick={() => validateMutation.mutate()}
              disabled={!baseURL || validateMutation.isPending}
              leadingVisual={SyncIcon}
            >
              {validateMutation.isPending ? 'Testing...' : 'Test Connection'}
            </SecondaryButton>
          )}
          <Button
            variant="primary"
            onClick={handleSave}
            disabled={!hasChanges || isSaving}
          >
            {isSaving ? 'Saving...' : 'Save Changes'}
          </Button>
        </div>
      </div>
    </div>
  );
}

