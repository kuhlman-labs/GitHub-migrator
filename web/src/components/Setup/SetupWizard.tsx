import { useState } from 'react';
import { Button, FormControl, TextInput, Select, Radio, RadioGroup, Heading, Text, Flash } from '@primer/react';
import { AlertIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import { StepIndicator } from './StepIndicator';
import { ConnectionTest } from './ConnectionTest';
import { CollapsibleSection } from './CollapsibleSection';
import { ConfigSummary } from './ConfigSummary';
import { RestartMonitor } from './RestartMonitor';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';
import { useSetupWizardForm } from '../../hooks/useSetupWizardForm';

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
  const { showError } = useToast();
  const [currentStep, setCurrentStep] = useState(1);
  const [applyingConfig, setApplyingConfig] = useState(false);
  const [restartingServer, setRestartingServer] = useState(false);

  // All form state is now managed by the useSetupWizardForm hook
  const { state: formState, setField, buildConfig } = useSetupWizardForm();

  // Destructure commonly accessed state for convenience
  const {
    sourceType, sourceBaseURL, sourceToken, sourceOrganization, sourceValidated,
    destBaseURL, destToken, destValidated,
    dbType, dbDSN, dbValidated,
    serverPort, migrationWorkers, pollInterval,
    destRepoExistsAction, publicReposVisibility, internalReposVisibility,
    logLevel, logFormat, logOutputFile,
    sourceGithubAppEnabled, sourceGithubAppID, sourceGithubAppPrivateKey, sourceGithubAppInstallationID,
    destGithubAppEnabled, destGithubAppID, destGithubAppPrivateKey, destGithubAppInstallationID,
    authEnabled, oauthClientID, oauthClientSecret, oauthBaseURL,
    azureADTenantID, azureADClientID, azureADClientSecret,
    callbackURL, frontendURL, sessionSecret, sessionDuration,
    requireOrgMembership, requireTeamMembership, requireEnterpriseAdmin,
    requireEnterpriseMembership, enterpriseSlug, privilegedTeams,
  } = formState;

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
      handleApiError(error, showError, 'Failed to apply configuration');
    } finally {
      setApplyingConfig(false);
    }
  };


  // Update source base URL when type changes
  const handleSourceTypeChange = (type: 'github' | 'azuredevops') => {
    setField('sourceType', type);
    setField('sourceValidated', false);
    if (type === 'github') {
      setField('sourceBaseURL', 'https://api.github.com');
    } else {
      setField('sourceBaseURL', 'https://dev.azure.com/your-organization');
    }
  };

  // Update DSN placeholder when database type changes
  const handleDbTypeChange = (type: 'sqlite' | 'postgres' | 'sqlserver') => {
    setField('dbType', type);
    setField('dbValidated', false);
    if (type === 'sqlite') {
      setField('dbDSN', './data/migrator.db');
    } else if (type === 'postgres') {
      setField('dbDSN', 'postgres://user:password@localhost:5432/migrator?sslmode=disable');
    } else {
      setField('dbDSN', 'sqlserver://user:password@localhost:1433?database=migrator');
    }
  };

  if (restartingServer) {
    return <RestartMonitor onServerOnline={() => {
      // Redirect to dashboard when server is back online
      window.location.href = '/';
    }} />;
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
            <RadioGroup name="source-type" onChange={(value) => handleSourceTypeChange(value as 'github' | 'azuredevops')}>
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
                setField('sourceBaseURL', e.target.value);
                setField('sourceValidated', false);
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
                setField('sourceToken', e.target.value);
                setField('sourceValidated', false);
              }}
              placeholder="ghp_xxx"
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
                  setField('sourceOrganization', e.target.value);
                  setField('sourceValidated', false);
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
              setField('sourceValidated', result.valid);
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
                setField('destBaseURL', e.target.value);
                setField('destValidated', false);
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
                setField('destToken', e.target.value);
                setField('destValidated', false);
              }}
              placeholder="ghp_xxx"
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
              setField('destValidated', result.valid);
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
            <Select value={dbType} onChange={(e) => handleDbTypeChange(e.target.value as 'sqlite' | 'postgres' | 'sqlserver')} block>
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
                setField('dbDSN', e.target.value);
                setField('dbValidated', false);
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
              setField('dbValidated', result.valid);
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
              onChange={(e) => setField('serverPort', parseInt(e.target.value) || 8080)}
              block
            />
            <FormControl.Caption>Default: 8080</FormControl.Caption>
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>Migration Workers</FormControl.Label>
            <TextInput
              type="number"
              value={migrationWorkers.toString()}
              onChange={(e) => setField('migrationWorkers', parseInt(e.target.value) || 5)}
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
              onChange={(e) => setField('pollInterval', parseInt(e.target.value) || 30)}
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
            <RadioGroup name="dest-repo-action" onChange={(value) => setField('destRepoExistsAction', value as 'fail' | 'skip' | 'delete')}>
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
              <Select value={publicReposVisibility} onChange={(e) => setField('publicReposVisibility', e.target.value as 'public' | 'internal' | 'private')} block>
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
              <Select value={internalReposVisibility} onChange={(e) => setField('internalReposVisibility', e.target.value as 'internal' | 'private')} block>
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
              <Select value={logLevel} onChange={(e) => setField('logLevel', e.target.value as 'debug' | 'info' | 'warn' | 'error')} block>
                <Select.Option value="debug">Debug (Verbose)</Select.Option>
                <Select.Option value="info">Info (Recommended)</Select.Option>
                <Select.Option value="warn">Warn (Warnings & Errors only)</Select.Option>
                <Select.Option value="error">Error (Errors only)</Select.Option>
              </Select>
              <FormControl.Caption>Default: info</FormControl.Caption>
            </FormControl>

            <FormControl className="mb-4">
              <FormControl.Label>Log Format</FormControl.Label>
              <RadioGroup name="log-format" onChange={(value) => setField('logFormat', value as 'json' | 'text')}>
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
              <TextInput value={logOutputFile} onChange={(e) => setField('logOutputFile', e.target.value)} block />
              <FormControl.Caption>Default: ./logs/migrator.log</FormControl.Caption>
            </FormControl>
          </CollapsibleSection>

          {/* GitHub App Discovery for Source (only when source is GitHub) */}
          {sourceType === 'github' && (
            <CollapsibleSection
              title="GitHub App Discovery (Source)"
              description="Use a GitHub App for enhanced repository discovery from source GitHub"
              isOptional
              defaultExpanded={false}
            >
              <FormControl className="mb-4">
                <FormControl.Label>
                  <input
                    type="checkbox"
                    checked={sourceGithubAppEnabled}
                    onChange={(e) => setField('sourceGithubAppEnabled', e.target.checked)}
                    className="mr-2"
                  />
                  Enable GitHub App for Source Discovery
                </FormControl.Label>
                <FormControl.Caption>
                  Use a GitHub App instead of PAT for source discovery operations. This is separate from user authentication.
                </FormControl.Caption>
              </FormControl>

              {sourceGithubAppEnabled && (
                <>
                  <FormControl className="mb-4">
                    <FormControl.Label>GitHub App ID</FormControl.Label>
                    <TextInput
                      type="number"
                      value={sourceGithubAppID}
                      onChange={(e) => setField('sourceGithubAppID', e.target.value)}
                      placeholder="123456"
                      block
                    />
                    <FormControl.Caption>Your GitHub App's ID from the app settings</FormControl.Caption>
                  </FormControl>

                  <FormControl className="mb-4">
                    <FormControl.Label>Installation ID</FormControl.Label>
                    <TextInput
                      type="number"
                      value={sourceGithubAppInstallationID}
                      onChange={(e) => setField('sourceGithubAppInstallationID', e.target.value)}
                      placeholder="12345678"
                      block
                    />
                    <FormControl.Caption>The installation ID for this app in the source organization. Leave blank for Enterprise GitHub Apps.</FormControl.Caption>
                  </FormControl>

                  <FormControl>
                    <FormControl.Label>Private Key (PEM)</FormControl.Label>
                    <textarea
                      value={sourceGithubAppPrivateKey}
                      onChange={(e) => setField('sourceGithubAppPrivateKey', e.target.value)}
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
          )}

          {/* GitHub App Discovery for Destination (always available) */}
          <CollapsibleSection
            title="GitHub App Discovery (Destination)"
            description="Use a GitHub App for enhanced repository discovery on destination GitHub"
            isOptional
            defaultExpanded={false}
          >
            <FormControl className="mb-4">
              <FormControl.Label>
                <input
                  type="checkbox"
                  checked={destGithubAppEnabled}
                  onChange={(e) => setField('destGithubAppEnabled', e.target.checked)}
                  className="mr-2"
                />
                Enable GitHub App for Destination Discovery
              </FormControl.Label>
              <FormControl.Caption>
                Use a GitHub App instead of PAT for destination discovery operations. This is separate from user authentication.
              </FormControl.Caption>
            </FormControl>

            {destGithubAppEnabled && (
              <>
                <FormControl className="mb-4">
                  <FormControl.Label>GitHub App ID</FormControl.Label>
                  <TextInput
                    type="number"
                    value={destGithubAppID}
                    onChange={(e) => setField('destGithubAppID', e.target.value)}
                    placeholder="123456"
                    block
                  />
                  <FormControl.Caption>Your GitHub App's ID from the app settings</FormControl.Caption>
                </FormControl>

                <FormControl className="mb-4">
                  <FormControl.Label>Installation ID</FormControl.Label>
                  <TextInput
                    type="number"
                    value={destGithubAppInstallationID}
                    onChange={(e) => setField('destGithubAppInstallationID', e.target.value)}
                    placeholder="12345678"
                    block
                  />
                  <FormControl.Caption>The installation ID for this app in the destination organization. Leave blank for Enterprise GitHub Apps.</FormControl.Caption>
                </FormControl>

                <FormControl>
                  <FormControl.Label>Private Key (PEM)</FormControl.Label>
                  <textarea
                    value={destGithubAppPrivateKey}
                    onChange={(e) => setField('destGithubAppPrivateKey', e.target.value)}
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
                  onChange={(e) => setField('authEnabled', e.target.checked)}
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
                        onChange={(e) => setField('oauthClientID', e.target.value)}
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
                        onChange={(e) => setField('oauthClientSecret', e.target.value)}
                        placeholder=""
                        block
                      />
                      <FormControl.Caption>OAuth App Client Secret</FormControl.Caption>
                    </FormControl>

                    <FormControl className="mb-4">
                      <FormControl.Label>OAuth Base URL (Optional)</FormControl.Label>
                      <TextInput
                        value={oauthBaseURL}
                        onChange={(e) => setField('oauthBaseURL', e.target.value)}
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
                        onChange={(e) => setField('azureADTenantID', e.target.value)}
                        placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                        block
                      />
                      <FormControl.Caption>Your Azure AD Tenant ID (GUID)</FormControl.Caption>
                    </FormControl>

                    <FormControl className="mb-4">
                      <FormControl.Label>Azure AD Application (Client) ID</FormControl.Label>
                      <TextInput
                        value={azureADClientID}
                        onChange={(e) => setField('azureADClientID', e.target.value)}
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
                        onChange={(e) => setField('azureADClientSecret', e.target.value)}
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
                    onChange={(e) => setField('callbackURL', e.target.value)}
                    placeholder="http://localhost:8080/api/v1/auth/callback"
                    block
                  />
                  <FormControl.Caption>OAuth callback URL (must match registration)</FormControl.Caption>
                </FormControl>

                <FormControl className="mb-4">
                  <FormControl.Label>Frontend URL</FormControl.Label>
                  <TextInput
                    value={frontendURL}
                    onChange={(e) => setField('frontendURL', e.target.value)}
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
                    onChange={(e) => setField('sessionSecret', e.target.value)}
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
                    onChange={(e) => setField('sessionDuration', parseInt(e.target.value) || 24)}
                    block
                  />
                  <FormControl.Caption>Default: 24 hours</FormControl.Caption>
                </FormControl>

                {/* Authorization Rules */}
                <div className="mt-6 pt-6" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
                  <Heading as="h4" className="text-base mb-3">
                    Authorization Rules (Optional)
                  </Heading>
                  <Text className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
                    Restrict access to specific organizations, teams, or enterprise members
                  </Text>

                  <FormControl className="mb-4">
                    <FormControl.Label>Required Organization Membership</FormControl.Label>
                    <TextInput
                      value={requireOrgMembership}
                      onChange={(e) => setField('requireOrgMembership', e.target.value)}
                      placeholder="org1,org2"
                      block
                    />
                    <FormControl.Caption>Comma-separated list of organization names (e.g., "acme-corp,platform-team")</FormControl.Caption>
                  </FormControl>

                  <FormControl className="mb-4">
                    <FormControl.Label>Required Team Membership</FormControl.Label>
                    <TextInput
                      value={requireTeamMembership}
                      onChange={(e) => setField('requireTeamMembership', e.target.value)}
                      placeholder="org1/team-slug,org2/migration-team"
                      block
                    />
                    <FormControl.Caption>Comma-separated list in "org/team-slug" format (e.g., "acme-corp/admins,platform/migration")</FormControl.Caption>
                  </FormControl>

                  <FormControl className="mb-4">
                    <FormControl.Label>
                      <input
                        type="checkbox"
                        checked={requireEnterpriseAdmin}
                        onChange={(e) => setField('requireEnterpriseAdmin', e.target.checked)}
                        className="mr-2"
                      />
                      Require GitHub Enterprise Admin Role
                    </FormControl.Label>
                    <FormControl.Caption>Only allow users with GitHub Enterprise admin privileges</FormControl.Caption>
                  </FormControl>

                  <FormControl className="mb-4">
                    <FormControl.Label>
                      <input
                        type="checkbox"
                        checked={requireEnterpriseMembership}
                        onChange={(e) => setField('requireEnterpriseMembership', e.target.checked)}
                        className="mr-2"
                      />
                      Require Enterprise Membership
                    </FormControl.Label>
                    <FormControl.Caption>Only allow users who are members of the specified enterprise</FormControl.Caption>
                  </FormControl>

                  {(requireEnterpriseAdmin || requireEnterpriseMembership) && (
                    <FormControl className="mb-4">
                      <FormControl.Label>Enterprise Slug</FormControl.Label>
                      <TextInput
                        value={enterpriseSlug}
                        onChange={(e) => setField('enterpriseSlug', e.target.value)}
                        placeholder="your-enterprise"
                        block
                      />
                      <FormControl.Caption>The slug of your GitHub Enterprise (required when using enterprise rules)</FormControl.Caption>
                    </FormControl>
                  )}

                  <FormControl>
                    <FormControl.Label>Privileged Teams (Full Access)</FormControl.Label>
                    <TextInput
                      value={privilegedTeams}
                      onChange={(e) => setField('privilegedTeams', e.target.value)}
                      placeholder="platform-eng/migration-admins"
                      block
                    />
                    <FormControl.Caption>Comma-separated list in "org/team-slug" format for teams with full migration access</FormControl.Caption>
                  </FormControl>
                </div>
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
