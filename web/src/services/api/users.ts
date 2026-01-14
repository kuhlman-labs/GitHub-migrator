/**
 * User and user mapping API endpoints.
 */
import { client } from './client';
import type {
  GitHubUser,
  UserMapping,
  UserMappingStats,
  UserStats,
  UserDetail,
  ImportResult,
} from '../../types';

export const usersApi = {
  // Users
  async list(filters?: {
    source_instance?: string;
    source_id?: number;
    limit?: number;
    offset?: number;
  }): Promise<{ users: GitHubUser[]; total: number }> {
    const { data } = await client.get('/users', { params: filters });
    return data;
  },

  async getStats(sourceId?: number): Promise<UserStats> {
    const params = sourceId !== undefined ? { source_id: sourceId } : undefined;
    const { data } = await client.get('/users/stats', { params });
    return data;
  },

  async discover(organization: string, sourceId?: number) {
    const { data } = await client.post('/users/discover', { 
      organization,
      source_id: sourceId,
    });
    return data;
  },

  // User Mappings
  async listMappings(filters?: {
    status?: string;
    source_org?: string;
    source_id?: number;
    has_destination?: boolean;
    has_mannequin?: boolean;
    reclaim_status?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ mappings: UserMapping[]; total: number }> {
    const { data } = await client.get('/user-mappings', { params: filters });
    return data;
  },

  async getMappingStats(sourceOrg?: string, sourceId?: number): Promise<UserMappingStats> {
    const params = new URLSearchParams();
    if (sourceOrg) params.append('source_org', sourceOrg);
    if (sourceId !== undefined) params.append('source_id', String(sourceId));
    const query = params.toString() ? `?${params.toString()}` : '';
    const { data } = await client.get(`/user-mappings/stats${query}`);
    return data;
  },

  async getDetail(login: string): Promise<UserDetail> {
    const { data } = await client.get(`/user-mappings/${encodeURIComponent(login)}`);
    return data;
  },

  async getSourceOrgs(): Promise<{ organizations: string[] }> {
    const { data } = await client.get('/user-mappings/source-orgs');
    return data;
  },

  async createMapping(mapping: Partial<UserMapping>): Promise<UserMapping> {
    const { data } = await client.post('/user-mappings', mapping);
    return data;
  },

  async updateMapping(sourceLogin: string, updates: Partial<UserMapping>): Promise<UserMapping> {
    const { data } = await client.patch(`/user-mappings/${encodeURIComponent(sourceLogin)}`, updates);
    return data;
  },

  async deleteMapping(sourceLogin: string): Promise<void> {
    await client.delete(`/user-mappings/${encodeURIComponent(sourceLogin)}`);
  },

  async importMappings(file: File): Promise<ImportResult> {
    const formData = new FormData();
    formData.append('file', file);
    const { data } = await client.post('/user-mappings/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return data;
  },

  async exportMappings(status?: string, sourceId?: number): Promise<Blob> {
    const { data } = await client.get('/user-mappings/export', {
      params: { status, source_id: sourceId },
      responseType: 'blob',
    });
    return data;
  },

  async generateGEICSV(org: string, status?: string): Promise<Blob> {
    const { data } = await client.get('/user-mappings/generate-gei-csv', {
      params: { org, status },
      responseType: 'blob',
    });
    return data;
  },

  async getMannequinOrgs(): Promise<{ orgs: string[] }> {
    const { data } = await client.get('/user-mappings/mannequin-orgs');
    return data;
  },

  async suggestMappings(): Promise<{ suggestions: unknown[]; total: number }> {
    const { data } = await client.post('/user-mappings/suggest');
    return data;
  },

  async syncMappings(): Promise<{ created: number; message: string }> {
    const { data } = await client.post('/user-mappings/sync');
    return data;
  },

  async fetchMannequins(
    destinationOrg: string,
    emuShortcode?: string
  ): Promise<{
    total_mannequins: number;
    total_dest_members: number;
    matched: number;
    unmatched: number;
    destination_org: string;
    emu_shortcode_applied: boolean;
    message: string;
  }> {
    const { data } = await client.post('/user-mappings/fetch-mannequins', {
      destination_org: destinationOrg,
      emu_shortcode: emuShortcode || undefined,
    });
    return data;
  },

  async sendAttributionInvitation(
    sourceLogin: string,
    destinationOrg: string
  ): Promise<{
    success: boolean;
    source_login: string;
    mannequin_login?: string;
    target_user?: string;
    message: string;
  }> {
    const { data } = await client.post(
      `/user-mappings/${encodeURIComponent(sourceLogin)}/send-invitation`,
      { destination_org: destinationOrg }
    );
    return data;
  },

  async bulkSendAttributionInvitations(
    destinationOrg: string,
    sourceLogins?: string[]
  ): Promise<{
    success: boolean;
    invited: number;
    failed: number;
    skipped: number;
    errors: string[];
    message: string;
  }> {
    const { data } = await client.post('/user-mappings/send-invitations', {
      destination_org: destinationOrg,
      source_logins: sourceLogins,
    });
    return data;
  },
};

