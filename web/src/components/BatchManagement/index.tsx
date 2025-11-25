import { useEffect, useState, useRef } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { TextInput, Button, UnderlineNav, ProgressBar, Dialog } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { SearchIcon, PlusIcon, CalendarIcon, GearIcon, ClockIcon, PackageIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { Batch, Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { Pagination } from '../common/Pagination';
import { formatBytes, formatDate } from '../../utils/format';
import { useToast } from '../../contexts/ToastContext';

type BatchTab = 'active' | 'completed';

export function BatchManagement() {
  const navigate = useNavigate();
  const location = useLocation();
  const locationState = location.state as { selectedBatchId?: number } | null;
  const { showSuccess, showError, showWarning } = useToast();
  const [batches, setBatches] = useState<Batch[]>([]);
  const [selectedBatch, setSelectedBatch] = useState<Batch | null>(null);
  const [batchRepositories, setBatchRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<BatchTab>('active');
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 10;
  
  // Ref for dialog focus management
  const dryRunButtonRef = useRef<HTMLButtonElement>(null);

  // Pagination for repository groups in batch detail
  const repoPageSize = 20;
  const [pendingPage, setPendingPage] = useState(1);
  const [inProgressPage, setInProgressPage] = useState(1);
  const [failedPage, setFailedPage] = useState(1);
  const [completePage, setCompletePage] = useState(1);
  const [dryRunCompletePage, setDryRunCompletePage] = useState(1);

  // Delete confirmation dialog state
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [batchToDelete, setBatchToDelete] = useState<Batch | null>(null);

  // Dry run confirmation dialog state
  const [showDryRunDialog, setShowDryRunDialog] = useState(false);
  const [dryRunBatchId, setDryRunBatchId] = useState<number | null>(null);
  const [dryRunOnlyPending, setDryRunOnlyPending] = useState(false);

  // Start migration confirmation dialog state
  const [showStartDialog, setShowStartDialog] = useState(false);
  const [startBatchId, setStartBatchId] = useState<number | null>(null);
  const [startSkipDryRun, setStartSkipDryRun] = useState(false);
  const [startDialogMessage, setStartDialogMessage] = useState('');

  // Retry migration confirmation dialog state
  const [showRetryDialog, setShowRetryDialog] = useState(false);
  const [retryBatchId, setRetryBatchId] = useState<number | null>(null);
  const [retryMessage, setRetryMessage] = useState('');

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
      
      // Reset pagination when switching batches
      setPendingPage(1);
      setInProgressPage(1);
      setFailedPage(1);
      setCompletePage(1);
      setDryRunCompletePage(1);
      
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
      // Fetch all repositories for this batch (without pagination)
      // Don't set limit/offset to get all repos in the batch
      const response = await api.listRepositories({ 
        batch_id: batchId
      });
      const repos = response.repositories || response as any;
      console.log(`Loaded ${repos.length} repositories for batch ${batchId}`);
      setBatchRepositories(repos);
    } catch (error) {
      console.error('Failed to load batch repositories:', error);
    }
  };

  const handleDryRunBatch = (batchId: number, onlyPending = false) => {
    setDryRunBatchId(batchId);
    setDryRunOnlyPending(onlyPending);
    setShowDryRunDialog(true);
  };

  const confirmDryRunBatch = async () => {
    if (!dryRunBatchId) return;

    try {
      await api.dryRunBatch(dryRunBatchId, dryRunOnlyPending);
      showSuccess('Dry run started successfully. Batch will move to "ready" status when all dry runs complete.');
      await loadBatches();
      if (selectedBatch?.id === dryRunBatchId) {
        await loadBatchRepositories(dryRunBatchId);
      }
      setShowDryRunDialog(false);
    } catch (error: any) {
      console.error('Failed to start dry run:', error);
      showError(error.response?.data?.error || 'Failed to start dry run');
      setShowDryRunDialog(false);
    }
  };

  const handleStartBatch = (batchId: number, skipDryRun = false) => {
    const batch = batches.find(b => b.id === batchId);
    
    let message = 'Are you sure you want to start migration for this entire batch?';
    if (batch?.status === 'pending' && !skipDryRun) {
      message = 'This batch has not completed a dry run. Do you want to start migration anyway? (Recommended: Cancel and run dry run first)';
    }
    
    setStartBatchId(batchId);
    setStartSkipDryRun(skipDryRun);
    setStartDialogMessage(message);
    setShowStartDialog(true);
  };

  const confirmStartBatch = async () => {
    if (!startBatchId) return;

    try {
      await api.startBatch(startBatchId, startSkipDryRun);
      showSuccess('Batch migration started successfully');
      await loadBatches();
      if (selectedBatch?.id === startBatchId) {
        await loadBatchRepositories(startBatchId);
      }
      setShowStartDialog(false);
    } catch (error: any) {
      console.error('Failed to start batch:', error);
      showError(error.response?.data?.error || 'Failed to start batch migration');
      setShowStartDialog(false);
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

    setRetryBatchId(selectedBatch.id);
    setRetryMessage(message);
    setShowRetryDialog(true);
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
      await loadBatchRepositories(selectedBatch.id);
      setShowRetryDialog(false);
    } catch (error: any) {
      console.error('Failed to retry batch failures:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to retry failed repositories';
      showError(errorMessage);
      setShowRetryDialog(false);
    }
  };

  const handleRetryRepository = async (repo: Repository) => {
    const isDryRunFailed = repo.status === 'dry_run_failed';
    const actionType = isDryRunFailed ? 'dry run' : 'migration';
    
    try {
      await api.retryRepository(repo.id, isDryRunFailed);
      showSuccess(`Repository queued for ${actionType} retry`);
      if (selectedBatch) {
        await loadBatchRepositories(selectedBatch.id);
      }
    } catch (error: any) {
      console.error('Failed to retry repository:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to retry repository';
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

    setBatchToDelete(batch);
    setShowDeleteDialog(true);
  };

  const confirmDeleteBatch = async () => {
    if (!batchToDelete) return;

    try {
      await api.deleteBatch(batchToDelete.id);
      showSuccess('Batch deleted successfully');
      // Clear selection if we deleted the selected batch
      if (selectedBatch?.id === batchToDelete.id) {
        setSelectedBatch(null);
      }
      await loadBatches();
      setShowDeleteDialog(false);
      setBatchToDelete(null);
    } catch (error: any) {
      console.error('Failed to delete batch:', error);
      showError(error.response?.data?.error || 'Failed to delete batch');
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
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>Batch Management</h1>
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
          <div 
            className="rounded-lg border"
            style={{
              backgroundColor: 'var(--bgColor-default)',
              borderColor: 'var(--borderColor-default)',
              boxShadow: 'var(--shadow-resting-small)'
            }}
          >
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
                <Blankslate>
                  <Blankslate.Visual>
                    <PackageIcon size={48} />
                  </Blankslate.Visual>
                  <Blankslate.Heading>
                    {searchTerm ? 'No batches match your search' : `No ${activeTab} batches`}
                  </Blankslate.Heading>
                  <Blankslate.Description>
                    {searchTerm 
                      ? 'Try a different search term to find batches.'
                      : activeTab === 'active'
                      ? 'Create a batch to group repositories for migration.'
                      : 'Completed batches will appear here once migrations finish.'}
                  </Blankslate.Description>
                  {!searchTerm && activeTab === 'active' && (
                    <Blankslate.PrimaryAction onClick={() => navigate('/batches/new')}>
                      Create New Batch
                    </Blankslate.PrimaryAction>
                  )}
                </Blankslate>
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
            <div 
              className="rounded-lg border p-6"
              style={{
                backgroundColor: 'var(--bgColor-default)',
                borderColor: 'var(--borderColor-default)',
                boxShadow: 'var(--shadow-resting-small)'
              }}
            >
              <div className="flex justify-between items-start mb-6">
                <div className="flex-1">
                  <h2 className="text-xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>{selectedBatch.name}</h2>
                  {selectedBatch.description && (
                    <p className="mt-1" style={{ color: 'var(--fgColor-muted)' }}>{selectedBatch.description}</p>
                  )}
                  <div className="flex items-center gap-3 mt-3">
                    <StatusBadge status={selectedBatch.status} />
                    <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                      {selectedBatch.repository_count} repositories
                    </span>
                    {selectedBatch.created_at && (
                      <>
                        <span style={{ color: 'var(--fgColor-muted)' }}>•</span>
                        <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
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
                          <span style={{ color: 'var(--fgColor-muted)' }}>
                            <GearIcon size={16} />
                          </span>
                          <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>Migration Settings</span>
                        </div>
                        <div className="space-y-2 pl-6">
                          {selectedBatch.destination_org && (
                            <div className="text-sm">
                              <span style={{ color: 'var(--fgColor-muted)' }}>Default Destination:</span>
                              <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-accent)' }}>{selectedBatch.destination_org}</div>
                              <div className="text-xs italic mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>For repos without specific destination</div>
                            </div>
                          )}
                          {selectedBatch.migration_api && selectedBatch.migration_api !== 'GEI' && (
                            <div className="text-sm">
                              <span style={{ color: 'var(--fgColor-muted)' }}>Migration API:</span>
                              <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-default)' }}>
                                {selectedBatch.migration_api === 'ELM' ? 'ELM (Enterprise Live Migrator)' : selectedBatch.migration_api}
                              </div>
                            </div>
                          )}
                          {selectedBatch.exclude_releases && (
                            <div className="text-sm">
                              <span style={{ color: 'var(--fgColor-muted)' }}>Exclude Releases:</span>
                              <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-attention)' }}>Yes</div>
                              <div className="text-xs italic mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>Repo settings can override</div>
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                    {/* Right Column: Schedule & Timestamps */}
                    <div>
                      <div className="flex items-center gap-2 mb-2">
                        <span style={{ color: 'var(--fgColor-muted)' }}>
                          <ClockIcon size={16} />
                        </span>
                        <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>Schedule & Timeline</span>
                      </div>
                      <div className="space-y-2 pl-6">
                        {selectedBatch.scheduled_at && (
                          <div className="text-sm">
                            <span style={{ color: 'var(--fgColor-muted)' }}>Scheduled:</span>
                            <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-default)' }}>
                              {formatDate(selectedBatch.scheduled_at)}
                            </div>
                            {new Date(selectedBatch.scheduled_at) > new Date() && (
                              <div className="text-xs italic mt-0.5" style={{ color: 'var(--fgColor-accent)' }}>Auto-start when ready</div>
                            )}
                          </div>
                        )}
                        {selectedBatch.last_dry_run_at && (
                          <div className="text-sm">
                            <span style={{ color: 'var(--fgColor-muted)' }}>Last Dry Run:</span>
                            <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-default)' }}>
                              {formatDate(selectedBatch.last_dry_run_at)}
                            </div>
                          </div>
                        )}
                        {selectedBatch.last_migration_attempt_at && (
                          <div className="text-sm">
                            <span style={{ color: 'var(--fgColor-muted)' }}>Last Migration:</span>
                            <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-default)' }}>
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
                        <Button
                          ref={dryRunButtonRef}
                          onClick={() => handleDryRunBatch(selectedBatch.id, true)}
                          variant="primary"
                        >
                          Run Dry Run ({groupedRepos.needs_dry_run.length} repos)
                        </Button>
                      )}
                      <button
                        onClick={() => handleStartBatch(selectedBatch.id, true)}
                        className="px-4 py-1.5 rounded-md text-sm font-medium border-0 transition-all cursor-pointer"
                        style={{ 
                          backgroundColor: '#2da44e',
                          color: '#ffffff',
                          fontWeight: 500
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.backgroundColor = '#2c974b';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.backgroundColor = '#2da44e';
                        }}
                      >
                        Skip & Migrate
                      </button>
                    </>
                  )}
                  
                  {selectedBatch.status === 'ready' && (
                    <>
                      <button
                        onClick={() => handleStartBatch(selectedBatch.id)}
                        className="px-4 py-1.5 rounded-md text-sm font-medium border-0 transition-all cursor-pointer"
                        style={{ 
                          backgroundColor: '#2da44e',
                          color: '#ffffff',
                          fontWeight: 500
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.backgroundColor = '#2c974b';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.backgroundColor = '#2da44e';
                        }}
                      >
                        Start Migration
                      </button>
                      {groupedRepos.needs_dry_run.length > 0 ? (
                        <Button
                          onClick={() => handleDryRunBatch(selectedBatch.id, true)}
                          variant="primary"
                          title="Run dry run only for repositories that need it"
                        >
                          Dry Run Pending ({groupedRepos.needs_dry_run.length})
                        </Button>
                      ) : null}
                      <Button
                        onClick={() => handleDryRunBatch(selectedBatch.id, false)}
                        variant="primary"
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
                        <RepositoryItem 
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
                        <RepositoryItem 
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
                        <RepositoryItem 
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
                        <RepositoryItem 
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
      {showDeleteDialog && batchToDelete && (
        <>
          {/* Backdrop */}
          <div 
            className="fixed inset-0 bg-black/50 z-50"
            onClick={() => {
              setShowDeleteDialog(false);
              setBatchToDelete(null);
            }}
          />
          
          {/* Dialog */}
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div 
              className="rounded-lg shadow-xl max-w-md w-full"
              style={{ backgroundColor: 'var(--bgColor-default)' }}
              onClick={(e) => e.stopPropagation()}
            >
              <div className="px-4 py-3 border-b border-gh-border-default">
                <h3 className="text-base font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  Delete Batch
                </h3>
              </div>
              
              <div className="p-4">
                <p className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
                  {batchToDelete.repository_count > 0 ? (
                    <>
                      Are you sure you want to delete batch <strong>"{batchToDelete.name}"</strong>?
                      <br /><br />
                      This will remove <strong>{batchToDelete.repository_count} {batchToDelete.repository_count === 1 ? 'repository' : 'repositories'}</strong> from the batch, making them available for other batches.
                    </>
                  ) : (
                    <>
                      Are you sure you want to delete batch <strong>"{batchToDelete.name}"</strong>?
                    </>
                  )}
                </p>
                <p className="text-sm text-gh-text-muted">
                  This action cannot be undone.
                </p>
              </div>
              
              <div className="px-4 py-3 border-t border-gh-border-default flex justify-end gap-2">
                <Button
                  onClick={() => {
                    setShowDeleteDialog(false);
                    setBatchToDelete(null);
                  }}
                >
                  Cancel
                </Button>
                <Button
                  onClick={confirmDeleteBatch}
                  variant="danger"
                >
                  Delete Batch
                </Button>
              </div>
            </div>
          </div>
        </>
      )}

      {/* Dry Run Confirmation Dialog */}
      {showDryRunDialog && (
        <Dialog
          returnFocusRef={dryRunButtonRef}
          onDismiss={() => setShowDryRunDialog(false)}
          aria-labelledby="dry-run-dialog-header"
        >
          <Dialog.Header id="dry-run-dialog-header">
            Run Dry Run
          </Dialog.Header>
          <div style={{ padding: '16px' }}>
            <p style={{ marginBottom: '12px', fontSize: '14px', color: 'var(--fgColor-default)' }}>
              {dryRunOnlyPending ? (
                <>
                  Run dry run for <strong>pending repositories</strong>?
                </>
              ) : (
                <>
                  Run dry run for <strong>all repositories</strong>?
                </>
              )}
            </p>
            <p style={{ fontSize: '14px', color: 'var(--fgColor-muted)' }}>
              This will validate repositories before migration.
            </p>
          </div>
          <div style={{ 
            padding: '12px 16px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px'
          }}>
            <Button onClick={() => setShowDryRunDialog(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={confirmDryRunBatch}>
              OK
            </Button>
          </div>
        </Dialog>
      )}

      {/* Start Migration Confirmation Dialog */}
      {showStartDialog && (
        <Dialog
          onDismiss={() => setShowStartDialog(false)}
          aria-labelledby="start-dialog-header"
        >
          <Dialog.Header id="start-dialog-header">
            Start Migration
          </Dialog.Header>
          <div style={{ padding: '16px' }}>
            <p style={{ fontSize: '14px', color: 'var(--fgColor-default)' }}>
              {startDialogMessage}
            </p>
          </div>
          <div style={{ 
            padding: '12px 16px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px'
          }}>
            <Button onClick={() => setShowStartDialog(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={confirmStartBatch}>
              Start Migration
            </Button>
          </div>
        </Dialog>
      )}

      {/* Retry Confirmation Dialog */}
      {showRetryDialog && (
        <Dialog
          onDismiss={() => setShowRetryDialog(false)}
          aria-labelledby="retry-dialog-header"
        >
          <Dialog.Header id="retry-dialog-header">
            Retry Failed Repositories
          </Dialog.Header>
          <div style={{ padding: '16px' }}>
            <p style={{ fontSize: '14px', color: 'var(--fgColor-default)' }}>
              {retryMessage}
            </p>
          </div>
          <div style={{ 
            padding: '12px 16px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px'
          }}>
            <Button onClick={() => setShowRetryDialog(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={confirmRetryFailed}>
              Retry
            </Button>
          </div>
        </Dialog>
      )}

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
      className="p-4 rounded-lg border-2 cursor-pointer transition-all"
      style={isSelected
        ? { borderColor: 'var(--accent-emphasis)', backgroundColor: 'var(--accent-subtle)' }
        : { borderColor: 'var(--borderColor-default)', backgroundColor: 'var(--bgColor-default)' }
      }
      onClick={onClick}
    >
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <h3 className="font-medium" style={{ color: 'var(--fgColor-default)' }}>{batch.name}</h3>
          <div className="flex gap-2 mt-2">
            <StatusBadge status={batch.status} size="small" />
            <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>{batch.repository_count} repos</span>
          </div>
          {batch.scheduled_at && (
            <div className="mt-1.5 text-xs flex items-center gap-1" style={{ color: 'var(--fgColor-accent)' }}>
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
            className="text-sm px-3 py-1 rounded border-0 transition-all cursor-pointer"
            style={{ 
              backgroundColor: '#2da44e',
              color: '#ffffff',
              fontWeight: 500
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = '#2c974b';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = '#2da44e';
            }}
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
  let isCustomDestination = false;
  let isBatchDestination = false;
  let isDefaultDestination = false;
  
  // Calculate what the batch default destination would be for this repository
    const sourceRepoName = repository.full_name.split('/')[1];
  const batchDefaultDestination = batch?.destination_org ? `${batch.destination_org}/${sourceRepoName}` : null;
  
  // Check if repo has a custom destination
  if (repository.destination_full_name && repository.destination_full_name !== repository.full_name) {
    // Check if the destination matches the batch default
    if (batchDefaultDestination && repository.destination_full_name === batchDefaultDestination) {
      // Destination matches batch default - show as batch destination
    isBatchDestination = true;
    } else {
      // Destination is truly custom (different from both source and batch default)
      isCustomDestination = true;
    }
  } else if (!repository.destination_full_name && batchDefaultDestination) {
    // If repo doesn't have custom destination but batch has destination_org, show that
    destination = batchDefaultDestination;
    isBatchDestination = true;
  } else if (!repository.destination_full_name) {
    // No custom destination and no batch destination - using default (same as source)
    isDefaultDestination = true;
  }

  return (
    <div className="flex justify-between items-center p-3 border border-gh-border-default rounded-lg hover:bg-gh-neutral-bg group">
      <Link 
        to={`/repository/${encodeURIComponent(repository.full_name)}`} 
        state={{ fromBatch: true, batchId, batchName }}
        className="flex-1 min-w-0"
      >
        <div className="font-semibold transition-colors" style={{ color: 'var(--fgColor-default)' }}>
          {repository.full_name}
        </div>
        <div className="text-sm mt-1 space-y-0.5" style={{ color: 'var(--fgColor-muted)' }}>
          <div>
            {formatBytes(repository.total_size || 0)} • {repository.branch_count} branches
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-xs">→ Destination:</span>
            <span 
              className="text-xs font-medium"
              style={{ color: isCustomDestination ? 'var(--fgColor-accent)' : isBatchDestination ? 'var(--fgColor-attention)' : 'var(--fgColor-muted)' }}
            >
              {destination}
            </span>
            {isCustomDestination && (
              <span 
                className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold uppercase tracking-wide"
                style={{ 
                  backgroundColor: '#0969da', 
                  color: '#ffffff',
                  border: '1px solid #0969da'
                }}
              >
                Custom
              </span>
            )}
            {isBatchDestination && (
              <span 
                className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold uppercase tracking-wide"
                style={{ 
                  backgroundColor: '#FB8500', 
                  color: '#ffffff',
                  border: '1px solid #FB8500'
                }}
              >
                Batch Default
              </span>
            )}
            {isDefaultDestination && (
              <span 
                className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold uppercase tracking-wide"
                style={{ 
                  backgroundColor: '#6e7781', 
                  color: '#ffffff',
                  border: '1px solid #6e7781'
                }}
              >
                Default
              </span>
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
