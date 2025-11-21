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
  MigrationHistoryEntry,
  RepositoryFilters,
  RepositoryListResponse,
  DependenciesResponse
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
    const params: Record<string, any> = { ...filters };
    
    if (filters?.status && Array.isArray(filters.status)) {
      params.status = filters.status.join(',');
    }
    
    if (filters?.organization && Array.isArray(filters.organization)) {
      params.organization = filters.organization.join(',');
    }
    
    if (filters?.project && Array.isArray(filters.project)) {
      params.project = filters.project.join(',');
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
};

