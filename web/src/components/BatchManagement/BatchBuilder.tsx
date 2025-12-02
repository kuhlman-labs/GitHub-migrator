import { useState, useEffect } from 'react';
import { Button } from '@primer/react';
import { ChevronDownIcon, UploadIcon } from '@primer/octicons-react';
import type { Repository, Batch, RepositoryFilters } from '../../types';
import { api } from '../../services/api';
import { UnifiedFilterSidebar } from '../common/UnifiedFilterSidebar';
import { ActiveFilterPills } from './ActiveFilterPills';
import { RepositoryListItem } from './RepositoryListItem';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { Pagination } from '../common/Pagination';
import { formatBytes, formatDateForInput } from '../../utils/format';
import { ImportDialog } from './ImportDialog';
import { ImportPreview, type ValidationGroup } from './ImportPreview';
import type { ImportParseResult } from '../../utils/import';

interface BatchBuilderProps {
  batch?: Batch; // If provided, we're editing; otherwise creating
  onClose: () => void;
  onSuccess: () => void;
}

export function BatchBuilder({ batch, onClose, onSuccess }: BatchBuilderProps) {
  const isEditMode = !!batch;

  // Batch metadata - ensure all inputs start with defined values (never undefined)
  const [batchName, setBatchName] = useState('');
  const [batchDescription, setBatchDescription] = useState('');
  const [scheduledAt, setScheduledAt] = useState('');

  // Migration settings
  const [destinationOrg, setDestinationOrg] = useState('');
  const [migrationAPI, setMigrationAPI] = useState<'GEI' | 'ELM'>('GEI');
  const [excludeReleases, setExcludeReleases] = useState(false);
  
  // Organization list for autocomplete
  const [organizations, setOrganizations] = useState<string[]>([]);

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
  const [showMigrationSettings, setShowMigrationSettings] = useState(false);
  
  // Import state
  const [showImportDialog, setShowImportDialog] = useState(false);
  const [showImportPreview, setShowImportPreview] = useState(false);
  const [importValidation, setImportValidation] = useState<ValidationGroup | null>(null);
  
  // Confirmation dialog state
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
    onConfirm: () => void;
  }>({
    isOpen: false,
    title: '',
    message: '',
    onConfirm: () => {},
  });

  // Load organizations for autocomplete
  useEffect(() => {
    const loadOrganizations = async () => {
      try {
        const orgList = await api.getOrganizationList();
        setOrganizations(orgList);
      } catch (err) {
        console.error('Failed to load organizations:', err);
      }
    };
    loadOrganizations();
  }, []);

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
      setDestinationOrg(batchData.destination_org || '');
      setMigrationAPI(batchData.migration_api || 'GEI');
      setExcludeReleases(batchData.exclude_releases || false);
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
      
      console.log('API Response:', { 
        isArray: Array.isArray(response), 
        reposCount: repos.length, 
        total: Array.isArray(response) ? 'N/A' : response.total,
        hasTotal: !Array.isArray(response) && 'total' in response,
        currentPage: currentPage,
        filters: filters
      });
      
      // Always update repos
      setAvailableRepos(repos);
      
      // Update total based on response
      if (Array.isArray(response)) {
        // No pagination - response is the full array
        console.log('Setting total from array length:', response.length);
        setTotalAvailable(response.length);
      } else if (response.total !== undefined && response.total !== null && response.total > 0) {
        // Backend provided a valid positive total
        console.log('Setting total from response:', response.total);
        setTotalAvailable(response.total);
      } else if (response.total === 0 && repos.length === 0 && currentPage === 1) {
        // Only accept total=0 if we're on the first page with no repos (legitimate empty result)
        console.log('Setting total to 0 (no matching repos on first page)');
        setTotalAvailable(0);
      } else if (currentPage === 1 && repos.length > 0) {
        // First page with repos but no total - estimate
        const estimatedTotal = repos.length < pageSize ? repos.length : repos.length;
        console.log('Estimating total for first page:', estimatedTotal);
        setTotalAvailable(estimatedTotal);
      }
      // Otherwise, keep existing totalAvailable value (important for pagination)
      
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
    setSelectedRepoIds(new Set()); // Clear selections when filters change
  };

  const handleClearFilters = () => {
    setFilters({ available_for_batch: true, limit: 50, offset: 0 });
    setCurrentPage(1);
    setSelectedRepoIds(new Set()); // Clear selections when filters clear
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
      setSelectedRepoIds(new Set()); // Clear selections when quick filter changes
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
    const newBatchRepos = [...currentBatchRepos, ...selectedRepos];
    setCurrentBatchRepos(newBatchRepos);
    setSelectedRepoIds(new Set());
    
    // Check if all non-added repos on current page were selected
    const addedIds = new Set(newBatchRepos.map(r => r.id));
    const remainingRepos = availableRepos.filter((r) => !addedIds.has(r.id));
    
    // Calculate pagination values
    const currentPageSize = filters.limit || 50;
    const totalPagesCount = Math.ceil(totalAvailable / currentPageSize);
    
    // If no repos remain on current page and there are more pages, auto-advance to next page
    if (remainingRepos.length === 0 && currentPage < totalPagesCount) {
      const nextPage = currentPage + 1;
      setCurrentPage(nextPage);
      setFilters({ ...filters, offset: (nextPage - 1) * currentPageSize });
    }
    // Note: We don't need to manually reload because the useEffect will trigger when filters change
  };

  const executeAddAll = async () => {
    const alreadyAddedIds = new Set(currentBatchRepos.map(r => r.id));
    
    setAvailableLoading(true);
    setConfirmDialog({ ...confirmDialog, isOpen: false });
    
    try {
      // Fetch all repositories matching the filters (without pagination)
      const filtersWithoutPagination = { ...filters };
      delete filtersWithoutPagination.limit;
      delete filtersWithoutPagination.offset;
      
      const response = await api.listRepositories(filtersWithoutPagination);
      const allRepos = Array.isArray(response) ? response : (response.repositories || []);
      
      // Filter out repos that are already in the batch
      const newRepos = allRepos.filter(r => !alreadyAddedIds.has(r.id));
      
      // Add all new repos to the batch
      const updatedBatch = [...currentBatchRepos, ...newRepos];
      setCurrentBatchRepos(updatedBatch);
      setSelectedRepoIds(new Set());
      
      console.log(`Added ${newRepos.length} repositories to batch (total now: ${updatedBatch.length})`);
      
      // Reload the current page to show updated state
    await loadAvailableRepos();
    } catch (err) {
      console.error('Failed to add all repositories:', err);
      setError('Failed to add all repositories. Please try again.');
    } finally {
      setAvailableLoading(false);
    }
  };

  const handleAddAll = () => {
    if (totalAvailable === 0) return;
    
    setConfirmDialog({
      isOpen: true,
      title: 'Add all repositories',
      message: `Add all ${totalAvailable} repositories matching your current filters to the batch?`,
      onConfirm: executeAddAll,
    });
  };

  const handleRemoveRepo = (repoId: number) => {
    setCurrentBatchRepos(currentBatchRepos.filter((r) => r.id !== repoId));
  };

  const handleClearAll = () => {
    setConfirmDialog({
      isOpen: true,
      title: 'Remove all repositories',
      message: 'Remove all repositories from this batch?',
      onConfirm: () => {
      setCurrentBatchRepos([]);
        setConfirmDialog({ ...confirmDialog, isOpen: false });
      },
    });
  };

  // Import handlers
  const handleImportClick = () => {
    setShowImportDialog(true);
  };

  const handleImportParsed = async (parseResult: ImportParseResult) => {
    setShowImportDialog(false);

    // Validate repositories against API
    try {
      // Fetch all repositories to validate
      const response = await api.listRepositories({});
      const allRepos = response.repositories || response;
      
      // Create lookup map
      const repoMap = new Map<string, Repository>();
      allRepos.forEach((repo: Repository) => {
        repoMap.set(repo.full_name.toLowerCase(), repo);
      });

      // Already in batch
      const alreadyInBatchIds = new Set(currentBatchRepos.map(r => r.id));

      // Categorize repositories
      const valid: Repository[] = [];
      const alreadyInBatch: Repository[] = [];
      const notFound: { full_name: string }[] = [];

      parseResult.rows.forEach((row) => {
        const repo = repoMap.get(row.full_name.toLowerCase());
        
        if (!repo) {
          notFound.push({ full_name: row.full_name });
        } else if (alreadyInBatchIds.has(repo.id)) {
          alreadyInBatch.push(repo);
        } else {
          // Add repository (migration settings will come from batch configuration)
          valid.push(repo);
        }
      });

      setImportValidation({ valid, alreadyInBatch, notFound });
      setShowImportPreview(true);
    } catch (err) {
      console.error('Failed to validate imported repositories:', err);
      setError('Failed to validate imported repositories');
    }
  };

  const handleImportConfirm = (selectedRepos: Repository[]) => {
    // Add imported repositories to the batch (they will use batch-level migration settings)
    const newBatchRepos = [...currentBatchRepos, ...selectedRepos];
    setCurrentBatchRepos(newBatchRepos);
    setShowImportPreview(false);
    setImportValidation(null);
  };

  const handleImportCancel = () => {
    setShowImportDialog(false);
    setShowImportPreview(false);
    setImportValidation(null);
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
          destination_org: destinationOrg.trim() || undefined,
          migration_api: migrationAPI,
          exclude_releases: excludeReleases,
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
          destination_org: destinationOrg.trim() || undefined,
          migration_api: migrationAPI,
          exclude_releases: excludeReleases,
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
      // For ADO repos, group by project; for GitHub repos, group by org (first part of full_name)
      const groupKey = repo.ado_project || repo.full_name.split('/')[0];
      if (!groups[groupKey]) groups[groupKey] = [];
      groups[groupKey].push(repo);
    });
    return groups;
  };

  // Track which repos have been added to the batch to disable their checkboxes
  const addedRepoIds = new Set(currentBatchRepos.map(r => r.id));
  
  // Group selected/current batch repos by org for better organization
  const currentGroups = groupReposByOrg(currentBatchRepos);
  
  // Filter out already-added repos from the available list for display
  const availableReposToShow = availableRepos.filter(r => !addedRepoIds.has(r.id));

  const totalSize = currentBatchRepos.reduce((sum, repo) => sum + (repo.total_size || 0), 0);
  const pageSize = filters.limit || 50;
  const totalPages = Math.ceil(totalAvailable / pageSize);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    setFilters({ ...filters, offset: (page - 1) * pageSize });
  };

  return (
    <div className="h-full flex" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
      {/* Filter Sidebar */}
      <UnifiedFilterSidebar
        filters={filters}
        onChange={handleFilterChange}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
        showStatus={false}
        showSearch={true}
      />

      {/* Middle Panel - Available Repositories */}
      <div 
        className={`flex-1 min-w-0 grid grid-rows-[auto_1fr_auto] border-r transition-all duration-300 h-full ${currentBatchRepos.length > 0 ? 'lg:w-[45%]' : 'lg:w-[60%]'}`}
        style={{ backgroundColor: 'var(--bgColor-default)', borderColor: 'var(--borderColor-default)' }}
      >
        <div 
          className="p-4 border-b row-start-1"
          style={{ borderColor: 'var(--borderColor-default)', backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="flex items-center justify-between mb-3">
            <div>
              <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>Available Repositories</h3>
              <p className="text-sm mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                {availableLoading ? 'Loading...' : `${totalAvailable} repositories available`}
              </p>
            </div>
              <div className="flex items-center gap-2">
              <button
                onClick={handleImportClick}
                disabled={loading}
                className="px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors hover:opacity-80 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                style={{
                  borderColor: 'var(--borderColor-default)',
                  color: 'var(--fgColor-default)',
                  backgroundColor: 'var(--control-bgColor-rest)'
                }}
                title="Import repositories from file"
              >
                <UploadIcon size={16} />
                Import from File
              </button>
              {selectedRepoIds.size > 0 && (
                <span 
                  className="px-3 py-1.5 rounded-full text-sm font-semibold"
                  style={{ backgroundColor: 'var(--accent-subtle)', color: 'var(--fgColor-accent)' }}
                >
                  {selectedRepoIds.size} selected
                </span>
              )}
              {totalAvailable > 0 && availableReposToShow.length > 0 && (
                <button
                  onClick={handleAddAll}
                  disabled={availableLoading}
                  className="px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors hover:opacity-80 disabled:opacity-50 disabled:cursor-not-allowed"
                  style={{
                    borderColor: 'var(--borderColor-default)',
                    color: 'var(--fgColor-accent)',
                    backgroundColor: 'var(--control-bgColor-rest)'
                  }}
                  title="Add all repositories matching current filters"
                >
                  <svg className="w-4 h-4 inline-block mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                  </svg>
                  Add All ({totalAvailable})
                </button>
            )}
            </div>
          </div>

          {/* Quick Filter Buttons */}
          <div className="flex flex-wrap gap-2 mb-3">
            <button
              onClick={() => handleQuickFilter()}
              className="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors"
              style={!filters.complexity
                ? { backgroundColor: 'var(--accent-emphasis)', color: 'var(--fgColor-onEmphasis)', borderColor: 'var(--accent-emphasis)' }
                : { borderColor: 'var(--borderColor-default)', color: 'var(--fgColor-default)', backgroundColor: 'var(--control-bgColor-rest)' }
              }
            >
              All
            </button>
            <button
              onClick={() => handleQuickFilter(['simple'])}
              className="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors"
              style={Array.isArray(filters.complexity) && filters.complexity.length === 1 && filters.complexity[0] === 'simple'
                ? { backgroundColor: 'var(--success-emphasis)', color: 'var(--fgColor-onEmphasis)', borderColor: 'var(--success-emphasis)' }
                : { borderColor: 'var(--borderColor-default)', color: 'var(--fgColor-default)', backgroundColor: 'var(--control-bgColor-rest)' }
              }
            >
              Simple
            </button>
            <button
              onClick={() => handleQuickFilter(['medium'])}
              className="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors"
              style={Array.isArray(filters.complexity) && filters.complexity.length === 1 && filters.complexity[0] === 'medium'
                ? { backgroundColor: '#FB8500', color: '#ffffff', borderColor: '#FB8500' }
                : { borderColor: 'var(--borderColor-default)', color: 'var(--fgColor-default)', backgroundColor: 'var(--control-bgColor-rest)' }
              }
            >
              Medium
            </button>
            <button
              onClick={() => handleQuickFilter(['complex', 'very_complex'])}
              className="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors"
              style={Array.isArray(filters.complexity) && filters.complexity.includes('complex')
                ? { backgroundColor: '#F97316', color: '#ffffff', borderColor: '#F97316' }
                : { borderColor: 'var(--borderColor-default)', color: 'var(--fgColor-default)', backgroundColor: 'var(--control-bgColor-rest)' }
              }
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
        <div className="overflow-y-auto p-4 space-y-2 row-start-2 min-h-0">
          {availableLoading ? (
            <div className="flex items-center justify-center py-12">
              <LoadingSpinner />
            </div>
          ) : availableReposToShow.length === 0 ? (
            <div className="text-center py-12">
              <svg className="mx-auto h-12 w-12" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
              </svg>
              <p className="mt-2 text-sm font-semibold" style={{ color: 'var(--fgColor-muted)' }}>
                {totalAvailable === 0
                  ? 'No repositories match your filters'
                  : availableRepos.length > 0 
                    ? 'All repositories on this page have been added to the batch'
                  : 'No repositories available'}
              </p>
              <p className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                {totalAvailable === 0
                  ? 'Try adjusting or clearing your filters to see more repositories'
                  : availableRepos.length > 0
                    ? (currentPage < totalPages 
                        ? 'Navigate to the next page to add more repositories'
                        : `All ${totalAvailable} matching repositories have been added`)
                    : 'Try adjusting your filters or add more repositories'}
              </p>
              {availableRepos.length > 0 && currentPage < totalPages && (
                <button
                  onClick={() => handlePageChange(currentPage + 1)}
                  className="mt-4 px-4 py-2 text-sm font-medium rounded-md transition-colors shadow-sm"
                  style={{ backgroundColor: 'var(--accent-emphasis)', color: 'var(--fgColor-onEmphasis)' }}
                >
                  Go to Next Page →
                </button>
              )}
            </div>
          ) : (
            <>
              {/* Select All header */}
              {availableReposToShow.length > 0 && (
                <div 
                  className="sticky top-0 p-3 mb-2 rounded-lg border flex items-center gap-3"
                  style={{ 
                    backgroundColor: 'var(--bgColor-muted)',
                    zIndex: 1,
                    borderColor: 'var(--borderColor-default)'
                  }}
                >
                  <input
                    type="checkbox"
                    checked={availableReposToShow.every((repo) => selectedRepoIds.has(repo.id))}
                    ref={(el) => {
                      if (el) {
                        const allSelected = availableReposToShow.every((repo) => selectedRepoIds.has(repo.id));
                        const someSelected = availableReposToShow.some((repo) => selectedRepoIds.has(repo.id)) && !allSelected;
                        el.indeterminate = someSelected;
                      }
                    }}
                    onChange={() => handleToggleAllInGroup(availableReposToShow.map(r => r.id))}
                    className="rounded text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                    style={{ borderColor: 'var(--borderColor-default)' }}
                  />
                  <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                    Select all {availableReposToShow.length} on this page
                  </span>
                </div>
              )}
              
              {/* Flat list of repositories */}
              {availableReposToShow.map((repo) => (
                <RepositoryListItem
                  key={repo.id}
                  repository={repo}
                  selected={selectedRepoIds.has(repo.id)}
                onToggle={handleToggleRepo}
              />
              ))}
            </>
          )}
        </div>

        {/* Bottom Section - Pagination & Add Button */}
        <div 
          className="shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] row-start-3"
          style={{ backgroundColor: 'var(--bgColor-default)', borderTop: '1px solid var(--borderColor-default)', zIndex: 1 }}
        >
          {/* Pagination */}
          <Pagination
            currentPage={currentPage}
            totalItems={totalAvailable}
            pageSize={pageSize}
            onPageChange={handlePageChange}
          />

          {/* Add Selected Button - Always Visible */}
          <div className="p-4">
            <button
              onClick={handleAddSelected}
              disabled={selectedRepoIds.size === 0 || loading}
              className="w-full px-4 py-2.5 font-medium rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center justify-center gap-2 shadow-md hover:shadow-lg border-0 cursor-pointer"
              style={{ 
                backgroundColor: '#2da44e',
                color: '#ffffff'
              }}
              onMouseEnter={(e) => {
                if (!e.currentTarget.disabled) {
                  e.currentTarget.style.backgroundColor = '#2c974b';
                }
              }}
              onMouseLeave={(e) => {
                if (!e.currentTarget.disabled) {
                  e.currentTarget.style.backgroundColor = '#2da44e';
                }
              }}
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
      <div 
        className={`flex-shrink-0 flex flex-col transition-all duration-300 h-full ${currentBatchRepos.length > 0 ? 'w-full lg:w-[40%]' : 'w-full lg:w-[30%]'}`}
        style={{ backgroundColor: 'var(--bgColor-default)' }}
      >
        {/* Sticky Header with Batch Info */}
        <div 
          className="flex-shrink-0 sticky top-0 shadow-sm"
          style={{ 
            backgroundColor: 'var(--bgColor-default)', 
            borderBottom: '1px solid var(--borderColor-default)',
            zIndex: 1
          }}
        >
          <div className="p-4">
            <div className="flex justify-between items-center mb-3">
            <div>
              <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                Selected Repositories
              </h3>
              <p className="text-sm mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                {currentBatchRepos.length} {currentBatchRepos.length === 1 ? 'repository' : 'repositories'}
              </p>
            </div>
            {currentBatchRepos.length > 0 && (
              <button
                onClick={handleClearAll}
                className="text-sm font-medium transition-colors hover:opacity-80"
                style={{ color: 'var(--fgColor-danger)' }}
              >
                Clear All
              </button>
            )}
            </div>
            {/* Batch Size Indicator */}
            <div 
              className="border p-2.5 rounded-lg"
              style={{ backgroundColor: 'var(--accent-subtle)', borderColor: 'var(--accent-muted)' }}
            >
              <div className="flex items-center justify-between">
                <div className="text-xs font-medium" style={{ color: 'var(--fgColor-accent)' }}>Total Batch Size</div>
                <div className="text-lg font-bold" style={{ color: 'var(--fgColor-accent)' }}>{formatBytes(totalSize)}</div>
              </div>
            </div>
          </div>
        </div>

        {/* Repository List - Scrollable with expanded height */}
        <div className="flex-1 overflow-y-auto p-4 min-h-0">
          {currentBatchRepos.length === 0 ? (
            <div className="text-center py-12">
              <svg className="mx-auto h-12 w-12" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p className="mt-2 text-sm" style={{ color: 'var(--fgColor-muted)' }}>No repositories selected</p>
              <p className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>Select repositories from the left</p>
            </div>
          ) : (
            <div className="space-y-3">
              {Object.entries(currentGroups).map(([org, repos]) => (
                <div 
                  key={org} 
                  className="rounded-lg overflow-hidden shadow-sm"
                  style={{ 
                    border: '1px solid var(--borderColor-default)',
                    backgroundColor: 'var(--bgColor-default)' 
                  }}
                >
                  <div 
                    className="px-3 py-2"
                    style={{ 
                      backgroundColor: 'var(--bgColor-muted)',
                      borderBottom: '1px solid var(--borderColor-default)' 
                    }}
                  >
                    <span className="font-semibold text-sm" style={{ color: 'var(--fgColor-default)' }}>{org}</span>
                    <span 
                      className="ml-2 px-2 py-0.5 rounded-full text-xs font-medium"
                      style={{
                        backgroundColor: 'var(--bgColor-default)',
                        color: 'var(--fgColor-default)',
                        border: '1px solid var(--borderColor-default)'
                      }}
                    >
                      {repos.length}
                    </span>
                  </div>
                  <div style={{ borderTop: '1px solid var(--borderColor-muted)' }}>
                    {repos.map((repo, index) => (
                      <div 
                        key={repo.id} 
                        className="p-3 flex items-center justify-between hover:opacity-80 transition-opacity"
                        style={{ borderTop: index > 0 ? '1px solid var(--borderColor-muted)' : 'none' }}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-sm truncate" style={{ color: 'var(--fgColor-default)' }}>
                            {repo.ado_project 
                              ? repo.full_name // For ADO, full_name is just the repo name
                              : repo.full_name.split('/')[1] || repo.full_name // For GitHub, extract repo name from org/repo
                            }
                          </div>
                          <div className="text-xs mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                            {formatBytes(repo.total_size || 0)} • {repo.branch_count} branches
                          </div>
                        </div>
                        <button
                          onClick={() => handleRemoveRepo(repo.id)}
                          className="ml-2 p-1 rounded transition-opacity hover:opacity-80"
                          style={{ color: 'var(--fgColor-danger)' }}
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

        {/* Bottom Batch Configuration Form - Compact */}
        <div 
          className="flex-shrink-0 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)]"
          style={{ 
            borderTop: '1px solid var(--borderColor-default)',
            backgroundColor: 'var(--bgColor-default)' 
          }}
        >
          {/* Essential Fields - Always Visible */}
          <div className="p-3 space-y-2.5">
          <div>
            <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
              Batch Name *
            </label>
            <input
              type="text"
              value={batchName}
              onChange={(e) => setBatchName(e.target.value)}
              placeholder="e.g., Wave 1, Q1 Migration"
              className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              style={{
                border: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-default)'
              }}
              disabled={loading}
              required
            />
          </div>

          <div>
            <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
              Description
            </label>
            <textarea
              value={batchDescription}
              onChange={(e) => setBatchDescription(e.target.value)}
              placeholder="Optional description"
              rows={1}
              className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
              style={{
                border: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-default)'
              }}
              disabled={loading}
            />
            </div>
          </div>

          {/* Collapsible Migration Settings */}
          <div style={{ borderTop: '1px solid var(--borderColor-default)' }}>
            <button
              type="button"
              onClick={() => setShowMigrationSettings(!showMigrationSettings)}
              className="w-full px-3 py-2.5 flex items-center justify-between text-sm font-medium hover:opacity-80 transition-opacity"
              style={{ color: 'var(--fgColor-default)' }}
            >
              <div className="flex items-center gap-2">
                <svg className="w-4 h-4" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <span>Migration Settings</span>
                {(destinationOrg || excludeReleases || migrationAPI !== 'GEI') && (
                  <span 
                    className="px-1.5 py-0.5 text-xs rounded-full font-medium"
                    style={{
                      backgroundColor: 'var(--accent-subtle)',
                      color: 'var(--fgColor-accent)'
                    }}
                  >
                    {[destinationOrg ? 1 : 0, excludeReleases ? 1 : 0, migrationAPI !== 'GEI' ? 1 : 0].reduce((a, b) => a + b, 0)} configured
                  </span>
                )}
              </div>
              <span style={{ color: 'var(--fgColor-muted)' }}>
              <ChevronDownIcon
                  className={`transition-transform ${showMigrationSettings ? 'rotate-180' : ''}`}
                size={20}
              />
              </span>
            </button>

            {showMigrationSettings && (
              <div 
                className="p-3 space-y-2.5"
                style={{ 
                  backgroundColor: 'var(--bgColor-muted)',
                  borderTop: '1px solid var(--borderColor-default)' 
                }}
              >
                <div>
                  <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
                    Destination Organization
                    <span className="ml-1 font-normal text-xs" style={{ color: 'var(--fgColor-muted)' }}>— Default for repos without specific destination</span>
                  </label>
                  <input
                    type="text"
                    value={destinationOrg}
                    onChange={(e) => setDestinationOrg(e.target.value)}
                    placeholder="Leave blank to use source org"
                    list="organizations-list"
                    className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    style={{
                      border: '1px solid var(--borderColor-default)',
                      backgroundColor: 'var(--control-bgColor-rest)',
                      color: 'var(--fgColor-default)'
                    }}
                    disabled={loading}
                  />
                  <datalist id="organizations-list">
                    {organizations.map((org) => (
                      <option key={org} value={org} />
                    ))}
                  </datalist>
                </div>

                <div>
                  <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
                    Migration API
                  </label>
                  <select
                    value={migrationAPI}
                    onChange={(e) => setMigrationAPI(e.target.value as 'GEI' | 'ELM')}
                    className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    style={{
                      border: '1px solid var(--borderColor-default)',
                      backgroundColor: 'var(--control-bgColor-rest)',
                      color: 'var(--fgColor-default)'
                    }}
                    disabled={loading}
                  >
                    <option value="GEI">GEI (GitHub Enterprise Importer)</option>
                    <option value="ELM">ELM (Enterprise Live Migrator) - Future</option>
                  </select>
                </div>

                <div className="flex items-start gap-2">
                  <input
                    type="checkbox"
                    id="exclude-releases"
                    checked={excludeReleases}
                    onChange={(e) => setExcludeReleases(e.target.checked)}
                    className="mt-0.5 h-4 w-4 rounded text-blue-600 focus:ring-2 focus:ring-blue-500"
                    style={{ borderColor: 'var(--borderColor-default)' }}
                    disabled={loading}
                  />
                  <label htmlFor="exclude-releases" className="text-xs cursor-pointer" style={{ color: 'var(--fgColor-default)' }}>
                    <span className="font-semibold">Exclude Releases</span>
                    <span className="block mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>Skip releases during migration (repo settings override)</span>
                  </label>
                </div>
              </div>
            )}
          </div>

          {/* Scheduled Date Section */}
          <div className="p-3" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
          <div className="relative z-[60]">
            <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
              Scheduled Date (Optional)
            </label>
            <input
              type="datetime-local"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
              min={formatDateForInput(new Date().toISOString())}
              className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              style={{
                border: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-default)'
              }}
              disabled={loading}
              placeholder="Select date and time"
            />
            <p className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
              Batch will auto-start at the scheduled time (after dry run is complete)
            </p>
            </div>
          </div>

          {/* Error Message */}
          {error && (
            <div className="px-3 pb-3">
            <div 
              className="px-2.5 py-1.5 rounded-lg text-xs"
              style={{
                backgroundColor: 'var(--danger-subtle)',
                border: '1px solid var(--borderColor-danger)',
                color: 'var(--fgColor-danger)'
              }}
            >
              {error}
              </div>
            </div>
          )}

          {/* Action Buttons */}
          <div 
            className="border-t p-3"
            style={{ borderColor: 'var(--borderColor-default)', backgroundColor: 'var(--bgColor-muted)' }}
          >
            <div className="flex flex-col gap-1.5">
            <Button
              onClick={() => handleSubmit(false)}
              disabled={loading || currentBatchRepos.length === 0}
              variant="primary"
              block
            >
              {loading ? 'Saving...' : isEditMode ? 'Update Batch' : 'Create Batch'}
            </Button>
            {!isEditMode && (
              <button
                onClick={() => handleSubmit(true)}
                disabled={loading || currentBatchRepos.length === 0}
                className="w-full px-3 py-2 text-sm font-medium rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md hover:shadow-lg border-0 cursor-pointer"
                style={{ 
                  backgroundColor: '#2da44e',
                  color: '#ffffff'
                }}
                onMouseEnter={(e) => {
                  if (!e.currentTarget.disabled) {
                    e.currentTarget.style.backgroundColor = '#2c974b';
                  }
                }}
                onMouseLeave={(e) => {
                  if (!e.currentTarget.disabled) {
                    e.currentTarget.style.backgroundColor = '#2da44e';
                  }
                }}
              >
                Create & Start
              </button>
            )}
            <Button
              type="button"
              onClick={onClose}
              disabled={loading}
              block
            >
              Cancel
            </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Confirmation Dialog */}
      {confirmDialog.isOpen && (
        <div 
          className="fixed inset-0 flex items-center justify-center"
          style={{ zIndex: 99999 }}
        >
          {/* Backdrop */}
          <div 
            className="absolute inset-0" 
            style={{ backgroundColor: 'rgba(0, 0, 0, 0.6)' }}
            onClick={() => setConfirmDialog(prev => ({ ...prev, isOpen: false }))}
          />
          
          {/* Modal Content */}
          <div 
            className="relative rounded-lg shadow-xl max-w-lg w-full mx-4"
            style={{ 
              backgroundColor: 'var(--bgColor-default)',
              border: '1px solid var(--borderColor-default)'
            }}
          >
            {/* Header */}
            <div 
              className="px-4 py-3 border-b"
              style={{ borderColor: 'var(--borderColor-default)' }}
            >
              <h2 
                className="text-lg font-semibold"
                style={{ color: 'var(--fgColor-default)' }}
              >
                {confirmDialog.title}
              </h2>
            </div>
            
            {/* Body */}
            <div className="px-4 py-4">
              <p style={{ color: 'var(--fgColor-default)' }}>
                {confirmDialog.message}
              </p>
            </div>
            
            {/* Footer */}
            <div 
              className="px-4 py-3 border-t flex gap-2 justify-end"
              style={{ 
                borderColor: 'var(--borderColor-default)',
                backgroundColor: 'var(--bgColor-muted)'
              }}
            >
              <Button 
                onClick={() => setConfirmDialog(prev => ({ ...prev, isOpen: false }))}
              >
                Cancel
              </Button>
              <Button
                variant="danger"
                onClick={() => {
                  setConfirmDialog(prev => ({ ...prev, isOpen: false }));
                  confirmDialog.onConfirm();
                }}
              >
                OK
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Import Dialog */}
      {showImportDialog && (
        <ImportDialog
          onImport={handleImportParsed}
          onCancel={handleImportCancel}
        />
      )}

      {/* Import Preview Dialog */}
      {showImportPreview && importValidation && (
        <ImportPreview
          validationResult={importValidation}
          onConfirm={handleImportConfirm}
          onCancel={handleImportCancel}
        />
      )}
    </div>
  );
}

