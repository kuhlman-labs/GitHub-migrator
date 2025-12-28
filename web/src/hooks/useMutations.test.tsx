import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useStartDiscovery,
  useStartADODiscovery,
  useDiscoverRepositories,
  useDiscoverOrgMembers,
  useDiscoverTeams,
  useCreateBatch,
  useUpdateBatch,
  useStartBatch,
  useDeleteBatch,
  useDryRunBatch,
  useRetryBatchFailures,
  useRetryRepository,
  useAddRepositoriesToBatch,
  useRemoveRepositoriesFromBatch,
  useStartMigration,
  useUpdateRepository,
  useBatchUpdateRepositoryStatus,
  useRollbackRepository,
  useCreateUserMapping,
  useUpdateUserMapping,
  useDeleteUserMapping,
  useCreateTeamMapping,
  useUpdateTeamMapping,
  useDeleteTeamMapping,
  useExecuteTeamMigration,
  useFetchMannequins,
  useSendAttributionInvitation,
} from './useMutations';
import { api } from '../services/api';

// Mock the API module
vi.mock('../services/api', () => ({
  api: {
    startDiscovery: vi.fn(),
    startADODiscovery: vi.fn(),
    discoverRepositories: vi.fn(),
    discoverOrgMembers: vi.fn(),
    discoverTeams: vi.fn(),
    createBatch: vi.fn(),
    updateBatch: vi.fn(),
    startBatch: vi.fn(),
    deleteBatch: vi.fn(),
    dryRunBatch: vi.fn(),
    retryBatchFailures: vi.fn(),
    retryRepository: vi.fn(),
    addRepositoriesToBatch: vi.fn(),
    removeRepositoriesFromBatch: vi.fn(),
    startMigration: vi.fn(),
    updateRepository: vi.fn(),
    batchUpdateRepositoryStatus: vi.fn(),
    rollbackRepository: vi.fn(),
    createUserMapping: vi.fn(),
    updateUserMapping: vi.fn(),
    deleteUserMapping: vi.fn(),
    createTeamMapping: vi.fn(),
    updateTeamMapping: vi.fn(),
    deleteTeamMapping: vi.fn(),
    executeTeamMigration: vi.fn(),
    fetchMannequins: vi.fn(),
    sendAttributionInvitation: vi.fn(),
  },
}));

