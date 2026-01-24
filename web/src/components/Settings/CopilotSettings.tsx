import { useState, useEffect } from 'react';
import { FormControl, Checkbox, TextInput, Text, Heading, Flash, Label, Button, Box } from '@primer/react';
import { AlertIcon, SyncIcon, CheckCircleIcon, CopilotIcon, CheckIcon, XIcon } from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsApi } from '../../services/api/settings';
import { copilotApi } from '../../services/api/copilot';

interface CopilotSettingsProps {
  readOnly?: boolean;
}

export function CopilotSettings({ readOnly = false }: CopilotSettingsProps) {
  const queryClient = useQueryClient();
  const [cliPath, setCliPath] = useState('');
  const [hasChanges, setHasChanges] = useState(false);
  const [isInitialized, setIsInitialized] = useState(false);
  
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
    mutationFn: (updates: Record<string, unknown>) => settingsApi.updateSettings(updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      queryClient.invalidateQueries({ queryKey: ['copilot-status'] });
      setHasChanges(false);
    },
  });

  const settings = settingsData?.settings;

  // Initialize CLI path from settings (properly in useEffect)
  useEffect(() => {
    if (settings?.copilot_cli_path !== undefined && !isInitialized) {
      setCliPath(settings.copilot_cli_path || '');
      setIsInitialized(true);
    }
  }, [settings?.copilot_cli_path, isInitialized]);

  // Reset initialization when settings refetch with new data (e.g., after save)
  useEffect(() => {
    if (!hasChanges && settings?.copilot_cli_path !== undefined) {
      setCliPath(settings.copilot_cli_path || '');
    }
  }, [settings?.copilot_cli_path, hasChanges]);

  const handleToggleEnabled = (checked: boolean) => {
    updateMutation.mutate({ copilot_enabled: checked });
  };

  const handleToggleLicense = (checked: boolean) => {
    updateMutation.mutate({ copilot_require_license: checked });
  };

  const handleSaveCLIPath = () => {
    updateMutation.mutate({ copilot_cli_path: cliPath });
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

  return (
    <div className="max-w-2xl">
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
        <CopilotIcon size={24} />
        <Heading as="h2" className="text-lg">Copilot Assistant</Heading>
      </Box>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure the AI-powered migration assistant. Copilot helps analyze repositories, 
        plan migration waves, create batches, and execute migrations through natural language.
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
          
          <FormControl>
            <FormControl.Label>CLI Path</FormControl.Label>
            <FormControl.Caption>
              Path to the Copilot CLI executable. Leave empty to use 'copilot' from PATH.
            </FormControl.Caption>
            <Box sx={{ display: 'flex', gap: 2, mt: 2 }}>
              <TextInput
                value={cliPath}
                onChange={(e) => {
                  setCliPath(e.target.value);
                  setHasChanges(true);
                }}
                placeholder="/usr/local/bin/copilot"
                disabled={readOnly}
                sx={{ flex: 1 }}
              />
              <Button
                onClick={handleValidateCLI}
                disabled={validateCliMutation.isPending}
              >
                {validateCliMutation.isPending ? <SyncIcon className="animate-spin" /> : 'Validate'}
              </Button>
              <Button
                variant="primary"
                onClick={handleSaveCLIPath}
                disabled={!hasChanges || updateMutation.isPending}
              >
                Save
              </Button>
            </Box>
          </FormControl>

          {validateCliMutation.data && (
            <Flash 
              variant={validateCliMutation.data.available ? 'success' : 'danger'} 
              sx={{ mt: 3 }}
            >
              {validateCliMutation.data.available ? (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <CheckIcon size={16} />
                  <span>CLI is available. Version: {validateCliMutation.data.version || 'Unknown'}</span>
                </Box>
              ) : (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <XIcon size={16} />
                  <span>{validateCliMutation.data.error || 'CLI not found'}</span>
                </Box>
              )}
            </Flash>
          )}
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
          </div>

          {copilotStatus?.unavailable_reason && (
            <Flash variant="warning" sx={{ mt: 3 }}>
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
