import { describe, it, expect, vi, beforeEach } from 'vitest';
import { discoveryApi } from './discovery';
import { client } from './client';

// Mock the axios client
vi.mock('./client', () => ({
  client: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('discoveryApi', () => {
  const mockClient = client as unknown as {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('start', () => {
    it('should start discovery for an organization', async () => {
      mockClient.post.mockResolvedValue({ data: { message: 'Discovery started' } });

      const result = await discoveryApi.start({ organization: 'my-org', workers: 5 });

      expect(mockClient.post).toHaveBeenCalledWith('/discovery/start', {
        organization: 'my-org',
        workers: 5,
      });
      expect(result).toEqual({ message: 'Discovery started' });
    });

    it('should start discovery for an enterprise', async () => {
      mockClient.post.mockResolvedValue({ data: { message: 'Discovery started' } });

      const result = await discoveryApi.start({ enterprise_slug: 'my-enterprise' });

      expect(mockClient.post).toHaveBeenCalledWith('/discovery/start', {
        enterprise_slug: 'my-enterprise',
      });
      expect(result).toEqual({ message: 'Discovery started' });
    });
  });

  describe('getStatus', () => {
    it('should fetch discovery status', async () => {
      const mockStatus = { status: 'running', progress: 50 };
      mockClient.get.mockResolvedValue({ data: mockStatus });

      const result = await discoveryApi.getStatus();

      expect(mockClient.get).toHaveBeenCalledWith('/discovery/status');
      expect(result).toEqual(mockStatus);
    });
  });

  describe('getProgress', () => {
    it('should fetch discovery progress', async () => {
      const mockProgress = { status: 'running', discovered: 50, total: 100 };
      mockClient.get.mockResolvedValue({ data: mockProgress });

      const result = await discoveryApi.getProgress();

      expect(mockClient.get).toHaveBeenCalledWith('/discovery/progress');
      expect(result).toEqual(mockProgress);
    });

    it('should return null when status is none', async () => {
      mockClient.get.mockResolvedValue({ data: { status: 'none' } });

      const result = await discoveryApi.getProgress();

      expect(result).toBeNull();
    });
  });

  describe('startADO', () => {
    it('should start ADO discovery', async () => {
      mockClient.post.mockResolvedValue({ data: { message: 'ADO discovery started' } });

      const result = await discoveryApi.startADO({ organization: 'my-ado-org', project: 'my-project' });

      expect(mockClient.post).toHaveBeenCalledWith('/ado/discover', {
        organization: 'my-ado-org',
        project: 'my-project',
      });
      expect(result).toEqual({ message: 'ADO discovery started' });
    });
  });

  describe('getADOStatus', () => {
    it('should fetch ADO discovery status', async () => {
      const mockStatus = { status: 'complete' };
      mockClient.get.mockResolvedValue({ data: mockStatus });

      const result = await discoveryApi.getADOStatus('my-ado-org');

      expect(mockClient.get).toHaveBeenCalledWith('/ado/discovery/status', {
        params: { organization: 'my-ado-org' },
      });
      expect(result).toEqual(mockStatus);
    });
  });

  describe('listADOProjects', () => {
    it('should list ADO projects', async () => {
      const mockProjects = [{ id: 1, name: 'Project 1' }];
      mockClient.get.mockResolvedValue({ data: { projects: mockProjects } });

      const result = await discoveryApi.listADOProjects('my-ado-org');

      expect(mockClient.get).toHaveBeenCalledWith('/ado/projects', {
        params: { organization: 'my-ado-org' },
      });
      expect(result).toEqual(mockProjects);
    });

    it('should return empty array when no projects', async () => {
      mockClient.get.mockResolvedValue({ data: {} });

      const result = await discoveryApi.listADOProjects();

      expect(result).toEqual([]);
    });
  });

  describe('getADOProject', () => {
    it('should fetch a specific ADO project', async () => {
      const mockProject = { id: 1, name: 'Project 1' };
      mockClient.get.mockResolvedValue({ data: mockProject });

      const result = await discoveryApi.getADOProject('my-org', 'my-project');

      expect(mockClient.get).toHaveBeenCalledWith('/ado/projects/my-org/my-project');
      expect(result).toEqual(mockProject);
    });

    it('should encode special characters in organization and project names', async () => {
      mockClient.get.mockResolvedValue({ data: {} });

      await discoveryApi.getADOProject('my org', 'my project');

      expect(mockClient.get).toHaveBeenCalledWith('/ado/projects/my%20org/my%20project');
    });
  });

  describe('listOrganizations', () => {
    it('should list organizations', async () => {
      const mockOrgs = [{ name: 'org1' }, { name: 'org2' }];
      mockClient.get.mockResolvedValue({ data: mockOrgs });

      const result = await discoveryApi.listOrganizations();

      expect(mockClient.get).toHaveBeenCalledWith('/organizations', { params: {} });
      expect(result).toEqual(mockOrgs);
    });
  });

  describe('listProjects', () => {
    it('should list projects', async () => {
      const mockProjects = [{ name: 'project1' }, { name: 'project2' }];
      mockClient.get.mockResolvedValue({ data: mockProjects });

      const result = await discoveryApi.listProjects();

      expect(mockClient.get).toHaveBeenCalledWith('/projects', { params: undefined });
      expect(result).toEqual(mockProjects);
    });
  });

  describe('getOrganizationList', () => {
    it('should get organization list as strings', async () => {
      const mockOrgs = ['org1', 'org2', 'org3'];
      mockClient.get.mockResolvedValue({ data: mockOrgs });

      const result = await discoveryApi.getOrganizationList();

      expect(mockClient.get).toHaveBeenCalledWith('/organizations/list');
      expect(result).toEqual(mockOrgs);
    });
  });

  describe('listTeams', () => {
    it('should list teams', async () => {
      const mockTeams = [{ id: 1, slug: 'team-1' }];
      mockClient.get.mockResolvedValue({ data: mockTeams });

      const result = await discoveryApi.listTeams();

      expect(mockClient.get).toHaveBeenCalledWith('/teams', { params: { organization: undefined } });
      expect(result).toEqual(mockTeams);
    });

    it('should list teams for a specific organization', async () => {
      const mockTeams = [{ id: 1, slug: 'team-1' }];
      mockClient.get.mockResolvedValue({ data: mockTeams });

      const result = await discoveryApi.listTeams('my-org');

      expect(mockClient.get).toHaveBeenCalledWith('/teams', { params: { organization: 'my-org' } });
      expect(result).toEqual(mockTeams);
    });
  });
});

