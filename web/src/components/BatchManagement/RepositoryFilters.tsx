import { useEffect, useState } from 'react';
import type { RepositoryFilters as Filters } from '../../types';
import { api } from '../../services/api';

interface RepositoryFiltersProps {
  filters: Filters;
  onChange: (filters: Filters) => void;
  onClear: () => void;
}

export function RepositoryFilters({ filters, onChange, onClear }: RepositoryFiltersProps) {
  const [organizations, setOrganizations] = useState<string[]>([]);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [loadingOrgs, setLoadingOrgs] = useState(false);

  useEffect(() => {
    loadOrganizations();
  }, []);

  const loadOrganizations = async () => {
    if (loadingOrgs) return; // Prevent duplicate loads
    
    setLoadingOrgs(true);
    try {
      const orgs = await api.getOrganizationList();
      setOrganizations(orgs || []); // Ensure it's always an array
    } catch (error) {
      console.error('Failed to load organizations:', error);
      setOrganizations([]); // Set to empty array on error
    } finally {
      setLoadingOrgs(false);
    }
  };

  const handleOrganizationChange = (org: string, checked: boolean) => {
    const currentOrgs = Array.isArray(filters.organization)
      ? filters.organization
      : filters.organization
      ? [filters.organization]
      : [];

    const newOrgs = checked
      ? [...currentOrgs, org]
      : currentOrgs.filter((o) => o !== org);

    onChange({
      ...filters,
      organization: newOrgs.length > 0 ? newOrgs : undefined,
    });
  };

  const isOrgSelected = (org: string) => {
    if (!filters.organization) return false;
    if (Array.isArray(filters.organization)) {
      return filters.organization.includes(org);
    }
    return filters.organization === org;
  };

  const activeFilterCount = () => {
    let count = 0;
    if (filters.organization) count++;
    if (filters.min_size || filters.max_size) count++;
    if (filters.has_lfs) count++;
    if (filters.has_submodules) count++;
    if (filters.has_actions) count++;
    if (filters.has_wiki) count++;
    if (filters.has_pages) count++;
    if (filters.is_archived !== undefined) count++;
    if (filters.sort_by && filters.sort_by !== 'name') count++;
    return count;
  };

  const handleQuickFilter = (type: 'all' | 'simple' | 'complex') => {
    switch (type) {
      case 'all':
        onClear();
        break;
      case 'simple':
        onChange({
          has_lfs: false,
          has_submodules: false,
          has_actions: false,
          is_archived: false,
        });
        break;
      case 'complex':
        onChange({
          ...filters,
          has_lfs: true,
        });
        break;
    }
  };

  const filterCount = activeFilterCount();

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-900">
          Filters {filterCount > 0 && <span className="ml-2 text-blue-600">({filterCount})</span>}
        </h3>
        <div className="flex gap-2">
          <button
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="text-sm text-blue-600 hover:text-blue-700"
          >
            {showAdvanced ? 'Hide' : 'Show'} Advanced
          </button>
          {filterCount > 0 && (
            <button
              onClick={onClear}
              className="text-sm text-gray-600 hover:text-gray-700"
            >
              Clear All
            </button>
          )}
        </div>
      </div>

      {/* Quick Filters */}
      <div className="flex gap-2">
        <button
          onClick={() => handleQuickFilter('all')}
          className="px-3 py-1 text-sm rounded-lg border border-gray-300 hover:bg-gray-50"
        >
          All
        </button>
        <button
          onClick={() => handleQuickFilter('simple')}
          className="px-3 py-1 text-sm rounded-lg border border-gray-300 hover:bg-gray-50"
        >
          Simple Repos
        </button>
        <button
          onClick={() => handleQuickFilter('complex')}
          className="px-3 py-1 text-sm rounded-lg border border-gray-300 hover:bg-gray-50"
        >
          Complex Repos
        </button>
      </div>

      {/* Search */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Search</label>
        <input
          type="text"
          value={filters.search || ''}
          onChange={(e) => onChange({ ...filters, search: e.target.value || undefined })}
          placeholder="Search repository names..."
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      {showAdvanced && (
        <>
          {/* Organization Filter */}
          {organizations.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Organizations
              </label>
              <div className="max-h-40 overflow-y-auto border border-gray-200 rounded-lg p-2 space-y-1">
                {organizations.map((org) => (
                  <label key={org} className="flex items-center gap-2 px-2 py-1 hover:bg-gray-50 rounded cursor-pointer">
                    <input
                      type="checkbox"
                      checked={isOrgSelected(org)}
                      onChange={(e) => handleOrganizationChange(org, e.target.checked)}
                      className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">{org}</span>
                  </label>
                ))}
              </div>
            </div>
          )}

          {/* Size Range */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Size Range (MB)
            </label>
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
                className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
                className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          {/* Feature Toggles */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Features</label>
            <div className="space-y-2">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={filters.has_lfs || false}
                  onChange={(e) => onChange({ ...filters, has_lfs: e.target.checked ? true : undefined })}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">Has LFS</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={filters.has_submodules || false}
                  onChange={(e) => onChange({ ...filters, has_submodules: e.target.checked ? true : undefined })}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">Has Submodules</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={filters.has_actions || false}
                  onChange={(e) => onChange({ ...filters, has_actions: e.target.checked ? true : undefined })}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">Has Actions</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={filters.has_wiki || false}
                  onChange={(e) => onChange({ ...filters, has_wiki: e.target.checked ? true : undefined })}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">Has Wiki</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={filters.has_pages || false}
                  onChange={(e) => onChange({ ...filters, has_pages: e.target.checked ? true : undefined })}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">Has Pages</span>
              </label>
            </div>
          </div>

          {/* Sort By */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Sort By</label>
            <select
              value={filters.sort_by || 'name'}
              onChange={(e) =>
                onChange({
                  ...filters,
                  sort_by: (e.target.value as any) || undefined,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="name">Name</option>
              <option value="size">Size</option>
              <option value="org">Organization</option>
              <option value="updated">Last Updated</option>
            </select>
          </div>
        </>
      )}
    </div>
  );
}

