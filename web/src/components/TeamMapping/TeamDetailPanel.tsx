import { useState } from 'react';
import {
  Button,
  Flash,
  Label,
  Spinner,
  UnderlineNav,
  ProgressBar,
} from '@primer/react';
import {
  XIcon,
  PeopleIcon,
  RepoIcon,
  ArrowRightIcon,
  CheckIcon,
  AlertIcon,
  ClockIcon,
  SyncIcon,
  RocketIcon,
} from '@primer/octicons-react';
import { useTeamDetail } from '../../hooks/useQueries';
import { useExecuteTeamMigration } from '../../hooks/useMutations';
import { TeamMigrationStatus, TeamMigrationCompleteness } from '../../types';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

interface TeamDetailPanelProps {
  org: string;
  teamSlug: string;
  onClose: () => void;
  onEditMapping?: (org: string, slug: string) => void;
  onMigrationStarted?: () => void;
}

const migrationStatusColors: Record<TeamMigrationStatus | string, 'default' | 'accent' | 'success' | 'attention' | 'danger'> = {
  pending: 'default',
  in_progress: 'accent',
  completed: 'success',
  failed: 'danger',
};

const migrationStatusIcons: Record<TeamMigrationStatus | string, React.ReactNode> = {
  pending: <ClockIcon size={12} />,
  in_progress: <Spinner size="small" />,
  completed: <CheckIcon size={12} />,
  failed: <AlertIcon size={12} />,
};

// Migration completeness colors and labels
const completenessColors: Record<TeamMigrationCompleteness, 'default' | 'accent' | 'success' | 'attention' | 'danger'> = {
  pending: 'default',
  team_only: 'accent',
  partial: 'attention',
  complete: 'success',
  needs_sync: 'attention',
};

const completenessLabels: Record<TeamMigrationCompleteness, string> = {
  pending: 'Pending',
  team_only: 'Team Only',
  partial: 'Partial',
  complete: 'Complete',
  needs_sync: 'Needs Sync',
};

const completenessDescriptions: Record<TeamMigrationCompleteness, string> = {
  pending: 'Team has not been migrated yet',
  team_only: 'Team created, awaiting repo migrations',
  partial: 'Some repo permissions synced',
  complete: 'All repo permissions synced',
  needs_sync: 'New repos available for sync',
};

const permissionColors: Record<string, 'default' | 'accent' | 'success' | 'attention' | 'danger' | 'done'> = {
  pull: 'default',
  triage: 'attention',
  push: 'accent',
  maintain: 'success',
  admin: 'danger',
};

