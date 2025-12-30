import { Link } from 'react-router-dom';
import { Button, Text, Heading } from '@primer/react';
import { CheckCircleIcon, CircleIcon, GearIcon } from '@primer/octicons-react';

interface SetupProgressProps {
  destinationConfigured: boolean;
  sourcesConfigured: boolean;
  sourceCount: number;
  batchesCreated: boolean;
  batchCount: number;
}

interface SetupStepProps {
  completed: boolean;
  title: string;
  description: string;
  actionLabel: string;
  actionLink: string;
  disabled?: boolean;
}

function SetupStep({ completed, title, description, actionLabel, actionLink, disabled }: SetupStepProps) {
  return (
    <div 
      className="flex items-start gap-4 p-4 rounded-lg border"
      style={{
        backgroundColor: completed ? 'var(--bgColor-success-muted)' : 'var(--bgColor-default)',
        borderColor: completed ? 'var(--borderColor-success-muted)' : 'var(--borderColor-default)',
        opacity: disabled ? 0.5 : 1,
      }}
    >
      <div className="flex-shrink-0 mt-1">
        {completed ? (
          <CheckCircleIcon size={24} fill="var(--fgColor-success)" />
        ) : (
          <CircleIcon size={24} fill="var(--fgColor-muted)" />
        )}
      </div>
      <div className="flex-grow">
        <Text className="font-semibold block" style={{ color: 'var(--fgColor-default)' }}>
          {title}
        </Text>
        <Text className="text-sm block mt-1" style={{ color: 'var(--fgColor-muted)' }}>
          {description}
        </Text>
      </div>
      <div className="flex-shrink-0">
        {!completed && !disabled && (
          <Link to={actionLink}>
            <Button variant="primary" size="small">
              {actionLabel}
            </Button>
          </Link>
        )}
        {completed && (
          <Link to={actionLink}>
            <Button variant="invisible" size="small">
              View
            </Button>
          </Link>
        )}
      </div>
    </div>
  );
}

export function SetupProgress({ 
  destinationConfigured, 
  sourcesConfigured, 
  sourceCount,
  batchesCreated,
  batchCount,
}: SetupProgressProps) {
  // Don't show if everything is configured
  if (destinationConfigured && sourcesConfigured && batchesCreated) {
    return null;
  }

  return (
    <div 
      className="mb-8 p-6 rounded-lg border"
      style={{
        backgroundColor: 'var(--bgColor-muted)',
        borderColor: 'var(--borderColor-default)',
      }}
    >
      <div className="flex items-center gap-3 mb-2">
        <GearIcon size={24} />
        <Heading as="h2" className="text-xl">
          Complete Your Setup
        </Heading>
      </div>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Follow these steps to configure GitHub Migrator and start migrating repositories.
      </Text>

      <div className="space-y-3">
        <SetupStep
          completed={destinationConfigured}
          title="Configure Destination"
          description={
            destinationConfigured 
              ? "Destination GitHub instance is configured and ready."
              : "Set up your destination GitHub instance where repositories will be migrated to."
          }
          actionLabel="Configure"
          actionLink="/settings"
        />

        <SetupStep
          completed={sourcesConfigured}
          title="Add Migration Sources"
          description={
            sourcesConfigured 
              ? `${sourceCount} source${sourceCount !== 1 ? 's' : ''} configured.`
              : "Add GitHub or Azure DevOps sources to discover repositories for migration."
          }
          actionLabel="Add Source"
          actionLink="/sources"
          disabled={!destinationConfigured}
        />

        <SetupStep
          completed={batchesCreated}
          title="Create Your First Batch"
          description={
            batchesCreated 
              ? `${batchCount} batch${batchCount !== 1 ? 'es' : ''} created.`
              : "Organize repositories into batches for coordinated migration."
          }
          actionLabel="Create Batch"
          actionLink="/batches/new"
          disabled={!sourcesConfigured}
        />
      </div>
    </div>
  );
}

