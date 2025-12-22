/**
 * Migration execution and history API endpoints.
 */
import { client } from './client';
import type { MigrationLogsResponse, MigrationHistoryEntry } from '../../types';

export const migrationsApi = {
  async start(params: {
    repository_ids?: number[];
    full_names?: string[];
    dry_run?: boolean;
    priority?: number;
  }) {
    const { data } = await client.post('/migrations/start', params);
    return data;
  },

  async retryRepository(repositoryId: number, dryRun: boolean = false) {
    const { data } = await client.post('/migrations/start', {
      repository_ids: [repositoryId],
      dry_run: dryRun,
      priority: 0,
    });
    return data;
  },

  async getStatus(repositoryId: number) {
    const { data } = await client.get(`/migrations/${repositoryId}`);
    return data;
  },

  async getHistory(repositoryId: number) {
    const { data } = await client.get(`/migrations/${repositoryId}/history`);
    return data;
  },

  async getLogs(
    repositoryId: number,
    params?: {
      level?: string;
      phase?: string;
      limit?: number;
      offset?: number;
    }
  ): Promise<MigrationLogsResponse> {
    const { data } = await client.get(`/migrations/${repositoryId}/logs`, { params });
    return data;
  },

  async getHistoryList(): Promise<{ migrations: MigrationHistoryEntry[]; total: number }> {
    const { data } = await client.get('/migrations/history');
    return data;
  },

  async exportHistory(format: 'csv' | 'json'): Promise<Blob> {
    const { data } = await client.get(`/migrations/history/export?format=${format}`, {
      responseType: 'blob',
    });
    return data;
  },

  // Self-Service Migration
  async selfService(params: {
    repositories: string[];
    mappings?: Record<string, string>;
    dry_run: boolean;
  }): Promise<{
    batch_id: number;
    batch_name: string;
    message: string;
    total_repositories: number;
    newly_discovered: number;
    already_existed: number;
    discovery_errors?: string[];
    execution_started: boolean;
  }> {
    const { data } = await client.post('/self-service/migrate', params);
    return data;
  },
};

