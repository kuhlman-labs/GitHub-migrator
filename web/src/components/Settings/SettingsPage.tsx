import { useState } from 'react';
import { UnderlineNav, Heading, Text, Flash, Spinner } from '@primer/react';
import { GearIcon, ServerIcon, ShieldCheckIcon, SyncIcon, RepoIcon } from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsApi } from '../../services/api/settings';
import { SourcesSettings } from './SourcesSettings';
import { DestinationSettings } from './DestinationSettings';
import { MigrationSettings } from './MigrationSettings';
import { AuthSettings } from './AuthSettings';
import { useToast } from '../../contexts/ToastContext';
import type { SettingsResponse, UpdateSettingsRequest } from '../../services/api/settings';

type SettingsTab = 'sources' | 'destination' | 'migration' | 'auth';

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('sources');
  const { showSuccess, showError } = useToast();
  const queryClient = useQueryClient();

  // Fetch current settings
  const { data: settings, isLoading, error } = useQuery<SettingsResponse, Error>({
    queryKey: ['settings'],
    queryFn: settingsApi.getSettings,
  });

  // Update settings mutation
  const updateMutation = useMutation({
    mutationFn: (request: UpdateSettingsRequest) => settingsApi.updateSettings(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      queryClient.invalidateQueries({ queryKey: ['setupProgress'] });
      showSuccess('Settings saved successfully');
    },
    onError: (error: Error) => {
      showError(`Failed to save settings: ${error.message}`);
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
      </UnderlineNav>

      {/* Tab Content */}
      <div className="mt-6">
        {activeTab === 'sources' && (
          <SourcesSettings />
        )}
        {activeTab === 'destination' && settings && (
          <DestinationSettings
            settings={settings}
            onSave={(updates) => updateMutation.mutate(updates)}
            isSaving={updateMutation.isPending}
          />
        )}
        {activeTab === 'migration' && settings && (
          <MigrationSettings
            settings={settings}
            onSave={(updates) => updateMutation.mutate(updates)}
            isSaving={updateMutation.isPending}
          />
        )}
        {activeTab === 'auth' && settings && (
          <AuthSettings
            settings={settings}
            onSave={(updates) => updateMutation.mutate(updates)}
            isSaving={updateMutation.isPending}
          />
        )}
      </div>
    </div>
  );
}

