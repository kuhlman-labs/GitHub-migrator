/**
 * Team mapping types for team migration between source and destination.
 */

export interface GitHubTeam {
  id: number;
  organization: string;
  slug: string;
  name: string;
  description?: string;
  privacy: string;
  full_slug: string;
}

export interface GitHubTeamMember {
  id: number;
  team_id: number;
  login: string;
  role: 'member' | 'maintainer';
  discovered_at: string;
}

export type TeamMappingStatus = 'unmapped' | 'mapped' | 'skipped';
export type TeamMigrationStatus = 'pending' | 'in_progress' | 'completed' | 'failed';
export type TeamMigrationCompleteness = 'pending' | 'team_only' | 'partial' | 'complete' | 'needs_sync';

export interface TeamMapping {
  id: number;
  organization: string;
  slug: string;
  name: string;
  description?: string;
  privacy: string;
  source_id?: number; // Added for multi-source support
  destination_org?: string;
  destination_team_slug?: string;
  destination_team_name?: string;
  mapping_status: TeamMappingStatus;
  migration_status: string;
  repos_synced: number;
  repos_eligible: number;
  total_source_repos: number;
  team_created_in_dest: boolean;
  sync_status: TeamMigrationCompleteness;
  // Legacy aliases
  source_org?: string;
  source_team_slug?: string;
  source_team_name?: string;
}

export interface TeamMappingStats {
  total: number;
  mapped: number;
  unmapped: number;
  skipped: number;
}

export interface TeamMappingSuggestion {
  source_full_slug: string;
  destination_full_slug: string;
  match_reason: string;
  confidence_percent: number;
}

export interface TeamMigrationProgress {
  total_teams: number;
  processed_teams: number;
  created_teams: number;
  skipped_teams: number;
  failed_teams: number;
  total_repos_synced: number;
  started_at: string;
  completed_at?: string;
  current_team?: string;
  status: string;
  errors?: string[];
}

export interface TeamMigrationExecutionStats {
  pending: number;
  in_progress: number;
  completed: number;
  failed: number;
  needs_sync: number;
  team_only: number;
  partial: number;
  total_repos_synced: number;
  total_repos_eligible: number;
}

export interface TeamMigrationStatusResponse {
  is_running: boolean;
  progress?: TeamMigrationProgress;
  execution_stats: TeamMigrationExecutionStats;
  mapping_stats: TeamMappingStats;
}

export interface TeamDetailMember {
  login: string;
  role: 'member' | 'maintainer';
}

export interface TeamDetailRepository {
  full_name: string;
  permission: 'pull' | 'triage' | 'push' | 'maintain' | 'admin';
  migration_status?: string;
}

export interface TeamDetailMapping {
  destination_org?: string;
  destination_team_slug?: string;
  mapping_status: TeamMappingStatus;
  migration_status?: TeamMigrationStatus;
  migrated_at?: string;
  repos_synced?: number;
  error_message?: string;
  total_source_repos: number;
  repos_eligible: number;
  team_created_in_dest: boolean;
  last_synced_at?: string;
  migration_completeness: TeamMigrationCompleteness;
  sync_status?: TeamMigrationCompleteness;
}

export interface TeamDetail {
  id: number;
  organization: string;
  slug: string;
  name: string;
  description?: string;
  privacy: string;
  source_id?: number; // Added for multi-source support
  discovered_at: string;
  members: TeamDetailMember[];
  repositories: TeamDetailRepository[];
  mapping?: TeamDetailMapping;
}

