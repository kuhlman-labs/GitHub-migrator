import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FormControl, TextInput, ActionMenu, ActionList } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { RepoIcon, DownloadIcon, SquareIcon, XIcon, TriangleDownIcon } from '@primer/octicons-react';
import { Button, BorderedButton, PrimaryButton } from '../common/buttons';
import { FormDialog } from '../common/FormDialog';
import type { RepositoryFilters } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Pagination } from '../common/Pagination';
import { useRepositories } from '../../hooks/useQueries';
import { useDiscoverRepositories } from '../../hooks/useMutations';
import { searchParamsToFilters, filtersToSearchParams } from '../../utils/filters';
import { UnifiedFilterSidebar } from '../common/UnifiedFilterSidebar';
import { RepositoryCard } from './RepositoryCard';
import { BulkActionsToolbar } from './BulkActionsToolbar';
import { exportToCSV, exportToExcel, exportToJSON, getTimestampedFilename } from '../../utils/export';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

export function Repositories() {
  const { showError, showSuccess } = useToast();
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Parse filters from URL
  const urlFilters = searchParamsToFilters(searchParams);
  
  const [currentPage, setCurrentPage] = useState(1);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedRepositoryIds, setSelectedRepositoryIds] = useState<Set<number>>(new Set());
  const [showDiscoverDialog, setShowDiscoverDialog] = useState(false);
  const [discoverOrg, setDiscoverOrg] = useState('');
  const pageSize = 12;
  
  // Fetch repositories with filters
  const { data, isLoading, isFetching} = useRepositories(urlFilters);
  const discoverRepositories = useDiscoverRepositories();
  const repositories = data?.repositories || [];

  // Selection helpers
  const toggleRepositorySelection = (id: number) => {
    setSelectedRepositoryIds(prev => {
      const newSet = new Set(prev);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
  };

  const selectAllOnPage = () => {
    const pageIds = paginatedRepos.map(r => r.id);
    setSelectedRepositoryIds(prev => {
      const newSet = new Set(prev);
      pageIds.forEach(id => newSet.add(id));
      return newSet;
    });
  };

  const clearSelection = () => {
    setSelectedRepositoryIds(new Set());
    setSelectionMode(false);
  };

  const toggleSelectionMode = () => {
    setSelectionMode(!selectionMode);
    if (selectionMode) {
      setSelectedRepositoryIds(new Set());
    }
  };

  // Update filters and URL
  const updateFilters = (newFilters: Partial<RepositoryFilters>) => {
    const merged = { ...urlFilters, ...newFilters };
    // Remove undefined values
    Object.keys(merged).forEach(key => {
      if (merged[key as keyof RepositoryFilters] === undefined) {
        delete merged[key as keyof RepositoryFilters];
      }
    });
    const params = filtersToSearchParams(merged);
    setSearchParams(params);
  };

  const clearAllFilters = () => {
    setSearchParams(new URLSearchParams());
  };

  // Count active filters
  const getActiveFilterCount = () => {
    let count = 0;
    if (urlFilters.status) count++;
    if (urlFilters.organization) count++;
    if (urlFilters.ado_organization) count++;
    if (urlFilters.project) count++;
    if (urlFilters.team) count++;
    if (urlFilters.search) count++;
    if (urlFilters.complexity) count++;
    if (urlFilters.size_category) count++;
    if (urlFilters.min_size || urlFilters.max_size) count++;
    if (urlFilters.has_lfs) count++;
    if (urlFilters.has_submodules) count++;
    if (urlFilters.has_large_files) count++;
    if (urlFilters.has_actions) count++;
    if (urlFilters.has_wiki) count++;
    if (urlFilters.has_pages) count++;
    if (urlFilters.has_discussions) count++;
    if (urlFilters.has_projects) count++;
    if (urlFilters.has_packages) count++;
    if (urlFilters.has_branch_protections) count++;
    if (urlFilters.has_rulesets) count++;
    if (urlFilters.is_archived !== undefined) count++;
    if (urlFilters.is_fork !== undefined) count++;
    if (urlFilters.has_code_scanning) count++;
    if (urlFilters.has_dependabot) count++;
    if (urlFilters.has_secret_scanning) count++;
    if (urlFilters.has_codeowners) count++;
    if (urlFilters.has_self_hosted_runners) count++;
    if (urlFilters.has_release_assets) count++;
    if (urlFilters.has_webhooks) count++;
    if (urlFilters.visibility) count++;
    if (urlFilters.sort_by && urlFilters.sort_by !== 'name') count++;
    return count;
  };

  const activeFilterCount = getActiveFilterCount();

  // Helper function to format filter values (handles both strings and arrays)
  const formatFilterValue = (value: string | string[] | undefined): string => {
    if (!value) return '';
    if (Array.isArray(value)) {
      return value.join(', ');
    }
    return value;
  };

  // Paginate
  const totalItems = repositories.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedRepos = repositories.slice(startIndex, endIndex);

  // Reset page and selection when filters change
  // This is a valid use case: resetting derived state when URL params change
  const searchParamsKey = searchParams.toString();
  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect */
    setCurrentPage(1);
    setSelectedRepositoryIds(new Set());
    /* eslint-enable react-hooks/set-state-in-effect */
  }, [searchParamsKey]);

  // Export functions
  const handleExport = async (format: 'csv' | 'excel' | 'json') => {
    if (repositories.length === 0) {
      alert('No repositories to export');
      return;
    }

    const baseName = 'repositories';
    
    try {
      switch (format) {
        case 'csv':
          exportToCSV(repositories, getTimestampedFilename(baseName, 'csv'));
          break;
        case 'excel':
          await exportToExcel(repositories, getTimestampedFilename(baseName, 'xlsx'));
          break;
        case 'json':
          exportToJSON(repositories, getTimestampedFilename(baseName, 'json'));
          break;
      }
    } catch (error) {
      handleApiError(error, showError, 'Failed to export repositories');
    }
  };

  return (
    <div className="h-full flex" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
      {/* Filter Sidebar */}
      <UnifiedFilterSidebar
        filters={urlFilters}
        onChange={updateFilters}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
        showStatus={true}
        showSearch={true}
      />

      {/* Main Content */}
      <div className="flex-1 min-w-0">
        <div className="relative max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <RefreshIndicator isRefreshing={isFetching && !isLoading} />
          
          {/* Header */}
          <div className="mb-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                {/* Selection Mode Toggle - Left Side */}
                <button
                  onClick={toggleSelectionMode}
                  className="p-2 rounded transition-colors hover:bg-[var(--control-bgColor-hover)]"
                  style={{
                    backgroundColor: selectionMode ? 'var(--control-bgColor-rest)' : 'transparent',
                    border: selectionMode ? '1px solid var(--borderColor-default)' : 'none',
                  }}
                  aria-label={selectionMode ? "Exit selection mode" : "Enter selection mode"}
                  title={selectionMode ? "Exit selection mode" : "Select repositories for batch operations"}
                >
                  {selectionMode ? (
                    <span style={{ color: 'var(--fgColor-default)' }}>
                      <XIcon size={24} />
                    </span>
                  ) : (
                    <span style={{ color: 'var(--fgColor-muted)' }}>
                      <SquareIcon size={24} />
                    </span>
                  )}
                </button>

                {/* Title and Info */}
                <div>
                  <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                    {urlFilters.team ? (
                      // Team filter takes precedence (most specific)
                      <>Repositories in Team <span style={{ color: 'var(--fgColor-accent)' }}>{formatFilterValue(urlFilters.team)}</span></>
                    ) : urlFilters.project ? (
                      // Project filter (more specific than org)
                      <>Repositories in Project <span style={{ color: 'var(--fgColor-accent)' }}>{formatFilterValue(urlFilters.project)}</span></>
                    ) : urlFilters.ado_organization ? (
                      <>Repositories in Organization <span style={{ color: 'var(--fgColor-accent)' }}>{formatFilterValue(urlFilters.ado_organization)}</span></>
                    ) : urlFilters.organization ? (
                      <>Repositories in Organization <span style={{ color: 'var(--fgColor-accent)' }}>{formatFilterValue(urlFilters.organization)}</span></>
                    ) : (
                      'Repositories'
                    )}
                  </h1>
                  <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                    {totalItems > 0 ? (
                      <>
                        Showing {startIndex + 1}-{Math.min(endIndex, totalItems)} of {totalItems} repositories
                        {activeFilterCount > 0 && ` with ${activeFilterCount} active filter${activeFilterCount > 1 ? 's' : ''}`}
                        {selectionMode && selectedRepositoryIds.size > 0 && ` · ${selectedRepositoryIds.size} selected`}
                      </>
                    ) : (
                      'No repositories found'
                    )}
                  </p>
                </div>
              </div>

              {/* Right Side Actions */}
              <div className="flex items-center gap-3">
                {/* Select All / Clear Selection (when in selection mode) */}
                {selectionMode && (
                  <>
                    {selectedRepositoryIds.size > 0 ? (
                      <Button onClick={clearSelection} variant="invisible">
                        Clear Selection
                      </Button>
                    ) : (
                      <Button onClick={selectAllOnPage} variant="invisible">
                        Select All on Page
                      </Button>
                    )}
                  </>
                )}

                {activeFilterCount > 0 && !selectionMode && (
                  <Button
                    onClick={clearAllFilters}
                    variant="invisible"
                  >
                    Clear All Filters
                  </Button>
                )}

                {/* Export Button with Dropdown */}
                <ActionMenu>
                  <ActionMenu.Anchor>
                    <BorderedButton
                      disabled={repositories.length === 0}
                      leadingVisual={DownloadIcon}
                      trailingAction={TriangleDownIcon}
                    >
                      Export
                    </BorderedButton>
                  </ActionMenu.Anchor>
                  <ActionMenu.Overlay>
                    <ActionList>
                      <ActionList.Item onSelect={() => handleExport('csv')}>
                        Export as CSV
                      </ActionList.Item>
                      <ActionList.Item onSelect={() => handleExport('excel')}>
                        Export as Excel
                      </ActionList.Item>
                      <ActionList.Item onSelect={() => handleExport('json')}>
                        Export as JSON
                      </ActionList.Item>
                    </ActionList>
                  </ActionMenu.Overlay>
                </ActionMenu>

                {/* Discover Repos Button */}
                <PrimaryButton
                  onClick={() => setShowDiscoverDialog(true)}
                  leadingVisual={RepoIcon}
                  disabled={discoverRepositories.isPending}
                >
                  {discoverRepositories.isPending ? 'Discovering...' : 'Discover Repos'}
                </PrimaryButton>
              </div>
            </div>
          </div>

          {/* Repository Grid */}
          {isLoading ? (
            <LoadingSpinner />
          ) : repositories.length === 0 ? (
            <Blankslate border>
              <Blankslate.Visual>
                <RepoIcon size={48} />
              </Blankslate.Visual>
              <Blankslate.Heading>No repositories found</Blankslate.Heading>
              <Blankslate.Description>
                {activeFilterCount > 0
                  ? 'Try adjusting your filters to find repositories.'
                  : 'No repositories have been discovered yet. Start by discovering repositories from your organizations.'}
              </Blankslate.Description>
              {activeFilterCount > 0 ? (
                <Blankslate.PrimaryAction onClick={clearAllFilters}>
                  Clear All Filters
                </Blankslate.PrimaryAction>
              ) : (
                <Blankslate.PrimaryAction onClick={() => setShowDiscoverDialog(true)}>
                  Discover Repos
                </Blankslate.PrimaryAction>
              )}
            </Blankslate>
          ) : (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
                {paginatedRepos.map((repo) => (
                  <RepositoryCard 
                    key={repo.id} 
                    repository={repo}
                    selectionMode={selectionMode}
                    selected={selectedRepositoryIds.has(repo.id)}
                    onToggleSelect={toggleRepositorySelection}
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

      {/* Bulk Actions Toolbar */}
      {selectionMode && selectedRepositoryIds.size > 0 && (
        <BulkActionsToolbar
          selectedCount={selectedRepositoryIds.size}
          selectedIds={Array.from(selectedRepositoryIds)}
          onClearSelection={clearSelection}
        />
      )}

      {/* Discover Repos Dialog */}
      <FormDialog
        isOpen={showDiscoverDialog}
        title="Discover Repositories"
        submitLabel={discoverRepositories.isPending ? 'Discovering...' : 'Discover'}
        onSubmit={() => {
          if (!discoverOrg.trim()) return;
          discoverRepositories.mutate(discoverOrg.trim(), {
            onSuccess: (data) => {
              showSuccess(data.message || 'Discovery started!');
              setDiscoverOrg('');
              setShowDiscoverDialog(false);
            },
            onError: (error) => {
              showError(error instanceof Error ? error.message : 'Discovery failed');
            },
          });
        }}
        onCancel={() => {
          setShowDiscoverDialog(false);
          setDiscoverOrg('');
        }}
        isLoading={discoverRepositories.isPending}
        isSubmitDisabled={!discoverOrg.trim()}
      >
        <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
          Discover all repositories from a GitHub organization. This will start repository discovery and profiling.
        </p>
        <FormControl>
          <FormControl.Label>Source Organization</FormControl.Label>
          <TextInput
            value={discoverOrg}
            onChange={(e) => setDiscoverOrg(e.target.value)}
            placeholder="e.g., my-org"
            block
          />
          <FormControl.Caption>
            Enter the GitHub organization to discover repositories from
          </FormControl.Caption>
        </FormControl>
      </FormDialog>
    </div>
  );
}
