/**
 * API service - re-exports from the modular api/ directory.
 * This file maintains backwards compatibility for existing imports.
 * 
 * For new code, prefer importing from './api' directory directly:
 *   import { api } from '../services/api';
 *   import { repositoriesApi } from '../services/api/repositories';
 */
export { api } from './api/index';
export { repositoriesApi } from './api/repositories';
export { batchesApi } from './api/batches';
export { usersApi } from './api/users';
export { teamsApi } from './api/teams';
export { discoveryApi } from './api/discovery';
export { migrationsApi } from './api/migrations';
export { analyticsApi } from './api/analytics';
export { configApi } from './api/config';
export { client } from './api/client';
