import { useState } from 'react';
import type { Repository } from '../../types';
import { RepositoryListItem } from './RepositoryListItem';

interface RepositoryGroupProps {
  organization: string;
  repositories: Repository[];
  selectedIds: Set<number>;
  onToggle: (repoId: number) => void;
  onToggleAll: (repoIds: number[]) => void;
}

export function RepositoryGroup({
  organization,
  repositories,
  selectedIds,
  onToggle,
  onToggleAll,
}: RepositoryGroupProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  const allSelected = repositories.every((repo) => selectedIds.has(repo.id));
  const someSelected = repositories.some((repo) => selectedIds.has(repo.id)) && !allSelected;

  const handleToggleAll = () => {
    const repoIds = repositories.map((r) => r.id);
    onToggleAll(repoIds);
  };

  return (
    <div className="border border-gray-200 rounded-lg overflow-hidden">
      <div className="bg-gray-50 border-b border-gray-200">
        <div className="flex items-center justify-between p-3">
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={allSelected}
              ref={(el) => {
                if (el) el.indeterminate = someSelected;
              }}
              onChange={handleToggleAll}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="flex items-center gap-2 text-left hover:text-blue-600"
            >
              <svg
                className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
              <span className="font-medium text-gray-900">{organization}</span>
              <span className="text-sm text-gray-600">({repositories.length})</span>
            </button>
          </div>
        </div>
      </div>

      {isExpanded && (
        <div className="divide-y divide-gray-200">
          {repositories.map((repo) => (
            <div key={repo.id} className="px-3 py-2">
              <RepositoryListItem
                repository={repo}
                selected={selectedIds.has(repo.id)}
                onToggle={onToggle}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

