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
    // Calculate complexity score matching backend logic:
    // Size tier * 3 + has_lfs (2) + has_submodules (2) + has_large_files (2) + branch_protections > 0 (1)
    const MB100 = 100 * 1024 * 1024;
    const GB1 = 1024 * 1024 * 1024;
    const GB5 = 5 * 1024 * 1024 * 1024;
    
    let sizeTier = 0;
    if (repository.total_size >= GB5) sizeTier = 3;
    else if (repository.total_size >= GB1) sizeTier = 2;
    else if (repository.total_size >= MB100) sizeTier = 1;
    
    let score = sizeTier * 3;
    if (repository.has_lfs) score += 2;
    if (repository.has_submodules) score += 2;
    if (repository.has_large_files) score += 2;
    if (repository.branch_protections > 0) score += 1;

    if (score <= 3) return { label: 'Simple', color: 'text-gh-success' };
    if (score <= 6) return { label: 'Medium', color: 'text-gh-warning' };
    if (score <= 9) return { label: 'Complex', color: 'text-gh-danger' };
    return { label: 'Very Complex', color: 'text-gh-danger' };
  };

  const complexity = getComplexityIndicator();

  return (
    <label
      className={`flex items-center gap-3 p-3 border rounded-md hover:bg-gh-neutral-bg cursor-pointer transition-colors ${
        selected ? 'border-gh-blue bg-gh-info-bg' : 'border-gh-border-default'
      }`}
    >
      <input
        type="checkbox"
        checked={selected}
        onChange={() => onToggle(repository.id)}
        className="rounded border-gh-border-default text-gh-blue focus:ring-gh-blue"
      />
      
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-semibold text-gh-text-primary truncate text-sm">{repository.full_name}</span>
          <StatusBadge status={repository.status} size="sm" />
        </div>
        
        <div className="flex items-center gap-3 text-xs text-gh-text-secondary">
          <span>{formatBytes(repository.total_size || 0)}</span>
          <span>{repository.branch_count} branches</span>
          
          <div className="flex items-center gap-1.5">
            {repository.has_lfs && (
              <span className="px-2 py-0.5 bg-purple-100 text-purple-800 rounded-full text-xs font-medium border border-purple-200">LFS</span>
            )}
            {repository.has_actions && (
              <span className="px-2 py-0.5 bg-gh-info-bg text-gh-blue rounded-full text-xs font-medium border border-gh-blue/20">Actions</span>
            )}
            {repository.has_submodules && (
              <span className="px-2 py-0.5 bg-orange-100 text-orange-800 rounded-full text-xs font-medium border border-orange-200">Submodules</span>
            )}
          </div>
          
          <span className={`ml-auto font-semibold ${complexity.color}`}>{complexity.label}</span>
        </div>
      </div>
    </label>
  );
}

