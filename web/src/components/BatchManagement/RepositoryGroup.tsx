import { useState } from 'react';
import { ChevronRightIcon } from '@primer/octicons-react';
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
    <div 
      className="rounded-lg overflow-hidden shadow-sm"
      style={{ 
        border: '1px solid var(--borderColor-default)',
        backgroundColor: 'var(--bgColor-default)' 
      }}
    >
      <div 
        style={{ 
          backgroundColor: 'var(--bgColor-muted)',
          borderBottom: '1px solid var(--borderColor-default)' 
        }}
      >
        <div className="flex items-center justify-between p-3">
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              checked={allSelected}
              ref={(el) => {
                if (el) el.indeterminate = someSelected;
              }}
              onChange={handleToggleAll}
              className="rounded text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
              style={{ borderColor: 'var(--borderColor-default)' }}
            />
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="flex items-center gap-2 text-left transition-opacity hover:opacity-80"
            >
              <span style={{ color: 'var(--fgColor-muted)' }}>
              <ChevronRightIcon
                className={`transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                size={16}
              />
              </span>
              <div className="flex items-center gap-2">
                <svg className="w-5 h-5" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                </svg>
                <span className="font-semibold text-sm" style={{ color: 'var(--fgColor-default)' }}>{organization}</span>
                <span 
                  className="px-2 py-0.5 rounded-full text-xs font-medium"
                  style={{
                    backgroundColor: 'var(--bgColor-default)',
                    color: 'var(--fgColor-default)',
                    border: '1px solid var(--borderColor-default)'
                  }}
                >
                  {repositories.length}
                </span>
              </div>
            </button>
          </div>
        </div>
      </div>

      {isExpanded && (
        <div className="p-2 space-y-2" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
          {repositories.map((repo) => (
            <RepositoryListItem
              key={repo.id}
              repository={repo}
              selected={selectedIds.has(repo.id)}
              onToggle={onToggle}
            />
          ))}
        </div>
      )}
    </div>
  );
}

