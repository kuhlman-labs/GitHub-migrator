import type { Repository } from '../../types';
import { formatBytes } from '../../utils/format';

interface BatchSummaryPanelProps {
  currentBatchRepos: Repository[];
  groupedRepos: Record<string, Repository[]>;
  totalSize: number;
  onRemoveRepo: (repoId: number) => void;
  onClearAll: () => void;
}

export function BatchSummaryPanel({
  currentBatchRepos,
  groupedRepos,
  totalSize,
  onRemoveRepo,
  onClearAll,
}: BatchSummaryPanelProps) {
  return (
    <>
      {/* Sticky Header with Batch Info */}
      <div 
        className="flex-shrink-0 sticky top-0 shadow-sm"
        style={{ 
          backgroundColor: 'var(--bgColor-default)', 
          borderBottom: '1px solid var(--borderColor-default)',
          zIndex: 1
        }}
      >
        <div className="p-4">
          <div className="flex justify-between items-center mb-3">
            <div>
              <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                Selected Repositories
              </h3>
              <p className="text-sm mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                {currentBatchRepos.length} {currentBatchRepos.length === 1 ? 'repository' : 'repositories'}
              </p>
            </div>
            {currentBatchRepos.length > 0 && (
              <button
                onClick={onClearAll}
                className="text-sm font-medium transition-colors hover:opacity-80"
                style={{ color: 'var(--fgColor-danger)' }}
              >
                Clear All
              </button>
            )}
          </div>
          {/* Batch Size Indicator */}
          <div 
            className="border p-2.5 rounded-lg"
            style={{ backgroundColor: 'var(--accent-subtle)', borderColor: 'var(--accent-muted)' }}
          >
            <div className="flex items-center justify-between">
              <div className="text-xs font-medium" style={{ color: 'var(--fgColor-accent)' }}>Total Batch Size</div>
              <div className="text-lg font-bold" style={{ color: 'var(--fgColor-accent)' }}>{formatBytes(totalSize)}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Repository List - Scrollable with expanded height */}
      <div className="flex-1 overflow-y-auto p-4 min-h-0">
        {currentBatchRepos.length === 0 ? (
          <div className="text-center py-12">
            <svg className="mx-auto h-12 w-12" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <p className="mt-2 text-sm" style={{ color: 'var(--fgColor-muted)' }}>No repositories selected</p>
            <p className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>Select repositories from the left</p>
          </div>
        ) : (
          <div className="space-y-3">
            {Object.entries(groupedRepos).map(([org, repos]) => (
              <div 
                key={org} 
                className="rounded-lg overflow-hidden shadow-sm"
                style={{ 
                  border: '1px solid var(--borderColor-default)',
                  backgroundColor: 'var(--bgColor-default)' 
                }}
              >
                <div 
                  className="px-3 py-2"
                  style={{ 
                    backgroundColor: 'var(--bgColor-muted)',
                    borderBottom: '1px solid var(--borderColor-default)' 
                  }}
                >
                  <span className="font-semibold text-sm" style={{ color: 'var(--fgColor-default)' }}>{org}</span>
                  <span 
                    className="ml-2 px-2 py-0.5 rounded-full text-xs font-medium"
                    style={{
                      backgroundColor: 'var(--bgColor-default)',
                      color: 'var(--fgColor-default)',
                      border: '1px solid var(--borderColor-default)'
                    }}
                  >
                    {repos.length}
                  </span>
                </div>
                <div style={{ borderTop: '1px solid var(--borderColor-muted)' }}>
                  {repos.map((repo, index) => (
                    <div 
                      key={repo.id} 
                      className="p-3 flex items-center justify-between hover:opacity-80 transition-opacity"
                      style={{ borderTop: index > 0 ? '1px solid var(--borderColor-muted)' : 'none' }}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm truncate" style={{ color: 'var(--fgColor-default)' }}>
                          {repo.ado_project 
                            ? repo.full_name // For ADO, full_name is just the repo name
                            : repo.full_name.split('/')[1] || repo.full_name // For GitHub, extract repo name from org/repo
                          }
                        </div>
                        <div className="text-xs mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                          {formatBytes(repo.total_size || 0)} • {repo.branch_count} branches
                        </div>
                      </div>
                      <button
                        onClick={() => onRemoveRepo(repo.id)}
                        className="ml-2 p-1 rounded transition-opacity hover:opacity-80"
                        style={{ color: 'var(--fgColor-danger)' }}
                        title="Remove repository"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
    </>
  );
}

