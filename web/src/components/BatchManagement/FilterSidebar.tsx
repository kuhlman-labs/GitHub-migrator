import { useEffect, useState } from 'react';
import type { RepositoryFilters } from '../../types';
import { api } from '../../services/api';
import { FilterSection } from './FilterSection';
import { OrganizationSelector } from './OrganizationSelector';

interface FilterSidebarProps {
  filters: RepositoryFilters;
  onChange: (filters: RepositoryFilters) => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

export function FilterSidebar({ filters, onChange, isCollapsed, onToggleCollapse }: FilterSidebarProps) {
  const [organizations, setOrganizations] = useState<string[]>([]);
  const [loadingOrgs, setLoadingOrgs] = useState(false);

  useEffect(() => {
    loadOrganizations();
  }, []);

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

  const activeFilterCount = () => {
    let count = 0;
    if (filters.organization) count++;
    if (filters.search) count++;
    if (filters.min_size || filters.max_size) count++;
    if (filters.size_category) count++;
    if (filters.complexity) count++;
    if (filters.has_lfs) count++;
    if (filters.has_submodules) count++;
    if (filters.has_large_files) count++;
    if (filters.has_actions) count++;
    if (filters.has_wiki) count++;
    if (filters.has_pages) count++;
    if (filters.has_discussions) count++;
    if (filters.has_projects) count++;
    if (filters.has_packages) count++;
    if (filters.has_branch_protections) count++;
    if (filters.is_archived !== undefined) count++;
    if (filters.is_fork !== undefined) count++;
    if (filters.sort_by && filters.sort_by !== 'name') count++;
    return count;
  };

  const filterCount = activeFilterCount();

  if (isCollapsed) {
    return (
      <div className="w-12 border-r border-gray-200 bg-white flex flex-col items-center py-4 flex-shrink-0">
        <button
          onClick={onToggleCollapse}
          className="relative p-2 hover:bg-gray-100 rounded-lg transition-colors group"
          title="Expand filters"
        >
          <svg className="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
            />
          </svg>
          {filterCount > 0 && (
            <span className="absolute -top-1 -right-1 flex items-center justify-center w-5 h-5 text-xs font-bold text-white bg-blue-600 rounded-full">
              {filterCount}
            </span>
          )}
        </button>
      </div>
    );
  }

  return (
    <div className="w-[280px] border-r border-gray-200 bg-white flex flex-col transition-all duration-300 flex-shrink-0">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-gray-900">Filters</h3>
          {filterCount > 0 && (
            <span className="flex items-center justify-center min-w-[20px] h-5 px-1.5 text-xs font-bold text-white bg-blue-600 rounded-full">
              {filterCount}
            </span>
          )}
        </div>
        <button
          onClick={onToggleCollapse}
          className="p-1 hover:bg-gray-100 rounded transition-colors"
          title="Collapse filters"
        >
          <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
      </div>

      {/* Scrollable Filter Content */}
      <div className="flex-1 overflow-y-auto">
        {/* Search */}
        <div className="p-4 border-b border-gray-200">
          <label className="block text-xs font-medium text-gray-700 mb-2">Search</label>
          <input
            type="text"
            value={filters.search || ''}
            onChange={(e) => onChange({ ...filters, search: e.target.value || undefined })}
            placeholder="Repository name..."
            className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Organization */}
        <FilterSection title="Organization" defaultExpanded={true}>
          <OrganizationSelector
            organizations={organizations}
            selectedOrganizations={getSelectedOrganizations()}
            onChange={handleOrganizationChange}
            loading={loadingOrgs}
          />
        </FilterSection>

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
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 capitalize">
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
              <label className="block text-xs font-medium text-gray-700 mb-2">Category</label>
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
                      className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">{category.label}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Size Range */}
            <div>
              <label className="block text-xs font-medium text-gray-700 mb-2">Range (MB)</label>
              <div className="grid grid-cols-2 gap-2">
                <input
                  type="number"
                  placeholder="Min"
                  value={filters.min_size ? Math.round(filters.min_size / 1024 / 1024) : ''}
                  onChange={(e) =>
                    onChange({
                      ...filters,
                      min_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 : undefined,
                    })
                  }
                  className="px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <input
                  type="number"
                  placeholder="Max"
                  value={filters.max_size ? Math.round(filters.max_size / 1024 / 1024) : ''}
                  onChange={(e) =>
                    onChange({
                      ...filters,
                      max_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 : undefined,
                    })
                  }
                  className="px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>
        </FilterSection>

        {/* Features */}
        <FilterSection title="Features" defaultExpanded={false}>
          <div className="space-y-2">
            {[
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
              { key: 'is_archived' as const, label: 'Archived' },
              { key: 'is_fork' as const, label: 'Fork' },
            ].map((feature) => (
              <label key={feature.key} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={filters[feature.key] || false}
                  onChange={(e) =>
                    onChange({ ...filters, [feature.key]: e.target.checked ? true : undefined })
                  }
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">{feature.label}</span>
              </label>
            ))}
          </div>
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
            className="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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

