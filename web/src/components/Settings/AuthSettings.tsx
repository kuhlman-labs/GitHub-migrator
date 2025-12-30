import { useState } from 'react';
import { FormControl, TextInput, Button, Text, Heading, Flash } from '@primer/react';
import { AlertIcon, ShieldCheckIcon } from '@primer/octicons-react';
import type { SettingsResponse, UpdateSettingsRequest } from '../../services/api/settings';

interface AuthSettingsProps {
  settings: SettingsResponse;
  onSave: (updates: UpdateSettingsRequest) => void;
  isSaving: boolean;
}

export function AuthSettings({ settings, onSave, isSaving }: AuthSettingsProps) {
  const [authEnabled, setAuthEnabled] = useState(settings.auth_enabled);
  const [sessionSecret, setSessionSecret] = useState('');
  const [sessionDuration, setSessionDuration] = useState(settings.auth_session_duration_hours);
  const [callbackURL, setCallbackURL] = useState(settings.auth_callback_url || '');
  const [frontendURL, setFrontendURL] = useState(settings.auth_frontend_url);

  const handleSave = () => {
    const updates: UpdateSettingsRequest = {
      auth_enabled: authEnabled,
      auth_session_duration_hours: sessionDuration,
      auth_frontend_url: frontendURL,
    };

    // Only include secrets if changed
    if (sessionSecret) {
      updates.auth_session_secret = sessionSecret;
    }
    if (callbackURL) {
      updates.auth_callback_url = callbackURL;
    }

    onSave(updates);
  };

  const hasChanges = 
    authEnabled !== settings.auth_enabled ||
    sessionSecret !== '' ||
    sessionDuration !== settings.auth_session_duration_hours ||
    callbackURL !== (settings.auth_callback_url || '') ||
    frontendURL !== settings.auth_frontend_url;

  return (
    <div className="max-w-2xl">
      <Heading as="h2" className="text-lg mb-2">Authentication</Heading>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure authentication for the GitHub Migrator web interface.
        OAuth settings for sources are configured per-source on the Sources page.
      </Text>

      {/* Auth Toggle */}
      <div 
        className="p-4 rounded-lg border mb-6 flex items-center justify-between"
        style={{ 
          backgroundColor: authEnabled ? 'var(--bgColor-success-muted)' : 'var(--bgColor-muted)',
          borderColor: 'var(--borderColor-default)',
        }}
      >
        <div className="flex items-center gap-3">
          <ShieldCheckIcon size={24} />
          <div>
            <Text className="font-semibold block">Enable Authentication</Text>
            <Text className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              {authEnabled 
                ? 'Users must authenticate to access the interface.'
                : 'The interface is publicly accessible without login.'}
            </Text>
          </div>
        </div>
        <Button
          variant={authEnabled ? 'danger' : 'primary'}
          size="small"
          onClick={() => setAuthEnabled(!authEnabled)}
        >
          {authEnabled ? 'Disable' : 'Enable'}
        </Button>
      </div>

      {authEnabled && (
        <>
          {!settings.auth_session_secret_set && (
            <Flash variant="warning" className="mb-4">
              <AlertIcon /> A session secret is required for authentication. Please set one below.
            </Flash>
          )}

          <div className="space-y-4">
            <FormControl>
              <FormControl.Label>Session Secret</FormControl.Label>
              <TextInput
                type="password"
                value={sessionSecret}
                onChange={(e) => setSessionSecret(e.target.value)}
                placeholder={settings.auth_session_secret_set 
                  ? '••••••••••••••••' 
                  : 'Enter a strong random secret'}
                block
                monospace
              />
              <FormControl.Caption>
                {settings.auth_session_secret_set 
                  ? 'Leave blank to keep the existing secret.'
                  : 'A random string used to sign session tokens. Should be at least 32 characters.'}
              </FormControl.Caption>
            </FormControl>

            <FormControl>
              <FormControl.Label>Session Duration (hours)</FormControl.Label>
              <TextInput
                type="number"
                value={sessionDuration}
                onChange={(e) => setSessionDuration(parseInt(e.target.value) || 24)}
                min={1}
                max={720}
                block
              />
              <FormControl.Caption>
                How long user sessions remain valid (1-720 hours).
              </FormControl.Caption>
            </FormControl>

            <FormControl>
              <FormControl.Label>Frontend URL</FormControl.Label>
              <TextInput
                value={frontendURL}
                onChange={(e) => setFrontendURL(e.target.value)}
                placeholder="http://localhost:3000"
                block
                monospace
              />
              <FormControl.Caption>
                The URL where the web interface is accessible.
              </FormControl.Caption>
            </FormControl>

            <FormControl>
              <FormControl.Label>OAuth Callback URL (Optional)</FormControl.Label>
              <TextInput
                value={callbackURL}
                onChange={(e) => setCallbackURL(e.target.value)}
                placeholder={`${frontendURL}/api/v1/auth/callback`}
                block
                monospace
              />
              <FormControl.Caption>
                The callback URL for OAuth providers. Defaults to frontend URL + /api/v1/auth/callback
              </FormControl.Caption>
            </FormControl>
          </div>

          <Flash className="mt-6">
            <Text>
              <strong>Note:</strong> OAuth configuration for GitHub and Azure DevOps sources
              is managed per-source on the <strong>Sources</strong> page. This enables source-scoped
              authentication where users log in with their source identity.
            </Text>
          </Flash>
        </>
      )}

      {!authEnabled && (
        <Flash variant="warning">
          <AlertIcon />
          <Text className="ml-2">
            Authentication is disabled. Anyone with network access to this server can perform migrations.
            Consider enabling authentication for production deployments.
          </Text>
        </Flash>
      )}

      {/* Actions */}
      <div className="flex gap-3 pt-6 mt-6 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
        <Button
          variant="primary"
          onClick={handleSave}
          disabled={!hasChanges || isSaving}
        >
          {isSaving ? 'Saving...' : 'Save Changes'}
        </Button>
      </div>
    </div>
  );
}

