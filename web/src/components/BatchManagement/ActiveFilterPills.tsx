import type { RepositoryFilters } from '../../types';

interface ActiveFilterPillsProps {
  filters: RepositoryFilters;
  onRemoveFilter: (filterKey: keyof RepositoryFilters) => void;
  onClearAll: () => void;
}

export function ActiveFilterPills({ filters, onRemoveFilter, onClearAll }: ActiveFilterPillsProps) {
  const activeFilters: Array<{ key: keyof RepositoryFilters; label: string; value: string }> = [];

  // Organization
  if (filters.organization) {
    const orgs = Array.isArray(filters.organization) ? filters.organization : [filters.organization];
    if (orgs.length > 0) {
      activeFilters.push({
        key: 'organization',
        label: 'Organization',
        value: orgs.length === 1 ? orgs[0] : `${orgs.length} selected`,
      });
    }
  }

  // Search
  if (filters.search) {
    activeFilters.push({
      key: 'search',
      label: 'Search',
      value: filters.search,
    });
  }

  // Size Category
  if (filters.size_category) {
    const categories = Array.isArray(filters.size_category) ? filters.size_category : [filters.size_category];
    if (categories.length > 0) {
      activeFilters.push({
        key: 'size_category',
        label: 'Size',
        value: categories.length === 1 ? categories[0] : `${categories.length} selected`,
      });
    }
  }

  // Complexity
  if (filters.complexity) {
    const complexities = Array.isArray(filters.complexity) ? filters.complexity : [filters.complexity];
    if (complexities.length > 0) {
      activeFilters.push({
        key: 'complexity',
        label: 'Complexity',
        value: complexities.length === 1 ? complexities[0] : `${complexities.length} selected`,
      });
    }
  }

  // Size Range
  if (filters.min_size || filters.max_size) {
    const minMB = filters.min_size ? Math.round(filters.min_size / 1024 / 1024) : 0;
    const maxMB = filters.max_size ? Math.round(filters.max_size / 1024 / 1024) : '∞';
    activeFilters.push({
      key: 'min_size',
      label: 'Size Range',
      value: `${minMB}-${maxMB} MB`,
    });
  }

  // Feature flags
  const featureLabels: Array<[keyof RepositoryFilters, string]> = [
    ['has_lfs', 'LFS'],
    ['has_submodules', 'Submodules'],
    ['has_large_files', 'Large Files'],
    ['has_actions', 'Actions'],
    ['has_wiki', 'Wiki'],
    ['has_pages', 'Pages'],
    ['has_discussions', 'Discussions'],
    ['has_projects', 'Projects'],
    ['has_branch_protections', 'Branch Protections'],
    ['is_archived', 'Archived'],
    ['has_environments', 'Environments'],
    ['has_secrets', 'Secrets'],
    ['has_variables', 'Variables'],
  ];

  featureLabels.forEach(([key, label]) => {
    if (filters[key]) {
      activeFilters.push({ key, label: 'Has', value: label });
    }
  });

  // Sort
  if (filters.sort_by && filters.sort_by !== 'name') {
    activeFilters.push({
      key: 'sort_by',
      label: 'Sort',
      value: filters.sort_by,
    });
  }

  if (activeFilters.length === 0) {
    return null;
  }

  return (
    <div 
      className="flex items-center gap-2 flex-wrap mb-4 p-3 rounded-lg"
      style={{ 
        backgroundColor: 'var(--accent-subtle)',
        border: '1px solid var(--borderColor-accent-muted)' 
      }}
    >
      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>Active filters:</span>
      {activeFilters.map((filter, index) => (
        <button
          key={`${filter.key}-${index}`}
          onClick={() => onRemoveFilter(filter.key)}
          className="inline-flex items-center gap-1.5 px-3 py-1 text-sm rounded-full transition-opacity hover:opacity-80 group"
          style={{
            backgroundColor: 'var(--bgColor-default)',
            border: '1px solid var(--borderColor-default)',
            color: 'var(--fgColor-default)'
          }}
        >
          <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>{filter.label}:</span>
          <span style={{ color: 'var(--fgColor-muted)' }}>{filter.value}</span>
          <svg
            className="w-3.5 h-3.5 group-hover:opacity-100"
            style={{ color: 'var(--fgColor-muted)' }}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      ))}
      {activeFilters.length > 1 && (
        <button
          onClick={onClearAll}
          className="ml-2 text-sm font-medium underline hover:opacity-80"
          style={{ color: 'var(--fgColor-accent)' }}
        >
          Clear all
        </button>
      )}
    </div>
  );
}

