import { describe, it, expect, vi, beforeEach } from 'vitest';
import { teamsApi } from './teams';
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

describe('teamsApi', () => {
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
    it('should list teams without organization filter', async () => {
      const mockTeams = [{ id: 1, slug: 'team-1' }];
      mockClient.get.mockResolvedValue({ data: mockTeams });

      const result = await teamsApi.list();

      expect(mockClient.get).toHaveBeenCalledWith('/teams', { params: { organization: undefined } });
      expect(result).toEqual(mockTeams);
    });

    it('should list teams for a specific organization', async () => {
      const mockTeams = [{ id: 1, slug: 'team-1' }];
      mockClient.get.mockResolvedValue({ data: mockTeams });

      const result = await teamsApi.list('my-org');

      expect(mockClient.get).toHaveBeenCalledWith('/teams', { params: { organization: 'my-org' } });
      expect(result).toEqual(mockTeams);
    });
  });

  describe('getDetail', () => {
    it('should get team detail', async () => {
      const mockTeam = { id: 1, slug: 'team-1', name: 'Team One' };
      mockClient.get.mockResolvedValue({ data: mockTeam });

      const result = await teamsApi.getDetail('my-org', 'team-1');

      expect(mockClient.get).toHaveBeenCalledWith('/teams/my-org/team-1');
      expect(result).toEqual(mockTeam);
    });

    it('should encode special characters', async () => {
      mockClient.get.mockResolvedValue({ data: {} });

      await teamsApi.getDetail('my org', 'team 1');

      expect(mockClient.get).toHaveBeenCalledWith('/teams/my%20org/team%201');
    });
  });

  describe('getMembers', () => {
    it('should get team members', async () => {
      const mockData = { members: [{ id: 1, login: 'user1' }], total: 1 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await teamsApi.getMembers('my-org', 'team-1');

      expect(mockClient.get).toHaveBeenCalledWith('/teams/my-org/team-1/members');
      expect(result).toEqual(mockData);
    });
  });

  describe('discover', () => {
    it('should discover teams for an organization', async () => {
      mockClient.post.mockResolvedValue({ data: { discovered: 10 } });

      const result = await teamsApi.discover('my-org');

      expect(mockClient.post).toHaveBeenCalledWith('/teams/discover', { organization: 'my-org' });
      expect(result).toEqual({ discovered: 10 });
    });
  });

  describe('listMappings', () => {
    it('should list team mappings without filters', async () => {
      const mockData = { mappings: [], total: 0 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await teamsApi.listMappings();

      expect(mockClient.get).toHaveBeenCalledWith('/team-mappings', { params: undefined });
      expect(result).toEqual(mockData);
    });

    it('should list team mappings with all filters', async () => {
      const mockData = { mappings: [], total: 0 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await teamsApi.listMappings({
        source_org: 'source-org',
        destination_org: 'dest-org',
        status: 'mapped',
        has_destination: true,
        search: 'team',
        limit: 10,
        offset: 20,
      });

      expect(mockClient.get).toHaveBeenCalledWith('/team-mappings', {
        params: {
          source_org: 'source-org',
          destination_org: 'dest-org',
          status: 'mapped',
          has_destination: true,
          search: 'team',
          limit: 10,
          offset: 20,
        },
      });
      expect(result).toEqual(mockData);
    });
  });

  describe('getMappingStats', () => {
    it('should fetch mapping stats without org', async () => {
      const mockStats = { total: 50, mapped: 25 };
      mockClient.get.mockResolvedValue({ data: mockStats });

      const result = await teamsApi.getMappingStats();

      expect(mockClient.get).toHaveBeenCalledWith('/team-mappings/stats');
      expect(result).toEqual(mockStats);
    });

    it('should fetch mapping stats for a specific org', async () => {
      const mockStats = { total: 25, mapped: 15 };
      mockClient.get.mockResolvedValue({ data: mockStats });

      const result = await teamsApi.getMappingStats('my-org');

      expect(mockClient.get).toHaveBeenCalledWith('/team-mappings/stats?organization=my-org');
      expect(result).toEqual(mockStats);
    });
  });

  describe('getSourceOrgs', () => {
    it('should fetch source organizations', async () => {
      mockClient.get.mockResolvedValue({ data: { organizations: ['org1', 'org2'] } });

      const result = await teamsApi.getSourceOrgs();

      expect(mockClient.get).toHaveBeenCalledWith('/team-mappings/source-orgs');
      expect(result).toEqual(['org1', 'org2']);
    });

    it('should return empty array when no organizations', async () => {
      mockClient.get.mockResolvedValue({ data: {} });

      const result = await teamsApi.getSourceOrgs();

      expect(result).toEqual([]);
    });
  });

  describe('createMapping', () => {
    it('should create a team mapping', async () => {
      const newMapping = { source_org: 'org1', source_team_slug: 'team-1' };
      mockClient.post.mockResolvedValue({ data: { ...newMapping, id: 1 } });

      const result = await teamsApi.createMapping(newMapping);

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings', newMapping);
      expect(result).toEqual({ ...newMapping, id: 1 });
    });
  });

  describe('updateMapping', () => {
    it('should update a team mapping', async () => {
      const updates = { destination_team_slug: 'new-team' };
      mockClient.patch.mockResolvedValue({ data: { id: 1, ...updates } });

      const result = await teamsApi.updateMapping('my-org', 'team-1', updates);

      expect(mockClient.patch).toHaveBeenCalledWith('/team-mappings/my-org/team-1', updates);
      expect(result).toEqual({ id: 1, ...updates });
    });
  });

  describe('deleteMapping', () => {
    it('should delete a team mapping', async () => {
      mockClient.delete.mockResolvedValue({});

      await teamsApi.deleteMapping('my-org', 'team-1');

      expect(mockClient.delete).toHaveBeenCalledWith('/team-mappings/my-org/team-1');
    });
  });

  describe('exportMappings', () => {
    it('should export team mappings as blob', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await teamsApi.exportMappings({ status: 'mapped', source_org: 'my-org' });

      expect(mockClient.get).toHaveBeenCalledWith('/team-mappings/export', {
        params: { status: 'mapped', source_org: 'my-org' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });
  });

  describe('suggestMappings', () => {
    it('should suggest team mappings', async () => {
      const mockData = { suggestions: [], total: 0 };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await teamsApi.suggestMappings('dest-org', ['team-1', 'team-2']);

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings/suggest', {
        destination_org: 'dest-org',
        dest_team_slugs: ['team-1', 'team-2'],
      });
      expect(result).toEqual(mockData);
    });
  });

  describe('syncMappings', () => {
    it('should sync team mappings', async () => {
      const mockData = { created: 5, message: 'Sync complete' };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await teamsApi.syncMappings();

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings/sync');
      expect(result).toEqual(mockData);
    });
  });

  describe('executeMigration', () => {
    it('should execute team migration', async () => {
      const mockData = { message: 'Migration started', dry_run: false };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await teamsApi.executeMigration({
        source_org: 'my-org',
        source_team_slug: 'team-1',
        dry_run: false,
      });

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings/execute', {
        source_org: 'my-org',
        source_team_slug: 'team-1',
        dry_run: false,
      });
      expect(result).toEqual(mockData);
    });

    it('should execute team migration without options', async () => {
      const mockData = { message: 'Migration started', dry_run: false };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await teamsApi.executeMigration();

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings/execute', undefined);
      expect(result).toEqual(mockData);
    });
  });

  describe('getMigrationStatus', () => {
    it('should get migration status', async () => {
      const mockStatus = { is_running: true, progress: 50 };
      mockClient.get.mockResolvedValue({ data: mockStatus });

      const result = await teamsApi.getMigrationStatus();

      expect(mockClient.get).toHaveBeenCalledWith('/team-mappings/execution-status');
      expect(result).toEqual(mockStatus);
    });
  });

  describe('cancelMigration', () => {
    it('should cancel migration', async () => {
      const mockData = { message: 'Migration cancelled' };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await teamsApi.cancelMigration();

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings/cancel');
      expect(result).toEqual(mockData);
    });
  });

  describe('resetMigrationStatus', () => {
    it('should reset migration status without org', async () => {
      const mockData = { message: 'Status reset' };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await teamsApi.resetMigrationStatus();

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings/reset', null, {
        params: undefined,
      });
      expect(result).toEqual(mockData);
    });

    it('should reset migration status for a specific org', async () => {
      const mockData = { message: 'Status reset' };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await teamsApi.resetMigrationStatus('my-org');

      expect(mockClient.post).toHaveBeenCalledWith('/team-mappings/reset', null, {
        params: { source_org: 'my-org' },
      });
      expect(result).toEqual(mockData);
    });
  });
});

