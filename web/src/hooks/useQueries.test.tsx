import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useOrganizations,
  useConfig,
  useProjects,
  useADOProjects,
  useRepositories,
  useRepository,
  useBatches,
  useBatch,
  useBatchRepositories,
  useAnalytics,
  useMigrationHistory,
  useDiscoveryStatus,
  useDashboardActionItems,
  useUsers,
  useUserStats,
  useUserMappings,
  useUserMappingStats,
  useUserMappingSourceOrgs,
  useTeams,
  useTeamMappings,
  useTeamMappingStats,
  useTeamSourceOrgs,
  useTeamMigrationStatus,
  useSetupStatus,
  useDiscoveryProgress,
} from './useQueries';
import { api } from '../services/api';

// Mock the API module
vi.mock('../services/api', () => ({
  api: {
    listOrganizations: vi.fn(),
    getConfig: vi.fn(),
    listProjects: vi.fn(),
    listADOProjects: vi.fn(),
    listRepositories: vi.fn(),
    getRepositoryDetail: vi.fn(),
    listBatches: vi.fn(),
    getBatch: vi.fn(),
    getBatchRepositories: vi.fn(),
    getAnalyticsSummary: vi.fn(),
    getMigrationHistoryList: vi.fn(),
    getDiscoveryStatus: vi.fn(),
    getDashboardActionItems: vi.fn(),
    listUsers: vi.fn(),
    getUserStats: vi.fn(),
    listUserMappings: vi.fn(),
    getUserMappingStats: vi.fn(),
    getUserMappingSourceOrgs: vi.fn(),
    listTeams: vi.fn(),
    listTeamMappings: vi.fn(),
    getTeamMappingStats: vi.fn(),
    getTeamSourceOrgs: vi.fn(),
    getTeamMigrationStatus: vi.fn(),
    getSetupStatus: vi.fn(),
    getDiscoveryProgress: vi.fn(),
  },
}));

