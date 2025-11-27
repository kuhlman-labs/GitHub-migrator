import { useState, useEffect } from 'react';
import { useParams, Link as RouterLink } from 'react-router-dom';
import { TextInput, Button, Token, Checkbox, FormControl, Link } from '@primer/react';
import { SearchIcon, FilterIcon } from '@primer/octicons-react';
import type { Repository, ADOProject, Organization } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { Pagination } from '../common/Pagination';
import { formatBytes } from '../../utils/format';
import { useRepositories, useOrganizations } from '../../hooks/useQueries';
import { api } from '../../services/api';

type FeatureFilter = {
  key: keyof Repository;
  label: string;
  color: string;
};

// Azure DevOps specific features
const ADO_FEATURE_FILTERS: FeatureFilter[] = [
  { key: 'ado_has_boards', label: 'Azure Boards', color: 'purple' },
  { key: 'ado_has_pipelines', label: 'Azure Pipelines', color: 'green' },
  { key: 'ado_has_ghas', label: 'GHAS (ADO)', color: 'green' },
  { key: 'ado_pull_request_count', label: 'Has Pull Requests', color: 'indigo' },
  { key: 'ado_work_item_count', label: 'Has Work Items', color: 'purple' },
  { key: 'ado_branch_policy_count', label: 'Has Branch Policies', color: 'red' },
];

// GitHub specific features
const GITHUB_FEATURE_FILTERS: FeatureFilter[] = [
  { key: 'is_archived', label: 'Archived', color: 'gray' },
  { key: 'is_fork', label: 'Fork', color: 'purple' },
  { key: 'has_lfs', label: 'LFS', color: 'blue' },
  { key: 'has_submodules', label: 'Submodules', color: 'purple' },
  { key: 'has_large_files', label: 'Large Files (>100MB)', color: 'orange' },
  { key: 'has_actions', label: 'GitHub Actions', color: 'green' },
  { key: 'has_wiki', label: 'Wiki', color: 'yellow' },
  { key: 'has_pages', label: 'Pages', color: 'pink' },
  { key: 'has_discussions', label: 'Discussions', color: 'indigo' },
  { key: 'has_projects', label: 'Projects', color: 'teal' },
  { key: 'has_packages', label: 'Packages', color: 'orange' },
  { key: 'branch_protections', label: 'Branch Protections', color: 'red' },
  { key: 'has_rulesets', label: 'Rulesets', color: 'red' },
  { key: 'has_code_scanning', label: 'Code Scanning', color: 'green' },
  { key: 'has_dependabot', label: 'Dependabot', color: 'green' },
  { key: 'has_secret_scanning', label: 'Secret Scanning', color: 'green' },
  { key: 'has_codeowners', label: 'CODEOWNERS', color: 'blue' },
  { key: 'has_self_hosted_runners', label: 'Self-Hosted Runners', color: 'purple' },
  { key: 'has_release_assets', label: 'Release Assets', color: 'pink' },
  { key: 'webhook_count', label: 'Webhooks', color: 'indigo' },
];

// Map simplified filter values to actual backend statuses
const STATUS_MAP: Record<string, string[]> = {
  all: [],
  pending: ['pending'],
  remediation_required: ['remediation_required'],
  in_progress: [
    'dry_run_queued',
    'dry_run_in_progress',
    'pre_migration',
    'archive_generating',
    'queued_for_migration',
    'migrating_content',
    'post_migration',
  ],
  complete: ['dry_run_complete', 'migration_complete', 'complete'],
  failed: ['dry_run_failed', 'migration_failed'],
  rolled_back: ['rolled_back'],
  wont_migrate: ['wont_migrate'],
};

