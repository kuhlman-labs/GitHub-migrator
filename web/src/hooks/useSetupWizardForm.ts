import { useReducer, useCallback } from 'react';
import type { SetupConfig } from '../types';

// Form state types
export interface SetupWizardFormState {
  // Step 1: Source selection
  sourceType: 'github' | 'azuredevops';

  // Step 2: Source configuration
  sourceBaseURL: string;
  sourceToken: string;
  sourceOrganization: string;
  sourceValidated: boolean;

  // Step 3: Destination configuration
  destBaseURL: string;
  destToken: string;
  destValidated: boolean;

  // Step 4: Database configuration
  dbType: 'sqlite' | 'postgres' | 'sqlserver';
  dbDSN: string;
  dbValidated: boolean;

  // Step 5: Server configuration
  serverPort: number;
  migrationWorkers: number;
  pollInterval: number;

  // Step 6: Migration behavior
  destRepoExistsAction: 'fail' | 'skip' | 'delete';
  publicReposVisibility: 'public' | 'internal' | 'private';
  internalReposVisibility: 'internal' | 'private';

  // Step 7: Logging & Advanced Configuration
  logLevel: 'debug' | 'info' | 'warn' | 'error';
  logFormat: 'json' | 'text';
  logOutputFile: string;

  // GitHub App Discovery for Source
  sourceGithubAppEnabled: boolean;
  sourceGithubAppID: string;
  sourceGithubAppPrivateKey: string;
  sourceGithubAppInstallationID: string;

  // GitHub App Discovery for Destination
  destGithubAppEnabled: boolean;
  destGithubAppID: string;
  destGithubAppPrivateKey: string;
  destGithubAppInstallationID: string;

  // Authentication
  authEnabled: boolean;
  // GitHub OAuth
  oauthClientID: string;
  oauthClientSecret: string;
  oauthBaseURL: string;
  // Azure AD
  azureADTenantID: string;
  azureADClientID: string;
  azureADClientSecret: string;
  // Common auth settings
  callbackURL: string;
  frontendURL: string;
  sessionSecret: string;
  sessionDuration: number;

  // Authorization rules
  requireOrgMembership: string;
  requireTeamMembership: string;
  requireEnterpriseAdmin: boolean;
  requireEnterpriseMembership: boolean;
  enterpriseSlug: string;
  privilegedTeams: string;
}

// Initial state
const initialState: SetupWizardFormState = {
  sourceType: 'github',
  sourceBaseURL: 'https://api.github.com',
  sourceToken: '',
  sourceOrganization: '',
  sourceValidated: false,
  destBaseURL: 'https://api.github.com',
  destToken: '',
  destValidated: false,
  dbType: 'sqlite',
  dbDSN: './data/migrator.db',
  dbValidated: false,
  serverPort: 8080,
  migrationWorkers: 5,
  pollInterval: 30,
  destRepoExistsAction: 'fail',
  publicReposVisibility: 'private',
  internalReposVisibility: 'private',
  logLevel: 'info',
  logFormat: 'json',
  logOutputFile: './logs/migrator.log',
  sourceGithubAppEnabled: false,
  sourceGithubAppID: '',
  sourceGithubAppPrivateKey: '',
  sourceGithubAppInstallationID: '',
  destGithubAppEnabled: false,
  destGithubAppID: '',
  destGithubAppPrivateKey: '',
  destGithubAppInstallationID: '',
  authEnabled: false,
  oauthClientID: '',
  oauthClientSecret: '',
  oauthBaseURL: '',
  azureADTenantID: '',
  azureADClientID: '',
  azureADClientSecret: '',
  callbackURL: '',
  frontendURL: '',
  sessionSecret: '',
  sessionDuration: 24,
  requireOrgMembership: '',
  requireTeamMembership: '',
  requireEnterpriseAdmin: false,
  requireEnterpriseMembership: false,
  enterpriseSlug: '',
  privilegedTeams: '',
};

// Action types
type SetupWizardAction =
  | { type: 'SET_FIELD'; field: keyof SetupWizardFormState; value: SetupWizardFormState[keyof SetupWizardFormState] }
  | { type: 'SET_FIELDS'; fields: Partial<SetupWizardFormState> }
  | { type: 'RESET' };

