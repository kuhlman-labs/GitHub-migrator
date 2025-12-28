import { TextInput, SegmentedControl } from '@primer/react';
import { SearchIcon } from '@primer/octicons-react';

export type DependencyTypeFilter = 'all' | 'submodule' | 'workflow' | 'dependency_graph' | 'package';

interface DependencyFiltersProps {
  typeFilter: DependencyTypeFilter;
  onTypeFilterChange: (filter: DependencyTypeFilter) => void;
  searchQuery: string;
  onSearchQueryChange: (query: string) => void;
}

const filterOptions: { value: DependencyTypeFilter; label: string }[] = [
  { value: 'all', label: 'All Types' },
  { value: 'submodule', label: 'Submodule' },
  { value: 'workflow', label: 'Workflow' },
  { value: 'dependency_graph', label: 'Dependency Graph' },
  { value: 'package', label: 'Package' },
];

export function DependencyFilters({
  typeFilter,
  onTypeFilterChange,
  searchQuery,
  onSearchQueryChange,
}: DependencyFiltersProps) {
  const selectedIndex = filterOptions.findIndex(opt => opt.value === typeFilter);

  return (
    <div className="flex flex-wrap gap-4 items-center justify-between">
      <SegmentedControl
        aria-label="Dependency type filter"
        onChange={(index) => onTypeFilterChange(filterOptions[index].value)}
        >
        {filterOptions.map((option, index) => (
          <SegmentedControl.Button
            key={option.value}
            selected={index === selectedIndex}
        >
            {option.label}
          </SegmentedControl.Button>
        ))}
      </SegmentedControl>
      
      <TextInput
        leadingVisual={SearchIcon}
        placeholder="Search repositories..."
        value={searchQuery}
        onChange={(e) => onSearchQueryChange(e.target.value)}
        className="w-[300px]"
      />
    </div>
  );
}

