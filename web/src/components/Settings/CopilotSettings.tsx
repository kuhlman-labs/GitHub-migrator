import { useState, useMemo } from 'react';
import { FormControl, Checkbox, TextInput, Text, Heading, Flash, Label, Button, Select } from '@primer/react';
import { AlertIcon, SyncIcon, CheckCircleIcon, CopilotIcon, CheckIcon, XIcon } from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsApi, type UpdateSettingsRequest } from '../../services/api/settings';
import { copilotApi } from '../../services/api/copilot';

interface CopilotSettingsProps {
  readOnly?: boolean;
}

export function CopilotSettings({ readOnly = false }: CopilotSettingsProps) {
  const queryClient = useQueryClient();
  // Track local edits separately from server state
  const [localCliPath, setLocalCliPath] = useState<string | null>(null);
  const [localModel, setLocalModel] = useState<string | null>(null);
  const [localSessionTimeout, setLocalSessionTimeout] = useState<number | null>(null);
  const [localLogLevel, setLocalLogLevel] = useState<string | null>(null);
  
  // Get current settings
  const { data: settingsData, isLoading, error } = useQuery({
    queryKey: ['settings'],
    queryFn: settingsApi.getSettings,
  });

  // Get Copilot status
  const { data: copilotStatus } = useQuery({
    queryKey: ['copilot-status'],
    queryFn: copilotApi.getStatus,
    refetchInterval: 60000, // Refresh every minute
  });

  // Validate CLI mutation
  const validateCliMutation = useMutation({
    mutationFn: (path: string) => copilotApi.validateCLI(path),
  });

  // Update settings mutation
  const updateMutation = useMutation({
    mutationFn: (updates: UpdateSettingsRequest) => settingsApi.updateSettings(updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      queryClient.invalidateQueries({ queryKey: ['copilot-status'] });
      // Clear local edits after save
      setLocalCliPath(null);
      setLocalModel(null);
      setLocalSessionTimeout(null);
      setLocalLogLevel(null);
    },
  });

  const settings = settingsData;

  // Derive displayed values: use local edits if present, otherwise server value
  const cliPath = useMemo(() => {
    if (localCliPath !== null) return localCliPath;
    return settings?.copilot_cli_path || '';
  }, [localCliPath, settings?.copilot_cli_path]);

  const model = useMemo(() => {
    if (localModel !== null) return localModel;
    return settings?.copilot_model || 'gpt-4.1';
  }, [localModel, settings?.copilot_model]);

  const sessionTimeout = useMemo(() => {
    if (localSessionTimeout !== null) return localSessionTimeout;
    return settings?.copilot_session_timeout_min || 30;
  }, [localSessionTimeout, settings?.copilot_session_timeout_min]);

  const logLevel = useMemo(() => {
    if (localLogLevel !== null) return localLogLevel;
    return settings?.copilot_log_level || 'info';
  }, [localLogLevel, settings?.copilot_log_level]);

  // Check if there are unsaved changes
  const hasChanges = 
    (localCliPath !== null && localCliPath !== (settings?.copilot_cli_path || '')) ||
    (localModel !== null && localModel !== (settings?.copilot_model || 'gpt-4.1')) ||
    (localSessionTimeout !== null && localSessionTimeout !== (settings?.copilot_session_timeout_min || 30)) ||
    (localLogLevel !== null && localLogLevel !== (settings?.copilot_log_level || 'info'));

  const handleToggleEnabled = (checked: boolean) => {
    updateMutation.mutate({ copilot_enabled: checked });
  };

  const handleToggleLicense = (checked: boolean) => {
    updateMutation.mutate({ copilot_require_license: checked });
  };

  const handleToggleStreaming = (checked: boolean) => {
    updateMutation.mutate({ copilot_streaming: checked });
  };

  const handleSaveSettings = () => {
    const updates: UpdateSettingsRequest = {};
    if (localCliPath !== null) updates.copilot_cli_path = localCliPath;
    if (localModel !== null) updates.copilot_model = localModel;
    if (localSessionTimeout !== null) updates.copilot_session_timeout_min = localSessionTimeout;
    if (localLogLevel !== null) updates.copilot_log_level = localLogLevel;
    updateMutation.mutate(updates);
  };

  const handleValidateCLI = () => {
    validateCliMutation.mutate(cliPath);
  };

  if (isLoading) {
    return (
      <div className="max-w-2xl">
        <Heading as="h2" className="text-lg mb-2">Copilot</Heading>
        <Text style={{ color: 'var(--fgColor-muted)' }}>Loading Copilot settings...</Text>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-2xl">
        <Heading as="h2" className="text-lg mb-2">Copilot</Heading>
        <Flash variant="danger">
          <AlertIcon size={16} />
          <span className="ml-2">Failed to load Copilot settings</span>
        </Flash>
      </div>
    );
  }

  const copilotEnabled = settings?.copilot_enabled ?? false;
  const requireLicense = settings?.copilot_require_license ?? true;
  const streaming = settings?.copilot_streaming ?? true;

  return (
    <div className="max-w-2xl">
      <div className="flex items-center gap-2 mb-2">
        <CopilotIcon size={24} />
        <Heading as="h2" className="text-lg">Copilot Assistant (SDK)</Heading>
      </div>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure the AI-powered migration assistant using the official Copilot SDK. 
        The assistant helps analyze repositories, plan migration waves, create batches, 
        and execute migrations through natural language with real-time streaming responses.
      </Text>

      <div className="space-y-6">
        {/* Enable/Disable Toggle */}
        <div 
          className="p-4 rounded-lg border"
          style={{ 
            borderColor: copilotEnabled ? 'var(--borderColor-success-emphasis)' : 'var(--borderColor-default)',
            backgroundColor: copilotEnabled ? 'var(--bgColor-success-muted)' : 'var(--bgColor-muted)',
          }}
        >
          <div className="flex items-start gap-3">
            <FormControl disabled={readOnly || updateMutation.isPending}>
              <Checkbox
                checked={copilotEnabled}
                onChange={(e) => handleToggleEnabled(e.target.checked)}
                disabled={readOnly || updateMutation.isPending}
              />
              <FormControl.Label className="font-semibold">
                Enable Copilot Assistant
                {updateMutation.isPending && (
                  <SyncIcon size={14} className="ml-2 animate-spin" />
                )}
              </FormControl.Label>
            </FormControl>
          </div>
          
          <Text className="block mt-2 ml-6" style={{ color: 'var(--fgColor-muted)' }}>
            When enabled, users can access the Copilot page to interact with the AI assistant
            for migration planning and execution.
          </Text>
        </div>

        {/* CLI Path Configuration */}
        <div className="p-4 rounded-lg border" style={{ borderColor: 'var(--borderColor-default)' }}>
          <Heading as="h3" className="text-sm mb-3">Copilot CLI Configuration</Heading>
          
          <FormControl className="mb-4">
            <FormControl.Label>CLI Path</FormControl.Label>
            <FormControl.Caption>
              Path to the Copilot CLI executable. Leave empty to auto-detect (checks /usr/local/bin/copilot, 
              COPILOT_CLI_PATH env var, then PATH). Pre-configured in Docker deployments.
            </FormControl.Caption>
            <div className="flex gap-2 mt-2">
              <TextInput
                value={cliPath}
                onChange={(e) => setLocalCliPath(e.target.value)}
                placeholder="/usr/local/bin/copilot"
                disabled={readOnly}
                className="flex-1"
              />
              <Button
                onClick={handleValidateCLI}
                disabled={validateCliMutation.isPending}
              >
                {validateCliMutation.isPending ? <SyncIcon className="animate-spin" /> : 'Validate'}
              </Button>
            </div>
          </FormControl>

          {validateCliMutation.data && (
            <Flash 
              variant={validateCliMutation.data.available ? 'success' : 'danger'} 
              className="mb-4"
            >
              {validateCliMutation.data.available ? (
                <div className="flex items-center gap-2">
                  <CheckIcon size={16} />
                  <span>CLI is available. Version: {validateCliMutation.data.version || 'Unknown'}</span>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <XIcon size={16} />
                  <span>{validateCliMutation.data.error || 'CLI not found'}</span>
                </div>
              )}
            </Flash>
          )}

          <FormControl className="mb-4">
            <FormControl.Label>Model</FormControl.Label>
            <FormControl.Caption>
              The AI model to use for conversations. Different models have different capabilities.
            </FormControl.Caption>
            <Select
              value={model}
              onChange={(e) => setLocalModel(e.target.value)}
              disabled={readOnly}
              className="mt-2"
            >
              <Select.Option value="gpt-4.1">GPT-4.1 (Recommended)</Select.Option>
              <Select.Option value="gpt-4o">GPT-4o</Select.Option>
              <Select.Option value="gpt-5">GPT-5</Select.Option>
              <Select.Option value="claude-sonnet-4">Claude Sonnet 4</Select.Option>
            </Select>
          </FormControl>

          <FormControl className="mb-4">
            <FormControl.Label>Session Timeout (minutes)</FormControl.Label>
            <FormControl.Caption>
              How long to keep sessions active before they expire. Default is 30 minutes.
            </FormControl.Caption>
            <TextInput
              type="number"
              value={sessionTimeout.toString()}
              onChange={(e) => setLocalSessionTimeout(parseInt(e.target.value) || 30)}
              min={5}
              max={120}
              disabled={readOnly}
              className="mt-2"
              style={{ width: '120px' }}
            />
          </FormControl>

          <FormControl>
            <FormControl.Label>Log Level</FormControl.Label>
            <FormControl.Caption>
              SDK logging verbosity. Use 'debug' for troubleshooting issues.
            </FormControl.Caption>
            <Select
              value={logLevel}
              onChange={(e) => setLocalLogLevel(e.target.value)}
              disabled={readOnly}
              className="mt-2"
            >
              <Select.Option value="error">Error</Select.Option>
              <Select.Option value="warn">Warning</Select.Option>
              <Select.Option value="info">Info (Default)</Select.Option>
              <Select.Option value="debug">Debug</Select.Option>
            </Select>
          </FormControl>

          {hasChanges && (
            <div className="mt-4 pt-4 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
              <Button
                variant="primary"
                onClick={handleSaveSettings}
                disabled={updateMutation.isPending}
              >
                {updateMutation.isPending ? <SyncIcon className="animate-spin mr-2" /> : null}
                Save Configuration
              </Button>
            </div>
          )}
        </div>

        {/* Streaming Toggle */}
        <div className="p-4 rounded-lg border" style={{ borderColor: 'var(--borderColor-default)' }}>
          <div className="flex items-start gap-3">
            <FormControl disabled={readOnly || updateMutation.isPending}>
              <Checkbox
                checked={streaming}
                onChange={(e) => handleToggleStreaming(e.target.checked)}
                disabled={readOnly || updateMutation.isPending}
              />
              <FormControl.Label className="font-semibold">
                Enable Streaming Responses
              </FormControl.Label>
            </FormControl>
          </div>
          
          <Text className="block mt-2 ml-6" style={{ color: 'var(--fgColor-muted)' }}>
            When enabled, responses are streamed in real-time as they are generated,
            providing a more interactive experience. Disable for slower connections.
          </Text>
        </div>

        {/* License Requirement */}
        <div className="p-4 rounded-lg border" style={{ borderColor: 'var(--borderColor-default)' }}>
          <div className="flex items-start gap-3">
            <FormControl disabled={readOnly || updateMutation.isPending}>
              <Checkbox
                checked={requireLicense}
                onChange={(e) => handleToggleLicense(e.target.checked)}
                disabled={readOnly || updateMutation.isPending}
              />
              <FormControl.Label className="font-semibold">
                Require Copilot License
              </FormControl.Label>
            </FormControl>
          </div>
          
          <Text className="block mt-2 ml-6" style={{ color: 'var(--fgColor-muted)' }}>
            When enabled, users must have a valid GitHub Copilot subscription to use the assistant.
            This is validated against the destination GitHub instance.
          </Text>
        </div>

        {/* Current Status */}
        <div className="p-4 rounded-lg border" style={{ borderColor: 'var(--borderColor-default)' }}>
          <Heading as="h3" className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
            Current Status
          </Heading>
          
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>Enabled:</Text>
              <Label variant={copilotEnabled ? 'success' : 'secondary'}>
                {copilotEnabled ? 'Yes' : 'No'}
              </Label>
            </div>
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>Available:</Text>
              <Label variant={copilotStatus?.available ? 'success' : 'danger'}>
                {copilotStatus?.available ? 'Yes' : 'No'}
              </Label>
            </div>
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>CLI Installed:</Text>
              <Label variant={copilotStatus?.cli_installed ? 'success' : 'secondary'}>
                {copilotStatus?.cli_installed ? 'Yes' : 'No'}
              </Label>
            </div>
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>License Valid:</Text>
              <Label variant={copilotStatus?.license_valid ? 'success' : 'secondary'}>
                {copilotStatus?.license_valid ? 'Yes' : 'N/A'}
              </Label>
            </div>
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>Streaming:</Text>
              <Label variant={streaming ? 'success' : 'secondary'}>
                {streaming ? 'Enabled' : 'Disabled'}
              </Label>
            </div>
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>CLI Version:</Text>
              <Label variant="secondary">
                {copilotStatus?.cli_version || 'Unknown'}
              </Label>
            </div>
          </div>

          {copilotStatus?.unavailable_reason && (
            <Flash variant="warning" className="mt-3">
              <AlertIcon size={16} />
              <span className="ml-2">{copilotStatus.unavailable_reason}</span>
            </Flash>
          )}
        </div>

        {/* Success/Error Feedback */}
        {updateMutation.isSuccess && (
          <Flash variant="success">
            <div className="flex items-center gap-2">
              <CheckCircleIcon size={16} />
              <span>Copilot settings updated successfully.</span>
            </div>
          </Flash>
        )}

        {updateMutation.isError && (
          <Flash variant="danger">
            <div className="flex items-center gap-2">
              <AlertIcon size={16} />
              <span>Failed to update Copilot settings. Please try again.</span>
            </div>
          </Flash>
        )}

        {/* Read-only notice */}
        {readOnly && (
          <Flash variant="warning">
            <AlertIcon size={16} />
            <span className="ml-2">
              You have read-only access. Only administrators can change Copilot settings.
            </span>
          </Flash>
        )}
      </div>
    </div>
  );
}
