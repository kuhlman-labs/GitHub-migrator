import { useState } from 'react';
import { api } from '../../services/api';
import { CsvUpload } from './CsvUpload';
import { Link } from 'react-router-dom';

type InputMethod = 'manual' | 'csv';

interface ParsedCsvData {
  repositories: string[];
  mappings?: Record<string, string>;
}

export function SelfServiceMigration() {
  const [inputMethod, setInputMethod] = useState<InputMethod>('manual');
  const [repoNames, setRepoNames] = useState('');
  const [useCustomDestination, setUseCustomDestination] = useState(false);
  const [destinationMappings, setDestinationMappings] = useState('');
  const [dryRun, setDryRun] = useState(false);
  const [loading, setLoading] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [result, setResult] = useState<{
    success?: boolean;
    message?: string;
    batchId?: number;
    batchName?: string;
    totalRepositories?: number;
    newlyDiscovered?: number;
    alreadyExisted?: number;
    discoveryErrors?: string[];
    error?: string;
  } | null>(null);

  const handleCsvDataParsed = (data: ParsedCsvData) => {
    // Convert CSV data to text format for display
    setRepoNames(data.repositories.join('\n'));
    
    if (data.mappings) {
      const mappingText = Object.entries(data.mappings)
        .map(([source, dest]) => `${source} -> ${dest}`)
        .join('\n');
      setDestinationMappings(mappingText);
      setUseCustomDestination(true);
    }

    // Switch to manual view to show the parsed data
    setInputMethod('manual');
  };

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
    const mappings: Record<string, string> = {};
    if (useCustomDestination && destinationMappings.trim()) {
      const mappingLines = destinationMappings
        .split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);

      for (const mapping of mappingLines) {
        const parts = mapping.split('->').map(p => p.trim());
        if (parts.length === 2 && parts[0].includes('/') && parts[1].includes('/')) {
          mappings[parts[0]] = parts[1];
        } else {
          alert(`Invalid mapping format: "${mapping}"\nExpected: source-org/repo -> dest-org/repo`);
          return;
        }
      }
    }

    const hasNewRepos = names.length > 0;
    const confirmMessage = hasNewRepos
      ? `Start ${dryRun ? 'dry run' : 'migration'} for ${names.length} repositories?\n\n` +
        `This will:\n` +
        `1. Check if repositories exist in the database\n` +
        `2. Run full discovery (including cloning) for new repositories\n` +
        `3. Create a batch and ${dryRun ? 'run dry run' : 'start migration'}\n\n` +
        `Note: Discovery may take several minutes per repository.`
      : `Start ${dryRun ? 'dry run' : 'migration'} for ${names.length} repositories?`;

    if (!confirm(confirmMessage)) {
      return;
    }

    setLoading(true);
    setDiscovering(true);
    setResult(null);

    try {
      // Call the new self-service API endpoint
      const response = await api.selfServiceMigration({
        repositories: names,
        mappings: Object.keys(mappings).length > 0 ? mappings : undefined,
        dry_run: dryRun,
      });

      setResult({
        success: true,
        message: response.message,
        batchId: response.batch_id,
        batchName: response.batch_name,
        totalRepositories: response.total_repositories,
        newlyDiscovered: response.newly_discovered,
        alreadyExisted: response.already_existed,
        discoveryErrors: response.discovery_errors,
      });

      // Clear input on success
      setRepoNames('');
      setDestinationMappings('');
      setUseCustomDestination(false);
    } catch (error: any) {
      const err = error as { response?: { data?: { error?: string } } };
      setResult({ 
        success: false,
        error: err.response?.data?.error || 'Failed to start migration' 
      });
    } finally {
      setLoading(false);
      setDiscovering(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <h1 className="text-3xl font-light text-gray-900 mb-4">Self-Service Migration</h1>
      <p className="text-gray-600 mb-8">
        Migrate your repositories independently while maintaining centralized tracking and history.
        You can enter repository names manually or upload a CSV file.
      </p>

      <div className="bg-white rounded-lg shadow-sm p-6">
        {/* Input Method Tabs */}
        <div className="border-b border-gray-200 mb-6">
          <nav className="-mb-px flex space-x-8">
            <button
              onClick={() => setInputMethod('manual')}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm transition-colors
                ${inputMethod === 'manual'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }
              `}
              disabled={loading}
            >
              Manual Entry
            </button>
            <button
              onClick={() => setInputMethod('csv')}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm transition-colors
                ${inputMethod === 'csv'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }
              `}
              disabled={loading}
            >
              CSV Upload
            </button>
          </nav>
        </div>

        {/* CSV Upload View */}
        {inputMethod === 'csv' && (
          <div className="mb-6">
            <CsvUpload onDataParsed={handleCsvDataParsed} disabled={loading} />
          </div>
        )}

        {/* Manual Entry View */}
        {inputMethod === 'manual' && (
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

            {/* Progress Indicator */}
            {loading && (
              <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
                <div className="flex items-center">
                  <svg
                    className="animate-spin h-5 w-5 text-blue-600 mr-3"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                  <div>
                    <div className="text-sm font-medium text-blue-900">
                      {discovering ? 'Running discovery and creating batch...' : 'Processing...'}
                    </div>
                    <div className="text-sm text-blue-700 mt-1">
                      {discovering && 'This may take several minutes for new repositories. Please be patient.'}
                    </div>
                  </div>
                </div>
              </div>
            )}

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
                      <div className="font-medium mb-2">✓ Migration Started Successfully!</div>
                      <div className="mb-3">{result.message}</div>
                      
                      {/* Batch Details */}
                      <div className="bg-white rounded border border-green-200 p-3 mb-3 space-y-2">
                        <div className="flex justify-between text-sm">
                          <span className="font-medium">Batch Name:</span>
                          <span className="font-mono text-xs">{result.batchName}</span>
                        </div>
                        <div className="flex justify-between text-sm">
                          <span className="font-medium">Total Repositories:</span>
                          <span>{result.totalRepositories}</span>
                        </div>
                        {result.newlyDiscovered !== undefined && result.newlyDiscovered > 0 && (
                          <div className="flex justify-between text-sm">
                            <span className="font-medium">Newly Discovered:</span>
                            <span className="text-blue-600">{result.newlyDiscovered}</span>
                          </div>
                        )}
                        {result.alreadyExisted !== undefined && result.alreadyExisted > 0 && (
                          <div className="flex justify-between text-sm">
                            <span className="font-medium">Already in Database:</span>
                            <span className="text-gray-600">{result.alreadyExisted}</span>
                          </div>
                        )}
                      </div>

                      {/* Discovery Errors */}
                      {result.discoveryErrors && result.discoveryErrors.length > 0 && (
                        <div className="bg-yellow-50 border border-yellow-200 rounded p-3 mb-3">
                          <div className="font-medium text-yellow-800 mb-2">
                            ⚠ Some repositories could not be discovered:
                          </div>
                          <ul className="text-sm text-yellow-700 space-y-1 ml-4 list-disc">
                            {result.discoveryErrors.map((error, index) => (
                              <li key={index}>{error}</li>
                            ))}
                          </ul>
                        </div>
                      )}

                      {/* Action Buttons */}
                      <div className="flex gap-3">
                        <Link
                          to="/"
                          className="flex-1 text-center px-4 py-2 bg-green-600 text-white rounded-md text-sm font-medium hover:bg-green-700 transition-colors"
                        >
                          View on Dashboard
                        </Link>
                        <Link
                          to="/batches"
                          className="flex-1 text-center px-4 py-2 border border-green-600 text-green-700 rounded-md text-sm font-medium hover:bg-green-50 transition-colors"
                        >
                          View Batch Management
                        </Link>
                      </div>
                    </>
                  ) : (
                    <>
                      <div className="font-medium mb-1">✗ Error</div>
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
                className="flex-1 px-4 py-3 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading 
                  ? 'Processing...' 
                  : dryRun 
                    ? 'Start Dry Run' 
                    : 'Start Migration'}
              </button>
              {repoNames && (
                <button
                  type="button"
                  onClick={() => {
                    setRepoNames('');
                    setDestinationMappings('');
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
        )}

        {/* Help Section */}
        <div className="mt-8 pt-6 border-t border-gray-200">
          <h3 className="text-sm font-medium text-gray-900 mb-3">How It Works</h3>
          <ul className="space-y-2 text-sm text-gray-600">
            <li>• <strong>Manual Entry:</strong> Enter repository names directly (one per line or comma-separated)</li>
            <li>• <strong>CSV Upload:</strong> Upload a CSV file with repository names and optional destination mappings</li>
            <li>• <strong>Discovery:</strong> New repositories will be discovered automatically (includes cloning and analysis)</li>
            <li>• <strong>Batch Creation:</strong> A batch will be created with a timestamp-based name</li>
            <li>• <strong>Execution:</strong> Migration (or dry run) will start immediately after discovery</li>
            <li>• <strong>Tracking:</strong> Monitor progress on the Dashboard or Batch Details page</li>
          </ul>
          
          <h3 className="text-sm font-medium text-gray-900 mb-3 mt-6">Tips</h3>
          <ul className="space-y-2 text-sm text-gray-600">
            <li>• Repository names must be in "organization/repository" format</li>
            <li>• <strong>Default behavior:</strong> Repositories migrate to the same organization name</li>
            <li>• <strong>Custom destinations:</strong> Specify different destination organizations if needed</li>
            <li>• Use "Dry Run" to test without making changes</li>
            <li>• Discovery may take several minutes per repository (includes full cloning)</li>
            <li>• All migrations are tracked in history for business records</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
