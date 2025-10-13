import axios from 'axios';
import type { 
  Repository, 
  Batch, 
  Analytics, 
  RepositoryDetailResponse, 
  MigrationLogsResponse,
  Organization,
  MigrationHistoryEntry 
} from '../types';

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
});

export const api = {
  // Discovery
  async startDiscovery(params: { organization?: string; enterprise_slug?: string; workers?: number }) {
    const { data } = await client.post('/discovery/start', params);
    return data;
  },

  async getDiscoveryStatus() {
    const { data } = await client.get('/discovery/status');
    return data;
  },

  // Repositories
  async listRepositories(filters?: Record<string, string | number | boolean | undefined>): Promise<Repository[]> {
    const { data } = await client.get('/repositories', { params: filters });
    return data;
  },

  async getRepository(fullName: string): Promise<RepositoryDetailResponse> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}`);
    return data;
  },

  async updateRepository(fullName: string, updates: Partial<Repository>) {
    const { data } = await client.patch(`/repositories/${encodeURIComponent(fullName)}`, updates);
    return data;
  },

  async rediscoverRepository(fullName: string) {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/rediscover`);
    return data;
  },

  // Organizations
  async listOrganizations(): Promise<Organization[]> {
    const { data } = await client.get('/organizations');
    return data;
  },

  // Batches
  async listBatches(): Promise<Batch[]> {
    const { data } = await client.get('/batches');
    return data;
  },

  async getBatch(id: number): Promise<Batch> {
    const { data } = await client.get(`/batches/${id}`);
    return data;
  },

  async createBatch(batch: Partial<Batch>): Promise<Batch> {
    const { data } = await client.post('/batches', batch);
    return data;
  },

  async updateBatch(id: number, updates: Partial<Batch>): Promise<Batch> {
    const { data } = await client.patch(`/batches/${id}`, updates);
    return data;
  },

  async addRepositoriesToBatch(batchId: number, repositoryIds: number[]) {
    const { data } = await client.post(`/batches/${batchId}/repositories`, {
      repository_ids: repositoryIds,
    });
    return data;
  },

  async removeRepositoriesFromBatch(batchId: number, repositoryIds: number[]) {
    const { data } = await client.delete(`/batches/${batchId}/repositories`, {
      data: { repository_ids: repositoryIds },
    });
    return data;
  },

  async retryBatchFailures(batchId: number, repositoryIds?: number[]) {
    const { data } = await client.post(`/batches/${batchId}/retry`, {
      repository_ids: repositoryIds,
    });
    return data;
  },

  async startBatch(id: number) {
    const { data } = await client.post(`/batches/${id}/start`);
    return data;
  },

  // Migrations
  async startMigration(params: { 
    repository_ids?: number[];
    full_names?: string[];
    dry_run?: boolean;
    priority?: number;
  }) {
    const { data } = await client.post('/migrations/start', params);
    return data;
  },

  async getMigrationStatus(repositoryId: number) {
    const { data } = await client.get(`/migrations/${repositoryId}`);
    return data;
  },

  async getMigrationHistory(repositoryId: number) {
    const { data } = await client.get(`/migrations/${repositoryId}/history`);
    return data;
  },

  async getMigrationLogs(
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

  // Analytics
  async getAnalyticsSummary(): Promise<Analytics> {
    const { data } = await client.get('/analytics/summary');
    return data;
  },

  async getMigrationProgress() {
    const { data } = await client.get('/analytics/progress');
    return data;
  },

  // Migration History
  async getMigrationHistoryList(): Promise<{ migrations: MigrationHistoryEntry[]; total: number }> {
    const { data } = await client.get('/migrations/history');
    return data;
  },

  async exportMigrationHistory(format: 'csv' | 'json'): Promise<Blob> {
    const { data } = await client.get(`/migrations/history/export?format=${format}`, {
      responseType: 'blob',
    });
    return data;
  },
};

