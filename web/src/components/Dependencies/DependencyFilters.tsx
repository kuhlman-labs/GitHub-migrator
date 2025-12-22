import { TextInput } from '@primer/react';
import { SearchIcon } from '@primer/octicons-react';

export type DependencyTypeFilter = 'all' | 'submodule' | 'workflow' | 'dependency_graph' | 'package';

interface DependencyFiltersProps {
  typeFilter: DependencyTypeFilter;
  onTypeFilterChange: (filter: DependencyTypeFilter) => void;
  searchQuery: string;
  onSearchQueryChange: (query: string) => void;
}

export function DependencyFilters({
  typeFilter,
  onTypeFilterChange,
  searchQuery,
  onSearchQueryChange,
}: DependencyFiltersProps) {
  return (
    <div className="flex flex-wrap gap-4 items-center justify-between">
      <div className="flex gap-2">
        <button
          onClick={() => onTypeFilterChange('all')}
          className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
          style={{
            backgroundColor: typeFilter === 'all' ? '#2da44e' : 'var(--control-bgColor-rest)',
            color: typeFilter === 'all' ? '#ffffff' : 'var(--fgColor-default)'
          }}
        >
          All Types
        </button>
        <button
          onClick={() => onTypeFilterChange('submodule')}
          className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
          style={{
            backgroundColor: typeFilter === 'submodule' ? '#0969DA' : 'var(--control-bgColor-rest)',
            color: typeFilter === 'submodule' ? '#ffffff' : 'var(--fgColor-default)'
          }}
        >
          Submodule
        </button>
        <button
          onClick={() => onTypeFilterChange('workflow')}
          className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
          style={{
            backgroundColor: typeFilter === 'workflow' ? '#8250DF' : 'var(--control-bgColor-rest)',
            color: typeFilter === 'workflow' ? '#ffffff' : 'var(--fgColor-default)'
          }}
        >
          Workflow
        </button>
        <button
          onClick={() => onTypeFilterChange('dependency_graph')}
          className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
          style={{
            backgroundColor: typeFilter === 'dependency_graph' ? '#1a7f37' : 'var(--control-bgColor-rest)',
            color: typeFilter === 'dependency_graph' ? '#ffffff' : 'var(--fgColor-default)'
          }}
        >
          Dependency Graph
        </button>
        <button
          onClick={() => onTypeFilterChange('package')}
          className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
          style={{
            backgroundColor: typeFilter === 'package' ? '#656D76' : 'var(--control-bgColor-rest)',
            color: typeFilter === 'package' ? '#ffffff' : 'var(--fgColor-default)'
          }}
        >
          Package
        </button>
      </div>
      
      <TextInput
        leadingVisual={SearchIcon}
        placeholder="Search repositories..."
        value={searchQuery}
        onChange={(e) => onSearchQueryChange(e.target.value)}
        style={{ width: 300 }}
      />
    </div>
  );
}

