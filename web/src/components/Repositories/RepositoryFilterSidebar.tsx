import { useEffect, useState } from 'react';
import type { RepositoryFilters } from '../../types';
import { api } from '../../services/api';
import { FilterSection } from '../BatchManagement/FilterSection';
import { OrganizationSelector } from '../BatchManagement/OrganizationSelector';

interface RepositoryFilterSidebarProps {
  filters: RepositoryFilters;
  onChange: (filters: RepositoryFilters) => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

// Categorized status groups matching organization detail view
const STATUS_CATEGORIES = [
  {
    group: 'Pending',
    statuses: [
      { value: 'pending', label: 'Pending' },
      { value: 'remediation_required', label: 'Remediation Required' },
    ]
  },
  {
    group: 'In Progress',
    statuses: [
      { value: 'dry_run_queued', label: 'Dry Run Queued' },
      { value: 'dry_run_in_progress', label: 'Dry Run In Progress' },
      { value: 'pre_migration', label: 'Pre Migration' },
      { value: 'archive_generating', label: 'Archive Generating' },
      { value: 'queued_for_migration', label: 'Queued for Migration' },
      { value: 'migrating_content', label: 'Migrating Content' },
      { value: 'post_migration', label: 'Post Migration' },
    ]
  },
  {
    group: 'Complete',
    statuses: [
      { value: 'dry_run_complete', label: 'Dry Run Complete' },
      { value: 'migration_complete', label: 'Migration Complete' },
      { value: 'complete', label: 'Complete' },
    ]
  },
  {
    group: 'Failed',
    statuses: [
      { value: 'dry_run_failed', label: 'Dry Run Failed' },
      { value: 'migration_failed', label: 'Migration Failed' },
    ]
  },
  {
    group: 'Other',
    statuses: [
      { value: 'rolled_back', label: 'Rolled Back' },
      { value: 'wont_migrate', label: "Won't Migrate" },
    ]
  },
];

export function RepositoryFilterSidebar({ filters, onChange, isCollapsed, onToggleCollapse }: RepositoryFilterSidebarProps) {
  const [organizations, setOrganizations] = useState<string[]>([]);
  const [projects, setProjects] = useState<string[]>([]);
  const [loadingOrgs, setLoadingOrgs] = useState(false);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [sourceType, setSourceType] = useState<'github' | 'azuredevops'>('github');

  useEffect(() => {
    loadConfig();
    loadOrganizations();
  }, []);

  const loadConfig = async () => {
    try {
      const config = await api.getConfig();
      setSourceType(config.source_type);
      if (config.source_type === 'azuredevops') {
        loadProjects();
      }
    } catch (error) {
      console.error('Failed to load config:', error);
    }
  };

  const loadOrganizations = async () => {
    setLoadingOrgs(true);
    try {
      const orgs = await api.getOrganizationList();
      setOrganizations(orgs || []);
    } catch (error) {
      console.error('Failed to load organizations:', error);
      setOrganizations([]);
    } finally {
      setLoadingOrgs(false);
    }
  };

  const loadProjects = async () => {
    setLoadingProjects(true);
    try {
      const projectList = await api.listADOProjects();
      const projectNames = projectList.map((p: any) => p.name || p.project_name);
      setProjects(projectNames || []);
    } catch (error) {
      console.error('Failed to load projects:', error);
      setProjects([]);
    } finally {
      setLoadingProjects(false);
    }
  };

  const getSelectedOrganizations = (): string[] => {
    if (!filters.organization) return [];
    return Array.isArray(filters.organization) ? filters.organization : [filters.organization];
  };

  const handleOrganizationChange = (selected: string[]) => {
    onChange({
      ...filters,
      organization: selected.length > 0 ? selected : undefined,
    });
  };

  const getSelectedProjects = (): string[] => {
    if (!filters.project) return [];
    return Array.isArray(filters.project) ? filters.project : [filters.project];
  };

  const handleProjectChange = (selected: string[]) => {
    onChange({
      ...filters,
      project: selected.length > 0 ? selected : undefined,
    });
  };

  const getSelectedStatuses = (): string[] => {
    if (!filters.status) return [];
    return Array.isArray(filters.status) ? filters.status : [filters.status];
  };

  const handleStatusChange = (status: string, checked: boolean) => {
    const current = getSelectedStatuses();
    const updated = checked
      ? [...current, status]
      : current.filter((s) => s !== status);
    onChange({
      ...filters,
      status: updated.length > 0 ? updated : undefined,
    });
  };

  const activeFilterCount = () => {
    let count = 0;
    if (filters.organization) count++;
    if (filters.project) count++;
    if (filters.status) count++;
    // Note: search is now in the page header, not counted here
    if (filters.min_size || filters.max_size) count++;
    if (filters.size_category) count++;
    if (filters.complexity) count++;
    // Common features
    if (filters.has_lfs) count++;
    if (filters.has_submodules) count++;
    if (filters.has_large_files) count++;
    // GitHub features
    if (filters.has_actions) count++;
    if (filters.has_wiki) count++;
    if (filters.has_pages) count++;
    if (filters.has_discussions) count++;
    if (filters.has_projects) count++;
    if (filters.has_packages) count++;
    if (filters.has_branch_protections) count++;
    if (filters.has_rulesets) count++;
    if (filters.is_archived !== undefined) count++;
    if (filters.is_fork !== undefined) count++;
    if (filters.has_code_scanning) count++;
    if (filters.has_dependabot) count++;
    if (filters.has_secret_scanning) count++;
    if (filters.has_codeowners) count++;
    if (filters.has_self_hosted_runners) count++;
    if (filters.has_release_assets) count++;
    if (filters.has_webhooks) count++;
    // ADO features
    if (filters.ado_is_git !== undefined) count++;
    if (filters.ado_has_boards) count++;
    if (filters.ado_has_pipelines) count++;
    if (filters.ado_has_ghas) count++;
    if (filters.ado_has_wiki) count++;
    // Other
    if (filters.visibility) count++;
    if (filters.sort_by && filters.sort_by !== 'name') count++;
    return count;
  };

  const filterCount = activeFilterCount();

  if (isCollapsed) {
    return (
      <div 
        className="w-12 flex flex-col items-center py-4 flex-shrink-0"
        style={{ 
          borderRight: '1px solid var(--borderColor-default)',
          backgroundColor: 'var(--bgColor-default)' 
        }}
      >
        <button
          onClick={onToggleCollapse}
          className="relative p-2 rounded-lg transition-opacity hover:opacity-80 group"
          title="Expand filters"
        >
          <svg className="w-6 h-6" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
            />
          </svg>
          {filterCount > 0 && (
            <span 
              className="absolute -top-1 -right-1 flex items-center justify-center w-5 h-5 text-xs font-bold rounded-full"
              style={{ 
                color: 'var(--fgColor-onEmphasis)',
                backgroundColor: 'var(--accent-emphasis)' 
              }}
            >
              {filterCount}
            </span>
          )}
        </button>
      </div>
    );
  }

  return (
    <div 
      className="w-[280px] flex flex-col transition-all duration-300 flex-shrink-0"
      style={{ 
        borderRight: '1px solid var(--borderColor-default)',
        backgroundColor: 'var(--bgColor-default)' 
      }}
    >
      {/* Header */}
      <div 
        className="flex items-center justify-between p-4"
        style={{ borderBottom: '1px solid var(--borderColor-default)' }}
      >
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>Filters</h3>
          {filterCount > 0 && (
            <span 
              className="flex items-center justify-center min-w-[20px] h-5 px-1.5 text-xs font-bold rounded-full"
              style={{ 
                color: 'var(--fgColor-onEmphasis)',
                backgroundColor: 'var(--accent-emphasis)' 
              }}
            >
              {filterCount}
            </span>
          )}
        </div>
        <button
          onClick={onToggleCollapse}
          className="p-1 rounded transition-opacity hover:opacity-80"
          title="Collapse filters"
        >
          <svg className="w-5 h-5" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
      </div>

      {/* Scrollable Filter Content */}
      <div className="flex-1 overflow-y-auto">
        {/* Status */}
        <FilterSection title="Status" defaultExpanded={true}>
          <div className="space-y-3 max-h-80 overflow-y-auto">
            {STATUS_CATEGORIES.map((category) => (
              <div key={category.group} className="space-y-2">
                <div className="text-xs font-semibold uppercase" style={{ color: 'var(--fgColor-muted)' }}>
                  {category.group}
                </div>
                <div className="space-y-2 pl-2">
                  {category.statuses.map((status) => (
                    <label key={status.value} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={getSelectedStatuses().includes(status.value)}
                        onChange={(e) => handleStatusChange(status.value, e.target.checked)}
                        className="rounded text-blue-600 focus:ring-blue-500"
                        style={{ borderColor: 'var(--borderColor-default)' }}
                      />
                      <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>
                        {status.label}
                      </span>
                    </label>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </FilterSection>

        {/* Organization */}
        <FilterSection title="Organization" defaultExpanded={true}>
          <OrganizationSelector
            organizations={organizations}
            selectedOrganizations={getSelectedOrganizations()}
            onChange={handleOrganizationChange}
            loading={loadingOrgs}
          />
        </FilterSection>

        {/* Project (for Azure DevOps only) */}
        {sourceType === 'azuredevops' && (
          <FilterSection title="Project" defaultExpanded={true}>
            <OrganizationSelector
              organizations={projects}
              selectedOrganizations={getSelectedProjects()}
              onChange={handleProjectChange}
              loading={loadingProjects}
              placeholder="All Projects"
              searchPlaceholder="Search projects..."
              emptyMessage="No projects found"
            />
          </FilterSection>
        )}

        {/* Complexity */}
        <FilterSection title="Complexity" defaultExpanded={true}>
          <div className="space-y-2">
            {['simple', 'medium', 'complex', 'very_complex'].map((complexity) => (
              <label key={complexity} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={
                    Array.isArray(filters.complexity)
                      ? filters.complexity.includes(complexity)
                      : filters.complexity === complexity
                  }
                  onChange={(e) => {
                    const current = Array.isArray(filters.complexity)
                      ? filters.complexity
                      : filters.complexity
                      ? [filters.complexity]
                      : [];
                    const updated = e.target.checked
                      ? [...current, complexity]
                      : current.filter((c) => c !== complexity);
                    onChange({
                      ...filters,
                      complexity: updated.length > 0 ? updated : undefined,
                    });
                  }}
                  className="rounded text-blue-600 focus:ring-blue-500"
                  style={{ borderColor: 'var(--borderColor-default)' }}
                />
                <span className="text-sm capitalize" style={{ color: 'var(--fgColor-default)' }}>
                  {complexity.replace('_', ' ')}
                </span>
              </label>
            ))}
          </div>
        </FilterSection>

        {/* Size */}
        <FilterSection title="Size" defaultExpanded={false}>
          <div className="space-y-3">
            {/* Size Category */}
            <div>
              <label className="block text-xs font-medium mb-2" style={{ color: 'var(--fgColor-default)' }}>Category</label>
              <div className="space-y-2">
                {[
                  { value: 'small', label: 'Small (<100MB)' },
                  { value: 'medium', label: 'Medium (100MB-1GB)' },
                  { value: 'large', label: 'Large (1GB-5GB)' },
                  { value: 'very_large', label: 'Very Large (>5GB)' },
                ].map((category) => (
                  <label key={category.value} className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={
                        Array.isArray(filters.size_category)
                          ? filters.size_category.includes(category.value)
                          : filters.size_category === category.value
                      }
                      onChange={(e) => {
                        const current = Array.isArray(filters.size_category)
                          ? filters.size_category
                          : filters.size_category
                          ? [filters.size_category]
                          : [];
                        const updated = e.target.checked
                          ? [...current, category.value]
                          : current.filter((c) => c !== category.value);
                        onChange({
                          ...filters,
                          size_category: updated.length > 0 ? updated : undefined,
                        });
                      }}
                      className="rounded text-blue-600 focus:ring-blue-500"
                      style={{ borderColor: 'var(--borderColor-default)' }}
                    />
                    <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>{category.label}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Size Range */}
            <div>
              <label className="block text-xs font-medium mb-2" style={{ color: 'var(--fgColor-default)' }}>Range (MB)</label>
              <div className="grid grid-cols-2 gap-2">
                <input
                  type="number"
                  placeholder="Min"
                  value={filters.min_size ? Math.round(filters.min_size / 1024 / 1024).toString() : ''}
                  onChange={(e) =>
                    onChange({
                      ...filters,
                      min_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 : undefined,
                    })
                  }
                  className="px-2 py-1.5 text-sm rounded focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  style={{
                    border: '1px solid var(--borderColor-default)',
                    backgroundColor: 'var(--control-bgColor-rest)',
                    color: 'var(--fgColor-default)'
                  }}
                />
                <input
                  type="number"
                  placeholder="Max"
                  value={filters.max_size ? Math.round(filters.max_size / 1024 / 1024).toString() : ''}
                  onChange={(e) =>
                    onChange({
                      ...filters,
                      max_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 : undefined,
                    })
                  }
                  className="px-2 py-1.5 text-sm rounded focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  style={{
                    border: '1px solid var(--borderColor-default)',
                    backgroundColor: 'var(--control-bgColor-rest)',
                    color: 'var(--fgColor-default)'
                  }}
                />
              </div>
            </div>
          </div>
        </FilterSection>

        {/* Features */}
        <FilterSection title="Features" defaultExpanded={false}>
          <div className="space-y-2">
            {sourceType === 'github' ? (
              // GitHub features
              [
              { key: 'has_lfs' as const, label: 'LFS' },
              { key: 'has_submodules' as const, label: 'Submodules' },
              { key: 'has_large_files' as const, label: 'Large Files (>100MB)' },
              { key: 'has_actions' as const, label: 'Actions' },
              { key: 'has_wiki' as const, label: 'Wiki' },
              { key: 'has_pages' as const, label: 'Pages' },
              { key: 'has_discussions' as const, label: 'Discussions' },
              { key: 'has_projects' as const, label: 'Projects' },
              { key: 'has_packages' as const, label: 'Packages' },
              { key: 'has_branch_protections' as const, label: 'Branch Protections' },
              { key: 'has_rulesets' as const, label: 'Rulesets' },
              { key: 'is_archived' as const, label: 'Archived' },
              { key: 'is_fork' as const, label: 'Fork' },
              { key: 'has_code_scanning' as const, label: 'Code Scanning' },
              { key: 'has_dependabot' as const, label: 'Dependabot' },
              { key: 'has_secret_scanning' as const, label: 'Secret Scanning' },
              { key: 'has_codeowners' as const, label: 'CODEOWNERS' },
              { key: 'has_self_hosted_runners' as const, label: 'Self-Hosted Runners' },
              { key: 'has_release_assets' as const, label: 'Release Assets' },
              { key: 'has_webhooks' as const, label: 'Webhooks' },
            ].map((feature) => (
              <label key={feature.key} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={filters[feature.key] || false}
                  onChange={(e) =>
                    onChange({ ...filters, [feature.key]: e.target.checked ? true : undefined })
                  }
                  className="rounded text-blue-600 focus:ring-blue-500"
                  style={{ borderColor: 'var(--borderColor-default)' }}
                />
                <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>{feature.label}</span>
              </label>
              ))
            ) : (
              // Azure DevOps features
              [
                { key: 'ado_is_git' as const, label: 'Git (vs TFVC)' },
                { key: 'ado_has_boards' as const, label: 'Azure Boards' },
                { key: 'ado_has_pipelines' as const, label: 'Azure Pipelines' },
                { key: 'ado_has_wiki' as const, label: 'Wiki' },
                { key: 'ado_has_ghas' as const, label: 'GHAS (Advanced Security)' },
                { key: 'has_lfs' as const, label: 'LFS' },
                { key: 'has_submodules' as const, label: 'Submodules' },
                { key: 'has_large_files' as const, label: 'Large Files (>100MB)' },
              ].map((feature) => (
                <label key={feature.key} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={filters[feature.key] || false}
                    onChange={(e) =>
                      onChange({ ...filters, [feature.key]: e.target.checked ? true : undefined })
                    }
                    className="rounded text-blue-600 focus:ring-blue-500"
                    style={{ borderColor: 'var(--borderColor-default)' }}
                  />
                  <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>{feature.label}</span>
                </label>
              ))
            )}
          </div>
        </FilterSection>

        {/* Visibility */}
        <FilterSection title="Visibility" defaultExpanded={false}>
          <select
            value={filters.visibility || ''}
            onChange={(e) =>
              onChange({
                ...filters,
                visibility: e.target.value || undefined,
              })
            }
            className="w-full px-3 py-2 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            style={{
              border: '1px solid var(--borderColor-default)',
              backgroundColor: 'var(--control-bgColor-rest)',
              color: 'var(--fgColor-default)'
            }}
          >
            <option value="">All</option>
            <option value="public">Public</option>
            <option value="private">Private</option>
            {sourceType === 'github' && <option value="internal">Internal</option>}
          </select>
        </FilterSection>

        {/* Sort */}
        <FilterSection title="Sort By" defaultExpanded={false}>
          <select
            value={filters.sort_by || 'name'}
            onChange={(e) =>
              onChange({
                ...filters,
                sort_by: (e.target.value as any) || undefined,
              })
            }
            className="w-full px-3 py-2 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            style={{
              border: '1px solid var(--borderColor-default)',
              backgroundColor: 'var(--control-bgColor-rest)',
              color: 'var(--fgColor-default)'
            }}
          >
            <option value="name">Name</option>
            <option value="size">Size</option>
            <option value="org">Organization</option>
            <option value="updated">Last Updated</option>
          </select>
        </FilterSection>
      </div>
    </div>
  );
}

