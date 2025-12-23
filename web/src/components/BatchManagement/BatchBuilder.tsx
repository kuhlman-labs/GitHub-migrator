import { useState, useEffect, useCallback } from 'react';
import { UploadIcon, PlusIcon, SidebarExpandIcon, SidebarCollapseIcon } from '@primer/octicons-react';
import { BorderedButton, PrimaryButton, IconButton } from '../common/buttons';
import type { Repository, Batch, RepositoryFilters } from '../../types';
import { api } from '../../services/api';
import { UnifiedFilterSidebar } from '../common/UnifiedFilterSidebar';
import { ActiveFilterPills } from './ActiveFilterPills';
import { RepositoryListItem } from './RepositoryListItem';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { Pagination } from '../common/Pagination';
import { ConfirmationDialog } from '../common/ConfirmationDialog';
import { formatDateForInput } from '../../utils/format';
import { ImportDialog } from './ImportDialog';
import { ImportPreview, type ValidationGroup } from './ImportPreview';
import { BatchMetadataForm } from './BatchMetadataForm';
import { BatchSummaryPanel } from './BatchSummaryPanel';
import type { ImportParseResult } from '../../utils/import';

interface BatchBuilderProps {
  batch?: Batch; // If provided, we're editing; otherwise creating
  onClose: () => void;
  onSuccess: () => void;
}

// Type for API response that may have nested batch structure
interface BatchResponse extends Batch {
  batch?: Batch;
  repositories?: Repository[];
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
  const [excludeAttachments, setExcludeAttachments] = useState(false);
  
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
  const [isAvailablePanelCollapsed, setIsAvailablePanelCollapsed] = useState(false);
  const [isSelectedReposExpanded, setIsSelectedReposExpanded] = useState(true);
  const [showMigrationSettings, setShowMigrationSettings] = useState(false);

