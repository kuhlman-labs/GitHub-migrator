import { useEffect, useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { TextInput, Button, UnderlineNav } from '@primer/react';
import { SearchIcon, PlusIcon, CalendarIcon, GearIcon, ClockIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { Batch, Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { Pagination } from '../common/Pagination';
import { formatBytes, formatDate } from '../../utils/format';

type BatchTab = 'active' | 'completed';

export function BatchManagement() {
  const navigate = useNavigate();
  const location = useLocation();
  const locationState = location.state as { selectedBatchId?: number } | null;
  const [batches, setBatches] = useState<Batch[]>([]);
  const [selectedBatch, setSelectedBatch] = useState<Batch | null>(null);
  const [batchRepositories, setBatchRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<BatchTab>('active');
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 10;

  useEffect(() => {
    loadBatches();

    // Poll for batch list updates every 30 seconds to catch scheduled batches starting
    const batchListInterval = setInterval(() => {
      loadBatches();
    }, 30000);

    return () => clearInterval(batchListInterval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Handle location state to auto-select batch when navigating from repository detail
  useEffect(() => {
    if (locationState?.selectedBatchId && batches.length > 0 && !selectedBatch) {
      const batch = batches.find((b) => b.id === locationState.selectedBatchId);
      if (batch) {
        setSelectedBatch(batch);
      }
    }
  }, [locationState, batches, selectedBatch]);

  useEffect(() => {
    if (selectedBatch) {
      loadBatchRepositories(selectedBatch.id);
      
      // Poll for updates more frequently if batch is in progress or scheduled/ready
      const shouldPoll = 
        selectedBatch.status === 'in_progress' || 
        selectedBatch.status === 'ready' ||
        (selectedBatch.scheduled_at && new Date(selectedBatch.scheduled_at) > new Date());

      if (shouldPoll) {
        const pollInterval = selectedBatch.status === 'in_progress' ? 10000 : 30000;
        const interval = setInterval(() => {
          loadBatches();
          loadBatchRepositories(selectedBatch.id);
        }, pollInterval);
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

    const dryRunFailedCount = failedRepos.filter(r => r.status === 'dry_run_failed').length;
    const migrationFailedCount = failedRepos.filter(r => r.status === 'migration_failed').length;
    
    let confirmMessage = '';
    if (dryRunFailedCount > 0 && migrationFailedCount > 0) {
      confirmMessage = `Retry ${dryRunFailedCount} failed dry run(s) and ${migrationFailedCount} failed migration(s)?`;
    } else if (dryRunFailedCount > 0) {
      confirmMessage = `Re-run dry run for ${dryRunFailedCount} failed repositories?`;
    } else {
      confirmMessage = `Retry migration for ${migrationFailedCount} failed repositories?`;
    }

    if (!confirm(confirmMessage)) {
      return;
    }

    try {
      // Retry each repository individually with the correct dry_run flag
      for (const repo of failedRepos) {
        const isDryRunFailed = repo.status === 'dry_run_failed';
        await api.retryRepository(repo.id, isDryRunFailed);
      }
      alert(`Queued ${failedRepos.length} repositories for retry`);
      await loadBatchRepositories(selectedBatch.id);
    } catch (error: any) {
      console.error('Failed to retry batch failures:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to retry failed repositories';
      alert(errorMessage);
    }
  };

  const handleRetryRepository = async (repo: Repository) => {
    const isDryRunFailed = repo.status === 'dry_run_failed';
    const actionType = isDryRunFailed ? 'dry run' : 'migration';
    
    try {
      await api.retryRepository(repo.id, isDryRunFailed);
      alert(`Repository queued for ${actionType} retry`);
      if (selectedBatch) {
        await loadBatchRepositories(selectedBatch.id);
      }
    } catch (error: any) {
      console.error('Failed to retry repository:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to retry repository';
      alert(errorMessage);
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

  // Filter batches by tab (active vs completed)
  const activeStatuses = ['pending', 'ready', 'in_progress'];
  const completedStatuses = ['completed', 'completed_with_errors', 'failed', 'cancelled'];
  
  const filteredByTab = batches.filter((batch) =>
    activeTab === 'active'
      ? activeStatuses.includes(batch.status)
      : completedStatuses.includes(batch.status)
  );

  // Filter by search term
  const filteredBatches = filteredByTab.filter((batch) =>
    batch.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  // Get counts for tabs
  const activeBatchCount = batches.filter((b) => activeStatuses.includes(b.status)).length;
  const completedBatchCount = batches.filter((b) => completedStatuses.includes(b.status)).length;

  // Paginate
  const totalItems = filteredBatches.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedBatches = filteredBatches.slice(startIndex, endIndex);

  // Reset page when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [activeTab, searchTerm]);

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold text-gh-text-primary">Batch Management</h1>
        <div className="flex items-center gap-4">
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search batches..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            style={{ width: 300 }}
          />
          <Button
            onClick={handleCreateBatch}
            variant="primary"
            leadingVisual={PlusIcon}
          >
            Create New Batch
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Batch List */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg border border-gh-border-default shadow-gh-card">
            {/* Tabs */}
            <UnderlineNav aria-label="Batch tabs">
              <UnderlineNav.Item
                aria-current={activeTab === 'active' ? 'page' : undefined}
                onSelect={() => setActiveTab('active')}
              >
                Active ({activeBatchCount})
              </UnderlineNav.Item>
              <UnderlineNav.Item
                aria-current={activeTab === 'completed' ? 'page' : undefined}
                onSelect={() => setActiveTab('completed')}
              >
                Completed ({completedBatchCount})
              </UnderlineNav.Item>
            </UnderlineNav>

            {/* Batch List */}
            <div className="p-4">
              {loading ? (
                <LoadingSpinner />
              ) : paginatedBatches.length === 0 ? (
                <div className="text-center py-8 text-gh-text-secondary">
                  {searchTerm ? 'No batches match your search' : 'No batches found'}
                </div>
              ) : (
                <>
                  <div className="space-y-2 mb-4">
                    {paginatedBatches.map((batch) => (
                      <BatchCard
                        key={batch.id}
                        batch={batch}
                        isSelected={selectedBatch?.id === batch.id}
                        onClick={() => setSelectedBatch(batch)}
                        onStart={() => handleStartBatch(batch.id)}
                      />
                    ))}
                  </div>
                  {totalItems > pageSize && (
                    <Pagination
                      currentPage={currentPage}
                      totalItems={totalItems}
                      pageSize={pageSize}
                      onPageChange={setCurrentPage}
                    />
                  )}
                </>
              )}
            </div>
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
                  <div className="flex items-center gap-3 mt-3">
                    <StatusBadge status={selectedBatch.status} />
                    <span className="text-sm text-gh-text-secondary">
                      {selectedBatch.repository_count} repositories
                    </span>
                    {selectedBatch.created_at && (
                      <>
                        <span className="text-gray-300">•</span>
                        <span className="text-xs text-gray-400">
                          Created {formatDate(selectedBatch.created_at)}
                        </span>
                      </>
                    )}
                  </div>
                  
                  {/* Two-column layout for settings and timestamps */}
                  <div className="mt-4 grid grid-cols-1 lg:grid-cols-2 gap-6 border-t border-gh-border-default pt-4">
                    {/* Left Column: Migration Settings */}
                    {(selectedBatch.destination_org || selectedBatch.exclude_releases || selectedBatch.migration_api !== 'GEI') && (
                      <div>
                        <div className="flex items-center gap-2 mb-2">
                          <GearIcon size={16} className="text-gray-500" />
                          <span className="text-sm font-semibold text-gh-text-primary">Migration Settings</span>
                        </div>
                        <div className="space-y-2 pl-6">
                          {selectedBatch.destination_org && (
                            <div className="text-sm">
                              <span className="text-gh-text-secondary">Default Destination:</span>
                              <div className="font-medium text-blue-700 mt-0.5">{selectedBatch.destination_org}</div>
                              <div className="text-xs text-gray-500 italic mt-0.5">For repos without specific destination</div>
                            </div>
                          )}
                          {selectedBatch.migration_api && selectedBatch.migration_api !== 'GEI' && (
                            <div className="text-sm">
                              <span className="text-gh-text-secondary">Migration API:</span>
                              <div className="font-medium text-gh-text-primary mt-0.5">
                                {selectedBatch.migration_api === 'ELM' ? 'ELM (Enterprise Live Migrator)' : selectedBatch.migration_api}
                              </div>
                            </div>
                          )}
                          {selectedBatch.exclude_releases && (
                            <div className="text-sm">
                              <span className="text-gh-text-secondary">Exclude Releases:</span>
                              <div className="font-medium text-orange-700 mt-0.5">Yes</div>
                              <div className="text-xs text-gray-500 italic mt-0.5">Repo settings can override</div>
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                    {/* Right Column: Schedule & Timestamps */}
                    <div>
                      <div className="flex items-center gap-2 mb-2">
                        <ClockIcon size={16} className="text-gray-500" />
                        <span className="text-sm font-semibold text-gh-text-primary">Schedule & Timeline</span>
                      </div>
                      <div className="space-y-2 pl-6">
                        {selectedBatch.scheduled_at && (
                          <div className="text-sm">
                            <span className="text-gh-text-secondary">Scheduled:</span>
                            <div className="font-medium text-blue-900 mt-0.5">
                              {formatDate(selectedBatch.scheduled_at)}
                            </div>
                            {new Date(selectedBatch.scheduled_at) > new Date() && (
                              <div className="text-xs text-blue-600 italic mt-0.5">Auto-start when ready</div>
                            )}
                          </div>
                        )}
                        {selectedBatch.last_dry_run_at && (
                          <div className="text-sm">
                            <span className="text-gh-text-secondary">Last Dry Run:</span>
                            <div className="font-medium text-gh-text-primary mt-0.5">
                              {formatDate(selectedBatch.last_dry_run_at)}
                            </div>
                          </div>
                        )}
                        {selectedBatch.last_migration_attempt_at && (
                          <div className="text-sm">
                            <span className="text-gh-text-secondary">Last Migration:</span>
                            <div className="font-medium text-gh-text-primary mt-0.5">
                              {formatDate(selectedBatch.last_migration_attempt_at)}
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </div>

                <div className="flex gap-2">
                  {(selectedBatch.status === 'pending' || selectedBatch.status === 'ready') && (
                    <>
                      <Button
                        onClick={() => handleEditBatch(selectedBatch)}
                        variant="default"
                      >
                        Edit Batch
                      </Button>
                      <Button
                        onClick={() => handleDeleteBatch(selectedBatch)}
                        variant="danger"
                      >
                        Delete
                      </Button>
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
                      <Button
                        onClick={() => handleStartBatch(selectedBatch.id, true)}
                        variant="default"
                      >
                        Skip & Migrate
                      </Button>
                    </>
                  )}
                  
                  {selectedBatch.status === 'ready' && (
                    <>
                      <button
                        onClick={() => handleStartBatch(selectedBatch.id)}
                        className="px-4 py-1.5 bg-green-600 text-white rounded-md text-sm font-medium hover:bg-green-700"
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
                      <Button
                        onClick={() => handleDryRunBatch(selectedBatch.id, false)}
                        variant="default"
                        title="Re-run dry run for all repositories"
                      >
                        Re-run All Dry Runs
                      </Button>
                    </>
                  )}
                  
                  {groupedRepos.failed.length > 0 && (() => {
                    const dryRunFailed = groupedRepos.failed.filter(r => r.status === 'dry_run_failed').length;
                    const migrationFailed = groupedRepos.failed.filter(r => r.status === 'migration_failed').length;
                    
                    let buttonText = '';
                    if (dryRunFailed > 0 && migrationFailed > 0) {
                      buttonText = `Retry All Failed (${dryRunFailed} dry run, ${migrationFailed} migration)`;
                    } else if (dryRunFailed > 0) {
                      buttonText = `Re-run All Dry Runs (${dryRunFailed})`;
                    } else {
                      buttonText = `Retry All Migrations (${migrationFailed})`;
                    }
                    
                    return (
                      <Button
                        onClick={handleRetryFailed}
                        variant="danger"
                        style={{ whiteSpace: 'nowrap' }}
                      >
                        {buttonText}
                      </Button>
                    );
                  })()}
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
                          onRetry={() => handleRetryRepository(repo)}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
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
                        <RepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
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
                        <RepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
                      ))}
                    </div>
                  </div>
                )}

                {/* Dry Run Complete (Ready for Migration) */}
                {groupedRepos.dry_run_complete.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium text-blue-800 mb-3">
                      Ready for Migration ({groupedRepos.dry_run_complete.length})
                    </h3>
                    <div className="space-y-2">
                      {groupedRepos.dry_run_complete.map((repo) => (
                        <RepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
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
                        <RepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
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
            <StatusBadge status={batch.status} size="small" />
            <span className="text-xs text-gray-600">{batch.repository_count} repos</span>
          </div>
          {batch.scheduled_at && (
            <div className="mt-1.5 text-xs text-blue-700 flex items-center gap-1">
              <CalendarIcon size={12} />
              {formatDate(batch.scheduled_at)}
            </div>
          )}
        </div>
        {batch.status === 'ready' && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onStart();
            }}
            className="text-sm px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700"
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
  batchId?: number;
  batchName?: string;
  batch?: Batch;
}

function RepositoryItem({ repository, onRetry, batchId, batchName, batch }: RepositoryItemProps) {
  const isFailed = repository.status === 'migration_failed' || repository.status === 'dry_run_failed';
  const isDryRunFailed = repository.status === 'dry_run_failed';
  
  // Determine destination with batch-level fallback
  let destination = repository.destination_full_name || repository.full_name;
  let destinationLabel = 'Destination';
  let isCustomDestination = repository.destination_full_name && repository.destination_full_name !== repository.full_name;
  let isBatchDestination = false;
  
  // If repo doesn't have custom destination but batch has destination_org, show that
  if (!repository.destination_full_name && batch?.destination_org) {
    const sourceRepoName = repository.full_name.split('/')[1];
    destination = `${batch.destination_org}/${sourceRepoName}`;
    isBatchDestination = true;
    destinationLabel = 'Destination (from batch)';
  }

  return (
    <div className="flex justify-between items-center p-3 border border-gh-border-default rounded-lg hover:bg-gh-neutral-bg group">
      <Link 
        to={`/repository/${encodeURIComponent(repository.full_name)}`} 
        state={{ fromBatch: true, batchId, batchName }}
        className="flex-1 min-w-0"
      >
        <div className="font-semibold text-gh-text-primary group-hover:text-gh-blue transition-colors">
          {repository.full_name}
        </div>
        <div className="text-sm text-gh-text-secondary mt-1 space-y-0.5">
          <div>
            {formatBytes(repository.total_size || 0)} • {repository.branch_count} branches
          </div>
          <div className="flex items-center gap-1">
            <span className="text-xs">→ {destinationLabel}:</span>
            <span className={`text-xs font-medium ${isCustomDestination ? 'text-blue-600' : isBatchDestination ? 'text-purple-600' : 'text-gray-600'}`}>
              {destination}
            </span>
            {isCustomDestination && (
              <span className="text-xs bg-blue-100 text-blue-700 px-1.5 py-0.5 rounded">custom</span>
            )}
            {isBatchDestination && (
              <span className="text-xs bg-purple-100 text-purple-700 px-1.5 py-0.5 rounded">batch default</span>
            )}
          </div>
        </div>
      </Link>
      <div className="flex items-center gap-3">
        <StatusBadge status={repository.status} size="small" />
        {isFailed && onRetry && (
          <Button
            onClick={(e) => {
              e.preventDefault();
              onRetry();
            }}
            variant="danger"
            size="small"
            title={isDryRunFailed ? 'Re-run the dry run for this repository' : 'Retry the production migration'}
          >
            {isDryRunFailed ? 'Re-run Dry Run' : 'Retry Migration'}
          </Button>
        )}
      </div>
    </div>
  );
}
