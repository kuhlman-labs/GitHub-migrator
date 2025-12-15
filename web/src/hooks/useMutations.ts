import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../services/api';

// Discovery mutations
export function useStartDiscovery() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (params: { organization?: string; enterprise_slug?: string }) => api.startDiscovery(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['discoveryStatus'] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['organizations'] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
    },
  });
}

// Azure DevOps Discovery mutations
export function useStartADODiscovery() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (params: { organization?: string; project?: string; workers?: number }) => api.startADODiscovery(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['adoDiscoveryStatus'] });
      queryClient.invalidateQueries({ queryKey: ['adoProjects'] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
    },
  });
}

// Standalone Discovery mutations (per entity type)
export function useDiscoverRepositories() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (organization: string) => api.discoverRepositories(organization),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['organizations'] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
    },
  });
}

export function useDiscoverOrgMembers() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (organization: string) => api.discoverOrgMembers(organization),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
      queryClient.invalidateQueries({ queryKey: ['userSourceOrgs'] });
    },
  });
}

export function useDiscoverTeams() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (organization: string) => api.discoverTeams(organization),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teams'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
      queryClient.invalidateQueries({ queryKey: ['teamSourceOrgs'] });
      // Team discovery also creates users
      queryClient.invalidateQueries({ queryKey: ['users'] });
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
    },
  });
}

export function useRediscoverRepository() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (fullName: string) => api.rediscoverRepository(fullName),
    onSuccess: (_, fullName) => {
      queryClient.invalidateQueries({ queryKey: ['repository', fullName] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
    },
  });
}

// Batch mutations
export function useCreateBatch() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (batch: { name: string; type: string; description?: string }) => api.createBatch(batch),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batches'] });
    },
  });
}

export function useUpdateBatch() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, updates }: { id: number; updates: Partial<{ name: string; description: string }> }) => 
      api.updateBatch(id, updates),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['batch', id] });
      queryClient.invalidateQueries({ queryKey: ['batches'] });
    },
  });
}

export function useStartBatch() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: number) => api.startBatch(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['batch', id] });
      queryClient.invalidateQueries({ queryKey: ['batches'] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
    },
  });
}

export function useAddRepositoriesToBatch() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ batchId, repositoryIds }: { batchId: number; repositoryIds: number[] }) => 
      api.addRepositoriesToBatch(batchId, repositoryIds),
    onSuccess: (_, { batchId }) => {
      queryClient.invalidateQueries({ queryKey: ['batch', batchId] });
      queryClient.invalidateQueries({ queryKey: ['batches'] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
    },
  });
}

export function useRemoveRepositoriesFromBatch() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ batchId, repositoryIds }: { batchId: number; repositoryIds: number[] }) => 
      api.removeRepositoriesFromBatch(batchId, repositoryIds),
    onSuccess: (_, { batchId }) => {
      queryClient.invalidateQueries({ queryKey: ['batch', batchId] });
      queryClient.invalidateQueries({ queryKey: ['batches'] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
    },
  });
}

// Migration mutations
export function useStartMigration() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ repositoryIds, dryRun }: { repositoryIds?: number[]; repositoryFullNames?: string[]; dryRun: boolean }) => 
      api.startMigration({ repository_ids: repositoryIds, dry_run: dryRun }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['migrationHistory'] });
    },
  });
}

// Repository mutations
export function useUpdateRepository() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ fullName, updates }: { fullName: string; updates: Partial<{ destination_full_name: string }> }) => 
      api.updateRepository(fullName, updates),
    onSuccess: (_, { fullName }) => {
      queryClient.invalidateQueries({ queryKey: ['repository', fullName] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
    },
  });
}

export function useUnlockRepository() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (fullName: string) => api.unlockRepository(fullName),
    onSuccess: (_, fullName) => {
      queryClient.invalidateQueries({ queryKey: ['repository', fullName] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
    },
  });
}

export function useRollbackRepository() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ fullName, reason }: { fullName: string; reason?: string }) => 
      api.rollbackRepository(fullName, reason),
    onSuccess: (_, { fullName }) => {
      queryClient.invalidateQueries({ queryKey: ['repository', fullName] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
      queryClient.invalidateQueries({ queryKey: ['migrationHistory'] });
    },
  });
}

