import { useState } from 'react';
import { useParams, Link, useLocation } from 'react-router-dom';
import { Button, UnderlineNav, Textarea, FormControl } from '@primer/react';
import { CalendarIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { MigrationReadinessTab } from './MigrationReadinessTab';
import { TechnicalProfileTab } from './TechnicalProfileTab';
import { DependenciesTab } from './DependenciesTab';
import { ActivityLogTab } from './ActivityLogTab';
import { useRepository, useBatches } from '../../hooks/useQueries';
import { useRediscoverRepository, useUnlockRepository, useRollbackRepository, useMarkRepositoryWontMigrate } from '../../hooks/useMutations';
import { useToast } from '../../contexts/ToastContext';

export function RepositoryDetail() {
  const { fullName } = useParams<{ fullName: string }>();
  const location = useLocation();
  const locationState = location.state as { fromBatch?: boolean; batchId?: number; batchName?: string } | null;
  const { data, isLoading, isFetching } = useRepository(fullName || '');
  const repository: Repository | undefined = data;
  const { data: allBatches = [] } = useBatches();
  const rediscoverMutation = useRediscoverRepository();
  const unlockMutation = useUnlockRepository();
  const rollbackMutation = useRollbackRepository();
  const markWontMigrateMutation = useMarkRepositoryWontMigrate();
  const { showSuccess, showError } = useToast();
  
  const [migrating, setMigrating] = useState(false);
  const [activeTab, setActiveTab] = useState<'readiness' | 'profile' | 'relationships' | 'activity'>('readiness');
  
  // Rollback state
  const [showRollbackDialog, setShowRollbackDialog] = useState(false);
  const [rollbackReason, setRollbackReason] = useState('');

  const handleStartMigration = async (dryRun: boolean = false) => {
    if (!repository || migrating) return;

    setMigrating(true);
    try {
      await api.startMigration({
        repository_ids: [repository.id],
        dry_run: dryRun,
      });
      
      // Show success message
      showSuccess(`${dryRun ? 'Dry run' : 'Migration'} started successfully!`);
    } catch (error: any) {
      console.error('Failed to start migration:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to start migration. Please try again.';
      showError(errorMessage);
    } finally {
      setMigrating(false);
    }
  };

  const handleRediscover = async () => {
    if (!repository || !fullName || rediscoverMutation.isPending) return;

    if (!confirm('Are you sure you want to re-discover this repository? This will update all repository data.')) {
      return;
    }

    try {
      await rediscoverMutation.mutateAsync(decodeURIComponent(fullName));
      showSuccess('Re-discovery started! Repository data will be updated shortly.');
    } catch (error: any) {
      console.error('Failed to start re-discovery:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to start re-discovery. Please try again.';
      showError(errorMessage);
    }
  };

  const handleUnlock = async () => {
    if (!repository || !fullName || unlockMutation.isPending) return;

    if (!confirm('Are you sure you want to unlock this repository? This will remove the lock from the source repository.')) {
      return;
    }

    try {
      await unlockMutation.mutateAsync(decodeURIComponent(fullName));
      showSuccess('Repository unlocked successfully!');
    } catch (error: any) {
      console.error('Failed to unlock repository:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to unlock repository. Please try again.';
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
      setShowRollbackDialog(false);
      setRollbackReason('');
      showSuccess('Repository rolled back successfully! It can now be migrated again.');
    } catch (error: any) {
      console.error('Failed to rollback repository:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to rollback repository. Please try again.';
      showError(errorMessage);
    }
  };

  const handleToggleWontMigrate = async () => {
    if (!repository || !fullName || markWontMigrateMutation.isPending) return;

    const isWontMigrate = repository.status === 'wont_migrate';
    const action = isWontMigrate ? 'unmark' : 'mark as won\'t migrate';
    const confirmMsg = isWontMigrate
      ? 'Are you sure you want to unmark this repository? It will be changed to pending status.'
      : 'Are you sure you want to mark this repository as won\'t migrate? It will be excluded from migration progress and cannot be added to batches.';

    if (!confirm(confirmMsg)) {
      return;
    }

    try {
      await markWontMigrateMutation.mutateAsync({ 
        fullName: decodeURIComponent(fullName), 
        unmark: isWontMigrate 
      });
      showSuccess(`Repository ${action}ed successfully!`);
    } catch (error: any) {
      console.error(`Failed to ${action} repository:`, error);
      const errorMsg = error.response?.data?.error || `Failed to ${action} repository. Please try again.`;
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
    <div className="max-w-7xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      {/* Header */}
      <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
        {/* Breadcrumbs */}
        <nav aria-label="Breadcrumb" className="mb-4">
          <ol className="flex items-center text-sm">
            <li>
              <Link to="/" className="text-blue-600 hover:underline" style={{ color: '#2563eb' }}>Dashboard</Link>
            </li>
            <li className="text-gh-text-muted mx-2">/</li>
            {locationState?.fromBatch && locationState?.batchId ? (
              <>
                <li>
                  <Link to="/batches" className="text-blue-600 hover:underline" style={{ color: '#2563eb' }}>Batches</Link>
                </li>
                <li className="text-gh-text-muted mx-2">/</li>
                <li className="font-semibold text-gh-text-primary">
                  {locationState.batchName || `Batch #${locationState.batchId}`}
                </li>
              </>
            ) : repository.ado_project ? (
              <>
                <li>
                  <Link 
                    to={`/org/${encodeURIComponent(repository.full_name.split('/')[0])}`}
                    className="text-blue-600 hover:underline"
                    style={{ color: '#2563eb' }}
                  >
                    {repository.full_name.split('/')[0]}
                  </Link>
                </li>
                <li className="text-gh-text-muted mx-2">/</li>
                <li>
                  <Link 
                    to={`/org/${encodeURIComponent(repository.full_name.split('/')[0])}/project/${encodeURIComponent(repository.ado_project)}`}
                    className="text-blue-600 hover:underline"
                    style={{ color: '#2563eb' }}
                  >
                    {repository.ado_project}
                  </Link>
                </li>
                <li className="text-gh-text-muted mx-2">/</li>
                <li className="font-semibold text-gh-text-primary">
                  {repository.full_name.split('/').slice(1).join('/')}
                </li>
              </>
            ) : (
              <>
                <li>
                  <Link 
                    to={`/org/${encodeURIComponent(repository.full_name.split('/')[0])}`}
                    className="text-blue-600 hover:underline"
                    style={{ color: '#2563eb' }}
                  >
                    {repository.full_name.split('/')[0]}
                  </Link>
                </li>
                <li className="text-gh-text-muted mx-2">/</li>
                <li className="font-semibold text-gh-text-primary">
                  {repository.full_name.split('/').slice(1).join('/')}
                </li>
              </>
            )}
          </ol>
        </nav>
        <div className="flex justify-between items-start">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-3xl font-light text-gray-900">
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
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                      ⚠️ Validation Failed
                    </span>
                  );
                } else if (hasWarnings) {
                  return (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                      ⚠ Has Warnings
                    </span>
                  );
                } else {
                  return (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
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
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gh-text-secondary mb-4 pb-4 border-b border-gh-border-default">
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
                  <span className="text-gh-text-secondary">•</span>
                  <TimestampDisplay 
                    timestamp={repository.last_discovery_at} 
                    label="Data refreshed"
                    size="sm"
                  />
                </div>
              )}
              {repository.last_dry_run_at && (
                <div className="flex items-center gap-1">
                  <span className="text-gh-text-secondary">•</span>
                  <TimestampDisplay 
                    timestamp={repository.last_dry_run_at} 
                    label="Dry run"
                    size="sm"
                  />
                </div>
              )}
              {repository.migrated_at && (
                <div className="flex items-center gap-1">
                  <span className="text-gh-text-secondary">•</span>
                  <TimestampDisplay 
                    timestamp={repository.migrated_at} 
                    label="Migrated"
                    size="sm"
                  />
                </div>
              )}
            </div>
          </div>

          {/* Migration Actions */}
          <div className="flex flex-col gap-3 ml-6">
            <Button
              onClick={handleRediscover}
              disabled={rediscoverMutation.isPending}
              variant="default"
              style={{ whiteSpace: 'nowrap' }}
            >
              {rediscoverMutation.isPending ? 'Re-discovering...' : 'Re-discover'}
            </Button>
            
            {/* Won't Migrate Toggle */}
            {!isInActiveMigration && repository.status !== 'complete' && (
              <button
                onClick={handleToggleWontMigrate}
                disabled={markWontMigrateMutation.isPending}
                className={`px-4 py-2 rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap ${
                  repository.status === 'wont_migrate'
                    ? 'bg-blue-600 text-white hover:bg-blue-700'
                    : 'bg-gray-100 text-gray-700 border border-gray-300 hover:bg-gray-200'
                }`}
              >
                {markWontMigrateMutation.isPending 
                  ? 'Processing...' 
                  : repository.status === 'wont_migrate' 
                    ? 'Unmark Won\'t Migrate'
                    : 'Mark as Won\'t Migrate'}
              </button>
            )}
            
            {canMigrate && repository.status !== 'migration_failed' && repository.status !== 'dry_run_failed' && (
              <>
                <button
                  onClick={() => handleStartMigration(true)}
                  disabled={migrating}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {migrating ? 'Processing...' : 'Dry Run'}
                </button>
                <button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  className="px-4 py-2 bg-green-600 text-white rounded-md text-sm font-medium hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
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
                onClick={() => setShowRollbackDialog(true)}
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
            className="text-blue-600 hover:underline font-medium"
            style={{ color: '#2563eb' }}
          >
            View Source Repository →
          </a>
          {repository.destination_url && (
            <a
              href={repository.destination_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-green-600 hover:underline font-medium"
              style={{ color: '#16a34a' }}
            >
              View Migrated Repository →
            </a>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white rounded-lg shadow-sm mb-6">
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
      {showRollbackDialog && (
        <>
          {/* Backdrop */}
          <div 
            className="fixed inset-0 bg-black/50 z-50"
            onClick={() => {
              setShowRollbackDialog(false);
              setRollbackReason('');
            }}
          />
          
          {/* Dialog */}
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div 
              className="bg-white rounded-lg shadow-xl max-w-md w-full"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="px-4 py-3 border-b border-gh-border-default">
                <h3 className="text-base font-semibold text-gh-text-primary">
                  Rollback Migration
                </h3>
              </div>
              
              <div className="p-4">
                <p className="text-sm text-gh-text-secondary mb-4">
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
              
              <div className="px-4 py-3 border-t border-gh-border-default flex justify-end gap-2">
                <Button
                  onClick={() => {
                    setShowRollbackDialog(false);
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
            </div>
          </div>
        </>
      )}
    </div>
  );
}
