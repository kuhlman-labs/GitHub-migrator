import { useQuery } from '@tanstack/react-query';
import { api } from '../services/api';
import { Organization, Project, Repository, Analytics, Batch, MigrationHistoryEntry, RepositoryFilters } from '../types';

// Organization queries
export function useOrganizations() {
  return useQuery<Organization[], Error>({
    queryKey: ['organizations'],
    queryFn: () => api.listOrganizations(),
  });
}

// Project queries
export function useProjects() {
  return useQuery<Project[], Error>({
    queryKey: ['projects'],
    queryFn: () => api.listProjects(),
  });
}

// Repository queries
export function useRepositories(filters: RepositoryFilters = {}) {
  return useQuery<{ repositories: Repository[]; total?: number }, Error>({
    queryKey: ['repositories', filters],
    queryFn: () => api.listRepositories(filters),
  });
}

export function useRepository(fullName: string) {
  return useQuery<Repository, Error>({
    queryKey: ['repository', fullName],
    queryFn: async () => {
      const response = await api.getRepository(fullName);
      return response.repository;
    },
    enabled: !!fullName,
  });
}

// Analytics queries
interface AnalyticsFilters {
  organization?: string;
  project?: string;
  batch_id?: string;
}

export function useAnalytics(filters: AnalyticsFilters = {}) {
  return useQuery<Analytics, Error>({
    queryKey: ['analytics', filters],
    queryFn: () => api.getAnalyticsSummary(filters),
  });
}

// Batch queries
export function useBatches() {
  return useQuery<Batch[], Error>({
    queryKey: ['batches'],
    queryFn: () => api.listBatches(),
  });
}

export function useBatch(id: number) {
  return useQuery<Batch, Error>({
    queryKey: ['batch', id],
    queryFn: () => api.getBatch(id),
    enabled: !!id,
  });
}

// Migration history queries
export function useMigrationHistory() {
  return useQuery<{ migrations: MigrationHistoryEntry[]; total: number }, Error>({
    queryKey: ['migrationHistory'],
    queryFn: () => api.getMigrationHistoryList(),
  });
}

// Discovery status query
export function useDiscoveryStatus() {
  return useQuery<{ status: string; discovered_count: number; is_running: boolean }, Error>({
    queryKey: ['discoveryStatus'],
    queryFn: () => api.getDiscoveryStatus(),
  });
}

