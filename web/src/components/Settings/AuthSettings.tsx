import { useState, useEffect } from 'react';
import { FormControl, TextInput, Button, Text, Heading, Flash, Box, ActionList, Label, Spinner } from '@primer/react';
import { AlertIcon, ShieldCheckIcon, ShieldLockIcon, PersonIcon, EyeIcon, ChevronDownIcon, ChevronRightIcon, LinkExternalIcon } from '@primer/octicons-react';
import { useQuery } from '@tanstack/react-query';
import type { SettingsResponse, UpdateSettingsRequest } from '../../services/api/settings';
import { useAuth } from '../../contexts/AuthContext';

interface AuthSettingsProps {
  settings: SettingsResponse;
  onSave: (updates: UpdateSettingsRequest) => void;
  isSaving: boolean;
}

// Authorization status response from the API
interface AuthorizationStatus {
  tier: 'admin' | 'self_service' | 'read_only';
  tier_name: string;
  permissions: {
    can_view_repos: boolean;
    can_migrate_own_repos: boolean;
    can_migrate_all_repos: boolean;
    can_manage_batches: boolean;
    can_manage_sources: boolean;
  };
  identity_mapping?: {
    completed: boolean;
    source_login?: string;
    source_id?: number;
    source_name?: string;
  };
  upgrade_path?: {
    action: string;
    message: string;
    link: string;
  };
}

