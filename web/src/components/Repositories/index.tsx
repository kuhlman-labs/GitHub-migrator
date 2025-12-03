import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Button } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { RepoIcon, DownloadIcon, ChevronDownIcon, SquareIcon, XIcon } from '@primer/octicons-react';
import type { RepositoryFilters } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Pagination } from '../common/Pagination';
import { useRepositories } from '../../hooks/useQueries';
import { searchParamsToFilters, filtersToSearchParams } from '../../utils/filters';
import { UnifiedFilterSidebar } from '../common/UnifiedFilterSidebar';
import { RepositoryCard } from './RepositoryCard';
import { BulkActionsToolbar } from './BulkActionsToolbar';
import { exportToCSV, exportToExcel, exportToJSON, getTimestampedFilename } from '../../utils/export';

export function Repositories() {
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Parse filters from URL
  const urlFilters = searchParamsToFilters(searchParams);
  
  const [currentPage, setCurrentPage] = useState(1);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedRepositoryIds, setSelectedRepositoryIds] = useState<Set<number>>(new Set());
  const pageSize = 12;
  
  // Fetch repositories with filters
  const { data, isLoading, isFetching} = useRepositories(urlFilters);
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
  useEffect(() => {
    setCurrentPage(1);
    setSelectedRepositoryIds(new Set());
  }, [searchParams]);

  // Export functions
  const handleExport = async (format: 'csv' | 'excel' | 'json') => {
    setShowExportMenu(false);

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
      console.error('Export failed:', error);
      alert('Failed to export repositories. Please try again.');
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
                    {urlFilters.project ? (
                      // Project filter takes precedence (more specific than org)
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
                <div className="relative">
                  <Button
                    onClick={() => setShowExportMenu(!showExportMenu)}
                    disabled={repositories.length === 0}
                    leadingVisual={DownloadIcon}
                    trailingVisual={ChevronDownIcon}
                    variant="primary"
                  >
                    Export
                  </Button>
                  {showExportMenu && (
                    <>
                      {/* Backdrop to close menu when clicking outside */}
                      <div 
                        className="fixed inset-0 z-10" 
                        onClick={() => setShowExportMenu(false)}
                      />
                      {/* Dropdown menu */}
                      <div 
                        className="absolute right-0 mt-2 w-48 rounded-lg shadow-lg z-20"
                        style={{
                          backgroundColor: 'var(--bgColor-default)',
                          border: '1px solid var(--borderColor-default)',
                          boxShadow: 'var(--shadow-floating-large)'
                        }}
                      >
                        <div className="py-1">
                          <button
                            onClick={() => handleExport('csv')}
                            className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                            style={{ color: 'var(--fgColor-default)' }}
                          >
                            Export as CSV
                          </button>
                          <button
                            onClick={() => handleExport('excel')}
                            className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                            style={{ color: 'var(--fgColor-default)' }}
                          >
                            Export as Excel
                          </button>
                          <button
                            onClick={() => handleExport('json')}
                            className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                            style={{ color: 'var(--fgColor-default)' }}
                          >
                            Export as JSON
                          </button>
                        </div>
                      </div>
                    </>
                  )}
                </div>
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
              {activeFilterCount > 0 && (
                <Blankslate.PrimaryAction onClick={clearAllFilters}>
                  Clear All Filters
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
    </div>
  );
}
