import axios from 'axios';
import type { 
  Repository, 
  Batch, 
  Analytics, 
  RepositoryDetailResponse, 
  MigrationLogsResponse 
} from '../types';

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
});

export const api = {
  // Discovery
  async startDiscovery(organization: string) {
    const { data } = await client.post('/discovery/start', { organization });
    return data;
  },

  async getDiscoveryStatus() {
    const { data } = await client.get('/discovery/status');
    return data;
  },

  // Repositories
  async listRepositories(filters?: Record<string, string | number | undefined>): Promise<Repository[]> {
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
};

