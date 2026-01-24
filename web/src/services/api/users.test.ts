import { describe, it, expect, vi, beforeEach } from 'vitest';
import { usersApi } from './users';
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

describe('usersApi', () => {
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
    it('should list users without filters', async () => {
      const mockData = { users: [{ id: 1, login: 'user1' }], total: 1 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await usersApi.list();

      expect(mockClient.get).toHaveBeenCalledWith('/users', { params: undefined });
      expect(result).toEqual(mockData);
    });

    it('should list users with filters', async () => {
      const mockData = { users: [], total: 0 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await usersApi.list({ source_instance: 'github.com', limit: 50, offset: 100 });

      expect(mockClient.get).toHaveBeenCalledWith('/users', {
        params: { source_instance: 'github.com', limit: 50, offset: 100 },
      });
      expect(result).toEqual(mockData);
    });
  });

  describe('getStats', () => {
    it('should fetch user stats', async () => {
      const mockStats = { total: 100, with_mapping: 50 };
      mockClient.get.mockResolvedValue({ data: mockStats });

      const result = await usersApi.getStats();

      expect(mockClient.get).toHaveBeenCalledWith('/users/stats', { params: undefined });
      expect(result).toEqual(mockStats);
    });
  });

  describe('discover', () => {
    it('should discover users for an organization', async () => {
      mockClient.post.mockResolvedValue({ data: { discovered: 25 } });

      const result = await usersApi.discover('my-org');

      expect(mockClient.post).toHaveBeenCalledWith('/users/discover', { organization: 'my-org' });
      expect(result).toEqual({ discovered: 25 });
    });
  });

  describe('listMappings', () => {
    it('should list user mappings without filters', async () => {
      const mockData = { mappings: [], total: 0 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await usersApi.listMappings();

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings', { params: undefined });
      expect(result).toEqual(mockData);
    });

    it('should list user mappings with all filters', async () => {
      const mockData = { mappings: [], total: 0 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await usersApi.listMappings({
        status: 'mapped',
        source_org: 'my-org',
        has_destination: true,
        has_mannequin: false,
        reclaim_status: 'pending',
        search: 'user',
        limit: 10,
        offset: 20,
      });

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings', {
        params: {
          status: 'mapped',
          source_org: 'my-org',
          has_destination: true,
          has_mannequin: false,
          reclaim_status: 'pending',
          search: 'user',
          limit: 10,
          offset: 20,
        },
      });
      expect(result).toEqual(mockData);
    });
  });

  describe('getMappingStats', () => {
    it('should fetch mapping stats without org', async () => {
      const mockStats = { total: 100, mapped: 50 };
      mockClient.get.mockResolvedValue({ data: mockStats });

      const result = await usersApi.getMappingStats();

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/stats');
      expect(result).toEqual(mockStats);
    });

    it('should fetch mapping stats for a specific org', async () => {
      const mockStats = { total: 50, mapped: 25 };
      mockClient.get.mockResolvedValue({ data: mockStats });

      const result = await usersApi.getMappingStats('my-org');

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/stats?source_org=my-org');
      expect(result).toEqual(mockStats);
    });
  });

  describe('getDetail', () => {
    it('should get user detail', async () => {
      const mockDetail = { login: 'user1', name: 'User One' };
      mockClient.get.mockResolvedValue({ data: mockDetail });

      const result = await usersApi.getDetail('user1');

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/user1');
      expect(result).toEqual(mockDetail);
    });

    it('should encode special characters in login', async () => {
      mockClient.get.mockResolvedValue({ data: {} });

      await usersApi.getDetail('user@org');

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/user%40org');
    });
  });

  describe('getSourceOrgs', () => {
    it('should fetch source organizations', async () => {
      const mockData = { organizations: ['org1', 'org2'] };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await usersApi.getSourceOrgs();

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/source-orgs');
      expect(result).toEqual(mockData);
    });
  });

  describe('createMapping', () => {
    it('should create a user mapping', async () => {
      const newMapping = { source_login: 'user1', destination_login: 'user2' };
      mockClient.post.mockResolvedValue({ data: { ...newMapping, id: 1 } });

      const result = await usersApi.createMapping(newMapping);

      expect(mockClient.post).toHaveBeenCalledWith('/user-mappings', newMapping);
      expect(result).toEqual({ ...newMapping, id: 1 });
    });
  });

  describe('updateMapping', () => {
    it('should update a user mapping', async () => {
      const updates = { destination_login: 'new-user' };
      mockClient.patch.mockResolvedValue({ data: { source_login: 'user1', ...updates } });

      const result = await usersApi.updateMapping('user1', updates);

      expect(mockClient.patch).toHaveBeenCalledWith('/user-mappings/user1', updates);
      expect(result).toEqual({ source_login: 'user1', ...updates });
    });
  });

  describe('deleteMapping', () => {
    it('should delete a user mapping', async () => {
      mockClient.delete.mockResolvedValue({});

      await usersApi.deleteMapping('user1');

      expect(mockClient.delete).toHaveBeenCalledWith('/user-mappings/user1');
    });
  });

  describe('exportMappings', () => {
    it('should export user mappings as blob', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await usersApi.exportMappings('mapped');

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/export', {
        params: { status: 'mapped' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });
  });

  describe('generateGEICSV', () => {
    it('should generate GEI CSV with required org parameter', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await usersApi.generateGEICSV('my-org');

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/generate-gei-csv', {
        params: { org: 'my-org', status: undefined },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });

    it('should generate GEI CSV with org and status filter', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await usersApi.generateGEICSV('target-org', 'mapped');

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/generate-gei-csv', {
        params: { org: 'target-org', status: 'mapped' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });
  });

  describe('getMannequinOrgs', () => {
    it('should get list of mannequin organizations', async () => {
      const mockResponse = { orgs: ['org-alpha', 'org-beta'] };
      mockClient.get.mockResolvedValue({ data: mockResponse });

      const result = await usersApi.getMannequinOrgs();

      expect(mockClient.get).toHaveBeenCalledWith('/user-mappings/mannequin-orgs');
      expect(result).toEqual(mockResponse);
    });
  });

  describe('suggestMappings', () => {
    it('should suggest user mappings', async () => {
      const mockData = { suggestions: [], total: 0 };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await usersApi.suggestMappings();

      expect(mockClient.post).toHaveBeenCalledWith('/user-mappings/suggest');
      expect(result).toEqual(mockData);
    });
  });

  describe('syncMappings', () => {
    it('should sync user mappings', async () => {
      const mockData = { created: 10, message: 'Sync complete' };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await usersApi.syncMappings();

      expect(mockClient.post).toHaveBeenCalledWith('/user-mappings/sync');
      expect(result).toEqual(mockData);
    });
  });

  describe('fetchMannequins', () => {
    it('should fetch mannequins for a destination org', async () => {
      const mockData = {
        total_mannequins: 10,
        total_dest_members: 100,
        matched: 5,
        unmatched: 5,
        destination_org: 'dest-org',
        emu_shortcode_applied: false,
        message: 'Mannequins fetched',
      };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await usersApi.fetchMannequins('dest-org');

      expect(mockClient.post).toHaveBeenCalledWith('/user-mappings/fetch-mannequins', {
        destination_org: 'dest-org',
        emu_shortcode: undefined,
      });
      expect(result).toEqual(mockData);
    });

    it('should fetch mannequins with EMU shortcode', async () => {
      mockClient.post.mockResolvedValue({ data: {} });

      await usersApi.fetchMannequins('dest-org', 'emu');

      expect(mockClient.post).toHaveBeenCalledWith('/user-mappings/fetch-mannequins', {
        destination_org: 'dest-org',
        emu_shortcode: 'emu',
      });
    });
  });

  describe('sendAttributionInvitation', () => {
    it('should send attribution invitation', async () => {
      const mockData = {
        success: true,
        source_login: 'user1',
        mannequin_login: 'mona',
        target_user: 'target-user',
        message: 'Invitation sent',
      };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await usersApi.sendAttributionInvitation('user1', 'dest-org');

      expect(mockClient.post).toHaveBeenCalledWith('/user-mappings/user1/send-invitation', {
        destination_org: 'dest-org',
      });
      expect(result).toEqual(mockData);
    });
  });

  describe('bulkSendAttributionInvitations', () => {
    it('should bulk send attribution invitations', async () => {
      const mockData = {
        success: true,
        invited: 5,
        failed: 1,
        skipped: 2,
        errors: ['Error for user3'],
        message: 'Bulk invitations sent',
      };
      mockClient.post.mockResolvedValue({ data: mockData });

      const result = await usersApi.bulkSendAttributionInvitations('dest-org', ['user1', 'user2']);

      expect(mockClient.post).toHaveBeenCalledWith('/user-mappings/send-invitations', {
        destination_org: 'dest-org',
        source_logins: ['user1', 'user2'],
      });
      expect(result).toEqual(mockData);
    });
  });
});

