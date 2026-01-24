/**
 * Unified API module - maintains backwards compatibility while organizing
 * endpoints into domain-specific modules.
 *
 * Usage (unchanged from before):
 *   import { api } from '../services/api';
 *   api.listRepositories(filters);
 *
 * Domain modules can also be imported directly if preferred:
 *   import { repositoriesApi } from '../services/api/repositories';
 */

import { repositoriesApi } from './repositories';
import { batchesApi } from './batches';
import { usersApi } from './users';
import { teamsApi } from './teams';
import { discoveryApi } from './discovery';
import { migrationsApi } from './migrations';
import { analyticsApi } from './analytics';
import { configApi } from './config';
import { sourcesApi } from './sources';
import { settingsApi } from './settings';
import { copilotApi } from './copilot';

// Export domain-specific APIs for direct access
export { repositoriesApi } from './repositories';
export { batchesApi } from './batches';
export { usersApi } from './users';
export { teamsApi } from './teams';
export { discoveryApi } from './discovery';
export { migrationsApi } from './migrations';
export { analyticsApi } from './analytics';
export { configApi } from './config';
export { sourcesApi } from './sources';
export { settingsApi } from './settings';
export { copilotApi } from './copilot';
export { client } from './client';

// Unified API object for backwards compatibility
export const api = {
  // Discovery
  startDiscovery: discoveryApi.start,
  getDiscoveryStatus: discoveryApi.getStatus,
  getDiscoveryProgress: discoveryApi.getProgress,
  cancelDiscovery: discoveryApi.cancel,
  discoverRepositories: repositoriesApi.discover,
  discoverOrgMembers: usersApi.discover,
  discoverTeams: teamsApi.discover,

  // Azure DevOps Discovery
  startADODiscovery: discoveryApi.startADO,
  getADODiscoveryStatus: discoveryApi.getADOStatus,
  listADOProjects: discoveryApi.listADOProjects,
  getADOProject: discoveryApi.getADOProject,

  // Repositories
  listRepositories: repositoriesApi.list,
  getRepository: repositoriesApi.get,
  updateRepository: repositoriesApi.update,
  rediscoverRepository: repositoriesApi.rediscover,
  unlockRepository: repositoriesApi.unlock,
  rollbackRepository: repositoriesApi.rollback,
  getRepositoryDependencies: repositoriesApi.getDependencies,
  getRepositoryDependents: repositoriesApi.getDependents,
  getDependencyGraph: repositoriesApi.getDependencyGraph,
  exportDependencies: repositoriesApi.exportDependencies,
  exportRepositoryDependencies: repositoriesApi.exportRepositoryDependencies,
  markRepositoryRemediated: repositoriesApi.markRemediated,
  markRepositoryWontMigrate: repositoriesApi.markWontMigrate,
  batchUpdateRepositoryStatus: repositoriesApi.batchUpdateStatus,

  // Organizations
  listOrganizations: discoveryApi.listOrganizations,
  listProjects: discoveryApi.listProjects,
  getOrganizationList: discoveryApi.getOrganizationList,
  listTeams: discoveryApi.listTeams,

  // Dashboard
  getDashboardActionItems: analyticsApi.getDashboardActionItems,

  // Batches
  listBatches: batchesApi.list,
  getBatch: batchesApi.get,
  createBatch: batchesApi.create,
  updateBatch: batchesApi.update,
  deleteBatch: batchesApi.delete,
  addRepositoriesToBatch: batchesApi.addRepositories,
  removeRepositoriesFromBatch: batchesApi.removeRepositories,
  retryBatchFailures: batchesApi.retryFailures,
  dryRunBatch: batchesApi.dryRun,
  startBatch: batchesApi.start,

  // Migrations
  startMigration: migrationsApi.start,
  retryRepository: migrationsApi.retryRepository,
  getMigrationStatus: migrationsApi.getStatus,
  getMigrationHistory: migrationsApi.getHistory,
  getMigrationLogs: migrationsApi.getLogs,
  getMigrationHistoryList: migrationsApi.getHistoryList,
  exportMigrationHistory: migrationsApi.exportHistory,
  selfServiceMigration: migrationsApi.selfService,

  // Analytics
  getAnalyticsSummary: analyticsApi.getSummary,
  getMigrationProgress: analyticsApi.getProgress,
  getExecutiveReport: analyticsApi.getExecutiveReport,
  exportExecutiveReport: analyticsApi.exportExecutiveReport,
  exportDetailedDiscoveryReport: analyticsApi.exportDetailedDiscoveryReport,

  // Configuration
  getConfig: configApi.getConfig,
  getAuthConfig: configApi.getAuthConfig,
  getCurrentUser: configApi.getCurrentUser,
  logout: configApi.logout,
  refreshToken: configApi.refreshToken,

  // Setup
  getSetupStatus: configApi.getSetupStatus,
  validateSourceConnection: configApi.validateSourceConnection,
  validateDestinationConnection: configApi.validateDestinationConnection,
  validateDatabaseConnection: configApi.validateDatabaseConnection,
  applySetup: configApi.applySetup,

  // Users
  listUsers: usersApi.list,
  getUserStats: usersApi.getStats,

  // User Mappings
  listUserMappings: usersApi.listMappings,
  getUserMappingStats: usersApi.getMappingStats,
  getUserDetail: usersApi.getDetail,
  getUserMappingSourceOrgs: usersApi.getSourceOrgs,
  createUserMapping: usersApi.createMapping,
  updateUserMapping: usersApi.updateMapping,
  deleteUserMapping: usersApi.deleteMapping,
  importUserMappings: usersApi.importMappings,
  exportUserMappings: usersApi.exportMappings,
  generateGEICSV: usersApi.generateGEICSV,
  suggestUserMappings: usersApi.suggestMappings,
  syncUserMappings: usersApi.syncMappings,
  fetchMannequins: usersApi.fetchMannequins,
  sendAttributionInvitation: usersApi.sendAttributionInvitation,
  bulkSendAttributionInvitations: usersApi.bulkSendAttributionInvitations,

  // Team Members
  getTeamMembers: teamsApi.getMembers,

  // Team Mappings
  listTeamMappings: teamsApi.listMappings,
  getTeamMappingStats: teamsApi.getMappingStats,
  getTeamSourceOrgs: teamsApi.getSourceOrgs,
  createTeamMapping: teamsApi.createMapping,
  updateTeamMapping: teamsApi.updateMapping,
  deleteTeamMapping: teamsApi.deleteMapping,
  importTeamMappings: teamsApi.importMappings,
  exportTeamMappings: teamsApi.exportMappings,
  suggestTeamMappings: teamsApi.suggestMappings,
  syncTeamMappings: teamsApi.syncMappings,

  // Team Detail
  getTeamDetail: teamsApi.getDetail,

  // Team Migration Execution
  executeTeamMigration: teamsApi.executeMigration,
  getTeamMigrationStatus: teamsApi.getMigrationStatus,
  cancelTeamMigration: teamsApi.cancelMigration,
  resetTeamMigrationStatus: teamsApi.resetMigrationStatus,

  // Sources (multi-source management)
  listSources: sourcesApi.list,
  getSource: sourcesApi.get,
  createSource: sourcesApi.create,
  updateSource: sourcesApi.update,
  deleteSource: sourcesApi.delete,
  validateSource: sourcesApi.validate,
  setSourceActive: sourcesApi.setActive,
  getSourceRepositories: sourcesApi.getRepositories,

  // Settings (dynamic configuration)
  getSettings: settingsApi.getSettings,
  getSetupProgress: settingsApi.getSetupProgress,
  updateSettings: settingsApi.updateSettings,
  validateDestination: settingsApi.validateDestination,

  // Copilot
  getCopilotStatus: copilotApi.getStatus,
  sendCopilotMessage: copilotApi.sendMessage,
  getCopilotSessions: copilotApi.getSessions,
  getCopilotSessionHistory: copilotApi.getSessionHistory,
  deleteCopilotSession: copilotApi.deleteSession,
  validateCopilotCLI: copilotApi.validateCLI,
};

