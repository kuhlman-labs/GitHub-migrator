import { useEffect, useState } from 'react';
import { TextInput, Select, Checkbox, FormControl } from '@primer/react';
import { SearchIcon } from '@primer/octicons-react';
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

  const handleSelectAllOrgs = () => {
    onChange({ ...filters, organization: organizations });
  };

  const handleDeselectAllOrgs = () => {
    onChange({ ...filters, organization: undefined });
  };

  const getSelectedOrgCount = () => {
    if (!filters.organization) return 0;
    if (Array.isArray(filters.organization)) return filters.organization.length;
    return 1;
  };

  const activeFilterCount = () => {
    let count = 0;
    if (filters.organization) count++;
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
    if (filters.has_rulesets) count++;
    if (filters.has_code_scanning) count++;
    if (filters.has_dependabot) count++;
    if (filters.has_secret_scanning) count++;
    if (filters.has_codeowners) count++;
    if (filters.has_self_hosted_runners) count++;
    if (filters.has_release_assets) count++;
    if (filters.has_webhooks) count++;
    if (filters.is_archived !== undefined) count++;
    if (filters.is_fork !== undefined) count++;
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
          complexity: ['simple'],
        });
        break;
      case 'complex':
        onChange({
          complexity: ['complex', 'very_complex'],
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

      {/* Organization Filter - Always Visible */}
      {organizations.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-sm font-medium text-gray-700">
              Filter by Organization
              {getSelectedOrgCount() > 0 && (
                <span className="ml-2 text-xs text-blue-600 font-semibold">
                  ({getSelectedOrgCount()} selected)
                </span>
              )}
            </label>
            <div className="flex gap-2">
              <button
                onClick={handleSelectAllOrgs}
                className="text-xs text-blue-600 hover:text-blue-700 font-medium"
              >
                Select All
              </button>
              {getSelectedOrgCount() > 0 && (
                <button
                  onClick={handleDeselectAllOrgs}
                  className="text-xs text-gray-600 hover:text-gray-700 font-medium"
                >
                  Clear
                </button>
              )}
            </div>
          </div>
          <div className="max-h-48 overflow-y-auto border border-gray-200 rounded-lg p-2 space-y-1 bg-white">
            {organizations.map((org) => (
              <div key={org} className="px-2 py-1.5 hover:bg-blue-50 rounded transition-colors">
                <Checkbox
                  checked={isOrgSelected(org)}
                  onChange={(e) => handleOrganizationChange(org, e.target.checked)}
                >
                  {org}
                </Checkbox>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Search */}
      <FormControl>
        <FormControl.Label>Search</FormControl.Label>
        <TextInput
          leadingVisual={SearchIcon}
          value={filters.search || ''}
          onChange={(e) => onChange({ ...filters, search: e.target.value || undefined })}
          placeholder="Search repository names..."
          block
        />
      </FormControl>

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

      {showAdvanced && (
        <>

          {/* Size Category */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Size Category</label>
            <select
              multiple
              value={Array.isArray(filters.size_category) ? filters.size_category : (filters.size_category ? [filters.size_category] : [])}
              onChange={(e) => {
                const selected = Array.from(e.target.selectedOptions, option => option.value);
                onChange({
                  ...filters,
                  size_category: selected.length > 0 ? selected : undefined,
                });
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              size={5}
            >
              <option value="small">Small (&lt;100MB)</option>
              <option value="medium">Medium (100MB-1GB)</option>
              <option value="large">Large (1GB-5GB)</option>
              <option value="very_large">Very Large (&gt;5GB)</option>
              <option value="unknown">Unknown</option>
            </select>
            <p className="text-xs text-gray-500 mt-1">Hold Ctrl/Cmd to select multiple</p>
          </div>

          {/* Complexity */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Complexity</label>
            <select
              multiple
              value={Array.isArray(filters.complexity) ? filters.complexity : (filters.complexity ? [filters.complexity] : [])}
              onChange={(e) => {
                const selected = Array.from(e.target.selectedOptions, option => option.value);
                onChange({
                  ...filters,
                  complexity: selected.length > 0 ? selected : undefined,
                });
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              size={4}
            >
              <option value="simple">Simple</option>
              <option value="medium">Medium</option>
              <option value="complex">Complex</option>
              <option value="very_complex">Very Complex</option>
            </select>
            <p className="text-xs text-gray-500 mt-1">Hold Ctrl/Cmd to select multiple</p>
          </div>

          {/* Size Range */}
          <FormControl>
            <FormControl.Label>Size Range (MB)</FormControl.Label>
            <div className="grid grid-cols-2 gap-2">
              <TextInput
                type="number"
                placeholder="Min"
                value={filters.min_size ? Math.round(filters.min_size / 1024 / 1024).toString() : ''}
                onChange={(e) =>
                  onChange({
                    ...filters,
                    min_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 : undefined,
                  })
                }
              />
              <TextInput
                type="number"
                placeholder="Max"
                value={filters.max_size ? Math.round(filters.max_size / 1024 / 1024).toString() : ''}
                onChange={(e) =>
                  onChange({
                    ...filters,
                    max_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 : undefined,
                  })
                }
              />
            </div>
          </FormControl>

          {/* Feature Toggles */}
          <FormControl>
            <FormControl.Label>Features</FormControl.Label>
            <div className="space-y-2">
              <Checkbox
                checked={filters.has_lfs || false}
                onChange={(e) => onChange({ ...filters, has_lfs: e.target.checked ? true : undefined })}
              >
                Has LFS
              </Checkbox>
              <Checkbox
                checked={filters.has_submodules || false}
                onChange={(e) => onChange({ ...filters, has_submodules: e.target.checked ? true : undefined })}
              >
                Has Submodules
              </Checkbox>
              <Checkbox
                checked={filters.has_large_files || false}
                onChange={(e) => onChange({ ...filters, has_large_files: e.target.checked ? true : undefined })}
              >
                Has Large Files (&gt;100MB)
              </Checkbox>
              <Checkbox
                checked={filters.has_actions || false}
                onChange={(e) => onChange({ ...filters, has_actions: e.target.checked ? true : undefined })}
              >
                Has Actions
              </Checkbox>
              <Checkbox
                checked={filters.has_wiki || false}
                onChange={(e) => onChange({ ...filters, has_wiki: e.target.checked ? true : undefined })}
              >
                Has Wiki
              </Checkbox>
              <Checkbox
                checked={filters.has_pages || false}
                onChange={(e) => onChange({ ...filters, has_pages: e.target.checked ? true : undefined })}
              >
                Has Pages
              </Checkbox>
              <Checkbox
                checked={filters.has_discussions || false}
                onChange={(e) => onChange({ ...filters, has_discussions: e.target.checked ? true : undefined })}
              >
                Has Discussions
              </Checkbox>
              <Checkbox
                checked={filters.has_projects || false}
                onChange={(e) => onChange({ ...filters, has_projects: e.target.checked ? true : undefined })}
              >
                Has Projects
              </Checkbox>
              <Checkbox
                checked={filters.has_branch_protections || false}
                onChange={(e) => onChange({ ...filters, has_branch_protections: e.target.checked ? true : undefined })}
              >
                Has Branch Protections
              </Checkbox>
              <Checkbox
                checked={filters.is_archived || false}
                onChange={(e) => onChange({ ...filters, is_archived: e.target.checked ? true : undefined })}
              >
                Is Archived
              </Checkbox>
            </div>
          </FormControl>

          {/* Sort By */}
          <FormControl>
            <FormControl.Label>Sort By</FormControl.Label>
            <Select
              value={filters.sort_by || 'name'}
              onChange={(e) =>
                onChange({
                  ...filters,
                  sort_by: (e.target.value as any) || undefined,
                })
              }
              block
            >
              <option value="name">Name</option>
              <option value="size">Size</option>
              <option value="org">Organization</option>
              <option value="updated">Last Updated</option>
            </Select>
          </FormControl>
        </>
      )}
    </div>
  );
}

