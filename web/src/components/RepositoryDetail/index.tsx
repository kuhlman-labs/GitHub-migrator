import { useEffect, useState } from 'react';
import { useParams, Link, useLocation } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import type { Repository, MigrationHistory, MigrationLog } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { ProfileCard } from '../common/ProfileCard';
import { ProfileItem } from '../common/ProfileItem';
import { ComplexityInfoModal } from '../common/ComplexityInfoModal';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { MigrationReadinessSection } from './MigrationReadinessSection';
import { DependenciesTab } from './DependenciesTab';
import { formatBytes, formatDate } from '../../utils/format';
import { useRepository, useBatches } from '../../hooks/useQueries';
import { useRediscoverRepository, useUpdateRepository, useUnlockRepository, useRollbackRepository, useMarkRepositoryWontMigrate } from '../../hooks/useMutations';

export function RepositoryDetail() {
  const { fullName } = useParams<{ fullName: string }>();
  const location = useLocation();
  const queryClient = useQueryClient();
  const locationState = location.state as { fromBatch?: boolean; batchId?: number; batchName?: string } | null;
  const { data, isLoading, isFetching } = useRepository(fullName || '');
  const repository: Repository | undefined = data;
  const { data: allBatches = [] } = useBatches();
  const rediscoverMutation = useRediscoverRepository();
  const updateRepositoryMutation = useUpdateRepository();
  const unlockMutation = useUnlockRepository();
  const rollbackMutation = useRollbackRepository();
  const markWontMigrateMutation = useMarkRepositoryWontMigrate();
  
  const [history, setHistory] = useState<MigrationHistory[]>([]);
  const [logs, setLogs] = useState<MigrationLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [migrating, setMigrating] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'history' | 'logs' | 'dependencies'>('overview');
  
  // Rollback state
  const [showRollbackDialog, setShowRollbackDialog] = useState(false);
  const [rollbackReason, setRollbackReason] = useState('');
  
  // Batch assignment state - show pending and ready batches
  const batches = allBatches.filter(b => b.status === 'pending' || b.status === 'ready');
  const [selectedBatchId, setSelectedBatchId] = useState<number | null>(null);
  const [assigningBatch, setAssigningBatch] = useState(false);
  
  // Destination configuration
  const [editingDestination, setEditingDestination] = useState(false);
  const [destinationFullName, setDestinationFullName] = useState<string>('');
  
  // Log filters
  const [logLevel, setLogLevel] = useState<string>('');
  const [logPhase, setLogPhase] = useState<string>('');
  const [logSearch, setLogSearch] = useState<string>('');

  useEffect(() => {
    if (repository) {
      // Use destination_full_name if set, otherwise default to source full_name
      setDestinationFullName(repository.destination_full_name || repository.full_name);
    }
    
    // Load migration history when repository changes
    if (repository?.id) {
      (async () => {
        try {
          const response = await api.getMigrationHistory(repository.id);
          setHistory(response || []);
        } catch (error) {
          console.error('Failed to load migration history:', error);
        }
      })();
    }
  }, [repository]);

  const loadLogs = async (repoId?: number) => {
    const id = repoId || repository?.id;
    if (!id) return;
    
    setLogsLoading(true);
    try {
      const response = await api.getMigrationLogs(id, {
        level: logLevel || undefined,
        phase: logPhase || undefined,
        limit: 500,
      });
      setLogs(response.logs || []);
    } catch (error) {
      console.error('Failed to load logs:', error);
    } finally {
      setLogsLoading(false);
    }
  };

  // Load logs when filters change
  useEffect(() => {
    if (activeTab === 'logs' && repository) {
      loadLogs();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [logLevel, logPhase, activeTab]);

  const handleSaveDestination = async () => {
    if (!repository || !fullName) return;

    // Validate format
    if (!destinationFullName.includes('/')) {
      alert('Destination must be in "organization/repository" format');
      return;
    }

    try {
      await updateRepositoryMutation.mutateAsync({
        fullName: decodeURIComponent(fullName),
        updates: { destination_full_name: destinationFullName },
      });
      
      setEditingDestination(false);
    } catch (error) {
      console.error('Failed to save destination:', error);
      alert('Failed to save destination. Please try again.');
    }
  };

  const handleStartMigration = async (dryRun: boolean = false) => {
    if (!repository || migrating) return;

    // Save destination first if it was changed
    if (editingDestination && destinationFullName !== repository.destination_full_name) {
      await handleSaveDestination();
    }

    setMigrating(true);
    try {
      await api.startMigration({
        repository_ids: [repository.id],
        dry_run: dryRun,
      });
      
      // Show success message
      alert(`${dryRun ? 'Dry run' : 'Migration'} started successfully!`);
    } catch (error) {
      console.error('Failed to start migration:', error);
      alert('Failed to start migration. Please try again.');
    } finally {
      setMigrating(false);
    }
  };

  const handleAssignToBatch = async () => {
    if (!repository || !selectedBatchId || assigningBatch || !fullName) return;

    setAssigningBatch(true);
    try {
      await api.addRepositoriesToBatch(selectedBatchId, [repository.id]);
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({ queryKey: ['repository', decodeURIComponent(fullName)] });
      await queryClient.invalidateQueries({ queryKey: ['batches'] });
      
      alert('Repository assigned to batch successfully!');
      setSelectedBatchId(null);
    } catch (error: any) {
      console.error('Failed to assign to batch:', error);
      const errorMsg = error.response?.data?.error || 'Failed to assign to batch. Please try again.';
      alert(errorMsg);
    } finally {
      setAssigningBatch(false);
    }
  };

  const handleRemoveFromBatch = async () => {
    if (!repository || !repository.batch_id || assigningBatch || !fullName) return;

    if (!confirm('Are you sure you want to remove this repository from its batch?')) {
      return;
    }

    setAssigningBatch(true);
    try {
      await api.removeRepositoriesFromBatch(repository.batch_id, [repository.id]);
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({ queryKey: ['repository', decodeURIComponent(fullName)] });
      await queryClient.invalidateQueries({ queryKey: ['batches'] });
      
      alert('Repository removed from batch successfully!');
    } catch (error: any) {
      console.error('Failed to remove from batch:', error);
      const errorMsg = error.response?.data?.error || 'Failed to remove from batch. Please try again.';
      alert(errorMsg);
    } finally {
      setAssigningBatch(false);
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
    } catch (error) {
      console.error('Failed to start re-discovery:', error);
      alert('Failed to start re-discovery. Please try again.');
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
    } catch (error) {
      console.error('Failed to unlock repository:', error);
      alert('Failed to unlock repository. Please try again.');
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
    } catch (error) {
      console.error('Failed to rollback repository:', error);
      alert('Failed to rollback repository. Please try again.');
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

  const canChangeBatch = !isInActiveMigration && repository.status !== 'complete';
  
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
              to={`/org/${encodeURIComponent(repository.full_name.split('/')[0])}`}
              className="text-blue-600 hover:text-blue-800 text-sm flex items-center gap-1"
            >
              ← Back to Repositories
            </Link>
          )}
        </div>
        <div className="flex justify-between items-start">
          <div className="flex-1">
            <h1 className="text-3xl font-light text-gray-900 mb-2">
              {repository.full_name}
            </h1>
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

            {/* Destination Configuration */}
            {canMigrate && (
              <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Destination (where to migrate)
                    </label>
                    {editingDestination ? (
                      <div className="flex items-center gap-2">
                        <input
                          type="text"
                          value={destinationFullName}
                          onChange={(e) => setDestinationFullName(e.target.value)}
                          placeholder="org/repo"
                          className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          disabled={updateRepositoryMutation.isPending}
                        />
                        <button
                          onClick={handleSaveDestination}
                          disabled={updateRepositoryMutation.isPending}
                          className="px-3 py-1.5 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50"
                        >
                          {updateRepositoryMutation.isPending ? 'Saving...' : 'Save'}
                        </button>
                        <button
                          onClick={() => {
                            setEditingDestination(false);
                            setDestinationFullName(repository.destination_full_name || repository.full_name);
                          }}
                          disabled={updateRepositoryMutation.isPending}
                          className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2">
                        <code className="flex-1 px-3 py-2 bg-white border border-gray-200 rounded-md text-sm text-gray-900">
                          {destinationFullName}
                        </code>
                        <button
                          onClick={() => setEditingDestination(true)}
                          className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
                        >
                          Edit
                        </button>
                      </div>
                    )}
                    <p className="mt-1 text-xs text-gray-500">
                      {destinationFullName === repository.full_name 
                        ? 'Using same organization as source (default)' 
                        : 'Using custom destination organization'}
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Batch Assignment */}
            {canChangeBatch && (
              <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Batch Assignment
                </label>
                {repository.batch_id ? (
                  <div className="flex items-center gap-2">
                    <div className="flex-1 px-3 py-2 bg-white border border-gray-200 rounded-md text-sm">
                      <Badge color="blue">{currentBatch?.name || `Batch #${repository.batch_id}`}</Badge>
                    </div>
                    <button
                      onClick={handleRemoveFromBatch}
                      disabled={assigningBatch}
                      className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
                    >
                      {assigningBatch ? 'Removing...' : 'Remove from Batch'}
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <select
                      value={selectedBatchId || ''}
                      onChange={(e) => setSelectedBatchId(e.target.value ? Number(e.target.value) : null)}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      disabled={assigningBatch}
                    >
                      <option value="">Select a batch...</option>
                      {batches.map((batch) => (
                        <option key={batch.id} value={batch.id}>
                          {batch.name} ({batch.type}) - {batch.status} - {batch.repository_count} repos
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={handleAssignToBatch}
                      disabled={!selectedBatchId || assigningBatch}
                      className="px-3 py-1.5 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50"
                    >
                      {assigningBatch ? 'Assigning...' : 'Assign to Batch'}
                    </button>
                  </div>
                )}
                <p className="mt-1 text-xs text-gray-500">
                  {repository.batch_id
                    ? 'Repository is assigned to a batch'
                    : batches.length === 0
                    ? 'No pending or ready batches available. Create a batch first.'
                    : 'Assign this repository to a batch for grouped migration'}
                </p>
              </div>
            )}
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

      {/* Migration Readiness - Validation & Configuration */}
      <div className="mb-6">
        <MigrationReadinessSection 
          repository={repository}
          onUpdate={() => queryClient.invalidateQueries({ queryKey: ['repository', fullName] })}
          onRevalidate={() => queryClient.invalidateQueries({ queryKey: ['repository', fullName] })}
        />
      </div>

      {/* Tabs */}
      <div className="bg-white rounded-lg shadow-sm mb-6">
        <div className="border-b border-gray-200">
          <nav className="flex -mb-px">
            {(['overview', 'dependencies', 'history', 'logs'] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-6 py-4 text-sm font-medium border-b-2 transition-colors capitalize ${
                  activeTab === tab
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                {tab === 'overview' ? 'Repository Profile' : tab}
              </button>
            ))}
          </nav>
        </div>

        <div className="p-6">
          {activeTab === 'overview' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <ProfileCard title="Migration Complexity Score">
                {(() => {
                  // Use backend-calculated breakdown when available, otherwise fallback to frontend calculation
                  const breakdown = repository.complexity_breakdown;
                  
                  // Determine size tier for display
                  const MB100 = 100 * 1024 * 1024;
                  const GB1 = 1024 * 1024 * 1024;
                  const GB5 = 5 * 1024 * 1024 * 1024;
                  
                  let sizeTier = 'Unknown';
                  if (repository.total_size !== null && repository.total_size !== undefined) {
                    if (repository.total_size >= GB5) {
                      sizeTier = '>5GB';
                    } else if (repository.total_size >= GB1) {
                      sizeTier = '1-5GB';
                    } else if (repository.total_size >= MB100) {
                      sizeTier = '100MB-1GB';
                    } else {
                      sizeTier = '<100MB';
                    }
                  }
                  
                  // If backend provides breakdown, use it; otherwise calculate
                  let sizePoints, largeFilesPoints, environmentsPoints, secretsPoints, packagesPoints, runnersPoints;
                  let variablesPoints, discussionsPoints, releasesPoints, lfsPoints, submodulesPoints, appsPoints, projectsPoints;
                  let securityPoints, webhooksPoints, tagProtectionsPoints, branchProtectionsPoints, rulesetsPoints;
                  let publicVisibilityPoints, internalVisibilityPoints, codeownersPoints, activityPoints;
                  
                  if (breakdown) {
                    // Use backend-calculated breakdown
                    sizePoints = breakdown.size_points;
                    largeFilesPoints = breakdown.large_files_points;
                    environmentsPoints = breakdown.environments_points;
                    secretsPoints = breakdown.secrets_points;
                    packagesPoints = breakdown.packages_points;
                    runnersPoints = breakdown.runners_points;
                    variablesPoints = breakdown.variables_points;
                    discussionsPoints = breakdown.discussions_points;
                    releasesPoints = breakdown.releases_points;
                    lfsPoints = breakdown.lfs_points;
                    submodulesPoints = breakdown.submodules_points;
                    appsPoints = breakdown.apps_points;
                    projectsPoints = breakdown.projects_points;
                    securityPoints = breakdown.security_points;
                    webhooksPoints = breakdown.webhooks_points;
                    tagProtectionsPoints = breakdown.tag_protections_points;
                    branchProtectionsPoints = breakdown.branch_protections_points;
                    rulesetsPoints = breakdown.rulesets_points;
                    publicVisibilityPoints = breakdown.public_visibility_points;
                    internalVisibilityPoints = breakdown.internal_visibility_points;
                    codeownersPoints = breakdown.codeowners_points;
                    activityPoints = breakdown.activity_points;
                  } else {
                    // Fallback: Calculate using frontend logic
                    if (repository.total_size >= GB5) {
                      sizePoints = 9;
                    } else if (repository.total_size >= GB1) {
                      sizePoints = 6;
                    } else if (repository.total_size >= MB100) {
                      sizePoints = 3;
                    } else {
                      sizePoints = 0;
                    }
                    
                    largeFilesPoints = repository.has_large_files ? 4 : 0;
                    environmentsPoints = repository.environment_count > 0 ? 3 : 0;
                    secretsPoints = repository.secret_count > 0 ? 3 : 0;
                    packagesPoints = repository.has_packages ? 3 : 0;
                    runnersPoints = repository.has_self_hosted_runners ? 3 : 0;
                    variablesPoints = repository.variable_count > 0 ? 2 : 0;
                    discussionsPoints = repository.has_discussions ? 2 : 0;
                    releasesPoints = repository.release_count > 0 ? 2 : 0;
                    lfsPoints = repository.has_lfs ? 2 : 0;
                    submodulesPoints = repository.has_submodules ? 2 : 0;
                    appsPoints = repository.installed_apps_count > 0 ? 2 : 0;
                    projectsPoints = repository.has_projects ? 2 : 0;
                    securityPoints = (repository.has_code_scanning || repository.has_dependabot || repository.has_secret_scanning) ? 1 : 0;
                    webhooksPoints = repository.webhook_count > 0 ? 1 : 0;
                    tagProtectionsPoints = repository.tag_protection_count > 0 ? 1 : 0;
                    branchProtectionsPoints = repository.branch_protections > 0 ? 1 : 0;
                    rulesetsPoints = repository.has_rulesets ? 1 : 0;
                    publicVisibilityPoints = repository.visibility === 'public' ? 1 : 0;
                    internalVisibilityPoints = repository.visibility === 'internal' ? 1 : 0;
                    codeownersPoints = repository.has_codeowners ? 1 : 0;
                    
                    // Activity-based scoring (approximated with static thresholds - less accurate than backend)
                    const activityScore = 
                      (repository.branch_count > 50 ? 0.5 : repository.branch_count > 10 ? 0.25 : 0) +
                      (repository.commit_count > 1000 ? 0.5 : repository.commit_count > 100 ? 0.25 : 0) +
                      (repository.issue_count > 100 ? 0.5 : repository.issue_count > 10 ? 0.25 : 0) +
                      (repository.pull_request_count > 50 ? 0.5 : repository.pull_request_count > 10 ? 0.25 : 0);
                    activityPoints = activityScore >= 1.5 ? 4 : activityScore >= 0.5 ? 2 : 0;
                  }
                  
                  // Use backend score when available, otherwise calculate from components
                  const totalPoints = repository.complexity_score ?? (
                    sizePoints + largeFilesPoints + environmentsPoints + secretsPoints + packagesPoints + runnersPoints +
                    variablesPoints + discussionsPoints + releasesPoints + lfsPoints + submodulesPoints + securityPoints + appsPoints + projectsPoints +
                    webhooksPoints + tagProtectionsPoints + branchProtectionsPoints + rulesetsPoints + publicVisibilityPoints + internalVisibilityPoints + codeownersPoints +
                    activityPoints
                  );
                  
                  let category = 'Simple';
                  let categoryColor = 'text-green-600';
                  let categoryBg = 'bg-green-50';
                  if (totalPoints > 17) {
                    category = 'Very Complex';
                    categoryColor = 'text-red-600';
                    categoryBg = 'bg-red-50';
                  } else if (totalPoints > 10) {
                    category = 'Complex';
                    categoryColor = 'text-orange-600';
                    categoryBg = 'bg-orange-50';
                  } else if (totalPoints > 5) {
                    category = 'Medium';
                    categoryColor = 'text-yellow-600';
                    categoryBg = 'bg-yellow-50';
                  }
                  
                  return (
                    <>
                      <div className={`mb-4 p-4 ${categoryBg} rounded-lg border-l-4 ${categoryColor.replace('text-', 'border-')}`}>
                        <div className="flex justify-between items-center mb-2">
                          <span className="text-sm font-medium text-gray-700">Total Complexity Score</span>
                          <span className={`text-3xl font-bold ${categoryColor}`}>{totalPoints}</span>
                        </div>
                        <div className="text-sm">
                          <span className="font-medium text-gray-700">Category: </span>
                          <span className={`font-semibold ${categoryColor}`}>{category}</span>
                        </div>
                      </div>
                      
                      <div className="space-y-3 mb-4">
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Repository Size</div>
                            <div className="text-xs text-gray-500">{formatBytes(repository.total_size)} ({sizeTier})</div>
                          </div>
                          <span className={`text-lg font-semibold ${sizePoints > 0 ? 'text-blue-600' : 'text-gray-400'}`}>
                            +{sizePoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Large Files (&gt;100MB)</div>
                            <div className="text-xs text-gray-500">
                              {repository.has_large_files ? `Yes (${repository.large_file_count}+ detected)` : 'No'}
                            </div>
                          </div>
                          <span className={`text-lg font-semibold ${largeFilesPoints > 0 ? 'text-red-600' : 'text-gray-400'}`}>
                            +{largeFilesPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Git LFS</div>
                            <div className="text-xs text-gray-500">{repository.has_lfs ? 'Yes' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${lfsPoints > 0 ? 'text-orange-600' : 'text-gray-400'}`}>
                            +{lfsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Submodules</div>
                            <div className="text-xs text-gray-500">{repository.has_submodules ? 'Yes' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${submodulesPoints > 0 ? 'text-orange-600' : 'text-gray-400'}`}>
                            +{submodulesPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">GitHub Packages</div>
                            <div className="text-xs text-gray-500">{repository.has_packages ? 'Yes (requires manual migration)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${packagesPoints > 0 ? 'text-red-600' : 'text-gray-400'}`}>
                            +{packagesPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Environments</div>
                            <div className="text-xs text-gray-500">{repository.environment_count > 0 ? `${repository.environment_count} (manual recreation required)` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${environmentsPoints > 0 ? 'text-red-600' : 'text-gray-400'}`}>
                            +{environmentsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Secrets</div>
                            <div className="text-xs text-gray-500">{repository.secret_count > 0 ? `${repository.secret_count} (manual recreation required)` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${secretsPoints > 0 ? 'text-red-600' : 'text-gray-400'}`}>
                            +{secretsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Variables</div>
                            <div className="text-xs text-gray-500">{repository.variable_count > 0 ? `${repository.variable_count} (manual recreation required)` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${variablesPoints > 0 ? 'text-orange-600' : 'text-gray-400'}`}>
                            +{variablesPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Discussions</div>
                            <div className="text-xs text-gray-500">{repository.has_discussions ? 'Yes (don\'t migrate)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${discussionsPoints > 0 ? 'text-orange-600' : 'text-gray-400'}`}>
                            +{discussionsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Releases</div>
                            <div className="text-xs text-gray-500">{repository.release_count > 0 ? `${repository.release_count} (GHES 3.5.0+ only)` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${releasesPoints > 0 ? 'text-orange-600' : 'text-gray-400'}`}>
                            +{releasesPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Webhooks</div>
                            <div className="text-xs text-gray-500">{repository.webhook_count > 0 ? `${repository.webhook_count} (must re-enable)` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${webhooksPoints > 0 ? 'text-yellow-600' : 'text-gray-400'}`}>
                            +{webhooksPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Tag Protections</div>
                            <div className="text-xs text-gray-500">{repository.tag_protection_count > 0 ? `${repository.tag_protection_count} rules` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${tagProtectionsPoints > 0 ? 'text-yellow-600' : 'text-gray-400'}`}>
                            +{tagProtectionsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Branch Protections</div>
                            <div className="text-xs text-gray-500">{repository.branch_protections > 0 ? `${repository.branch_protections} rules` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${branchProtectionsPoints > 0 ? 'text-yellow-600' : 'text-gray-400'}`}>
                            +{branchProtectionsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Rulesets</div>
                            <div className="text-xs text-gray-500">{repository.has_rulesets ? 'Yes (requires manual recreation)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${rulesetsPoints > 0 ? 'text-yellow-600' : 'text-gray-400'}`}>
                            +{rulesetsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Advanced Security (GHAS)</div>
                            <div className="text-xs text-gray-500">
                              {repository.has_code_scanning || repository.has_dependabot || repository.has_secret_scanning 
                                ? 'Enabled (simple re-enablement)' 
                                : 'Not enabled'}
                            </div>
                          </div>
                          <span className={`text-lg font-semibold ${securityPoints > 0 ? 'text-yellow-600' : 'text-gray-400'}`}>
                            +{securityPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Self-Hosted Runners</div>
                            <div className="text-xs text-gray-500">{repository.has_self_hosted_runners ? 'Yes (infrastructure dependency)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${runnersPoints > 0 ? 'text-red-600' : 'text-gray-400'}`}>
                            +{runnersPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">GitHub Apps</div>
                            <div className="text-xs text-gray-500">{repository.installed_apps_count > 0 ? `${repository.installed_apps_count} installed` : 'None'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${appsPoints > 0 ? 'text-orange-600' : 'text-gray-400'}`}>
                            +{appsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">ProjectsV2</div>
                            <div className="text-xs text-gray-500">{repository.has_projects ? 'Yes (don\'t migrate, manual recreation required)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${projectsPoints > 0 ? 'text-orange-600' : 'text-gray-400'}`}>
                            +{projectsPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Public Visibility</div>
                            <div className="text-xs text-gray-500">{repository.visibility === 'public' ? 'Yes (transformation may be required)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${publicVisibilityPoints > 0 ? 'text-blue-600' : 'text-gray-400'}`}>
                            +{publicVisibilityPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Internal Visibility</div>
                            <div className="text-xs text-gray-500">{repository.visibility === 'internal' ? 'Yes (transformation may be required)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${internalVisibilityPoints > 0 ? 'text-yellow-600' : 'text-gray-400'}`}>
                            +{internalVisibilityPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2 border-b border-gray-200">
                          <div>
                            <div className="text-sm font-medium text-gray-900">CODEOWNERS</div>
                            <div className="text-xs text-gray-500">{repository.has_codeowners ? 'Yes (requires verification)' : 'No'}</div>
                          </div>
                          <span className={`text-lg font-semibold ${codeownersPoints > 0 ? 'text-yellow-600' : 'text-gray-400'}`}>
                            +{codeownersPoints}
                          </span>
                        </div>
                        
                        <div className="flex justify-between items-center py-2">
                          <div>
                            <div className="text-sm font-medium text-gray-900">Activity Level</div>
                            <div className="text-xs text-gray-500">
                              {activityPoints === 4 ? 'High (top 25% - extensive coordination needed)' : 
                               activityPoints === 2 ? 'Moderate (25-75% - some coordination needed)' : 
                               'Low (bottom 25%)'}
                              {breakdown && (
                                <span className="ml-1 text-blue-600">✓ Quantile-based</span>
                              )}
                            </div>
                          </div>
                          <span className={`text-lg font-semibold ${activityPoints > 0 ? 'text-purple-600' : 'text-gray-400'}`}>
                            +{activityPoints}
                          </span>
                        </div>
                      </div>
                      
                      <div className="mt-4 p-3 bg-blue-50 rounded text-xs text-blue-700">
                        <p className="font-medium mb-1">💡 Scoring based on GitHub migration documentation</p>
                        <p>Weights reflect remediation difficulty for features that don't migrate automatically.</p>
                        {breakdown && (
                          <p className="mt-1">Activity level uses percentile-based calculation across all repositories for accurate comparison.</p>
                        )}
                      </div>
                      
                      <div className="pt-2 border-t border-gray-200">
                        <ComplexityInfoModal />
                      </div>
                    </>
                  );
                })()}
              </ProfileCard>

              <ProfileCard title="GitHub Features">
                <ProfileItem label="Archived" value={repository.is_archived ? 'Yes' : 'No'} />
                <ProfileItem label="Fork" value={repository.is_fork ? 'Yes' : 'No'} />
                <ProfileItem label="Wikis" value={repository.has_wiki ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Pages" value={repository.has_pages ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Discussions" value={repository.has_discussions ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Actions" value={repository.has_actions ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Projects" value={repository.has_projects ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Packages" value={repository.has_packages ? 'Yes' : 'No'} />
                <ProfileItem label="Branch Protections" value={repository.branch_protections} />
                <ProfileItem label="Rulesets" value={repository.has_rulesets ? 'Yes' : 'No'} />
                <ProfileItem label="Environments" value={repository.environment_count} />
                <ProfileItem label="Secrets" value={repository.secret_count} />
                <ProfileItem label="Webhooks" value={repository.webhook_count} />
                <ProfileItem label="Visibility" value={repository.visibility} />
                <ProfileItem label="Workflows" value={repository.workflow_count} />
                <ProfileItem label="Code Scanning" value={repository.has_code_scanning ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Dependabot" value={repository.has_dependabot ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Secret Scanning" value={repository.has_secret_scanning ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="CODEOWNERS" value={repository.has_codeowners ? 'Yes' : 'No'} />
                <ProfileItem label="Self-Hosted Runners" value={repository.has_self_hosted_runners ? 'Yes' : 'No'} />
                <ProfileItem label="Outside Collaborators" value={repository.collaborator_count} />
                <ProfileItem label="GitHub Apps" value={repository.installed_apps_count} />
                <ProfileItem label="Releases" value={repository.release_count} />
                <ProfileItem label="Has Release Assets" value={repository.has_release_assets ? 'Yes' : 'No'} />
              </ProfileCard>

              <ProfileCard title="Git Properties">
                <ProfileItem label="Default Branch" value={repository.default_branch} />
                <ProfileItem label="Total Size" value={formatBytes(repository.total_size)} />
                <ProfileItem label="Branches" value={repository.branch_count} />
                <ProfileItem label="Commits" value={repository.commit_count.toLocaleString()} />
                <ProfileItem label="Has LFS" value={repository.has_lfs ? 'Yes' : 'No'} />
                <ProfileItem label="Has Submodules" value={repository.has_submodules ? 'Yes' : 'No'} />
                
                {/* Largest File */}
                {repository.largest_file && (
                  <ProfileItem 
                    label="Largest File" 
                    value={
                      <code className="text-xs bg-gray-100 px-2 py-1 rounded break-all">
                        {repository.largest_file}
                      </code>
                    } 
                  />
                )}
                
                {/* Largest File Size */}
                {repository.largest_file_size && (
                  <ProfileItem 
                    label="Largest File Size" 
                    value={formatBytes(repository.largest_file_size)} 
                  />
                )}
                
                {/* Largest Commit */}
                {repository.largest_commit && (
                  <ProfileItem 
                    label="Largest Commit" 
                    value={
                      <code className="text-xs bg-gray-100 px-2 py-1 rounded">
                        {repository.largest_commit.substring(0, 8)}
                      </code>
                    } 
                  />
                )}
                
                {/* Largest Commit Size */}
                {repository.largest_commit_size && (
                  <ProfileItem 
                    label="Largest Commit Size" 
                    value={formatBytes(repository.largest_commit_size)} 
                  />
                )}
              </ProfileCard>

              <ProfileCard title="Verification Metrics">
                <ProfileItem 
                  label="Last Commit SHA" 
                  value={repository.last_commit_sha ? (
                    <code className="text-xs bg-gray-100 px-2 py-1 rounded">{repository.last_commit_sha.substring(0, 8)}</code>
                  ) : 'Unknown'} 
                />
                <ProfileItem label="Branches" value={repository.branch_count} />
                <ProfileItem label="Tags/Releases" value={repository.tag_count} />
                <ProfileItem 
                  label="Issues" 
                  value={`${repository.open_issue_count} open / ${repository.issue_count} total`} 
                />
                <ProfileItem 
                  label="Pull Requests" 
                  value={`${repository.open_pr_count} open / ${repository.pull_request_count} total`} 
                />
                <ProfileItem label="Contributors" value={repository.contributor_count} />
              </ProfileCard>
            </div>
          )}

          {activeTab === 'history' && (
            <div>
              {history.length === 0 ? (
                <p className="text-gray-500">No migration history yet</p>
              ) : (
                <div className="space-y-3">
                  {history.map((event) => (
                    <MigrationEvent key={event.id} event={event} />
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === 'dependencies' && fullName && (
            <DependenciesTab fullName={fullName} />
          )}

          {activeTab === 'logs' && (
            <div>
              {/* Log Filters */}
              <div className="flex gap-4 mb-4 flex-wrap">
                <select
                  value={logLevel}
                  onChange={(e) => setLogLevel(e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">All Levels</option>
                  <option value="DEBUG">Debug</option>
                  <option value="INFO">Info</option>
                  <option value="WARN">Warning</option>
                  <option value="ERROR">Error</option>
                </select>

                <select
                  value={logPhase}
                  onChange={(e) => setLogPhase(e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">All Phases</option>
                  <option value="discovery">Discovery</option>
                  <option value="pre_migration">Pre-migration</option>
                  <option value="archive_generation">Archive Generation</option>
                  <option value="migration">Migration</option>
                  <option value="post_migration">Post-migration</option>
                </select>

                <input
                  type="text"
                  placeholder="Search logs..."
                  value={logSearch}
                  onChange={(e) => setLogSearch(e.target.value)}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />

                <button
                  onClick={() => loadLogs()}
                  className="px-4 py-1.5 border border-gh-border-default text-gh-text-primary rounded-md text-sm font-medium hover:bg-gh-neutral-bg"
                >
                  Refresh
                </button>
              </div>

              {/* Logs Display */}
              {logsLoading ? (
                <div className="text-center py-8 text-gray-500">Loading logs...</div>
              ) : logs.length === 0 ? (
                <p className="text-gray-500">No logs available</p>
              ) : (
                <div className="space-y-1 font-mono text-sm max-h-96 overflow-y-auto bg-gray-50 rounded-lg p-4">
                  {logs
                    .filter((log) =>
                      logSearch ? log.message.toLowerCase().includes(logSearch.toLowerCase()) : true
                    )
                    .map((log) => (
                      <LogEntry key={log.id} log={log} />
                    ))}
                </div>
              )}

              {logs.length > 0 && (
                <div className="mt-4 text-sm text-gray-500">
                  Showing {logs.filter((log) => logSearch ? log.message.toLowerCase().includes(logSearch.toLowerCase()) : true).length} of {logs.length} logs
                </div>
              )}
            </div>
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

function MigrationEvent({ event }: { event: MigrationHistory }) {
  return (
    <div className="border-l-4 border-blue-500 pl-4 py-2">
      <div className="flex justify-between items-start">
        <div>
          <div className="font-medium text-gray-900">{event.phase}</div>
          <div className="text-sm text-gray-600">{event.message}</div>
          {event.error_message && (
            <div className="text-sm text-red-600 mt-1">{event.error_message}</div>
          )}
        </div>
        <div className="text-sm text-gray-500">
          {formatDate(event.started_at)}
        </div>
      </div>
      {event.duration_seconds !== undefined && event.duration_seconds !== null && (
        <div className="text-sm text-gray-500 mt-1">
          Duration: {event.duration_seconds}s
        </div>
      )}
    </div>
  );
}

function LogEntry({ log }: { log: MigrationLog }) {
  const [expanded, setExpanded] = useState(false);

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'ERROR': return 'text-red-600 bg-red-50';
      case 'WARN': return 'text-yellow-600 bg-yellow-50';
      case 'INFO': return 'text-blue-600 bg-blue-50';
      case 'DEBUG': return 'text-gray-600 bg-gray-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  const getLevelIcon = (level: string) => {
    switch (level) {
      case 'ERROR': return '❌';
      case 'WARN': return '⚠️';
      case 'INFO': return 'ℹ️';
      case 'DEBUG': return '🔍';
      default: return '•';
    }
  };

  return (
    <div className="hover:bg-gray-100 p-2 rounded cursor-pointer" onClick={() => setExpanded(!expanded)}>
      <div className="flex items-start gap-2">
        {/* Timestamp */}
        <span className="text-gray-500 whitespace-nowrap text-xs">
          {new Date(log.timestamp).toLocaleTimeString()}
        </span>
        
        {/* Level Badge */}
        <span className={`px-2 py-0.5 rounded text-xs font-medium ${getLevelColor(log.level)}`}>
          {getLevelIcon(log.level)} {log.level}
        </span>
        
        {/* Phase & Operation */}
        <span className="text-gray-600 whitespace-nowrap text-xs">
          [{log.phase}:{log.operation}]
        </span>
        
        {/* Message */}
        <span className={`flex-1 text-xs ${log.level === 'ERROR' ? 'text-red-700 font-medium' : 'text-gray-800'}`}>
          {log.message}
        </span>
      </div>
      
      {/* Expanded Details */}
      {expanded && log.details && (
        <div className="mt-2 pl-4 border-l-2 border-gray-300">
          <pre className="text-xs text-gray-600 whitespace-pre-wrap break-words">
            {log.details}
          </pre>
        </div>
      )}
    </div>
  );
}

