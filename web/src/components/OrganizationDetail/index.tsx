import { useState, useEffect } from 'react';
import { useParams, Link as RouterLink, useSearchParams } from 'react-router-dom';
import { Link } from '@primer/react';
import type { Repository, ADOProject, Organization, RepositoryFilters } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { Pagination } from '../common/Pagination';
import { formatBytes } from '../../utils/format';
import { useRepositories, useOrganizations } from '../../hooks/useQueries';
import { api } from '../../services/api';
import { RepositoryFilterSidebar } from '../Repositories/RepositoryFilterSidebar';


export function OrganizationDetail() {
  const { orgName, projectName } = useParams<{ orgName: string; projectName?: string }>();
  const [searchParams] = useSearchParams();
  const searchTerm = searchParams.get('search') || '';
  const [filters, setFilters] = useState<RepositoryFilters>({});
  const [isFiltersCollapsed, setIsFiltersCollapsed] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 12;
  const [isADOOrg, setIsADOOrg] = useState(false);
  const [adoProjects, setAdoProjects] = useState<ADOProject[]>([]);
  const [projectsLoading, setProjectsLoading] = useState(false);
  const projectSearchTerm = searchParams.get('search') || ''; // Projects also use the same search param
  const [projectCurrentPage, setProjectCurrentPage] = useState(1);
  const projectPageSize = 12;

  const { data, isLoading, isFetching } = useRepositories(filters);
  const { data: orgsData } = useOrganizations();

  // Set filters to lock organization/project when component loads
  useEffect(() => {
    if (orgName) {
      setFilters(prev => ({
        ...prev,
        organization: [orgName],
        project: projectName ? [projectName] : undefined,
      }));
    }
  }, [orgName, projectName]);

  // Reset pagination when organization or project changes
  useEffect(() => {
    setCurrentPage(1);
    setProjectCurrentPage(1);
  }, [orgName, projectName]);

  // Reset repository pagination when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [filters, searchTerm]);

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

  // Filter repositories - API already handles filtering based on the filters object
  const repositories = data?.repositories || [];

  // Apply search term from header (client-side since it's in URL params)
  const filteredRepos = repositories.filter((repo: Repository) => {
    if (searchTerm && !repo.full_name.toLowerCase().includes(searchTerm.toLowerCase())) {
      return false;
    }
    return true;
  });

  // Paginate
  const totalItems = filteredRepos.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedRepos = filteredRepos.slice(startIndex, endIndex);

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

  const handleFilterChange = (newFilters: RepositoryFilters) => {
    // Always lock the organization/project to the current view
    setFilters({
      ...newFilters,
      organization: [orgName!],
      project: projectName ? [projectName] : undefined,
    });
  };

  return (
    <div className="flex h-screen" style={{ backgroundColor: 'var(--bgColor-default)' }}>
      {/* Filter Sidebar - only for repository view */}
      {!isADOOrg && (
        <RepositoryFilterSidebar
          filters={filters}
          onChange={handleFilterChange}
          isCollapsed={isFiltersCollapsed}
          onToggleCollapse={() => setIsFiltersCollapsed(!isFiltersCollapsed)}
          hideOrganization={true}
          hideProject={true}
        />
      )}
      
      {/* Main Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-8 relative">
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
          </div>

          {/* Repository count - only for repository view */}
          {!isADOOrg && (
            <div className="mb-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              {totalItems > 0 ? (
                <>
                  Showing {startIndex + 1}-{Math.min(endIndex, totalItems)} of {filteredRepos.length} repositories
                </>
              ) : (
                'No repositories found'
              )}
            </div>
          )}

          {/* Show projects for ADO organizations, repositories for everything else */}
          {isADOOrg ? (
            // ADO Organization view - show projects
            <>
              {/* Project count */}
              <div className="mb-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
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
                <div className="text-center py-12" style={{ color: 'var(--fgColor-muted)' }}>
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
              <div className="text-center py-12" style={{ color: 'var(--fgColor-muted)' }}>
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
      </div>
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

