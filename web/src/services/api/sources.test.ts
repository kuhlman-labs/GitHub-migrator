import { describe, it, expect, vi, beforeEach } from 'vitest';
import { sourcesApi } from './sources';
import { client } from './client';

// Mock the axios client
vi.mock('./client', () => ({
  client: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('sourcesApi', () => {
  const mockClient = client as unknown as {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
    put: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  const mockSource = {
    id: 1,
    name: 'GHES Production',
    type: 'github' as const,
    base_url: 'https://github.example.com/api/v3',
    is_active: true,
    has_app_auth: false,
    repository_count: 42,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    masked_token: 'ghp_...xxxx',
  };

  describe('list', () => {
    it('should fetch all sources', async () => {
      mockClient.get.mockResolvedValue({ data: [mockSource] });

      const result = await sourcesApi.list();

      expect(mockClient.get).toHaveBeenCalledWith('/sources', { params: {} });
      expect(result).toEqual([mockSource]);
    });

    it('should fetch only active sources when activeOnly is true', async () => {
      mockClient.get.mockResolvedValue({ data: [mockSource] });

      const result = await sourcesApi.list(true);

      expect(mockClient.get).toHaveBeenCalledWith('/sources', { params: { active: 'true' } });
      expect(result).toEqual([mockSource]);
    });
  });

  describe('get', () => {
    it('should fetch a single source by ID', async () => {
      mockClient.get.mockResolvedValue({ data: mockSource });

      const result = await sourcesApi.get(1);

      expect(mockClient.get).toHaveBeenCalledWith('/sources/1');
      expect(result).toEqual(mockSource);
    });
  });

  describe('create', () => {
    it('should create a new source', async () => {
      const createRequest = {
        name: 'New Source',
        type: 'github' as const,
        base_url: 'https://api.github.com',
        token: 'ghp_test_token',
      };

      mockClient.post.mockResolvedValue({ data: { ...mockSource, name: 'New Source' } });

      const result = await sourcesApi.create(createRequest);

      expect(mockClient.post).toHaveBeenCalledWith('/sources', createRequest);
      expect(result.name).toEqual('New Source');
    });

    it('should create an Azure DevOps source with organization', async () => {
      const createRequest = {
        name: 'ADO Source',
        type: 'azuredevops' as const,
        base_url: 'https://dev.azure.com/myorg',
        token: 'ado_pat',
        organization: 'myorg',
      };

      mockClient.post.mockResolvedValue({ data: { ...mockSource, type: 'azuredevops', organization: 'myorg' } });

      const result = await sourcesApi.create(createRequest);

      expect(mockClient.post).toHaveBeenCalledWith('/sources', createRequest);
      expect(result.type).toEqual('azuredevops');
    });
  });

  describe('update', () => {
    it('should update a source', async () => {
      const updates = { name: 'Updated Name' };

      mockClient.put.mockResolvedValue({ data: { ...mockSource, name: 'Updated Name' } });

      const result = await sourcesApi.update(1, updates);

      expect(mockClient.put).toHaveBeenCalledWith('/sources/1', updates);
      expect(result.name).toEqual('Updated Name');
    });

    it('should update source active status', async () => {
      const updates = { is_active: false };

      mockClient.put.mockResolvedValue({ data: { ...mockSource, is_active: false } });

      const result = await sourcesApi.update(1, updates);

      expect(mockClient.put).toHaveBeenCalledWith('/sources/1', updates);
      expect(result.is_active).toEqual(false);
    });
  });

  describe('delete', () => {
    it('should delete a source', async () => {
      mockClient.delete.mockResolvedValue({ data: {} });

      await sourcesApi.delete(1);

      expect(mockClient.delete).toHaveBeenCalledWith('/sources/1');
    });
  });

  describe('validate', () => {
    it('should validate using stored source credentials', async () => {
      mockClient.post.mockResolvedValue({
        data: { valid: true, details: { authenticated_user: 'testuser' } },
      });

      const result = await sourcesApi.validate({ source_id: 1 });

      expect(mockClient.post).toHaveBeenCalledWith('/sources/1/validate', { source_id: 1 });
      expect(result.valid).toBe(true);
    });

    it('should validate using inline credentials', async () => {
      const request = {
        type: 'github' as const,
        base_url: 'https://api.github.com',
        token: 'ghp_test',
      };

      mockClient.post.mockResolvedValue({ data: { valid: true } });

      const result = await sourcesApi.validate(request);

      expect(mockClient.post).toHaveBeenCalledWith('/sources/validate', request);
      expect(result.valid).toBe(true);
    });

    it('should return validation errors', async () => {
      mockClient.post.mockResolvedValue({
        data: { valid: false, error: 'Authentication failed' },
      });

      const result = await sourcesApi.validate({ source_id: 1 });

      expect(result.valid).toBe(false);
      expect(result.error).toBe('Authentication failed');
    });
  });

  describe('setActive', () => {
    it('should set source as active', async () => {
      mockClient.post.mockResolvedValue({
        data: { success: true, source_id: 1, is_active: true },
      });

      const result = await sourcesApi.setActive(1, true);

      expect(mockClient.post).toHaveBeenCalledWith('/sources/1/set-active', { is_active: true });
      expect(result.is_active).toBe(true);
    });

    it('should set source as inactive', async () => {
      mockClient.post.mockResolvedValue({
        data: { success: true, source_id: 1, is_active: false },
      });

      const result = await sourcesApi.setActive(1, false);

      expect(mockClient.post).toHaveBeenCalledWith('/sources/1/set-active', { is_active: false });
      expect(result.is_active).toBe(false);
    });
  });

  describe('getRepositories', () => {
    it('should fetch repositories for a source', async () => {
      const mockRepos = [
        { id: 1, full_name: 'org/repo1' },
        { id: 2, full_name: 'org/repo2' },
      ];

      mockClient.get.mockResolvedValue({ data: mockRepos });

      const result = await sourcesApi.getRepositories(1);

      expect(mockClient.get).toHaveBeenCalledWith('/sources/1/repositories');
      expect(result).toHaveLength(2);
    });
  });
});

