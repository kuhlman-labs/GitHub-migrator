import { useState } from 'react';
import { FormControl, TextInput, Select, Button, Text, Heading, Flash } from '@primer/react';
import type { SettingsResponse, UpdateSettingsRequest } from '../../services/api/settings';

interface MigrationSettingsProps {
  settings: SettingsResponse;
  onSave: (updates: UpdateSettingsRequest) => void;
  isSaving: boolean;
  readOnly?: boolean;
}

export function MigrationSettings({ settings, onSave, isSaving, readOnly = false }: MigrationSettingsProps) {
  const [workers, setWorkers] = useState(settings.migration_workers);
  const [pollInterval, setPollInterval] = useState(settings.migration_poll_interval_seconds);
  const [destRepoAction, setDestRepoAction] = useState(settings.migration_dest_repo_exists_action);
  const [visibilityPublic, setVisibilityPublic] = useState(settings.migration_visibility_public);
  const [visibilityInternal, setVisibilityInternal] = useState(settings.migration_visibility_internal);

  const handleSave = () => {
    const updates: UpdateSettingsRequest = {
      migration_workers: workers,
      migration_poll_interval_seconds: pollInterval,
      migration_dest_repo_exists_action: destRepoAction,
      migration_visibility_public: visibilityPublic,
      migration_visibility_internal: visibilityInternal,
    };
    onSave(updates);
  };

  const hasChanges = 
    workers !== settings.migration_workers ||
    pollInterval !== settings.migration_poll_interval_seconds ||
    destRepoAction !== settings.migration_dest_repo_exists_action ||
    visibilityPublic !== settings.migration_visibility_public ||
    visibilityInternal !== settings.migration_visibility_internal;

  return (
    <div className="max-w-2xl">
      <Heading as="h2" className="text-lg mb-2">Migration Settings</Heading>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure how migrations are processed, including worker count and visibility handling.
      </Text>

      <div className="space-y-6">
        {/* Worker Settings */}
        <div>
          <Heading as="h3" className="text-base mb-4">Worker Configuration</Heading>
          
          <div className="grid grid-cols-2 gap-4">
            <FormControl>
              <FormControl.Label>Worker Count</FormControl.Label>
              <TextInput
                type="number"
                value={workers}
                onChange={(e) => setWorkers(parseInt(e.target.value) || 5)}
                min={1}
                max={20}
                block
              />
              <FormControl.Caption>
                Number of concurrent migrations (1-20)
              </FormControl.Caption>
            </FormControl>

            <FormControl>
              <FormControl.Label>Poll Interval (seconds)</FormControl.Label>
              <TextInput
                type="number"
                value={pollInterval}
                onChange={(e) => setPollInterval(parseInt(e.target.value) || 30)}
                min={10}
                max={300}
                block
              />
              <FormControl.Caption>
                How often to check for new migrations
              </FormControl.Caption>
            </FormControl>
          </div>
        </div>

        {/* Conflict Handling */}
        <div>
          <Heading as="h3" className="text-base mb-4">Conflict Handling</Heading>
          
          <FormControl>
            <FormControl.Label>Destination Repository Exists</FormControl.Label>
            <Select
              value={destRepoAction}
              onChange={(e) => setDestRepoAction(e.target.value)}
            >
              <Select.Option value="fail">Fail - Stop migration if destination exists</Select.Option>
              <Select.Option value="skip">Skip - Skip repository if destination exists</Select.Option>
              <Select.Option value="delete">Delete - Delete existing destination and migrate</Select.Option>
            </Select>
            <FormControl.Caption>
              Action to take when the destination repository already exists.
            </FormControl.Caption>
          </FormControl>

          {destRepoAction === 'delete' && (
            <Flash variant="danger" className="mt-2">
              <strong>Warning:</strong> The "Delete" option will permanently delete the existing destination
              repository and all its data before migrating. This action cannot be undone.
            </Flash>
          )}
        </div>

        {/* Visibility Handling */}
        <div>
          <Heading as="h3" className="text-base mb-4">Visibility Handling</Heading>
          <Text className="block mb-4" style={{ color: 'var(--fgColor-muted)' }}>
            Control how repository visibility is mapped from source to destination.
          </Text>
          
          <div className="grid grid-cols-2 gap-4">
            <FormControl>
              <FormControl.Label>Public Repositories</FormControl.Label>
              <Select
                value={visibilityPublic}
                onChange={(e) => setVisibilityPublic(e.target.value)}
              >
                <Select.Option value="public">Keep Public</Select.Option>
                <Select.Option value="internal">Convert to Internal</Select.Option>
                <Select.Option value="private">Convert to Private</Select.Option>
              </Select>
              <FormControl.Caption>
                Visibility for source public repositories
              </FormControl.Caption>
            </FormControl>

            <FormControl>
              <FormControl.Label>Internal Repositories</FormControl.Label>
              <Select
                value={visibilityInternal}
                onChange={(e) => setVisibilityInternal(e.target.value)}
              >
                <Select.Option value="internal">Keep Internal</Select.Option>
                <Select.Option value="private">Convert to Private</Select.Option>
              </Select>
              <FormControl.Caption>
                Visibility for source internal repositories
              </FormControl.Caption>
            </FormControl>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3 pt-4 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
          <Button
            variant="primary"
            onClick={handleSave}
            disabled={!hasChanges || isSaving || readOnly}
            title={readOnly ? 'Administrator access required to modify settings' : undefined}
          >
            {isSaving ? 'Saving...' : 'Save Changes'}
          </Button>
        </div>
      </div>
    </div>
  );
}

