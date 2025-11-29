import { useState, useEffect } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { Button } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { RepoIcon, DownloadIcon, ChevronDownIcon } from '@primer/octicons-react';
import type { Repository, RepositoryFilters } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { Pagination } from '../common/Pagination';
import { formatBytes } from '../../utils/format';
import { useRepositories } from '../../hooks/useQueries';
import { searchParamsToFilters, filtersToSearchParams } from '../../utils/filters';
import { RepositoryFilterSidebar } from './RepositoryFilterSidebar';
import { exportToCSV, exportToExcel, exportToJSON, getTimestampedFilename } from '../../utils/export';

export function Repositories() {
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Parse filters from URL
  const urlFilters = searchParamsToFilters(searchParams);
  
  const [currentPage, setCurrentPage] = useState(1);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);
  const pageSize = 12;
  
  // Fetch repositories with filters
  const { data, isLoading, isFetching} = useRepositories(urlFilters);
  const repositories = data?.repositories || [];

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

  // Paginate
  const totalItems = repositories.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedRepos = repositories.slice(startIndex, endIndex);

  // Reset page when filters change
  useEffect(() => {
    setCurrentPage(1);
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
      <RepositoryFilterSidebar
        filters={urlFilters}
        onChange={updateFilters}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
      />

      {/* Main Content */}
      <div className="flex-1 min-w-0">
        <div className="relative max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <RefreshIndicator isRefreshing={isFetching && !isLoading} />
          
          {/* Header */}
          <div className="mb-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>Repositories</h1>
                <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                  {totalItems > 0 ? (
                    <>
                      Showing {startIndex + 1}-{Math.min(endIndex, totalItems)} of {totalItems} repositories
                      {activeFilterCount > 0 && ` with ${activeFilterCount} active filter${activeFilterCount > 1 ? 's' : ''}`}
                    </>
                  ) : (
                    'No repositories found'
                  )}
                </p>
              </div>
              <div className="flex items-center gap-3">
                {activeFilterCount > 0 && (
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
                  <RepositoryCard key={repo.id} repository={repo} />
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
  );
}

function RepositoryCard({ repository }: { repository: Repository }) {
  return (
    <Link
      to={`/repository/${encodeURIComponent(repository.full_name)}`}
      className="rounded-lg border transition-opacity hover:opacity-80 p-6 block"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)'
      }}
    >
      <h3 className="text-base font-semibold mb-3 truncate" style={{ color: 'var(--fgColor-default)' }}>
        {repository.full_name}
      </h3>
      <div className="mb-3 flex items-center justify-between">
        <StatusBadge status={repository.status} size="small" />
      </div>
      <div className="space-y-1.5 text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
        <div>Size: {formatBytes(repository.total_size)}</div>
        <div>Branches: {repository.branch_count}</div>
      </div>
      
      {/* Timestamps */}
      <div 
        className="space-y-1 mb-3 pt-3"
        style={{ borderTop: '1px solid var(--borderColor-default)' }}
      >
        {repository.last_discovery_at && (
          <TimestampDisplay 
            timestamp={repository.last_discovery_at} 
            label="Discovered"
            size="sm"
          />
        )}
        {repository.last_dry_run_at && (
          <TimestampDisplay 
            timestamp={repository.last_dry_run_at} 
            label="Dry run"
            size="sm"
          />
        )}
      </div>

      <div className="flex gap-1.5 flex-wrap">
        {repository.is_archived && <Badge color="gray">Archived</Badge>}
        {repository.is_fork && <Badge color="purple">Fork</Badge>}
        {repository.has_lfs && <Badge color="blue">LFS</Badge>}
        {repository.has_submodules && <Badge color="purple">Submodules</Badge>}
        {repository.has_large_files && <Badge color="orange">Large Files</Badge>}
        {repository.has_actions && <Badge color="green">Actions</Badge>}
        {repository.has_packages && <Badge color="orange">Packages</Badge>}
        {repository.has_wiki && <Badge color="yellow">Wiki</Badge>}
        {repository.has_pages && <Badge color="pink">Pages</Badge>}
        {repository.has_discussions && <Badge color="indigo">Discussions</Badge>}
        {repository.has_projects && <Badge color="teal">Projects</Badge>}
        {repository.branch_protections > 0 && <Badge color="red">Protected</Badge>}
        {repository.has_rulesets && <Badge color="red">Rulesets</Badge>}
        {repository.has_code_scanning && <Badge color="green">Code Scanning</Badge>}
        {repository.has_dependabot && <Badge color="green">Dependabot</Badge>}
        {repository.has_secret_scanning && <Badge color="green">Secret Scanning</Badge>}
        {repository.has_codeowners && <Badge color="blue">CODEOWNERS</Badge>}
        {repository.has_self_hosted_runners && <Badge color="purple">Self-Hosted</Badge>}
        {repository.visibility === 'public' && <Badge color="blue">Public</Badge>}
        {repository.visibility === 'internal' && <Badge color="yellow">Internal</Badge>}
        {repository.has_release_assets && <Badge color="pink">Releases</Badge>}
      </div>
    </Link>
  );
}
