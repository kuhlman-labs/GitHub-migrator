import { useState } from 'react';

export function ComplexityInfoModal() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="inline-flex items-center gap-1 text-blue-600 hover:text-blue-800 text-sm font-medium"
        title="Learn how complexity is calculated"
      >
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
        </svg>
        How is complexity calculated?
      </button>

      {isOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4">
              <div className="flex justify-between items-start">
                <h2 className="text-xl font-semibold text-gray-900">
                  Repository Complexity Scoring
                </h2>
                <button
                  onClick={() => setIsOpen(false)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            <div className="px-6 py-4 space-y-6">
              {/* Overview */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Overview</h3>
                <p className="text-gray-700 mb-2">
                  We calculate a GitHub-specific complexity score for each repository to estimate migration effort and potential challenges. 
                  The score combines multiple factors, each weighted by their remediation difficulty based on GitHub's migration documentation.
                </p>
                <p className="text-sm text-gray-600 italic">
                  Activity levels are calculated using quantiles relative to your repository dataset, making the scoring adaptive to your specific environment.
                </p>
              </div>

              {/* Scoring Factors */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-3">Scoring Factors</h3>
                <div className="space-y-4">
                  {/* Repository Size */}
                  <div className="border-l-4 border-blue-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Repository Size</h4>
                      <span className="text-sm font-semibold text-blue-600">Weight: 3 (max 9 points)</span>
                    </div>
                    <p className="text-sm text-gray-600 mb-2">
                      Larger repositories take longer to migrate and have higher resource requirements.
                    </p>
                    <ul className="text-sm text-gray-700 space-y-1">
                      <li>• &lt;100MB: <span className="font-medium">0 points</span></li>
                      <li>• 100MB - 1GB: <span className="font-medium">3 points</span></li>
                      <li>• 1GB - 5GB: <span className="font-medium">6 points</span></li>
                      <li>• &gt;5GB: <span className="font-medium">9 points</span></li>
                    </ul>
                  </div>

                  {/* High Impact Features */}
                  <div className="mb-3">
                    <h4 className="font-semibold text-gray-900 mb-2">High Impact (3-4 points)</h4>
                    <p className="text-sm text-gray-600 mb-3">Features requiring significant remediation effort before or after migration</p>
                    
                    <div className="space-y-3">
                      <div className="border-l-4 border-red-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Large Files (&gt;100MB)</h5>
                          <span className="text-sm font-semibold text-red-600">4 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Must be remediated before migration (migrate to LFS, remove from history). Highest weight.
                        </p>
                      </div>

                      <div className="border-l-4 border-red-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Environments</h5>
                          <span className="text-sm font-semibold text-red-600">3 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Don't migrate. Manual recreation of all configs, protection rules, and deployment branches required.
                        </p>
                      </div>

                      <div className="border-l-4 border-red-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Secrets</h5>
                          <span className="text-sm font-semibold text-red-600">3 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Don't migrate. Manual recreation required with high security sensitivity, affects CI/CD.
                        </p>
                      </div>

                      <div className="border-l-4 border-red-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">GitHub Packages</h5>
                          <span className="text-sm font-semibold text-red-600">3 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Don't migrate with GEI. Manual migration planning and execution required.
                        </p>
                      </div>

                      <div className="border-l-4 border-red-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Self-Hosted Runners</h5>
                          <span className="text-sm font-semibold text-red-600">3 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Infrastructure reconfiguration and setup required on destination.
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Moderate Impact Features */}
                  <div className="mb-3">
                    <h4 className="font-semibold text-gray-900 mb-2">Moderate Impact (2 points)</h4>
                    <p className="text-sm text-gray-600 mb-3">Features requiring manual intervention but less complexity</p>
                    
                    <div className="space-y-3">
                      <div className="border-l-4 border-orange-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Variables</h5>
                          <span className="text-sm font-semibold text-orange-600">2 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Don't migrate. Manual recreation required, less sensitive than secrets.
                        </p>
                      </div>

                      <div className="border-l-4 border-orange-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Discussions</h5>
                          <span className="text-sm font-semibold text-orange-600">2 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Don't migrate. Community impact, manual recreation loses history.
                        </p>
                      </div>

                      <div className="border-l-4 border-orange-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Releases</h5>
                          <span className="text-sm font-semibold text-orange-600">2 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Only migrate on GHES 3.5.0+. May require manual migration on older versions.
                        </p>
                      </div>

                      <div className="border-l-4 border-orange-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Git LFS</h5>
                          <span className="text-sm font-semibold text-orange-600">2 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Special handling during migration, proper configuration required on destination.
                        </p>
                      </div>

                      <div className="border-l-4 border-orange-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Submodules</h5>
                          <span className="text-sm font-semibold text-orange-600">2 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Dependencies on other repositories that must be migrated first.
                        </p>
                      </div>

                      <div className="border-l-4 border-orange-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">GitHub Apps</h5>
                          <span className="text-sm font-semibold text-orange-600">2 points</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Reconfiguration or reinstallation required on destination.
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Low Impact Features */}
                  <div className="mb-3">
                    <h4 className="font-semibold text-gray-900 mb-2">Low Impact (1 point)</h4>
                    <p className="text-sm text-gray-600 mb-3">Features requiring straightforward manual steps</p>
                    
                    <div className="space-y-3">
                      <div className="border-l-4 border-yellow-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Advanced Security (GHAS)</h5>
                          <span className="text-sm font-semibold text-yellow-600">1 point</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Code scanning, Dependabot, secret scanning. Simple toggles to re-enable.
                        </p>
                      </div>

                      <div className="border-l-4 border-yellow-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Webhooks</h5>
                          <span className="text-sm font-semibold text-yellow-600">1 point</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Must re-enable after migration. Straightforward but critical for integrations.
                        </p>
                      </div>

                      <div className="border-l-4 border-yellow-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Tag Protections</h5>
                          <span className="text-sm font-semibold text-yellow-600">1 point</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Don't migrate. Manual configuration required, similar to branch protections.
                        </p>
                      </div>

                      <div className="border-l-4 border-yellow-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Branch Protections</h5>
                          <span className="text-sm font-semibold text-yellow-600">1 point</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Migrate but certain rules don't (e.g., bypass actors, force push settings).
                        </p>
                      </div>

                      <div className="border-l-4 border-yellow-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Rulesets</h5>
                          <span className="text-sm font-semibold text-yellow-600">1 point</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Don't migrate. Manual recreation on destination repository required.
                        </p>
                      </div>

                      <div className="border-l-4 border-yellow-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">Public/Internal Visibility</h5>
                          <span className="text-sm font-semibold text-yellow-600">1 point each</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          May require visibility transformation (e.g., EMU doesn't support public repos).
                        </p>
                      </div>

                      <div className="border-l-4 border-yellow-500 pl-4">
                        <div className="flex justify-between items-start mb-1">
                          <h5 className="font-medium text-gray-900">CODEOWNERS</h5>
                          <span className="text-sm font-semibold text-yellow-600">1 point</span>
                        </div>
                        <p className="text-sm text-gray-600">
                          Migrates but verification required to ensure team references are correct.
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Activity-Based Scoring */}
                  <div className="border-l-4 border-purple-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Activity Level (Quantile-Based)</h4>
                      <span className="text-sm font-semibold text-purple-600">0-4 points</span>
                    </div>
                    <p className="text-sm text-gray-600 mb-2">
                      Activity level is calculated using quantiles relative to all your repositories. High-activity repos require significantly more planning, coordination, and stakeholder communication.
                    </p>
                    <ul className="text-sm text-gray-700 space-y-1">
                      <li>• High activity (top 25%): <span className="font-medium">+4 points</span> - Many users, extensive coordination needed</li>
                      <li>• Moderate activity (25-75%): <span className="font-medium">+2 points</span> - Some coordination needed</li>
                      <li>• Low activity (bottom 25%): <span className="font-medium">0 points</span> - Few users, minimal coordination</li>
                    </ul>
                    <p className="text-xs text-gray-500 mt-2 italic">
                      Combines: branch count, commit count, issue count, and pull request count
                    </p>
                  </div>
                </div>
              </div>

              {/* Categories */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-3">Complexity Categories</h3>
                <div className="space-y-2">
                  <div className="flex items-center gap-3 p-3 bg-green-50 rounded-lg">
                    <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                    <div className="flex-1">
                      <span className="font-medium text-green-900">Simple</span>
                      <span className="text-sm text-green-700 ml-2">(Score ≤ 5)</span>
                    </div>
                    <span className="text-xs text-green-600">Low effort, standard migration</span>
                  </div>
                  <div className="flex items-center gap-3 p-3 bg-yellow-50 rounded-lg">
                    <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                    <div className="flex-1">
                      <span className="font-medium text-yellow-900">Medium</span>
                      <span className="text-sm text-yellow-700 ml-2">(Score 6-10)</span>
                    </div>
                    <span className="text-xs text-yellow-600">Moderate effort, may need planning</span>
                  </div>
                  <div className="flex items-center gap-3 p-3 bg-orange-50 rounded-lg">
                    <div className="w-3 h-3 bg-orange-500 rounded-full"></div>
                    <div className="flex-1">
                      <span className="font-medium text-orange-900">Complex</span>
                      <span className="text-sm text-orange-700 ml-2">(Score 11-17)</span>
                    </div>
                    <span className="text-xs text-orange-600">High effort, requires careful planning</span>
                  </div>
                  <div className="flex items-center gap-3 p-3 bg-red-50 rounded-lg">
                    <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                    <div className="flex-1">
                      <span className="font-medium text-red-900">Very Complex</span>
                      <span className="text-sm text-red-700 ml-2">(Score ≥ 18)</span>
                    </div>
                    <span className="text-xs text-red-600">Significant effort, likely needs remediation</span>
                  </div>
                </div>
              </div>

              {/* Example */}
              <div className="bg-blue-50 rounded-lg p-4">
                <h3 className="text-lg font-medium text-blue-900 mb-2">Example Calculation</h3>
                <div className="text-sm text-blue-800 space-y-1">
                  <p className="font-medium">Repository with:</p>
                  <ul className="ml-4 space-y-1">
                    <li>• Size: 2.5 GB → <span className="font-semibold">6 points</span></li>
                    <li>• Has large files → <span className="font-semibold">+4 points</span></li>
                    <li>• Has 3 environments → <span className="font-semibold">+3 points</span></li>
                    <li>• Has 15 secrets → <span className="font-semibold">+3 points</span></li>
                    <li>• Has 8 variables → <span className="font-semibold">+2 points</span></li>
                    <li>• Has discussions → <span className="font-semibold">+2 points</span></li>
                    <li>• Uses LFS → <span className="font-semibold">+2 points</span></li>
                    <li>• Has Advanced Security → <span className="font-semibold">+1 point</span></li>
                    <li>• Has 2 webhooks → <span className="font-semibold">+1 point</span></li>
                    <li>• Has branch protections → <span className="font-semibold">+1 point</span></li>
                    <li>• Has rulesets → <span className="font-semibold">+1 point</span></li>
                    <li>• High activity (top 25%) → <span className="font-semibold">+4 points</span></li>
                  </ul>
                  <p className="font-bold text-blue-900 pt-2">Total Score: 31 → Very Complex</p>
                  <p className="text-xs text-blue-700 pt-1 italic">
                    This repository requires significant planning for environments, secrets, and variables that don't migrate.
                  </p>
                </div>
              </div>
            </div>

            <div className="sticky bottom-0 bg-gray-50 px-6 py-4 border-t border-gray-200">
              <button
                onClick={() => setIsOpen(false)}
                className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium"
              >
                Got it!
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

