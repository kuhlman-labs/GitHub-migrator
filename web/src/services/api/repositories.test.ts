import { describe, it, expect, vi, beforeEach } from 'vitest';
import { repositoriesApi } from './repositories';
import { client } from './client';

// Mock the axios client
vi.mock('./client', () => ({
  client: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  },
}));

describe('repositoriesApi', () => {
  const mockClient = client as unknown as {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
    patch: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('list', () => {
    it('should fetch repositories without filters', async () => {
      mockClient.get.mockResolvedValue({ data: { repositories: [] } });

      const result = await repositoriesApi.list();

      expect(mockClient.get).toHaveBeenCalledWith('/repositories', { params: {} });
      expect(result).toEqual({ repositories: [] });
    });

    it('should handle array response format', async () => {
      mockClient.get.mockResolvedValue({ data: [{ id: 1, full_name: 'org/repo' }] });

      const result = await repositoriesApi.list();

      expect(result).toEqual({ repositories: [{ id: 1, full_name: 'org/repo' }] });
    });

    it('should join array filters with commas', async () => {
      mockClient.get.mockResolvedValue({ data: { repositories: [] } });

      await repositoriesApi.list({
        status: ['pending', 'complete'],
        organization: ['org1', 'org2'],
      });

      expect(mockClient.get).toHaveBeenCalledWith('/repositories', {
        params: expect.objectContaining({
          status: 'pending,complete',
          organization: 'org1,org2',
        }),
      });
    });

    it('should handle all array filter types', async () => {
      mockClient.get.mockResolvedValue({ data: { repositories: [] } });

      await repositoriesApi.list({
        status: ['pending'],
        organization: ['org1'],
        ado_organization: ['ado-org'],
        project: ['project1'],
        team: ['team1'],
        complexity: ['low', 'medium'],
        size_category: ['small', 'large'],
      });

      expect(mockClient.get).toHaveBeenCalledWith('/repositories', {
        params: expect.objectContaining({
          status: 'pending',
          organization: 'org1',
          ado_organization: 'ado-org',
          project: 'project1',
          team: 'team1',
          complexity: 'low,medium',
          size_category: 'small,large',
        }),
      });
    });
  });

  describe('get', () => {
    it('should fetch a repository by full name', async () => {
      const mockRepo = { repository: { id: 1, full_name: 'org/repo' }, history: [] };
      mockClient.get.mockResolvedValue({ data: mockRepo });

      const result = await repositoriesApi.get('org/repo');

      expect(mockClient.get).toHaveBeenCalledWith('/repositories/org%2Frepo');
      expect(result).toEqual(mockRepo);
    });

    it('should encode special characters in full name', async () => {
      mockClient.get.mockResolvedValue({ data: {} });

      await repositoriesApi.get('org/repo-with-special');

      expect(mockClient.get).toHaveBeenCalledWith('/repositories/org%2Frepo-with-special');
    });
  });

  describe('update', () => {
    it('should update repository with partial data', async () => {
      mockClient.patch.mockResolvedValue({ data: { id: 1, status: 'complete' } });

      const result = await repositoriesApi.update('org/repo', { status: 'complete' });

      expect(mockClient.patch).toHaveBeenCalledWith('/repositories/org%2Frepo', { status: 'complete' });
      expect(result).toEqual({ id: 1, status: 'complete' });
    });
  });

  describe('rediscover', () => {
    it('should trigger repository rediscovery', async () => {
      mockClient.post.mockResolvedValue({ data: { message: 'Rediscovery started' } });

      const result = await repositoriesApi.rediscover('org/repo');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/org%2Frepo/rediscover');
      expect(result).toEqual({ message: 'Rediscovery started' });
    });
  });

  describe('unlock', () => {
    it('should unlock a repository', async () => {
      mockClient.post.mockResolvedValue({ data: { message: 'Unlocked' } });

      const result = await repositoriesApi.unlock('org/repo');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/org%2Frepo/unlock');
      expect(result).toEqual({ message: 'Unlocked' });
    });
  });

  describe('rollback', () => {
    it('should rollback repository with reason', async () => {
      const mockRepo = { id: 1, status: 'pending' };
      mockClient.post.mockResolvedValue({ data: { repository: mockRepo } });

      const result = await repositoriesApi.rollback('org/repo', 'Need to redo');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/org%2Frepo/rollback', {
        reason: 'Need to redo',
      });
      expect(result).toEqual(mockRepo);
    });

    it('should rollback repository without reason', async () => {
      const mockRepo = { id: 1, status: 'pending' };
      mockClient.post.mockResolvedValue({ data: { repository: mockRepo } });

      await repositoriesApi.rollback('org/repo');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/org%2Frepo/rollback', {
        reason: '',
      });
    });
  });

  describe('getDependencies', () => {
    it('should fetch repository dependencies', async () => {
      mockClient.get.mockResolvedValue({ data: { dependencies: [] } });

      const result = await repositoriesApi.getDependencies('org/repo');

      expect(mockClient.get).toHaveBeenCalledWith('/repositories/org%2Frepo/dependencies');
      expect(result).toEqual({ dependencies: [] });
    });
  });

  describe('getDependents', () => {
    it('should fetch repository dependents', async () => {
      mockClient.get.mockResolvedValue({ data: { dependents: [] } });

      const result = await repositoriesApi.getDependents('org/repo');

      expect(mockClient.get).toHaveBeenCalledWith('/repositories/org%2Frepo/dependents');
      expect(result).toEqual({ dependents: [] });
    });
  });

  describe('getDependencyGraph', () => {
    it('should fetch dependency graph', async () => {
      mockClient.get.mockResolvedValue({ data: { nodes: [], links: [] } });

      const result = await repositoriesApi.getDependencyGraph();

      expect(mockClient.get).toHaveBeenCalledWith('/dependencies/graph', { params: undefined });
      expect(result).toEqual({ nodes: [], links: [] });
    });

    it('should fetch dependency graph with type filter', async () => {
      mockClient.get.mockResolvedValue({ data: { nodes: [], links: [] } });

      await repositoriesApi.getDependencyGraph({ dependency_type: 'npm' });

      expect(mockClient.get).toHaveBeenCalledWith('/dependencies/graph', {
        params: { dependency_type: 'npm' },
      });
    });
  });

  describe('exportDependencies', () => {
    it('should export dependencies as CSV blob', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await repositoriesApi.exportDependencies('csv');

      expect(mockClient.get).toHaveBeenCalledWith('/dependencies/export', {
        params: { format: 'csv' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });

    it('should export dependencies with type filter', async () => {
      mockClient.get.mockResolvedValue({ data: new Blob() });

      await repositoriesApi.exportDependencies('json', { dependency_type: 'npm' });

      expect(mockClient.get).toHaveBeenCalledWith('/dependencies/export', {
        params: { format: 'json', dependency_type: 'npm' },
        responseType: 'blob',
      });
    });
  });

  describe('markRemediated', () => {
    it('should mark repository as remediated', async () => {
      mockClient.post.mockResolvedValue({ data: { success: true } });

      const result = await repositoriesApi.markRemediated('org/repo');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/org%2Frepo/mark-remediated');
      expect(result).toEqual({ success: true });
    });
  });

  describe('markWontMigrate', () => {
    it('should mark repository as wont migrate', async () => {
      const mockRepo = { id: 1, status: 'wont_migrate' };
      mockClient.post.mockResolvedValue({ data: { repository: mockRepo } });

      const result = await repositoriesApi.markWontMigrate('org/repo');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/org%2Frepo/mark-wont-migrate', {
        unmark: false,
      });
      expect(result).toEqual(mockRepo);
    });

    it('should unmark repository from wont migrate', async () => {
      const mockRepo = { id: 1, status: 'pending' };
      mockClient.post.mockResolvedValue({ data: { repository: mockRepo } });

      const result = await repositoriesApi.markWontMigrate('org/repo', true);

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/org%2Frepo/mark-wont-migrate', {
        unmark: true,
      });
      expect(result).toEqual(mockRepo);
    });
  });

  describe('batchUpdateStatus', () => {
    it('should batch update repository statuses', async () => {
      mockClient.post.mockResolvedValue({ data: { updated: 3 } });

      const result = await repositoriesApi.batchUpdateStatus(
        [1, 2, 3],
        'mark_migrated',
        'Migration complete'
      );

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/batch-update', {
        repository_ids: [1, 2, 3],
        action: 'mark_migrated',
        reason: 'Migration complete',
      });
      expect(result).toEqual({ updated: 3 });
    });

    it('should batch update without reason', async () => {
      mockClient.post.mockResolvedValue({ data: { updated: 2 } });

      await repositoriesApi.batchUpdateStatus([1, 2], 'mark_wont_migrate');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/batch-update', {
        repository_ids: [1, 2],
        action: 'mark_wont_migrate',
        reason: '',
      });
    });
  });

  describe('discover', () => {
    it('should discover repositories for an organization', async () => {
      mockClient.post.mockResolvedValue({ data: { discovered: 10 } });

      const result = await repositoriesApi.discover('my-org');

      expect(mockClient.post).toHaveBeenCalledWith('/repositories/discover', {
        organization: 'my-org',
      });
      expect(result).toEqual({ discovered: 10 });
    });
  });
});

