import { useState, useEffect } from 'react';
import { FormControl, Checkbox, Text, Heading, Flash, Label } from '@primer/react';
import { AlertIcon, SyncIcon, CheckCircleIcon } from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsApi } from '../../services/api/settings';
import type { LoggingSettingsResponse } from '../../services/api/settings';

interface LoggingSettingsProps {
  readOnly?: boolean;
}

export function LoggingSettings({ readOnly = false }: LoggingSettingsProps) {
  const queryClient = useQueryClient();
  
  const { data: loggingSettings, isLoading, error } = useQuery<LoggingSettingsResponse>({
    queryKey: ['loggingSettings'],
    queryFn: settingsApi.getLoggingSettings,
    refetchInterval: 30000, // Refresh every 30s to stay in sync
  });

  const [debugEnabled, setDebugEnabled] = useState(false);

  // Sync local state with server state
  useEffect(() => {
    if (loggingSettings) {
      setDebugEnabled(loggingSettings.debug_enabled);
    }
  }, [loggingSettings]);

  const updateMutation = useMutation({
    mutationFn: (enabled: boolean) => 
      settingsApi.updateLoggingSettings({ debug_enabled: enabled }),
    onSuccess: (data) => {
      queryClient.setQueryData(['loggingSettings'], data);
    },
  });

  const handleToggle = (checked: boolean) => {
    setDebugEnabled(checked);
    updateMutation.mutate(checked);
  };

  const hasUnsavedChanges = loggingSettings && debugEnabled !== loggingSettings.debug_enabled;

  if (isLoading) {
    return (
      <div className="max-w-2xl">
        <Heading as="h2" className="text-lg mb-2">Logging</Heading>
        <Text style={{ color: 'var(--fgColor-muted)' }}>Loading logging settings...</Text>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-2xl">
        <Heading as="h2" className="text-lg mb-2">Logging</Heading>
        <Flash variant="danger">
          <AlertIcon size={16} />
          <span className="ml-2">Failed to load logging settings</span>
        </Flash>
      </div>
    );
  }

  return (
    <div className="max-w-2xl">
      <Heading as="h2" className="text-lg mb-2">Logging</Heading>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure runtime logging behavior. Debug logging changes take effect immediately without restart.
      </Text>

      <div className="space-y-6">
        {/* Debug Mode Toggle */}
        <div 
          className="p-4 rounded-lg border"
          style={{ 
            borderColor: debugEnabled ? 'var(--borderColor-success-emphasis)' : 'var(--borderColor-default)',
            backgroundColor: debugEnabled ? 'var(--bgColor-success-muted)' : 'var(--bgColor-muted)',
          }}
        >
          <div className="flex items-start gap-3">
            <FormControl disabled={readOnly || updateMutation.isPending}>
              <Checkbox
                checked={debugEnabled}
                onChange={(e) => handleToggle(e.target.checked)}
                disabled={readOnly || updateMutation.isPending}
              />
              <FormControl.Label className="font-semibold">
                Enable Debug Logging
                {updateMutation.isPending && (
                  <SyncIcon size={14} className="ml-2 animate-spin" />
                )}
              </FormControl.Label>
            </FormControl>
          </div>
          
          <Text className="block mt-2 ml-6" style={{ color: 'var(--fgColor-muted)' }}>
            Turn on verbose debug logging for troubleshooting. This will output detailed information
            about API calls, database queries, and internal processing. Debug mode resets to default 
            ({loggingSettings?.default_level || 'info'}) on server restart.
          </Text>

          {debugEnabled && (
            <Flash variant="warning" className="mt-4 ml-6">
              <AlertIcon size={16} />
              <span className="ml-2">
                Debug logging is active. This may impact performance and create large log files.
                Disable when troubleshooting is complete.
              </span>
            </Flash>
          )}
        </div>

        {/* Current Status */}
        <div className="p-4 rounded-lg border" style={{ borderColor: 'var(--borderColor-default)' }}>
          <Heading as="h3" className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
            Current Status
          </Heading>
          
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>Current Level:</Text>
              <Label variant={debugEnabled ? 'success' : 'secondary'}>
                {loggingSettings?.current_level || 'info'}
              </Label>
            </div>
            <div className="flex justify-between">
              <Text style={{ color: 'var(--fgColor-muted)' }}>Default Level:</Text>
              <Label variant="secondary">
                {loggingSettings?.default_level || 'info'}
              </Label>
            </div>
          </div>
        </div>

        {/* Success/Error Feedback */}
        {updateMutation.isSuccess && !hasUnsavedChanges && (
          <Flash variant="success">
            <div className="flex items-center gap-2">
              <CheckCircleIcon size={16} />
              <span>Logging settings updated successfully. Changes are active immediately.</span>
            </div>
          </Flash>
        )}

        {updateMutation.isError && (
          <Flash variant="danger">
            <div className="flex items-center gap-2">
              <AlertIcon size={16} />
              <span>Failed to update logging settings. Please try again.</span>
            </div>
          </Flash>
        )}

        {/* Read-only notice */}
        {readOnly && (
          <Flash variant="warning">
            <AlertIcon size={16} />
            <span className="ml-2">
              You have read-only access. Only administrators can change logging settings.
            </span>
          </Flash>
        )}
      </div>
    </div>
  );
}

