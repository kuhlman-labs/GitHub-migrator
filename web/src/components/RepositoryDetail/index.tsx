import { useState } from 'react';
import { useParams, Link, useLocation } from 'react-router-dom';
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
      alert(`${dryRun ? 'Dry run' : 'Migration'} started successfully!`);
    } catch (error: any) {
      console.error('Failed to start migration:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to start migration. Please try again.';
      alert(errorMessage);
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
      alert('Re-discovery started! Repository data will be updated shortly.');
    } catch (error: any) {
      console.error('Failed to start re-discovery:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to start re-discovery. Please try again.';
      alert(errorMessage);
    }
  };

  const handleUnlock = async () => {
    if (!repository || !fullName || unlockMutation.isPending) return;

    if (!confirm('Are you sure you want to unlock this repository? This will remove the lock from the source repository.')) {
      return;
    }

    try {
      await unlockMutation.mutateAsync(decodeURIComponent(fullName));
      alert('Repository unlocked successfully!');
    } catch (error: any) {
      console.error('Failed to unlock repository:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to unlock repository. Please try again.';
      alert(errorMessage);
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
      alert('Repository rolled back successfully! It can now be migrated again.');
    } catch (error: any) {
      console.error('Failed to rollback repository:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to rollback repository. Please try again.';
      alert(errorMessage);
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
      alert(`Repository ${action}ed successfully!`);
    } catch (error: any) {
      console.error(`Failed to ${action} repository:`, error);
      const errorMsg = error.response?.data?.error || `Failed to ${action} repository. Please try again.`;
      alert(errorMsg);
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
    <div className="max-w-6xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      {/* Header */}
      <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
        <div className="mb-4">
          {locationState?.fromBatch && locationState?.batchId ? (
            <Link 
              to="/batches"
              state={{ selectedBatchId: locationState.batchId }}
              className="text-blue-600 hover:text-blue-800 text-sm flex items-center gap-1"
            >
              ← Back to Batch {locationState.batchName ? `"${locationState.batchName}"` : `#${locationState.batchId}`}
            </Link>
          ) : (
            <Link 
              to={
                // For ADO repos, navigate to /org/{organization}/project/{project}
                repository.ado_project ? 
                  `/org/${encodeURIComponent(repository.full_name.split('/')[0])}/project/${encodeURIComponent(repository.ado_project)}` :
                  // For GitHub repos, navigate to /org/{organization}
                  `/org/${encodeURIComponent(repository.full_name.split('/')[0])}`
              }
              className="text-blue-600 hover:text-blue-800 text-sm flex items-center gap-1"
            >
              ← Back to Repositories
            </Link>
          )}
        </div>
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
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
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
            <button
              onClick={handleRediscover}
              disabled={rediscoverMutation.isPending}
              className="px-4 py-2 border border-gh-border-default text-gh-text-primary rounded-md text-sm font-medium hover:bg-gh-neutral-bg disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
            >
              {rediscoverMutation.isPending ? 'Re-discovering...' : 'Re-discover'}
            </button>
            
            {/* Won't Migrate Toggle */}
            {!isInActiveMigration && repository.status !== 'complete' && (
              <button
                onClick={handleToggleWontMigrate}
                disabled={markWontMigrateMutation.isPending}
                className={`px-4 py-2 border rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap ${
                  repository.status === 'wont_migrate'
                    ? 'border-blue-500 text-blue-600 hover:bg-blue-50'
                    : 'border-gray-500 text-gray-700 hover:bg-gray-50'
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
                  className="px-4 py-2 border border-gh-border-default rounded-md text-sm font-medium text-gh-text-primary hover:bg-gh-neutral-bg disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {migrating ? 'Processing...' : 'Dry Run'}
                </button>
                <button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  className="px-4 py-2 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {migrating ? 'Processing...' : 'Start Migration'}
                </button>
              </>
            )}
            {repository.status === 'dry_run_failed' && (
              <>
                <button
                  onClick={() => handleStartMigration(true)}
                  disabled={migrating}
                  className="px-4 py-2 bg-gh-warning text-white rounded-md text-sm font-medium hover:bg-gh-warning-emphasis disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {migrating ? 'Re-running...' : 'Re-run Dry Run'}
                </button>
                <button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  className="px-4 py-2 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {migrating ? 'Starting...' : 'Start Migration Anyway'}
                </button>
              </>
            )}
            {repository.status === 'migration_failed' && (
              <>
                <button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  className="px-4 py-2 bg-gh-warning text-white rounded-md text-sm font-medium hover:bg-gh-warning-emphasis disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  Retry Migration
                </button>
                {repository.is_source_locked && repository.source_migration_id && (
                  <button
                    onClick={handleUnlock}
                    disabled={unlockMutation.isPending}
                    className="px-4 py-2 bg-orange-600 text-white rounded-md text-sm font-medium hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                  >
                    {unlockMutation.isPending ? 'Unlocking...' : '🔓 Unlock Source'}
                  </button>
                )}
              </>
            )}
            {repository.status === 'complete' && (
              <button
                onClick={() => setShowRollbackDialog(true)}
                disabled={rollbackMutation.isPending}
                className="px-4 py-2 bg-orange-600 text-white rounded-md text-sm font-medium hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
              >
                {rollbackMutation.isPending ? 'Rolling back...' : 'Rollback Migration'}
              </button>
            )}
          </div>
        </div>

        {/* Links */}
        <div className="mt-4 flex gap-4 text-sm">
          <a
            href={repository.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-600 hover:underline"
          >
            View Source Repository →
          </a>
          {repository.destination_url && (
            <a
              href={repository.destination_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-green-600 hover:underline"
            >
              View Migrated Repository →
            </a>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white rounded-lg shadow-sm mb-6">
        <div className="border-b border-gray-200">
          <nav className="flex -mb-px">
            {[
              { id: 'readiness' as const, label: 'Migration Readiness' },
              { id: 'profile' as const, label: 'Technical Profile' },
              { id: 'relationships' as const, label: 'Relationships' },
              { id: 'activity' as const, label: 'Activity Log' }
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`px-6 py-4 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.id
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>
        </div>

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
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Rollback Migration</h3>
            <p className="text-sm text-gray-600 mb-4">
              This will mark the repository as rolled back and allow it to be migrated again in the future.
              You can optionally provide a reason for the rollback.
            </p>
            
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Reason (optional)
              </label>
              <textarea
                value={rollbackReason}
                onChange={(e) => setRollbackReason(e.target.value)}
                placeholder="e.g., CI/CD integration issues, workflow failures..."
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                rows={3}
                disabled={rollbackMutation.isPending}
              />
            </div>

            <div className="flex justify-end gap-3">
              <button
                onClick={() => {
                  setShowRollbackDialog(false);
                  setRollbackReason('');
                }}
                disabled={rollbackMutation.isPending}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleRollback}
                disabled={rollbackMutation.isPending}
                className="px-4 py-2 bg-orange-600 text-white rounded-md text-sm font-medium hover:bg-orange-700 disabled:opacity-50"
              >
                {rollbackMutation.isPending ? 'Rolling back...' : 'Confirm Rollback'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
