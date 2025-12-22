import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSetupWizardForm, setupWizardInitialState } from './useSetupWizardForm';

describe('useSetupWizardForm', () => {
  describe('initial state', () => {
    it('should return initial state values', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      expect(result.current.state.sourceType).toBe('github');
      expect(result.current.state.sourceBaseURL).toBe('https://api.github.com');
      expect(result.current.state.sourceToken).toBe('');
      expect(result.current.state.dbType).toBe('sqlite');
      expect(result.current.state.serverPort).toBe(8080);
      expect(result.current.state.migrationWorkers).toBe(5);
      expect(result.current.state.logLevel).toBe('info');
      expect(result.current.state.authEnabled).toBe(false);
    });

    it('should match exported initial state', () => {
      const { result } = renderHook(() => useSetupWizardForm());
      expect(result.current.state).toEqual(setupWizardInitialState);
    });
  });

  describe('setField', () => {
    it('should update a single field', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setField('sourceToken', 'my-token');
      });

      expect(result.current.state.sourceToken).toBe('my-token');
    });

    it('should update source type', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setField('sourceType', 'azuredevops');
      });

      expect(result.current.state.sourceType).toBe('azuredevops');
    });

    it('should update database type', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setField('dbType', 'postgres');
      });

      expect(result.current.state.dbType).toBe('postgres');
    });

    it('should update numeric fields', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setField('serverPort', 9000);
      });

      expect(result.current.state.serverPort).toBe(9000);
    });

    it('should update boolean fields', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setField('authEnabled', true);
      });

      expect(result.current.state.authEnabled).toBe(true);
    });

    it('should preserve other fields when updating', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setField('sourceToken', 'token-1');
        result.current.setField('destToken', 'token-2');
      });

      expect(result.current.state.sourceToken).toBe('token-1');
      expect(result.current.state.destToken).toBe('token-2');
    });
  });

  describe('setFields', () => {
    it('should update multiple fields at once', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          sourceToken: 'source-token',
          destToken: 'dest-token',
          serverPort: 9090,
        });
      });

      expect(result.current.state.sourceToken).toBe('source-token');
      expect(result.current.state.destToken).toBe('dest-token');
      expect(result.current.state.serverPort).toBe(9090);
    });

    it('should preserve unspecified fields', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({ sourceToken: 'token' });
      });

      expect(result.current.state.sourceToken).toBe('token');
      expect(result.current.state.sourceType).toBe('github');
      expect(result.current.state.dbType).toBe('sqlite');
    });
  });

  describe('reset', () => {
    it('should reset all fields to initial state', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          sourceType: 'azuredevops',
          sourceToken: 'my-token',
          serverPort: 9000,
          authEnabled: true,
        });
      });

      expect(result.current.state.sourceType).toBe('azuredevops');

      act(() => {
        result.current.reset();
      });

      expect(result.current.state).toEqual(setupWizardInitialState);
    });
  });

  describe('buildConfig', () => {
    it('should build basic GitHub config', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          sourceToken: 'source-token',
          destToken: 'dest-token',
        });
      });

      const config = result.current.buildConfig();

      expect(config.source.type).toBe('github');
      expect(config.source.base_url).toBe('https://api.github.com');
      expect(config.source.token).toBe('source-token');
      expect(config.destination.token).toBe('dest-token');
    });

    it('should build Azure DevOps config with organization', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          sourceType: 'azuredevops',
          sourceBaseURL: 'https://dev.azure.com',
          sourceOrganization: 'my-org',
          sourceToken: 'ado-token',
        });
      });

      const config = result.current.buildConfig();

      expect(config.source.type).toBe('azuredevops');
      expect(config.source.organization).toBe('my-org');
    });

    it('should include database configuration', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          dbType: 'postgres',
          dbDSN: 'postgres://localhost/db',
        });
      });

      const config = result.current.buildConfig();

      expect(config.database.type).toBe('postgres');
      expect(config.database.dsn).toBe('postgres://localhost/db');
    });

    it('should include server configuration', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setField('serverPort', 9000);
      });

      const config = result.current.buildConfig();

      expect(config.server.port).toBe(9000);
    });

    it('should include migration configuration', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          migrationWorkers: 10,
          pollInterval: 60,
          destRepoExistsAction: 'skip',
          publicReposVisibility: 'internal',
          internalReposVisibility: 'private',
        });
      });

      const config = result.current.buildConfig();

      expect(config.migration.workers).toBe(10);
      expect(config.migration.poll_interval_seconds).toBe(60);
      expect(config.migration.dest_repo_exists_action).toBe('skip');
      expect(config.migration.visibility_handling?.public_repos).toBe('internal');
    });

    it('should include logging configuration', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          logLevel: 'debug',
          logFormat: 'text',
          logOutputFile: '/var/log/app.log',
        });
      });

      const config = result.current.buildConfig();

      expect(config.logging?.level).toBe('debug');
      expect(config.logging?.format).toBe('text');
      expect(config.logging?.output_file).toBe('/var/log/app.log');
    });

    it('should build config with auth disabled by default', () => {
      const { result } = renderHook(() => useSetupWizardForm());
      const config = result.current.buildConfig();

      expect(config.auth?.enabled).toBe(false);
    });

    it('should build config with GitHub OAuth when auth enabled', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          authEnabled: true,
          oauthClientID: 'client-id',
          oauthClientSecret: 'client-secret',
          callbackURL: 'http://localhost:8080/callback',
          frontendURL: 'http://localhost:3000',
          sessionSecret: 'secret',
          sessionDuration: 48,
        });
      });

      const config = result.current.buildConfig();

      expect(config.auth?.enabled).toBe(true);
      expect(config.auth?.github_oauth_client_id).toBe('client-id');
      expect(config.auth?.github_oauth_client_secret).toBe('client-secret');
      expect(config.auth?.callback_url).toBe('http://localhost:8080/callback');
      expect(config.auth?.session_duration_hours).toBe(48);
    });

    it('should build config with Azure AD when auth enabled for ADO', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          sourceType: 'azuredevops',
          authEnabled: true,
          azureADTenantID: 'tenant-id',
          azureADClientID: 'azure-client-id',
          azureADClientSecret: 'azure-client-secret',
        });
      });

      const config = result.current.buildConfig();

      expect(config.auth?.azure_ad_tenant_id).toBe('tenant-id');
      expect(config.auth?.azure_ad_client_id).toBe('azure-client-id');
    });

    it('should parse authorization rules from comma-separated strings', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          authEnabled: true,
          requireOrgMembership: 'org1, org2, org3',
          requireTeamMembership: 'team-a, team-b',
          privilegedTeams: 'admin-team, super-team',
        });
      });

      const config = result.current.buildConfig();

      expect(config.auth?.authorization_rules?.require_org_membership).toEqual(['org1', 'org2', 'org3']);
      expect(config.auth?.authorization_rules?.require_team_membership).toEqual(['team-a', 'team-b']);
      expect(config.auth?.authorization_rules?.privileged_teams).toEqual(['admin-team', 'super-team']);
    });

    it('should include GitHub App config when enabled', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          sourceGithubAppEnabled: true,
          sourceGithubAppID: '12345',
          sourceGithubAppPrivateKey: 'private-key-content',
          sourceGithubAppInstallationID: '67890',
        });
      });

      const config = result.current.buildConfig();

      expect(config.source.app_id).toBe(12345);
      expect(config.source.app_private_key).toBe('private-key-content');
      expect(config.source.app_installation_id).toBe(67890);
    });

    it('should not include GitHub App config when disabled', () => {
      const { result } = renderHook(() => useSetupWizardForm());

      act(() => {
        result.current.setFields({
          sourceGithubAppEnabled: false,
          sourceGithubAppID: '12345',
        });
      });

      const config = result.current.buildConfig();

      expect(config.source.app_id).toBeUndefined();
    });
  });
});

