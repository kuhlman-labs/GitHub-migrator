/**
 * Types for multi-source configuration management.
 */

export type SourceType = 'github' | 'azuredevops';

/**
 * Source represents a configured migration source.
 * This is the response format from the API (with masked token).
 */
export interface Source {
  id: number;
  name: string;
  type: SourceType;
  base_url: string;
  organization?: string;
  has_app_auth: boolean;
  has_oauth: boolean;  // True if OAuth is configured for user self-service
  app_id?: number;
  is_active: boolean;
  repository_count: number;
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
  masked_token: string;
}

/**
 * Request payload for creating a new source.
 */
export interface CreateSourceRequest {
  name: string;
  type: SourceType;
  base_url: string;
  token: string;
  organization?: string;
  app_id?: number;
  app_private_key?: string;
  app_installation_id?: number;
  // OAuth configuration for user self-service (GitHub/GHES sources)
  oauth_client_id?: string;
  oauth_client_secret?: string;
  // OAuth configuration for user self-service (Azure DevOps sources via Entra ID)
  entra_tenant_id?: string;
  entra_client_id?: string;
  entra_client_secret?: string;
}

/**
 * Request payload for updating an existing source.
 */
export interface UpdateSourceRequest {
  name?: string;
  base_url?: string;
  token?: string;
  organization?: string;
  app_id?: number;
  app_private_key?: string;
  app_installation_id?: number;
  is_active?: boolean;
  // OAuth configuration for user self-service (GitHub/GHES sources)
  oauth_client_id?: string;
  oauth_client_secret?: string;
  // OAuth configuration for user self-service (Azure DevOps sources via Entra ID)
  entra_tenant_id?: string;
  entra_client_id?: string;
  entra_client_secret?: string;
}

/**
 * Request payload for validating a source connection.
 */
export interface ValidateSourceRequest {
  source_id?: number;
  type?: SourceType;
  base_url?: string;
  token?: string;
  organization?: string;
}

/**
 * Response from source validation.
 */
export interface SourceValidationResponse {
  valid: boolean;
  error?: string;
  warnings?: string[];
  details?: Record<string, unknown>;
}

/**
 * Filter for source list view.
 * 'all' shows all sources, a number filters to a specific source ID.
 */
export type SourceFilter = 'all' | number;

/**
 * Response from set-active endpoint.
 */
export interface SetSourceActiveResponse {
  success: boolean;
  source_id: number;
  is_active: boolean;
}

