import { useState } from 'react';
import { useParams, Link as RouterLink, useLocation } from 'react-router-dom';
import { Button, UnderlineNav, Textarea, FormControl, Link, useTheme, Dialog } from '@primer/react';
import { CalendarIcon, AlertIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { Repository, MigrationHistory } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { MigrationReadinessTab } from './MigrationReadinessTab';
import { TechnicalProfileTab } from './TechnicalProfileTab';
import { DependenciesTab } from './DependenciesTab';
import { ActivityLogTab } from './ActivityLogTab';
import { useRepositoryWithHistory, useBatches } from '../../hooks/useQueries';
import { useRediscoverRepository, useUnlockRepository, useRollbackRepository, useMarkRepositoryWontMigrate } from '../../hooks/useMutations';
import { useToast } from '../../contexts/ToastContext';
import { useDialogState } from '../../hooks/useDialogState';
import { formatDuration } from '../../utils/format';

// Helper to get duration from migration history
function getDurationFromHistory(history: MigrationHistory[], phase: 'dry_run' | 'migration'): number | null {
  // Find the most recent completed entry for this phase
  const entry = history
    .filter(h => h.phase === phase && h.status === 'completed' && h.duration_seconds)
    .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())[0];
  return entry?.duration_seconds ?? null;
}

