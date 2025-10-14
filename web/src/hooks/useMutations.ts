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

