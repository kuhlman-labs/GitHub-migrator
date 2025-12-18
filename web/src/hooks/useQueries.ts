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
  UserDetail,
  GitHubTeam,
  GitHubTeamMember,
  TeamMapping,
  TeamMappingStats,
  TeamDetail,
  TeamMigrationStatusResponse,
  DiscoveryProgress,
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
  source_org?: string;
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

export function useUserMappingStats(sourceOrg?: string) {
  return useQuery<UserMappingStats, Error>({
    queryKey: ['userMappingStats', sourceOrg || 'all'],
    queryFn: () => api.getUserMappingStats(sourceOrg),
  });
}

export function useUserMappingSourceOrgs() {
  return useQuery<{ organizations: string[] }, Error>({
    queryKey: ['userMappingSourceOrgs'],
    queryFn: () => api.getUserMappingSourceOrgs(),
  });
}

export function useUserDetail(login: string | null) {
  return useQuery<UserDetail, Error>({
    queryKey: ['userDetail', login],
    queryFn: () => api.getUserDetail(login!),
    enabled: !!login,
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

export function useTeamMappingStats(organization?: string) {
  return useQuery<TeamMappingStats, Error>({
    queryKey: ['teamMappingStats', organization || 'all'],
    queryFn: () => api.getTeamMappingStats(organization),
  });
}

// Team source organizations for filter dropdown
export function useTeamSourceOrgs() {
  return useQuery<string[], Error>({
    queryKey: ['teamSourceOrgs'],
    queryFn: () => api.getTeamSourceOrgs(),
  });
}

// Team detail query
export function useTeamDetail(org: string, teamSlug: string) {
  return useQuery<TeamDetail, Error>({
    queryKey: ['teamDetail', org, teamSlug],
    queryFn: () => api.getTeamDetail(org, teamSlug),
    enabled: !!org && !!teamSlug,
  });
}

// Team migration status query
export function useTeamMigrationStatus(enabled = true) {
  return useQuery<TeamMigrationStatusResponse, Error>({
    queryKey: ['teamMigrationStatus'],
    queryFn: () => api.getTeamMigrationStatus(),
    enabled,
    refetchInterval: (query) => {
      // Poll every 2 seconds while migration is running
      if (query.state.data?.is_running) {
        return 2000;
      }
      return false;
    },
  });
}

// Discovery progress query with polling
export function useDiscoveryProgress(enabled = true) {
  return useQuery<DiscoveryProgress | null, Error>({
    queryKey: ['discoveryProgress'],
    queryFn: () => api.getDiscoveryProgress(),
    enabled,
    refetchInterval: (query) => {
      // Poll every 2 seconds while discovery is in progress
      const data = query.state.data;
      if (data?.status === 'in_progress') {
        return 2000;
      }
      return false;
    },
  });
}

