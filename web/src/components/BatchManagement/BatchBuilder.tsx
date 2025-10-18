import { useState, useEffect } from 'react';
import type { Repository, Batch, RepositoryFilters } from '../../types';
import { api } from '../../services/api';
import { RepositoryFilters as FilterComponent } from './RepositoryFilters';
import { RepositoryGroup } from './RepositoryGroup';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { formatBytes } from '../../utils/format';

interface BatchBuilderProps {
  batch?: Batch; // If provided, we're editing; otherwise creating
  onClose: () => void;
  onSuccess: () => void;
}

export function BatchBuilder({ batch, onClose, onSuccess }: BatchBuilderProps) {
  const isEditMode = !!batch;

  // Batch metadata
  const [batchName, setBatchName] = useState(batch?.name || '');
  const [batchDescription, setBatchDescription] = useState(batch?.description || '');
  const [scheduledAt, setScheduledAt] = useState(
    batch?.scheduled_at ? new Date(batch.scheduled_at).toISOString().slice(0, 16) : ''
  );

  // Repository lists
  const [availableRepos, setAvailableRepos] = useState<Repository[]>([]);
  const [selectedRepoIds, setSelectedRepoIds] = useState<Set<number>>(new Set());
  const [currentBatchRepos, setCurrentBatchRepos] = useState<Repository[]>([]);

  // Filters and pagination
  const [filters, setFilters] = useState<RepositoryFilters>({
    available_for_batch: true,
    limit: 50,
    offset: 0,
  });
  const [totalAvailable, setTotalAvailable] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);

  // Loading states
  const [loading, setLoading] = useState(false);
  const [availableLoading, setAvailableLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load current batch repositories in edit mode
  useEffect(() => {
    if (isEditMode && batch) {
      loadCurrentBatchRepos();
    }
  }, [isEditMode, batch]);

  // Load available repositories
  useEffect(() => {
    loadAvailableRepos();
  }, [filters]);

  const loadCurrentBatchRepos = async () => {
    if (!batch) return;
    try {
      const response = await api.listRepositories({ batch_id: batch.id });
      setCurrentBatchRepos(response.repositories || response as any);
    } catch (err) {
      console.error('Failed to load current batch repos:', err);
    }
  };

  const loadAvailableRepos = async () => {
    setAvailableLoading(true);
    try {
      const response = await api.listRepositories(filters);
      setAvailableRepos(response.repositories || response as any);
      setTotalAvailable(response.total || (response.repositories?.length || 0));
    } catch (err) {
      console.error('Failed to load available repos:', err);
      setError('Failed to load repositories');
    } finally {
      setAvailableLoading(false);
    }
  };

  const handleFilterChange = (newFilters: RepositoryFilters) => {
    setFilters({ ...newFilters, available_for_batch: true, limit: 50, offset: 0 });
    setCurrentPage(1);
  };

  const handleClearFilters = () => {
    setFilters({ available_for_batch: true, limit: 50, offset: 0 });
    setCurrentPage(1);
  };

  const handleToggleRepo = (repoId: number) => {
    const newSelected = new Set(selectedRepoIds);
    if (newSelected.has(repoId)) {
      newSelected.delete(repoId);
    } else {
      newSelected.add(repoId);
    }
    setSelectedRepoIds(newSelected);
  };

  const handleToggleAllInGroup = (repoIds: number[]) => {
    const newSelected = new Set(selectedRepoIds);
    const allSelected = repoIds.every((id) => newSelected.has(id));
    
    if (allSelected) {
      repoIds.forEach((id) => newSelected.delete(id));
    } else {
      repoIds.forEach((id) => newSelected.add(id));
    }
    
    setSelectedRepoIds(newSelected);
  };

  const handleAddSelected = async () => {
    if (selectedRepoIds.size === 0) return;

    const selectedRepos = availableRepos.filter((r) => selectedRepoIds.has(r.id));
    setCurrentBatchRepos([...currentBatchRepos, ...selectedRepos]);
    setSelectedRepoIds(new Set());
    
    // Refresh available repos to exclude newly added ones
    await loadAvailableRepos();
  };

  const handleRemoveRepo = (repoId: number) => {
    setCurrentBatchRepos(currentBatchRepos.filter((r) => r.id !== repoId));
  };

  const handleClearAll = () => {
    if (confirm('Remove all repositories from this batch?')) {
      setCurrentBatchRepos([]);
    }
  };

  const handleSubmit = async (startImmediately: boolean) => {
    if (!batchName.trim()) {
      setError('Batch name is required');
      return;
    }

    if (currentBatchRepos.length === 0) {
      setError('Please add at least one repository to the batch');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      let batchId: number;

      if (isEditMode && batch) {
        // Update existing batch
        await api.updateBatch(batch.id, {
          name: batchName.trim(),
          description: batchDescription.trim() || undefined,
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        });
        
        // Update repositories - add new ones, remove old ones
        const currentIds = new Set(currentBatchRepos.map((r) => r.id));
        const originalResponse = await api.listRepositories({ batch_id: batch.id });
        const originalRepos = originalResponse.repositories || originalResponse as any;
        const originalIds = new Set(originalRepos.map((r: Repository) => r.id));
        
        const toAdd = Array.from(currentIds).filter((id) => !originalIds.has(id));
        const toRemove = Array.from(originalIds).filter((id) => !currentIds.has(id));
        
        if (toAdd.length > 0) {
          await api.addRepositoriesToBatch(batch.id, toAdd);
        }
        if (toRemove.length > 0) {
          await api.removeRepositoriesFromBatch(batch.id, toRemove);
        }
        
        batchId = batch.id;
      } else {
        // Create new batch
        const newBatch = await api.createBatch({
          name: batchName.trim(),
          description: batchDescription.trim() || undefined,
          type: 'batch',
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        });
        
        batchId = newBatch.id;
        
        // Add repositories to batch
        if (currentBatchRepos.length > 0) {
          await api.addRepositoriesToBatch(batchId, currentBatchRepos.map((r) => r.id));
        }
      }

      // Start batch immediately if requested
      if (startImmediately) {
        await api.startBatch(batchId);
      }

      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save batch');
    } finally {
      setLoading(false);
    }
  };

  const groupReposByOrg = (repos: Repository[]) => {
    const groups: Record<string, Repository[]> = {};
    repos.forEach((repo) => {
      const org = repo.full_name.split('/')[0];
      if (!groups[org]) groups[org] = [];
      groups[org].push(repo);
    });
    return groups;
  };

  const availableGroups = groupReposByOrg(availableRepos.filter((r) => !currentBatchRepos.some((cr) => cr.id === r.id)));
  const currentGroups = groupReposByOrg(currentBatchRepos);

  const totalSize = currentBatchRepos.reduce((sum, repo) => sum + (repo.total_size || 0), 0);
  const pageSize = filters.limit || 50;
  const totalPages = Math.ceil(totalAvailable / pageSize);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    setFilters({ ...filters, offset: (page - 1) * pageSize });
  };

  return (
    <div className="bg-white rounded-lg shadow h-full flex flex-col">
      {/* Content */}
      <div className="flex-1 overflow-hidden flex">
          {/* Left Panel - Available Repositories */}
          <div className="w-1/2 border-r flex flex-col p-6 overflow-hidden">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Available Repositories</h3>
            
            <FilterComponent
              filters={filters}
              onChange={handleFilterChange}
              onClear={handleClearFilters}
            />

            <div className="mt-4 flex-1 overflow-y-auto space-y-4">
              {availableLoading ? (
                <div className="flex items-center justify-center py-12">
                  <LoadingSpinner />
                </div>
              ) : Object.keys(availableGroups).length === 0 ? (
                <div className="text-center py-12 text-gray-500">
                  No repositories available
                </div>
              ) : (
                Object.entries(availableGroups).map(([org, repos]) => (
                  <RepositoryGroup
                    key={org}
                    organization={org}
                    repositories={repos}
                    selectedIds={selectedRepoIds}
                    onToggle={handleToggleRepo}
                    onToggleAll={handleToggleAllInGroup}
                  />
                ))
              )}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t pt-4">
                <div className="text-sm text-gray-600">
                  Page {currentPage} of {totalPages} ({totalAvailable} total)
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handlePageChange(currentPage - 1)}
                    disabled={currentPage === 1}
                    className="px-3 py-1 border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => handlePageChange(currentPage + 1)}
                    disabled={currentPage === totalPages}
                    className="px-3 py-1 border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}

            <button
              onClick={handleAddSelected}
              disabled={selectedRepoIds.size === 0 || loading}
              className="mt-4 w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Add Selected ({selectedRepoIds.size})
            </button>
          </div>

          {/* Right Panel - Selected Repositories & Batch Info */}
          <div className="w-1/2 flex flex-col p-6 overflow-hidden">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-medium text-gray-900">
                Selected Repositories ({currentBatchRepos.length})
              </h3>
              {currentBatchRepos.length > 0 && (
                <button
                  onClick={handleClearAll}
                  className="text-sm text-red-600 hover:text-red-700"
                >
                  Clear All
                </button>
              )}
            </div>

            <div className="flex-1 overflow-y-auto space-y-4 mb-6">
              {currentBatchRepos.length === 0 ? (
                <div className="text-center py-12 text-gray-500">
                  No repositories selected. Add repositories from the left panel.
                </div>
              ) : (
                Object.entries(currentGroups).map(([org, repos]) => (
                  <div key={org} className="border border-gray-200 rounded-lg overflow-hidden">
                    <div className="bg-gray-50 px-4 py-2 border-b">
                      <span className="font-medium text-gray-900">{org}</span>
                      <span className="ml-2 text-sm text-gray-600">({repos.length})</span>
                    </div>
                    <div className="divide-y">
                      {repos.map((repo) => (
                        <div key={repo.id} className="p-3 flex items-center justify-between hover:bg-gray-50">
                          <div className="flex-1 min-w-0">
                            <div className="font-medium text-gray-900 truncate">{repo.full_name}</div>
                            <div className="text-xs text-gray-600 mt-1">
                              {formatBytes(repo.total_size || 0)} • {repo.branch_count} branches
                            </div>
                          </div>
                          <button
                            onClick={() => handleRemoveRepo(repo.id)}
                            className="ml-2 text-red-600 hover:text-red-700"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      ))}
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* Batch Metadata */}
            <div className="border-t pt-6 space-y-4">
              <div className="bg-gray-50 p-4 rounded-lg">
                <div className="text-sm text-gray-600">Total Size</div>
                <div className="text-2xl font-semibold text-gray-900">{formatBytes(totalSize)}</div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Batch Name *
                </label>
                <input
                  type="text"
                  value={batchName}
                  onChange={(e) => setBatchName(e.target.value)}
                  placeholder="e.g., Wave 1, Q1 Migration"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  disabled={loading}
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Description
                </label>
                <textarea
                  value={batchDescription}
                  onChange={(e) => setBatchDescription(e.target.value)}
                  placeholder="Optional description"
                  rows={2}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  disabled={loading}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Scheduled Date (Optional)
                </label>
                <input
                  type="datetime-local"
                  value={scheduledAt}
                  onChange={(e) => setScheduledAt(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  disabled={loading}
                />
              </div>

              {error && (
                <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg text-sm">
                  {error}
                </div>
              )}

              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={onClose}
                  disabled={loading}
                  className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleSubmit(false)}
                  disabled={loading}
                  className="flex-1 px-4 py-2 border border-gh-border-default text-gh-text-primary rounded-md text-sm font-medium hover:bg-gh-neutral-bg disabled:opacity-50"
                >
                  {loading ? 'Saving...' : isEditMode ? 'Update' : 'Create'}
                </button>
                {!isEditMode && (
                  <button
                    onClick={() => handleSubmit(true)}
                    disabled={loading}
                    className="flex-1 px-4 py-2 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50"
                  >
                    Create & Start
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>
    </div>
  );
}