export function RepositoryDetail() {
  const { fullName } = useParams<{ fullName: string }>();
  const location = useLocation();
  const locationState = location.state as { fromBatch?: boolean; batchId?: number; batchName?: string } | null;
  const { data, isLoading, isFetching } = useRepositoryWithHistory(fullName || '');
  const repository: Repository | undefined = data?.repository;
  const migrationHistory: MigrationHistory[] = data?.history ?? [];
  const { data: allBatches = [] } = useBatches();
  const rediscoverMutation = useRediscoverRepository();
  const unlockMutation = useUnlockRepository();
  const rollbackMutation = useRollbackRepository();
  const markWontMigrateMutation = useMarkRepositoryWontMigrate();
  const { showSuccess, showError } = useToast();
  const { theme } = useTheme();
  
  const [migrating, setMigrating] = useState(false);
  const [activeTab, setActiveTab] = useState<'readiness' | 'profile' | 'relationships' | 'activity'>('readiness');
  const [rollbackReason, setRollbackReason] = useState('');
  
  // Dialog states using useDialogState hook
  const rollbackDialog = useDialogState();
  const rediscoverDialog = useDialogState();
  const unlockDialog = useDialogState();
  const wontMigrateDialog = useDialogState();
  const migrationDialog = useDialogState<{ isDryRun: boolean }>();

  const handleStartMigration = (dryRun: boolean = false) => {
    if (!repository || migrating) return;
    migrationDialog.open({ isDryRun: dryRun });
  };

  const confirmStartMigration = async () => {
    if (!repository || !migrationDialog.data) return;

    const isDryRun = migrationDialog.data.isDryRun;
    migrationDialog.close();
    setMigrating(true);
    try {
      await api.startMigration({
        repository_ids: [repository.id],
        dry_run: isDryRun,
      });
      
      // Show success message
      showSuccess(`${isDryRun ? 'Dry run' : 'Migration'} started successfully!`);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      const errorMessage = err.response?.data?.error || err.message || 'Failed to start migration. Please try again.';
      showError(errorMessage);
    } finally {
      setMigrating(false);
    }
  };

  const handleRediscover = () => {
    if (!repository || !fullName || rediscoverMutation.isPending) return;
    rediscoverDialog.open();
  };

  const confirmRediscover = async () => {
    if (!fullName) return;

    rediscoverDialog.close();
    try {
      await rediscoverMutation.mutateAsync(decodeURIComponent(fullName));
      showSuccess('Re-discovery started! Repository data will be updated shortly.');
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      const errorMessage = err.response?.data?.error || err.message || 'Failed to start re-discovery. Please try again.';
      showError(errorMessage);
    }
  };

  const handleUnlock = () => {
    if (!repository || !fullName || unlockMutation.isPending) return;
    unlockDialog.open();
  };

  const confirmUnlock = async () => {
    if (!fullName) return;

    unlockDialog.close();
    try {
      await unlockMutation.mutateAsync(decodeURIComponent(fullName));
      showSuccess('Repository unlocked successfully!');
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      const errorMessage = err.response?.data?.error || err.message || 'Failed to unlock repository. Please try again.';
      showError(errorMessage);
    }
  };

  const handleRollback = async () => {
    if (!repository || !fullName || rollbackMutation.isPending) return;

    try {
      await rollbackMutation.mutateAsync({ 
        fullName: decodeURIComponent(fullName), 
        reason: rollbackReason 
      });
      rollbackDialog.close();
      setRollbackReason('');
      showSuccess('Repository rolled back successfully! It can now be migrated again.');
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      const errorMessage = err.response?.data?.error || err.message || 'Failed to rollback repository. Please try again.';
      showError(errorMessage);
    }
  };

  const handleToggleWontMigrate = () => {
    if (!repository || !fullName || markWontMigrateMutation.isPending) return;
    wontMigrateDialog.open();
  };

  const confirmToggleWontMigrate = async () => {
    if (!repository || !fullName) return;

    const isWontMigrate = repository.status === 'wont_migrate';
    const action = isWontMigrate ? 'unmark' : 'mark';
    
    wontMigrateDialog.close();
    try {
      await markWontMigrateMutation.mutateAsync({ 
        fullName: decodeURIComponent(fullName), 
        unmark: isWontMigrate 
      });
      showSuccess(`Repository ${action}ed successfully!`);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      const errorMsg = err.response?.data?.error || `Failed to ${action} repository. Please try again.`;
      showError(errorMsg);
    }
  };

  if (isLoading) return <LoadingSpinner />;
  if (!repository) return <div className="text-center py-12 text-gray-500">Repository not found</div>;

  const canMigrate = ['pending', 'dry_run_complete', 'dry_run_failed', 'pre_migration_complete', 'migration_failed', 'rolled_back'].includes(
    repository.status
  );

  const isInActiveMigration = [
    'queued_for_migration',
    'dry_run_in_progress',
    'dry_run_queued',
    'migrating_content',
    'pre_migration',
    'archive_generating',
    'post_migration',
  ].includes(repository.status);
  
  // Find the current batch name
  const currentBatch = repository.batch_id 
    ? allBatches.find(b => b.id === repository.batch_id)
    : null;

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      {/* Header */}
      <div className="rounded-lg shadow-sm p-6 mb-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
        {/* Breadcrumbs */}
        <nav aria-label="Breadcrumb" className="mb-4">
          <ol className="flex items-center text-sm">
            <li>
              <Link as={RouterLink} to="/" muted>Dashboard</Link>
            </li>
            <li className="mx-2" style={{ color: 'var(--fgColor-muted)' }}>/</li>
            {locationState?.fromBatch && locationState?.batchId ? (
              <>
                <li>
                  <Link as={RouterLink} to="/batches" muted>Batches</Link>
                </li>
                <li className="mx-2" style={{ color: 'var(--fgColor-muted)' }}>/</li>
                <li className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  {locationState.batchName || `Batch #${locationState.batchId}`}
                </li>
              </>
            ) : repository.ado_project ? (
              <>
                <li>
                  <Link 
                    as={RouterLink}
                    to={`/repositories?organization=${encodeURIComponent(repository.full_name.split('/')[0])}`}
                    muted
                  >
                    {repository.full_name.split('/')[0]}
                  </Link>
                </li>
                <li className="mx-2" style={{ color: 'var(--fgColor-muted)' }}>/</li>
                <li>
                  <Link 
                    as={RouterLink}
                    to={`/repositories?organization=${encodeURIComponent(repository.full_name.split('/')[0])}&project=${encodeURIComponent(repository.ado_project)}`}
                    muted
                  >
                    {repository.ado_project}
                  </Link>
                </li>
                <li className="mx-2" style={{ color: 'var(--fgColor-muted)' }}>/</li>
                <li className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  {repository.full_name.split('/').slice(1).join('/')}
                </li>
              </>
            ) : (
              <>
                <li>
                  <Link 
                    as={RouterLink}
                    to={`/repositories?organization=${encodeURIComponent(repository.full_name.split('/')[0])}`}
                    muted
                  >
                    {repository.full_name.split('/')[0]}
                  </Link>
                </li>
                <li className="mx-2" style={{ color: 'var(--fgColor-muted)' }}>/</li>
                <li className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  {repository.full_name.split('/').slice(1).join('/')}
                </li>
              </>
            )}
          </ol>
        </nav>
        <div className="flex justify-between items-start">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-3xl font-light" style={{ color: 'var(--fgColor-default)' }}>
                {repository.full_name}
              </h1>
              {(() => {
                // Determine validation status
                const hasBlockingIssues = repository.has_oversized_repository || 
                  repository.has_oversized_commits || 
                  repository.has_long_refs || 
                  repository.has_blocking_files;
                const hasWarnings = (repository.estimated_metadata_size && repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024) || 
                  repository.has_large_file_warnings;
                
                if (hasBlockingIssues) {
                  return (
                    <span 
                      className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                      style={{
                        backgroundColor: 'var(--danger-subtle)',
                        color: 'var(--fgColor-danger)'
                      }}
                    >
                      ⚠️ Validation Failed
                    </span>
                  );
                } else if (hasWarnings) {
                  return (
                    <span 
                      className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                      style={{
                        backgroundColor: 'var(--attention-subtle)',
                        color: 'var(--fgColor-attention)'
                      }}
                    >
                      ⚠ Has Warnings
                    </span>
                  );
                } else {
                  return (
                    <span 
                      className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                      style={{
                        backgroundColor: 'var(--success-subtle)',
                        color: 'var(--fgColor-success)'
                      }}
                    >
                      ✓ Validation Passed
                    </span>
                  );
                }
              })()}
            </div>
            <div className="flex items-center gap-4 mb-4">
              <StatusBadge status={repository.status} />
              {repository.priority === 1 && <Badge color="purple">High Priority</Badge>}
              {currentBatch && <Badge color="blue">{currentBatch.name}</Badge>}
              {repository.is_source_locked && <Badge color="orange">🔒 Source Locked</Badge>}
            </div>

            {/* Compact Timestamp Display */}
            <div 
              className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm mb-4 pb-4"
              style={{ 
                color: 'var(--fgColor-muted)',
                borderBottom: '1px solid var(--borderColor-default)' 
              }}
            >
              <div className="flex items-center gap-1.5">
                <CalendarIcon size={16} />
                <TimestampDisplay 
                  timestamp={repository.discovered_at} 
                  label="Discovered"
                  size="sm"
                />
              </div>
              {repository.last_discovery_at && (
                <div className="flex items-center gap-1">
                  <span style={{ color: 'var(--fgColor-muted)' }}>•</span>
                  <TimestampDisplay 
                    timestamp={repository.last_discovery_at} 
                    label="Data refreshed"
                    size="sm"
                  />
                </div>
              )}
              {repository.last_dry_run_at && (
                <div className="flex items-center gap-1">
                  <span style={{ color: 'var(--fgColor-muted)' }}>•</span>
                  <TimestampDisplay 
                    timestamp={repository.last_dry_run_at} 
                    label="Dry run"
                    size="sm"
                  />
                  {(() => {
                    const duration = getDurationFromHistory(migrationHistory, 'dry_run');
                    return duration ? (
                      <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                        ({formatDuration(duration)})
                      </span>
                    ) : null;
                  })()}
                </div>
              )}
              {repository.migrated_at && (
                <div className="flex items-center gap-1">
                  <span style={{ color: 'var(--fgColor-muted)' }}>•</span>
                  <TimestampDisplay 
                    timestamp={repository.migrated_at} 
                    label="Migrated"
                    size="sm"
                  />
                  {(() => {
                    const duration = getDurationFromHistory(migrationHistory, 'migration');
                    return duration ? (
                      <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                        ({formatDuration(duration)})
                      </span>
                    ) : null;
                  })()}
                </div>
              )}
            </div>
          </div>

          {/* Migration Actions */}
          <div className="flex flex-col gap-3 ml-6">
            <button
              onClick={handleRediscover}
              disabled={rediscoverMutation.isPending}
              className="px-4 py-2 rounded-md text-sm font-medium border disabled:opacity-50 disabled:cursor-not-allowed transition-all cursor-pointer"
              style={{ 
                whiteSpace: 'nowrap',
                border: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--bgColor-muted)',
                color: 'var(--fgColor-default)'
              }}
              onMouseEnter={(e) => {
                if (!rediscoverMutation.isPending) {
                  e.currentTarget.style.backgroundColor = 'var(--control-bgColor-hover)';
                }
              }}
              onMouseLeave={(e) => {
                if (!rediscoverMutation.isPending) {
                  e.currentTarget.style.backgroundColor = 'var(--bgColor-muted)';
                }
              }}
            >
              {rediscoverMutation.isPending ? 'Re-discovering...' : 'Re-discover'}
            </button>
            
            {/* Won't Migrate Toggle */}
            {!isInActiveMigration && repository.status !== 'complete' && (
              repository.status === 'wont_migrate' ? (
                <Button
                  onClick={handleToggleWontMigrate}
                  disabled={markWontMigrateMutation.isPending}
                  variant="primary"
                  style={{ whiteSpace: 'nowrap' }}
                >
                  {markWontMigrateMutation.isPending ? 'Processing...' : 'Unmark Won\'t Migrate'}
                </Button>
              ) : (
                <button
                  onClick={handleToggleWontMigrate}
                  disabled={markWontMigrateMutation.isPending}
                  className="px-4 py-2 rounded-md text-sm font-medium border disabled:opacity-50 disabled:cursor-not-allowed transition-all cursor-pointer"
                  style={{ 
                    whiteSpace: 'nowrap',
                    border: '1px solid var(--borderColor-attention-emphasis)',
                    backgroundColor: 'var(--bgColor-attention-muted)',
                    color: 'var(--fgColor-attention)'
                  }}
                  onMouseEnter={(e) => {
                    if (!markWontMigrateMutation.isPending) {
                      e.currentTarget.style.backgroundColor = 'var(--control-attention-bgColor-hover)';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!markWontMigrateMutation.isPending) {
                      e.currentTarget.style.backgroundColor = 'var(--bgColor-attention-muted)';
                    }
                  }}
                >
                  Mark as Won't Migrate
                </button>
              )
            )}
            
            {canMigrate && repository.status !== 'migration_failed' && repository.status !== 'dry_run_failed' && (
              <>
                <Button
                  onClick={() => handleStartMigration(true)}
                  disabled={migrating}
                  variant="primary"
                  style={{ whiteSpace: 'nowrap' }}
                >
                  {migrating ? 'Processing...' : 'Dry Run'}
                </Button>
                <button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  className="px-4 py-2 rounded-md text-sm font-medium border disabled:opacity-50 disabled:cursor-not-allowed transition-all cursor-pointer"
                  style={{ 
                    whiteSpace: 'nowrap',
                    backgroundColor: '#1a7f37',
                    color: '#ffffff',
                    border: '1px solid #1a7f37',
                    fontWeight: 600
                  }}
                  onMouseEnter={(e) => {
                    if (!migrating) {
                      e.currentTarget.style.backgroundColor = '#2da44e';
                      e.currentTarget.style.borderColor = '#2da44e';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!migrating) {
                      e.currentTarget.style.backgroundColor = '#1a7f37';
                      e.currentTarget.style.borderColor = '#1a7f37';
                    }
                  }}
                >
                  {migrating ? 'Processing...' : 'Start Migration'}
                </button>
              </>
            )}
            {repository.status === 'dry_run_failed' && (
              <>
                <Button
                  onClick={() => handleStartMigration(true)}
                  disabled={migrating}
                  variant="danger"
                  style={{ whiteSpace: 'nowrap' }}
                >
                  {migrating ? 'Re-running...' : 'Re-run Dry Run'}
                </Button>
                <Button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  variant="primary"
                  style={{ whiteSpace: 'nowrap' }}
                >
                  {migrating ? 'Starting...' : 'Start Migration Anyway'}
                </Button>
              </>
            )}
            {repository.status === 'migration_failed' && (
              <>
                <Button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  variant="danger"
                  style={{ whiteSpace: 'nowrap' }}
                >
                  Retry Migration
                </Button>
                {repository.is_source_locked && repository.source_migration_id && (
                  <Button
                    onClick={handleUnlock}
                    disabled={unlockMutation.isPending}
                    variant="danger"
                    style={{ whiteSpace: 'nowrap' }}
                  >
                    {unlockMutation.isPending ? 'Unlocking...' : '🔓 Unlock Source'}
                  </Button>
                )}
              </>
            )}
            {repository.status === 'complete' && (
              <Button
                onClick={() => rollbackDialog.open()}
                disabled={rollbackMutation.isPending}
                variant="danger"
                style={{ whiteSpace: 'nowrap' }}
              >
                {rollbackMutation.isPending ? 'Rolling back...' : 'Rollback Migration'}
              </Button>
            )}
          </div>
        </div>

        {/* Links */}
        <div className="mt-4 flex gap-4 text-sm">
          <a
            href={repository.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:underline font-medium"
            style={{ color: theme?.colors.accent.fg }}
          >
            View Source Repository →
          </a>
          {repository.destination_url && (
            <a
              href={repository.destination_url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:underline font-medium"
              style={{ color: theme?.colors.success.fg }}
            >
              View Migrated Repository →
            </a>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="rounded-lg shadow-sm mb-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
        <UnderlineNav aria-label="Repository details">
          <UnderlineNav.Item
            aria-current={activeTab === 'readiness' ? 'page' : undefined}
            onSelect={() => setActiveTab('readiness')}
          >
            Migration Readiness
          </UnderlineNav.Item>
          <UnderlineNav.Item
            aria-current={activeTab === 'profile' ? 'page' : undefined}
            onSelect={() => setActiveTab('profile')}
          >
            Technical Profile
          </UnderlineNav.Item>
          <UnderlineNav.Item
            aria-current={activeTab === 'relationships' ? 'page' : undefined}
            onSelect={() => setActiveTab('relationships')}
          >
            Relationships
          </UnderlineNav.Item>
          <UnderlineNav.Item
            aria-current={activeTab === 'activity' ? 'page' : undefined}
            onSelect={() => setActiveTab('activity')}
          >
            Activity Log
          </UnderlineNav.Item>
        </UnderlineNav>

        <div className="p-6">
          {activeTab === 'readiness' && (
            <MigrationReadinessTab
              repository={repository}
              allBatches={allBatches}
            />
          )}

          {activeTab === 'profile' && (
            <TechnicalProfileTab repository={repository} />
          )}

          {activeTab === 'relationships' && fullName && (
            <DependenciesTab fullName={fullName} />
          )}

          {activeTab === 'activity' && (
            <ActivityLogTab repository={repository} />
          )}
        </div>
      </div>

      {/* Rollback Dialog */}
      {rollbackDialog.isOpen && (
        <Dialog
          returnFocusRef={rollbackDialog.returnFocusRef}
          onClose={() => {
            rollbackDialog.close();
            setRollbackReason('');
          }}
          aria-labelledby="rollback-dialog-header"
        >
          <Dialog.Header id="rollback-dialog-header">
            Rollback Migration
          </Dialog.Header>
          <div style={{ padding: '16px' }}>
            <p style={{ fontSize: '14px', color: 'var(--fgColor-muted)', marginBottom: '16px' }}>
              This will mark the repository as rolled back and allow it to be migrated again in the future.
              You can optionally provide a reason for the rollback.
            </p>
            
            <FormControl>
              <FormControl.Label>Reason (optional)</FormControl.Label>
              <Textarea
                value={rollbackReason}
                onChange={(e) => setRollbackReason(e.target.value)}
                placeholder="e.g., CI/CD integration issues, workflow failures..."
                rows={3}
                disabled={rollbackMutation.isPending}
                block
              />
            </FormControl>
          </div>
          <div style={{ 
            padding: '12px 16px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px'
          }}>
            <Button
              onClick={() => {
                rollbackDialog.close();
                setRollbackReason('');
              }}
              disabled={rollbackMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              onClick={handleRollback}
              disabled={rollbackMutation.isPending}
              variant="danger"
            >
              {rollbackMutation.isPending ? 'Rolling back...' : 'Confirm Rollback'}
            </Button>
          </div>
        </Dialog>
      )}

      {/* Rediscover Confirmation Dialog */}
      {rediscoverDialog.isOpen && (
        <Dialog
          returnFocusRef={rediscoverDialog.returnFocusRef}
          onClose={rediscoverDialog.close}
          aria-labelledby="rediscover-dialog-header"
        >
          <Dialog.Header id="rediscover-dialog-header">
            Re-discover Repository
          </Dialog.Header>
          <div style={{ padding: '16px' }}>
            <p style={{ fontSize: '14px', color: 'var(--fgColor-default)' }}>
              Are you sure you want to re-discover this repository? This will update all repository data.
            </p>
          </div>
          <div style={{ 
            padding: '12px 16px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px'
          }}>
            <Button onClick={rediscoverDialog.close}>
              Cancel
            </Button>
            <Button variant="primary" onClick={confirmRediscover}>
              Re-discover
            </Button>
          </div>
        </Dialog>
      )}

      {/* Unlock Confirmation Dialog */}
      {unlockDialog.isOpen && (
        <Dialog
          returnFocusRef={unlockDialog.returnFocusRef}
          onClose={unlockDialog.close}
          aria-labelledby="unlock-dialog-header"
        >
          <Dialog.Header id="unlock-dialog-header">
            Unlock Repository
          </Dialog.Header>
          <div style={{ padding: '16px' }}>
            <p style={{ fontSize: '14px', color: 'var(--fgColor-default)' }}>
              Are you sure you want to unlock this repository? This will remove the lock from the source repository.
            </p>
          </div>
          <div style={{ 
            padding: '12px 16px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px'
          }}>
            <Button onClick={unlockDialog.close}>
              Cancel
            </Button>
            <Button variant="danger" onClick={confirmUnlock}>
              Unlock
            </Button>
          </div>
        </Dialog>
      )}

      {/* Won't Migrate Confirmation Dialog */}
      {wontMigrateDialog.isOpen && repository && (
        <Dialog
          returnFocusRef={wontMigrateDialog.returnFocusRef}
          onClose={wontMigrateDialog.close}
          aria-labelledby="wont-migrate-dialog-header"
        >
          <Dialog.Header id="wont-migrate-dialog-header">
            {repository.status === 'wont_migrate' ? 'Unmark Repository' : 'Mark Repository as Won\'t Migrate'}
          </Dialog.Header>
          <div style={{ padding: '16px 24px' }}>
            {repository.status === 'wont_migrate' ? (
              <p style={{ 
                fontSize: '14px', 
                color: 'var(--fgColor-default)',
                lineHeight: '1.5',
                margin: 0
              }}>
                Are you sure you want to unmark this repository? It will be changed to <strong>pending</strong> status and can be added to migration batches again.
              </p>
            ) : (
              <>
                <div style={{
                  display: 'flex',
                  gap: '12px',
                  padding: '12px',
                  borderRadius: '6px',
                  backgroundColor: 'var(--bgColor-attention-muted)',
                  border: '1px solid var(--borderColor-attention-emphasis)',
                  marginBottom: '16px'
                }}>
                  <div style={{ flexShrink: 0, marginTop: '2px', color: 'var(--fgColor-attention)' }}>
                    <AlertIcon size={16} />
                  </div>
                  <div>
                    <p style={{ 
                      fontSize: '14px', 
                      fontWeight: 600,
                      color: 'var(--fgColor-attention)',
                      margin: '0 0 4px 0'
                    }}>
                      This will exclude the repository from migration
                    </p>
                    <p style={{ 
                      fontSize: '13px', 
                      color: 'var(--fgColor-default)',
                      lineHeight: '1.5',
                      margin: 0
                    }}>
                      The repository will be marked as won't migrate and cannot be added to batches or included in migration progress tracking.
                    </p>
                  </div>
                </div>
                <p style={{ 
                  fontSize: '14px', 
                  color: 'var(--fgColor-muted)',
                  lineHeight: '1.5',
                  margin: 0
                }}>
                  Use this for repositories that don't need to be migrated, such as archived projects or test repositories.
                </p>
              </>
            )}
          </div>
          <div style={{ 
            padding: '16px 24px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px',
            backgroundColor: 'var(--bgColor-muted)'
          }}>
            <Button onClick={wontMigrateDialog.close}>
              Cancel
            </Button>
            <Button variant={repository.status === 'wont_migrate' ? 'primary' : 'danger'} onClick={confirmToggleWontMigrate}>
              {repository.status === 'wont_migrate' ? 'Unmark Repository' : 'Mark as Won\'t Migrate'}
            </Button>
          </div>
        </Dialog>
      )}

      {/* Migration Confirmation Dialog */}
      {migrationDialog.isOpen && repository && migrationDialog.data && (
        <Dialog
          returnFocusRef={migrationDialog.returnFocusRef}
          onClose={migrationDialog.close}
          aria-labelledby="migration-dialog-header"
        >
          <Dialog.Header id="migration-dialog-header">
            {migrationDialog.data.isDryRun ? 'Confirm Dry Run' : 'Confirm Migration'}
          </Dialog.Header>
          <div style={{ padding: '16px 24px' }}>
            <p style={{ 
              fontSize: '14px', 
              color: 'var(--fgColor-default)', 
              lineHeight: '1.5',
              margin: '0 0 16px 0'
            }}>
              {migrationDialog.data.isDryRun
                ? 'This will simulate the migration process without making any actual changes to the repository.'
                : 'This will begin the migration process for this repository.'}
            </p>
            {!migrationDialog.data.isDryRun && (
              <div style={{
                display: 'flex',
                gap: '12px',
                padding: '12px',
                borderRadius: '6px',
                backgroundColor: 'var(--bgColor-attention-muted)',
                border: '1px solid var(--borderColor-attention-emphasis)'
              }}>
                <div style={{ flexShrink: 0, marginTop: '2px', color: 'var(--fgColor-attention)' }}>
                  <AlertIcon size={16} />
                </div>
                <div>
                  <p style={{ 
                    fontSize: '14px', 
                    fontWeight: 600,
                    color: 'var(--fgColor-attention)',
                    margin: '0 0 4px 0'
                  }}>
                    This is a permanent action
                  </p>
                  <p style={{ 
                    fontSize: '13px', 
                    color: 'var(--fgColor-default)',
                    lineHeight: '1.5',
                    margin: 0
                  }}>
                    Make sure you have reviewed the migration readiness assessment and have a backup if needed.
                  </p>
                </div>
              </div>
            )}
          </div>
          <div style={{ 
            padding: '16px 24px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px',
            backgroundColor: 'var(--bgColor-muted)'
          }}>
            <Button onClick={migrationDialog.close}>
              Cancel
            </Button>
            <Button variant="primary" onClick={confirmStartMigration}>
              {migrationDialog.data.isDryRun ? 'Start Dry Run' : 'Start Migration'}
            </Button>
          </div>
        </Dialog>
      )}
    </div>
  );
}
