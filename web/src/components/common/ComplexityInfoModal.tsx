import { useState } from 'react';
import { Button, IconButton } from '@primer/react';
import { InfoIcon, XIcon } from '@primer/octicons-react';

interface ComplexityInfoModalProps {
  source?: 'github' | 'ghes' | 'azuredevops' | 'all';
}

export function ComplexityInfoModal({ source = 'all' }: ComplexityInfoModalProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="inline-flex items-center gap-1 text-sm font-medium hover:underline cursor-pointer"
        style={{ 
          color: 'var(--fgColor-accent)',
          background: 'none',
          border: 'none',
          padding: 0
        }}
      >
        <InfoIcon size={16} />
        How is complexity calculated?
      </button>

      {isOpen && (
        <div 
          className="fixed inset-0 flex items-center justify-center"
          style={{ zIndex: 9999 }}
        >
          {/* Backdrop */}
          <div 
            className="absolute inset-0" 
            style={{ backgroundColor: 'rgba(0, 0, 0, 0.6)' }}
            onClick={() => setIsOpen(false)}
          />
          
          {/* Dialog */}
          <div
            className="relative rounded-lg shadow-2xl"
            style={{
              backgroundColor: 'var(--bgColor-default)',
              border: '1px solid var(--borderColor-default)',
              width: '90%',
              maxWidth: '800px',
              maxHeight: '90vh',
              display: 'flex',
              flexDirection: 'column',
              zIndex: 10000,
            }}
          >
            {/* Header */}
            <div 
              className="flex items-center justify-between px-6 py-4"
              style={{ borderBottom: '1px solid var(--borderColor-default)' }}
            >
              <h2 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                {source === 'azuredevops' ? 'Azure DevOps Migration Complexity' : 
                 source === 'github' ? 'GitHub Migration Complexity' :
                 'Repository Complexity Scoring'}
              </h2>
              <IconButton
                icon={XIcon}
                onClick={() => setIsOpen(false)}
                aria-label="Close"
                variant="invisible"
              />
            </div>

            <div 
              className="overflow-y-auto flex-grow"
              style={{ maxHeight: 'calc(90vh - 120px)' }}
            >
              <div className="p-6 space-y-6">
                {/* Overview */}
                <div>
                  <h3 className="text-base font-semibold mb-2" style={{ color: 'var(--fgColor-default)' }}>Overview</h3>
                  <p className="text-sm mb-2" style={{ color: 'var(--fgColor-default)' }}>
                    {source === 'azuredevops' 
                      ? 'We calculate an Azure DevOps-specific complexity score to estimate migration effort from ADO to GitHub.'
                      : (source === 'github' || source === 'ghes')
                      ? 'We calculate a GitHub-specific complexity score to estimate migration effort between GitHub instances.'
                      : 'We calculate source-specific complexity scores to estimate migration effort and potential challenges.'}
                  </p>
                  <p className="text-xs italic" style={{ color: 'var(--fgColor-muted)' }}>
                    Activity levels are calculated using quantiles relative to your repository dataset.
                  </p>
                </div>

                {/* Scoring Factors - GitHub Specific */}
                {(source === 'github' || source === 'ghes') && (
                  <div>
                    <h3 className="text-base font-semibold mb-3" style={{ color: 'var(--fgColor-default)' }}>Scoring Factors</h3>
                    
                    {/* Repository Size */}
                    <div 
                      className="mb-4 p-4 rounded-lg"
                      style={{
                        backgroundColor: 'var(--accent-subtle)',
                        border: '1px solid var(--borderColor-accent-muted)'
                      }}
                    >
                      <div className="flex justify-between items-start mb-2">
                        <h4 className="font-semibold" style={{ color: 'var(--fgColor-accent)' }}>Repository Size</h4>
                        <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-accent)' }}>Weight: 3 (max 9 points)</span>
                      </div>
                      <p className="text-sm mb-2" style={{ color: 'var(--fgColor-accent)' }}>
                        Larger repositories take longer to migrate and have higher resource requirements.
                      </p>
                      <ul className="text-sm space-y-1 ml-4" style={{ color: 'var(--fgColor-accent)' }}>
                        <li>• &lt;100MB: <span className="font-medium">0 points</span></li>
                        <li>• 100MB - 1GB: <span className="font-medium">3 points</span></li>
                        <li>• 1GB - 5GB: <span className="font-medium">6 points</span></li>
                        <li>• &gt;5GB: <span className="font-medium">9 points</span></li>
                      </ul>
                    </div>

                    {/* High Impact Features */}
                    <div className="mb-3">
                      <h4 className="font-semibold mb-2" style={{ color: 'var(--fgColor-danger)' }}>High Impact (3-4 points each)</h4>
                      <p className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>Features requiring significant remediation effort</p>
                      
                      <div className="space-y-2">
                        <div 
                          className="p-3 rounded"
                          style={{
                            backgroundColor: 'var(--danger-subtle)',
                            border: '1px solid var(--borderColor-danger)'
                          }}
                        >
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium" style={{ color: 'var(--fgColor-danger)' }}>Large Files (&gt;100MB)</span>
                            <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-danger)' }}>4 points</span>
                          </div>
                          <p className="text-sm" style={{ color: 'var(--fgColor-danger)' }}>Must be remediated before migration (migrate to LFS, remove from history)</p>
                        </div>

                        <div 
                          className="p-3 rounded"
                          style={{
                            backgroundColor: 'var(--danger-subtle)',
                            border: '1px solid var(--borderColor-danger)'
                          }}
                        >
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium" style={{ color: 'var(--fgColor-danger)' }}>Environments</span>
                            <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-danger)' }}>3 points</span>
                          </div>
                          <p className="text-sm" style={{ color: 'var(--fgColor-danger)' }}>Don't migrate. Manual recreation of all configs and protection rules required</p>
                        </div>

                        <div 
                          className="p-3 rounded"
                          style={{
                            backgroundColor: 'var(--danger-subtle)',
                            border: '1px solid var(--borderColor-danger)'
                          }}
                        >
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium" style={{ color: 'var(--fgColor-danger)' }}>Secrets</span>
                            <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-danger)' }}>3 points</span>
                          </div>
                          <p className="text-sm" style={{ color: 'var(--fgColor-danger)' }}>Don't migrate. Manual recreation required with high security sensitivity</p>
                        </div>

                        <div 
                          className="p-3 rounded"
                          style={{
                            backgroundColor: 'var(--danger-subtle)',
                            border: '1px solid var(--borderColor-danger)'
                          }}
                        >
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium" style={{ color: 'var(--fgColor-danger)' }}>GitHub Packages</span>
                            <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-danger)' }}>3 points</span>
                          </div>
                          <p className="text-sm" style={{ color: 'var(--fgColor-danger)' }}>Don't migrate with GEI. Manual migration planning required</p>
                        </div>
                      </div>
                    </div>

                    {/* Moderate Impact Features */}
                    <div className="mb-3">
                      <h4 className="font-semibold mb-2" style={{ color: 'var(--fgColor-attention)' }}>Moderate Impact (2 points each)</h4>
                      <p className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>Features requiring manual intervention</p>
                      
                      <div className="space-y-2">
                        <div 
                          className="p-2 rounded"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)'
                          }}
                        >
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-sm" style={{ color: 'var(--fgColor-attention)' }}>Variables</span>
                            <span className="text-xs font-semibold" style={{ color: 'var(--fgColor-attention)' }}>2 points</span>
                          </div>
                        </div>
                        <div 
                          className="p-2 rounded"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)'
                          }}
                        >
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-sm" style={{ color: 'var(--fgColor-attention)' }}>Discussions</span>
                            <span className="text-xs font-semibold" style={{ color: 'var(--fgColor-attention)' }}>2 points</span>
                          </div>
                        </div>
                        <div 
                          className="p-2 rounded"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)'
                          }}
                        >
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-sm" style={{ color: 'var(--fgColor-attention)' }}>Git LFS</span>
                            <span className="text-xs font-semibold" style={{ color: 'var(--fgColor-attention)' }}>2 points</span>
                          </div>
                        </div>
                        <div 
                          className="p-2 rounded"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)'
                          }}
                        >
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-sm" style={{ color: 'var(--fgColor-attention)' }}>Submodules</span>
                            <span className="text-xs font-semibold" style={{ color: 'var(--fgColor-attention)' }}>2 points</span>
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Low Impact Features */}
                    <div className="mb-3">
                      <h4 className="font-semibold mb-2" style={{ color: 'var(--fgColor-attention)' }}>Low Impact (1 point each)</h4>
                      <p className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>Features requiring straightforward manual steps</p>
                      
                      <div className="grid grid-cols-2 gap-2">
                        <div 
                          className="p-2 rounded text-sm"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)',
                            color: 'var(--fgColor-attention)'
                          }}
                        >Advanced Security</div>
                        <div 
                          className="p-2 rounded text-sm"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)',
                            color: 'var(--fgColor-attention)'
                          }}
                        >Webhooks</div>
                        <div 
                          className="p-2 rounded text-sm"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)',
                            color: 'var(--fgColor-attention)'
                          }}
                        >Branch Protections</div>
                        <div 
                          className="p-2 rounded text-sm"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)',
                            color: 'var(--fgColor-attention)'
                          }}
                        >Rulesets</div>
                        <div 
                          className="p-2 rounded text-sm"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)',
                            color: 'var(--fgColor-attention)'
                          }}
                        >Public/Internal Repos</div>
                        <div 
                          className="p-2 rounded text-sm"
                          style={{
                            backgroundColor: 'var(--attention-subtle)',
                            border: '1px solid var(--borderColor-attention)',
                            color: 'var(--fgColor-attention)'
                          }}
                        >CODEOWNERS</div>
                      </div>
                    </div>

                    {/* Activity Level */}
                    <div 
                      className="p-4 rounded-lg"
                      style={{
                        backgroundColor: 'var(--done-subtle)',
                        border: '1px solid var(--borderColor-done-muted)'
                      }}
                    >
                      <div className="flex justify-between items-start mb-2">
                        <h4 className="font-semibold" style={{ color: 'var(--fgColor-done)' }}>Activity Level (Quantile-Based)</h4>
                        <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-done)' }}>0-4 points</span>
                      </div>
                      <p className="text-sm mb-2" style={{ color: 'var(--fgColor-done)' }}>
                        Based on branch count, commits, issues, and pull requests relative to your repository dataset
                      </p>
                      <ul className="text-sm space-y-1 ml-4" style={{ color: 'var(--fgColor-done)' }}>
                        <li>• High activity (top 25%): <span className="font-medium">+4 points</span></li>
                        <li>• Moderate activity (25-75%): <span className="font-medium">+2 points</span></li>
                        <li>• Low activity (bottom 25%): <span className="font-medium">0 points</span></li>
                      </ul>
                    </div>
                  </div>
                )}

                {/* Complexity Categories */}
                <div>
                  <h3 className="text-base font-semibold mb-3" style={{ color: 'var(--fgColor-default)' }}>Complexity Categories</h3>
                  <div className="space-y-2">
                    <div 
                      className="flex items-center gap-3 p-3 rounded-lg"
                      style={{
                        backgroundColor: 'var(--success-subtle)',
                        border: '1px solid var(--borderColor-success)'
                      }}
                    >
                      <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: 'var(--success-emphasis)' }}></div>
                      <div className="flex-1">
                        <span className="font-semibold" style={{ color: 'var(--fgColor-success)' }}>Simple</span>
                        <span className="text-xs ml-2" style={{ color: 'var(--fgColor-success)' }}>(Score ≤ 5)</span>
                      </div>
                      <span className="text-xs font-medium" style={{ color: 'var(--fgColor-success)' }}>Low effort</span>
                    </div>

                    <div 
                      className="flex items-center gap-3 p-3 rounded-lg"
                      style={{
                        backgroundColor: 'var(--attention-subtle)',
                        border: '1px solid var(--borderColor-attention)'
                      }}
                    >
                      <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: 'var(--attention-emphasis)' }}></div>
                      <div className="flex-1">
                        <span className="font-semibold" style={{ color: 'var(--fgColor-attention)' }}>Medium</span>
                        <span className="text-xs ml-2" style={{ color: 'var(--fgColor-attention)' }}>(Score 6-10)</span>
                      </div>
                      <span className="text-xs font-medium" style={{ color: 'var(--fgColor-attention)' }}>Moderate effort</span>
                    </div>

                    <div 
                      className="flex items-center gap-3 p-3 rounded-lg"
                      style={{
                        backgroundColor: 'var(--attention-subtle)',
                        border: '1px solid var(--borderColor-attention)'
                      }}
                    >
                      <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: 'var(--attention-emphasis)' }}></div>
                      <div className="flex-1">
                        <span className="font-semibold" style={{ color: 'var(--fgColor-attention)' }}>Complex</span>
                        <span className="text-xs ml-2" style={{ color: 'var(--fgColor-attention)' }}>(Score 11-17)</span>
                      </div>
                      <span className="text-xs font-medium" style={{ color: 'var(--fgColor-attention)' }}>High effort</span>
                    </div>

                    <div 
                      className="flex items-center gap-3 p-3 rounded-lg"
                      style={{
                        backgroundColor: 'var(--danger-subtle)',
                        border: '1px solid var(--borderColor-danger)'
                      }}
                    >
                      <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: 'var(--danger-emphasis)' }}></div>
                      <div className="flex-1">
                        <span className="font-semibold" style={{ color: 'var(--fgColor-danger)' }}>Very Complex</span>
                        <span className="text-xs ml-2" style={{ color: 'var(--fgColor-danger)' }}>(Score ≥ 18)</span>
                      </div>
                      <span className="text-xs font-medium" style={{ color: 'var(--fgColor-danger)' }}>Significant effort</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            
            <div 
              className="p-4"
              style={{ 
                flexShrink: 0,
                borderTop: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--bgColor-muted)'
              }}
            >
              <Button variant="primary" onClick={() => setIsOpen(false)} block>
                Got it!
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
