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

  const handleStartBatch = async (batchId: number) => {
    if (!confirm('Are you sure you want to start migration for this entire batch?')) {
      return;
    }

    try {
      await api.startBatch(batchId);
      alert('Batch migration started successfully');
      await loadBatches();
      if (selectedBatch?.id === batchId) {
        await loadBatchRepositories(batchId);
      }
    } catch (error) {
      console.error('Failed to start batch:', error);
      alert('Failed to start batch migration');
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
    };

    repos.forEach((repo) => {
      if (repo.status === 'complete') {
        groups.complete.push(repo);
      } else if (repo.status === 'migration_failed' || repo.status === 'dry_run_failed') {
        groups.failed.push(repo);
      } else if (
        repo.status === 'queued_for_migration' ||
        repo.status === 'migrating_content' ||
        repo.status === 'dry_run_in_progress'
      ) {
        groups.in_progress.push(repo);
      } else {
        groups.pending.push(repo);
      }
    });

    return groups;
  };

  const progress = selectedBatch ? getBatchProgress(selectedBatch, batchRepositories) : null;
  const groupedRepos = groupReposByStatus(batchRepositories);

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-light text-gray-900">Batch Management</h1>
        <button
          onClick={handleCreateBatch}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700"
        >
          Create New Batch
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Batch List */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg shadow-sm p-4">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Batches</h2>
            {loading ? (
              <LoadingSpinner />
            ) : batches.length === 0 ? (
              <div className="text-center py-8 text-gray-500">No batches found</div>
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
            <div className="bg-white rounded-lg shadow-sm p-6">
              <div className="flex justify-between items-start mb-6">
                <div className="flex-1">
                  <h2 className="text-2xl font-medium text-gray-900">{selectedBatch.name}</h2>
                  {selectedBatch.description && (
                    <p className="text-gray-600 mt-1">{selectedBatch.description}</p>
                  )}
                  <div className="flex gap-3 mt-3">
                    <StatusBadge status={selectedBatch.status} />
                    <span className="text-sm text-gray-600">
                      {selectedBatch.repository_count} repositories
                    </span>
                  </div>
                  {selectedBatch.scheduled_at && (
                    <div className="text-sm text-gray-600 mt-2">
                      Scheduled: {formatDate(selectedBatch.scheduled_at)}
                    </div>
                  )}
                </div>

                <div className="flex gap-2">
                  {selectedBatch.status === 'ready' && (
                    <>
                      <button
                        onClick={() => handleEditBatch(selectedBatch)}
                        className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50"
                      >
                        Edit Batch
                      </button>
                      <button
                        onClick={() => handleStartBatch(selectedBatch.id)}
                        className="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700"
                      >
                        Start Migration
                      </button>
                    </>
                  )}
                  {groupedRepos.failed.length > 0 && (
                    <button
                      onClick={handleRetryFailed}
                      className="px-4 py-2 bg-yellow-600 text-white rounded-lg text-sm font-medium hover:bg-yellow-700"
                    >
                      Retry All Failed ({groupedRepos.failed.length})
                    </button>
                  )}
                </div>
              </div>

              {/* Progress Bar */}
              {progress && progress.total > 0 && (
                <div className="mb-6 bg-gray-50 p-4 rounded-lg">
                  <div className="flex justify-between text-sm text-gray-600 mb-2">
                    <span>Progress</span>
                    <span>
                      {progress.completed} / {progress.total} ({progress.percentage}%)
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
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
            className="text-sm px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Start
          </button>
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
