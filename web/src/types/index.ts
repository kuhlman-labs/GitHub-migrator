/**
 * Type exports - barrel file for backwards compatibility.
 * Types are organized in domain-specific modules for better maintainability.
 */

// Repository types
export type {
  Repository,
  ComplexityBreakdown,
  RepositoryFilters,
  RepositoryListResponse,
  DependencyType,
  RepositoryDependency,
  DependencySummary,
  DependenciesResponse,
  DependentRepository,
  DependentsResponse,
  DependencyGraphNode,
  DependencyGraphEdge,
  DependencyGraphStats,
  DependencyGraphResponse,
  DependencyExportRow,
  ImportedMigrationSettings,
  ImportedRepository,
} from './repository';

// Batch types
export type { Batch, BatchStatus } from './batch';
export { getBatchDuration, formatBatchDuration } from './batch';

// Migration types
export type {
  MigrationHistory,
  MigrationLog,
  MigrationHistoryEntry,
  RepositoryDetailResponse,
  MigrationLogsResponse,
  RollbackRequest,
  MigrationStatus,
} from './migration';

// User mapping types
export type {
  GitHubUser,
  UserMappingStatus,
  ReclaimStatus,
  UserMapping,
  UserMappingStats,
  UserStats,
  SendInvitationResult,
  BulkInvitationResult,
  FetchMannequinsResult,
  UserMappingSuggestion,
  UserOrgMembership,
  UserContributionStats,
  UserMappingDetail,
  UserDetail,
} from './user-mapping';

// Team mapping types
export type {
  GitHubTeam,
  GitHubTeamMember,
  TeamMappingStatus,
  TeamMigrationStatus,
  TeamMigrationCompleteness,
  TeamMapping,
  TeamMappingStats,
  TeamMappingSuggestion,
  TeamMigrationProgress,
  TeamMigrationExecutionStats,
  TeamMigrationStatusResponse,
  TeamDetailMember,
  TeamDetailRepository,
  TeamDetailMapping,
  TeamDetail,
} from './team-mapping';

// Analytics types
export type {
  SizeDistribution,
  MigrationCompletionStats,
  ComplexityDistribution,
  MigrationVelocity,
  MigrationTimeSeriesPoint,
  Analytics,
  ExecutiveSummary,
  VelocityMetrics,
  RiskAnalysis,
  ADORiskAnalysis,
  BatchPerformance,
  ExecutiveReport,
} from './analytics';

// Common/shared types
export type {
  Organization,
  Project,
  ADOProject,
  ADODiscoveryRequest,
  ADODiscoveryStatus,
  FeatureStats,
  FailedRepository,
  DashboardActionItems,
  SetupStatus,
  MaskedConfigData,
  SetupConfig,
  ValidationResult,
  ImportResult,
  DiscoveryPhase,
  DiscoveryStatus,
  DiscoveryType,
  DiscoveryProgress,
} from './common';
