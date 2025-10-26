import { useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import type { Repository, RepositoryFilters } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { formatBytes } from '../../utils/format';
import { useRepositories } from '../../hooks/useQueries';
import { searchParamsToFilters, filtersToSearchParams } from '../../utils/filters';

// Complexity mapping for display
const COMPLEXITY_LABELS: Record<string, string> = {
  simple: 'Simple',
  medium: 'Medium',
  complex: 'Complex',
  very_complex: 'Very Complex',
};

// Size category mapping
const SIZE_CATEGORY_LABELS: Record<string, string> = {
  small: 'Small (<100MB)',
  medium: 'Medium (100MB-1GB)',
  large: 'Large (1GB-5GB)',
  very_large: 'Very Large (>5GB)',
  unknown: 'Unknown Size',
};

export function Repositories() {
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Parse filters from URL
  const urlFilters = searchParamsToFilters(searchParams);
  
  // Initialize local search from URL (directly in state initialization)
  const [localSearch, setLocalSearch] = useState(urlFilters.search || '');
  
  // Fetch repositories with filters - backend now supports all filters
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

  const removeFilter = (filterKey: keyof RepositoryFilters) => {
    const newFilters = { ...urlFilters };
    delete newFilters[filterKey];
    const params = filtersToSearchParams(newFilters);
    setSearchParams(params);
  };

  const clearAllFilters = () => {
    setSearchParams(new URLSearchParams());
    setLocalSearch('');
  };

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    updateFilters({ search: localSearch || undefined });
  };

  // Count active filters
  const getActiveFilterCount = () => {
    let count = 0;
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
    if (urlFilters.has_code_scanning) count++;
    if (urlFilters.has_dependabot) count++;
    if (urlFilters.has_secret_scanning) count++;
    if (urlFilters.has_codeowners) count++;
    if (urlFilters.has_self_hosted_runners) count++;
    if (urlFilters.has_release_assets) count++;
    if (urlFilters.is_archived !== undefined) count++;
    if (urlFilters.is_fork !== undefined) count++;
    if (urlFilters.complexity) count++;
    if (urlFilters.size_category) count++;
    if (urlFilters.organization) count++;
    if (urlFilters.status) count++;
    if (urlFilters.min_size || urlFilters.max_size) count++;
    return count;
  };

  const activeFilterCount = getActiveFilterCount();

  return (
    <div className="max-w-7xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-semibold text-gh-text-primary">Repositories</h1>
            <p className="text-sm text-gh-text-secondary mt-1">
              {repositories.length} repositories found
              {activeFilterCount > 0 && ` with ${activeFilterCount} active filter${activeFilterCount > 1 ? 's' : ''}`}
            </p>
          </div>
          <Link
            to="/analytics"
            className="px-4 py-2 text-sm font-medium text-gh-blue hover:text-gh-blue-hover"
          >
            ← Back to Analytics
          </Link>
        </div>

        {/* Search Bar */}
        <form onSubmit={handleSearchSubmit} className="mb-4">
          <div className="flex gap-2">
            <input
              type="text"
              placeholder="Search repositories..."
              value={localSearch}
              onChange={(e) => setLocalSearch(e.target.value)}
              className="flex-1 px-3 py-2 text-sm border border-gh-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-gh-blue focus:border-transparent"
            />
            <button
              type="submit"
              className="px-4 py-2 text-sm font-medium bg-gh-blue text-white rounded-md hover:bg-gh-blue-hover transition-colors"
            >
              Search
            </button>
            {localSearch && (
              <button
                type="button"
                onClick={() => {
                  setLocalSearch('');
                  removeFilter('search');
                }}
                className="px-4 py-2 text-sm font-medium text-gh-text-secondary hover:text-gh-text-primary transition-colors"
              >
                Clear
              </button>
            )}
          </div>
        </form>

        {/* Active Filters Display */}
        {activeFilterCount > 0 && (
          <div className="bg-white rounded-lg border border-gh-border-default shadow-gh-card p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-gh-text-primary">Active Filters</h3>
              <button
                onClick={clearAllFilters}
                className="text-sm text-gh-danger hover:underline font-medium"
              >
                Clear All
              </button>
            </div>
            <div className="flex flex-wrap gap-2">
              {/* Complexity filters */}
              {urlFilters.complexity && (
                <FilterBadge
                  label={Array.isArray(urlFilters.complexity) 
                    ? urlFilters.complexity.map(c => COMPLEXITY_LABELS[c] || c).join(', ')
                    : COMPLEXITY_LABELS[urlFilters.complexity] || urlFilters.complexity
                  }
                  onRemove={() => removeFilter('complexity')}
                />
              )}

              {/* Size category filters */}
              {urlFilters.size_category && (
                <FilterBadge
                  label={Array.isArray(urlFilters.size_category)
                    ? urlFilters.size_category.map(s => SIZE_CATEGORY_LABELS[s] || s).join(', ')
                    : SIZE_CATEGORY_LABELS[urlFilters.size_category] || urlFilters.size_category
                  }
                  onRemove={() => removeFilter('size_category')}
                />
              )}

              {/* Feature filters */}
              {urlFilters.has_lfs && <FilterBadge label="LFS" onRemove={() => removeFilter('has_lfs')} />}
              {urlFilters.has_submodules && <FilterBadge label="Submodules" onRemove={() => removeFilter('has_submodules')} />}
              {urlFilters.has_large_files && <FilterBadge label="Large Files (>100MB)" onRemove={() => removeFilter('has_large_files')} />}
              {urlFilters.has_actions && <FilterBadge label="GitHub Actions" onRemove={() => removeFilter('has_actions')} />}
              {urlFilters.has_wiki && <FilterBadge label="Wiki" onRemove={() => removeFilter('has_wiki')} />}
              {urlFilters.has_pages && <FilterBadge label="Pages" onRemove={() => removeFilter('has_pages')} />}
              {urlFilters.has_discussions && <FilterBadge label="Discussions" onRemove={() => removeFilter('has_discussions')} />}
              {urlFilters.has_projects && <FilterBadge label="Projects" onRemove={() => removeFilter('has_projects')} />}
              {urlFilters.has_packages && <FilterBadge label="Packages" onRemove={() => removeFilter('has_packages')} />}
              {urlFilters.has_branch_protections && <FilterBadge label="Branch Protections" onRemove={() => removeFilter('has_branch_protections')} />}
              {urlFilters.has_rulesets && <FilterBadge label="Rulesets" onRemove={() => removeFilter('has_rulesets')} />}
              {urlFilters.has_code_scanning && <FilterBadge label="Code Scanning" onRemove={() => removeFilter('has_code_scanning')} />}
              {urlFilters.has_dependabot && <FilterBadge label="Dependabot" onRemove={() => removeFilter('has_dependabot')} />}
              {urlFilters.has_secret_scanning && <FilterBadge label="Secret Scanning" onRemove={() => removeFilter('has_secret_scanning')} />}
              {urlFilters.has_codeowners && <FilterBadge label="CODEOWNERS" onRemove={() => removeFilter('has_codeowners')} />}
              {urlFilters.has_self_hosted_runners && <FilterBadge label="Self-Hosted Runners" onRemove={() => removeFilter('has_self_hosted_runners')} />}
              {urlFilters.has_release_assets && <FilterBadge label="Release Assets" onRemove={() => removeFilter('has_release_assets')} />}
              {urlFilters.is_fork !== undefined && (
                <FilterBadge 
                  label={urlFilters.is_fork ? "Fork" : "Not Fork"} 
                  onRemove={() => removeFilter('is_fork')} 
                />
              )}
              {urlFilters.is_archived !== undefined && (
                <FilterBadge 
                  label={urlFilters.is_archived ? "Archived" : "Not Archived"} 
                  onRemove={() => removeFilter('is_archived')} 
                />
              )}

              {/* Organization filter */}
              {urlFilters.organization && (
                <FilterBadge
                  label={`Org: ${Array.isArray(urlFilters.organization) ? urlFilters.organization.join(', ') : urlFilters.organization}`}
                  onRemove={() => removeFilter('organization')}
                />
              )}

              {/* Status filter */}
              {urlFilters.status && (
                <FilterBadge
                  label={`Status: ${urlFilters.status}`}
                  onRemove={() => removeFilter('status')}
                />
              )}

              {/* Size range filter */}
              {(urlFilters.min_size || urlFilters.max_size) && (
                <FilterBadge
                  label={`Size: ${urlFilters.min_size ? formatBytes(urlFilters.min_size) : '0'} - ${urlFilters.max_size ? formatBytes(urlFilters.max_size) : '∞'}`}
                  onRemove={() => {
                    removeFilter('min_size');
                    removeFilter('max_size');
                  }}
                />
              )}
            </div>
          </div>
        )}
      </div>

      {/* Repository Grid */}
      {isLoading ? (
        <LoadingSpinner />
      ) : repositories.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg border border-gh-border-default">
          <svg
            className="mx-auto h-12 w-12 text-gh-text-secondary"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gh-text-primary">No repositories found</h3>
          <p className="mt-1 text-sm text-gh-text-secondary">
            Try adjusting your filters or search term.
          </p>
          {activeFilterCount > 0 && (
            <button
              onClick={clearAllFilters}
              className="mt-4 px-4 py-2 text-sm font-medium bg-gh-blue text-white rounded-md hover:bg-gh-blue-hover"
            >
              Clear All Filters
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {repositories.map((repo) => (
            <RepositoryCard key={repo.id} repository={repo} />
          ))}
        </div>
      )}
    </div>
  );
}

function FilterBadge({ label, onRemove }: { label: string; onRemove: () => void }) {
  return (
    <span className="inline-flex items-center gap-2 px-3 py-1 bg-gh-info-bg text-gh-blue rounded-full text-sm font-medium border border-gh-blue/20">
      {label}
      <button
        onClick={onRemove}
        className="hover:bg-gh-blue/10 rounded-full p-0.5 transition-colors"
        aria-label={`Remove ${label} filter`}
      >
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>
    </span>
  );
}

function RepositoryCard({ repository }: { repository: Repository }) {
  return (
    <Link
      to={`/repository/${encodeURIComponent(repository.full_name)}`}
      className="bg-white rounded-lg border border-gh-border-default hover:border-gh-border-hover transition-colors p-6 block shadow-gh-card"
    >
      <h3 className="text-base font-semibold text-gh-text-primary mb-3 truncate">
        {repository.full_name}
      </h3>
      <div className="mb-3 flex items-center justify-between">
        <StatusBadge status={repository.status} size="sm" />
      </div>
      <div className="space-y-1.5 text-sm text-gh-text-secondary mb-3">
        <div>Size: {formatBytes(repository.total_size)}</div>
        <div>Branches: {repository.branch_count}</div>
      </div>
      
      {/* Timestamps */}
      <div className="space-y-1 mb-3 border-t border-gh-border-default pt-3">
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
        {repository.visibility === 'internal' && <Badge color="yellow">Internal</Badge>}
        {repository.has_release_assets && <Badge color="pink">Releases</Badge>}
      </div>
    </Link>
  );
}

