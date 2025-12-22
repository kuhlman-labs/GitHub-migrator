import { describe, it, expect, vi, beforeEach } from 'vitest';
import { batchesApi } from './batches';
import { client } from './client';

// Mock the axios client
vi.mock('./client', () => ({
  client: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('batchesApi', () => {
  const mockClient = client as unknown as {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
    patch: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('list', () => {
    it('should fetch all batches', async () => {
      const mockBatches = [{ id: 1, name: 'Batch 1' }, { id: 2, name: 'Batch 2' }];
      mockClient.get.mockResolvedValue({ data: mockBatches });

      const result = await batchesApi.list();

      expect(mockClient.get).toHaveBeenCalledWith('/batches');
      expect(result).toEqual(mockBatches);
    });
  });

  describe('get', () => {
    it('should fetch a single batch by id', async () => {
      const mockBatch = { id: 1, name: 'Batch 1', repository_count: 10 };
      mockClient.get.mockResolvedValue({ data: mockBatch });

      const result = await batchesApi.get(1);

      expect(mockClient.get).toHaveBeenCalledWith('/batches/1');
      expect(result).toEqual(mockBatch);
    });
  });

  describe('create', () => {
    it('should create a new batch', async () => {
      const newBatch = { name: 'New Batch', description: 'Test batch' };
      const createdBatch = { id: 3, ...newBatch, repository_count: 0 };
      mockClient.post.mockResolvedValue({ data: createdBatch });

      const result = await batchesApi.create(newBatch);

      expect(mockClient.post).toHaveBeenCalledWith('/batches', newBatch);
      expect(result).toEqual(createdBatch);
    });
  });

  describe('update', () => {
    it('should update an existing batch', async () => {
      const updates = { name: 'Updated Name' };
      const updatedBatch = { id: 1, name: 'Updated Name', repository_count: 5 };
      mockClient.patch.mockResolvedValue({ data: updatedBatch });

      const result = await batchesApi.update(1, updates);

      expect(mockClient.patch).toHaveBeenCalledWith('/batches/1', updates);
      expect(result).toEqual(updatedBatch);
    });
  });

  describe('delete', () => {
    it('should delete a batch', async () => {
      mockClient.delete.mockResolvedValue({ data: { message: 'Deleted' } });

      const result = await batchesApi.delete(1);

      expect(mockClient.delete).toHaveBeenCalledWith('/batches/1');
      expect(result).toEqual({ message: 'Deleted' });
    });
  });

  describe('addRepositories', () => {
    it('should add repositories to a batch', async () => {
      mockClient.post.mockResolvedValue({ data: { added: 3 } });

      const result = await batchesApi.addRepositories(1, [10, 20, 30]);

      expect(mockClient.post).toHaveBeenCalledWith('/batches/1/repositories', {
        repository_ids: [10, 20, 30],
      });
      expect(result).toEqual({ added: 3 });
    });
  });

  describe('removeRepositories', () => {
    it('should remove repositories from a batch', async () => {
      mockClient.delete.mockResolvedValue({ data: { removed: 2 } });

      const result = await batchesApi.removeRepositories(1, [10, 20]);

      expect(mockClient.delete).toHaveBeenCalledWith('/batches/1/repositories', {
        data: { repository_ids: [10, 20] },
      });
      expect(result).toEqual({ removed: 2 });
    });
  });

  describe('retryFailures', () => {
    it('should retry all failed repositories', async () => {
      mockClient.post.mockResolvedValue({ data: { retried: 5 } });

      const result = await batchesApi.retryFailures(1);

      expect(mockClient.post).toHaveBeenCalledWith('/batches/1/retry', {
        repository_ids: undefined,
      });
      expect(result).toEqual({ retried: 5 });
    });

    it('should retry specific repositories', async () => {
      mockClient.post.mockResolvedValue({ data: { retried: 2 } });

      const result = await batchesApi.retryFailures(1, [10, 20]);

      expect(mockClient.post).toHaveBeenCalledWith('/batches/1/retry', {
        repository_ids: [10, 20],
      });
      expect(result).toEqual({ retried: 2 });
    });
  });

  describe('dryRun', () => {
    it('should start dry run for all repositories', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 10 } });

      const result = await batchesApi.dryRun(1);

      expect(mockClient.post).toHaveBeenCalledWith('/batches/1/dry-run', {
        only_pending: false,
      });
      expect(result).toEqual({ started: 10 });
    });

    it('should start dry run for only pending repositories', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 5 } });

      const result = await batchesApi.dryRun(1, true);

      expect(mockClient.post).toHaveBeenCalledWith('/batches/1/dry-run', {
        only_pending: true,
      });
      expect(result).toEqual({ started: 5 });
    });
  });

  describe('start', () => {
    it('should start batch migration', async () => {
      mockClient.post.mockResolvedValue({ data: { message: 'Started' } });

      const result = await batchesApi.start(1);

      expect(mockClient.post).toHaveBeenCalledWith('/batches/1/start', {
        skip_dry_run: false,
      });
      expect(result).toEqual({ message: 'Started' });
    });

    it('should start batch migration skipping dry run', async () => {
      mockClient.post.mockResolvedValue({ data: { message: 'Started' } });

      const result = await batchesApi.start(1, true);

      expect(mockClient.post).toHaveBeenCalledWith('/batches/1/start', {
        skip_dry_run: true,
      });
      expect(result).toEqual({ message: 'Started' });
    });
  });
});

