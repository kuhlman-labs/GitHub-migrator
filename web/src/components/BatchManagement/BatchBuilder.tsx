import { useState, useEffect } from 'react';
import type { Repository, Batch, RepositoryFilters } from '../../types';
import { api } from '../../services/api';
import { FilterSidebar } from './FilterSidebar';
import { ActiveFilterPills } from './ActiveFilterPills';
import { RepositoryGroup } from './RepositoryGroup';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { formatBytes, formatDateForInput } from '../../utils/format';

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
  const [scheduledAt, setScheduledAt] = useState(formatDateForInput(batch?.scheduled_at));

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

  // Update form fields when batch loads in edit mode
  useEffect(() => {
    if (batch) {
      console.log('Populating form with batch data:', batch);
      console.log('batch.name:', batch.name);
      console.log('batch.id:', batch.id);
      console.log('Has nested batch property?', 'batch' in (batch as any));
      
      // Handle nested batch structure from API
      const batchData = (batch as any).batch || batch;
      console.log('Using batch data:', batchData);
      console.log('batchData.name:', batchData.name);
      
      setBatchName(batchData.name || '');
      setBatchDescription(batchData.description || '');
      setScheduledAt(formatDateForInput(batchData.scheduled_at));
    }
  }, [batch]);

  // Load current batch repositories in edit mode
  useEffect(() => {
    if (isEditMode && batch) {
      // Handle nested batch structure
      const batchData = (batch as any).batch || batch;
      const batchId = batchData?.id || batch.id;
      
      console.log('Edit mode: batch object:', batch);
      console.log('Edit mode: extracted batch ID:', batchId);
      
      if (batchId) {
        // Check if repositories are already included in the batch response
        const repos = (batch as any).repositories;
        if (repos && Array.isArray(repos)) {
          console.log('✓ Using repositories from batch response:', repos.length);
          setCurrentBatchRepos(repos);
          const repoIds = repos.map((r: any) => r.id);
          setSelectedRepoIds(new Set(repoIds));
          console.log('✓ Auto-selected', repoIds.length, 'repository IDs:', repoIds);
        } else {
          console.log('Edit mode: Loading repos for batch', batchId);
          loadCurrentBatchRepos();
        }
      }
    } else if (isEditMode && !batch) {
      console.log('Edit mode: Waiting for batch to load...');
    }
  }, [isEditMode, batch]);

  // Load available repositories
  useEffect(() => {
    loadAvailableRepos();
  }, [filters]);

  const loadCurrentBatchRepos = async () => {
    // Handle nested batch structure
    const batchData = (batch as any)?.batch || batch;
    const batchId = batchData?.id || batch?.id;
    
    if (!batchId) {
      console.error('Cannot load batch repos: batch ID is undefined', { batch, batchData, batchId });
      return;
    }
    
    try {
      console.log('Fetching repos for batch ID:', batchId);
      const response = await api.listRepositories({ batch_id: batchId });
      // Ensure we only set repositories that belong to this batch
      const repos = Array.isArray(response) ? response : (response.repositories || []);
      console.log('✓ Loaded', repos.length, 'repositories for batch', batchId);
      setCurrentBatchRepos(repos);
      
      // Auto-select these repositories
      const repoIds = repos.map(r => r.id);
      setSelectedRepoIds(new Set(repoIds));
      console.log('✓ Auto-selected', repoIds.length, 'repository IDs:', repoIds);
    } catch (err) {
      console.error('Failed to load current batch repos:', err);
      setCurrentBatchRepos([]);
    }
  };

  const loadAvailableRepos = async () => {
    setAvailableLoading(true);
    try {
      const response = await api.listRepositories(filters);
      const repos = Array.isArray(response) ? response : (response.repositories || []);
      const total = Array.isArray(response) ? response.length : (response.total || repos.length);
      
      console.log('Loading available repos:', repos.length, 'repositories, total:', total);
      setAvailableRepos(repos);
      setTotalAvailable(total);
    } catch (err) {
      console.error('Failed to load available repos:', err);
      setError('Failed to load repositories');
      setAvailableRepos([]);
      setTotalAvailable(0);
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

    // Validate scheduled time if provided
    if (scheduledAt) {
      const scheduledDate = new Date(scheduledAt);
      const now = new Date();
      
      // Check if the date is valid
      if (isNaN(scheduledDate.getTime())) {
        setError('Invalid scheduled date format');
        return;
      }
      
      // Allow scheduling up to 5 minutes in the past to account for clock drift/processing time
      const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
      if (scheduledDate < fiveMinutesAgo) {
        setError('Scheduled time cannot be in the past. Please select a future date and time.');
        return;
      }
    }

    setLoading(true);
    setError(null);

    try {
      let batchId: number;

      if (isEditMode && batch) {
        // Handle nested batch structure
        const batchData = (batch as any).batch || batch;
        const existingBatchId = batchData?.id || batch.id;
        
        if (!existingBatchId) {
          throw new Error('Cannot update batch: batch ID is undefined');
        }
        
        console.log('Updating batch with ID:', existingBatchId);
        
        // Update existing batch
        await api.updateBatch(existingBatchId, {
          name: batchName.trim(),
          description: batchDescription.trim() || undefined,
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        });
        
        // Update repositories - add new ones, remove old ones
        const currentIds = new Set(currentBatchRepos.map((r) => r.id));
        const originalResponse = await api.listRepositories({ batch_id: existingBatchId });
        const originalRepos = originalResponse.repositories || originalResponse as any;
        const originalIds = new Set(originalRepos.map((r: Repository) => r.id));
        
        const toAdd = Array.from(currentIds).filter((id) => !originalIds.has(id));
        const toRemove = Array.from(originalIds).filter((id) => !currentIds.has(id));
        
        console.log('Repository changes:', { toAdd, toRemove });
        
        if (toAdd.length > 0) {
          await api.addRepositoriesToBatch(existingBatchId, toAdd);
        }
        if (toRemove.length > 0) {
          await api.removeRepositoriesFromBatch(existingBatchId, toRemove);
        }
        
        batchId = existingBatchId;
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
    } catch (err: any) {
      // Extract error message from axios error response
      let errorMessage = 'Failed to save batch';
      
      if (err.response?.data?.error) {
        // Backend returned a structured error message
        errorMessage = err.response.data.error;
      } else if (err.message) {
        // Use the error message from the Error object
        errorMessage = err.message;
      }
      
      setError(errorMessage);
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
    <div className="bg-gray-50 h-full flex">
      {/* Filter Sidebar */}
      <FilterSidebar
        filters={filters}
        onChange={handleFilterChange}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
      />

      {/* Middle Panel - Available Repositories */}
      <div className={`flex-1 min-w-0 grid grid-rows-[auto_1fr_auto] bg-white border-r border-gray-200 transition-all duration-300 h-full ${currentBatchRepos.length > 0 ? 'lg:w-[45%]' : 'lg:w-[60%]'}`}>
        <div className="p-4 border-b border-gray-200 bg-white row-start-1">
          <div className="flex items-center justify-between mb-3">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">Available Repositories</h3>
              <p className="text-sm text-gray-600 mt-0.5">
                {totalAvailable} repositories available
              </p>
            </div>
            {selectedRepoIds.size > 0 && (
              <div className="flex items-center gap-2">
                <span className="px-3 py-1.5 bg-blue-100 text-blue-800 rounded-full text-sm font-semibold">
                  {selectedRepoIds.size} selected
                </span>
              </div>
            )}
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

        {/* Repository List - Scrollable */}
        <div className="overflow-y-auto p-4 space-y-3 row-start-2 min-h-0">
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

        {/* Bottom Section - Pagination & Add Button */}
        <div className="bg-white border-t border-gray-200 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] z-10 row-start-3">
          {/* Pagination */}
          {totalPages > 1 && (
            <div className="px-4 py-3 border-b border-gray-100 bg-gray-50">
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

          {/* Add Selected Button - Always Visible */}
          <div className="p-4">
            <button
              onClick={handleAddSelected}
              disabled={selectedRepoIds.size === 0 || loading}
              className="w-full px-4 py-2.5 bg-green-600 text-white font-medium rounded-lg hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center justify-center gap-2 shadow-md hover:shadow-lg"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add Selected ({selectedRepoIds.size})
            </button>
          </div>
        </div>
      </div>

      {/* Right Panel - Selected Repositories & Batch Info */}
      <div className={`flex-shrink-0 grid grid-rows-[auto_1fr_auto] bg-white transition-all duration-300 h-full ${currentBatchRepos.length > 0 ? 'w-full lg:w-[40%]' : 'w-full lg:w-[30%]'}`}>
        <div className="p-4 border-b border-gray-200 bg-white row-start-1">
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
                className="text-sm text-red-600 hover:text-red-700 font-medium transition-colors"
              >
                Clear All
              </button>
            )}
          </div>
        </div>

        <div className="overflow-y-auto p-4 row-start-2 min-h-0">
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

        {/* Bottom Batch Metadata Form */}
        <div className="border-t border-gray-200 p-3 bg-gray-50 space-y-2.5 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] z-50 relative row-start-3">
          <div className="bg-blue-50 border border-blue-200 p-2 rounded-lg">
            <div className="text-xs font-medium text-blue-900">Total Batch Size</div>
            <div className="text-lg font-bold text-blue-900">{formatBytes(totalSize)}</div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1">
              Batch Name *
            </label>
            <input
              type="text"
              value={batchName}
              onChange={(e) => setBatchName(e.target.value)}
              placeholder="e.g., Wave 1, Q1 Migration"
              className="w-full px-2.5 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
              required
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={batchDescription}
              onChange={(e) => setBatchDescription(e.target.value)}
              placeholder="Optional description"
              rows={1}
              className="w-full px-2.5 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
              disabled={loading}
            />
          </div>

          <div className="relative z-[60]">
            <label className="block text-xs font-semibold text-gray-700 mb-1">
              Scheduled Date (Optional)
            </label>
            <input
              type="datetime-local"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
              min={formatDateForInput(new Date().toISOString())}
              className="w-full px-2.5 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
              placeholder="Select date and time"
            />
            <p className="text-xs text-gray-500 mt-1">
              Batch will auto-start at the scheduled time (after dry run is complete)
            </p>
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-800 px-2.5 py-1.5 rounded-lg text-xs">
              {error}
            </div>
          )}

          <div className="flex flex-col gap-1.5 pt-1 min-h-[140px]">
            <button
              onClick={() => handleSubmit(false)}
              disabled={loading || currentBatchRepos.length === 0}
              className="w-full px-3 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md hover:shadow-lg"
            >
              {loading ? 'Saving...' : isEditMode ? 'Update Batch' : 'Create Batch'}
            </button>
            {!isEditMode && (
              <button
                onClick={() => handleSubmit(true)}
                disabled={loading || currentBatchRepos.length === 0}
                className="w-full px-3 py-2 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md hover:shadow-lg"
              >
                Create & Start
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="w-full px-3 py-1.5 border border-gray-300 text-gray-700 text-sm font-medium rounded-lg hover:bg-gray-50 disabled:opacity-50 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

