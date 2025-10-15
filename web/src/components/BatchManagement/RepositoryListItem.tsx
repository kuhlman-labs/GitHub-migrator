import type { Repository } from '../../types';
import { StatusBadge } from '../common/StatusBadge';
import { formatBytes } from '../../utils/format';

interface RepositoryListItemProps {
  repository: Repository;
  selected: boolean;
  onToggle: (repoId: number) => void;
}

export function RepositoryListItem({ repository, selected, onToggle }: RepositoryListItemProps) {
  const getComplexityIndicator = () => {
    let complexity = 0;
    if (repository.has_lfs) complexity++;
    if (repository.has_submodules) complexity++;
    if (repository.has_actions) complexity++;
    if (repository.branch_protections > 0) complexity++;
    if (repository.branch_count > 10) complexity++;

    if (complexity === 0) return { label: 'Simple', color: 'text-green-600' };
    if (complexity <= 2) return { label: 'Moderate', color: 'text-yellow-600' };
    return { label: 'Complex', color: 'text-red-600' };
  };

  const complexity = getComplexityIndicator();

  return (
    <label
      className={`flex items-center gap-3 p-3 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors ${
        selected ? 'border-blue-500 bg-blue-50' : 'border-gray-200'
      }`}
    >
      <input
        type="checkbox"
        checked={selected}
        onChange={() => onToggle(repository.id)}
        className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
      />
      
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-medium text-gray-900 truncate">{repository.full_name}</span>
          <StatusBadge status={repository.status} size="sm" />
        </div>
        
        <div className="flex items-center gap-4 text-xs text-gray-600">
          <span>{formatBytes(repository.total_size || 0)}</span>
          <span>{repository.branch_count} branches</span>
          
          <div className="flex items-center gap-1">
            {repository.has_lfs && (
              <span className="px-1.5 py-0.5 bg-purple-100 text-purple-700 rounded text-xs">LFS</span>
            )}
            {repository.has_actions && (
              <span className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">Actions</span>
            )}
            {repository.has_submodules && (
              <span className="px-1.5 py-0.5 bg-orange-100 text-orange-700 rounded text-xs">Submodules</span>
            )}
          </div>
          
          <span className={`ml-auto font-medium ${complexity.color}`}>{complexity.label}</span>
        </div>
      </div>
    </label>
  );
}

