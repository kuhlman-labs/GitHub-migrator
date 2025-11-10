import { useEffect, useState } from 'react';
import type { Repository, MigrationHistory, MigrationLog } from '../../types';
import { api } from '../../services/api';
import { formatDate } from '../../utils/format';

interface ActivityLogTabProps {
  repository: Repository;
}

type ViewMode = 'history' | 'logs';

export function ActivityLogTab({ repository }: ActivityLogTabProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('history');
  const [history, setHistory] = useState<MigrationHistory[]>([]);
  const [logs, setLogs] = useState<MigrationLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  
  // Log filters
  const [logLevel, setLogLevel] = useState<string>('');
  const [logPhase, setLogPhase] = useState<string>('');
  const [logSearch, setLogSearch] = useState<string>('');

  // Load migration history
  useEffect(() => {
    if (repository?.id) {
      (async () => {
        try {
          const response = await api.getMigrationHistory(repository.id);
          setHistory(response || []);
        } catch (error) {
          console.error('Failed to load migration history:', error);
        }
      })();
    }
  }, [repository]);

  // Load logs when view mode changes to logs
  useEffect(() => {
    if (viewMode === 'logs' && repository) {
      loadLogs();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [viewMode, logLevel, logPhase, repository]);

  const loadLogs = async () => {
    if (!repository?.id) return;
    
    setLogsLoading(true);
    try {
      const response = await api.getMigrationLogs(repository.id, {
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

  const filteredLogs = logs.filter((log) =>
    logSearch ? log.message.toLowerCase().includes(logSearch.toLowerCase()) : true
  );

  return (
    <div className="space-y-4">
      {/* View Mode Toggle */}
      <div className="flex items-center gap-2 border-b border-gray-200 pb-4">
        <button
          onClick={() => setViewMode('history')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            viewMode === 'history'
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          Migration History
        </button>
        <button
          onClick={() => setViewMode('logs')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            viewMode === 'logs'
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          Detailed Logs
        </button>
      </div>

      {/* Migration History View */}
      {viewMode === 'history' && (
        <div>
          {history.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <svg className="w-12 h-12 mx-auto mb-3 text-gray-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="font-medium">No migration history yet</p>
              <p className="text-sm mt-1">History will appear here once migration activities begin</p>
            </div>
          ) : (
            <div className="space-y-3">
              {history.map((event) => (
                <MigrationEvent key={event.id} event={event} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Detailed Logs View */}
      {viewMode === 'logs' && (
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
              onClick={loadLogs}
              className="px-4 py-1.5 border border-gh-border-default text-gh-text-primary rounded-md text-sm font-medium hover:bg-gh-neutral-bg"
            >
              Refresh
            </button>
          </div>

          {/* Logs Display */}
          {logsLoading ? (
            <div className="text-center py-8 text-gray-500">Loading logs...</div>
          ) : logs.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <svg className="w-12 h-12 mx-auto mb-3 text-gray-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
              </svg>
              <p className="font-medium">No logs available</p>
              <p className="text-sm mt-1">Logs will appear here during migration activities</p>
            </div>
          ) : (
            <>
              <div className="space-y-1 font-mono text-sm max-h-96 overflow-y-auto bg-gray-50 rounded-lg p-4">
                {filteredLogs.map((log) => (
                  <LogEntry key={log.id} log={log} />
                ))}
              </div>
              <div className="mt-4 text-sm text-gray-500">
                Showing {filteredLogs.length} of {logs.length} logs
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

function MigrationEvent({ event }: { event: MigrationHistory }) {
  return (
    <div className="border-l-4 border-blue-500 pl-4 py-3 bg-white rounded-r-lg shadow-sm">
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
              {event.phase}
            </span>
            <span className="text-sm text-gray-500">
              {formatDate(event.started_at)}
            </span>
          </div>
          <div className="text-sm text-gray-700 mb-1">{event.message}</div>
          {event.error_message && (
            <div className="text-sm text-red-600 bg-red-50 p-2 rounded mt-2">
              <span className="font-medium">Error: </span>
              {event.error_message}
            </div>
          )}
          {event.duration_seconds !== undefined && event.duration_seconds !== null && (
            <div className="text-xs text-gray-500 mt-1">
              Duration: {event.duration_seconds}s
            </div>
          )}
        </div>
      </div>
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
        
        {/* Initiated By */}
        {log.initiated_by && (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700 whitespace-nowrap">
            <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
            </svg>
            {log.initiated_by}
          </span>
        )}
        
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