// Create wrapper with QueryClient
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useMutations hooks', () => {
  const mockApi = api as unknown as {
    startDiscovery: ReturnType<typeof vi.fn>;
    startADODiscovery: ReturnType<typeof vi.fn>;
    discoverRepositories: ReturnType<typeof vi.fn>;
    discoverOrgMembers: ReturnType<typeof vi.fn>;
    discoverTeams: ReturnType<typeof vi.fn>;
    createBatch: ReturnType<typeof vi.fn>;
    updateBatch: ReturnType<typeof vi.fn>;
    startBatch: ReturnType<typeof vi.fn>;
    deleteBatch: ReturnType<typeof vi.fn>;
    dryRunBatch: ReturnType<typeof vi.fn>;
    retryBatchFailures: ReturnType<typeof vi.fn>;
    retryRepository: ReturnType<typeof vi.fn>;
    addRepositoriesToBatch: ReturnType<typeof vi.fn>;
    removeRepositoriesFromBatch: ReturnType<typeof vi.fn>;
    startMigration: ReturnType<typeof vi.fn>;
    updateRepository: ReturnType<typeof vi.fn>;
    batchUpdateRepositoryStatus: ReturnType<typeof vi.fn>;
    rollbackRepository: ReturnType<typeof vi.fn>;
    createUserMapping: ReturnType<typeof vi.fn>;
    updateUserMapping: ReturnType<typeof vi.fn>;
    deleteUserMapping: ReturnType<typeof vi.fn>;
    createTeamMapping: ReturnType<typeof vi.fn>;
    updateTeamMapping: ReturnType<typeof vi.fn>;
    deleteTeamMapping: ReturnType<typeof vi.fn>;
    executeTeamMigration: ReturnType<typeof vi.fn>;
    fetchMannequins: ReturnType<typeof vi.fn>;
    sendAttributionInvitation: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('useStartDiscovery', () => {
    it('should start discovery', async () => {
      mockApi.startDiscovery.mockResolvedValue({ message: 'Started' });

      const { result } = renderHook(() => useStartDiscovery(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ organization: 'my-org' });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.startDiscovery).toHaveBeenCalledWith({ organization: 'my-org' });
    });
  });

  describe('useCreateBatch', () => {
    it('should create a batch', async () => {
      const newBatch = { id: 1, name: 'Test Batch', type: 'migration' };
      mockApi.createBatch.mockResolvedValue(newBatch);

      const { result } = renderHook(() => useCreateBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ name: 'Test Batch', type: 'migration' });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual(newBatch);
    });
  });

  describe('useUpdateBatch', () => {
    it('should update a batch', async () => {
      const updatedBatch = { id: 1, name: 'Updated Name' };
      mockApi.updateBatch.mockResolvedValue(updatedBatch);

      const { result } = renderHook(() => useUpdateBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ id: 1, updates: { name: 'Updated Name' } });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.updateBatch).toHaveBeenCalledWith(1, { name: 'Updated Name' });
    });
  });

  describe('useStartBatch', () => {
    it('should start a batch', async () => {
      mockApi.startBatch.mockResolvedValue({ message: 'Started' });

      const { result } = renderHook(() => useStartBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ id: 1 });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.startBatch).toHaveBeenCalledWith(1, undefined);
    });

    it('should start batch with skip dry run option', async () => {
      mockApi.startBatch.mockResolvedValue({ message: 'Started' });

      const { result } = renderHook(() => useStartBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ id: 1, skipDryRun: true });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.startBatch).toHaveBeenCalledWith(1, true);
    });
  });

  describe('useDeleteBatch', () => {
    it('should delete a batch', async () => {
      mockApi.deleteBatch.mockResolvedValue({ message: 'Deleted' });

      const { result } = renderHook(() => useDeleteBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate(1);
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.deleteBatch).toHaveBeenCalledWith(1);
    });
  });

  describe('useDryRunBatch', () => {
    it('should run dry run for a batch', async () => {
      mockApi.dryRunBatch.mockResolvedValue({ started: 10 });

      const { result } = renderHook(() => useDryRunBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ id: 1, onlyPending: true });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.dryRunBatch).toHaveBeenCalledWith(1, true);
    });
  });

  describe('useAddRepositoriesToBatch', () => {
    it('should add repositories to a batch', async () => {
      mockApi.addRepositoriesToBatch.mockResolvedValue({ added: 3 });

      const { result } = renderHook(() => useAddRepositoriesToBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ batchId: 1, repositoryIds: [10, 20, 30] });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.addRepositoriesToBatch).toHaveBeenCalledWith(1, [10, 20, 30]);
    });
  });

  describe('useRemoveRepositoriesFromBatch', () => {
    it('should remove repositories from a batch', async () => {
      mockApi.removeRepositoriesFromBatch.mockResolvedValue({ removed: 2 });

      const { result } = renderHook(() => useRemoveRepositoriesFromBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ batchId: 1, repositoryIds: [10, 20] });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.removeRepositoriesFromBatch).toHaveBeenCalledWith(1, [10, 20]);
    });
  });

  describe('useStartMigration', () => {
    it('should start migration', async () => {
      mockApi.startMigration.mockResolvedValue({ started: 5 });

      const { result } = renderHook(() => useStartMigration(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ repositoryIds: [1, 2, 3], dryRun: true });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.startMigration).toHaveBeenCalledWith({
        repository_ids: [1, 2, 3],
        dry_run: true,
      });
    });
  });

  describe('useUpdateRepository', () => {
    it('should update repository', async () => {
      mockApi.updateRepository.mockResolvedValue({ full_name: 'org/repo' });

      const { result } = renderHook(() => useUpdateRepository(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({
          fullName: 'org/repo',
          updates: { destination_full_name: 'new-org/repo' },
        });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.updateRepository).toHaveBeenCalledWith('org/repo', {
        destination_full_name: 'new-org/repo',
      });
    });
  });

  describe('useRollbackRepository', () => {
    it('should rollback repository', async () => {
      mockApi.rollbackRepository.mockResolvedValue({ status: 'pending' });

      const { result } = renderHook(() => useRollbackRepository(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ fullName: 'org/repo', reason: 'Need to redo' });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.rollbackRepository).toHaveBeenCalledWith('org/repo', 'Need to redo');
    });
  });

  describe('useCreateUserMapping', () => {
    it('should create user mapping', async () => {
      const mapping = { source_login: 'user1', destination_login: 'user2' };
      mockApi.createUserMapping.mockResolvedValue({ id: 1, ...mapping });

      const { result } = renderHook(() => useCreateUserMapping(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate(mapping);
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createUserMapping).toHaveBeenCalledWith(mapping);
    });
  });

  describe('useUpdateUserMapping', () => {
    it('should update user mapping', async () => {
      mockApi.updateUserMapping.mockResolvedValue({ source_login: 'user1' });

      const { result } = renderHook(() => useUpdateUserMapping(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({
          sourceLogin: 'user1',
          updates: { destination_login: 'new-user' },
        });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.updateUserMapping).toHaveBeenCalledWith('user1', {
        destination_login: 'new-user',
      });
    });
  });

  describe('useDeleteUserMapping', () => {
    it('should delete user mapping', async () => {
      mockApi.deleteUserMapping.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteUserMapping(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate('user1');
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.deleteUserMapping).toHaveBeenCalledWith('user1');
    });
  });

  describe('useCreateTeamMapping', () => {
    it('should create team mapping', async () => {
      const mapping = { source_org: 'org1', source_team_slug: 'team-1' };
      mockApi.createTeamMapping.mockResolvedValue({ id: 1, ...mapping });

      const { result } = renderHook(() => useCreateTeamMapping(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate(mapping);
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createTeamMapping).toHaveBeenCalledWith(mapping);
    });
  });

  describe('useUpdateTeamMapping', () => {
    it('should update team mapping', async () => {
      mockApi.updateTeamMapping.mockResolvedValue({ id: 1 });

      const { result } = renderHook(() => useUpdateTeamMapping(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({
          sourceOrg: 'org1',
          sourceTeamSlug: 'team-1',
          updates: { destination_team_slug: 'new-team' },
        });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.updateTeamMapping).toHaveBeenCalledWith('org1', 'team-1', {
        destination_team_slug: 'new-team',
      });
    });
  });

  describe('useDeleteTeamMapping', () => {
    it('should delete team mapping', async () => {
      mockApi.deleteTeamMapping.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteTeamMapping(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ sourceOrg: 'org1', sourceTeamSlug: 'team-1' });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.deleteTeamMapping).toHaveBeenCalledWith('org1', 'team-1');
    });
  });

  describe('useStartADODiscovery', () => {
    it('should start ADO discovery', async () => {
      mockApi.startADODiscovery.mockResolvedValue({ message: 'ADO Discovery started' });

      const { result } = renderHook(() => useStartADODiscovery(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ organization: 'ado-org', project: 'my-project' });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.startADODiscovery).toHaveBeenCalledWith({ organization: 'ado-org', project: 'my-project' });
    });
  });

  describe('useDiscoverRepositories', () => {
    it('should discover repositories for organization', async () => {
      mockApi.discoverRepositories.mockResolvedValue({ discovered: 10 });

      const { result } = renderHook(() => useDiscoverRepositories(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate('test-org');
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.discoverRepositories).toHaveBeenCalledWith('test-org');
    });
  });

  describe('useDiscoverOrgMembers', () => {
    it('should discover organization members', async () => {
      mockApi.discoverOrgMembers.mockResolvedValue({ discovered: 5 });

      const { result } = renderHook(() => useDiscoverOrgMembers(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate('test-org');
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.discoverOrgMembers).toHaveBeenCalledWith('test-org');
    });
  });

  describe('useDiscoverTeams', () => {
    it('should discover teams for organization', async () => {
      mockApi.discoverTeams.mockResolvedValue({ discovered: 3 });

      const { result } = renderHook(() => useDiscoverTeams(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate('test-org');
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.discoverTeams).toHaveBeenCalledWith('test-org');
    });
  });

  describe('useRetryBatchFailures', () => {
    it('should retry batch failures', async () => {
      mockApi.retryBatchFailures.mockResolvedValue({ retried: 3 });

      const { result } = renderHook(() => useRetryBatchFailures(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ id: 1, repositoryIds: [1, 2, 3] });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.retryBatchFailures).toHaveBeenCalledWith(1, [1, 2, 3]);
    });
  });

  describe('useRetryRepository', () => {
    it('should retry repository migration', async () => {
      mockApi.retryRepository.mockResolvedValue({ started: true });

      const { result } = renderHook(() => useRetryRepository(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ repositoryId: 1, dryRun: false });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.retryRepository).toHaveBeenCalledWith(1, false);
    });
  });

  describe('useBatchUpdateRepositoryStatus', () => {
    it('should batch update repository status with mark_migrated action', async () => {
      mockApi.batchUpdateRepositoryStatus.mockResolvedValue({ updated: 5 });

      const { result } = renderHook(() => useBatchUpdateRepositoryStatus(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ repositoryIds: [1, 2, 3], action: 'mark_migrated' as const });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.batchUpdateRepositoryStatus).toHaveBeenCalled();
    });
  });

  describe('useExecuteTeamMigration', () => {
    it('should execute team migration', async () => {
      mockApi.executeTeamMigration.mockResolvedValue({ migrated: true });

      const { result } = renderHook(() => useExecuteTeamMigration(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ sourceOrg: 'source-org', sourceTeamSlug: 'team-slug', dryRun: false });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.executeTeamMigration).toHaveBeenCalled();
    });
  });

  describe('useFetchMannequins', () => {
    it('should fetch mannequins with destination org', async () => {
      mockApi.fetchMannequins.mockResolvedValue({ fetched: 10 });

      const { result } = renderHook(() => useFetchMannequins(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ destinationOrg: 'dest-org' });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.fetchMannequins).toHaveBeenCalled();
    });
  });

  describe('useSendAttributionInvitation', () => {
    it('should send attribution invitation', async () => {
      mockApi.sendAttributionInvitation.mockResolvedValue({ sent: true });

      const { result } = renderHook(() => useSendAttributionInvitation(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ userId: 1 });
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.sendAttributionInvitation).toHaveBeenCalled();
    });
  });

  describe('error handling', () => {
    it('should handle errors', async () => {
      mockApi.createBatch.mockRejectedValue(new Error('Failed to create batch'));

      const { result } = renderHook(() => useCreateBatch(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate({ name: 'Test', type: 'migration' });
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(result.current.error?.message).toBe('Failed to create batch');
    });
  });
});

