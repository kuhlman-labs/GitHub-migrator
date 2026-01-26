import { client } from './client';

// Authorization rules response
export interface AuthorizationRulesResponse {
  migration_admin_teams: string[];
  allow_org_admin_migrations: boolean;
  allow_enterprise_admin_migrations: boolean;
  enable_self_service: boolean;
}

// Settings types
export interface SettingsResponse {
  id: number;
  destination_base_url: string;
  destination_token_configured: boolean;
  destination_app_id?: number;
  destination_app_key_configured: boolean;
  destination_app_installation_id?: number;
  destination_enterprise_slug?: string;
  migration_workers: number;
  migration_poll_interval_seconds: number;
  migration_dest_repo_exists_action: string;
  migration_visibility_public: string;
  migration_visibility_internal: string;
  auth_enabled: boolean;
  auth_github_oauth_client_id?: string;
  auth_github_oauth_client_secret_set: boolean;
  auth_session_secret_set: boolean;
  auth_session_duration_hours: number;
  auth_callback_url?: string;
  auth_frontend_url: string;
  authorization_rules: AuthorizationRulesResponse;
  // Copilot SDK settings
  copilot_enabled: boolean;
  copilot_require_license: boolean;
  copilot_cli_path?: string;
  copilot_cli_configured: boolean;
  copilot_model?: string;
  copilot_session_timeout_min: number;
  copilot_streaming: boolean;
  copilot_log_level: string;
  destination_configured: boolean;
  updated_at: string;
}

export interface SetupProgressResponse {
  destination_configured: boolean;
  sources_configured: boolean;
  source_count: number;
  batches_created: boolean;
  batch_count: number;
  setup_complete: boolean;
}

// Update authorization rules request
export interface UpdateAuthorizationRulesRequest {
  migration_admin_teams?: string[];
  allow_org_admin_migrations?: boolean;
  allow_enterprise_admin_migrations?: boolean;
  enable_self_service?: boolean;
}

export interface UpdateSettingsRequest {
  destination_base_url?: string;
  destination_token?: string;
  destination_app_id?: number;
  destination_app_private_key?: string;
  destination_app_installation_id?: number;
  destination_enterprise_slug?: string;
  migration_workers?: number;
  migration_poll_interval_seconds?: number;
  migration_dest_repo_exists_action?: string;
  migration_visibility_public?: string;
  migration_visibility_internal?: string;
  auth_enabled?: boolean;
  auth_github_oauth_client_id?: string;
  auth_github_oauth_client_secret?: string;
  auth_session_secret?: string;
  auth_session_duration_hours?: number;
  auth_callback_url?: string;
  auth_frontend_url?: string;
  authorization_rules?: UpdateAuthorizationRulesRequest;
  // Copilot SDK settings
  copilot_enabled?: boolean;
  copilot_require_license?: boolean;
  copilot_cli_path?: string;
  copilot_model?: string;
  copilot_session_timeout_min?: number;
  copilot_streaming?: boolean;
  copilot_log_level?: string;
}

export interface ValidateDestinationRequest {
  base_url: string;
  token: string;
  app_id?: number;
  app_private_key?: string;
  app_installation_id?: number;
}

export interface ValidationResponse {
  valid: boolean;
  error?: string;
  warnings?: string[];
  details?: Record<string, unknown>;
}

// Team validation types
export interface TeamValidationResult {
  team: string;
  valid: boolean;
  error?: string;
}

export interface ValidateTeamsRequest {
  teams: string[];
}

export interface ValidateTeamsResponse {
  valid: boolean;
  teams: TeamValidationResult[];
  invalid_teams?: string[];
  error_message?: string;
}

// OAuth validation types
export interface ValidateOAuthRequest {
  oauth_base_url: string;
  oauth_client_id: string;
  callback_url?: string;
  session_secret: string;
  frontend_url?: string;
}

export interface ValidateOAuthResponse {
  valid: boolean;
  error?: string;
  warnings?: string[];
  details?: Record<string, unknown>;
}

// Logging settings types
export interface LoggingSettingsResponse {
  debug_enabled: boolean;
  current_level: string;
  default_level: string;
}

export interface UpdateLoggingRequest {
  debug_enabled?: boolean;
}

// Get current settings (with sensitive data masked)
export async function getSettings(): Promise<SettingsResponse> {
  const response = await client.get<SettingsApiResponse>('/settings');
  // API returns wrapped structure; extract settings
  return response.data.settings;
}

// Get setup progress for guided empty states
export async function getSetupProgress(): Promise<SetupProgressResponse> {
  const response = await client.get<SetupProgressResponse>('/settings/setup-progress');
  return response.data;
}

// Settings API response structure (consistent for both GET and PUT)
export interface SettingsApiResponse {
  settings: SettingsResponse;
  restart_required: boolean;
  message: string;
}

// Update settings
export async function updateSettings(request: UpdateSettingsRequest): Promise<SettingsApiResponse> {
  const response = await client.put<SettingsApiResponse>('/settings', request);
  return response.data;
}

// Validate destination connection
export async function validateDestination(request: ValidateDestinationRequest): Promise<ValidationResponse> {
  const response = await client.post<ValidationResponse>('/settings/destination/validate', request);
  return response.data;
}

// Validate that teams exist in the destination GitHub instance
export async function validateTeams(teams: string[]): Promise<ValidateTeamsResponse> {
  const response = await client.post<ValidateTeamsResponse>('/settings/teams/validate', { teams });
  return response.data;
}

// Validate OAuth configuration before enabling auth
export async function validateOAuth(request: ValidateOAuthRequest): Promise<ValidateOAuthResponse> {
  const response = await client.post<ValidateOAuthResponse>('/settings/oauth/validate', request);
  return response.data;
}

// Get logging settings
export async function getLoggingSettings(): Promise<LoggingSettingsResponse> {
  const response = await client.get<LoggingSettingsResponse>('/settings/logging');
  return response.data;
}

// Update logging settings
export async function updateLoggingSettings(request: UpdateLoggingRequest): Promise<LoggingSettingsResponse> {
  const response = await client.put<LoggingSettingsResponse>('/settings/logging', request);
  return response.data;
}

// Export as an object for consistency with other API modules
export const settingsApi = {
  getSettings,
  getSetupProgress,
  updateSettings,
  validateDestination,
  validateTeams,
  validateOAuth,
  getLoggingSettings,
  updateLoggingSettings,
};
