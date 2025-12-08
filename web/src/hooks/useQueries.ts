import { useQuery } from '@tanstack/react-query';
import { api } from '../services/api';
import {
  Organization,
  Project,
  Repository,
  Analytics,
  Batch,
  MigrationHistoryEntry,
  RepositoryFilters,
  DashboardActionItems,
  GitHubUser,
  UserMapping,
  UserMappingStats,
  UserStats,
  GitHubTeam,
  GitHubTeamMember,
  TeamMapping,
  TeamMappingStats,
} from '../types';

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

// Dashboard queries
export function useDashboardActionItems() {
  return useQuery<DashboardActionItems, Error>({
    queryKey: ['dashboardActionItems'],
    queryFn: () => api.getDashboardActionItems(),
  });
}

// User queries
interface UserFilters {
  source_instance?: string;
  limit?: number;
  offset?: number;
}

export function useUsers(filters: UserFilters = {}) {
  return useQuery<{ users: GitHubUser[]; total: number }, Error>({
    queryKey: ['users', filters],
    queryFn: () => api.listUsers(filters),
  });
}

export function useUserStats() {
  return useQuery<UserStats, Error>({
    queryKey: ['userStats'],
    queryFn: () => api.getUserStats(),
  });
}

// User mapping queries
interface UserMappingFilters {
  status?: string;
  has_destination?: boolean;
  has_mannequin?: boolean;
  reclaim_status?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export function useUserMappings(filters: UserMappingFilters = {}) {
  return useQuery<{ mappings: UserMapping[]; total: number }, Error>({
    queryKey: ['userMappings', filters],
    queryFn: () => api.listUserMappings(filters),
  });
}

export function useUserMappingStats() {
  return useQuery<UserMappingStats, Error>({
    queryKey: ['userMappingStats'],
    queryFn: () => api.getUserMappingStats(),
  });
}

// Team queries
export function useTeams(organization?: string) {
  return useQuery<GitHubTeam[], Error>({
    queryKey: ['teams', organization],
    queryFn: () => api.listTeams(organization),
  });
}

export function useTeamMembers(org: string, teamSlug: string) {
  return useQuery<{ members: GitHubTeamMember[]; total: number }, Error>({
    queryKey: ['teamMembers', org, teamSlug],
    queryFn: () => api.getTeamMembers(org, teamSlug),
    enabled: !!org && !!teamSlug,
  });
}

// Team mapping queries
interface TeamMappingFilters {
  source_org?: string;
  destination_org?: string;
  status?: string;
  has_destination?: boolean;
  search?: string;
  limit?: number;
  offset?: number;
}

export function useTeamMappings(filters: TeamMappingFilters = {}) {
  return useQuery<{ mappings: TeamMapping[]; total: number }, Error>({
    queryKey: ['teamMappings', filters],
    queryFn: () => api.listTeamMappings(filters),
  });
}

export function useTeamMappingStats() {
  return useQuery<TeamMappingStats, Error>({
    queryKey: ['teamMappingStats'],
    queryFn: () => api.getTeamMappingStats(),
  });
}

