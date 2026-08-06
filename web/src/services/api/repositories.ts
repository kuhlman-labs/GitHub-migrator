/**
 * Repository-related API endpoints.
 */
import { client } from './client';
import type {
  Repository,
  RepositoryFilters,
  RepositoryListResponse,
  RepositoryDetailResponse,
  DependenciesResponse,
  DependentsResponse,
  DependencyGraphResponse,
} from '../../types';
import type { ELMStatus, MigrationRoute } from '../../types/migration';

export const repositoriesApi = {
  async list(filters?: RepositoryFilters & { source_id?: number }): Promise<RepositoryListResponse> {
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

    if (Array.isArray(data)) {
      return { repositories: data };
    }
    return data;
  },

  async get(fullName: string): Promise<RepositoryDetailResponse> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}`);
    return data;
  },

  async update(fullName: string, updates: Partial<Repository>) {
    const { data } = await client.patch(`/repositories/${encodeURIComponent(fullName)}`, updates);
    return data;
  },

  async rediscover(fullName: string) {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/rediscover`);
    return data;
  },

  async unlock(fullName: string) {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/unlock`);
    return data;
  },

  async rollback(fullName: string, reason?: string): Promise<Repository> {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/rollback`, {
      reason: reason || '',
    });
    return data.repository;
  },

  /**
   * Reads the ELM (Enterprise Live Migrations) state of a repository: its route
   * and, once a live migration exists, its phase, progress and cutover readiness.
   */
  async getELMStatus(fullName: string): Promise<ELMStatus> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}/elm`);
    return data;
  },

  /**
   * Triggers ELM cutover. This is the one irreversible, downtime-causing step of a
   * live migration and is only ever called from a deliberate operator action; the
   * backend refuses with 409 unless it reports the migration ready.
   */
  async cutover(fullName: string): Promise<Repository> {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/cutover`);
    return data.repository;
  },

  /**
   * Sets (or, with null, clears back to the GEI default) the migration route
   * recorded on a repository.
   */
  async setMigrationRoute(fullName: string, route: MigrationRoute | null) {
    const { data } = await client.post(
      `/repositories/${encodeURIComponent(fullName)}/migration-route`,
      { route }
    );
    return data;
  },

  async getDependencies(fullName: string): Promise<DependenciesResponse> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}/dependencies`);
    return data;
  },

  async getDependents(fullName: string): Promise<DependentsResponse> {
    const { data } = await client.get(`/repositories/${encodeURIComponent(fullName)}/dependents`);
    return data;
  },

  async getDependencyGraph(params?: { dependency_type?: string; source_id?: number }): Promise<DependencyGraphResponse> {
    const { data } = await client.get('/dependencies/graph', { params });
    return data;
  },

  async exportDependencies(format: 'csv' | 'json', params?: { dependency_type?: string; source_id?: number }): Promise<Blob> {
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

  async markRemediated(fullName: string) {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/mark-remediated`);
    return data;
  },

  async markWontMigrate(fullName: string, unmark: boolean = false): Promise<Repository> {
    const { data } = await client.post(`/repositories/${encodeURIComponent(fullName)}/mark-wont-migrate`, {
      unmark,
    });
    return data.repository;
  },

  async batchUpdateStatus(
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

  async discover(organization: string) {
    const { data } = await client.post('/repositories/discover', { organization });
    return data;
  },
};

