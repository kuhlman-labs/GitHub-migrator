import { useState } from 'react';
import { api } from '../../services/api';

export function SelfServiceMigration() {
  const [repoNames, setRepoNames] = useState('');
  const [useCustomDestination, setUseCustomDestination] = useState(false);
  const [destinationMappings, setDestinationMappings] = useState('');
  const [dryRun, setDryRun] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{
    success?: boolean;
    message?: string;
    count?: number;
    error?: string;
  } | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Parse repository names (one per line or comma-separated)
    const names = repoNames
      .split(/[\n,]/)
      .map(name => name.trim())
      .filter(name => name.length > 0);

    if (names.length === 0) {
      alert('Please enter at least one repository name');
      return;
    }

    // Validate format (org/repo)
    const invalidNames = names.filter(name => !name.includes('/'));
    if (invalidNames.length > 0) {
      alert(`Invalid repository names (must be in "org/repo" format):\n${invalidNames.join('\n')}`);
      return;
    }

    // Parse destination mappings if provided
    const destinationMap = new Map<string, string>();
    if (useCustomDestination && destinationMappings.trim()) {
      const mappings = destinationMappings
        .split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);

      for (const mapping of mappings) {
        const parts = mapping.split('->').map(p => p.trim());
        if (parts.length === 2 && parts[0].includes('/') && parts[1].includes('/')) {
          destinationMap.set(parts[0], parts[1]);
        } else {
          alert(`Invalid mapping format: "${mapping}"\nExpected: source-org/repo -> dest-org/repo`);
          return;
        }
      }
    }

    if (!confirm(`Start ${dryRun ? 'dry run' : 'migration'} for ${names.length} repositories?`)) {
      return;
    }

    setLoading(true);
    setResult(null);

    try {
      // First, update destination_full_name for repositories with custom destinations
      if (destinationMap.size > 0) {
        for (const [sourceName, destName] of destinationMap.entries()) {
          if (names.includes(sourceName)) {
            try {
              await api.updateRepository(sourceName, {
                destination_full_name: destName,
              });
            } catch (error) {
              console.error(`Failed to update destination for ${sourceName}:`, error);
            }
          }
        }
      }

      // Then start the migration
      const response = await api.startMigration({
        full_names: names,
        dry_run: dryRun,
        priority: 0,
      });

      setResult({
        success: true,
        message: response.message || 'Migration started successfully',
        count: response.count || names.length,
      });
      setRepoNames(''); // Clear input on success
      setDestinationMappings('');
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      setResult({ 
        success: false,
        error: err.response?.data?.error || 'Failed to start migration' 
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-3xl font-light text-gray-900 mb-4">Self-Service Migration</h1>
      <p className="text-gray-600 mb-8">
        Enter repository names (in "org/repo" format) to start migration. You can enter multiple repositories,
        one per line or separated by commas.
      </p>

      <div className="bg-white rounded-lg shadow-sm p-6">
        <form onSubmit={handleSubmit}>
          {/* Repository Names Input */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Repository Names
            </label>
            <textarea
              value={repoNames}
              onChange={(e) => setRepoNames(e.target.value)}
              placeholder="org/repo1&#10;org/repo2&#10;org/repo3"
              rows={8}
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
              disabled={loading}
            />
            <p className="mt-2 text-sm text-gray-500">
              Example: myorg/my-repo or myorg/repo1, myorg/repo2
            </p>
          </div>

          {/* Custom Destination Toggle */}
          <div className="mb-6">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={useCustomDestination}
                onChange={(e) => setUseCustomDestination(e.target.checked)}
                disabled={loading}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <span className="ml-2 text-sm text-gray-700">
                Use custom destination organizations (default: migrates to same org name)
              </span>
            </label>
          </div>

          {/* Destination Mappings */}
          {useCustomDestination && (
            <div className="mb-6">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Destination Mappings
              </label>
              <textarea
                value={destinationMappings}
                onChange={(e) => setDestinationMappings(e.target.value)}
                placeholder="source-org/repo1 -> dest-org/repo1&#10;source-org/repo2 -> other-org/repo2"
                rows={6}
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
                disabled={loading}
              />
              <p className="mt-2 text-sm text-gray-500">
                Format: <code className="bg-gray-100 px-1 rounded">source-org/repo -&gt; dest-org/repo</code> (one per line)
              </p>
              <p className="mt-1 text-sm text-gray-500">
                Repositories not listed here will migrate to the same organization name as the source.
              </p>
            </div>
          )}

          {/* Dry Run Checkbox */}
          <div className="mb-6">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={dryRun}
                onChange={(e) => setDryRun(e.target.checked)}
                disabled={loading}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <span className="ml-2 text-sm text-gray-700">
                Dry run (test migration without making changes)
              </span>
            </label>
          </div>

          {/* Result Message */}
          {result && (
            <div className={`mb-6 p-4 rounded-lg ${
              result.success 
                ? 'bg-green-50 border border-green-200' 
                : 'bg-red-50 border border-red-200'
            }`}>
              <div className={`text-sm ${result.success ? 'text-green-800' : 'text-red-800'}`}>
                {result.success ? (
                  <>
                    <div className="font-medium mb-1">Success!</div>
                    <div>{result.message}</div>
                    {result.count && (
                      <div className="mt-2">
                        {result.count} {result.count === 1 ? 'repository' : 'repositories'} queued for {dryRun ? 'dry run' : 'migration'}
                      </div>
                    )}
                    <div className="mt-3">
                      <a href="/" className="text-green-700 underline hover:text-green-900">
                        View on Dashboard →
                      </a>
                    </div>
                  </>
                ) : (
                  <>
                    <div className="font-medium mb-1">Error</div>
                    <div>{result.error}</div>
                  </>
                )}
              </div>
            </div>
          )}

          {/* Submit Button */}
          <div className="flex gap-3">
            <button
              type="submit"
              disabled={loading || !repoNames.trim()}
              className="flex-1 px-6 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? 'Processing...' : dryRun ? 'Start Dry Run' : 'Start Migration'}
            </button>
            {repoNames && (
              <button
                type="button"
                onClick={() => {
                  setRepoNames('');
                  setResult(null);
                }}
                disabled={loading}
                className="px-6 py-3 border border-gray-300 text-gray-700 rounded-lg font-medium hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Clear
              </button>
            )}
          </div>
        </form>

        {/* Help Section */}
        <div className="mt-8 pt-6 border-t border-gray-200">
          <h3 className="text-sm font-medium text-gray-900 mb-3">Tips</h3>
          <ul className="space-y-2 text-sm text-gray-600">
            <li>• Repository names must be in "organization/repository" format</li>
            <li>• Separate multiple repositories with new lines or commas</li>
            <li>• <strong>Default behavior:</strong> Repositories migrate to the same organization name (e.g., <code className="bg-gray-100 px-1 rounded">acme-corp/api</code> → <code className="bg-gray-100 px-1 rounded">acme-corp/api</code> on destination)</li>
            <li>• <strong>Custom destinations:</strong> Check the box above to specify different destination organizations for specific repositories</li>
            <li>• Use "Dry Run" to test the migration without making any changes</li>
            <li>• You can monitor progress on the Dashboard after submission</li>
          </ul>
        </div>
      </div>
    </div>
  );
}