export function useMarkRepositoryWontMigrate() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ fullName, unmark }: { fullName: string; unmark?: boolean }) => 
      api.markRepositoryWontMigrate(fullName, unmark),
    onSuccess: (_, { fullName }) => {
      queryClient.invalidateQueries({ queryKey: ['repository', fullName] });
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
      queryClient.invalidateQueries({ queryKey: ['batches'] });
    },
  });
}

export function useBatchUpdateRepositoryStatus() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ 
      repositoryIds, 
      action,
      reason
    }: { 
      repositoryIds: number[]; 
      action: 'mark_migrated' | 'mark_wont_migrate' | 'unmark_wont_migrate' | 'rollback';
      reason?: string;
    }) => api.batchUpdateRepositoryStatus(repositoryIds, action, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
      queryClient.invalidateQueries({ queryKey: ['batches'] });
      queryClient.invalidateQueries({ queryKey: ['organizations'] });
    },
  });
}

// User Mapping mutations
export function useCreateUserMapping() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (mapping: { source_login: string; destination_login?: string; destination_email?: string }) => 
      api.createUserMapping(mapping),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

export function useUpdateUserMapping() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ sourceLogin, updates }: { 
      sourceLogin: string; 
      updates: { 
        destination_login?: string; 
        destination_email?: string; 
        mapping_status?: 'unmapped' | 'mapped' | 'reclaimed' | 'skipped';
      } 
    }) => api.updateUserMapping(sourceLogin, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

export function useDeleteUserMapping() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (sourceLogin: string) => api.deleteUserMapping(sourceLogin),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

export function useImportUserMappings() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (file: File) => api.importUserMappings(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

export function useSyncUserMappings() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: () => api.syncUserMappings(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

export function useFetchMannequins() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ destinationOrg, emuShortcode }: { destinationOrg: string; emuShortcode?: string }) => 
      api.fetchMannequins(destinationOrg, emuShortcode),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

export function useSendAttributionInvitation() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ sourceLogin, destinationOrg }: { sourceLogin: string; destinationOrg: string }) =>
      api.sendAttributionInvitation(sourceLogin, destinationOrg),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

export function useBulkSendAttributionInvitations() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ destinationOrg, sourceLogins }: { destinationOrg: string; sourceLogins?: string[] }) =>
      api.bulkSendAttributionInvitations(destinationOrg, sourceLogins),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userMappings'] });
      queryClient.invalidateQueries({ queryKey: ['userMappingStats'] });
    },
  });
}

// Team Mapping mutations
export function useCreateTeamMapping() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (mapping: { source_org: string; source_team_slug: string; destination_org?: string; destination_team_slug?: string }) =>
      api.createTeamMapping(mapping),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
    },
  });
}

export function useUpdateTeamMapping() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ sourceOrg, sourceTeamSlug, updates }: { 
      sourceOrg: string; 
      sourceTeamSlug: string; 
      updates: { 
        destination_org?: string; 
        destination_team_slug?: string; 
        mapping_status?: 'unmapped' | 'mapped' | 'skipped';
      } 
    }) => api.updateTeamMapping(sourceOrg, sourceTeamSlug, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
    },
  });
}

export function useDeleteTeamMapping() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ sourceOrg, sourceTeamSlug }: { sourceOrg: string; sourceTeamSlug: string }) =>
      api.deleteTeamMapping(sourceOrg, sourceTeamSlug),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
    },
  });
}

export function useImportTeamMappings() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (file: File) => api.importTeamMappings(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
    },
  });
}

export function useSyncTeamMappings() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: () => api.syncTeamMappings(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
    },
  });
}

// Team migration execution mutations
export function useExecuteTeamMigration() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (options?: { source_org?: string; source_team_slug?: string; dry_run?: boolean }) => api.executeTeamMigration(options),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMigrationStatus'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
    },
  });
}

export function useCancelTeamMigration() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: () => api.cancelTeamMigration(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMigrationStatus'] });
    },
  });
}

export function useResetTeamMigrationStatus() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (sourceOrg?: string) => api.resetTeamMigrationStatus(sourceOrg),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['teamMigrationStatus'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappings'] });
      queryClient.invalidateQueries({ queryKey: ['teamMappingStats'] });
    },
  });
}

