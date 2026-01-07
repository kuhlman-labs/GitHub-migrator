import { describe, it, expect, vi, beforeEach } from 'vitest';
import { migrationsApi } from './migrations';
import { client } from './client';

// Mock the axios client
vi.mock('./client', () => ({
  client: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('migrationsApi', () => {
  const mockClient = client as unknown as {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('start', () => {
    it('should start migration by repository IDs', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 3 } });

      const result = await migrationsApi.start({ repository_ids: [1, 2, 3] });

      expect(mockClient.post).toHaveBeenCalledWith('/migrations/start', {
        repository_ids: [1, 2, 3],
      });
      expect(result).toEqual({ started: 3 });
    });

    it('should start migration by full names', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 2 } });

      const result = await migrationsApi.start({ full_names: ['org/repo1', 'org/repo2'] });

      expect(mockClient.post).toHaveBeenCalledWith('/migrations/start', {
        full_names: ['org/repo1', 'org/repo2'],
      });
      expect(result).toEqual({ started: 2 });
    });

    it('should start dry run migration', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 1 } });

      const result = await migrationsApi.start({ repository_ids: [1], dry_run: true });

      expect(mockClient.post).toHaveBeenCalledWith('/migrations/start', {
        repository_ids: [1],
        dry_run: true,
      });
      expect(result).toEqual({ started: 1 });
    });

    it('should start migration with priority', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 1 } });

      const result = await migrationsApi.start({ repository_ids: [1], priority: 10 });

      expect(mockClient.post).toHaveBeenCalledWith('/migrations/start', {
        repository_ids: [1],
        priority: 10,
      });
      expect(result).toEqual({ started: 1 });
    });
  });

  describe('retryRepository', () => {
    it('should retry a repository migration', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 1 } });

      const result = await migrationsApi.retryRepository(123);

      expect(mockClient.post).toHaveBeenCalledWith('/migrations/start', {
        repository_ids: [123],
        dry_run: false,
        priority: 0,
      });
      expect(result).toEqual({ started: 1 });
    });

    it('should retry a repository as dry run', async () => {
      mockClient.post.mockResolvedValue({ data: { started: 1 } });

      const result = await migrationsApi.retryRepository(123, true);

      expect(mockClient.post).toHaveBeenCalledWith('/migrations/start', {
        repository_ids: [123],
        dry_run: true,
        priority: 0,
      });
      expect(result).toEqual({ started: 1 });
    });
  });

  describe('getStatus', () => {
    it('should get migration status for a repository', async () => {
      const mockStatus = { status: 'in_progress', phase: 'archive_generating' };
      mockClient.get.mockResolvedValue({ data: mockStatus });

      const result = await migrationsApi.getStatus(123);

      expect(mockClient.get).toHaveBeenCalledWith('/migrations/123');
      expect(result).toEqual(mockStatus);
    });
  });

  describe('getHistory', () => {
    it('should get migration history for a repository', async () => {
      const mockHistory = [{ id: 1, status: 'complete' }, { id: 2, status: 'failed' }];
      mockClient.get.mockResolvedValue({ data: mockHistory });

      const result = await migrationsApi.getHistory(123);

      expect(mockClient.get).toHaveBeenCalledWith('/migrations/123/history');
      expect(result).toEqual(mockHistory);
    });
  });

  describe('getLogs', () => {
    it('should get migration logs without params', async () => {
      const mockLogs = { logs: [], total: 0 };
      mockClient.get.mockResolvedValue({ data: mockLogs });

      const result = await migrationsApi.getLogs(123);

      expect(mockClient.get).toHaveBeenCalledWith('/migrations/123/logs', { params: undefined });
      expect(result).toEqual(mockLogs);
    });

    it('should get migration logs with filters', async () => {
      const mockLogs = { logs: [{ id: 1, level: 'ERROR' }], total: 1 };
      mockClient.get.mockResolvedValue({ data: mockLogs });

      const result = await migrationsApi.getLogs(123, {
        level: 'ERROR',
        phase: 'archive_generating',
        limit: 50,
        offset: 0,
      });

      expect(mockClient.get).toHaveBeenCalledWith('/migrations/123/logs', {
        params: { level: 'ERROR', phase: 'archive_generating', limit: 50, offset: 0 },
      });
      expect(result).toEqual(mockLogs);
    });
  });

  describe('getHistoryList', () => {
    it('should get migration history list', async () => {
      const mockData = { migrations: [{ id: 1, full_name: 'org/repo' }], total: 1 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await migrationsApi.getHistoryList();

      expect(mockClient.get).toHaveBeenCalledWith('/migrations/history', { params: undefined });
      expect(result).toEqual(mockData);
    });
  });

  describe('exportHistory', () => {
    it('should export migration history as CSV', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await migrationsApi.exportHistory('csv');

      expect(mockClient.get).toHaveBeenCalledWith('/migrations/history/export', {
        params: { format: 'csv' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });

    it('should export migration history as JSON', async () => {
      const mockBlob = new Blob(['{}']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await migrationsApi.exportHistory('json');

      expect(mockClient.get).toHaveBeenCalledWith('/migrations/history/export', {
        params: { format: 'json' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });
  });

  describe('selfService', () => {
    it('should start self-service migration', async () => {
      const mockResponse = {
        batch_id: 5,
        batch_name: 'Self-Service Batch',
        message: 'Migration started',
        total_repositories: 3,
        newly_discovered: 2,
        already_existed: 1,
        execution_started: true,
      };
      mockClient.post.mockResolvedValue({ data: mockResponse });

      const result = await migrationsApi.selfService({
        repositories: ['org/repo1', 'org/repo2', 'org/repo3'],
        dry_run: false,
      });

      expect(mockClient.post).toHaveBeenCalledWith('/self-service/migrate', {
        repositories: ['org/repo1', 'org/repo2', 'org/repo3'],
        dry_run: false,
      });
      expect(result).toEqual(mockResponse);
    });

    it('should start self-service migration with mappings', async () => {
      const mockResponse = {
        batch_id: 5,
        batch_name: 'Self-Service Batch',
        message: 'Migration started',
        total_repositories: 1,
        newly_discovered: 1,
        already_existed: 0,
        execution_started: true,
      };
      mockClient.post.mockResolvedValue({ data: mockResponse });

      const result = await migrationsApi.selfService({
        repositories: ['org/repo1'],
        mappings: { 'org/repo1': 'new-org/repo1' },
        dry_run: true,
      });

      expect(mockClient.post).toHaveBeenCalledWith('/self-service/migrate', {
        repositories: ['org/repo1'],
        mappings: { 'org/repo1': 'new-org/repo1' },
        dry_run: true,
      });
      expect(result).toEqual(mockResponse);
    });
  });
});