export function TeamDetailPanel({ org, teamSlug, onClose, onEditMapping, onMigrationStarted }: TeamDetailPanelProps) {
  const { showError } = useToast();
  const [activeTab, setActiveTab] = useState<'members' | 'repositories'>('members');
  const [isMigrating, setIsMigrating] = useState(false);
  const { data: team, isLoading, error, refetch } = useTeamDetail(org, teamSlug);
  const executeMigration = useExecuteTeamMigration();

  const handleMigrateTeam = async () => {
    if (!team?.mapping?.destination_org || !team?.mapping?.destination_team_slug) {
      return;
    }
    
    setIsMigrating(true);
    try {
      await executeMigration.mutateAsync({
        source_org: org,
        source_team_slug: teamSlug,
        dry_run: false,
      });
      onMigrationStarted?.();
      // Refetch team details to get updated status
      setTimeout(() => {
        refetch();
        setIsMigrating(false);
      }, 1000);
    } catch (error) {
      handleApiError(error, showError, 'Failed to migrate team');
      setIsMigrating(false);
    }
  };

  // Determine if team can be migrated
  const canMigrate = team?.mapping?.mapping_status === 'mapped' && 
    team?.mapping?.destination_org && 
    team?.mapping?.destination_team_slug;
  
  // Determine if team needs sync (already created but has repos to sync)
  const needsSync = team?.mapping?.sync_status === 'needs_sync' || 
    team?.mapping?.sync_status === 'partial' ||
    (team?.mapping?.team_created_in_dest && 
     (team?.mapping?.repos_synced ?? 0) < (team?.mapping?.repos_eligible ?? 0));

  if (error) {
    return (
      <div
        className="fixed top-0 right-0 w-[500px] h-screen flex flex-col overflow-hidden"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderLeft: '1px solid var(--borderColor-default)',
          boxShadow: '0 0 20px rgba(0,0,0,0.2)',
          zIndex: 100,
        }}
      >
        <div
          className="p-4 flex justify-between items-center"
          style={{ borderBottom: '1px solid var(--borderColor-default)' }}
        >
          <h2 className="text-base font-semibold m-0">Team Details</h2>
          <Button variant="invisible" onClick={onClose} aria-label="Close">
            <XIcon size={16} />
          </Button>
        </div>
        <div className="p-6">
          <Flash variant="danger">Failed to load team details: {error.message}</Flash>
        </div>
      </div>
    );
  }

  return (
    <div
      className="fixed top-0 right-0 w-[500px] h-screen flex flex-col overflow-hidden"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderLeft: '1px solid var(--borderColor-default)',
        boxShadow: '0 0 20px rgba(0,0,0,0.2)',
        zIndex: 100,
      }}
    >
      {/* Header */}
      <div
        className="p-4 flex justify-between items-center"
        style={{ borderBottom: '1px solid var(--borderColor-default)' }}
      >
        <div>
          <h2 className="text-base font-semibold m-0 mb-1">
            {isLoading ? 'Loading...' : team?.name || teamSlug}
          </h2>
          <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            {org}/{teamSlug}
          </span>
        </div>
        <Button variant="invisible" onClick={onClose} aria-label="Close">
          <XIcon size={16} />
        </Button>
      </div>

      {isLoading ? (
        <div className="p-6 flex justify-center">
          <Spinner size="large" />
        </div>
      ) : team ? (
        <>
          {/* Team Info */}
          <div className="p-4" style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
            <div className="flex gap-2 flex-wrap mb-2">
              <Label variant="accent">{team.privacy}</Label>
              <Label>{team.members.length} members</Label>
              <Label>{team.repositories.length} repos</Label>
            </div>
            {team.description && (
              <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                {team.description}
              </span>
            )}
          </div>

          {/* Mapping Status */}
          <div
            className="p-4"
            style={{
              borderBottom: '1px solid var(--borderColor-default)',
              backgroundColor: 'var(--bgColor-muted)',
            }}
          >
            <h3 className="text-xs font-semibold mb-2 mt-0">Destination Mapping</h3>
            {team.mapping ? (
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <span className="font-medium">{org}/{teamSlug}</span>
                  <ArrowRightIcon size={16} />
                  {team.mapping.destination_org && team.mapping.destination_team_slug ? (
                    <span className="font-medium" style={{ color: 'var(--fgColor-success)' }}>
                      {team.mapping.destination_org}/{team.mapping.destination_team_slug}
                    </span>
                  ) : (
                    <span className="italic" style={{ color: 'var(--fgColor-muted)' }}>
                      Not mapped
                    </span>
                  )}
                </div>
                <div className="flex gap-2 items-center flex-wrap">
                  <Label
                    variant={
                      team.mapping.mapping_status === 'mapped'
                        ? 'success'
                        : team.mapping.mapping_status === 'skipped'
                        ? 'default'
                        : 'attention'
                    }
                  >
                    {team.mapping.mapping_status}
                  </Label>
                  {team.mapping.migration_status && (
                    <Label variant={migrationStatusColors[team.mapping.migration_status] || 'default'}>
                      <span className="flex items-center gap-1">
                        {migrationStatusIcons[team.mapping.migration_status]}
                        <span>{team.mapping.migration_status}</span>
                      </span>
                    </Label>
                  )}
                  {/* Show migration completeness badge */}
                  {team.mapping.migration_completeness && team.mapping.migration_completeness !== 'pending' && (
                    <Label variant={completenessColors[team.mapping.migration_completeness] || 'default'}>
                      <span className="flex items-center gap-1">
                        {team.mapping.migration_completeness === 'needs_sync' && <SyncIcon size={12} />}
                        {completenessLabels[team.mapping.migration_completeness]}
                      </span>
                    </Label>
                  )}
                </div>

                {/* Show repo sync progress */}
                {team.mapping.team_created_in_dest && (
                  <div className="mt-3">
                    <div className="flex justify-between text-xs mb-1">
                      <span style={{ color: 'var(--fgColor-muted)' }}>
                        Repo Permissions Synced
                      </span>
                      <span>
                        {team.mapping.repos_synced ?? 0} / {team.mapping.repos_eligible ?? 0}
                        {team.mapping.total_source_repos > 0 && team.mapping.repos_eligible < team.mapping.total_source_repos && (
                          <span style={{ color: 'var(--fgColor-muted)' }}>
                            {' '}(of {team.mapping.total_source_repos} total)
                          </span>
                        )}
                      </span>
                    </div>
                    {team.mapping.repos_eligible > 0 && (
                      <ProgressBar
                        progress={Math.round(((team.mapping.repos_synced ?? 0) / team.mapping.repos_eligible) * 100)}
                        bg={(team.mapping.repos_synced ?? 0) >= team.mapping.repos_eligible ? 'success.emphasis' : 'accent.emphasis'}
                      />
                    )}
                    {team.mapping.migration_completeness && (
                      <span className="text-xs mt-1 block" style={{ color: 'var(--fgColor-muted)' }}>
                        {completenessDescriptions[team.mapping.migration_completeness]}
                      </span>
                    )}
                    {team.mapping.last_synced_at && (
                      <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                        Last synced: {new Date(team.mapping.last_synced_at).toLocaleString()}
                      </span>
                    )}
                  </div>
                )}

                {team.mapping.error_message && (
                  <Flash variant="danger" className="mt-2 text-xs">
                    {team.mapping.error_message}
                  </Flash>
                )}
                <div className="flex gap-2 mt-3">
                  {onEditMapping && (
                    <Button
                      size="small"
                      onClick={() => onEditMapping(org, teamSlug)}
                    >
                      Edit Mapping
                    </Button>
                  )}
                  {canMigrate && (
                    <Button
                      size="small"
                      variant="primary"
                      onClick={handleMigrateTeam}
                      disabled={isMigrating || executeMigration.isPending}
                      leadingVisual={isMigrating ? () => <Spinner size="small" /> : needsSync ? SyncIcon : RocketIcon}
                    >
                      {isMigrating ? 'Migrating...' : needsSync ? 'Sync Permissions' : team.mapping.team_created_in_dest ? 'Re-sync Team' : 'Migrate Team'}
                    </Button>
                  )}
                </div>
              </div>
            ) : (
              <div>
                <span
                  className="italic mb-2 block"
                  style={{ color: 'var(--fgColor-muted)' }}
                >
                  No mapping configured
                </span>
                {onEditMapping && (
                  <Button size="small" onClick={() => onEditMapping(org, teamSlug)}>
                    Create Mapping
                  </Button>
                )}
              </div>
            )}
          </div>

          {/* Tabs */}
          <UnderlineNav aria-label="Team sections">
            <UnderlineNav.Item
              aria-current={activeTab === 'members' ? 'page' : undefined}
              onClick={() => setActiveTab('members')}
              icon={PeopleIcon}
            >
              Members ({team.members.length})
            </UnderlineNav.Item>
            <UnderlineNav.Item
              aria-current={activeTab === 'repositories' ? 'page' : undefined}
              onClick={() => setActiveTab('repositories')}
              icon={RepoIcon}
            >
              Repositories ({team.repositories.length})
            </UnderlineNav.Item>
          </UnderlineNav>

          {/* Tab Content */}
          <div className="flex-1 overflow-auto p-4">
            {activeTab === 'members' ? (
              <div>
                {team.members.length === 0 ? (
                  <span className="italic" style={{ color: 'var(--fgColor-muted)' }}>
                    No members discovered
                  </span>
                ) : (
                  <ul className="list-none p-0 m-0">
                    {team.members.map((member) => (
                      <li
                        key={member.login}
                        className="flex justify-between items-center py-2"
                        style={{ borderBottom: '1px solid var(--borderColor-muted)' }}
                      >
                        <span className="font-medium">{member.login}</span>
                        <Label
                          variant={member.role === 'maintainer' ? 'accent' : 'default'}
                          size="small"
                        >
                          {member.role}
                        </Label>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            ) : (
              <div>
                {team.repositories.length === 0 ? (
                  <span className="italic" style={{ color: 'var(--fgColor-muted)' }}>
                    No repositories discovered
                  </span>
                ) : (
                  <ul className="list-none p-0 m-0">
                    {team.repositories.map((repo) => (
                      <li
                        key={repo.full_name}
                        className="flex justify-between items-center py-2"
                        style={{ borderBottom: '1px solid var(--borderColor-muted)' }}
                      >
                        <div>
                          <span className="font-medium block">{repo.full_name}</span>
                          {repo.migration_status && (
                            <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                              Migration: {repo.migration_status}
                            </span>
                          )}
                        </div>
                        <Label
                          variant={permissionColors[repo.permission] || 'default'}
                          size="small"
                        >
                          {repo.permission}
                        </Label>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </div>
        </>
      ) : null}
    </div>
  );
}