// Reducer
function setupWizardReducer(state: SetupWizardFormState, action: SetupWizardAction): SetupWizardFormState {
  switch (action.type) {
    case 'SET_FIELD':
      return { ...state, [action.field]: action.value };
    case 'SET_FIELDS':
      return { ...state, ...action.fields };
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

// Hook return type
interface UseSetupWizardFormReturn {
  state: SetupWizardFormState;
  setField: <K extends keyof SetupWizardFormState>(field: K, value: SetupWizardFormState[K]) => void;
  setFields: (fields: Partial<SetupWizardFormState>) => void;
  reset: () => void;
  buildConfig: () => SetupConfig;
}

/**
 * Custom hook for managing SetupWizard form state.
 * Uses a reducer pattern to consolidate 49+ useState calls into a single state object.
 */
export function useSetupWizardForm(): UseSetupWizardFormReturn {
  const [state, dispatch] = useReducer(setupWizardReducer, initialState);

  const setField = useCallback(<K extends keyof SetupWizardFormState>(
    field: K,
    value: SetupWizardFormState[K]
  ) => {
    dispatch({ type: 'SET_FIELD', field, value });
  }, []);

  const setFields = useCallback((fields: Partial<SetupWizardFormState>) => {
    dispatch({ type: 'SET_FIELDS', fields });
  }, []);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
  }, []);

  const buildConfig = useCallback((): SetupConfig => ({
    source: {
      type: state.sourceType,
      base_url: state.sourceBaseURL,
      token: state.sourceToken,
      organization: state.sourceType === 'azuredevops' ? state.sourceOrganization : undefined,
      app_id: state.sourceType === 'github' && state.sourceGithubAppEnabled && state.sourceGithubAppID 
        ? parseInt(state.sourceGithubAppID) : undefined,
      app_private_key: state.sourceType === 'github' && state.sourceGithubAppEnabled && state.sourceGithubAppPrivateKey 
        ? state.sourceGithubAppPrivateKey : undefined,
      app_installation_id: state.sourceType === 'github' && state.sourceGithubAppEnabled && state.sourceGithubAppInstallationID 
        ? parseInt(state.sourceGithubAppInstallationID) : undefined,
    },
    destination: {
      base_url: state.destBaseURL,
      token: state.destToken,
      app_id: state.destGithubAppEnabled && state.destGithubAppID 
        ? parseInt(state.destGithubAppID) : undefined,
      app_private_key: state.destGithubAppEnabled && state.destGithubAppPrivateKey 
        ? state.destGithubAppPrivateKey : undefined,
      app_installation_id: state.destGithubAppEnabled && state.destGithubAppInstallationID 
        ? parseInt(state.destGithubAppInstallationID) : undefined,
    },
    database: {
      type: state.dbType,
      dsn: state.dbDSN,
    },
    server: {
      port: state.serverPort,
    },
    migration: {
      workers: state.migrationWorkers,
      poll_interval_seconds: state.pollInterval,
      dest_repo_exists_action: state.destRepoExistsAction,
      visibility_handling: {
        public_repos: state.publicReposVisibility,
        internal_repos: state.internalReposVisibility,
      },
    },
    logging: {
      level: state.logLevel,
      format: state.logFormat,
      output_file: state.logOutputFile,
    },
    auth: state.authEnabled ? {
      enabled: true,
      github_oauth_client_id: state.sourceType === 'github' ? state.oauthClientID : undefined,
      github_oauth_client_secret: state.sourceType === 'github' ? state.oauthClientSecret : undefined,
      github_oauth_base_url: state.sourceType === 'github' ? state.oauthBaseURL : undefined,
      azure_ad_tenant_id: state.sourceType === 'azuredevops' ? state.azureADTenantID : undefined,
      azure_ad_client_id: state.sourceType === 'azuredevops' ? state.azureADClientID : undefined,
      azure_ad_client_secret: state.sourceType === 'azuredevops' ? state.azureADClientSecret : undefined,
      callback_url: state.callbackURL,
      frontend_url: state.frontendURL,
      session_secret: state.sessionSecret,
      session_duration_hours: state.sessionDuration,
      authorization_rules: {
        require_org_membership: state.requireOrgMembership ? state.requireOrgMembership.split(',').map(o => o.trim()) : undefined,
        require_team_membership: state.requireTeamMembership ? state.requireTeamMembership.split(',').map(t => t.trim()) : undefined,
        require_enterprise_admin: state.requireEnterpriseAdmin || undefined,
        require_enterprise_membership: state.requireEnterpriseMembership || undefined,
        enterprise_slug: state.enterpriseSlug || undefined,
        privileged_teams: state.privilegedTeams ? state.privilegedTeams.split(',').map(t => t.trim()) : undefined,
      },
    } : { enabled: false },
  }), [state]);

  return {
    state,
    setField,
    setFields,
    reset,
    buildConfig,
  };
}

export { initialState as setupWizardInitialState };