  // Handler to toggle selected repos and auto-expand migration settings
  const handleToggleSelectedRepos = () => {
    const willBeCollapsed = isSelectedReposExpanded;
    setIsSelectedReposExpanded(!isSelectedReposExpanded);
    // Auto-expand migration settings when collapsing the repo list
    if (willBeCollapsed) {
      setShowMigrationSettings(true);
    }
  };
  
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
      } catch {
        // Organization load failed, autocomplete will be empty
      }
    };
    loadOrganizations();
  }, []);

  // Update form fields when batch loads in edit mode
  useEffect(() => {
    if (batch) {
      const batchResp = batch as BatchResponse;
      // Handle nested batch structure from API
      const batchData = batchResp.batch || batch;
      
      setBatchName(batchData.name || '');
      setBatchDescription(batchData.description || '');
      setScheduledAt(formatDateForInput(batchData.scheduled_at));
      setDestinationOrg(batchData.destination_org || '');
      setMigrationAPI(batchData.migration_api || 'GEI');
      setExcludeReleases(batchData.exclude_releases || false);
      setExcludeAttachments(batchData.exclude_attachments || false);
    }
  }, [batch]);

  // Load current batch repositories in edit mode
  useEffect(() => {
    if (isEditMode && batch) {
      // Handle nested batch structure
      const batchResp = batch as BatchResponse;
      const batchData = batchResp.batch || batch;
      const batchId = batchData?.id || batch.id;
      
      if (batchId) {
        // Check if repositories are already included in the batch response
        const repos = batchResp.repositories;
        if (repos && Array.isArray(repos)) {
          setCurrentBatchRepos(repos);
          const repoIds = repos.map((r: Repository) => r.id);
          setSelectedRepoIds(new Set(repoIds));
        } else {
          loadCurrentBatchRepos();
        }
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isEditMode, batch]);

  // Load available repositories
  useEffect(() => {
    loadAvailableRepos();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const loadCurrentBatchRepos = async () => {
    // Handle nested batch structure
    const batchResp = batch as BatchResponse | undefined;
    const batchData = batchResp?.batch || batch;
    const batchId = batchData?.id || batch?.id;
    
    if (!batchId) {
      return;
    }
    
    try {
      const response = await api.listRepositories({ batch_id: batchId });
      // Ensure we only set repositories that belong to this batch
      const repos = Array.isArray(response) ? response : (response.repositories || []);
      setCurrentBatchRepos(repos);
      
      // Auto-select these repositories
      const repoIds = repos.map(r => r.id);
      setSelectedRepoIds(new Set(repoIds));
    } catch {
      setCurrentBatchRepos([]);
    }
  };

  const loadAvailableRepos = async () => {
    setAvailableLoading(true);
    try {
      const response = await api.listRepositories(filters);
      const repos = Array.isArray(response) ? response : (response.repositories || []);
      
      // Always update repos
      setAvailableRepos(repos);
      
      // Update total based on response
      if (Array.isArray(response)) {
        // No pagination - response is the full array
        setTotalAvailable(response.length);
      } else if (response.total !== undefined && response.total !== null && response.total > 0) {
        // Backend provided a valid positive total
        setTotalAvailable(response.total);
      } else if (response.total === 0 && repos.length === 0 && currentPage === 1) {
        // Only accept total=0 if we're on the first page with no repos (legitimate empty result)
        setTotalAvailable(0);
      } else if (currentPage === 1 && repos.length > 0) {
        // First page with repos but no total - estimate
        const estimatedTotal = repos.length < pageSize ? repos.length : repos.length;
        setTotalAvailable(estimatedTotal);
      }
      // Otherwise, keep existing totalAvailable value (important for pagination)
      
    } catch {
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
      
      // Reload the current page to show updated state
    await loadAvailableRepos();
    } catch {
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
    } catch {
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
        const batchResp = batch as BatchResponse;
        const batchData = batchResp.batch || batch;
        const existingBatchId = batchData?.id || batch.id;
        
        if (!existingBatchId) {
          throw new Error('Cannot update batch: batch ID is undefined');
        }
        
        // Update existing batch
        await api.updateBatch(existingBatchId, {
          name: batchName.trim(),
          description: batchDescription.trim() || undefined,
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
          destination_org: destinationOrg.trim() || undefined,
          migration_api: migrationAPI,
          exclude_releases: excludeReleases,
          exclude_attachments: excludeAttachments,
        });
        
        // Update repositories - add new ones, remove old ones
        const currentIds = new Set(currentBatchRepos.map((r) => r.id));
        const originalResponse = await api.listRepositories({ batch_id: existingBatchId });
        const originalRepos = originalResponse.repositories || [];
        const originalIds = new Set(originalRepos.map((r: Repository) => r.id));
        
        const toAdd = Array.from(currentIds).filter((id) => !originalIds.has(id));
        const toRemove = Array.from(originalIds).filter((id) => !currentIds.has(id));
        
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
          exclude_attachments: excludeAttachments,
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
    } catch (err: unknown) {
      // Extract error message from axios error response
      let errorMessage = 'Failed to save batch';
      
      const axiosError = err as { response?: { data?: { error?: string } }; message?: string };
      if (axiosError.response?.data?.error) {
        // Backend returned a structured error message
        errorMessage = axiosError.response.data.error;
      } else if (axiosError.message) {
        // Use the error message from the Error object
        errorMessage = axiosError.message;
      }
      
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const groupReposByOrg = useCallback((repos: Repository[]) => {
    const groups: Record<string, Repository[]> = {};
    repos.forEach((repo) => {
      // For ADO repos, group by project; for GitHub repos, group by org (first part of full_name)
      const groupKey = repo.ado_project || repo.full_name.split('/')[0];
      if (!groups[groupKey]) groups[groupKey] = [];
      groups[groupKey].push(repo);
    });
    return groups;
  }, []);

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
    <div className="h-full flex overflow-hidden" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
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
      {isAvailablePanelCollapsed ? (
        /* Collapsed State - Narrow vertical bar */
        <div 
          className="flex-shrink-0 border-r flex flex-col items-center py-4 transition-all duration-300"
          style={{ 
            backgroundColor: 'var(--bgColor-muted)', 
            borderColor: 'var(--borderColor-default)',
            width: '56px'
          }}
        >
          <IconButton
            icon={SidebarExpandIcon}
            aria-label="Expand available repositories"
            onClick={() => setIsAvailablePanelCollapsed(false)}
            size="small"
            variant="invisible"
          />
          <div 
            className="mt-4 flex flex-col items-center gap-1 cursor-pointer hover:opacity-80"
            onClick={() => setIsAvailablePanelCollapsed(false)}
            title="Click to expand and add more repositories"
          >
            <span 
              className="text-2xl font-bold"
              style={{ color: 'var(--fgColor-muted)' }}
            >
              {availableReposToShow.length}
            </span>
            <span 
              className="text-xs text-center px-1"
              style={{ color: 'var(--fgColor-muted)', writingMode: 'vertical-rl', textOrientation: 'mixed' }}
            >
              remaining
            </span>
          </div>
          {currentBatchRepos.length > 0 && (
            <div 
              className="mt-4 flex flex-col items-center gap-1"
              title={`${currentBatchRepos.length} repositories in batch`}
            >
              <span 
                className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"
                style={{ backgroundColor: 'var(--success-subtle)', color: 'var(--fgColor-success)' }}
              >
                {currentBatchRepos.length}
              </span>
              <span 
                className="text-xs"
                style={{ color: 'var(--fgColor-muted)', writingMode: 'vertical-rl', textOrientation: 'mixed' }}
              >
                in batch
              </span>
            </div>
          )}
        </div>
      ) : (
        /* Expanded State - Full panel */
        <div 
          className={`flex-1 min-w-0 min-h-0 grid grid-rows-[auto_1fr_auto] border-r transition-all duration-300 h-full overflow-hidden ${isAvailablePanelCollapsed ? 'lg:w-0' : currentBatchRepos.length > 0 ? 'lg:w-[45%]' : 'lg:w-[60%]'}`}
          style={{ backgroundColor: 'var(--bgColor-default)', borderColor: 'var(--borderColor-default)' }}
        >
          <div 
            className="p-4 border-b row-start-1 flex-shrink-0"
            style={{ borderColor: 'var(--borderColor-default)', backgroundColor: 'var(--bgColor-default)' }}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <IconButton
                  icon={SidebarCollapseIcon}
                  aria-label="Collapse available repositories"
                  onClick={() => setIsAvailablePanelCollapsed(true)}
                  size="small"
                  variant="invisible"
                />
                <div>
                  <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>Available Repositories</h3>
                  <p className="text-sm mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                    {availableLoading ? 'Loading...' : `${totalAvailable} repositories available`}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {/* Import Button - BorderedButton style */}
                <BorderedButton
                  onClick={handleImportClick}
                  disabled={loading}
                  leadingVisual={UploadIcon}
                  size="small"
                  title="Import repositories from file"
                >
                  Import
                </BorderedButton>
                
                {/* Add Matching Button - Blue bordered style */}
                {totalAvailable > 0 && availableReposToShow.length > 0 && (
                  <BorderedButton
                    onClick={handleAddAll}
                    disabled={availableLoading}
                    leadingVisual={PlusIcon}
                    size="small"
                    title="Add all repositories matching current filters to the batch"
                    className="btn-bordered-accent"
                  >
                    Add Matching ({totalAvailable})
                  </BorderedButton>
                )}
                
                {/* Selected count badge - positioned next to Add Selected */}
                {selectedRepoIds.size > 0 && (
                  <span 
                    className="px-2.5 py-1 rounded-full text-xs font-semibold"
                    style={{ backgroundColor: 'var(--accent-subtle)', color: 'var(--fgColor-accent)' }}
                  >
                    {selectedRepoIds.size} selected
                  </span>
                )}
                
                {/* Add Selected Button - Primary style */}
                <PrimaryButton
                  onClick={handleAddSelected}
                  disabled={selectedRepoIds.size === 0 || loading}
                  size="small"
                  title="Add only the selected repositories to the batch"
                >
                  Add Selected
                </PrimaryButton>
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

          {/* Bottom Section - Pagination */}
          <div 
            className="row-start-3 flex-shrink-0"
            style={{ backgroundColor: 'var(--bgColor-default)', borderTop: '1px solid var(--borderColor-default)' }}
          >
            <Pagination
              currentPage={currentPage}
              totalItems={totalAvailable}
              pageSize={pageSize}
              onPageChange={handlePageChange}
            />
          </div>
        </div>
      )}

      {/* Right Panel - Selected Repositories & Batch Info */}
      <div 
        className={`flex-shrink-0 flex flex-col transition-all duration-300 h-full min-h-0 overflow-hidden ${
          isAvailablePanelCollapsed 
            ? 'flex-1' 
            : currentBatchRepos.length > 0 
              ? 'w-full lg:w-[40%]' 
              : 'w-full lg:w-[30%]'
        }`}
        style={{ backgroundColor: 'var(--bgColor-default)' }}
      >
        <BatchSummaryPanel
          currentBatchRepos={currentBatchRepos}
          groupedRepos={currentGroups}
          totalSize={totalSize}
          onRemoveRepo={handleRemoveRepo}
          onClearAll={handleClearAll}
          isExpanded={isSelectedReposExpanded}
          onToggleExpanded={handleToggleSelectedRepos}
        />
        
        <BatchMetadataForm
          batchName={batchName}
          setBatchName={setBatchName}
          batchDescription={batchDescription}
          setBatchDescription={setBatchDescription}
          scheduledAt={scheduledAt}
          setScheduledAt={setScheduledAt}
          migrationSettings={{
            destinationOrg,
            migrationAPI,
            excludeReleases,
            excludeAttachments,
          }}
          onMigrationSettingsChange={(settings) => {
            if (settings.destinationOrg !== undefined) setDestinationOrg(settings.destinationOrg);
            if (settings.migrationAPI !== undefined) setMigrationAPI(settings.migrationAPI);
            if (settings.excludeReleases !== undefined) setExcludeReleases(settings.excludeReleases);
            if (settings.excludeAttachments !== undefined) setExcludeAttachments(settings.excludeAttachments);
          }}
          showMigrationSettings={showMigrationSettings}
          setShowMigrationSettings={setShowMigrationSettings}
          organizations={organizations}
          loading={loading}
          isEditMode={isEditMode}
          currentBatchReposCount={currentBatchRepos.length}
          error={error}
          onSave={handleSubmit}
          onClose={onClose}
        />
      </div>

      {/* Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={confirmDialog.isOpen}
        title={confirmDialog.title}
        message={confirmDialog.message}
        confirmLabel="OK"
        variant="danger"
        onConfirm={() => {
          setConfirmDialog(prev => ({ ...prev, isOpen: false }));
          confirmDialog.onConfirm();
        }}
        onCancel={() => setConfirmDialog(prev => ({ ...prev, isOpen: false }))}
      />

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