// Create wrapper with QueryClient
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useQueries hooks', () => {
  const mockApi = api as unknown as {
    listOrganizations: ReturnType<typeof vi.fn>;
    getConfig: ReturnType<typeof vi.fn>;
    listProjects: ReturnType<typeof vi.fn>;
    listADOProjects: ReturnType<typeof vi.fn>;
    listRepositories: ReturnType<typeof vi.fn>;
    getRepositoryDetail: ReturnType<typeof vi.fn>;
    listBatches: ReturnType<typeof vi.fn>;
    getBatch: ReturnType<typeof vi.fn>;
    getBatchRepositories: ReturnType<typeof vi.fn>;
    getAnalyticsSummary: ReturnType<typeof vi.fn>;
    getMigrationHistoryList: ReturnType<typeof vi.fn>;
    getDiscoveryStatus: ReturnType<typeof vi.fn>;
    getDashboardActionItems: ReturnType<typeof vi.fn>;
    listUsers: ReturnType<typeof vi.fn>;
    getUserStats: ReturnType<typeof vi.fn>;
    listUserMappings: ReturnType<typeof vi.fn>;
    getUserMappingStats: ReturnType<typeof vi.fn>;
    getUserMappingSourceOrgs: ReturnType<typeof vi.fn>;
    listTeams: ReturnType<typeof vi.fn>;
    listTeamMappings: ReturnType<typeof vi.fn>;
    getTeamMappingStats: ReturnType<typeof vi.fn>;
    getTeamSourceOrgs: ReturnType<typeof vi.fn>;
    getTeamMigrationStatus: ReturnType<typeof vi.fn>;
    getSetupStatus: ReturnType<typeof vi.fn>;
    getDiscoveryProgress: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('useOrganizations', () => {
    it('should fetch organizations', async () => {
      const mockOrgs = [{ name: 'org1' }, { name: 'org2' }];
      mockApi.listOrganizations.mockResolvedValue(mockOrgs);

      const { result } = renderHook(() => useOrganizations(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockOrgs);
      expect(mockApi.listOrganizations).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.listOrganizations.mockRejectedValue(new Error('Failed'));

      const { result } = renderHook(() => useOrganizations(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(result.current.error?.message).toBe('Failed');
    });
  });

  describe('useConfig', () => {
    it('should fetch config', async () => {
      const mockConfig = { source_type: 'github', auth_enabled: true };
      mockApi.getConfig.mockResolvedValue(mockConfig);

      const { result } = renderHook(() => useConfig(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockConfig);
    });
  });

  describe('useProjects', () => {
    it('should fetch projects', async () => {
      const mockProjects = [{ name: 'project1' }];
      mockApi.listProjects.mockResolvedValue(mockProjects);

      const { result } = renderHook(() => useProjects(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockProjects);
    });
  });

  describe('useRepositories', () => {
    it('should fetch repositories without filters', async () => {
      const mockData = { repositories: [{ id: 1, full_name: 'org/repo' }] };
      mockApi.listRepositories.mockResolvedValue(mockData);

      const { result } = renderHook(() => useRepositories(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockData);
      expect(mockApi.listRepositories).toHaveBeenCalledWith({});
    });

    it('should fetch repositories with filters', async () => {
      const mockData = { repositories: [] };
      mockApi.listRepositories.mockResolvedValue(mockData);

      const { result } = renderHook(
        () => useRepositories({ status: ['pending'], organization: ['my-org'] }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.listRepositories).toHaveBeenCalledWith({
        status: ['pending'],
        organization: ['my-org'],
      });
    });
  });

  describe('useBatches', () => {
    it('should fetch batches', async () => {
      const mockBatches = [{ id: 1, name: 'Batch 1' }];
      mockApi.listBatches.mockResolvedValue(mockBatches);

      const { result } = renderHook(() => useBatches(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockBatches);
    });
  });

  describe('useBatch', () => {
    it('should fetch a single batch', async () => {
      const mockBatch = { id: 1, name: 'Batch 1' };
      mockApi.getBatch.mockResolvedValue(mockBatch);

      const { result } = renderHook(() => useBatch(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockBatch);
      expect(mockApi.getBatch).toHaveBeenCalledWith(1);
    });

    it('should not fetch when id is 0', async () => {
      const { result } = renderHook(() => useBatch(0), {
        wrapper: createWrapper(),
      });

      // Should be idle when disabled
      expect(result.current.fetchStatus).toBe('idle');
      expect(mockApi.getBatch).not.toHaveBeenCalled();
    });
  });

  describe('useAnalytics', () => {
    it('should fetch analytics', async () => {
      const mockAnalytics = { total_repositories: 100, migrated_count: 50 };
      mockApi.getAnalyticsSummary.mockResolvedValue(mockAnalytics);

      const { result } = renderHook(() => useAnalytics(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockAnalytics);
    });

    it('should fetch analytics with filters', async () => {
      const mockAnalytics = { total_repositories: 25 };
      mockApi.getAnalyticsSummary.mockResolvedValue(mockAnalytics);

      const { result } = renderHook(
        () => useAnalytics({ organization: 'my-org', batch_id: '5' }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getAnalyticsSummary).toHaveBeenCalledWith({
        organization: 'my-org',
        batch_id: '5',
      });
    });
  });

  describe('useMigrationHistory', () => {
    it('should fetch migration history', async () => {
      const mockData = { migrations: [{ id: 1 }], total: 1 };
      mockApi.getMigrationHistoryList.mockResolvedValue(mockData);

      const { result } = renderHook(() => useMigrationHistory(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockData);
    });
  });

  describe('useDiscoveryStatus', () => {
    it('should fetch discovery status', async () => {
      const mockStatus = { status: 'running', discovered_count: 50, is_running: true };
      mockApi.getDiscoveryStatus.mockResolvedValue(mockStatus);

      const { result } = renderHook(() => useDiscoveryStatus(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockStatus);
    });
  });

  describe('useUserStats', () => {
    it('should fetch user stats', async () => {
      const mockStats = { total: 100, with_mapping: 50 };
      mockApi.getUserStats.mockResolvedValue(mockStats);

      const { result } = renderHook(() => useUserStats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockStats);
    });
  });

  describe('useUserMappingStats', () => {
    it('should fetch user mapping stats', async () => {
      const mockStats = { total: 100, mapped: 50 };
      mockApi.getUserMappingStats.mockResolvedValue(mockStats);

      const { result } = renderHook(() => useUserMappingStats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockStats);
    });

    it('should fetch user mapping stats for specific org', async () => {
      const mockStats = { total: 25, mapped: 15 };
      mockApi.getUserMappingStats.mockResolvedValue(mockStats);

      const { result } = renderHook(() => useUserMappingStats('my-org'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getUserMappingStats).toHaveBeenCalledWith('my-org', undefined);
    });
  });

  describe('useTeams', () => {
    it('should fetch teams', async () => {
      const mockTeams = [{ id: 1, slug: 'team-1' }];
      mockApi.listTeams.mockResolvedValue(mockTeams);

      const { result } = renderHook(() => useTeams(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockTeams);
    });

    it('should fetch teams for a specific organization', async () => {
      const mockTeams = [{ id: 1, slug: 'team-1' }];
      mockApi.listTeams.mockResolvedValue(mockTeams);

      const { result } = renderHook(() => useTeams('my-org'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.listTeams).toHaveBeenCalledWith('my-org', undefined);
    });
  });

  describe('useTeamMappingStats', () => {
    it('should fetch team mapping stats', async () => {
      const mockStats = { total: 50, mapped: 25 };
      mockApi.getTeamMappingStats.mockResolvedValue(mockStats);

      const { result } = renderHook(() => useTeamMappingStats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockStats);
    });
  });

  describe('useSetupStatus', () => {
    it('should fetch setup status', async () => {
      const mockStatus = { setup_completed: true, completed_at: '2024-01-01' };
      mockApi.getSetupStatus.mockResolvedValue(mockStatus);

      const { result } = renderHook(() => useSetupStatus(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockStatus);
    });
  });

  describe('useADOProjects', () => {
    it('should return loading state initially', () => {
      mockApi.listADOProjects.mockResolvedValue([]);

      const { result } = renderHook(() => useADOProjects('my-org'), {
        wrapper: createWrapper(),
      });

      // Just verify the hook can be called
      expect(result.current).toBeDefined();
    });
  });

  describe('useRepository', () => {
    it('should return loading state initially', () => {
      mockApi.getRepositoryDetail.mockResolvedValue({ repository: {}, history: [] });

      const { result } = renderHook(() => useRepository('org/repo'), {
        wrapper: createWrapper(),
      });

      // Just verify the hook can be called
      expect(result.current).toBeDefined();
    });
  });

  describe('useBatchRepositories', () => {
    it('should not fetch when batchId is null', () => {
      const { result } = renderHook(() => useBatchRepositories(null), {
        wrapper: createWrapper(),
      });

      // When batchId is null, query should be disabled
      expect(result.current.data).toBeUndefined();
    });

    it('should return loading state for valid batchId', () => {
      mockApi.getBatchRepositories.mockResolvedValue({ repositories: [], total: 0 });

      const { result } = renderHook(() => useBatchRepositories(1), {
        wrapper: createWrapper(),
      });

      // Just verify the hook can be called
      expect(result.current).toBeDefined();
    });
  });

  describe('useDashboardActionItems', () => {
    it('should fetch dashboard action items', async () => {
      const mockItems = {
        remediation_required: [],
        failed_migrations: [],
        pending_dry_runs: [],
        total_action_items: 0,
      };
      mockApi.getDashboardActionItems.mockResolvedValue(mockItems);

      const { result } = renderHook(() => useDashboardActionItems(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockItems);
    });
  });

  describe('useUsers', () => {
    it('should fetch users with filters', async () => {
      const mockUsers = {
        users: [{ id: 1, login: 'user1' }],
        total: 1,
      };
      mockApi.listUsers.mockResolvedValue(mockUsers);

      const { result } = renderHook(() => useUsers({ search: 'user1' }), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockUsers);
    });
  });

  describe('useUserMappings', () => {
    it('should fetch user mappings', async () => {
      const mockMappings = {
        mappings: [{ id: 1, source_login: 'user1', destination_login: 'user1-dest' }],
        total: 1,
      };
      mockApi.listUserMappings.mockResolvedValue(mockMappings);

      const { result } = renderHook(() => useUserMappings({}), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockMappings);
    });
  });

  describe('useUserMappingSourceOrgs', () => {
    it('should fetch user mapping source orgs', async () => {
      const mockOrgs = ['org1', 'org2'];
      mockApi.getUserMappingSourceOrgs.mockResolvedValue(mockOrgs);

      const { result } = renderHook(() => useUserMappingSourceOrgs(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockOrgs);
    });
  });

  describe('useTeamMappings', () => {
    it('should fetch team mappings', async () => {
      const mockMappings = {
        mappings: [{ id: 1, source_team: 'team1', destination_team: 'team1-dest' }],
        total: 1,
      };
      mockApi.listTeamMappings.mockResolvedValue(mockMappings);

      const { result } = renderHook(() => useTeamMappings({}), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockMappings);
    });
  });

  describe('useTeamSourceOrgs', () => {
    it('should fetch team source orgs', async () => {
      const mockOrgs = ['org1', 'org2'];
      mockApi.getTeamSourceOrgs.mockResolvedValue(mockOrgs);

      const { result } = renderHook(() => useTeamSourceOrgs(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockOrgs);
    });
  });

  describe('useTeamMigrationStatus', () => {
    it('should fetch team migration status', async () => {
      const mockStatus = { in_progress: 5, completed: 10, failed: 2 };
      mockApi.getTeamMigrationStatus.mockResolvedValue(mockStatus);

      const { result } = renderHook(() => useTeamMigrationStatus(true), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockStatus);
    });

    it('should not fetch when disabled', async () => {
      const { result } = renderHook(() => useTeamMigrationStatus(false), {
        wrapper: createWrapper(),
      });

      expect(result.current.isLoading).toBe(false);
    });
  });

  describe('useDiscoveryProgress', () => {
    it('should fetch discovery progress', async () => {
      const mockProgress = { status: 'in_progress', percent: 50, discovered: 25, total: 50 };
      mockApi.getDiscoveryProgress.mockResolvedValue(mockProgress);

      const { result } = renderHook(() => useDiscoveryProgress(true), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(mockProgress);
    });

    it('should not fetch when disabled', async () => {
      const { result } = renderHook(() => useDiscoveryProgress(false), {
        wrapper: createWrapper(),
      });

      expect(result.current.isLoading).toBe(false);
    });
  });
});

