import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
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

const FEATURE_FILTERS: FeatureFilter[] = [
  // Azure DevOps specific
  { key: 'ado_has_boards', label: 'Azure Boards', color: 'purple' },
  { key: 'ado_has_pipelines', label: 'Azure Pipelines', color: 'green' },
  { key: 'ado_has_ghas', label: 'GHAS (ADO)', color: 'green' },
  { key: 'ado_pull_request_count', label: 'Has Pull Requests', color: 'indigo' },
  { key: 'ado_work_item_count', label: 'Has Work Items', color: 'purple' },
  { key: 'ado_branch_policy_count', label: 'Has Branch Policies', color: 'red' },
  
  // GitHub specific
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

  const statuses = ['all', 'pending', 'in_progress', 'complete', 'failed', 'rolled_back'];
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
    <div className="max-w-7xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <div className="mb-6">
        {projectName ? (
          <Link to={`/org/${encodeURIComponent(orgName || '')}`} className="text-gh-blue hover:underline text-sm font-medium">
            ← Back to {orgName}
          </Link>
        ) : (
          <Link to="/" className="text-gh-blue hover:underline text-sm font-medium">
            ← Back to Organizations
          </Link>
        )}
      </div>

      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold text-gh-text-primary">{projectName || orgName || ''}</h1>
        
        {/* Search for projects when viewing ADO org */}
        {isADOOrg && (
          <input
            type="text"
            placeholder="Search projects..."
            value={projectSearchTerm}
            onChange={(e) => setProjectSearchTerm(e.target.value)}
            className="px-3 py-1.5 text-sm border border-gh-border-default rounded-md"
          />
        )}
        
        {/* Only show search and filters when NOT viewing an ADO org (i.e., viewing repos) */}
        {!isADOOrg && (
          <div className="flex gap-3">
            <input
              type="text"
              placeholder="Search repositories..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="px-3 py-1.5 text-sm border border-gh-border-default rounded-md"
            />
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="px-3 py-1.5 text-sm border border-gh-border-default rounded-md"
            >
              {statuses.map((status) => (
                <option key={status} value={status}>
                  {status === 'all' ? 'All Status' : status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ')}
                </option>
              ))}
            </select>
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors font-medium ${
                selectedFeatures.size > 0
                  ? 'bg-gh-blue text-white hover:bg-gh-blue-hover'
                  : 'bg-gh-neutral-bg text-gh-text-primary hover:bg-gh-canvas-inset border border-gh-border-default'
              }`}
            >
              <span className="flex items-center gap-2">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
                </svg>
                Features
                {selectedFeatures.size > 0 && (
                  <span className="bg-white text-gh-blue rounded-full px-2 py-0.5 text-xs font-medium">
                    {selectedFeatures.size}
                  </span>
                )}
              </span>
            </button>
            {hasActiveFilters && (
              <button
                onClick={clearAllFilters}
                className="px-3 py-1.5 text-sm text-gh-text-secondary hover:text-gh-text-primary transition-colors font-medium"
              >
                Clear All
              </button>
            )}
          </div>
        )}
      </div>

      {/* Feature Filters Panel - only for repository view */}
      {!isADOOrg && showFilters && (
        <div className="bg-white rounded-lg border border-gh-border-default shadow-gh-card p-6 mb-6">
          <h3 className="text-base font-semibold text-gh-text-primary mb-4">Filter by Features</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
            {FEATURE_FILTERS.map((featureFilter) => {
              const count = repositories.filter(r => {
                const value = r[featureFilter.key];
                return typeof value === 'boolean' ? value : (typeof value === 'number' && value > 0);
              }).length;
              return (
                <label
                  key={featureFilter.key}
                  className={`flex items-center gap-2 p-3 rounded-md border cursor-pointer transition-all ${
                    selectedFeatures.has(featureFilter.key)
                      ? 'border-gh-blue bg-gh-info-bg'
                      : 'border-gh-border-default hover:border-gh-border-hover'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selectedFeatures.has(featureFilter.key)}
                    onChange={() => toggleFeature(featureFilter.key)}
                    className="w-4 h-4 text-gh-blue rounded border-gh-border-default focus:ring-gh-blue"
                  />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-gh-text-primary truncate">{featureFilter.label}</div>
                    <div className="text-xs text-gh-text-secondary">{count} repos</div>
                  </div>
                </label>
              );
            })}
          </div>
        </div>
      )}

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
                const featureConfig = FEATURE_FILTERS.find(f => f.key === feature);
                return (
                  <span
                    key={feature}
                    className="inline-flex items-center gap-1 px-2 py-1 bg-gh-info-bg text-gh-blue rounded-full text-xs font-medium border border-gh-blue/20"
                  >
                    {featureConfig?.label}
                    <button
                      onClick={() => toggleFeature(feature)}
                      className="hover:bg-gh-blue/10 rounded-full p-0.5"
                    >
                      <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                      </svg>
                    </button>
                  </span>
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
  return (
    <Link
      to={`/org/${encodeURIComponent(project.organization)}/project/${encodeURIComponent(project.name)}`}
      className="bg-white rounded-lg border border-gh-border-default hover:border-gh-border-hover transition-colors p-6 block shadow-gh-card"
    >
      <h3 className="text-base font-semibold text-gh-text-primary mb-3">
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
      
      <div className="flex gap-2 text-xs text-gh-text-secondary">
        {project.visibility && (
          <Badge color={project.visibility === 'public' ? 'green' : 'gray'}>
            {project.visibility}
          </Badge>
        )}
      </div>
    </Link>
  );
}

function RepositoryCard({ repository }: { repository: Repository }) {
  // Format display name: for ADO repos, show "project/repo" instead of "org/project/repo"
  const displayName = repository.ado_project 
    ? repository.full_name.split('/').slice(1).join('/') // Remove org prefix for ADO repos
    : repository.full_name; // Keep full name for GitHub repos

  return (
    <Link
      to={`/repository/${encodeURIComponent(repository.full_name)}`}
      className="bg-white rounded-lg border border-gh-border-default hover:border-gh-border-hover transition-colors p-6 block shadow-gh-card"
    >
      <h3 className="text-base font-semibold text-gh-text-primary mb-3 truncate">
        {displayName}
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
        {/* Azure DevOps specific badges */}
        {repository.ado_project && <Badge color="blue">ADO: {repository.ado_project}</Badge>}
        {repository.ado_is_git === false && <Badge color="red">TFVC</Badge>}
        {repository.ado_has_boards && <Badge color="purple">Azure Boards</Badge>}
        {repository.ado_has_pipelines && <Badge color="green">Pipelines</Badge>}
        {repository.ado_has_ghas && <Badge color="green">GHAS</Badge>}
        {repository.ado_pull_request_count > 0 && <Badge color="indigo">PRs: {repository.ado_pull_request_count}</Badge>}
        {repository.ado_work_item_count > 0 && <Badge color="purple">Work Items: {repository.ado_work_item_count}</Badge>}
        {repository.ado_branch_policy_count > 0 && <Badge color="red">Policies: {repository.ado_branch_policy_count}</Badge>}
        
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
    </Link>
  );
}

