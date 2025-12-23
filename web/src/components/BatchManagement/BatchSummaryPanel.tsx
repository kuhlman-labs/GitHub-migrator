import { ChevronDownIcon, ChevronUpIcon } from '@primer/octicons-react';
import type { Repository } from '../../types';
import { formatBytes } from '../../utils/format';

interface BatchSummaryPanelProps {
  currentBatchRepos: Repository[];
  groupedRepos: Record<string, Repository[]>;
  totalSize: number;
  onRemoveRepo: (repoId: number) => void;
  onClearAll: () => void;
  isExpanded: boolean;
  onToggleExpanded: () => void;
}

export function BatchSummaryPanel({
  currentBatchRepos,
  groupedRepos,
  totalSize,
  onRemoveRepo,
  onClearAll,
  isExpanded,
  onToggleExpanded,
}: BatchSummaryPanelProps) {
  
  return (
    <div 
      className="flex-shrink-0 border-b"
      style={{ 
        backgroundColor: 'var(--bgColor-muted)', 
        borderColor: 'var(--borderColor-default)'
      }}
    >
      {/* Section Header */}
      <div 
        className="px-4 py-3 flex items-center justify-between cursor-pointer hover:opacity-90 transition-opacity"
        onClick={onToggleExpanded}
        style={{ borderBottom: isExpanded ? '1px solid var(--borderColor-default)' : 'none' }}
      >
        <div className="flex items-center gap-3">
          <div 
            className="p-1 rounded"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            {isExpanded ? <ChevronUpIcon size={16} /> : <ChevronDownIcon size={16} />}
          </div>
          <div>
            <h3 className="text-base font-semibold" style={{ color: 'var(--fgColor-default)' }}>
              Selected Repositories
            </h3>
            <p className="text-xs mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
              {currentBatchRepos.length} {currentBatchRepos.length === 1 ? 'repository' : 'repositories'} · {formatBytes(totalSize)}
            </p>
          </div>
        </div>
        
        <div className="flex items-center gap-3">
          {/* Compact badge showing count when collapsed */}
          {!isExpanded && currentBatchRepos.length > 0 && (
            <span 
              className="px-2.5 py-1 rounded-full text-xs font-bold"
              style={{ 
                backgroundColor: 'var(--accent-subtle)', 
                color: 'var(--fgColor-accent)' 
              }}
            >
              {currentBatchRepos.length}
            </span>
          )}
          {currentBatchRepos.length > 0 && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onClearAll();
              }}
              className="text-xs font-medium transition-colors hover:opacity-80 px-2 py-1 rounded"
              style={{ color: 'var(--fgColor-danger)' }}
            >
              Clear All
            </button>
          )}
        </div>
      </div>

      {/* Collapsible Repository List */}
      {isExpanded && (
        <div 
          className="overflow-y-auto"
          style={{ 
            maxHeight: '280px',
            backgroundColor: 'var(--bgColor-default)'
          }}
        >
          {currentBatchRepos.length === 0 ? (
            <div className="text-center py-8 px-4">
              <svg className="mx-auto h-10 w-10" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p className="mt-2 text-sm font-medium" style={{ color: 'var(--fgColor-muted)' }}>No repositories selected</p>
              <p className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>Select repositories from the left panel</p>
            </div>
          ) : (
            <div className="p-3 space-y-2">
              {Object.entries(groupedRepos).map(([org, repos]) => (
                <div 
                  key={org} 
                  className="rounded-lg overflow-hidden"
                  style={{ 
                    border: '1px solid var(--borderColor-default)',
                    backgroundColor: 'var(--bgColor-default)' 
                  }}
                >
                  <div 
                    className="px-3 py-1.5 flex items-center justify-between"
                    style={{ 
                      backgroundColor: 'var(--bgColor-muted)',
                      borderBottom: '1px solid var(--borderColor-muted)' 
                    }}
                  >
                    <span className="font-medium text-xs" style={{ color: 'var(--fgColor-default)' }}>{org}</span>
                    <span 
                      className="px-1.5 py-0.5 rounded text-xs font-medium"
                      style={{
                        backgroundColor: 'var(--bgColor-default)',
                        color: 'var(--fgColor-muted)',
                        border: '1px solid var(--borderColor-default)'
                      }}
                    >
                      {repos.length}
                    </span>
                  </div>
                  <div>
                    {repos.map((repo, index) => (
                      <div 
                        key={repo.id} 
                        className="px-3 py-2 flex items-center justify-between hover:opacity-80 transition-opacity group"
                        style={{ borderTop: index > 0 ? '1px solid var(--borderColor-muted)' : 'none' }}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-sm truncate" style={{ color: 'var(--fgColor-default)' }}>
                            {repo.ado_project 
                              ? repo.full_name
                              : repo.full_name.split('/')[1] || repo.full_name
                            }
                          </div>
                          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                            {formatBytes(repo.total_size || 0)} · {repo.branch_count} branches
                          </div>
                        </div>
                        <button
                          onClick={() => onRemoveRepo(repo.id)}
                          className="ml-2 p-1 rounded transition-opacity opacity-50 group-hover:opacity-100"
                          style={{ color: 'var(--fgColor-danger)' }}
                          title="Remove repository"
                        >
                          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
