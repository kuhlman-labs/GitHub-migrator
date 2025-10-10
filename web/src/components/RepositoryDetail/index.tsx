import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api } from '../../services/api';
import type { Repository, MigrationHistory, MigrationLog, Batch } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { ProfileCard } from '../common/ProfileCard';
import { ProfileItem } from '../common/ProfileItem';
import { formatBytes, formatDate } from '../../utils/format';

export function RepositoryDetail() {
  const { fullName } = useParams<{ fullName: string }>();
  const [repository, setRepository] = useState<Repository | null>(null);
  const [history, setHistory] = useState<MigrationHistory[]>([]);
  const [logs, setLogs] = useState<MigrationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [logsLoading, setLogsLoading] = useState(false);
  const [migrating, setMigrating] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'history' | 'logs'>('overview');
  
  // Batch assignment state
  const [batches, setBatches] = useState<Batch[]>([]);
  const [selectedBatchId, setSelectedBatchId] = useState<number | null>(null);
  const [assigningBatch, setAssigningBatch] = useState(false);
  
  // Destination configuration
  const [editingDestination, setEditingDestination] = useState(false);
  const [destinationFullName, setDestinationFullName] = useState<string>('');
  const [savingDestination, setSavingDestination] = useState(false);
  
  // Log filters
  const [logLevel, setLogLevel] = useState<string>('');
  const [logPhase, setLogPhase] = useState<string>('');
  const [logSearch, setLogSearch] = useState<string>('');

  useEffect(() => {
    loadRepository();
    loadBatches();
    // Poll for status updates every 10 seconds
    const interval = setInterval(loadRepository, 10000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fullName]);

  const loadBatches = async () => {
    try {
      const allBatches = await api.listBatches();
      // Only show ready batches for assignment
      const readyBatches = allBatches.filter(b => b.status === 'ready');
      setBatches(readyBatches);
    } catch (error) {
      console.error('Failed to load batches:', error);
    }
  };

  const loadRepository = async () => {
    if (!fullName) return;
    
    try {
      const response = await api.getRepository(decodeURIComponent(fullName));
      setRepository(response.repository);
      setHistory(response.history || []);
      
      // Set destination full name (defaults to source full name if not set)
      setDestinationFullName(response.repository.destination_full_name || response.repository.full_name);
      
      // Load logs if tab is active
      if (activeTab === 'logs') {
        await loadLogs(response.repository.id);
      }
    } catch (error) {
      console.error('Failed to load repository:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadLogs = async (repoId?: number) => {
    const id = repoId || repository?.id;
    if (!id) return;
    
    setLogsLoading(true);
    try {
      const response = await api.getMigrationLogs(id, {
        level: logLevel || undefined,
        phase: logPhase || undefined,
        limit: 500,
      });
      setLogs(response.logs || []);
    } catch (error) {
      console.error('Failed to load logs:', error);
    } finally {
      setLogsLoading(false);
    }
  };

  // Load logs when filters change
  useEffect(() => {
    if (activeTab === 'logs' && repository) {
      loadLogs();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [logLevel, logPhase, activeTab]);

  const handleSaveDestination = async () => {
    if (!repository || !fullName) return;

    // Validate format
    if (!destinationFullName.includes('/')) {
      alert('Destination must be in "organization/repository" format');
      return;
    }

    setSavingDestination(true);
    try {
      await api.updateRepository(decodeURIComponent(fullName), {
        destination_full_name: destinationFullName,
      });
      
      setEditingDestination(false);
      await loadRepository();
    } catch (error) {
      console.error('Failed to save destination:', error);
      alert('Failed to save destination. Please try again.');
    } finally {
      setSavingDestination(false);
    }
  };

  const handleStartMigration = async (dryRun: boolean = false) => {
    if (!repository || migrating) return;

    // Save destination first if it was changed
    if (editingDestination && destinationFullName !== repository.destination_full_name) {
      await handleSaveDestination();
    }

    setMigrating(true);
    try {
      await api.startMigration({
        repository_ids: [repository.id],
        dry_run: dryRun,
      });
      
      // Show success message
      alert(`${dryRun ? 'Dry run' : 'Migration'} started successfully!`);
      
      // Reload to get updated status
      await loadRepository();
    } catch (error) {
      console.error('Failed to start migration:', error);
      alert('Failed to start migration. Please try again.');
    } finally {
      setMigrating(false);
    }
  };

  const handleAssignToBatch = async () => {
    if (!repository || !selectedBatchId || assigningBatch) return;

    setAssigningBatch(true);
    try {
      await api.addRepositoriesToBatch(selectedBatchId, [repository.id]);
      alert('Repository assigned to batch successfully!');
      await loadRepository();
      setSelectedBatchId(null);
    } catch (error) {
      console.error('Failed to assign to batch:', error);
      alert('Failed to assign to batch. Please try again.');
    } finally {
      setAssigningBatch(false);
    }
  };

  const handleRemoveFromBatch = async () => {
    if (!repository || !repository.batch_id || assigningBatch) return;

    if (!confirm('Are you sure you want to remove this repository from its batch?')) {
      return;
    }

    setAssigningBatch(true);
    try {
      await api.removeRepositoriesFromBatch(repository.batch_id, [repository.id]);
      alert('Repository removed from batch successfully!');
      await loadRepository();
    } catch (error) {
      console.error('Failed to remove from batch:', error);
      alert('Failed to remove from batch. Please try again.');
    } finally {
      setAssigningBatch(false);
    }
  };

  if (loading) return <LoadingSpinner />;
  if (!repository) return <div className="text-center py-12 text-gray-500">Repository not found</div>;

  const canMigrate = ['pending', 'dry_run_complete', 'pre_migration_complete', 'migration_failed'].includes(
    repository.status
  );

  const isInActiveMigration = [
    'queued_for_migration',
    'dry_run_in_progress',
    'dry_run_queued',
    'migrating_content',
    'archive_generating',
    'post_migration',
  ].includes(repository.status);

  const canChangeBatch = !isInActiveMigration && repository.status !== 'complete';

  return (
    <div className="max-w-6xl mx-auto">
      {/* Header */}
      <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
        <div className="flex justify-between items-start">
          <div className="flex-1">
            <h1 className="text-3xl font-light text-gray-900 mb-2">
              {repository.full_name}
            </h1>
            <div className="flex items-center gap-4 mb-4">
              <StatusBadge status={repository.status} />
              {repository.priority === 1 && <Badge color="purple">High Priority</Badge>}
              {repository.batch_id && <Badge color="blue">Batch #{repository.batch_id}</Badge>}
            </div>

            {/* Destination Configuration */}
            {canMigrate && (
              <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Destination (where to migrate)
                    </label>
                    {editingDestination ? (
                      <div className="flex items-center gap-2">
                        <input
                          type="text"
                          value={destinationFullName}
                          onChange={(e) => setDestinationFullName(e.target.value)}
                          placeholder="org/repo"
                          className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          disabled={savingDestination}
                        />
                        <button
                          onClick={handleSaveDestination}
                          disabled={savingDestination}
                          className="px-3 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                        >
                          {savingDestination ? 'Saving...' : 'Save'}
                        </button>
                        <button
                          onClick={() => {
                            setEditingDestination(false);
                            setDestinationFullName(repository.destination_full_name || repository.full_name);
                          }}
                          disabled={savingDestination}
                          className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2">
                        <code className="flex-1 px-3 py-2 bg-white border border-gray-200 rounded-md text-sm text-gray-900">
                          {destinationFullName}
                        </code>
                        <button
                          onClick={() => setEditingDestination(true)}
                          className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
                        >
                          Edit
                        </button>
                      </div>
                    )}
                    <p className="mt-1 text-xs text-gray-500">
                      {destinationFullName === repository.full_name 
                        ? 'Using same organization as source (default)' 
                        : 'Using custom destination organization'}
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Batch Assignment */}
            {canChangeBatch && (
              <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Batch Assignment
                </label>
                {repository.batch_id ? (
                  <div className="flex items-center gap-2">
                    <div className="flex-1 px-3 py-2 bg-white border border-gray-200 rounded-md text-sm">
                      <Badge color="blue">Batch #{repository.batch_id}</Badge>
                    </div>
                    <button
                      onClick={handleRemoveFromBatch}
                      disabled={assigningBatch}
                      className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
                    >
                      {assigningBatch ? 'Removing...' : 'Remove from Batch'}
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <select
                      value={selectedBatchId || ''}
                      onChange={(e) => setSelectedBatchId(e.target.value ? Number(e.target.value) : null)}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      disabled={assigningBatch}
                    >
                      <option value="">Select a batch...</option>
                      {batches.map((batch) => (
                        <option key={batch.id} value={batch.id}>
                          {batch.name} ({batch.type}) - {batch.repository_count} repos
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={handleAssignToBatch}
                      disabled={!selectedBatchId || assigningBatch}
                      className="px-3 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                    >
                      {assigningBatch ? 'Assigning...' : 'Assign to Batch'}
                    </button>
                  </div>
                )}
                <p className="mt-1 text-xs text-gray-500">
                  {repository.batch_id
                    ? 'Repository is assigned to a batch'
                    : batches.length === 0
                    ? 'No ready batches available. Create a batch first.'
                    : 'Assign this repository to a batch for grouped migration'}
                </p>
              </div>
            )}
          </div>

          {/* Migration Actions */}
          <div className="flex flex-col gap-3 ml-6">
            {canMigrate && (
              <>
                <button
                  onClick={() => handleStartMigration(true)}
                  disabled={migrating}
                  className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {migrating ? 'Processing...' : 'Dry Run'}
                </button>
                <button
                  onClick={() => handleStartMigration(false)}
                  disabled={migrating}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
                >
                  {migrating ? 'Processing...' : 'Start Migration'}
                </button>
              </>
            )}
            {repository.status === 'migration_failed' && (
              <button
                onClick={() => handleStartMigration(false)}
                disabled={migrating}
                className="px-4 py-2 bg-yellow-600 text-white rounded-lg text-sm font-medium hover:bg-yellow-700 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap"
              >
                Retry Migration
              </button>
            )}
          </div>
        </div>

        {/* Links */}
        <div className="mt-4 flex gap-4 text-sm">
          <a
            href={repository.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-600 hover:underline"
          >
            View Source Repository →
          </a>
          {repository.destination_url && (
            <a
              href={repository.destination_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-green-600 hover:underline"
            >
              View Migrated Repository →
            </a>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white rounded-lg shadow-sm mb-6">
        <div className="border-b border-gray-200">
          <nav className="flex -mb-px">
            {(['overview', 'history', 'logs'] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-6 py-4 text-sm font-medium border-b-2 transition-colors capitalize ${
                  activeTab === tab
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                {tab === 'overview' ? 'Repository Profile' : tab}
              </button>
            ))}
          </nav>
        </div>

        <div className="p-6">
          {activeTab === 'overview' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <ProfileCard title="Migration Complexity Assessment">
                <ProfileItem label="Repository Size" value={formatBytes(repository.total_size)} />
                <ProfileItem 
                  label="Large Files (>100MB)" 
                  value={repository.has_large_files ? `Yes (${repository.large_file_count}+)` : 'No'} 
                />
                {repository.largest_file && (
                  <ProfileItem 
                    label="Largest File" 
                    value={`${repository.largest_file} (${formatBytes(repository.largest_file_size || 0)})`} 
                  />
                )}
                <ProfileItem 
                  label="Last Activity" 
                  value={repository.last_commit_date ? formatDate(repository.last_commit_date) : 'Unknown'} 
                />
                <ProfileItem label="Uses LFS" value={repository.has_lfs ? 'Yes' : 'No'} />
                <ProfileItem label="Has Submodules" value={repository.has_submodules ? 'Yes' : 'No'} />
                <ProfileItem label="Total Commits" value={repository.commit_count.toLocaleString()} />
              </ProfileCard>

              <ProfileCard title="Verification Metrics">
                <ProfileItem 
                  label="Last Commit SHA" 
                  value={repository.last_commit_sha ? (
                    <code className="text-xs bg-gray-100 px-2 py-1 rounded">{repository.last_commit_sha.substring(0, 8)}</code>
                  ) : 'Unknown'} 
                />
                <ProfileItem label="Branches" value={repository.branch_count} />
                <ProfileItem label="Tags/Releases" value={repository.tag_count} />
                <ProfileItem 
                  label="Issues" 
                  value={`${repository.open_issue_count} open / ${repository.issue_count} total`} 
                />
                <ProfileItem 
                  label="Pull Requests" 
                  value={`${repository.open_pr_count} open / ${repository.pull_request_count} total`} 
                />
                <ProfileItem label="Contributors" value={repository.contributor_count} />
              </ProfileCard>

              <ProfileCard title="Git Properties">
                <ProfileItem label="Default Branch" value={repository.default_branch} />
                <ProfileItem label="Total Size" value={formatBytes(repository.total_size)} />
                <ProfileItem label="Branches" value={repository.branch_count} />
                <ProfileItem label="Commits" value={repository.commit_count.toLocaleString()} />
                <ProfileItem label="Has LFS" value={repository.has_lfs ? 'Yes' : 'No'} />
                <ProfileItem label="Has Submodules" value={repository.has_submodules ? 'Yes' : 'No'} />
              </ProfileCard>

              <ProfileCard title="GitHub Features">
                <ProfileItem label="Wikis" value={repository.has_wiki ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Pages" value={repository.has_pages ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Discussions" value={repository.has_discussions ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Actions" value={repository.has_actions ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Projects" value={repository.has_projects ? 'Enabled' : 'Disabled'} />
                <ProfileItem label="Branch Protections" value={repository.branch_protections} />
                <ProfileItem label="Environments" value={repository.environment_count} />
                <ProfileItem label="Secrets" value={repository.secret_count} />
                <ProfileItem label="Webhooks" value={repository.webhook_count} />
              </ProfileCard>
            </div>
          )}

          {activeTab === 'history' && (
            <div>
              {history.length === 0 ? (
                <p className="text-gray-500">No migration history yet</p>
              ) : (
                <div className="space-y-3">
                  {history.map((event) => (
                    <MigrationEvent key={event.id} event={event} />
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === 'logs' && (
            <div>
              {/* Log Filters */}
              <div className="flex gap-4 mb-4 flex-wrap">
                <select
                  value={logLevel}
                  onChange={(e) => setLogLevel(e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">All Levels</option>
                  <option value="DEBUG">Debug</option>
                  <option value="INFO">Info</option>
                  <option value="WARN">Warning</option>
                  <option value="ERROR">Error</option>
                </select>

                <select
                  value={logPhase}
                  onChange={(e) => setLogPhase(e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">All Phases</option>
                  <option value="discovery">Discovery</option>
                  <option value="pre_migration">Pre-migration</option>
                  <option value="archive_generation">Archive Generation</option>
                  <option value="migration">Migration</option>
                  <option value="post_migration">Post-migration</option>
                </select>

                <input
                  type="text"
                  placeholder="Search logs..."
                  value={logSearch}
                  onChange={(e) => setLogSearch(e.target.value)}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />

                <button
                  onClick={() => loadLogs()}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700"
                >
                  Refresh
                </button>
              </div>

              {/* Logs Display */}
              {logsLoading ? (
                <div className="text-center py-8 text-gray-500">Loading logs...</div>
              ) : logs.length === 0 ? (
                <p className="text-gray-500">No logs available</p>
              ) : (
                <div className="space-y-1 font-mono text-sm max-h-96 overflow-y-auto bg-gray-50 rounded-lg p-4">
                  {logs
                    .filter((log) =>
                      logSearch ? log.message.toLowerCase().includes(logSearch.toLowerCase()) : true
                    )
                    .map((log) => (
                      <LogEntry key={log.id} log={log} />
                    ))}
                </div>
              )}

              {logs.length > 0 && (
                <div className="mt-4 text-sm text-gray-500">
                  Showing {logs.filter((log) => logSearch ? log.message.toLowerCase().includes(logSearch.toLowerCase()) : true).length} of {logs.length} logs
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function MigrationEvent({ event }: { event: MigrationHistory }) {
  return (
    <div className="border-l-4 border-blue-500 pl-4 py-2">
      <div className="flex justify-between items-start">
        <div>
          <div className="font-medium text-gray-900">{event.phase}</div>
          <div className="text-sm text-gray-600">{event.message}</div>
          {event.error_message && (
            <div className="text-sm text-red-600 mt-1">{event.error_message}</div>
          )}
        </div>
        <div className="text-sm text-gray-500">
          {formatDate(event.started_at)}
        </div>
      </div>
      {event.duration_seconds !== undefined && event.duration_seconds !== null && (
        <div className="text-sm text-gray-500 mt-1">
          Duration: {event.duration_seconds}s
        </div>
      )}
    </div>
  );
}

function LogEntry({ log }: { log: MigrationLog }) {
  const [expanded, setExpanded] = useState(false);

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'ERROR': return 'text-red-600 bg-red-50';
      case 'WARN': return 'text-yellow-600 bg-yellow-50';
      case 'INFO': return 'text-blue-600 bg-blue-50';
      case 'DEBUG': return 'text-gray-600 bg-gray-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  const getLevelIcon = (level: string) => {
    switch (level) {
      case 'ERROR': return '❌';
      case 'WARN': return '⚠️';
      case 'INFO': return 'ℹ️';
      case 'DEBUG': return '🔍';
      default: return '•';
    }
  };

  return (
    <div className="hover:bg-gray-100 p-2 rounded cursor-pointer" onClick={() => setExpanded(!expanded)}>
      <div className="flex items-start gap-2">
        {/* Timestamp */}
        <span className="text-gray-500 whitespace-nowrap text-xs">
          {new Date(log.timestamp).toLocaleTimeString()}
        </span>
        
        {/* Level Badge */}
        <span className={`px-2 py-0.5 rounded text-xs font-medium ${getLevelColor(log.level)}`}>
          {getLevelIcon(log.level)} {log.level}
        </span>
        
        {/* Phase & Operation */}
        <span className="text-gray-600 whitespace-nowrap text-xs">
          [{log.phase}:{log.operation}]
        </span>
        
        {/* Message */}
        <span className={`flex-1 text-xs ${log.level === 'ERROR' ? 'text-red-700 font-medium' : 'text-gray-800'}`}>
          {log.message}
        </span>
      </div>
      
      {/* Expanded Details */}
      {expanded && log.details && (
        <div className="mt-2 pl-4 border-l-2 border-gray-300">
          <pre className="text-xs text-gray-600 whitespace-pre-wrap break-words">
            {log.details}
          </pre>
        </div>
      )}
    </div>
  );
}