export function AuthSettings({ settings, onSave, isSaving }: AuthSettingsProps) {
  const { user, isAuthenticated } = useAuth();
  const [authEnabled, setAuthEnabled] = useState(settings.auth_enabled);
  const [sessionSecret, setSessionSecret] = useState('');
  const [sessionDuration, setSessionDuration] = useState(settings.auth_session_duration_hours);
  const [callbackURL, setCallbackURL] = useState(settings.auth_callback_url || '');
  const [frontendURL, setFrontendURL] = useState(settings.auth_frontend_url);
  const [showHowItWorks, setShowHowItWorks] = useState(true);

  // Fetch authorization status
  const { data: authStatus, isLoading: isLoadingStatus } = useQuery<AuthorizationStatus>({
    queryKey: ['authorizationStatus'],
    queryFn: async () => {
      const response = await fetch('/api/v1/auth/authorization-status', {
        credentials: 'include',
      });
      if (!response.ok) {
        throw new Error('Failed to fetch authorization status');
      }
      return response.json();
    },
    enabled: isAuthenticated && settings.auth_enabled,
  });

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

  const getTierColor = (tier: string) => {
    switch (tier) {
      case 'admin': return 'success';
      case 'self_service': return 'accent';
      case 'read_only': return 'secondary';
      default: return 'secondary';
    }
  };

  const getTierIcon = (tier: string) => {
    switch (tier) {
      case 'admin': return <ShieldLockIcon size={24} />;
      case 'self_service': return <PersonIcon size={24} />;
      case 'read_only': return <EyeIcon size={24} />;
      default: return <ShieldCheckIcon size={24} />;
    }
  };

  return (
    <div className="max-w-2xl">
      <Heading as="h2" className="text-lg mb-2">Authentication & Authorization</Heading>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure authentication and understand your authorization level for performing migrations.
      </Text>

      {/* Current User Authorization Status */}
      {settings.auth_enabled && isAuthenticated && (
        <Box
          className="p-4 rounded-lg border mb-6"
          sx={{
            backgroundColor: 'canvas.subtle',
            borderColor: 'border.default',
          }}
        >
          <Heading as="h3" className="text-base mb-3 flex items-center gap-2">
            <ShieldCheckIcon size={20} />
            Your Authorization Level
          </Heading>

          {isLoadingStatus ? (
            <div className="flex items-center gap-2 py-4">
              <Spinner size="small" />
              <Text>Loading authorization status...</Text>
            </div>
          ) : authStatus ? (
            <div className="space-y-4">
              {/* Authorization Tier Display */}
              <Box
                className="p-4 rounded-lg"
                sx={{
                  backgroundColor: authStatus.tier === 'admin' ? 'success.subtle' : 
                                   authStatus.tier === 'self_service' ? 'accent.subtle' : 'neutral.subtle',
                  borderWidth: 1,
                  borderStyle: 'solid',
                  borderColor: authStatus.tier === 'admin' ? 'success.muted' : 
                               authStatus.tier === 'self_service' ? 'accent.muted' : 'border.default',
                }}
              >
                <div className="flex items-center gap-3 mb-2">
                  {getTierIcon(authStatus.tier)}
                  <div>
                    <Text className="font-semibold block">{authStatus.tier_name}</Text>
                    <Text className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                      Logged in as @{user?.login}
                    </Text>
                  </div>
                </div>

                {authStatus.tier === 'admin' && (
                  <Text className="text-sm mt-2 block">
                    You can migrate any repository discovered from any source.
                  </Text>
                )}
                {authStatus.tier === 'self_service' && (
                  <Text className="text-sm mt-2 block">
                    You can migrate repositories where your source identity has admin access.
                  </Text>
                )}
                {authStatus.tier === 'read_only' && (
                  <Text className="text-sm mt-2 block">
                    You can view repositories and migration status but cannot initiate migrations.
                  </Text>
                )}
              </Box>

              {/* Identity Mapping Status */}
              {authStatus.identity_mapping && (
                <div className="p-3 rounded border" style={{ borderColor: 'var(--borderColor-default)' }}>
                  <div className="flex items-center justify-between">
                    <div>
                      <Text className="font-medium block">Identity Mapping</Text>
                      {authStatus.identity_mapping.completed ? (
                        <Text className="text-sm" style={{ color: 'var(--fgColor-success)' }}>
                          ✓ Mapped to {authStatus.identity_mapping.source_login}
                          {authStatus.identity_mapping.source_name && ` (${authStatus.identity_mapping.source_name})`}
                        </Text>
                      ) : (
                        <Text className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                          Not mapped
                        </Text>
                      )}
                    </div>
                    {!authStatus.identity_mapping.completed && (
                      <Button
                        as="a"
                        href="/user-mappings"
                        size="small"
                        trailingVisual={LinkExternalIcon}
                      >
                        Complete Mapping
                      </Button>
                    )}
                  </div>
                </div>
              )}

              {/* Upgrade Path */}
              {authStatus.upgrade_path && (
                <Flash variant="warning">
                  <div className="flex items-center justify-between">
                    <Text>{authStatus.upgrade_path.message}</Text>
                    <Button
                      as="a"
                      href={authStatus.upgrade_path.link}
                      size="small"
                      variant="primary"
                    >
                      Get Started
                    </Button>
                  </div>
                </Flash>
              )}
            </div>
          ) : (
            <Flash variant="warning">
              Unable to load authorization status. Please refresh the page.
            </Flash>
          )}
        </Box>
      )}

      {/* How Authorization Works */}
      <Box
        className="rounded-lg border mb-6 overflow-hidden"
        sx={{ borderColor: 'border.default' }}
      >
        <button
          className="w-full p-4 flex items-center justify-between cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800"
          onClick={() => setShowHowItWorks(!showHowItWorks)}
          style={{ backgroundColor: 'var(--bgColor-muted)', border: 'none', textAlign: 'left' }}
        >
          <div className="flex items-center gap-2">
            <ShieldLockIcon size={20} />
            <Text className="font-semibold">How Authorization Works</Text>
          </div>
          {showHowItWorks ? <ChevronDownIcon size={20} /> : <ChevronRightIcon size={20} />}
        </button>

        {showHowItWorks && (
          <div className="p-4 space-y-4" style={{ backgroundColor: 'var(--bgColor-default)' }}>
            <Text className="block" style={{ color: 'var(--fgColor-muted)' }}>
              This application uses destination-based authorization. Your access level is determined
              by your GitHub account and identity mapping status.
            </Text>

            {/* Tier 1 */}
            <div className="p-3 rounded border-l-4" style={{ borderColor: 'var(--borderColor-success-emphasis)', backgroundColor: 'var(--bgColor-success-muted)' }}>
              <div className="flex items-center gap-2 mb-1">
                <Label variant="success">Tier 1</Label>
                <Text className="font-semibold">Full Migration Rights</Text>
              </div>
              <Text className="text-sm block" style={{ color: 'var(--fgColor-muted)' }}>
                Enterprise admins, organization owners/admins, or members of designated migration admin teams
                can migrate any discovered repository without restrictions.
              </Text>
            </div>

            {/* Tier 2 */}
            <div className="p-3 rounded border-l-4" style={{ borderColor: 'var(--borderColor-accent-emphasis)', backgroundColor: 'var(--bgColor-accent-muted)' }}>
              <div className="flex items-center gap-2 mb-1">
                <Label variant="accent">Tier 2</Label>
                <Text className="font-semibold">Self-Service</Text>
              </div>
              <Text className="text-sm block" style={{ color: 'var(--fgColor-muted)' }}>
                Users who complete identity mapping can migrate repositories where their source identity
                has admin access. This requires linking your source system username to your GitHub account.
              </Text>
            </div>

            {/* Tier 3 */}
            <div className="p-3 rounded border-l-4" style={{ borderColor: 'var(--borderColor-default)', backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="flex items-center gap-2 mb-1">
                <Label>Tier 3</Label>
                <Text className="font-semibold">Read-Only</Text>
              </div>
              <Text className="text-sm block" style={{ color: 'var(--fgColor-muted)' }}>
                All authenticated users can view repositories, batches, and migration status, but
                cannot initiate migrations until identity mapping is completed.
              </Text>
            </div>

            {/* Self-Service Steps */}
            {settings.auth_enabled && authStatus?.tier !== 'admin' && (
              <div className="mt-4 p-3 rounded" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                <Text className="font-semibold block mb-2">To enable self-service migrations:</Text>
                <ol className="list-decimal ml-5 space-y-1 text-sm">
                  <li style={{ color: isAuthenticated ? 'var(--fgColor-success)' : 'var(--fgColor-muted)' }}>
                    {isAuthenticated ? '✓' : '○'} Authenticate with GitHub (destination)
                  </li>
                  <li style={{ color: authStatus?.identity_mapping?.completed ? 'var(--fgColor-success)' : 'var(--fgColor-muted)' }}>
                    {authStatus?.identity_mapping?.completed ? '✓' : '○'} Complete identity mapping
                  </li>
                  <li style={{ color: 'var(--fgColor-muted)' }}>
                    ○ Verify source admin access on repositories
                  </li>
                </ol>
              </div>
            )}
          </div>
        )}
      </Box>

      {/* Auth Toggle */}
      <Heading as="h3" className="text-base mb-3">Authentication Settings</Heading>
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

          <Flash className="mt-6" variant="default">
            <Text>
              <strong>Destination-Centric Auth:</strong> Users authenticate with GitHub (the destination)
              using a single OAuth flow. Authorization is determined by their GitHub roles and identity mapping status.
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
