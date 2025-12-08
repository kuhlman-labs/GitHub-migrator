import axios from 'axios';
import type { 
  Repository, 
  Batch, 
  Analytics, 
  ExecutiveReport,
  RepositoryDetailResponse, 
  MigrationLogsResponse,
  Organization,
  Project,
  GitHubTeam,
  GitHubTeamMember,
  MigrationHistoryEntry,
  RepositoryFilters,
  RepositoryListResponse,
  DependenciesResponse,
  DependentsResponse,
  DependencyGraphResponse,
  SetupStatus,
  SetupConfig,
  ValidationResult,
  DashboardActionItems,
  GitHubUser,
  UserMapping,
  UserMappingStats,
  UserStats,
  TeamMapping,
  TeamMappingStats,
  ImportResult,
} from '../types';

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  withCredentials: true, // Send cookies with requests
});

// Response interceptor to handle 401 errors
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Only redirect if not already on login page or auth endpoints
      const currentPath = window.location.pathname;
      if (!currentPath.includes('/login') && !currentPath.includes('/auth/')) {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

export const api = {
  // Discovery
  async startDiscovery(params: { organization?: string; enterprise_slug?: string; workers?: number }) {
    const { data } = await client.post('/discovery/start', params);
    return data;
  },

  async getDiscoveryStatus() {
    const { data} = await client.get('/discovery/status');
    return data;
  },

  // Azure DevOps Discovery
  async startADODiscovery(params: { organization?: string; project?: string; workers?: number }) {
    const { data } = await client.post('/ado/discover', params);
    return data;
  },

  async getADODiscoveryStatus(organization?: string) {
    const { data } = await client.get('/ado/discovery/status', { params: { organization } });
    return data;
  },

  async listADOProjects(organization?: string) {
    const { data } = await client.get('/ado/projects', { params: { organization } });
    // Backend returns { projects: [...], total: n }, extract the projects array
    return data.projects || [];
  },

  async getADOProject(organization: string, project: string) {
    const { data } = await client.get(`/ado/projects/${encodeURIComponent(organization)}/${encodeURIComponent(project)}`);
    return data;
  },

  // Repositories
  async listRepositories(filters?: RepositoryFilters): Promise<RepositoryListResponse> {
    // Convert array filters to comma-separated strings for API
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const params: Record<string, any> = { ...filters };
    
    if (filters?.status && Array.isArray(filters.status)) {
      params.status = filters.status.join(',');
    }
    
    if (filters?.organization && Array.isArray(filters.organization)) {
      params.organization = filters.organization.join(',');
    }
    
    if (filters?.ado_organization && Array.isArray(filters.ado_organization)) {
      params.ado_organization = filters.ado_organization.join(',');
    }
    
    if (filters?.project && Array.isArray(filters.project)) {
      params.project = filters.project.join(',');
    }
    
    if (filters?.team && Array.isArray(filters.team)) {
      params.team = filters.team.join(',');
    }
    
    if (filters?.complexity && Array.isArray(filters.complexity)) {
      params.complexity = filters.complexity.join(',');
    }
    
    if (filters?.size_category && Array.isArray(filters.size_category)) {
      params.size_category = filters.size_category.join(',');
    }
    
    const { data } = await client.get('/repositories', { params });
    
    // Handle both old and new response formats
    if (Array.isArray(data)) {
      return { repositories: data };
    }
    return data;
  },

  async getRepository(fullName: string): Promise<RepositoryDetailResponse> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}`);
    return data;
  },

  async updateRepository(fullName: string, updates: Partial<Repository>) {
    const { data } = await client.patch(`/repositories/${encodeURIComponent(fullName)}`, updates);
    return data;
  },

  async rediscoverRepository(fullName: string) {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/rediscover`);
    return data;
  },

  async unlockRepository(fullName: string) {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/unlock`);
    return data;
  },

  async rollbackRepository(fullName: string, reason?: string): Promise<Repository> {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/rollback`, {
      reason: reason || '',
    });
    return data.repository;
  },

  async getRepositoryDependencies(fullName: string): Promise<DependenciesResponse> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}/dependencies`);
    return data;
  },

  async getRepositoryDependents(fullName: string): Promise<DependentsResponse> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}/dependents`);
    return data;
  },

  async getDependencyGraph(params?: { dependency_type?: string }): Promise<DependencyGraphResponse> {
    const { data } = await client.get('/dependencies/graph', { params });
    return data;
  },

  async exportDependencies(format: 'csv' | 'json', params?: { dependency_type?: string }): Promise<Blob> {
    const { data } = await client.get('/dependencies/export', {
      params: { format, ...params },
      responseType: 'blob',
    });
    return data;
  },

  async exportRepositoryDependencies(fullName: string, format: 'csv' | 'json'): Promise<Blob> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}/dependencies/export`, {
      params: { format },
      responseType: 'blob',
    });
    return data;
  },

  async markRepositoryRemediated(fullName: string) {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/mark-remediated`);
    return data;
  },

  async markRepositoryWontMigrate(fullName: string, unmark: boolean = false): Promise<Repository> {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/mark-wont-migrate`, {
      unmark,
    });
    return data.repository;
  },

  async batchUpdateRepositoryStatus(
    repositoryIds: number[], 
    action: 'mark_migrated' | 'mark_wont_migrate' | 'unmark_wont_migrate' | 'rollback',
    reason?: string
  ) {
    const { data } = await client.post('/repositories/batch-update', {
      repository_ids: repositoryIds,
      action,
      reason: reason || '',
    });
    return data;
  },

  // Organizations
  async listOrganizations(): Promise<Organization[]> {
    const { data } = await client.get('/organizations');
    return data;
  },

  async listProjects(): Promise<Project[]> {
    const { data } = await client.get('/projects');
    return data;
  },

  async getOrganizationList(): Promise<string[]> {
    const { data } = await client.get('/organizations/list');
    return data;
  },

  // Teams (GitHub only)
  async listTeams(organization?: string): Promise<GitHubTeam[]> {
    const { data } = await client.get('/teams', { params: { organization } });
    return data;
  },

  // Dashboard
  async getDashboardActionItems(): Promise<DashboardActionItems> {
    const { data } = await client.get('/dashboard/action-items');
    return data;
  },

  // Batches
  async listBatches(): Promise<Batch[]> {
    const { data } = await client.get('/batches');
    return data;
  },

  async getBatch(id: number): Promise<Batch> {
    const { data } = await client.get(`/batches/${id}`);
    return data;
  },

  async createBatch(batch: Partial<Batch>): Promise<Batch> {
    const { data } = await client.post('/batches', batch);
    return data;
  },

  async updateBatch(id: number, updates: Partial<Batch>): Promise<Batch> {
    const { data } = await client.patch(`/batches/${id}`, updates);
    return data;
  },

  async deleteBatch(id: number) {
    const { data } = await client.delete(`/batches/${id}`);
    return data;
  },

  async addRepositoriesToBatch(batchId: number, repositoryIds: number[]) {
    const { data } = await client.post(`/batches/${batchId}/repositories`, {
      repository_ids: repositoryIds,
    });
    return data;
  },

  async removeRepositoriesFromBatch(batchId: number, repositoryIds: number[]) {
    const { data } = await client.delete(`/batches/${batchId}/repositories`, {
      data: { repository_ids: repositoryIds },
    });
    return data;
  },

  async retryBatchFailures(batchId: number, repositoryIds?: number[]) {
    const { data } = await client.post(`/batches/${batchId}/retry`, {
      repository_ids: repositoryIds,
    });
    return data;
  },

  async dryRunBatch(id: number, onlyPending?: boolean) {
    const { data } = await client.post(`/batches/${id}/dry-run`, {
      only_pending: onlyPending || false,
    });
    return data;
  },

  async startBatch(id: number, skipDryRun?: boolean) {
    const { data } = await client.post(`/batches/${id}/start`, {
      skip_dry_run: skipDryRun || false,
    });
    return data;
  },

  // Migrations
  async startMigration(params: { 
    repository_ids?: number[];
    full_names?: string[];
    dry_run?: boolean;
    priority?: number;
  }) {
    const { data } = await client.post('/migrations/start', params);
    return data;
  },

  async retryRepository(repositoryId: number, dryRun: boolean = false) {
    const { data } = await client.post('/migrations/start', {
      repository_ids: [repositoryId],
      dry_run: dryRun,
      priority: 0,
    });
    return data;
  },

  async getMigrationStatus(repositoryId: number) {
    const { data } = await client.get(`/migrations/${repositoryId}`);
    return data;
  },

  async getMigrationHistory(repositoryId: number) {
    const { data } = await client.get(`/migrations/${repositoryId}/history`);
    return data;
  },

  async getMigrationLogs(
    repositoryId: number,
    params?: {
      level?: string;
      phase?: string;
      limit?: number;
      offset?: number;
    }
  ): Promise<MigrationLogsResponse> {
    const { data } = await client.get(`/migrations/${repositoryId}/logs`, { params });
    return data;
  },

  // Analytics
  async getAnalyticsSummary(filters?: { organization?: string; batch_id?: string }): Promise<Analytics> {
    const { data } = await client.get('/analytics/summary', { params: filters });
    return data;
  },

  async getMigrationProgress() {
    const { data } = await client.get('/analytics/progress');
    return data;
  },

  async getExecutiveReport(filters?: { organization?: string; batch_id?: string }): Promise<ExecutiveReport> {
    const { data } = await client.get('/analytics/executive-report', { params: filters });
    return data;
  },

  async exportExecutiveReport(format: 'csv' | 'json', filters?: { organization?: string; batch_id?: string }): Promise<Blob> {
    const { data } = await client.get('/analytics/executive-report/export', {
      params: { format, ...filters },
      responseType: 'blob',
    });
    return data;
  },

  async exportDetailedDiscoveryReport(format: 'csv' | 'json', filters?: { organization?: string; project?: string; batch_id?: string }): Promise<Blob> {
    const { data } = await client.get('/analytics/detailed-discovery-report/export', {
      params: { format, ...filters },
      responseType: 'blob',
    });
    return data;
  },

  // Migration History
  async getMigrationHistoryList(): Promise<{ migrations: MigrationHistoryEntry[]; total: number }> {
    const { data } = await client.get('/migrations/history');
    return data;
  },

  async exportMigrationHistory(format: 'csv' | 'json'): Promise<Blob> {
    const { data } = await client.get(`/migrations/history/export?format=${format}`, {
      responseType: 'blob',
    });
    return data;
  },

  // Self-Service
  async selfServiceMigration(params: {
    repositories: string[];
    mappings?: Record<string, string>;
    dry_run: boolean;
  }): Promise<{
    batch_id: number;
    batch_name: string;
    message: string;
    total_repositories: number;
    newly_discovered: number;
    already_existed: number;
    discovery_errors?: string[];
    execution_started: boolean;
  }> {
    const { data } = await client.post('/self-service/migrate', params);
    return data;
  },

  // Configuration
  async getConfig(): Promise<{
    source_type: 'github' | 'azuredevops';
    auth_enabled: boolean;
    entraid_enabled?: boolean;
  }> {
    const { data } = await client.get('/config');
    return data;
  },

  // Authentication
  async getAuthConfig(): Promise<{
    enabled: boolean;
    login_url?: string;
    entraid_login_url?: string;
    authorization_rules?: {
      requires_org_membership?: boolean;
      required_orgs?: string[];
      requires_team_membership?: boolean;
      required_teams?: string[];
      requires_enterprise_admin?: boolean;
      requires_enterprise_membership?: boolean;
      enterprise?: string;
    };
  }> {
    const { data } = await client.get('/auth/config');
    return data;
  },

  async getCurrentUser(): Promise<{
    id: number;
    login: string;
    name: string;
    email: string;
    avatar_url: string;
    roles?: string[];
  }> {
    const { data } = await client.get('/auth/user');
    return data;
  },

  async logout(): Promise<void> {
    await client.post('/auth/logout');
  },

  async refreshToken(): Promise<void> {
    await client.post('/auth/refresh');
  },

  // Setup
  async getSetupStatus(): Promise<SetupStatus> {
    const { data } = await client.get('/setup/status');
    return data;
  },

  async validateSourceConnection(config: SetupConfig['source']): Promise<ValidationResult> {
    const { data } = await client.post('/setup/validate-source', config);
    return data;
  },

  async validateDestinationConnection(config: SetupConfig['destination']): Promise<ValidationResult> {
    const { data } = await client.post('/setup/validate-destination', config);
    return data;
  },

  async validateDatabaseConnection(config: SetupConfig['database']): Promise<ValidationResult> {
    const { data } = await client.post('/setup/validate-database', config);
    return data;
  },

  async applySetup(config: SetupConfig): Promise<void> {
    const { data } = await client.post('/setup/apply', config);
    return data;
  },

  // Users
  async listUsers(filters?: { source_instance?: string; limit?: number; offset?: number }): Promise<{ users: GitHubUser[]; total: number }> {
    const { data } = await client.get('/users', { params: filters });
    return data;
  },

  async getUserStats(): Promise<UserStats> {
    const { data } = await client.get('/users/stats');
    return data;
  },

  // User Mappings
  async listUserMappings(filters?: {
    status?: string;
    has_destination?: boolean;
    has_mannequin?: boolean;
    reclaim_status?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ mappings: UserMapping[]; total: number }> {
    const { data } = await client.get('/user-mappings', { params: filters });
    return data;
  },

  async getUserMappingStats(): Promise<UserMappingStats> {
    const { data } = await client.get('/user-mappings/stats');
    return data;
  },

  async createUserMapping(mapping: Partial<UserMapping>): Promise<UserMapping> {
    const { data } = await client.post('/user-mappings', mapping);
    return data;
  },

  async updateUserMapping(sourceLogin: string, updates: Partial<UserMapping>): Promise<UserMapping> {
    const { data } = await client.patch(`/user-mappings/${encodeURIComponent(sourceLogin)}`, updates);
    return data;
  },

  async deleteUserMapping(sourceLogin: string): Promise<void> {
    await client.delete(`/user-mappings/${encodeURIComponent(sourceLogin)}`);
  },

  async importUserMappings(file: File): Promise<ImportResult> {
    const formData = new FormData();
    formData.append('file', file);
    const { data } = await client.post('/user-mappings/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return data;
  },

  async exportUserMappings(status?: string): Promise<Blob> {
    const { data } = await client.get('/user-mappings/export', {
      params: { status },
      responseType: 'blob',
    });
    return data;
  },

  async generateGEICSV(mannequinsOnly?: boolean): Promise<Blob> {
    const { data } = await client.get('/user-mappings/generate-gei-csv', {
      params: { mannequins_only: mannequinsOnly },
      responseType: 'blob',
    });
    return data;
  },

  async suggestUserMappings(): Promise<{ suggestions: unknown[]; total: number }> {
    const { data } = await client.post('/user-mappings/suggest');
    return data;
  },

  async syncUserMappings(): Promise<{ created: number; message: string }> {
    const { data } = await client.post('/user-mappings/sync');
    return data;
  },

  // Team Members
  async getTeamMembers(org: string, teamSlug: string): Promise<{ members: GitHubTeamMember[]; total: number }> {
    const { data } = await client.get(`/teams/${encodeURIComponent(org)}/${encodeURIComponent(teamSlug)}/members`);
    return data;
  },

  // Team Mappings
  async listTeamMappings(filters?: {
    source_org?: string;
    destination_org?: string;
    status?: string;
    has_destination?: boolean;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ mappings: TeamMapping[]; total: number }> {
    const { data } = await client.get('/team-mappings', { params: filters });
    return data;
  },

  async getTeamMappingStats(): Promise<TeamMappingStats> {
    const { data } = await client.get('/team-mappings/stats');
    return data;
  },

  async createTeamMapping(mapping: Partial<TeamMapping>): Promise<TeamMapping> {
    const { data } = await client.post('/team-mappings', mapping);
    return data;
  },

  async updateTeamMapping(sourceOrg: string, sourceTeamSlug: string, updates: Partial<TeamMapping>): Promise<TeamMapping> {
    const { data } = await client.patch(`/team-mappings/${encodeURIComponent(sourceOrg)}/${encodeURIComponent(sourceTeamSlug)}`, updates);
    return data;
  },

  async deleteTeamMapping(sourceOrg: string, sourceTeamSlug: string): Promise<void> {
    await client.delete(`/team-mappings/${encodeURIComponent(sourceOrg)}/${encodeURIComponent(sourceTeamSlug)}`);
  },

  async importTeamMappings(file: File): Promise<ImportResult> {
    const formData = new FormData();
    formData.append('file', file);
    const { data } = await client.post('/team-mappings/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return data;
  },

  async exportTeamMappings(filters?: { status?: string; source_org?: string }): Promise<Blob> {
    const { data } = await client.get('/team-mappings/export', {
      params: filters,
      responseType: 'blob',
    });
    return data;
  },

  async suggestTeamMappings(destinationOrg: string, destTeamSlugs?: string[]): Promise<{ suggestions: unknown[]; total: number }> {
    const { data } = await client.post('/team-mappings/suggest', {
      destination_org: destinationOrg,
      dest_team_slugs: destTeamSlugs,
    });
    return data;
  },

  async syncTeamMappings(): Promise<{ created: number; message: string }> {
    const { data } = await client.post('/team-mappings/sync');
    return data;
  },
};

