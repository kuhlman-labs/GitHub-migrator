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

// Polling options interface
interface PollingOptions {
  /** Enable polling at specified interval in ms */
  refetchInterval?: number | false;
  /** Only poll when window is focused */
  refetchIntervalInBackground?: boolean;
}

// Organization queries
export function useOrganizations(options?: PollingOptions) {
  return useQuery<Organization[], Error>({
    queryKey: ['organizations'],
    queryFn: () => api.listOrganizations(),
    refetchInterval: options?.refetchInterval,
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? false,
  });
}

// Config query
export function useConfig() {
  return useQuery<{ source_type: 'github' | 'azuredevops'; auth_enabled: boolean; entraid_enabled?: boolean }, Error>({
    queryKey: ['config'],
    queryFn: () => api.getConfig(),
    staleTime: 5 * 60 * 1000, // Config rarely changes, cache for 5 minutes
  });
}

// Project queries
export function useProjects() {
  return useQuery<Project[], Error>({
    queryKey: ['projects'],
    queryFn: () => api.listProjects(),
  });
}

// ADO Projects query
interface ADOProject {
  name?: string;
  project_name?: string;
}

export function useADOProjects(organization?: string) {
  return useQuery<string[], Error>({
    queryKey: ['adoProjects', organization],
    queryFn: async () => {
      const projectList = await api.listADOProjects(organization);
      return projectList.map((p: ADOProject) => p.name || p.project_name || '').filter(Boolean);
    },
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

export function useRepositoryWithHistory(fullName: string) {
  return useQuery<{ repository: Repository; history: import('../types').MigrationHistory[] }, Error>({
    queryKey: ['repository-with-history', fullName],
    queryFn: async () => {
      const response = await api.getRepository(fullName);
      return { repository: response.repository, history: response.history };
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

export function useAnalytics(filters: AnalyticsFilters = {}, options?: PollingOptions) {
  return useQuery<Analytics, Error>({
    queryKey: ['analytics', filters],
    queryFn: () => api.getAnalyticsSummary(filters),
    refetchInterval: options?.refetchInterval,
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? false,
  });
}

// Batch queries
export function useBatches(options?: PollingOptions) {
  return useQuery<Batch[], Error>({
    queryKey: ['batches'],
    queryFn: () => api.listBatches(),
    refetchInterval: options?.refetchInterval,
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? false,
  });
}

export function useBatch(id: number) {
  return useQuery<Batch, Error>({
    queryKey: ['batch', id],
    queryFn: () => api.getBatch(id),
    enabled: !!id,
  });
}

export function useBatchRepositories(batchId: number | null, options?: PollingOptions) {
  return useQuery<{ repositories: Repository[]; total?: number }, Error>({
    queryKey: ['batchRepositories', batchId],
    queryFn: () => api.listRepositories({ batch_id: batchId! }),
    enabled: !!batchId,
    refetchInterval: options?.refetchInterval,
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? false,
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
export function useDashboardActionItems(options?: PollingOptions) {
  return useQuery<DashboardActionItems, Error>({
    queryKey: ['dashboardActionItems'],
    queryFn: () => api.getDashboardActionItems(),
    refetchInterval: options?.refetchInterval,
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? false,
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

// Setup status query
export function useSetupStatus() {
  return useQuery<{ setup_completed: boolean; completed_at?: string }, Error>({
    queryKey: ['setupStatus'],
    queryFn: () => api.getSetupStatus(),
    staleTime: Infinity, // Only fetch once per session unless invalidated
    retry: 1, // Only retry once on failure
  });
}

// Setup progress query for guided empty states
export function useSetupProgress() {
  return useQuery<{
    destination_configured: boolean;
    sources_configured: boolean;
    source_count: number;
    batches_created: boolean;
    batch_count: number;
    setup_complete: boolean;
  }, Error>({
    queryKey: ['setupProgress'],
    queryFn: () => api.getSetupProgress(),
    staleTime: 30000, // Refetch every 30 seconds
  });
}

// Discovery progress query with polling
export function useDiscoveryProgress(enabled = true) {
  return useQuery<DiscoveryProgress | null, Error>({
    queryKey: ['discoveryProgress'],
    queryFn: () => api.getDiscoveryProgress(),
    enabled,
    staleTime: 0, // Always refetch when invalidated
    refetchInterval: (query) => {
      // Poll every 1 second while discovery is in progress for real-time updates
      const data = query.state.data;
      if (data?.status === 'in_progress') {
        return 1000;
      }
      // Poll every 30 seconds when idle to detect new discoveries started elsewhere
      return 30000;
    },
  });
}

