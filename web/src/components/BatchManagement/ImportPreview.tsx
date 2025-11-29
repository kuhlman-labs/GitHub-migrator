import { useState } from 'react';
import { Button } from '@primer/react';
import type { Repository } from '../../types';

export interface ValidationGroup {
  valid: Repository[];
  alreadyInBatch: Repository[];
  notFound: { full_name: string }[];
}

interface ImportPreviewProps {
  validationResult: ValidationGroup;
  onConfirm: (selectedRepos: Repository[]) => void;
  onCancel: () => void;
}

export function ImportPreview({ validationResult, onConfirm, onCancel }: ImportPreviewProps) {
  const [selectedIds, setSelectedIds] = useState<Set<number>>(
    new Set(validationResult.valid.map(r => r.id))
  );

  const handleToggle = (repoId: number) => {
    const newSelected = new Set(selectedIds);
    if (newSelected.has(repoId)) {
      newSelected.delete(repoId);
    } else {
      newSelected.add(repoId);
    }
    setSelectedIds(newSelected);
  };

  const handleToggleAll = () => {
    if (selectedIds.size === validationResult.valid.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(validationResult.valid.map(r => r.id)));
    }
  };

  const handleConfirm = () => {
    const selected = validationResult.valid.filter(r => selectedIds.has(r.id));
    onConfirm(selected);
  };

  const totalCount = validationResult.valid.length + validationResult.alreadyInBatch.length + validationResult.notFound.length;

  return (
    <div 
      className="fixed inset-0 flex items-center justify-center z-[100]"
      style={{ backgroundColor: 'rgba(0, 0, 0, 0.6)' }}
    >
      <div 
        className="relative rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[80vh] flex flex-col"
        style={{ 
          backgroundColor: 'var(--bgColor-default)',
          border: '1px solid var(--borderColor-default)'
        }}
      >
        {/* Header */}
        <div 
          className="px-6 py-4 border-b"
          style={{ borderColor: 'var(--borderColor-default)' }}
        >
          <h2 className="text-xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            Import Preview
          </h2>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Reviewed {totalCount} repositories from file
          </p>
        </div>

        {/* Content - Scrollable */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {/* Valid & Available Section */}
          {validationResult.valid.length > 0 && (
            <div>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-lg font-semibold flex items-center gap-2" style={{ color: 'var(--fgColor-default)' }}>
                  <span 
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: 'var(--success-emphasis)' }}
                  />
                  Valid & Available ({validationResult.valid.length})
                </h3>
                <button
                  onClick={handleToggleAll}
                  className="text-sm font-medium hover:underline"
                  style={{ color: 'var(--fgColor-accent)' }}
                >
                  {selectedIds.size === validationResult.valid.length ? 'Deselect All' : 'Select All'}
                </button>
              </div>
              <div 
                className="rounded-lg border"
                style={{ 
                  backgroundColor: 'var(--success-subtle)',
                  borderColor: 'var(--success-muted)'
                }}
              >
                <div className="divide-y" style={{ borderColor: 'var(--borderColor-muted)' }}>
                  {validationResult.valid.map((repo) => (
                    <div 
                      key={repo.id}
                      className="p-4 hover:bg-[var(--control-bgColor-hover)] transition-colors"
                    >
                      <div className="flex items-start gap-3">
                        <input
                          type="checkbox"
                          checked={selectedIds.has(repo.id)}
                          onChange={() => handleToggle(repo.id)}
                          className="mt-1 rounded text-blue-600 focus:ring-blue-500"
                          style={{ borderColor: 'var(--borderColor-default)' }}
                        />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between">
                            <h4 className="font-medium" style={{ color: 'var(--fgColor-default)' }}>
                              {repo.full_name}
                            </h4>
                            <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                              {repo.source}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Already in Batch Section */}
          {validationResult.alreadyInBatch.length > 0 && (
            <div>
              <h3 className="text-lg font-semibold flex items-center gap-2 mb-3" style={{ color: 'var(--fgColor-default)' }}>
                <span 
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: 'var(--attention-emphasis)' }}
                />
                Already in a Batch ({validationResult.alreadyInBatch.length})
              </h3>
              <div 
                className="rounded-lg border"
                style={{ 
                  backgroundColor: 'var(--attention-subtle)',
                  borderColor: 'var(--attention-muted)'
                }}
              >
                <div className="divide-y" style={{ borderColor: 'var(--borderColor-muted)' }}>
                  {validationResult.alreadyInBatch.map((repo) => (
                    <div key={repo.id} className="p-4">
                      <div className="flex items-center justify-between">
                        <h4 className="font-medium" style={{ color: 'var(--fgColor-default)' }}>
                          {repo.full_name}
                        </h4>
                        {repo.batch_id && (
                          <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                            Batch ID: {repo.batch_id}
                          </span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Not Found Section */}
          {validationResult.notFound.length > 0 && (
            <div>
              <h3 className="text-lg font-semibold flex items-center gap-2 mb-3" style={{ color: 'var(--fgColor-default)' }}>
                <span 
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: 'var(--danger-emphasis)' }}
                />
                Not Found ({validationResult.notFound.length})
              </h3>
              <div 
                className="rounded-lg border"
                style={{ 
                  backgroundColor: 'var(--danger-subtle)',
                  borderColor: 'var(--danger-muted)'
                }}
              >
                <div className="divide-y" style={{ borderColor: 'var(--borderColor-muted)' }}>
                  {validationResult.notFound.map((item, idx) => (
                    <div key={idx} className="p-4">
                      <h4 className="font-medium" style={{ color: 'var(--fgColor-default)' }}>
                        {item.full_name}
                      </h4>
                      <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                        Repository not found in database
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Empty state */}
          {validationResult.valid.length === 0 && 
           validationResult.alreadyInBatch.length === 0 && 
           validationResult.notFound.length === 0 && (
            <div className="text-center py-12">
              <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                No repositories found in the import file
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div 
          className="px-6 py-4 border-t flex items-center justify-between"
          style={{ 
            borderColor: 'var(--borderColor-default)',
            backgroundColor: 'var(--bgColor-muted)'
          }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
            {selectedIds.size} of {validationResult.valid.length} repositories selected
          </div>
          <div className="flex gap-2">
            <Button onClick={onCancel}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleConfirm}
              disabled={selectedIds.size === 0}
            >
              Add {selectedIds.size > 0 ? `${selectedIds.size} ` : ''}Repositories
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

