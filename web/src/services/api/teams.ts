/**
 * Team and team mapping API endpoints.
 */
import { client } from './client';
import type {
  GitHubTeam,
  GitHubTeamMember,
  TeamMapping,
  TeamMappingStats,
  TeamDetail,
  TeamMigrationStatusResponse,
  ImportResult,
} from '../../types';

export const teamsApi = {
  // Teams
  async list(organization?: string, sourceId?: number): Promise<GitHubTeam[]> {
    const params: Record<string, string | number> = {};
    if (organization) params.organization = organization;
    if (sourceId !== undefined) params.source_id = sourceId;
    const { data } = await client.get('/teams', { params });
    return data;
  },

  async getDetail(org: string, teamSlug: string): Promise<TeamDetail> {
    const { data } = await client.get(
      `/teams/${encodeURIComponent(org)}/${encodeURIComponent(teamSlug)}`
    );
    return data;
  },

  async getMembers(
    org: string,
    teamSlug: string
  ): Promise<{ members: GitHubTeamMember[]; total: number }> {
    const { data } = await client.get(
      `/teams/${encodeURIComponent(org)}/${encodeURIComponent(teamSlug)}/members`
    );
    return data;
  },

  async discover(organization: string, sourceId?: number) {
    const { data } = await client.post('/teams/discover', { 
      organization,
      source_id: sourceId,
    });
    return data;
  },

  // Team Mappings
  async listMappings(filters?: {
    source_org?: string;
    destination_org?: string;
    source_id?: number;
    status?: string;
    has_destination?: boolean;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ mappings: TeamMapping[]; total: number }> {
    const { data } = await client.get('/team-mappings', { params: filters });
    return data;
  },

  async getMappingStats(organization?: string, sourceId?: number): Promise<TeamMappingStats> {
    const params = new URLSearchParams();
    if (organization) params.append('organization', organization);
    if (sourceId !== undefined) params.append('source_id', String(sourceId));
    const query = params.toString() ? `?${params.toString()}` : '';
    const { data } = await client.get(`/team-mappings/stats${query}`);
    return data;
  },

  async getSourceOrgs(): Promise<string[]> {
    const { data } = await client.get('/team-mappings/source-orgs');
    return data.organizations || [];
  },

  async createMapping(mapping: Partial<TeamMapping>): Promise<TeamMapping> {
    const { data } = await client.post('/team-mappings', mapping);
    return data;
  },

  async updateMapping(
    sourceOrg: string,
    sourceTeamSlug: string,
    updates: Partial<TeamMapping>
  ): Promise<TeamMapping> {
    const { data } = await client.patch(
      `/team-mappings/${encodeURIComponent(sourceOrg)}/${encodeURIComponent(sourceTeamSlug)}`,
      updates
    );
    return data;
  },

  async deleteMapping(sourceOrg: string, sourceTeamSlug: string): Promise<void> {
    await client.delete(
      `/team-mappings/${encodeURIComponent(sourceOrg)}/${encodeURIComponent(sourceTeamSlug)}`
    );
  },

  async importMappings(file: File): Promise<ImportResult> {
    const formData = new FormData();
    formData.append('file', file);
    const { data } = await client.post('/team-mappings/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return data;
  },

  async exportMappings(filters?: { status?: string; source_org?: string }): Promise<Blob> {
    const { data } = await client.get('/team-mappings/export', {
      params: filters,
      responseType: 'blob',
    });
    return data;
  },

  async suggestMappings(
    destinationOrg: string,
    destTeamSlugs?: string[]
  ): Promise<{ suggestions: unknown[]; total: number }> {
    const { data } = await client.post('/team-mappings/suggest', {
      destination_org: destinationOrg,
      dest_team_slugs: destTeamSlugs,
    });
    return data;
  },

  async syncMappings(): Promise<{ created: number; message: string }> {
    const { data } = await client.post('/team-mappings/sync');
    return data;
  },

  // Team Migration Execution
  async executeMigration(options?: {
    source_org?: string;
    source_team_slug?: string;
    dry_run?: boolean;
  }): Promise<{ message: string; dry_run: boolean; source_org?: string }> {
    const { data } = await client.post('/team-mappings/execute', options);
    return data;
  },

  async getMigrationStatus(): Promise<TeamMigrationStatusResponse> {
    const { data } = await client.get('/team-mappings/execution-status');
    return data;
  },

  async cancelMigration(): Promise<{ message: string }> {
    const { data } = await client.post('/team-mappings/cancel');
    return data;
  },

  async resetMigrationStatus(sourceOrg?: string): Promise<{ message: string }> {
    const { data } = await client.post('/team-mappings/reset', null, {
      params: sourceOrg ? { source_org: sourceOrg } : undefined,
    });
    return data;
  },
};

