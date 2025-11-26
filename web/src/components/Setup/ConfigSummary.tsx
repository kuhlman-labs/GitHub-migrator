import { Text, Heading } from '@primer/react';
import type { SetupConfig } from '../../types';

interface ConfigSummaryProps {
  config: SetupConfig;
}

function maskToken(token: string): string {
  if (token.length <= 4) return '****';
  return '****' + token.slice(-4);
}

function maskDSN(dsn: string): string {
  // For SQLite (file paths), show as-is
  if (!dsn.includes('://') && !dsn.includes('@')) {
    return dsn;
  }

  // For connection strings with passwords, mask them
  if (dsn.includes('@')) {
    const parts = dsn.split('@');
    if (parts.length >= 2) {
      const beforeAt = parts[0];
      if (beforeAt.includes(':')) {
        const userPass = beforeAt.split(':');
        if (userPass.length >= 2) {
          return userPass[0] + ':****@' + parts.slice(1).join('@');
        }
      }
      return '****@' + parts.slice(1).join('@');
    }
  }

  return dsn;
}

export function ConfigSummary({ config }: ConfigSummaryProps) {
  return (
    <div>
      {/* Source Configuration */}
      <div className="mb-6">
        <Heading as="h3" className="text-lg mb-3">
          Source
        </Heading>
        <div
          className="p-4 rounded-lg border"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SummaryRow label="Type" value={config.source.type.toUpperCase()} />
          <SummaryRow label="Base URL" value={config.source.base_url} />
          <SummaryRow label="Token" value={maskToken(config.source.token)} />
          {config.source.organization && (
            <SummaryRow label="Organization" value={config.source.organization} />
          )}
        </div>
      </div>

      {/* Destination Configuration */}
      <div className="mb-6">
        <Heading as="h3" className="text-lg mb-3">
          Destination
        </Heading>
        <div
          className="p-4 rounded-lg border"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SummaryRow label="Base URL" value={config.destination.base_url} />
          <SummaryRow label="Token" value={maskToken(config.destination.token)} />
          {config.destination.app_id && (
            <>
              <SummaryRow label="GitHub App ID" value={config.destination.app_id.toString()} />
              {config.destination.app_installation_id && (
                <SummaryRow label="Installation ID" value={config.destination.app_installation_id.toString()} />
              )}
              {config.destination.app_private_key && (
                <SummaryRow label="Private Key" value="Configured ✓" />
              )}
            </>
          )}
        </div>
      </div>

      {/* Database Configuration */}
      <div className="mb-6">
        <Heading as="h3" className="text-lg mb-3">
          Database
        </Heading>
        <div
          className="p-4 rounded-lg border"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SummaryRow label="Type" value={config.database.type.toUpperCase()} />
          <SummaryRow label="DSN" value={maskDSN(config.database.dsn)} />
        </div>
      </div>

      {/* Server Configuration */}
      <div className="mb-6">
        <Heading as="h3" className="text-lg mb-3">
          Server
        </Heading>
        <div
          className="p-4 rounded-lg border"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SummaryRow label="Port" value={config.server.port.toString()} />
        </div>
      </div>

      {/* Migration Configuration */}
      <div className="mb-6">
        <Heading as="h3" className="text-lg mb-3">
          Migration
        </Heading>
        <div
          className="p-4 rounded-lg border"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SummaryRow label="Workers" value={config.migration.workers.toString()} />
          <SummaryRow label="Poll Interval" value={`${config.migration.poll_interval_seconds}s`} />
          <SummaryRow label="Dest Repo Exists Action" value={config.migration.dest_repo_exists_action} />
          <SummaryRow label="Public Repos Visibility" value={config.migration.visibility_handling.public_repos} />
          <SummaryRow label="Internal Repos Visibility" value={config.migration.visibility_handling.internal_repos} />
        </div>
      </div>

      {/* Logging Configuration */}
      <div className="mb-6">
        <Heading as="h3" className="text-lg mb-3">
          Logging
        </Heading>
        <div
          className="p-4 rounded-lg border"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SummaryRow label="Level" value={config.logging.level} />
          <SummaryRow label="Format" value={config.logging.format} />
          <SummaryRow label="Output File" value={config.logging.output_file} />
        </div>
      </div>

      {/* Authentication Configuration (if enabled) */}
      {config.auth?.enabled && (
        <div className="mb-6">
          <Heading as="h3" className="text-lg mb-3">
            Authentication
          </Heading>
          <div
            className="p-4 rounded-lg border"
            style={{
              backgroundColor: 'var(--bgColor-muted)',
              borderColor: 'var(--borderColor-default)',
            }}
          >
            <SummaryRow label="Enabled" value="Yes" />
            {config.auth.github_oauth_client_id && (
              <SummaryRow label="GitHub OAuth Client ID" value={config.auth.github_oauth_client_id} />
            )}
            {config.auth.azure_ad_tenant_id && (
              <SummaryRow label="Azure AD Tenant ID" value={config.auth.azure_ad_tenant_id} />
            )}
            {config.auth.azure_ad_client_id && (
              <SummaryRow label="Azure AD Client ID" value={config.auth.azure_ad_client_id} />
            )}
            {config.auth.callback_url && (
              <SummaryRow label="Callback URL" value={config.auth.callback_url} />
            )}
            {config.auth.frontend_url && (
              <SummaryRow label="Frontend URL" value={config.auth.frontend_url} />
            )}
            {config.auth.session_duration_hours && (
              <SummaryRow label="Session Duration" value={`${config.auth.session_duration_hours} hours`} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div
      className="flex justify-between py-3 border-b last:border-b-0"
      style={{ borderColor: 'var(--borderColor-subtle)' }}
    >
      <Text className="font-semibold" style={{ color: 'var(--fgColor-muted)' }}>
        {label}
      </Text>
      <Text className="font-mono text-sm">{value}</Text>
    </div>
  );
}
