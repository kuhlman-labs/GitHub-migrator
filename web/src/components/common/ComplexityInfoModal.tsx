import { useState } from 'react';
import { Dialog, Button, IconButton } from '@primer/react';
import { InfoIcon, XIcon } from '@primer/octicons-react';

interface ComplexityInfoModalProps {
  source?: 'github' | 'azuredevops' | 'all';
}

export function ComplexityInfoModal({ source = 'all' }: ComplexityInfoModalProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="inline-flex items-center gap-1 text-blue-600 hover:text-blue-800 text-sm font-medium"
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
            className="absolute inset-0 bg-black/60" 
            onClick={() => setIsOpen(false)}
          />
          
          {/* Dialog */}
          <div
            className="relative bg-white rounded-lg shadow-2xl border border-gray-200"
            style={{
              width: '90%',
              maxWidth: '800px',
              maxHeight: '90vh',
              display: 'flex',
              flexDirection: 'column',
              zIndex: 10000,
            }}
          >
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold text-gray-900">
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
                  <h3 className="text-base font-semibold mb-2 text-gh-text-primary">Overview</h3>
                  <p className="text-sm text-gh-text-secondary mb-2">
                    {source === 'azuredevops' 
                      ? 'We calculate an Azure DevOps-specific complexity score to estimate migration effort from ADO to GitHub.'
                      : (source === 'github' || source === 'ghes')
                      ? 'We calculate a GitHub-specific complexity score to estimate migration effort between GitHub instances.'
                      : 'We calculate source-specific complexity scores to estimate migration effort and potential challenges.'}
                  </p>
                  <p className="text-xs text-gh-text-muted italic">
                    Activity levels are calculated using quantiles relative to your repository dataset.
                  </p>
                </div>

                {/* Scoring Factors - GitHub Specific */}
                {(source === 'github' || source === 'ghes') && (
                  <div>
                    <h3 className="text-base font-semibold mb-3 text-gh-text-primary">Scoring Factors</h3>
                    
                    {/* Repository Size */}
                    <div className="mb-4 p-4 bg-blue-50 rounded-lg border border-blue-200">
                      <div className="flex justify-between items-start mb-2">
                        <h4 className="font-semibold text-blue-900">Repository Size</h4>
                        <span className="text-sm font-semibold text-blue-600">Weight: 3 (max 9 points)</span>
                      </div>
                      <p className="text-sm text-blue-800 mb-2">
                        Larger repositories take longer to migrate and have higher resource requirements.
                      </p>
                      <ul className="text-sm text-blue-900 space-y-1 ml-4">
                        <li>• &lt;100MB: <span className="font-medium">0 points</span></li>
                        <li>• 100MB - 1GB: <span className="font-medium">3 points</span></li>
                        <li>• 1GB - 5GB: <span className="font-medium">6 points</span></li>
                        <li>• &gt;5GB: <span className="font-medium">9 points</span></li>
                      </ul>
                    </div>

                    {/* High Impact Features */}
                    <div className="mb-3">
                      <h4 className="font-semibold text-red-800 mb-2">High Impact (3-4 points each)</h4>
                      <p className="text-sm text-gray-600 mb-3">Features requiring significant remediation effort</p>
                      
                      <div className="space-y-2">
                        <div className="p-3 bg-red-50 rounded border border-red-200">
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium text-red-900">Large Files (&gt;100MB)</span>
                            <span className="text-sm font-semibold text-red-600">4 points</span>
                          </div>
                          <p className="text-sm text-red-800">Must be remediated before migration (migrate to LFS, remove from history)</p>
                        </div>

                        <div className="p-3 bg-red-50 rounded border border-red-200">
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium text-red-900">Environments</span>
                            <span className="text-sm font-semibold text-red-600">3 points</span>
                          </div>
                          <p className="text-sm text-red-800">Don't migrate. Manual recreation of all configs and protection rules required</p>
                        </div>

                        <div className="p-3 bg-red-50 rounded border border-red-200">
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium text-red-900">Secrets</span>
                            <span className="text-sm font-semibold text-red-600">3 points</span>
                          </div>
                          <p className="text-sm text-red-800">Don't migrate. Manual recreation required with high security sensitivity</p>
                        </div>

                        <div className="p-3 bg-red-50 rounded border border-red-200">
                          <div className="flex justify-between items-start mb-1">
                            <span className="font-medium text-red-900">GitHub Packages</span>
                            <span className="text-sm font-semibold text-red-600">3 points</span>
                          </div>
                          <p className="text-sm text-red-800">Don't migrate with GEI. Manual migration planning required</p>
                        </div>
                      </div>
                    </div>

                    {/* Moderate Impact Features */}
                    <div className="mb-3">
                      <h4 className="font-semibold text-orange-800 mb-2">Moderate Impact (2 points each)</h4>
                      <p className="text-sm text-gray-600 mb-3">Features requiring manual intervention</p>
                      
                      <div className="space-y-2">
                        <div className="p-2 bg-orange-50 rounded border border-orange-200">
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-orange-900 text-sm">Variables</span>
                            <span className="text-xs font-semibold text-orange-600">2 points</span>
                          </div>
                        </div>
                        <div className="p-2 bg-orange-50 rounded border border-orange-200">
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-orange-900 text-sm">Discussions</span>
                            <span className="text-xs font-semibold text-orange-600">2 points</span>
                          </div>
                        </div>
                        <div className="p-2 bg-orange-50 rounded border border-orange-200">
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-orange-900 text-sm">Git LFS</span>
                            <span className="text-xs font-semibold text-orange-600">2 points</span>
                          </div>
                        </div>
                        <div className="p-2 bg-orange-50 rounded border border-orange-200">
                          <div className="flex justify-between items-center">
                            <span className="font-medium text-orange-900 text-sm">Submodules</span>
                            <span className="text-xs font-semibold text-orange-600">2 points</span>
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Low Impact Features */}
                    <div className="mb-3">
                      <h4 className="font-semibold text-yellow-800 mb-2">Low Impact (1 point each)</h4>
                      <p className="text-sm text-gray-600 mb-3">Features requiring straightforward manual steps</p>
                      
                      <div className="grid grid-cols-2 gap-2">
                        <div className="p-2 bg-yellow-50 rounded border border-yellow-200 text-sm text-yellow-900">Advanced Security</div>
                        <div className="p-2 bg-yellow-50 rounded border border-yellow-200 text-sm text-yellow-900">Webhooks</div>
                        <div className="p-2 bg-yellow-50 rounded border border-yellow-200 text-sm text-yellow-900">Branch Protections</div>
                        <div className="p-2 bg-yellow-50 rounded border border-yellow-200 text-sm text-yellow-900">Rulesets</div>
                        <div className="p-2 bg-yellow-50 rounded border border-yellow-200 text-sm text-yellow-900">Public/Internal Repos</div>
                        <div className="p-2 bg-yellow-50 rounded border border-yellow-200 text-sm text-yellow-900">CODEOWNERS</div>
                      </div>
                    </div>

                    {/* Activity Level */}
                    <div className="p-4 bg-purple-50 rounded-lg border border-purple-200">
                      <div className="flex justify-between items-start mb-2">
                        <h4 className="font-semibold text-purple-900">Activity Level (Quantile-Based)</h4>
                        <span className="text-sm font-semibold text-purple-600">0-4 points</span>
                      </div>
                      <p className="text-sm text-purple-800 mb-2">
                        Based on branch count, commits, issues, and pull requests relative to your repository dataset
                      </p>
                      <ul className="text-sm text-purple-900 space-y-1 ml-4">
                        <li>• High activity (top 25%): <span className="font-medium">+4 points</span></li>
                        <li>• Moderate activity (25-75%): <span className="font-medium">+2 points</span></li>
                        <li>• Low activity (bottom 25%): <span className="font-medium">0 points</span></li>
                      </ul>
                    </div>
                  </div>
                )}

                {/* Complexity Categories */}
                <div>
                  <h3 className="text-base font-semibold mb-3 text-gh-text-primary">Complexity Categories</h3>
                  <div className="space-y-2">
                    <div className="flex items-center gap-3 p-3 bg-green-50 rounded-lg border border-green-200">
                      <div className="w-3 h-3 bg-green-500 rounded-full flex-shrink-0"></div>
                      <div className="flex-1">
                        <span className="font-semibold text-green-900">Simple</span>
                        <span className="text-xs text-green-700 ml-2">(Score ≤ 5)</span>
                      </div>
                      <span className="text-xs text-green-600 font-medium">Low effort</span>
                    </div>

                    <div className="flex items-center gap-3 p-3 bg-yellow-50 rounded-lg border border-yellow-200">
                      <div className="w-3 h-3 bg-yellow-500 rounded-full flex-shrink-0"></div>
                      <div className="flex-1">
                        <span className="font-semibold text-yellow-900">Medium</span>
                        <span className="text-xs text-yellow-700 ml-2">(Score 6-10)</span>
                      </div>
                      <span className="text-xs text-yellow-600 font-medium">Moderate effort</span>
                    </div>

                    <div className="flex items-center gap-3 p-3 bg-orange-50 rounded-lg border border-orange-200">
                      <div className="w-3 h-3 bg-orange-500 rounded-full flex-shrink-0"></div>
                      <div className="flex-1">
                        <span className="font-semibold text-orange-900">Complex</span>
                        <span className="text-xs text-orange-700 ml-2">(Score 11-17)</span>
                      </div>
                      <span className="text-xs text-orange-600 font-medium">High effort</span>
                    </div>

                    <div className="flex items-center gap-3 p-3 bg-red-50 rounded-lg border border-red-200">
                      <div className="w-3 h-3 bg-red-500 rounded-full flex-shrink-0"></div>
                      <div className="flex-1">
                        <span className="font-semibold text-red-900">Very Complex</span>
                        <span className="text-xs text-red-700 ml-2">(Score ≥ 18)</span>
                      </div>
                      <span className="text-xs text-red-600 font-medium">Significant effort</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            
            <div 
              className="border-t border-gray-200 bg-gray-50 p-4"
              style={{ flexShrink: 0 }}
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
