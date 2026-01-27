import { useState } from 'react';
import { UnderlineNav, Heading, Text, Flash, Spinner } from '@primer/react';
import { GearIcon, ServerIcon, ShieldCheckIcon, SyncIcon, RepoIcon, AlertIcon, LogIcon, CopilotIcon } from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsApi } from '../../services/api/settings';
import { configApi } from '../../services/api/config';
import { SourcesSettings } from './SourcesSettings';
import { DestinationSettings } from './DestinationSettings';
import { MigrationSettings } from './MigrationSettings';
import { AuthSettings } from './AuthSettings';
import { LoggingSettings } from './LoggingSettings';
import { CopilotSettings } from './CopilotSettings';
import { useToast } from '../../contexts/ToastContext';
import { useAuth } from '../../contexts/AuthContext';
import type { SettingsResponse, UpdateSettingsRequest } from '../../services/api/settings';
import { AxiosError } from 'axios';

type SettingsTab = 'sources' | 'destination' | 'migration' | 'auth' | 'logging' | 'copilot';

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('sources');
  const { showSuccess, showError, showWarning } = useToast();
  const queryClient = useQueryClient();
  const { authEnabled } = useAuth();

  // Fetch authorization status to check if user is admin
  const { data: authStatus } = useQuery({
    queryKey: ['authorizationStatus'],
    queryFn: configApi.getAuthorizationStatus,
    enabled: authEnabled, // Only fetch if auth is enabled
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  });

  const isAdmin = !authEnabled || authStatus?.tier === 'admin';

  // Fetch current settings
  const { data: settings, isLoading, error } = useQuery<SettingsResponse, Error>({
    queryKey: ['settings'],
    queryFn: settingsApi.getSettings,
  });

  // Update settings mutation
  const updateMutation = useMutation({
    mutationFn: (request: UpdateSettingsRequest) => settingsApi.updateSettings(request),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      queryClient.invalidateQueries({ queryKey: ['setupProgress'] });
      
      // Check if restart is required (auth settings changed)
      if (response.restart_required) {
        showWarning(response.message || 'Settings saved. Server restart required for changes to take effect.');
      } else {
        showSuccess('Settings saved successfully');
      }
    },
    onError: (error: Error | AxiosError) => {
      // Handle 403 Forbidden specifically
      if (error instanceof AxiosError && error.response?.status === 403) {
        showError('Access denied: Only administrators can modify settings');
      } else {
      showError(`Failed to save settings: ${error.message}`);
      }
    },
  });

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center min-h-[400px]">
        <Spinner size="large" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <Flash variant="danger">
          Failed to load settings: {error.message}
        </Flash>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center gap-3 mb-2">
        <GearIcon size={28} />
        <Heading as="h1" className="text-2xl">Settings</Heading>
      </div>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure sources, destination, migration behavior, and authentication.
        Changes are applied immediately without requiring a server restart.
      </Text>

      {/* Non-admin warning */}
      {authEnabled && !isAdmin && (
        <Flash variant="warning" className="mb-4">
          <div className="flex items-center gap-2">
            <AlertIcon size={16} />
            <Text>
              <strong>Read-Only Access:</strong> Only Tier 1 administrators can modify settings.
              You can view the current configuration but cannot make changes.
            </Text>
          </div>
        </Flash>
      )}

      {/* Tabs */}
      <UnderlineNav aria-label="Settings">
        <UnderlineNav.Item
          aria-current={activeTab === 'sources' ? 'page' : undefined}
          onClick={() => setActiveTab('sources')}
          icon={RepoIcon}
        >
          Sources
        </UnderlineNav.Item>
        <UnderlineNav.Item
          aria-current={activeTab === 'destination' ? 'page' : undefined}
          onClick={() => setActiveTab('destination')}
          icon={ServerIcon}
        >
          Destination
        </UnderlineNav.Item>
        <UnderlineNav.Item
          aria-current={activeTab === 'migration' ? 'page' : undefined}
          onClick={() => setActiveTab('migration')}
          icon={SyncIcon}
        >
          Migration
        </UnderlineNav.Item>
        <UnderlineNav.Item
          aria-current={activeTab === 'auth' ? 'page' : undefined}
          onClick={() => setActiveTab('auth')}
          icon={ShieldCheckIcon}
        >
          Authentication
        </UnderlineNav.Item>
        <UnderlineNav.Item
          aria-current={activeTab === 'logging' ? 'page' : undefined}
          onClick={() => setActiveTab('logging')}
          icon={LogIcon}
        >
          Logging
        </UnderlineNav.Item>
        <UnderlineNav.Item
          aria-current={activeTab === 'copilot' ? 'page' : undefined}
          onClick={() => setActiveTab('copilot')}
          icon={CopilotIcon}
        >
          Copilot
        </UnderlineNav.Item>
      </UnderlineNav>

      {/* Tab Content */}
      <div className="mt-6">
        {activeTab === 'sources' && (
          <SourcesSettings readOnly={!isAdmin} />
        )}
        {activeTab === 'destination' && settings && (
          <DestinationSettings
            settings={settings}
            onSave={(updates) => updateMutation.mutate(updates)}
            isSaving={updateMutation.isPending}
            readOnly={!isAdmin}
          />
        )}
        {activeTab === 'migration' && settings && (
          <MigrationSettings
            settings={settings}
            onSave={(updates) => updateMutation.mutate(updates)}
            isSaving={updateMutation.isPending}
            readOnly={!isAdmin}
          />
        )}
        {activeTab === 'auth' && settings && (
          <AuthSettings
            settings={settings}
            onSave={(updates) => updateMutation.mutate(updates)}
            isSaving={updateMutation.isPending}
            readOnly={!isAdmin}
          />
        )}
        {activeTab === 'logging' && (
          <LoggingSettings readOnly={!isAdmin} />
        )}
        {activeTab === 'copilot' && (
          <CopilotSettings readOnly={!isAdmin} />
        )}
      </div>
    </div>
  );
}

