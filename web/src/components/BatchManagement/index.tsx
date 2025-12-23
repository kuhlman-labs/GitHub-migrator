import { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import { useNavigate, useLocation, useSearchParams } from 'react-router-dom';
import { Button, ProgressBar } from '@primer/react';
import { PlusIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { Batch, Repository } from '../../types';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Pagination } from '../common/Pagination';
import { ConfirmationDialog } from '../common/ConfirmationDialog';
import { useToast } from '../../contexts/ToastContext';
import { useBatches, useBatchRepositories } from '../../hooks/useQueries';
import { useBatchUpdateRepositoryStatus } from '../../hooks/useMutations';
import { useDialogState } from '../../hooks/useDialogState';
import { BatchListPanel } from './BatchListPanel';
import { BatchDetailHeader } from './BatchDetailHeader';
import { BatchRepositoryItem } from './BatchRepositoryItem';

// Dialog data types
interface DryRunDialogData {
  batchId: number;
  onlyPending: boolean;
}

interface StartDialogData {
  batchId: number;
  skipDryRun: boolean;
  message: string;
}

interface RetryDialogData {
  message: string;
}

type BatchTab = 'active' | 'completed';

export function BatchManagement() {
  const navigate = useNavigate();
  const location = useLocation();
  const locationState = location.state as { selectedBatchId?: number; refreshData?: boolean } | null;
  const { showSuccess, showError, showWarning } = useToast();
  const [searchParams] = useSearchParams();
  const [selectedBatchId, setSelectedBatchId] = useState<number | null>(null);
  const [activeTab, setActiveTab] = useState<BatchTab>('active');
  const searchTerm = searchParams.get('search') || '';
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 10;

  // Pagination for repository groups in batch detail
  const repoPageSize = 20;
  const [pendingPage, setPendingPage] = useState(1);
  const [inProgressPage, setInProgressPage] = useState(1);
  const [failedPage, setFailedPage] = useState(1);
  const [completePage, setCompletePage] = useState(1);
  const [dryRunCompletePage, setDryRunCompletePage] = useState(1);

  // Dialog state management using useDialogState hook
  const deleteDialog = useDialogState<Batch>();
  const dryRunDialog = useDialogState<DryRunDialogData>();
  const startDialog = useDialogState<StartDialogData>();
  const retryDialog = useDialogState<RetryDialogData>();
  const rollbackDialog = useDialogState<Batch>();
  
  // Mutations
  const batchUpdateStatus = useBatchUpdateRepositoryStatus();
  
  // Ref for dry run button focus management
  const dryRunButtonRef = useRef<HTMLButtonElement>(null);

  // React Query for batches - always poll every 30 seconds
  const { 
    data: batches = [], 
    isLoading: loading, 
    isFetching: refreshing,
    refetch: refetchBatches
  } = useBatches({ refetchInterval: 30000 });

  // Find selected batch from batches list
  const selectedBatch = useMemo(() => {
    if (!selectedBatchId) return null;
    return batches.find((b) => b.id === selectedBatchId) || null;
  }, [batches, selectedBatchId]);

  // Determine polling interval for batch repositories based on selected batch status
  const batchRepoPollingInterval = useMemo(() => {
    if (!selectedBatch) return false;
    const shouldPoll = 
      selectedBatch.status === 'in_progress' || 
      selectedBatch.status === 'ready' ||
      (selectedBatch.scheduled_at && new Date(selectedBatch.scheduled_at) > new Date());
    
    if (!shouldPoll) return false;
    return selectedBatch.status === 'in_progress' ? 5000 : 15000;
  }, [selectedBatch]);

  // React Query for batch repositories - polls based on batch status
  const { 
    data: batchRepoData,
    refetch: refetchBatchRepositories
  } = useBatchRepositories(selectedBatchId, { 
    refetchInterval: batchRepoPollingInterval 
  });
  
  const batchRepositories: Repository[] = batchRepoData?.repositories || [];

  // Handle immediate refresh when navigating back from create/edit
  useEffect(() => {
    if (locationState?.refreshData) {
      refetchBatches();
      // Clear the state to prevent refresh on subsequent renders
      navigate(location.pathname, { replace: true, state: {} });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [locationState?.refreshData]);

  // Handle location state to auto-select batch when navigating from repository detail
  useEffect(() => {
    if (locationState?.selectedBatchId && batches.length > 0 && !selectedBatchId) {
      setSelectedBatchId(locationState.selectedBatchId);
    }
  }, [locationState, batches, selectedBatchId]);

  // Reset pagination when switching batches
  useEffect(() => {
    if (selectedBatchId) {
      setPendingPage(1);
      setInProgressPage(1);
      setFailedPage(1);
      setCompletePage(1);
      setDryRunCompletePage(1);
    }
  }, [selectedBatchId]);

  // Helper function to refresh after mutations
  const refreshData = useCallback(async () => {
    await refetchBatches();
    if (selectedBatchId) {
      await refetchBatchRepositories();
    }
  }, [refetchBatches, refetchBatchRepositories, selectedBatchId]);

  const handleDryRunBatch = (batchId: number, onlyPending = false) => {
    dryRunDialog.open({ batchId, onlyPending });
  };

  const confirmDryRunBatch = async () => {
    const data = dryRunDialog.data;
    if (!data) return;

    try {
      await api.dryRunBatch(data.batchId, data.onlyPending);
      showSuccess('Dry run started successfully. Batch will move to "ready" status when all dry runs complete.');
      await refreshData();
      dryRunDialog.close();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      showError(err.response?.data?.error || 'Failed to start dry run');
      dryRunDialog.close();
    }
  };

  const handleStartBatch = (batchId: number, skipDryRun = false) => {
    const batch = batches.find(b => b.id === batchId);
    
    let message = 'Are you sure you want to start migration for this entire batch?';
    if (batch?.status === 'pending' && !skipDryRun) {
      message = 'This batch has not completed a dry run. Do you want to start migration anyway? (Recommended: Cancel and run dry run first)';
    }
    
    startDialog.open({ batchId, skipDryRun, message });
  };

  const confirmStartBatch = async () => {
    const data = startDialog.data;
    if (!data) return;

    try {
      await api.startBatch(data.batchId, data.skipDryRun);
      showSuccess('Batch migration started successfully');
      await refreshData();
      startDialog.close();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      showError(err.response?.data?.error || 'Failed to start batch migration');
      startDialog.close();
    }
  };

  const handleRetryFailed = () => {
    if (!selectedBatch) return;

    const failedRepos = batchRepositories.filter(
      (r) => r.status === 'migration_failed' || r.status === 'dry_run_failed'
    );

    if (failedRepos.length === 0) return;

    const dryRunFailedCount = failedRepos.filter(r => r.status === 'dry_run_failed').length;
    const migrationFailedCount = failedRepos.filter(r => r.status === 'migration_failed').length;
    
    let message = '';
    if (dryRunFailedCount > 0 && migrationFailedCount > 0) {
      message = `Retry ${dryRunFailedCount} failed dry run(s) and ${migrationFailedCount} failed migration(s)?`;
    } else if (dryRunFailedCount > 0) {
      message = `Re-run dry run for ${dryRunFailedCount} failed repositories?`;
    } else {
      message = `Retry migration for ${migrationFailedCount} failed repositories?`;
    }

    retryDialog.open({ message });
  };

  const confirmRetryFailed = async () => {
    if (!selectedBatch) return;

    const failedRepos = batchRepositories.filter(
      (r) => r.status === 'migration_failed' || r.status === 'dry_run_failed'
    );

    try {
      // Retry each repository individually with the correct dry_run flag
      for (const repo of failedRepos) {
        const isDryRunFailed = repo.status === 'dry_run_failed';
        await api.retryRepository(repo.id, isDryRunFailed);
      }
      showSuccess(`Queued ${failedRepos.length} repositories for retry`);
      await refreshData();
      retryDialog.close();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      const errorMessage = err.response?.data?.error || err.message || 'Failed to retry failed repositories';
      showError(errorMessage);
      retryDialog.close();
    }
  };

  const handleRetryRepository = async (repo: Repository) => {
    const isDryRunFailed = repo.status === 'dry_run_failed';
    const actionType = isDryRunFailed ? 'dry run' : 'migration';
    
    try {
      await api.retryRepository(repo.id, isDryRunFailed);
      showSuccess(`Repository queued for ${actionType} retry`);
      await refreshData();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      const errorMessage = err.response?.data?.error || err.message || 'Failed to retry repository';
      showError(errorMessage);
    }
  };

  const handleCreateBatch = () => {
    navigate('/batches/new');
  };

  const handleEditBatch = (batch: Batch) => {
    navigate(`/batches/${batch.id}/edit`);
  };

  const handleDeleteBatch = (batch: Batch) => {
    if (batch.status === 'in_progress') {
      showWarning('Cannot delete a batch that is currently in progress.');
      return;
    }

    deleteDialog.open(batch);
  };

  const handleRollbackBatch = (batch: Batch) => {
    rollbackDialog.open(batch);
  };

  const confirmRollbackBatch = async () => {
    const batch = rollbackDialog.data;
    if (!batch || !batchRepositories.length) return;

    try {
      const repositoryIds = batchRepositories.map(r => r.id);
      await batchUpdateStatus.mutateAsync({
        repositoryIds,
        action: 'rollback',
        reason: `Batch rollback: ${batch.name}`,
      });
      showSuccess(`Successfully rolled back ${repositoryIds.length} repositories in batch "${batch.name}"`);
      rollbackDialog.close();
      await refetchBatches();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      const errorMessage = err.response?.data?.error || err.message || 'Failed to rollback batch';
      showError(errorMessage);
    }
  };

  const confirmDeleteBatch = async () => {
    const batch = deleteDialog.data;
    if (!batch) return;

    try {
      await api.deleteBatch(batch.id);
      showSuccess('Batch deleted successfully');
      // Clear selection if we deleted the selected batch
      if (selectedBatchId === batch.id) {
        setSelectedBatchId(null);
      }
      await refetchBatches();
      deleteDialog.close();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      showError(err.response?.data?.error || 'Failed to delete batch');
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

  // Helper to paginate an array
  const paginateArray = <T,>(items: T[], page: number, pageSize: number) => {
    const startIndex = (page - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    return items.slice(startIndex, endIndex);
  };

  const progress = selectedBatch ? getBatchProgress(selectedBatch, batchRepositories) : null;
  const groupedRepos = groupReposByStatus(batchRepositories);

  // Reset page when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [activeTab, searchTerm]);

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={refreshing && !loading} />
      
      <div className="flex justify-between items-start mb-6">
        <div>
          <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>Batch Management</h1>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Group repositories into batches for coordinated migration
          </p>
        </div>
        <div className="flex items-center gap-4">
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
          <BatchListPanel
            batches={batches}
            loading={loading}
            activeTab={activeTab}
            onTabChange={setActiveTab}
            selectedBatchId={selectedBatchId}
            onSelectBatch={setSelectedBatchId}
            onStartBatch={handleStartBatch}
            searchTerm={searchTerm}
            currentPage={currentPage}
            pageSize={pageSize}
            onPageChange={setCurrentPage}
          />
        </div>

        {/* Batch Detail */}
        <div className="lg:col-span-2">
          {selectedBatch ? (
            <div 
              className="rounded-lg border p-6"
              style={{
                backgroundColor: 'var(--bgColor-default)',
                borderColor: 'var(--borderColor-default)',
                boxShadow: 'var(--shadow-resting-small)'
              }}
            >
              <BatchDetailHeader
                batch={selectedBatch}
                batchRepositories={batchRepositories}
                onEdit={handleEditBatch}
                onDelete={handleDeleteBatch}
                onDryRun={handleDryRunBatch}
                onStart={handleStartBatch}
                onRetryFailed={handleRetryFailed}
                onRollback={handleRollbackBatch}
                dryRunButtonRef={dryRunButtonRef}
              />

              {/* Progress Bar */}
              {progress && progress.total > 0 && (
                <div 
                  className="mb-6 p-4 rounded-lg"
                  style={{ backgroundColor: 'var(--bgColor-muted)' }}
                >
                  <div className="flex justify-between text-sm mb-2" style={{ color: 'var(--fgColor-muted)' }}>
                    <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>Progress</span>
                    <span>
                      <span className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                        {progress.completed}
                      </span>
                      {' '}/{' '}
                      {progress.total}
                      {' '}({progress.percentage}%)
                    </span>
                  </div>
                  <ProgressBar 
                    progress={progress.percentage} 
                    aria-label={`${progress.completed} of ${progress.total} repositories completed`}
                    bg="success.emphasis"
                  />
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
                      {paginateArray(groupedRepos.failed, failedPage, repoPageSize).map((repo) => (
                        <BatchRepositoryItem
                          key={repo.id}
                          repository={repo}
                          onRetry={() => handleRetryRepository(repo)}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
                      ))}
                    </div>
                    {groupedRepos.failed.length > repoPageSize && (
                      <div className="mt-4">
                        <Pagination
                          currentPage={failedPage}
                          totalItems={groupedRepos.failed.length}
                          pageSize={repoPageSize}
                          onPageChange={setFailedPage}
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* In Progress Repositories */}
                {groupedRepos.in_progress.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium mb-3" style={{ color: 'var(--fgColor-accent)' }}>
                      In Progress ({groupedRepos.in_progress.length})
                    </h3>
                    <div className="space-y-2">
                      {paginateArray(groupedRepos.in_progress, inProgressPage, repoPageSize).map((repo) => (
                        <BatchRepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
                      ))}
                    </div>
                    {groupedRepos.in_progress.length > repoPageSize && (
                      <div className="mt-4">
                        <Pagination
                          currentPage={inProgressPage}
                          totalItems={groupedRepos.in_progress.length}
                          pageSize={repoPageSize}
                          onPageChange={setInProgressPage}
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* Completed Repositories */}
                {groupedRepos.complete.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium mb-3" style={{ color: 'var(--fgColor-success)' }}>
                      Completed ({groupedRepos.complete.length})
                    </h3>
                    <div className="space-y-2">
                      {paginateArray(groupedRepos.complete, completePage, repoPageSize).map((repo) => (
                        <BatchRepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
                      ))}
                    </div>
                    {groupedRepos.complete.length > repoPageSize && (
                      <div className="mt-4">
                        <Pagination
                          currentPage={completePage}
                          totalItems={groupedRepos.complete.length}
                          pageSize={repoPageSize}
                          onPageChange={setCompletePage}
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* Dry Run Complete (Ready for Migration) */}
                {groupedRepos.dry_run_complete.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium mb-3" style={{ color: 'var(--fgColor-accent)' }}>
                      Ready for Migration ({groupedRepos.dry_run_complete.length})
                    </h3>
                    <div className="space-y-2">
                      {paginateArray(groupedRepos.dry_run_complete, dryRunCompletePage, repoPageSize).map((repo) => (
                        <BatchRepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
                      ))}
                    </div>
                    {groupedRepos.dry_run_complete.length > repoPageSize && (
                      <div className="mt-4">
                        <Pagination
                          currentPage={dryRunCompletePage}
                          totalItems={groupedRepos.dry_run_complete.length}
                          pageSize={repoPageSize}
                          onPageChange={setDryRunCompletePage}
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* Pending Repositories */}
                {groupedRepos.pending.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium mb-3" style={{ color: 'var(--fgColor-default)' }}>
                      Pending ({groupedRepos.pending.length})
                    </h3>
                    <div className="space-y-2">
                      {paginateArray(groupedRepos.pending, pendingPage, repoPageSize).map((repo) => (
                        <BatchRepositoryItem 
                          key={repo.id} 
                          repository={repo}
                          batchId={selectedBatch.id}
                          batchName={selectedBatch.name}
                          batch={selectedBatch}
                        />
                      ))}
                    </div>
                    {groupedRepos.pending.length > repoPageSize && (
                      <div className="mt-4">
                        <Pagination
                          currentPage={pendingPage}
                          totalItems={groupedRepos.pending.length}
                          pageSize={repoPageSize}
                          onPageChange={setPendingPage}
                        />
                      </div>
                    )}
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
            <div className="rounded-lg shadow-sm p-6 text-center" style={{ backgroundColor: 'var(--bgColor-default)', color: 'var(--fgColor-muted)' }}>
              Select a batch to view details
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={deleteDialog.isOpen && deleteDialog.data !== null}
        title="Delete Batch"
        message={
          deleteDialog.data ? (
            <>
              Are you sure you want to delete batch <strong>"{deleteDialog.data.name}"</strong>?
              {deleteDialog.data.repository_count > 0 && (
                <>
                  <br /><br />
                  This will remove <strong>{deleteDialog.data.repository_count} {deleteDialog.data.repository_count === 1 ? 'repository' : 'repositories'}</strong> from the batch, making them available for other batches.
                </>
              )}
              <br /><br />
              <span style={{ color: 'var(--fgColor-muted)' }}>This action cannot be undone.</span>
            </>
          ) : ''
        }
        confirmLabel="Delete Batch"
        variant="danger"
        onConfirm={confirmDeleteBatch}
        onCancel={deleteDialog.close}
      />

      {/* Dry Run Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={dryRunDialog.isOpen}
        title="Run Dry Run"
        message={
          <>
            {dryRunDialog.data?.onlyPending ? (
              <>Run dry run for <strong>pending repositories</strong>?</>
            ) : (
              <>Run dry run for <strong>all repositories</strong>?</>
            )}
            <br /><br />
            <span style={{ color: 'var(--fgColor-muted)' }}>
              This will validate repositories before migration.
            </span>
          </>
        }
        confirmLabel="OK"
        onConfirm={confirmDryRunBatch}
        onCancel={dryRunDialog.close}
      />

      {/* Start Migration Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={startDialog.isOpen}
        title="Start Migration"
        message={startDialog.data?.message || ''}
        confirmLabel="Start Migration"
        onConfirm={confirmStartBatch}
        onCancel={startDialog.close}
      />

      {/* Retry Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={retryDialog.isOpen}
        title="Retry Failed Repositories"
        message={retryDialog.data?.message || ''}
        confirmLabel="Retry"
        onConfirm={confirmRetryFailed}
        onCancel={retryDialog.close}
      />

      {/* Rollback Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={rollbackDialog.isOpen && rollbackDialog.data !== null}
        title="Rollback Batch"
        message={
          rollbackDialog.data ? (
            <>
              Are you sure you want to rollback batch <strong>"{rollbackDialog.data.name}"</strong>?
              <br /><br />
              This will reset <strong>{batchRepositories.length} {batchRepositories.length === 1 ? 'repository' : 'repositories'}</strong> to their pre-migration state, allowing them to be migrated again.
              <br /><br />
              <span style={{ color: 'var(--fgColor-attention)' }}>
                ⚠️ The migrated repositories in the destination will NOT be deleted. You may need to manually clean them up.
              </span>
            </>
          ) : ''
        }
        confirmLabel="Rollback"
        variant="danger"
        onConfirm={confirmRollbackBatch}
        onCancel={rollbackDialog.close}
      />

    </div>
  );
}
