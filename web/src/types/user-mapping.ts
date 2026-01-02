/**
 * User mapping types for identity migration between source and destination.
 */

export interface GitHubUser {
  id: number;
  login: string;
  name?: string;
  email?: string;
  avatar_url?: string;
  source_instance: string;
  discovered_at: string;
  updated_at: string;
  commit_count: number;
  issue_count: number;
  pr_count: number;
  comment_count: number;
  repository_count: number;
}

export type UserMappingStatus = 'unmapped' | 'mapped' | 'reclaimed' | 'skipped';
export type ReclaimStatus = 'pending' | 'invited' | 'completed' | 'failed';

export interface UserMapping {
  id: number;
  login: string;
  name?: string;
  email?: string;
  avatar_url?: string;
  source_instance: string;
  source_id?: number; // Added for multi-source support
  source_org?: string;
  destination_login?: string;
  mapping_status: UserMappingStatus;
  mannequin_id?: string;
  mannequin_login?: string;
  reclaim_status?: ReclaimStatus;
  match_confidence?: number;
  match_reason?: 'email_exact' | 'login_exact' | 'login_contains' | 'name_fuzzy';
  // Legacy aliases
  source_login?: string;
  source_email?: string;
  source_name?: string;
}

export interface UserMappingStats {
  total: number;
  mapped: number;
  unmapped: number;
  skipped: number;
  reclaimed: number;
  pending_reclaim: number;
  invitable: number;
}

export interface UserStats {
  total_users: number;
  users_with_email: number;
  total_commits: number;
  total_prs: number;
  total_issues: number;
}

export interface SendInvitationResult {
  success: boolean;
  source_login: string;
  mannequin_login?: string;
  target_user?: string;
  message: string;
  error?: string;
}

export interface BulkInvitationResult {
  success: boolean;
  invited: number;
  failed: number;
  skipped: number;
  errors: string[];
  message: string;
}

export interface FetchMannequinsResult {
  total_mannequins: number;
  matched: number;
  unmatched: number;
  message: string;
}

export interface UserMappingSuggestion {
  source_login: string;
  suggested_login?: string;
  suggested_email?: string;
  match_reason: string;
  confidence_percent: number;
}

export interface UserOrgMembership {
  id: number;
  user_login: string;
  organization: string;
  role: 'member' | 'admin';
  discovered_at: string;
}

export interface UserContributionStats {
  commit_count: number;
  issue_count: number;
  pr_count: number;
  comment_count: number;
  repository_count: number;
}

export interface UserMappingDetail {
  source_org?: string;
  destination_login?: string;
  destination_email?: string;
  mapping_status: UserMappingStatus;
  mannequin_id?: string;
  mannequin_login?: string;
  reclaim_status?: ReclaimStatus;
  reclaim_error?: string;
  match_confidence?: number;
  match_reason?: string;
}

export interface UserDetail {
  login: string;
  name?: string;
  email?: string;
  avatar_url?: string;
  source_instance: string;
  source_id?: number; // Added for multi-source support
  discovered_at: string;
  updated_at: string;
  stats: UserContributionStats;
  organizations: UserOrgMembership[];
  mapping?: UserMappingDetail;
}

