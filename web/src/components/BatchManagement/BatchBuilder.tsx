import { useState, useEffect } from 'react';
import type { Repository, Batch, RepositoryFilters } from '../../types';
import { api } from '../../services/api';
import { FilterSidebar } from './FilterSidebar';
import { ActiveFilterPills } from './ActiveFilterPills';
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

  // UI state
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

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

  const handleRemoveFilter = (filterKey: keyof RepositoryFilters) => {
    const newFilters = { ...filters };
    
    // Handle size range specially
    if (filterKey === 'min_size') {
      delete newFilters.min_size;
      delete newFilters.max_size;
    } else {
      delete newFilters[filterKey];
    }
    
    setFilters({ ...newFilters, available_for_batch: true, limit: 50, offset: 0 });
    setCurrentPage(1);
  };

  const handleQuickFilter = (complexity?: string[]) => {
    if (!complexity) {
      handleClearFilters();
    } else {
      setFilters({
        ...filters,
        complexity,
        available_for_batch: true,
        limit: 50,
        offset: 0,
      });
      setCurrentPage(1);
    }
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
    <div className="bg-gray-50 h-full flex overflow-hidden">
      {/* Filter Sidebar */}
      <FilterSidebar
        filters={filters}
        onChange={handleFilterChange}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
      />

      {/* Middle Panel - Available Repositories */}
      <div className={`flex-1 min-w-0 flex flex-col bg-white border-r border-gray-200 overflow-hidden transition-all duration-300 ${currentBatchRepos.length > 0 ? 'lg:w-[45%]' : 'lg:w-[60%]'}`}>
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-3">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">Available Repositories</h3>
              <p className="text-sm text-gray-600 mt-0.5">
                {totalAvailable} repositories available
              </p>
            </div>
          </div>

          {/* Quick Filter Buttons */}
          <div className="flex flex-wrap gap-2 mb-3">
            <button
              onClick={() => handleQuickFilter()}
              className={`flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
                !filters.complexity
                  ? 'bg-blue-600 text-white border-blue-600'
                  : 'border-gray-300 text-gray-700 hover:bg-gray-50'
              }`}
            >
              All
            </button>
            <button
              onClick={() => handleQuickFilter(['simple'])}
              className={`flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
                Array.isArray(filters.complexity) && filters.complexity.length === 1 && filters.complexity[0] === 'simple'
                  ? 'bg-green-600 text-white border-green-600'
                  : 'border-gray-300 text-gray-700 hover:bg-gray-50'
              }`}
            >
              Simple
            </button>
            <button
              onClick={() => handleQuickFilter(['medium'])}
              className={`flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
                Array.isArray(filters.complexity) && filters.complexity.length === 1 && filters.complexity[0] === 'medium'
                  ? 'bg-yellow-600 text-white border-yellow-600'
                  : 'border-gray-300 text-gray-700 hover:bg-gray-50'
              }`}
            >
              Medium
            </button>
            <button
              onClick={() => handleQuickFilter(['complex', 'very_complex'])}
              className={`flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
                Array.isArray(filters.complexity) && filters.complexity.includes('complex')
                  ? 'bg-orange-600 text-white border-orange-600'
                  : 'border-gray-300 text-gray-700 hover:bg-gray-50'
              }`}
            >
              Complex
            </button>
          </div>

          {/* Active Filter Pills */}
          <ActiveFilterPills
            filters={filters}
            onRemoveFilter={handleRemoveFilter}
            onClearAll={handleClearFilters}
          />
        </div>

        {/* Repository List */}
        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {availableLoading ? (
            <div className="flex items-center justify-center py-12">
              <LoadingSpinner />
            </div>
          ) : Object.keys(availableGroups).length === 0 ? (
            <div className="text-center py-12">
              <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
              </svg>
              <p className="mt-2 text-sm text-gray-500">No repositories available</p>
              <p className="text-xs text-gray-400 mt-1">Try adjusting your filters</p>
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
          <div className="border-t border-gray-200 px-4 py-3 bg-white">
            <div className="flex items-center justify-between">
              <div className="text-sm text-gray-600">
                Page {currentPage} of {totalPages}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => handlePageChange(currentPage - 1)}
                  disabled={currentPage === 1}
                  className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Previous
                </button>
                <button
                  onClick={() => handlePageChange(currentPage + 1)}
                  disabled={currentPage === totalPages}
                  className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Next
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Add Selected Button */}
        <div className="border-t border-gray-200 p-4 bg-white">
          <button
            onClick={handleAddSelected}
            disabled={selectedRepoIds.size === 0 || loading}
            className="w-full px-4 py-2.5 bg-green-600 text-white font-medium rounded-lg hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Selected ({selectedRepoIds.size})
          </button>
        </div>
      </div>

      {/* Right Panel - Selected Repositories & Batch Info */}
      <div className={`flex-shrink-0 flex flex-col bg-white overflow-hidden transition-all duration-300 ${currentBatchRepos.length > 0 ? 'w-full lg:w-[40%]' : 'w-full lg:w-[30%]'}`}>
        <div className="p-4 border-b border-gray-200">
          <div className="flex justify-between items-center">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                Selected Repositories
              </h3>
              <p className="text-sm text-gray-600 mt-0.5">
                {currentBatchRepos.length} {currentBatchRepos.length === 1 ? 'repository' : 'repositories'}
              </p>
            </div>
            {currentBatchRepos.length > 0 && (
              <button
                onClick={handleClearAll}
                className="text-sm text-red-600 hover:text-red-700 font-medium"
              >
                Clear All
              </button>
            )}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          {currentBatchRepos.length === 0 ? (
            <div className="text-center py-12">
              <svg className="mx-auto h-12 w-12 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p className="mt-2 text-sm text-gray-500">No repositories selected</p>
              <p className="text-xs text-gray-400 mt-1">Select repositories from the left</p>
            </div>
          ) : (
            <div className="space-y-3">
              {Object.entries(currentGroups).map(([org, repos]) => (
                <div key={org} className="border border-gray-200 rounded-lg overflow-hidden bg-white shadow-sm">
                  <div className="bg-gradient-to-r from-gray-50 to-gray-100 px-3 py-2 border-b border-gray-200">
                    <span className="font-semibold text-gray-900 text-sm">{org}</span>
                    <span className="ml-2 px-2 py-0.5 bg-white text-gray-700 rounded-full text-xs font-medium border border-gray-200">
                      {repos.length}
                    </span>
                  </div>
                  <div className="divide-y divide-gray-200">
                    {repos.map((repo) => (
                      <div key={repo.id} className="p-3 flex items-center justify-between hover:bg-gray-50 transition-colors">
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-gray-900 text-sm truncate">
                            {repo.full_name.split('/')[1] || repo.full_name}
                          </div>
                          <div className="text-xs text-gray-600 mt-0.5">
                            {formatBytes(repo.total_size || 0)} • {repo.branch_count} branches
                          </div>
                        </div>
                        <button
                          onClick={() => handleRemoveRepo(repo.id)}
                          className="ml-2 p-1 text-red-600 hover:text-red-700 hover:bg-red-50 rounded transition-colors"
                          title="Remove repository"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Batch Metadata Form */}
        <div className="border-t border-gray-200 p-4 bg-gray-50 space-y-4">
          <div className="bg-blue-50 border border-blue-200 p-3 rounded-lg">
            <div className="text-xs font-medium text-blue-900 mb-1">Total Batch Size</div>
            <div className="text-xl font-bold text-blue-900">{formatBytes(totalSize)}</div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1.5">
              Batch Name *
            </label>
            <input
              type="text"
              value={batchName}
              onChange={(e) => setBatchName(e.target.value)}
              placeholder="e.g., Wave 1, Q1 Migration"
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
              required
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1.5">
              Description
            </label>
            <textarea
              value={batchDescription}
              onChange={(e) => setBatchDescription(e.target.value)}
              placeholder="Optional description"
              rows={2}
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1.5">
              Scheduled Date (Optional)
            </label>
            <input
              type="datetime-local"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
            />
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-800 px-3 py-2 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex flex-col gap-2">
            <button
              onClick={() => handleSubmit(false)}
              disabled={loading || currentBatchRepos.length === 0}
              className="w-full px-4 py-2.5 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? 'Saving...' : isEditMode ? 'Update Batch' : 'Create Batch'}
            </button>
            {!isEditMode && (
              <button
                onClick={() => handleSubmit(true)}
                disabled={loading || currentBatchRepos.length === 0}
                className="w-full px-4 py-2.5 bg-green-600 text-white font-medium rounded-lg hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Create & Start
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="w-full px-4 py-2 border border-gray-300 text-gray-700 font-medium rounded-lg hover:bg-gray-50 disabled:opacity-50 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

