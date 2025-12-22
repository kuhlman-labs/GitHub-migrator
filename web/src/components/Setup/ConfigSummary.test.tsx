import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { ConfigSummary } from './ConfigSummary';
import type { SetupConfig } from '../../types';

const baseConfig: SetupConfig = {
  source: {
    type: 'github',
    base_url: 'https://api.github.com',
    token: 'ghp_abcdefghijklmnop',
  },
  destination: {
    base_url: 'https://github.company.com/api/v3',
    token: 'ghp_zyxwvutsrqponmlk',
  },
  database: {
    type: 'sqlite',
    dsn: './data/migrator.db',
  },
  server: {
    port: 8080,
  },
  migration: {
    workers: 5,
    poll_interval_seconds: 30,
    dest_repo_exists_action: 'fail',
    visibility_handling: {
      public_repos: 'private',
      internal_repos: 'private',
    },
  },
  logging: {
    level: 'info',
    format: 'json',
    output_file: './logs/migrator.log',
  },
};

describe('ConfigSummary', () => {
  it('renders source configuration', () => {
    render(<ConfigSummary config={baseConfig} />);

    expect(screen.getByText('Source')).toBeInTheDocument();
    expect(screen.getByText('GITHUB')).toBeInTheDocument();
    expect(screen.getByText('https://api.github.com')).toBeInTheDocument();
    // Token should be masked
    expect(screen.getByText('****mnop')).toBeInTheDocument();
  });

  it('renders destination configuration', () => {
    render(<ConfigSummary config={baseConfig} />);

    expect(screen.getByText('Destination')).toBeInTheDocument();
    expect(screen.getByText('https://github.company.com/api/v3')).toBeInTheDocument();
  });

  it('renders database configuration', () => {
    render(<ConfigSummary config={baseConfig} />);

    expect(screen.getByText('Database')).toBeInTheDocument();
    expect(screen.getByText('SQLITE')).toBeInTheDocument();
    expect(screen.getByText('./data/migrator.db')).toBeInTheDocument();
  });

  it('renders server configuration', () => {
    render(<ConfigSummary config={baseConfig} />);

    expect(screen.getByText('Server')).toBeInTheDocument();
    expect(screen.getByText('8080')).toBeInTheDocument();
  });

  it('renders migration configuration', () => {
    render(<ConfigSummary config={baseConfig} />);

    expect(screen.getByText('Migration')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('30s')).toBeInTheDocument();
    expect(screen.getByText('fail')).toBeInTheDocument();
  });

  it('renders logging configuration', () => {
    render(<ConfigSummary config={baseConfig} />);

    expect(screen.getByText('Logging')).toBeInTheDocument();
    expect(screen.getByText('info')).toBeInTheDocument();
    expect(screen.getByText('json')).toBeInTheDocument();
    expect(screen.getByText('./logs/migrator.log')).toBeInTheDocument();
  });

  it('masks postgres DSN password', () => {
    const configWithPostgres: SetupConfig = {
      ...baseConfig,
      database: {
        type: 'postgres',
        dsn: 'postgres://user:secretpassword@localhost:5432/migrator',
      },
    };

    render(<ConfigSummary config={configWithPostgres} />);

    // Check that the masked DSN is displayed (format: protocol:****@host)
    expect(screen.getByText(/postgres:\*\*\*\*@localhost:5432\/migrator/)).toBeInTheDocument();
    // Ensure the actual password is not visible
    expect(screen.queryByText(/secretpassword/)).not.toBeInTheDocument();
  });

  it('masks short tokens correctly', () => {
    const configWithShortToken: SetupConfig = {
      ...baseConfig,
      source: {
        ...baseConfig.source,
        token: 'abc',
      },
    };

    render(<ConfigSummary config={configWithShortToken} />);

    // Short tokens should just show ****
    expect(screen.getByText('****')).toBeInTheDocument();
  });

  it('renders source organization when present', () => {
    const configWithOrg: SetupConfig = {
      ...baseConfig,
      source: {
        ...baseConfig.source,
        organization: 'my-org',
      },
    };

    render(<ConfigSummary config={configWithOrg} />);

    expect(screen.getByText('Organization')).toBeInTheDocument();
    expect(screen.getByText('my-org')).toBeInTheDocument();
  });

  it('renders source GitHub App configuration when present', () => {
    const configWithApp: SetupConfig = {
      ...baseConfig,
      source: {
        ...baseConfig.source,
        app_id: 12345,
        app_installation_id: 67890,
        app_private_key: '-----BEGIN RSA PRIVATE KEY-----...',
      },
    };

    render(<ConfigSummary config={configWithApp} />);

    expect(screen.getByText('GitHub App (Source)')).toBeInTheDocument();
    expect(screen.getByText('12345')).toBeInTheDocument();
    expect(screen.getByText('67890')).toBeInTheDocument();
    expect(screen.getByText('Configured ✓')).toBeInTheDocument();
  });

  it('renders destination GitHub App configuration when present', () => {
    const configWithApp: SetupConfig = {
      ...baseConfig,
      destination: {
        ...baseConfig.destination,
        app_id: 54321,
        app_installation_id: 98765,
        app_private_key: '-----BEGIN RSA PRIVATE KEY-----...',
      },
    };

    render(<ConfigSummary config={configWithApp} />);

    expect(screen.getByText('GitHub App (Destination)')).toBeInTheDocument();
    expect(screen.getByText('54321')).toBeInTheDocument();
    expect(screen.getByText('98765')).toBeInTheDocument();
    expect(screen.getAllByText('Configured ✓').length).toBeGreaterThanOrEqual(1);
  });

  it('renders authentication configuration when enabled', () => {
    const configWithAuth: SetupConfig = {
      ...baseConfig,
      auth: {
        enabled: true,
        github_oauth_client_id: 'Iv1.abc123',
        callback_url: 'http://localhost:8080/api/v1/auth/callback',
        frontend_url: 'http://localhost:3000',
        session_duration_hours: 24,
      },
    };

    render(<ConfigSummary config={configWithAuth} />);

    expect(screen.getByText('Authentication')).toBeInTheDocument();
    expect(screen.getByText('Iv1.abc123')).toBeInTheDocument();
    expect(screen.getByText('http://localhost:8080/api/v1/auth/callback')).toBeInTheDocument();
    expect(screen.getByText('http://localhost:3000')).toBeInTheDocument();
    expect(screen.getByText('24 hours')).toBeInTheDocument();
  });

  it('renders Azure AD authentication configuration', () => {
    const configWithAzureAD: SetupConfig = {
      ...baseConfig,
      auth: {
        enabled: true,
        azure_ad_tenant_id: 'tenant-id-123',
        azure_ad_client_id: 'client-id-456',
        callback_url: 'http://localhost:8080/api/v1/auth/callback',
      },
    };

    render(<ConfigSummary config={configWithAzureAD} />);

    expect(screen.getByText('Authentication')).toBeInTheDocument();
    expect(screen.getByText('tenant-id-123')).toBeInTheDocument();
    expect(screen.getByText('client-id-456')).toBeInTheDocument();
  });

  it('renders authorization rules when present', () => {
    const configWithAuthRules: SetupConfig = {
      ...baseConfig,
      auth: {
        enabled: true,
        authorization_rules: {
          require_org_membership: ['org-1', 'org-2'],
          require_team_membership: ['org-1/admins', 'org-2/devs'],
          require_enterprise_admin: true,
          require_enterprise_membership: true,
          enterprise_slug: 'my-enterprise',
          privileged_teams: ['org-1/super-admins'],
        },
      },
    };

    render(<ConfigSummary config={configWithAuthRules} />);

    expect(screen.getByText('Authorization Rules')).toBeInTheDocument();
    expect(screen.getByText('org-1, org-2')).toBeInTheDocument();
    expect(screen.getByText('org-1/admins, org-2/devs')).toBeInTheDocument();
    expect(screen.getAllByText('Yes').length).toBe(3); // Enabled, enterprise admin, enterprise membership
    expect(screen.getByText('my-enterprise')).toBeInTheDocument();
    expect(screen.getByText('org-1/super-admins')).toBeInTheDocument();
  });

  it('does not render authentication section when not enabled', () => {
    render(<ConfigSummary config={baseConfig} />);

    expect(screen.queryByText('Authentication')).not.toBeInTheDocument();
  });
});

