import { useState } from 'react';
import { Button, FormControl, TextInput, Select, Radio, RadioGroup, Heading, Text, Flash } from '@primer/react';
import { AlertIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import { StepIndicator } from './StepIndicator';
import { ConnectionTest } from './ConnectionTest';
import { CollapsibleSection } from './CollapsibleSection';
import { ConfigSummary } from './ConfigSummary';
import { RestartInstructions } from './RestartInstructions';
import type { SetupConfig } from '../../types';

const STEP_TITLES = [
  'Welcome',
  'Source',
  'Destination',
  'Database',
  'Server',
  'Migration',
  'Advanced',
  'Review',
];

export function SetupWizard() {
  const [currentStep, setCurrentStep] = useState(1);
  const [applyingConfig, setApplyingConfig] = useState(false);
  const [restartingServer, setRestartingServer] = useState(false);

  // Step 1: Source selection
  const [sourceType, setSourceType] = useState<'github' | 'azuredevops'>('github');

  // Step 2: Source configuration
  const [sourceBaseURL, setSourceBaseURL] = useState('https://api.github.com');
  const [sourceToken, setSourceToken] = useState('');
  const [sourceOrganization, setSourceOrganization] = useState('');
  const [sourceValidated, setSourceValidated] = useState(false);

  // Step 3: Destination configuration
  const [destBaseURL, setDestBaseURL] = useState('https://api.github.com');
  const [destToken, setDestToken] = useState('');
  const [destValidated, setDestValidated] = useState(false);

  // Step 4: Database configuration
  const [dbType, setDbType] = useState<'sqlite' | 'postgres' | 'sqlserver'>('sqlite');
  const [dbDSN, setDbDSN] = useState('./data/migrator.db');
  const [dbValidated, setDbValidated] = useState(false);

  // Step 5: Server configuration
  const [serverPort, setServerPort] = useState(8080);
  const [migrationWorkers, setMigrationWorkers] = useState(5);
  const [pollInterval, setPollInterval] = useState(30);

  // Step 6: Migration behavior
  const [destRepoExistsAction, setDestRepoExistsAction] = useState<'fail' | 'skip' | 'delete'>('fail');
  const [publicReposVisibility, setPublicReposVisibility] = useState<'public' | 'internal' | 'private'>('private');
  const [internalReposVisibility, setInternalReposVisibility] = useState<'internal' | 'private'>('private');

  // Step 7: Logging & Advanced Configuration
  const [logLevel, setLogLevel] = useState<'debug' | 'info' | 'warn' | 'error'>('info');
  const [logFormat, setLogFormat] = useState<'json' | 'text'>('json');
  const [logOutputFile, setLogOutputFile] = useState('./logs/migrator.log');

  // GitHub App Discovery (optional, always available)
  const [githubAppEnabled, setGithubAppEnabled] = useState(false);
  const [githubAppID, setGithubAppID] = useState('');
  const [githubAppPrivateKey, setGithubAppPrivateKey] = useState('');
  const [githubAppInstallationID, setGithubAppInstallationID] = useState('');

  // Authentication (optional, conditional on source type)
  const [authEnabled, setAuthEnabled] = useState(false);
  // GitHub OAuth (when source is GitHub)
  const [oauthClientID, setOauthClientID] = useState('');
  const [oauthClientSecret, setOauthClientSecret] = useState('');
  const [oauthBaseURL, setOauthBaseURL] = useState('');
  // Azure AD (when source is Azure DevOps)
  const [azureADTenantID, setAzureADTenantID] = useState('');
  const [azureADClientID, setAzureADClientID] = useState('');
  const [azureADClientSecret, setAzureADClientSecret] = useState('');
  // Common auth settings
  const [callbackURL, setCallbackURL] = useState('');
  const [frontendURL, setFrontendURL] = useState('');
  const [sessionSecret, setSessionSecret] = useState('');
  const [sessionDuration, setSessionDuration] = useState(24);

  const buildConfig = (): SetupConfig => ({
    source: {
      type: sourceType,
      base_url: sourceBaseURL,
      token: sourceToken,
      organization: sourceType === 'azuredevops' ? sourceOrganization : undefined,
    },
    destination: {
      base_url: destBaseURL,
      token: destToken,
      app_id: githubAppEnabled && githubAppID ? parseInt(githubAppID) : undefined,
      app_private_key: githubAppEnabled && githubAppPrivateKey ? githubAppPrivateKey : undefined,
      app_installation_id: githubAppEnabled && githubAppInstallationID ? parseInt(githubAppInstallationID) : undefined,
    },
    database: {
      type: dbType,
      dsn: dbDSN,
    },
    server: {
      port: serverPort,
    },
    migration: {
      workers: migrationWorkers,
      poll_interval_seconds: pollInterval,
      dest_repo_exists_action: destRepoExistsAction,
      visibility_handling: {
        public_repos: publicReposVisibility,
        internal_repos: internalReposVisibility,
      },
    },
    logging: {
      level: logLevel,
      format: logFormat,
      output_file: logOutputFile,
    },
    auth: authEnabled ? {
      enabled: true,
      github_oauth_client_id: sourceType === 'github' ? oauthClientID : undefined,
      github_oauth_client_secret: sourceType === 'github' ? oauthClientSecret : undefined,
      github_oauth_base_url: sourceType === 'github' ? oauthBaseURL : undefined,
      azure_ad_tenant_id: sourceType === 'azuredevops' ? azureADTenantID : undefined,
      azure_ad_client_id: sourceType === 'azuredevops' ? azureADClientID : undefined,
      azure_ad_client_secret: sourceType === 'azuredevops' ? azureADClientSecret : undefined,
      callback_url: callbackURL,
      frontend_url: frontendURL,
      session_secret: sessionSecret,
      session_duration_hours: sessionDuration,
    } : undefined,
  });

  const canProceedFromStep = (step: number): boolean => {
    switch (step) {
      case 1:
        return true; // Welcome screen, can always proceed
      case 2:
        return sourceValidated && sourceToken.length > 0;
      case 3:
        return destValidated && destToken.length > 0;
      case 4:
        return dbValidated && dbDSN.length > 0;
      case 5:
        return serverPort > 0 && migrationWorkers > 0 && pollInterval > 0;
      case 6:
      case 7:
        return true; // Optional steps
      case 8:
        return true; // Review
      default:
        return false;
    }
  };

  const handleNext = () => {
    if (canProceedFromStep(currentStep)) {
      setCurrentStep(currentStep + 1);
    }
  };

  const handleBack = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleApplyConfig = async () => {
    setApplyingConfig(true);
    try {
      await api.applySetup(buildConfig());
      // Configuration saved successfully, show restart instructions
      setRestartingServer(true);
    } catch (error) {
      console.error('Failed to apply configuration:', error);
      alert('Failed to apply configuration. Please check the logs and try again.');
    } finally {
      setApplyingConfig(false);
    }
  };


  // Update source base URL when type changes
  const handleSourceTypeChange = (type: 'github' | 'azuredevops') => {
    setSourceType(type);
    setSourceValidated(false);
    if (type === 'github') {
      setSourceBaseURL('https://api.github.com');
    } else {
      setSourceBaseURL('https://dev.azure.com/your-organization');
    }
  };

  // Update DSN placeholder when database type changes
  const handleDbTypeChange = (type: 'sqlite' | 'postgres' | 'sqlserver') => {
    setDbType(type);
    setDbValidated(false);
    if (type === 'sqlite') {
      setDbDSN('./data/migrator.db');
    } else if (type === 'postgres') {
      setDbDSN('postgres://user:password@localhost:5432/migrator?sslmode=disable');
    } else {
      setDbDSN('sqlserver://user:password@localhost:1433?database=migrator');
    }
  };

  if (restartingServer) {
    return <RestartInstructions />;
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-8">
      <StepIndicator currentStep={currentStep} totalSteps={8} stepTitles={STEP_TITLES} />

      {/* Step 1: Welcome & Source Selection */}
      {currentStep === 1 && (
        <div>
          <Heading as="h2" className="mb-4">
            Welcome to GitHub Migrator Setup
          </Heading>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            This wizard will guide you through configuring your migration server. We'll set up your source and
            destination repositories, database, and migration settings.
          </Text>

          <FormControl required className="mb-6">
            <FormControl.Label>Select your migration source</FormControl.Label>
            <RadioGroup name="source-type" onChange={(value) => handleSourceTypeChange(value as any)}>
              <label className="flex items-start cursor-pointer">
                <Radio value="github" checked={sourceType === 'github'} />
                <div className="ml-3 flex-1">
                  <Text className="font-bold block">GitHub</Text>
                  <Text className="text-xs block" style={{ color: 'var(--fgColor-muted)' }}>
                    Migrate from GitHub.com or GitHub Enterprise Server
                  </Text>
                </div>
              </label>
              <label className="mt-3 flex items-start cursor-pointer">
                <Radio value="azuredevops" checked={sourceType === 'azuredevops'} />
                <div className="ml-3 flex-1">
                  <Text className="font-bold block">Azure DevOps</Text>
                  <Text className="text-xs block" style={{ color: 'var(--fgColor-muted)' }}>
                    Migrate from Azure DevOps Services
                  </Text>
                </div>
              </label>
            </RadioGroup>
          </FormControl>
        </div>
      )}

      {/* Step 2: Source Configuration */}
      {currentStep === 2 && (
        <div>
          <Heading as="h2" className="mb-4">
            Configure Source Repository
          </Heading>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Enter the API endpoint and authentication token for your source system.
          </Text>

          <FormControl required className="mb-4">
            <FormControl.Label>Source Type</FormControl.Label>
            <TextInput value={sourceType.toUpperCase()} disabled block />
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>Base URL</FormControl.Label>
            <TextInput
              value={sourceBaseURL}
              onChange={(e) => {
                setSourceBaseURL(e.target.value);
                setSourceValidated(false);
              }}
              placeholder={
                sourceType === 'github'
                  ? 'https://api.github.com or https://github.company.com/api/v3'
                  : 'https://dev.azure.com/your-organization'
              }
              block
            />
            <FormControl.Caption>
              {sourceType === 'github'
                ? 'For GitHub.com use https://api.github.com, for GitHub Enterprise use your server URL'
                : 'Your Azure DevOps organization URL'}
            </FormControl.Caption>
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>Personal Access Token</FormControl.Label>
            <TextInput
              type="password"
              value={sourceToken}
              onChange={(e) => {
                setSourceToken(e.target.value);
                setSourceValidated(false);
              }}
              placeholder="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
              block
            />
            <FormControl.Caption>
              {sourceType === 'github'
                ? 'Required scopes: repo, admin:org, workflow, read:user'
                : 'Required scopes: Code (Read), Build (Read), Work Items (Read), Project and Team (Read)'}
            </FormControl.Caption>
          </FormControl>

          {sourceType === 'azuredevops' && (
            <FormControl className="mb-4">
              <FormControl.Label>Organization Name</FormControl.Label>
              <TextInput
                value={sourceOrganization}
                onChange={(e) => {
                  setSourceOrganization(e.target.value);
                  setSourceValidated(false);
                }}
                placeholder="your-ado-org"
                block
              />
              <FormControl.Caption>Your Azure DevOps organization name</FormControl.Caption>
            </FormControl>
          )}

          <ConnectionTest
            onTest={async () => {
              const result = await api.validateSourceConnection({
                type: sourceType,
                base_url: sourceBaseURL,
                token: sourceToken,
                organization: sourceOrganization,
              });
              setSourceValidated(result.valid);
              return result;
            }}
            disabled={!sourceBaseURL || !sourceToken}
          />

          {!sourceValidated && (
            <Flash variant="warning" className="flex items-start" style={{ marginTop: '24px', marginBottom: '24px', padding: '16px' }}>
              <AlertIcon size={16} />
              <Text className="ml-3">Please test your connection before proceeding to the next step.</Text>
            </Flash>
          )}
        </div>
      )}

      {/* Step 3: Destination Configuration */}
      {currentStep === 3 && (
        <div>
          <Heading as="h2" className="mb-4">
            Configure Destination Repository
          </Heading>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Enter the GitHub API endpoint and authentication token for your destination.
          </Text>

          <FormControl required className="mb-4">
            <FormControl.Label>Base URL</FormControl.Label>
            <TextInput
              value={destBaseURL}
              onChange={(e) => {
                setDestBaseURL(e.target.value);
                setDestValidated(false);
              }}
              placeholder="https://api.github.com or https://github.company.com/api/v3"
              block
            />
            <FormControl.Caption>
              For GitHub.com use https://api.github.com, for GitHub Enterprise use your server URL
            </FormControl.Caption>
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>Personal Access Token</FormControl.Label>
            <TextInput
              type="password"
              value={destToken}
              onChange={(e) => {
                setDestToken(e.target.value);
                setDestValidated(false);
              }}
              placeholder="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
              block
            />
            <FormControl.Caption>Required scopes: repo, admin:org, workflow</FormControl.Caption>
          </FormControl>

          <ConnectionTest
            onTest={async () => {
              const result = await api.validateDestinationConnection({
                base_url: destBaseURL,
                token: destToken,
              });
              setDestValidated(result.valid);
              return result;
            }}
            disabled={!destBaseURL || !destToken}
          />

          {!destValidated && (
            <Flash variant="warning" className="flex items-start" style={{ marginTop: '24px', marginBottom: '24px', padding: '16px' }}>
              <AlertIcon size={16} />
              <Text className="ml-3">Please test your connection before proceeding to the next step.</Text>
            </Flash>
          )}
        </div>
      )}

      {/* Step 4: Database Configuration */}
      {currentStep === 4 && (
        <div>
          <Heading as="h2" className="mb-4">
            Configure Database
          </Heading>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Select your database backend and provide the connection details.
          </Text>

          <FormControl required className="mb-4">
            <FormControl.Label>Database Type</FormControl.Label>
            <Select value={dbType} onChange={(e) => handleDbTypeChange(e.target.value as any)} block>
              <Select.Option value="sqlite">SQLite</Select.Option>
              <Select.Option value="postgres">PostgreSQL</Select.Option>
              <Select.Option value="sqlserver">SQL Server</Select.Option>
            </Select>
            <FormControl.Caption>
              {dbType === 'sqlite' && 'Simple file-based database, good for development and small deployments'}
              {dbType === 'postgres' && 'Production-ready, scalable database (recommended for production)'}
              {dbType === 'sqlserver' && 'Enterprise database option for Azure environments'}
            </FormControl.Caption>
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>Connection String (DSN)</FormControl.Label>
            <TextInput
              value={dbDSN}
              onChange={(e) => {
                setDbDSN(e.target.value);
                setDbValidated(false);
              }}
              placeholder={
                dbType === 'sqlite'
                  ? './data/migrator.db'
                  : dbType === 'postgres'
                  ? 'postgres://user:password@localhost:5432/migrator?sslmode=disable'
                  : 'sqlserver://user:password@localhost:1433?database=migrator'
              }
              block
            />
            <FormControl.Caption>
              {dbType === 'sqlite' && 'File path for SQLite database'}
              {dbType === 'postgres' && 'PostgreSQL connection string'}
              {dbType === 'sqlserver' && 'SQL Server connection string'}
            </FormControl.Caption>
          </FormControl>

          <ConnectionTest
            onTest={async () => {
              const result = await api.validateDatabaseConnection({
                type: dbType,
                dsn: dbDSN,
              });
              setDbValidated(result.valid);
              return result;
            }}
            disabled={!dbDSN}
            label="Test Database Connection"
          />

          {!dbValidated && (
            <Flash variant="warning" className="flex items-start" style={{ marginTop: '24px', marginBottom: '24px', padding: '16px' }}>
              <AlertIcon size={16} />
              <Text className="ml-3">Please test your database connection before proceeding.</Text>
            </Flash>
          )}
        </div>
      )}

      {/* Step 5: Server Settings */}
      {currentStep === 5 && (
        <div>
          <Heading as="h2" className="mb-4">
            Server Settings
          </Heading>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Configure server port and migration worker settings.
          </Text>

          <FormControl required className="mb-4">
            <FormControl.Label>Server Port</FormControl.Label>
            <TextInput
              type="number"
              value={serverPort.toString()}
              onChange={(e) => setServerPort(parseInt(e.target.value) || 8080)}
              block
            />
            <FormControl.Caption>Default: 8080</FormControl.Caption>
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>Migration Workers</FormControl.Label>
            <TextInput
              type="number"
              value={migrationWorkers.toString()}
              onChange={(e) => setMigrationWorkers(parseInt(e.target.value) || 5)}
              min="1"
              max="20"
              block
            />
            <FormControl.Caption>
              Number of concurrent migrations (1-20). Higher = faster but more API usage. Default: 5
            </FormControl.Caption>
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>Poll Interval (seconds)</FormControl.Label>
            <TextInput
              type="number"
              value={pollInterval.toString()}
              onChange={(e) => setPollInterval(parseInt(e.target.value) || 30)}
              block
            />
            <FormControl.Caption>How often workers check for new work. Default: 30 seconds</FormControl.Caption>
          </FormControl>
        </div>
      )}

      {/* Step 6: Migration Behavior */}
      {currentStep === 6 && (
        <div>
          <Heading as="h2" className="mb-4">
            Migration Behavior (Optional)
          </Heading>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Configure how the migration handles edge cases and repository visibility.
          </Text>

          <CollapsibleSection
            title="Destination Repository Handling"
            description="What to do if a destination repository already exists"
            isOptional
            defaultExpanded
          >
            <RadioGroup name="dest-repo-action" onChange={(value) => setDestRepoExistsAction(value as any)}>
              <label className="flex items-start cursor-pointer">
                <Radio value="fail" checked={destRepoExistsAction === 'fail'} />
                <div className="ml-3 flex-1">
                  <Text className="font-bold block">Fail (Recommended)</Text>
                  <Text className="text-xs block" style={{ color: 'var(--fgColor-muted)' }}>
                    Stop migration and report error
                  </Text>
                </div>
              </label>
              <label className="mt-3 flex items-start cursor-pointer">
                <Radio value="skip" checked={destRepoExistsAction === 'skip'} />
                <div className="ml-3 flex-1">
                  <Text className="font-bold block">Skip</Text>
                  <Text className="text-xs block" style={{ color: 'var(--fgColor-muted)' }}>
                    Skip this repository and continue with others
                  </Text>
                </div>
              </label>
              <label className="mt-3 flex items-start cursor-pointer">
                <Radio value="delete" checked={destRepoExistsAction === 'delete'} />
                <div className="ml-3 flex-1">
                  <Text className="font-bold block" style={{ color: 'var(--fgColor-danger)' }}>
                    Delete (Dangerous!)
                  </Text>
                  <Text className="text-xs block" style={{ color: 'var(--fgColor-muted)' }}>
                    Delete existing repo and recreate
                  </Text>
                </div>
              </label>
            </RadioGroup>
          </CollapsibleSection>

          <CollapsibleSection
            title="Visibility Transformation"
            description="How to transform repository visibility during migration"
            isOptional
            defaultExpanded
          >
            <FormControl className="mb-4">
              <FormControl.Label>Public Repositories</FormControl.Label>
              <Select value={publicReposVisibility} onChange={(e) => setPublicReposVisibility(e.target.value as any)} block>
                <Select.Option value="public">Keep Public</Select.Option>
                <Select.Option value="internal">Convert to Internal</Select.Option>
                <Select.Option value="private">Convert to Private (Default)</Select.Option>
              </Select>
              <FormControl.Caption>
                Note: EMU and data residency environments don't support public repositories
              </FormControl.Caption>
            </FormControl>

            <FormControl>
              <FormControl.Label>Internal Repositories</FormControl.Label>
              <Select value={internalReposVisibility} onChange={(e) => setInternalReposVisibility(e.target.value as any)} block>
                <Select.Option value="internal">Keep Internal</Select.Option>
                <Select.Option value="private">Convert to Private (Default)</Select.Option>
              </Select>
              <FormControl.Caption>Private repositories always migrate as private</FormControl.Caption>
            </FormControl>
          </CollapsibleSection>
        </div>
      )}

      {/* Step 7: Advanced Configuration */}
      {currentStep === 7 && (
        <div>
          <Heading as="h2" className="mb-4">
            Advanced Configuration (Optional)
          </Heading>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Configure logging, authentication, and GitHub App discovery. All settings are optional.
          </Text>

          {/* Logging Section */}
          <CollapsibleSection
            title="Logging"
            description="Configure log level, format, and output location"
            isOptional
            defaultExpanded={false}
          >
            <FormControl className="mb-4">
              <FormControl.Label>Log Level</FormControl.Label>
              <Select value={logLevel} onChange={(e) => setLogLevel(e.target.value as any)} block>
                <Select.Option value="debug">Debug (Verbose)</Select.Option>
                <Select.Option value="info">Info (Recommended)</Select.Option>
                <Select.Option value="warn">Warn (Warnings & Errors only)</Select.Option>
                <Select.Option value="error">Error (Errors only)</Select.Option>
              </Select>
              <FormControl.Caption>Default: info</FormControl.Caption>
            </FormControl>

            <FormControl className="mb-4">
              <FormControl.Label>Log Format</FormControl.Label>
              <RadioGroup name="log-format" onChange={(value) => setLogFormat(value as any)}>
                <label className="flex items-start cursor-pointer">
                  <Radio value="json" checked={logFormat === 'json'} />
                  <div className="ml-3 flex-1">
                    <Text className="font-bold block">JSON (Default)</Text>
                    <Text className="text-xs block" style={{ color: 'var(--fgColor-muted)' }}>
                      Structured logs, good for log aggregation
                    </Text>
                  </div>
                </label>
                <label className="mt-3 flex items-start cursor-pointer">
                  <Radio value="text" checked={logFormat === 'text'} />
                  <div className="ml-3 flex-1">
                    <Text className="font-bold block">Text</Text>
                    <Text className="text-xs block" style={{ color: 'var(--fgColor-muted)' }}>
                      Human-readable, good for development
                    </Text>
                  </div>
                </label>
              </RadioGroup>
            </FormControl>

            <FormControl>
              <FormControl.Label>Output File Path</FormControl.Label>
              <TextInput value={logOutputFile} onChange={(e) => setLogOutputFile(e.target.value)} block />
              <FormControl.Caption>Default: ./logs/migrator.log</FormControl.Caption>
            </FormControl>
          </CollapsibleSection>

          {/* GitHub App Discovery Section */}
          <CollapsibleSection
            title="GitHub App Discovery"
            description="Use a GitHub App for enhanced repository discovery (destination only)"
            isOptional
            defaultExpanded={false}
          >
            <FormControl className="mb-4">
              <FormControl.Label>
                <input
                  type="checkbox"
                  checked={githubAppEnabled}
                  onChange={(e) => setGithubAppEnabled(e.target.checked)}
                  className="mr-2"
                />
                Enable GitHub App Authentication
              </FormControl.Label>
              <FormControl.Caption>
                Use a GitHub App instead of PAT for discovery operations. This is separate from user authentication.
              </FormControl.Caption>
            </FormControl>

            {githubAppEnabled && (
              <>
                <FormControl className="mb-4">
                  <FormControl.Label>GitHub App ID</FormControl.Label>
                  <TextInput
                    type="number"
                    value={githubAppID}
                    onChange={(e) => setGithubAppID(e.target.value)}
                    placeholder="123456"
                    block
                  />
                  <FormControl.Caption>Your GitHub App's ID from the app settings</FormControl.Caption>
                </FormControl>

                <FormControl className="mb-4">
                  <FormControl.Label>Installation ID</FormControl.Label>
                  <TextInput
                    type="number"
                    value={githubAppInstallationID}
                    onChange={(e) => setGithubAppInstallationID(e.target.value)}
                    placeholder="12345678"
                    block
                  />
                  <FormControl.Caption>Installation ID for your organization</FormControl.Caption>
                </FormControl>

                <FormControl>
                  <FormControl.Label>Private Key (PEM)</FormControl.Label>
                  <textarea
                    value={githubAppPrivateKey}
                    onChange={(e) => setGithubAppPrivateKey(e.target.value)}
                    placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
                    rows={6}
                    className="form-control"
                    style={{
                      width: '100%',
                      fontFamily: 'monospace',
                      fontSize: '12px',
                      backgroundColor: 'var(--bgColor-inset)',
                      color: 'var(--fgColor-default)',
                      border: '1px solid var(--borderColor-default)',
                      borderRadius: '6px',
                      padding: '8px 12px',
                    }}
                  />
                  <FormControl.Caption>Paste your GitHub App's private key (file path or full PEM content)</FormControl.Caption>
                </FormControl>
              </>
            )}
          </CollapsibleSection>

          {/* Authentication Section */}
          <CollapsibleSection
            title="User Authentication"
            description={sourceType === 'github' ? 'Enable GitHub OAuth for user login' : 'Enable Azure AD for user login'}
            isOptional
            defaultExpanded={false}
          >
            <FormControl className="mb-4">
              <FormControl.Label>
                <input
                  type="checkbox"
                  checked={authEnabled}
                  onChange={(e) => setAuthEnabled(e.target.checked)}
                  className="mr-2"
                />
                Enable User Authentication
              </FormControl.Label>
              <FormControl.Caption>
                {sourceType === 'github' 
                  ? 'Require users to authenticate with GitHub OAuth before accessing the migration tool'
                  : 'Require users to authenticate with Azure AD before accessing the migration tool'}
              </FormControl.Caption>
            </FormControl>

            {authEnabled && (
              <>
                {sourceType === 'github' ? (
                  <>
                    <FormControl className="mb-4">
                      <FormControl.Label>GitHub OAuth Client ID</FormControl.Label>
                      <TextInput
                        value={oauthClientID}
                        onChange={(e) => setOauthClientID(e.target.value)}
                        placeholder="Iv1.a1b2c3d4e5f6g7h8"
                        block
                      />
                      <FormControl.Caption>OAuth App Client ID from GitHub settings</FormControl.Caption>
                    </FormControl>

                    <FormControl className="mb-4">
                      <FormControl.Label>GitHub OAuth Client Secret</FormControl.Label>
                      <TextInput
                        type="password"
                        value={oauthClientSecret}
                        onChange={(e) => setOauthClientSecret(e.target.value)}
                        placeholder="gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                        block
                      />
                      <FormControl.Caption>OAuth App Client Secret</FormControl.Caption>
                    </FormControl>

                    <FormControl className="mb-4">
                      <FormControl.Label>OAuth Base URL (Optional)</FormControl.Label>
                      <TextInput
                        value={oauthBaseURL}
                        onChange={(e) => setOauthBaseURL(e.target.value)}
                        placeholder="Defaults to source URL if blank"
                        block
                      />
                      <FormControl.Caption>Leave blank to use source URL</FormControl.Caption>
                    </FormControl>
                  </>
                ) : (
                  <>
                    <FormControl className="mb-4">
                      <FormControl.Label>Azure AD Tenant ID</FormControl.Label>
                      <TextInput
                        value={azureADTenantID}
                        onChange={(e) => setAzureADTenantID(e.target.value)}
                        placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                        block
                      />
                      <FormControl.Caption>Your Azure AD Tenant ID (GUID)</FormControl.Caption>
                    </FormControl>

                    <FormControl className="mb-4">
                      <FormControl.Label>Azure AD Application (Client) ID</FormControl.Label>
                      <TextInput
                        value={azureADClientID}
                        onChange={(e) => setAzureADClientID(e.target.value)}
                        placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                        block
                      />
                      <FormControl.Caption>Application (client) ID from Azure AD app registration</FormControl.Caption>
                    </FormControl>

                    <FormControl className="mb-4">
                      <FormControl.Label>Azure AD Client Secret</FormControl.Label>
                      <TextInput
                        type="password"
                        value={azureADClientSecret}
                        onChange={(e) => setAzureADClientSecret(e.target.value)}
                        placeholder="Client secret value"
                        block
                      />
                      <FormControl.Caption>Client secret from Azure AD app registration</FormControl.Caption>
                    </FormControl>
                  </>
                )}

                <FormControl className="mb-4">
                  <FormControl.Label>Callback URL</FormControl.Label>
                  <TextInput
                    value={callbackURL}
                    onChange={(e) => setCallbackURL(e.target.value)}
                    placeholder="http://localhost:8080/auth/callback"
                    block
                  />
                  <FormControl.Caption>OAuth callback URL (must match registration)</FormControl.Caption>
                </FormControl>

                <FormControl className="mb-4">
                  <FormControl.Label>Frontend URL</FormControl.Label>
                  <TextInput
                    value={frontendURL}
                    onChange={(e) => setFrontendURL(e.target.value)}
                    placeholder="http://localhost:3000"
                    block
                  />
                  <FormControl.Caption>Frontend URL for redirects after authentication</FormControl.Caption>
                </FormControl>

                <FormControl className="mb-4">
                  <FormControl.Label>Session Secret</FormControl.Label>
                  <TextInput
                    type="password"
                    value={sessionSecret}
                    onChange={(e) => setSessionSecret(e.target.value)}
                    placeholder="Generate a random secret (min 32 characters)"
                    block
                  />
                  <FormControl.Caption>Random secret for session encryption (generate a long random string)</FormControl.Caption>
                </FormControl>

                <FormControl>
                  <FormControl.Label>Session Duration (hours)</FormControl.Label>
                  <TextInput
                    type="number"
                    value={sessionDuration.toString()}
                    onChange={(e) => setSessionDuration(parseInt(e.target.value) || 24)}
                    block
                  />
                  <FormControl.Caption>Default: 24 hours</FormControl.Caption>
                </FormControl>
              </>
            )}
          </CollapsibleSection>
        </div>
      )}

      {/* Step 8: Review & Apply */}
      {currentStep === 8 && (
        <div>
          <Heading as="h2" className="mb-4">
            Review Configuration
          </Heading>
          <Text className="mb-8" style={{ color: 'var(--fgColor-muted)' }}>
            Please review all settings below before applying the configuration.
          </Text>

          <div className="mb-6">
            <Flash variant="warning" className="flex items-start" style={{ padding: '16px' }}>
              <AlertIcon size={16} />
              <Text className="ml-3 font-bold">
                The server will restart after applying configuration. This may take 10-30 seconds.
              </Text>
            </Flash>
          </div>

          <ConfigSummary config={buildConfig()} />

          <div
            className="mt-6 p-4 rounded-lg border"
            style={{
              backgroundColor: 'var(--bgColor-muted)',
              borderColor: 'var(--borderColor-default)',
            }}
          >
            <Heading as="h4" className="text-sm mb-3">
              Next Steps
            </Heading>
            <Text className="text-xs">
              1. Click "Apply Configuration" below
              <br />
              2. Wait for the server to restart (10-30 seconds)
              <br />
              3. You'll be redirected to the dashboard automatically
            </Text>
          </div>
        </div>
      )}

      {/* Navigation Buttons */}
      <div
        className="flex justify-between mt-8 pt-6 border-t"
        style={{ borderColor: 'var(--borderColor-default)' }}
      >
        <Button onClick={handleBack} disabled={currentStep === 1 || applyingConfig}>
          Back
        </Button>

        {currentStep < 8 && (
          <Button variant="primary" onClick={handleNext} disabled={!canProceedFromStep(currentStep)}>
            Next
          </Button>
        )}

        {currentStep === 8 && (
          <Button variant="primary" onClick={handleApplyConfig} disabled={applyingConfig}>
            {applyingConfig ? 'Applying Configuration...' : 'Apply Configuration'}
          </Button>
        )}
      </div>
    </div>
  );
}
