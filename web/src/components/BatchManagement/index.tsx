import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../../services/api';
import type { Batch, Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { formatBytes, formatDate } from '../../utils/format';

export function BatchManagement() {
  const navigate = useNavigate();
  const [batches, setBatches] = useState<Batch[]>([]);
  const [selectedBatch, setSelectedBatch] = useState<Batch | null>(null);
  const [batchRepositories, setBatchRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadBatches();
  }, []);

  useEffect(() => {
    if (selectedBatch) {
      loadBatchRepositories(selectedBatch.id);
      
      // Poll for updates every 10 seconds if batch is in progress
      if (selectedBatch.status === 'in_progress') {
        const interval = setInterval(() => {
          loadBatches();
          loadBatchRepositories(selectedBatch.id);
        }, 10000);
        return () => clearInterval(interval);
      }
    }
  }, [selectedBatch]);

  const loadBatches = async () => {
    try {
      const data = await api.listBatches();
      setBatches(data);
      
      // Update selected batch if it's in the list
      if (selectedBatch) {
        const updated = data.find((b) => b.id === selectedBatch.id);
        if (updated) {
          setSelectedBatch(updated);
        }
      }
    } catch (error) {
      console.error('Failed to load batches:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadBatchRepositories = async (batchId: number) => {
    try {
      const response = await api.listRepositories({ batch_id: batchId });
      const repos = response.repositories || response as any;
      setBatchRepositories(repos);
    } catch (error) {
      console.error('Failed to load batch repositories:', error);
    }
  };

  const handleDryRunBatch = async (batchId: number, onlyPending = false) => {
    const actionType = onlyPending ? 'pending repositories' : 'all repositories';
    if (!confirm(`Run dry run for ${actionType}? This will validate repositories before migration.`)) {
      return;
    }

    try {
      await api.dryRunBatch(batchId, onlyPending);
      alert('Dry run started successfully. Batch will move to "ready" status when all dry runs complete.');
      await loadBatches();
      if (selectedBatch?.id === batchId) {
        await loadBatchRepositories(batchId);
      }
    } catch (error: any) {
      console.error('Failed to start dry run:', error);
      alert(error.response?.data?.error || 'Failed to start dry run');
    }
  };

  const handleStartBatch = async (batchId: number, skipDryRun = false) => {
    const batch = batches.find(b => b.id === batchId);
    
    if (batch?.status === 'pending' && !skipDryRun) {
      const shouldSkip = confirm(
        'This batch has not completed a dry run. Do you want to start migration anyway? ' +
        '(Recommended: Cancel and run dry run first)'
      );
      
      if (!shouldSkip) {
        return;
      }
    }

    if (!confirm('Are you sure you want to start migration for this entire batch?')) {
      return;
    }

    try {
      await api.startBatch(batchId, skipDryRun);
      alert('Batch migration started successfully');
      await loadBatches();
      if (selectedBatch?.id === batchId) {
        await loadBatchRepositories(batchId);
      }
    } catch (error: any) {
      console.error('Failed to start batch:', error);
      alert(error.response?.data?.error || 'Failed to start batch migration');
    }
  };

  const handleRetryFailed = async () => {
    if (!selectedBatch) return;

    const failedRepos = batchRepositories.filter(
      (r) => r.status === 'migration_failed' || r.status === 'dry_run_failed'
    );

    if (failedRepos.length === 0) return;

    if (!confirm(`Retry migration for ${failedRepos.length} failed repositories?`)) {
      return;
    }

    try {
      await api.retryBatchFailures(selectedBatch.id);
      alert(`Queued ${failedRepos.length} repositories for retry`);
      await loadBatchRepositories(selectedBatch.id);
    } catch (error) {
      console.error('Failed to retry batch failures:', error);
      alert('Failed to retry failed repositories');
    }
  };

  const handleRetryRepository = async (repoId: number) => {
    try {
      await api.retryRepository(repoId);
      alert('Repository queued for retry');
      if (selectedBatch) {
        await loadBatchRepositories(selectedBatch.id);
      }
    } catch (error) {
      console.error('Failed to retry repository:', error);
      alert('Failed to retry repository');
    }
  };

  const handleCreateBatch = () => {
    navigate('/batches/new');
  };

  const handleEditBatch = (batch: Batch) => {
    navigate(`/batches/${batch.id}/edit`);
  };

  const handleDeleteBatch = async (batch: Batch) => {
    if (batch.status === 'in_progress') {
      alert('Cannot delete a batch that is currently in progress.');
      return;
    }

    const confirmMessage = batch.repository_count > 0
      ? `Delete batch "${batch.name}"? This will remove ${batch.repository_count} repositories from the batch, making them available for other batches.`
      : `Delete batch "${batch.name}"?`;

    if (!confirm(confirmMessage)) {
      return;
    }

    try {
      await api.deleteBatch(batch.id);
      alert('Batch deleted successfully');
      // Clear selection if we deleted the selected batch
      if (selectedBatch?.id === batch.id) {
        setSelectedBatch(null);
      }
      await loadBatches();
    } catch (error: any) {
      console.error('Failed to delete batch:', error);
      alert(error.response?.data?.error || 'Failed to delete batch');
    }
  };

  const getBatchProgress = (_batch: Batch, repos: Repository[]) => {
    if (repos.length === 0) return { completed: 0, total: 0, percentage: 0 };
    
    const completed = repos.filter((r) => r.status === 'complete').length;
    const total = repos.length;
    const percentage = Math.round((completed / total) * 100);
    
    return { completed, total, percentage };
  };

  const groupReposByStatus = (repos: Repository[]) => {
    const groups: Record<string, Repository[]> = {
      complete: [],
      in_progress: [],
      failed: [],
      pending: [],
      needs_dry_run: [],
      dry_run_complete: [],
    };

    repos.forEach((repo) => {
      if (repo.status === 'complete') {
        groups.complete.push(repo);
      } else if (repo.status === 'migration_failed' || repo.status === 'dry_run_failed') {
        groups.failed.push(repo);
      } else if (
        repo.status === 'queued_for_migration' ||
        repo.status === 'migrating_content' ||
        repo.status === 'dry_run_in_progress' ||
        repo.status === 'dry_run_queued' ||
        repo.status === 'pre_migration' ||
        repo.status === 'archive_generating' ||
        repo.status === 'post_migration'
      ) {
        groups.in_progress.push(repo);
      } else if (repo.status === 'dry_run_complete') {
        groups.dry_run_complete.push(repo);
      } else {
        groups.pending.push(repo);
      }

      // Track repos that need dry runs (pending, failed, or rolled back)
      if (
        repo.status === 'pending' ||
        repo.status === 'dry_run_failed' ||
        repo.status === 'migration_failed' ||
        repo.status === 'rolled_back'
      ) {
        groups.needs_dry_run.push(repo);
      }
    });

    return groups;
  };

  const progress = selectedBatch ? getBatchProgress(selectedBatch, batchRepositories) : null;
  const groupedRepos = groupReposByStatus(batchRepositories);

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold text-gh-text-primary">Batch Management</h1>
        <button
          onClick={handleCreateBatch}
          className="px-4 py-1.5 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover"
        >
          Create New Batch
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Batch List */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg border border-gh-border-default shadow-gh-card p-4">
            <h2 className="text-base font-semibold text-gh-text-primary mb-4">Batches</h2>
            {loading ? (
              <LoadingSpinner />
            ) : batches.length === 0 ? (
              <div className="text-center py-8 text-gh-text-secondary">No batches found</div>
            ) : (
              <div className="space-y-2">
                {batches.map((batch) => (
                  <BatchCard
                    key={batch.id}
                    batch={batch}
                    isSelected={selectedBatch?.id === batch.id}
                    onClick={() => setSelectedBatch(batch)}
                    onStart={() => handleStartBatch(batch.id)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Batch Detail */}
        <div className="lg:col-span-2">
          {selectedBatch ? (
            <div className="bg-white rounded-lg border border-gh-border-default shadow-gh-card p-6">
              <div className="flex justify-between items-start mb-6">
                <div className="flex-1">
                  <h2 className="text-xl font-semibold text-gh-text-primary">{selectedBatch.name}</h2>
                  {selectedBatch.description && (
                    <p className="text-gh-text-secondary mt-1">{selectedBatch.description}</p>
                  )}
                  <div className="flex gap-3 mt-3">
                    <StatusBadge status={selectedBatch.status} />
                    <span className="text-sm text-gh-text-secondary">
                      {selectedBatch.repository_count} repositories
                    </span>
                  </div>
                  {selectedBatch.scheduled_at && (
                    <div className="text-sm text-gh-text-secondary mt-2">
                      Scheduled: {formatDate(selectedBatch.scheduled_at)}
                    </div>
                  )}
                </div>

                <div className="flex gap-2">
                  {(selectedBatch.status === 'pending' || selectedBatch.status === 'ready') && (
                    <>
                      <button
                        onClick={() => handleEditBatch(selectedBatch)}
                        className="px-4 py-1.5 border border-gh-border-default text-gh-text-primary rounded-md text-sm font-medium hover:bg-gh-neutral-bg"
                      >
                        Edit Batch
                      </button>
                      <button
                        onClick={() => handleDeleteBatch(selectedBatch)}
                        className="px-4 py-1.5 border border-red-600 text-red-600 rounded-md text-sm font-medium hover:bg-red-50"
                      >
                        Delete
                      </button>
                    </>
                  )}
                  
                  {selectedBatch.status === 'pending' && (
                    <>
                      {groupedRepos.needs_dry_run.length > 0 && (
                        <button
                          onClick={() => handleDryRunBatch(selectedBatch.id, true)}
                          className="px-4 py-1.5 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
                        >
                          Run Dry Run ({groupedRepos.needs_dry_run.length} repos)
                        </button>
                      )}
                      <button
                        onClick={() => handleStartBatch(selectedBatch.id, true)}
                        className="px-4 py-1.5 border border-gh-border-default text-gh-text-primary rounded-md text-sm font-medium hover:bg-gh-neutral-bg"
                      >
                        Skip & Migrate
                      </button>
                    </>
                  )}
                  
                  {selectedBatch.status === 'ready' && (
                    <>
                      <button
                        onClick={() => handleStartBatch(selectedBatch.id)}
                        className="px-4 py-1.5 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover"
                      >
                        Start Migration
                      </button>
                      {groupedRepos.needs_dry_run.length > 0 ? (
                        <button
                          onClick={() => handleDryRunBatch(selectedBatch.id, true)}
                          className="px-3 py-1.5 border border-blue-600 text-blue-600 rounded-md text-sm font-medium hover:bg-blue-50"
                          title="Run dry run only for repositories that need it"
                        >
                          Dry Run Pending ({groupedRepos.needs_dry_run.length})
                        </button>
                      ) : null}
                      <button
                        onClick={() => handleDryRunBatch(selectedBatch.id, false)}
                        className="px-3 py-1.5 border border-gh-border-default text-gh-text-secondary rounded-md text-sm font-medium hover:bg-gh-neutral-bg"
                        title="Re-run dry run for all repositories"
                      >
                        Re-run All Dry Runs
                      </button>
                    </>
                  )}
                  
                  {groupedRepos.failed.length > 0 && (
                    <button
                      onClick={handleRetryFailed}
                      className="px-4 py-1.5 bg-gh-warning text-white rounded-md text-sm font-medium hover:bg-gh-warning-emphasis"
                    >
                      Retry All Failed ({groupedRepos.failed.length})
                    </button>
                  )}
                </div>
              </div>

              {/* Progress Bar */}
              {progress && progress.total > 0 && (
                <div className="mb-6 bg-gh-neutral-bg p-4 rounded-lg">
                  <div className="flex justify-between text-sm text-gh-text-secondary mb-2">
                    <span>Progress</span>
                    <span>
                      {progress.completed} / {progress.total} ({progress.percentage}%)
                    </span>
                  </div>
                  <div className="w-full bg-gh-border-default rounded-full h-2">
                    <div
                      className="bg-gh-success h-2 rounded-full transition-all duration-300"
                      style={{ width: `${progress.percentage}%` }}
                    />
                  </div>
                </div>
              )}

              {/* Repositories by Status */}
              <div className="space-y-6">
                {/* Failed Repositories */}
                {groupedRepos.failed.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium text-red-800 mb-3">
                      Failed ({groupedRepos.failed.length})
                    </h3>
                    <div className="space-y-2">
                      {groupedRepos.failed.map((repo) => (
                        <RepositoryItem
                          key={repo.id}
                          repository={repo}
                          onRetry={() => handleRetryRepository(repo.id)}
                        />
                      ))}
                    </div>
                  </div>
                )}

                {/* In Progress Repositories */}
                {groupedRepos.in_progress.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium text-blue-800 mb-3">
                      In Progress ({groupedRepos.in_progress.length})
                    </h3>
                    <div className="space-y-2">
                      {groupedRepos.in_progress.map((repo) => (
                        <RepositoryItem key={repo.id} repository={repo} />
                      ))}
                    </div>
                  </div>
                )}

                {/* Completed Repositories */}
                {groupedRepos.complete.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium text-green-800 mb-3">
                      Completed ({groupedRepos.complete.length})
                    </h3>
                    <div className="space-y-2">
                      {groupedRepos.complete.map((repo) => (
                        <RepositoryItem key={repo.id} repository={repo} />
                      ))}
                    </div>
                  </div>
                )}

                {/* Pending Repositories */}
                {groupedRepos.pending.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium text-gray-800 mb-3">
                      Pending ({groupedRepos.pending.length})
                    </h3>
                    <div className="space-y-2">
                      {groupedRepos.pending.map((repo) => (
                        <RepositoryItem key={repo.id} repository={repo} />
                      ))}
                    </div>
                  </div>
                )}

                {batchRepositories.length === 0 && (
                  <div className="text-center py-8 text-gray-500">
                    No repositories in this batch
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="bg-white rounded-lg shadow-sm p-6 text-center text-gray-500">
              Select a batch to view details
            </div>
          )}
        </div>
      </div>

    </div>
  );
}

interface BatchCardProps {
  batch: Batch;
  isSelected: boolean;
  onClick: () => void;
  onStart: () => void;
}

function BatchCard({ batch, isSelected, onClick, onStart }: BatchCardProps) {
  return (
    <div
      className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
        isSelected
          ? 'border-blue-500 bg-blue-50'
          : 'border-gray-200 hover:border-gray-300'
      }`}
      onClick={onClick}
    >
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <h3 className="font-medium text-gray-900">{batch.name}</h3>
          <div className="flex gap-2 mt-2">
            <StatusBadge status={batch.status} size="sm" />
            <span className="text-xs text-gray-600">{batch.repository_count} repos</span>
          </div>
        </div>
        {batch.status === 'ready' && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onStart();
            }}
            className="text-sm px-3 py-1 bg-gh-success text-white rounded hover:bg-gh-success-hover"
          >
            Start
          </button>
        )}
        {batch.status === 'pending' && (
          <span className="text-xs text-gray-500">
            Dry run needed
          </span>
        )}
      </div>
    </div>
  );
}

interface RepositoryItemProps {
  repository: Repository;
  onRetry?: () => void;
}

function RepositoryItem({ repository, onRetry }: RepositoryItemProps) {
  const isFailed = repository.status === 'migration_failed' || repository.status === 'dry_run_failed';
  const isDryRunFailed = repository.status === 'dry_run_failed';

  return (
    <div className="flex justify-between items-center p-3 border border-gh-border-default rounded-lg hover:bg-gh-neutral-bg group">
      <Link to={`/repository/${encodeURIComponent(repository.full_name)}`} className="flex-1 min-w-0">
        <div className="font-semibold text-gh-text-primary group-hover:text-gh-blue transition-colors">
          {repository.full_name}
        </div>
        <div className="text-sm text-gh-text-secondary mt-1">
          {formatBytes(repository.total_size || 0)} • {repository.branch_count} branches
        </div>
      </Link>
      <div className="flex items-center gap-3">
        <StatusBadge status={repository.status} size="sm" />
        {isFailed && onRetry && (
          <button
            onClick={(e) => {
              e.preventDefault();
              onRetry();
            }}
            className="text-sm px-3 py-1.5 bg-gh-warning text-white rounded-md font-medium hover:bg-gh-warning-emphasis"
            title={isDryRunFailed ? 'Re-run dry run or view details to start migration' : 'Retry migration'}
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
}
