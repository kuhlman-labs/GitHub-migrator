/**
 * Source management API endpoints.
 * Handles CRUD operations for migration sources (GitHub, Azure DevOps).
 */
import { client } from './client';
import type {
  Source,
  CreateSourceRequest,
  UpdateSourceRequest,
  ValidateSourceRequest,
  SourceValidationResponse,
  SetSourceActiveResponse,
} from '../../types/source';
import type { Repository } from '../../types';

export const sourcesApi = {
  /**
   * List all configured sources.
   * @param activeOnly - If true, only returns active sources
   */
  async list(activeOnly?: boolean): Promise<Source[]> {
    const params = activeOnly ? { active: 'true' } : {};
    const { data } = await client.get<Source[]>('/sources', { params });
    return data;
  },

  /**
   * Get a single source by ID.
   */
  async get(id: number): Promise<Source> {
    const { data } = await client.get<Source>(`/sources/${id}`);
    return data;
  },

  /**
   * Create a new source.
   */
  async create(source: CreateSourceRequest): Promise<Source> {
    const { data } = await client.post<Source>('/sources', source);
    return data;
  },

  /**
   * Update an existing source.
   */
  async update(id: number, updates: UpdateSourceRequest): Promise<Source> {
    const { data } = await client.put<Source>(`/sources/${id}`, updates);
    return data;
  },

  /**
   * Delete a source.
   * Will fail if there are repositories associated with the source.
   */
  async delete(id: number): Promise<void> {
    await client.delete(`/sources/${id}`);
  },

  /**
   * Validate a source connection.
   * Can validate either by source_id (for stored credentials) or inline credentials.
   */
  async validate(request: ValidateSourceRequest): Promise<SourceValidationResponse> {
    const endpoint = request.source_id 
      ? `/sources/${request.source_id}/validate`
      : '/sources/validate';
    const { data } = await client.post<SourceValidationResponse>(endpoint, request);
    return data;
  },

  /**
   * Set a source's active status.
   */
  async setActive(id: number, isActive: boolean): Promise<SetSourceActiveResponse> {
    const { data } = await client.post<SetSourceActiveResponse>(
      `/sources/${id}/set-active`,
      { is_active: isActive }
    );
    return data;
  },

  /**
   * Get repositories associated with a source.
   */
  async getRepositories(id: number): Promise<Repository[]> {
    const { data } = await client.get<Repository[]>(`/sources/${id}/repositories`);
    return data;
  },
};