export function OrganizationDetail() {
  const { orgName, projectName } = useParams<{ orgName: string; projectName?: string }>();
  const [filter, setFilter] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedFeatures, setSelectedFeatures] = useState<Set<keyof Repository>>(new Set());
  const [showFilters, setShowFilters] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 12;
  const [isADOOrg, setIsADOOrg] = useState(false);
  const [adoProjects, setAdoProjects] = useState<ADOProject[]>([]);
  const [projectsLoading, setProjectsLoading] = useState(false);
  const [projectSearchTerm, setProjectSearchTerm] = useState('');
  const [projectCurrentPage, setProjectCurrentPage] = useState(1);
  const projectPageSize = 12;

  const { data, isLoading, isFetching } = useRepositories({});
  const { data: orgsData } = useOrganizations();

  // Reset pagination when organization or project changes
  useEffect(() => {
    setCurrentPage(1);
    setProjectCurrentPage(1);
  }, [orgName, projectName]);

  // Reset repository pagination when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [filter, searchTerm, selectedFeatures]);

  // Reset project pagination when project search changes
  useEffect(() => {
    setProjectCurrentPage(1);
  }, [projectSearchTerm]);

  // Determine what we're viewing based on route params
  // If projectName exists, we're viewing an ADO project (show repos)
  // If only orgName exists, check if it's an ADO org (show projects) or GitHub org (show repos)
  useEffect(() => {
    const checkViewType = async () => {
      // If projectName is in the URL, we're viewing a project (show repos)
      if (projectName) {
        setIsADOOrg(false);
        setAdoProjects([]);
        return;
      }

      // Otherwise, check if orgName is an ADO organization
      const organizations = orgsData || [];
      const org = organizations.find((o: Organization) => o.organization === orgName);
      
      // It's an ADO org if it has total_projects defined
      if (org && org.total_projects !== undefined) {
        setIsADOOrg(true);
        // Fetch projects for this organization
        setProjectsLoading(true);
        try {
          const response = await api.listADOProjects(orgName);
          // Ensure we have an array
          setAdoProjects(Array.isArray(response) ? response : []);
        } catch (error) {
          console.error('Failed to load ADO projects:', error);
          setAdoProjects([]); // Set empty array on error
        } finally {
          setProjectsLoading(false);
        }
      } else {
        setIsADOOrg(false);
        setAdoProjects([]); // Reset projects when not an ADO org
      }
    };

    if (orgName) {
      checkViewType();
    }
  }, [orgName, projectName, orgsData]);

  // Filter repositories for this organization/project (client-side)
  const repositories = (data?.repositories || []).filter((repo: Repository) => {
    // If viewing an ADO project, filter by project name
    if (projectName) {
      return repo.ado_project === projectName;
    }
    
    // For ADO repos without a specific project, check the ado_project field
    if (repo.ado_project) {
      return repo.ado_project === orgName;
    }
    
    // For GitHub repos, extract org from full_name
    const org = repo.full_name.split('/')[0];
    return org === orgName;
  });

  const toggleFeature = (feature: keyof Repository) => {
    const newSelected = new Set(selectedFeatures);
    if (newSelected.has(feature)) {
      newSelected.delete(feature);
    } else {
      newSelected.add(feature);
    }
    setSelectedFeatures(newSelected);
  };

  const clearAllFilters = () => {
    setSelectedFeatures(new Set());
    setFilter('all');
    setSearchTerm('');
  };

  // Calculate dynamic feature counts based on currently filtered repos
  const getFeatureCount = (feature: keyof Repository) => {
    // Get repos that match status and search filters
    let baseRepos = repositories.filter((repo: Repository) => {
      // Status filter
      if (filter !== 'all') {
        const allowedStatuses = STATUS_MAP[filter] || [];
        if (!allowedStatuses.includes(repo.status)) {
          return false;
        }
      }

      // Search filter
      if (searchTerm && !repo.full_name.toLowerCase().includes(searchTerm.toLowerCase())) {
        return false;
      }

      return true;
    });

    // Apply all OTHER selected features (not the current one)
    if (selectedFeatures.size > 0) {
      baseRepos = baseRepos.filter((repo: Repository) => {
        for (const selectedFeature of selectedFeatures) {
          // Skip the feature we're counting
          if (selectedFeature === feature) continue;
          
          const value = repo[selectedFeature];
          const hasFeature = typeof value === 'boolean' ? value : (typeof value === 'number' && value > 0);
          if (!hasFeature) {
            return false;
          }
        }
        return true;
      });
    }

    // Count how many of these repos have the target feature
    return baseRepos.filter((repo: Repository) => {
      const value = repo[feature];
      return typeof value === 'boolean' ? value : (typeof value === 'number' && value > 0);
    }).length;
  };

  const filteredRepos = repositories.filter((repo: Repository) => {
    // Status filter - check if repo status matches any of the mapped backend statuses
    if (filter !== 'all') {
      const allowedStatuses = STATUS_MAP[filter] || [];
      if (!allowedStatuses.includes(repo.status)) {
        return false;
      }
    }

    // Search filter
    if (searchTerm && !repo.full_name.toLowerCase().includes(searchTerm.toLowerCase())) {
      return false;
    }

    // Feature filters (AND logic - repo must have ALL selected features)
    if (selectedFeatures.size > 0) {
      for (const feature of selectedFeatures) {
        const value = repo[feature];
        const hasFeature = typeof value === 'boolean' ? value : (typeof value === 'number' && value > 0);
        if (!hasFeature) {
          return false;
        }
      }
    }

    return true;
  });

  const statuses = ['all', 'pending', 'remediation_required', 'in_progress', 'complete', 'failed', 'rolled_back', 'wont_migrate'];
  const hasActiveFilters = selectedFeatures.size > 0 || filter !== 'all' || searchTerm !== '';

  // Paginate
  const totalItems = filteredRepos.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedRepos = filteredRepos.slice(startIndex, endIndex);

  // Reset page when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [filter, searchTerm, selectedFeatures.size]);

  // Reset project page when search changes
  useEffect(() => {
    setProjectCurrentPage(1);
  }, [projectSearchTerm]);

  // Filter and paginate projects
  const filteredProjects = adoProjects.filter((project) => {
    if (projectSearchTerm && !project.name.toLowerCase().includes(projectSearchTerm.toLowerCase())) {
      return false;
    }
    return true;
  });

  const projectStartIndex = (projectCurrentPage - 1) * projectPageSize;
  const projectEndIndex = projectStartIndex + projectPageSize;
  const paginatedProjects = filteredProjects.slice(projectStartIndex, projectEndIndex);
  const totalProjects = filteredProjects.length;

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      {/* Breadcrumbs */}
      <nav aria-label="Breadcrumb" className="mb-6">
        <ol className="flex items-center text-sm">
          <li>
            <Link as={RouterLink} to="/" muted>Dashboard</Link>
          </li>
          <li className="mx-2" style={{ color: 'var(--fgColor-muted)' }}>/</li>
          {projectName ? (
            <>
              <li>
                <Link as={RouterLink} to={`/org/${encodeURIComponent(orgName || '')}`} muted>
                  {orgName}
                </Link>
              </li>
              <li className="mx-2" style={{ color: 'var(--fgColor-muted)' }}>/</li>
              <li className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>{projectName}</li>
            </>
          ) : (
            <li className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>{orgName}</li>
          )}
        </ol>
      </nav>

      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>{projectName || orgName || ''}</h1>
        
        {/* Search for projects when viewing ADO org */}
        {isADOOrg && (
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search projects..."
            value={projectSearchTerm}
            onChange={(e) => setProjectSearchTerm(e.target.value)}
            style={{ width: 300 }}
          />
        )}
        
        {/* Only show search and filters when NOT viewing an ADO org (i.e., viewing repos) */}
        {!isADOOrg && (
          <div className="flex gap-3">
            <TextInput
              leadingVisual={SearchIcon}
              placeholder="Search repositories..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              style={{ width: 300 }}
            />
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="px-3 py-1.5 text-sm rounded-md"
              style={{
                border: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-default)'
              }}
            >
              {statuses.map((status) => {
                let label = '';
                if (status === 'all') {
                  label = 'All Status';
                } else if (status === 'wont_migrate') {
                  label = "Won't Migrate";
                } else if (status === 'remediation_required') {
                  label = "Needs Remediation";
                } else {
                  label = status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ');
                }
                return (
                  <option key={status} value={status}>
                    {label}
                  </option>
                );
              })}
            </select>
            <Button
              onClick={() => setShowFilters(!showFilters)}
              variant={selectedFeatures.size > 0 ? 'primary' : 'invisible'}
              leadingVisual={FilterIcon}
            >
                Features
              {selectedFeatures.size > 0 && ` (${selectedFeatures.size})`}
            </Button>
            {hasActiveFilters && (
              <Button
                onClick={clearAllFilters}
                variant="invisible"
              >
                Clear All
              </Button>
            )}
          </div>
        )}
      </div>

      {/* Feature Filters Panel - only for repository view */}
      {!isADOOrg && showFilters && (() => {
        // Determine the source type from the repositories
        const sourceType = repositories.length > 0 ? repositories[0].source : null;
        const isADOSource = sourceType === 'azuredevops';
        
        // Select the appropriate feature filters based on source
        const featureFilters = isADOSource ? ADO_FEATURE_FILTERS : GITHUB_FEATURE_FILTERS;
        
        return (
          <div 
            className="rounded-lg border p-6 mb-6"
            style={{
              backgroundColor: 'var(--bgColor-default)',
              borderColor: 'var(--borderColor-default)',
              boxShadow: 'var(--shadow-resting-small)'
            }}
          >
            <h3 className="text-base font-semibold mb-4" style={{ color: 'var(--fgColor-default)' }}>
              Filter by Features
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
              {featureFilters.map((featureFilter) => {
                const count = getFeatureCount(featureFilter.key);
                const isSelected = selectedFeatures.has(featureFilter.key);
                const isDisabled = count === 0 && !isSelected;
                
                return (
                  <FormControl key={featureFilter.key} disabled={isDisabled}>
                    <div 
                      className={`flex items-center gap-2 p-2 rounded transition-all ${
                        !isDisabled ? 'cursor-pointer' : ''
                      }`}
                      style={{
                        backgroundColor: isSelected ? 'var(--accent-subtle)' : 'transparent'
                      }}
                      onMouseEnter={(e) => {
                        if (!isDisabled && !isSelected) {
                          e.currentTarget.style.backgroundColor = 'var(--control-bgColor-hover)';
                        }
                      }}
                      onMouseLeave={(e) => {
                        if (!isDisabled && !isSelected) {
                          e.currentTarget.style.backgroundColor = 'transparent';
                        }
                      }}
                      onClick={() => !isDisabled && toggleFeature(featureFilter.key)}
                  >
                      <Checkbox
                        checked={isSelected}
                        onChange={(e) => {
                          e.stopPropagation();
                          toggleFeature(featureFilter.key);
                        }}
                        disabled={isDisabled}
                        value={featureFilter.key}
                      />
                      <FormControl.Label className="flex-1 select-none">
                        <div className="flex items-center justify-between">
                          <span 
                            className="text-sm font-medium"
                            style={{ 
                              color: isDisabled ? 'var(--fgColor-disabled)' : 'var(--fgColor-default)' 
                            }}
                          >
                        {featureFilter.label}
                          </span>
                          <span 
                            className={`ml-2 text-xs ${isSelected ? 'font-semibold' : ''}`}
                            style={{ 
                              color: isSelected 
                                ? 'var(--fgColor-accent)' 
                                : isDisabled 
                                  ? 'var(--fgColor-disabled)' 
                                  : 'var(--fgColor-muted)' 
                            }}
                          >
                            ({count})
                          </span>
                      </div>
                      </FormControl.Label>
                    </div>
                  </FormControl>
                );
              })}
            </div>
          </div>
        );
      })()}

      {/* Repository count - only for repository view */}
      {!isADOOrg && (
        <div className="mb-4 flex items-center justify-between">
          <div className="text-sm text-gh-text-secondary">
            {totalItems > 0 ? (
              <>
                Showing {startIndex + 1}-{Math.min(endIndex, totalItems)} of {repositories.length} repositories
                {hasActiveFilters && ` (${totalItems} match filters)`}
              </>
            ) : (
              'No repositories found'
            )}
          </div>
          {selectedFeatures.size > 0 && (
            <div className="flex gap-2 flex-wrap">
              {Array.from(selectedFeatures).map((feature) => {
                // Look in both filter arrays for the feature config
                const featureConfig = ADO_FEATURE_FILTERS.find(f => f.key === feature) || 
                                     GITHUB_FEATURE_FILTERS.find(f => f.key === feature);
                return (
                  <Token
                    key={feature}
                    text={featureConfig?.label || String(feature)}
                    onRemove={() => toggleFeature(feature)}
                    leadingVisual={FilterIcon}
                  />
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Show projects for ADO organizations, repositories for everything else */}
      {isADOOrg ? (
        // ADO Organization view - show projects
        <>
          {/* Project count */}
          <div className="mb-4 text-sm text-gh-text-secondary">
            {totalProjects > 0 ? (
              <>
                Showing {projectStartIndex + 1}-{Math.min(projectEndIndex, totalProjects)} of {totalProjects} {totalProjects === 1 ? 'project' : 'projects'}
              </>
            ) : (
              'No projects found'
            )}
          </div>

          {projectsLoading ? (
            <LoadingSpinner />
          ) : filteredProjects.length === 0 ? (
            <div className="text-center py-12 text-gh-text-secondary">
              No projects found in this organization
            </div>
          ) : (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
                {paginatedProjects.map((project) => (
                  <ProjectCard key={project.id} project={project} />
                ))}
              </div>
              {totalProjects > projectPageSize && (
                <Pagination
                  currentPage={projectCurrentPage}
                  totalItems={totalProjects}
                  pageSize={projectPageSize}
                  onPageChange={setProjectCurrentPage}
                />
              )}
            </>
          )}
        </>
      ) : (
        // GitHub Organization or ADO Project view - show repositories
        isLoading ? (
          <LoadingSpinner />
        ) : filteredRepos.length === 0 ? (
          <div className="text-center py-12 text-gh-text-secondary">
            No repositories found
          </div>
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
        )
      )}
    </div>
  );
}

function ProjectCard({ project }: { project: ADOProject }) {
  const getStatusColor = (status: string) => {
    // Map all backend statuses to GitHub color scheme
    const colors: Record<string, string> = {
      // Pending
      pending: 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default',
      
      // In Progress (blue)
      dry_run_queued: 'bg-gh-blue text-white',
      dry_run_in_progress: 'bg-gh-blue text-white',
      pre_migration: 'bg-gh-blue text-white',
      archive_generating: 'bg-gh-blue text-white',
      queued_for_migration: 'bg-gh-blue text-white',
      migrating_content: 'bg-gh-blue text-white',
      post_migration: 'bg-gh-blue text-white',
      
      // Complete (green)
      dry_run_complete: 'bg-gh-success text-white',
      migration_complete: 'bg-gh-success text-white',
      complete: 'bg-gh-success text-white',
      
      // Failed (red)
      dry_run_failed: 'bg-gh-danger text-white',
      migration_failed: 'bg-gh-danger text-white',
      
      // Rolled Back (yellow/orange)
      rolled_back: 'bg-gh-warning text-white',
    };
    return colors[status] || 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default';
  };

  return (
    <RouterLink
      to={`/org/${encodeURIComponent(project.organization)}/project/${encodeURIComponent(project.name)}`}
      className="rounded-lg border transition-colors p-6 block"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)'
      }}
    >
      <h3 className="text-base font-semibold mb-3" style={{ color: 'var(--fgColor-default)' }}>
        {project.name}
      </h3>
      
      {project.description && (
        <p className="text-sm text-gh-text-secondary mb-4 line-clamp-2">
          {project.description}
        </p>
      )}
      
      <div className="mb-4">
        <div className="text-2xl font-semibold text-gh-blue mb-1">{project.repository_count}</div>
        <div className="text-sm text-gh-text-secondary">Repositories</div>
      </div>
      
      <div className="flex gap-2 text-xs mb-4">
        {project.visibility && (
          <Badge color={project.visibility === 'public' ? 'green' : 'gray'}>
            {project.visibility}
          </Badge>
        )}
      </div>
      
      {project.status_counts && Object.keys(project.status_counts).length > 0 && (
        <div className="space-y-2">
          <div className="text-xs font-semibold text-gh-text-secondary uppercase tracking-wide">Status Breakdown</div>
          <div className="flex flex-wrap gap-2">
            {Object.entries(project.status_counts).map(([status, count]) => (
              <span
                key={status}
                className={`px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(status)}`}
              >
                {status.replace(/_/g, ' ')}: {count}
              </span>
            ))}
          </div>
        </div>
      )}
    </RouterLink>
  );
}

function RepositoryCard({ repository }: { repository: Repository }) {
  // Format display name: for ADO repos, show "project/repo" instead of "org/project/repo"
  const displayName = repository.ado_project 
    ? repository.full_name.split('/').slice(1).join('/') // Remove org prefix for ADO repos
    : repository.full_name; // Keep full name for GitHub repos

  return (
    <RouterLink
      to={`/repository/${encodeURIComponent(repository.full_name)}`}
      className="rounded-lg border transition-colors p-6 block"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)'
      }}
    >
      <h3 className="text-base font-semibold mb-3 truncate" style={{ color: 'var(--fgColor-default)' }}>
        {displayName}
      </h3>
      <div className="mb-3 flex items-center justify-between">
        <StatusBadge status={repository.status} size="small" />
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
        {/* Azure DevOps specific badges - only show for ADO sources */}
        {repository.source === 'azuredevops' && repository.ado_project && <Badge color="blue">ADO: {repository.ado_project}</Badge>}
        {repository.source === 'azuredevops' && repository.ado_is_git === false && <Badge color="red">TFVC</Badge>}
        {repository.source === 'azuredevops' && repository.ado_has_boards && <Badge color="purple">Azure Boards</Badge>}
        {repository.source === 'azuredevops' && repository.ado_has_pipelines && <Badge color="green">Pipelines</Badge>}
        {repository.source === 'azuredevops' && repository.ado_has_ghas && <Badge color="green">GHAS</Badge>}
        {repository.source === 'azuredevops' && repository.ado_pull_request_count > 0 && <Badge color="indigo">PRs: {repository.ado_pull_request_count}</Badge>}
        {repository.source === 'azuredevops' && repository.ado_work_item_count > 0 && <Badge color="purple">Work Items: {repository.ado_work_item_count}</Badge>}
        {repository.source === 'azuredevops' && repository.ado_branch_policy_count > 0 && <Badge color="red">Policies: {repository.ado_branch_policy_count}</Badge>}
        
        {/* GitHub specific badges */}
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
    </RouterLink>
  );
}

