/**
 * Batch-related API endpoints.
 */
import { client } from './client';
import type { Batch } from '../../types';

export const batchesApi = {
  async list(): Promise<Batch[]> {
    const { data } = await client.get('/batches');
    return data;
  },

  async get(id: number): Promise<Batch> {
    const { data } = await client.get(`/batches/${id}`);
    return data;
  },

  async create(batch: Partial<Batch>): Promise<Batch> {
    const { data } = await client.post('/batches', batch);
    return data;
  },

  async update(id: number, updates: Partial<Batch>): Promise<Batch> {
    const { data } = await client.patch(`/batches/${id}`, updates);
    return data;
  },

  async delete(id: number) {
    const { data } = await client.delete(`/batches/${id}`);
    return data;
  },

  async addRepositories(batchId: number, repositoryIds: number[]) {
    const { data } = await client.post(`/batches/${batchId}/repositories`, {
      repository_ids: repositoryIds,
    });
    return data;
  },

  async removeRepositories(batchId: number, repositoryIds: number[]) {
    const { data } = await client.delete(`/batches/${batchId}/repositories`, {
      data: { repository_ids: repositoryIds },
    });
    return data;
  },

  async retryFailures(batchId: number, repositoryIds?: number[]) {
    const { data } = await client.post(`/batches/${batchId}/retry`, {
      repository_ids: repositoryIds,
    });
    return data;
  },

  async dryRun(id: number, onlyPending?: boolean) {
    const { data } = await client.post(`/batches/${id}/dry-run`, {
      only_pending: onlyPending || false,
    });
    return data;
  },

  async start(id: number, skipDryRun?: boolean) {
    const { data } = await client.post(`/batches/${id}/start`, {
      skip_dry_run: skipDryRun || false,
    });
    return data;
  },
};

