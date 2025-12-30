import { useState } from 'react';
import { Button, FormControl, TextInput, Select, Heading, Text, Flash } from '@primer/react';
import { AlertIcon, DatabaseIcon, ServerIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import { StepIndicator } from './StepIndicator';
import { ConnectionTest } from './ConnectionTest';
import { RestartMonitor } from './RestartMonitor';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

const STEP_TITLES = [
  'Database',
  'Server',
];

interface MinimalSetupConfig {
  database: {
    type: 'sqlite' | 'postgres' | 'sqlserver';
    dsn: string;
  };
  server: {
    port: number;
  };
}

export function SetupWizard() {
  const { showError } = useToast();
  const [currentStep, setCurrentStep] = useState(1);
  const [applyingConfig, setApplyingConfig] = useState(false);
  const [restartingServer, setRestartingServer] = useState(false);

  // Minimal form state - only database and server port
  const [dbType, setDbType] = useState<'sqlite' | 'postgres' | 'sqlserver'>('sqlite');
  const [dbDSN, setDbDSN] = useState('./data/migrator.db');
  const [dbValidated, setDbValidated] = useState(false);
  const [serverPort, setServerPort] = useState(8080);

  const canProceedFromStep = (step: number): boolean => {
    switch (step) {
      case 1:
        return dbValidated && dbDSN.length > 0;
      case 2:
        return serverPort > 0 && serverPort < 65536;
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

  const buildMinimalConfig = (): MinimalSetupConfig => ({
    database: {
      type: dbType,
      dsn: dbDSN,
    },
    server: {
      port: serverPort,
    },
  });

  const handleApplyConfig = async () => {
    setApplyingConfig(true);
    try {
      // For minimal setup, we only need to save database and server config
      // The backend will create the .env file with just these essential settings
      await api.applySetup({
        source: {
          type: 'github',
          base_url: 'https://api.github.com',
          token: 'placeholder', // Required by API but not used - sources added via Sources page
        },
        destination: {
          base_url: 'https://api.github.com',
          token: 'placeholder', // Required by API but not used - configured via Settings page
        },
        ...buildMinimalConfig(),
        migration: {
          workers: 5,
          poll_interval_seconds: 30,
          dest_repo_exists_action: 'fail',
          visibility_handling: {
            public_repos: 'private',
            internal_repos: 'private',
          },
        },
        logging: {
          level: 'info',
          format: 'json',
          output_file: './logs/migrator.log',
        },
        auth: { enabled: false },
      });
      // Configuration saved successfully, show restart instructions
      setRestartingServer(true);
    } catch (error) {
      handleApiError(error, showError, 'Failed to apply configuration');
    } finally {
      setApplyingConfig(false);
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
    return <RestartMonitor onServerOnline={() => {
      // Redirect to dashboard when server is back online
      window.location.href = '/';
    }} />;
  }

  return (
    <div className="max-w-2xl mx-auto px-6 py-8">
      <StepIndicator currentStep={currentStep} totalSteps={2} stepTitles={STEP_TITLES} />

      {/* Step 1: Database Configuration */}
      {currentStep === 1 && (
        <div>
          <div className="flex items-center gap-3 mb-4">
            <DatabaseIcon size={24} />
            <Heading as="h2">Database Configuration</Heading>
          </div>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Configure the database where GitHub Migrator will store its data.
            After initial setup, you can configure sources and destination from the dashboard.
          </Text>

          <Flash className="mb-6">
            <Text>
              <strong>Quick Start:</strong> SQLite is perfect for getting started quickly.
              You can migrate to PostgreSQL or SQL Server later if needed.
            </Text>
          </Flash>

          <FormControl required className="mb-4">
            <FormControl.Label>Database Type</FormControl.Label>
            <Select
              value={dbType}
              onChange={(e) => handleDbTypeChange(e.target.value as 'sqlite' | 'postgres' | 'sqlserver')}
            >
              <Select.Option value="sqlite">SQLite (Recommended for Quick Start)</Select.Option>
              <Select.Option value="postgres">PostgreSQL</Select.Option>
              <Select.Option value="sqlserver">SQL Server</Select.Option>
            </Select>
            <FormControl.Caption>
              {dbType === 'sqlite' && 'SQLite stores data in a local file. No additional setup required.'}
              {dbType === 'postgres' && 'PostgreSQL is recommended for production deployments.'}
              {dbType === 'sqlserver' && 'SQL Server is ideal for Microsoft-centric environments.'}
            </FormControl.Caption>
          </FormControl>

          <FormControl required className="mb-4">
            <FormControl.Label>
              {dbType === 'sqlite' ? 'Database File Path' : 'Connection String (DSN)'}
            </FormControl.Label>
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
                  ? 'postgres://user:password@host:5432/database'
                  : 'sqlserver://user:password@host:1433?database=migrator'
              }
              block
              monospace
            />
            <FormControl.Caption>
              {dbType === 'sqlite' && 'Path to the SQLite database file (will be created if it does not exist)'}
              {dbType === 'postgres' && 'PostgreSQL connection string with credentials'}
              {dbType === 'sqlserver' && 'SQL Server connection string with credentials'}
            </FormControl.Caption>
          </FormControl>

          <ConnectionTest
            label="Test Database Connection"
            disabled={!dbDSN}
            onTest={async () => {
              const result = await api.validateDatabaseConnection({ type: dbType, dsn: dbDSN });
              setDbValidated(result.valid);
              return result;
            }}
          />
        </div>
      )}

      {/* Step 2: Server Port */}
      {currentStep === 2 && (
        <div>
          <div className="flex items-center gap-3 mb-4">
            <ServerIcon size={24} />
            <Heading as="h2">Server Configuration</Heading>
          </div>
          <Text className="mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Configure the port for the GitHub Migrator web interface.
          </Text>

          <FormControl required className="mb-4">
            <FormControl.Label>Server Port</FormControl.Label>
            <TextInput
              type="number"
              value={serverPort}
              onChange={(e) => setServerPort(parseInt(e.target.value) || 8080)}
              min={1}
              max={65535}
              block
            />
            <FormControl.Caption>
              The port number for the web interface (default: 8080)
            </FormControl.Caption>
          </FormControl>

          {/* Summary */}
          <div 
            className="mt-8 p-4 rounded-lg border"
            style={{ 
              backgroundColor: 'var(--bgColor-muted)',
              borderColor: 'var(--borderColor-default)',
            }}
          >
            <Heading as="h3" className="text-base mb-3">Configuration Summary</Heading>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <Text style={{ color: 'var(--fgColor-muted)' }}>Database:</Text>
                <Text className="font-mono">{dbType}</Text>
              </div>
              <div className="flex justify-between">
                <Text style={{ color: 'var(--fgColor-muted)' }}>Server Port:</Text>
                <Text className="font-mono">{serverPort}</Text>
              </div>
            </div>
          </div>

          <Flash variant="warning" className="mt-6">
            <AlertIcon />
            <Text className="ml-2">
              Applying this configuration will restart the server. After restart, you'll be guided through
              configuring your destination and adding your first source.
            </Text>
          </Flash>
        </div>
      )}

      {/* Navigation Buttons */}
      <div className="flex justify-between mt-8 pt-6 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
        <Button
          variant="default"
          onClick={handleBack}
          disabled={currentStep === 1}
        >
          Back
        </Button>

        <div className="flex gap-2">
          {currentStep < 2 ? (
            <Button
              variant="primary"
              onClick={handleNext}
              disabled={!canProceedFromStep(currentStep)}
            >
              Continue
            </Button>
          ) : (
            <Button
              variant="primary"
              onClick={handleApplyConfig}
              disabled={!canProceedFromStep(currentStep) || applyingConfig}
            >
              {applyingConfig ? 'Applying...' : 'Apply & Restart'}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
