import { useState } from 'react';
import { FormControl, TextInput, Button, Text, Heading, Flash, Label, Spinner, Checkbox } from '@primer/react';
import { AlertIcon, ShieldCheckIcon, ShieldLockIcon, PersonIcon, EyeIcon, ChevronDownIcon, ChevronRightIcon, LinkExternalIcon, PeopleIcon } from '@primer/octicons-react';
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
  const [oauthClientId, setOAuthClientId] = useState(settings.auth_github_oauth_client_id || '');
  const [oauthClientSecret, setOAuthClientSecret] = useState('');
  const [sessionSecret, setSessionSecret] = useState('');
  const [sessionDuration, setSessionDuration] = useState(settings.auth_session_duration_hours);
  const [callbackURL, setCallbackURL] = useState(settings.auth_callback_url || '');
  const [frontendURL, setFrontendURL] = useState(settings.auth_frontend_url);
  const [showHowItWorks, setShowHowItWorks] = useState(true);
  const [showAuthorizationRules, setShowAuthorizationRules] = useState(true);

  // Authorization rules state
  const [migrationAdminTeams, setMigrationAdminTeams] = useState(
    settings.authorization_rules?.migration_admin_teams?.join(', ') || ''
  );
  const [allowOrgAdminMigrations, setAllowOrgAdminMigrations] = useState(
    settings.authorization_rules?.allow_org_admin_migrations || false
  );
  const [allowEnterpriseAdminMigrations, setAllowEnterpriseAdminMigrations] = useState(
    settings.authorization_rules?.allow_enterprise_admin_migrations || false
  );
  const [requireIdentityMappingForSelfService, setRequireIdentityMappingForSelfService] = useState(
    settings.authorization_rules?.require_identity_mapping_for_self_service || false
  );

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

    // Include OAuth settings
    if (oauthClientId) {
      updates.auth_github_oauth_client_id = oauthClientId;
    }
    if (oauthClientSecret) {
      updates.auth_github_oauth_client_secret = oauthClientSecret;
    }

    // Only include secrets if changed
    if (sessionSecret) {
      updates.auth_session_secret = sessionSecret;
    }
    if (callbackURL) {
      updates.auth_callback_url = callbackURL;
    }

    // Include authorization rules
    updates.authorization_rules = {
      migration_admin_teams: migrationAdminTeams
        .split(',')
        .map(s => s.trim())
        .filter(s => s !== ''),
      allow_org_admin_migrations: allowOrgAdminMigrations,
      allow_enterprise_admin_migrations: allowEnterpriseAdminMigrations,
      require_identity_mapping_for_self_service: requireIdentityMappingForSelfService,
    };

    onSave(updates);
  };

  const hasChanges = 
    authEnabled !== settings.auth_enabled ||
    oauthClientId !== (settings.auth_github_oauth_client_id || '') ||
    oauthClientSecret !== '' ||
    sessionSecret !== '' ||
    sessionDuration !== settings.auth_session_duration_hours ||
    callbackURL !== (settings.auth_callback_url || '') ||
    frontendURL !== settings.auth_frontend_url ||
    migrationAdminTeams !== (settings.authorization_rules?.migration_admin_teams?.join(', ') || '') ||
    allowOrgAdminMigrations !== (settings.authorization_rules?.allow_org_admin_migrations || false) ||
    allowEnterpriseAdminMigrations !== (settings.authorization_rules?.allow_enterprise_admin_migrations || false) ||
    requireIdentityMappingForSelfService !== (settings.authorization_rules?.require_identity_mapping_for_self_service || false);

  const getTierIcon = (tier: string) => {
    switch (tier) {
      case 'admin': return <ShieldLockIcon size={24} />;
      case 'self_service': return <PersonIcon size={24} />;
      case 'read_only': return <EyeIcon size={24} />;
      default: return <ShieldCheckIcon size={24} />;
    }
  };

  // Check if at least one Tier 1 (Admin) access path is configured
  // This prevents accidentally enabling auth with no way for anyone to migrate
  const hasTier1AccessConfigured = () => {
    const hasAdminTeams = migrationAdminTeams.split(',').map(s => s.trim()).filter(s => s !== '').length > 0;
    const hasOrgAdminAccess = allowOrgAdminMigrations;
    const hasEnterpriseAdminAccess = allowEnterpriseAdminMigrations && settings.destination_enterprise_slug;
    
    return hasAdminTeams || hasOrgAdminAccess || hasEnterpriseAdminAccess;
  };

  // Get reason why auth cannot be enabled
  const getAuthEnableBlockReason = (): string | undefined => {
    if (!settings.destination_enterprise_slug) {
      return 'Configure destination enterprise slug in Destination Settings first';
    }
    if (!hasTier1AccessConfigured()) {
      return 'Configure at least one Tier 1 admin group (Admin Teams, Org Admins, or Enterprise Admins) to prevent lockout';
    }
    return undefined;
  };

  return (
    <div className="max-w-2xl">
      <Heading as="h2" className="text-lg mb-2">Authentication & Authorization</Heading>
      <Text className="block mb-4" style={{ color: 'var(--fgColor-muted)' }}>
        Configure authentication and understand your authorization level for performing migrations.
      </Text>

      <div 
        className="mb-6 p-3 rounded-md border"
        style={{ 
          backgroundColor: 'transparent', 
          borderColor: 'rgba(46, 160, 67, 0.5)',
          borderWidth: '1px',
          borderStyle: 'solid',
        }}
      >
        <Text style={{ color: 'var(--fgColor-default)' }}>
          <strong>GitHub Authentication:</strong> Users authenticate to the destination GitHub environment 
          via OAuth. Authorization is based on their GitHub roles and identity mapping status.
        </Text>
      </div>

      {/* Current User Authorization Status */}
      {settings.auth_enabled && isAuthenticated && (
        <div
          className="p-4 rounded-lg border mb-6"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
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
              <div
                className="p-4 rounded-lg border"
                style={{
                  backgroundColor: authStatus.tier === 'admin' ? 'var(--bgColor-success-muted)' : 
                                   authStatus.tier === 'self_service' ? 'var(--bgColor-accent-muted)' : 'var(--bgColor-muted)',
                  borderColor: authStatus.tier === 'admin' ? 'var(--borderColor-success-muted)' : 
                               authStatus.tier === 'self_service' ? 'var(--borderColor-accent-muted)' : 'var(--borderColor-default)',
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
              </div>

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
        </div>
      )}

      {/* How Authorization Works */}
      <div
        className="rounded-lg border mb-6 overflow-hidden"
        style={{ borderColor: 'var(--borderColor-default)' }}
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
      </div>

      {/* Auth Toggle */}
      <Heading as="h3" className="text-base mb-3">Authentication Settings</Heading>
      
      {/* Check if destination enterprise slug is configured */}
      {!authEnabled && !settings.destination_enterprise_slug && (
        <Flash variant="warning" className="mb-4">
          <AlertIcon />
          <Text className="ml-2">
            <strong>Prerequisite:</strong> Authentication requires the destination enterprise slug to be configured.
            Please set the Enterprise Slug in <a href="/settings" style={{ textDecoration: 'underline' }}>Destination Settings</a> first.
          </Text>
        </Flash>
      )}

      {/* Check if at least one Tier 1 group is configured */}
      {!authEnabled && settings.destination_enterprise_slug && !hasTier1AccessConfigured() && (
        <Flash variant="warning" className="mb-4">
          <AlertIcon />
          <Text className="ml-2">
            <strong>Prerequisite:</strong> Configure at least one Tier 1 admin group before enabling authentication.
            Without this, no one will be able to initiate migrations. Set one of:
            <ul className="list-disc ml-6 mt-1">
              <li>Migration Admin Teams (org/team format)</li>
              <li>Allow Organization Admin Migrations</li>
              <li>Allow Enterprise Admin Migrations</li>
            </ul>
          </Text>
        </Flash>
      )}
      
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
          disabled={!authEnabled && !!getAuthEnableBlockReason()}
          title={!authEnabled ? getAuthEnableBlockReason() : undefined}
        >
          {authEnabled ? 'Disable' : 'Enable'}
        </Button>
      </div>

      {authEnabled && (
        <>
          {!settings.auth_github_oauth_client_secret_set && (
            <Flash variant="warning" className="mb-4">
              <AlertIcon /> GitHub OAuth is not configured. Please set up your OAuth App credentials below.
            </Flash>
          )}

          {/* GitHub OAuth Configuration */}
          <Heading as="h3" className="text-base mb-3 mt-6">GitHub OAuth App</Heading>
          <Text className="block mb-4" style={{ color: 'var(--fgColor-muted)' }}>
            Create an OAuth App in your GitHub organization's settings: 
            Settings → Developer settings → OAuth Apps
          </Text>

          <div className="space-y-4 mb-6">
            <FormControl>
              <FormControl.Label>OAuth Client ID</FormControl.Label>
              <TextInput
                value={oauthClientId}
                onChange={(e) => setOAuthClientId(e.target.value)}
                placeholder="Iv1.xxxxxxxxxxxxxxxxx"
                block
                monospace
              />
              <FormControl.Caption>
                The Client ID from your GitHub OAuth App.
              </FormControl.Caption>
            </FormControl>

            <FormControl>
              <FormControl.Label>OAuth Client Secret {settings.auth_github_oauth_client_secret_set && '(leave blank to keep existing)'}</FormControl.Label>
              <TextInput
                type="password"
                value={oauthClientSecret}
                onChange={(e) => setOAuthClientSecret(e.target.value)}
                placeholder={settings.auth_github_oauth_client_secret_set ? '••••••••••••••••' : 'Enter client secret'}
                block
                monospace
              />
              <FormControl.Caption>
                The Client Secret from your GitHub OAuth App.
              </FormControl.Caption>
            </FormControl>
          </div>

          {/* Session Configuration */}
          <Heading as="h3" className="text-base mb-3">Session Settings</Heading>

          <div className="space-y-4">
            <FormControl>
              <FormControl.Label>Session Secret {settings.auth_session_secret_set && '(leave blank to keep existing)'}</FormControl.Label>
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

          {/* Authorization Rules Configuration */}
          <div
            className="rounded-lg border overflow-hidden"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <button
              className="w-full p-4 flex items-center justify-between cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800"
              onClick={() => setShowAuthorizationRules(!showAuthorizationRules)}
              style={{ backgroundColor: 'var(--bgColor-muted)', border: 'none', textAlign: 'left' }}
            >
              <div className="flex items-center gap-2">
                <PeopleIcon size={20} />
                <Text className="font-semibold">Authorization Rules</Text>
              </div>
              {showAuthorizationRules ? <ChevronDownIcon size={20} /> : <ChevronRightIcon size={20} />}
            </button>

            {showAuthorizationRules && (
              <div className="p-4 space-y-4" style={{ backgroundColor: 'var(--bgColor-default)' }}>
                <Text className="block" style={{ color: 'var(--fgColor-muted)' }}>
                  Configure who can perform migrations. These settings determine which users get
                  Tier 1 (full migration) access and how Tier 2 (self-service) access works.
                </Text>

                {/* Migration Admin Teams */}
                <FormControl>
                  <FormControl.Label>Migration Admin Teams (Tier 1)</FormControl.Label>
                  <TextInput
                    value={migrationAdminTeams}
                    onChange={(e) => setMigrationAdminTeams(e.target.value)}
                    placeholder="org/team-slug, another-org/team"
                    block
                    monospace
                  />
                  <FormControl.Caption>
                    Comma-separated list of GitHub teams (format: "org/team-slug") whose members
                    will have full migration rights. Leave empty to disable.
                  </FormControl.Caption>
                </FormControl>

                {/* Allow Org Admin Migrations */}
                <FormControl>
                  <Checkbox
                    checked={allowOrgAdminMigrations}
                    onChange={(e) => setAllowOrgAdminMigrations(e.target.checked)}
                  />
                  <FormControl.Label>
                    Allow Organization Admins to have full migration rights (Tier 1)
                  </FormControl.Label>
                  <FormControl.Caption>
                    When enabled, users who are admins of any GitHub organization can migrate any repository.
                  </FormControl.Caption>
                </FormControl>

                {/* Allow Enterprise Admin Migrations */}
                <FormControl>
                  <Checkbox
                    checked={allowEnterpriseAdminMigrations}
                    onChange={(e) => setAllowEnterpriseAdminMigrations(e.target.checked)}
                  />
                  <FormControl.Label>
                    Allow Enterprise Admins to have full migration rights (Tier 1)
                  </FormControl.Label>
                  <FormControl.Caption>
                    When enabled, GitHub Enterprise administrators can migrate any repository.
                  </FormControl.Caption>
                </FormControl>

                {/* Enable Self-Service Migrations */}
                <div className="pt-4 mt-4 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
                  <Text className="font-semibold block mb-3">Self-Service Access (Tier 2)</Text>
                  <FormControl>
                    <Checkbox
                      checked={requireIdentityMappingForSelfService}
                      onChange={(e) => setRequireIdentityMappingForSelfService(e.target.checked)}
                    />
                    <FormControl.Label>
                      Enable self-service migrations
                    </FormControl.Label>
                    <FormControl.Caption>
                      When enabled, users who complete identity mapping (linking their source and destination 
                      accounts) can migrate repositories where their source identity has admin access.
                      When disabled, only Tier 1 administrators can initiate migrations.
                    </FormControl.Caption>
                  </FormControl>
                </div>
              </div>
            )}
          </div>
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
