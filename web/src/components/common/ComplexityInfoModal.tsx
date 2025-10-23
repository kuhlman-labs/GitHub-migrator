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
                <p className="text-gray-700">
                  We calculate a complexity score for each repository to estimate migration effort and potential challenges. 
                  The score combines multiple factors, each weighted by their impact on migration complexity.
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

                  {/* Large Files */}
                  <div className="border-l-4 border-red-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Large Files (&gt;100MB)</h4>
                      <span className="text-sm font-semibold text-red-600">Weight: 4 points</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Files larger than 100MB typically require remediation (migration to LFS, removal from history) 
                      before migration can succeed. This is the highest individual feature weight.
                    </p>
                  </div>

                  {/* LFS */}
                  <div className="border-l-4 border-orange-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Git LFS Usage</h4>
                      <span className="text-sm font-semibold text-orange-600">Weight: 2 points</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      LFS requires special handling during migration and must be properly configured on the destination.
                    </p>
                  </div>

                  {/* Submodules */}
                  <div className="border-l-4 border-orange-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Submodules</h4>
                      <span className="text-sm font-semibold text-orange-600">Weight: 2 points</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Submodules add complexity due to dependencies on other repositories that must be migrated first.
                    </p>
                  </div>

                  {/* Packages */}
                  <div className="border-l-4 border-amber-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">GitHub Packages</h4>
                      <span className="text-sm font-semibold text-amber-600">Weight: 3 points</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Packages do not migrate with GEI APIs and require manual migration planning, 
                      making them a significant complexity factor.
                    </p>
                  </div>

                  {/* Branch Protections */}
                  <div className="border-l-4 border-yellow-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Branch Protections</h4>
                      <span className="text-sm font-semibold text-yellow-600">Weight: 1 point</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Branch protection rules require manual reconfiguration after migration.
                    </p>
                  </div>

                  {/* Rulesets */}
                  <div className="border-l-4 border-red-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Rulesets</h4>
                      <span className="text-sm font-semibold text-red-600">Weight: 1 point</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Repository rulesets do not migrate with GEI APIs and must be manually recreated on the destination repository.
                    </p>
                  </div>

                  {/* Advanced Security */}
                  <div className="border-l-4 border-green-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Advanced Security Features</h4>
                      <span className="text-sm font-semibold text-green-600">Weight: 2 points</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Code scanning, Dependabot, or secret scanning require GitHub Advanced Security licenses 
                      and special configuration on the destination.
                    </p>
                  </div>

                  {/* Self-Hosted Runners */}
                  <div className="border-l-4 border-purple-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Self-Hosted Runners</h4>
                      <span className="text-sm font-semibold text-purple-600">Weight: 3 points</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Self-hosted runners require infrastructure setup and configuration on the destination, 
                      making migration more complex.
                    </p>
                  </div>

                  {/* GitHub Apps */}
                  <div className="border-l-4 border-blue-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">GitHub Apps</h4>
                      <span className="text-sm font-semibold text-blue-600">Weight: 2 points</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Installed GitHub Apps need to be reconfigured or reinstalled on the destination repository.
                    </p>
                  </div>

                  {/* Internal Visibility */}
                  <div className="border-l-4 border-yellow-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">Internal Visibility</h4>
                      <span className="text-sm font-semibold text-yellow-600">Weight: 1 point</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      Internal repositories become private when migrating to GitHub.com, requiring permission review.
                    </p>
                  </div>

                  {/* CODEOWNERS */}
                  <div className="border-l-4 border-blue-500 pl-4">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="font-medium text-gray-900">CODEOWNERS</h4>
                      <span className="text-sm font-semibold text-blue-600">Weight: 1 point</span>
                    </div>
                    <p className="text-sm text-gray-600">
                      CODEOWNERS files require verification after migration to ensure team references are correct.
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
                      <span className="text-sm text-green-700 ml-2">(Score ≤ 3)</span>
                    </div>
                    <span className="text-xs text-green-600">Low effort, standard migration</span>
                  </div>
                  <div className="flex items-center gap-3 p-3 bg-yellow-50 rounded-lg">
                    <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                    <div className="flex-1">
                      <span className="font-medium text-yellow-900">Medium</span>
                      <span className="text-sm text-yellow-700 ml-2">(Score 4-6)</span>
                    </div>
                    <span className="text-xs text-yellow-600">Moderate effort, may need planning</span>
                  </div>
                  <div className="flex items-center gap-3 p-3 bg-orange-50 rounded-lg">
                    <div className="w-3 h-3 bg-orange-500 rounded-full"></div>
                    <div className="flex-1">
                      <span className="font-medium text-orange-900">Complex</span>
                      <span className="text-sm text-orange-700 ml-2">(Score 7-9)</span>
                    </div>
                    <span className="text-xs text-orange-600">High effort, requires careful planning</span>
                  </div>
                  <div className="flex items-center gap-3 p-3 bg-red-50 rounded-lg">
                    <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                    <div className="flex-1">
                      <span className="font-medium text-red-900">Very Complex</span>
                      <span className="text-sm text-red-700 ml-2">(Score ≥ 10)</span>
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
                    <li>• Uses LFS → <span className="font-semibold">+2 points</span></li>
                    <li>• Has packages → <span className="font-semibold">+3 points</span></li>
                    <li>• Has branch protections → <span className="font-semibold">+1 point</span></li>
                    <li>• Has rulesets → <span className="font-semibold">+1 point</span></li>
                    <li>• Has Advanced Security (code scanning) → <span className="font-semibold">+2 points</span></li>
                    <li>• Has self-hosted runners → <span className="font-semibold">+3 points</span></li>
                    <li>• Has GitHub Apps installed → <span className="font-semibold">+2 points</span></li>
                    <li>• Internal visibility → <span className="font-semibold">+1 point</span></li>
                    <li>• Has CODEOWNERS → <span className="font-semibold">+1 point</span></li>
                  </ul>
                  <p className="font-bold text-blue-900 pt-2">Total Score: 26 → Very Complex</p>
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

