/**
 * Configuration, setup, and authentication API endpoints.
 */
import { client } from './client';
import type { SetupStatus, SetupConfig, ValidationResult } from '../../types';

export const configApi = {
  // Configuration
  async getConfig(): Promise<{
    source_type: 'github' | 'azuredevops';
    auth_enabled: boolean;
  }> {
    const { data } = await client.get('/config');
    return data;
  },

  // Authentication
  async getAuthConfig(): Promise<{
    enabled: boolean;
    login_url?: string;
    authorization_rules?: {
      requires_org_membership?: boolean;
      required_orgs?: string[];
      requires_team_membership?: boolean;
      required_teams?: string[];
      requires_enterprise_admin?: boolean;
      requires_enterprise_membership?: boolean;
      enterprise?: string;
    };
  }> {
    const { data } = await client.get('/auth/config');
    return data;
  },

  async getCurrentUser(): Promise<{
    id: number;
    login: string;
    name: string;
    email: string;
    avatar_url: string;
    roles?: string[];
  }> {
    const { data } = await client.get('/auth/user');
    return data;
  },

  async logout(): Promise<void> {
    await client.post('/auth/logout');
  },

  async refreshToken(): Promise<void> {
    await client.post('/auth/refresh');
  },

  /** Get current user's authorization status including their tier */
  async getAuthorizationStatus(): Promise<{
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
      status: string;
      source_login?: string;
      destination_login?: string;
    };
    upgrade_path?: string;
  }> {
    const { data } = await client.get('/auth/authorization-status');
    return data;
  },

  // Setup
  async getSetupStatus(): Promise<SetupStatus> {
    const { data } = await client.get('/setup/status');
    return data;
  },

  async validateSourceConnection(config: SetupConfig['source']): Promise<ValidationResult> {
    const { data } = await client.post('/setup/validate-source', config);
    return data;
  },

  async validateDestinationConnection(config: SetupConfig['destination']): Promise<ValidationResult> {
    const { data } = await client.post('/setup/validate-destination', config);
    return data;
  },

  async validateDatabaseConnection(config: SetupConfig['database']): Promise<ValidationResult> {
    const { data } = await client.post('/setup/validate-database', config);
    return data;
  },

  async applySetup(config: SetupConfig): Promise<void> {
    const { data } = await client.post('/setup/apply', config);
    return data;
  },
};

