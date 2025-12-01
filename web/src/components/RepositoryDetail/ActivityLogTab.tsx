import { useEffect, useState } from 'react';
import { UnderlineNav } from '@primer/react';
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
      // Sort logs by timestamp descending (most recent first)
      const sortedLogs = (response.logs || []).sort((a, b) => 
        new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
      );
      setLogs(sortedLogs);
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
      <UnderlineNav aria-label="Activity view mode">
        <UnderlineNav.Item
          aria-current={viewMode === 'history' ? 'page' : undefined}
          onSelect={() => setViewMode('history')}
        >
          Migration History
        </UnderlineNav.Item>
        <UnderlineNav.Item
          aria-current={viewMode === 'logs' ? 'page' : undefined}
          onSelect={() => setViewMode('logs')}
        >
          Detailed Logs
        </UnderlineNav.Item>
      </UnderlineNav>

      {/* Migration History View */}
      {viewMode === 'history' && (
        <div>
          {history.length === 0 ? (
            <div className="text-center py-12" style={{ color: 'var(--fgColor-muted)' }}>
              <svg 
                className="w-12 h-12 mx-auto mb-3" 
                fill="none" 
                viewBox="0 0 24 24" 
                strokeWidth={1.5} 
                stroke="currentColor"
                style={{ color: 'var(--fgColor-muted)' }}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="font-medium" style={{ color: 'var(--fgColor-default)' }}>No migration history yet</p>
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
              className="px-3 py-2 rounded-lg text-sm cursor-pointer"
              style={{
                backgroundColor: 'var(--control-bgColor-rest)',
                border: '1px solid var(--borderColor-default)',
                color: 'var(--fgColor-default)'
              }}
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
              className="px-3 py-2 rounded-lg text-sm cursor-pointer"
              style={{
                backgroundColor: 'var(--control-bgColor-rest)',
                border: '1px solid var(--borderColor-default)',
                color: 'var(--fgColor-default)'
              }}
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
              className="flex-1 px-3 py-2 rounded-lg text-sm"
              style={{
                backgroundColor: 'var(--control-bgColor-rest)',
                border: '1px solid var(--borderColor-default)',
                color: 'var(--fgColor-default)'
              }}
            />

            <button
              onClick={loadLogs}
              className="px-4 py-1.5 rounded-md text-sm font-medium cursor-pointer transition-opacity hover:opacity-80"
              style={{
                border: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-default)'
              }}
            >
              Refresh
            </button>
          </div>

          {/* Logs Display */}
          {logsLoading ? (
            <div className="text-center py-8" style={{ color: 'var(--fgColor-muted)' }}>Loading logs...</div>
          ) : logs.length === 0 ? (
            <div className="text-center py-12" style={{ color: 'var(--fgColor-muted)' }}>
              <svg 
                className="w-12 h-12 mx-auto mb-3" 
                fill="none" 
                viewBox="0 0 24 24" 
                strokeWidth={1.5} 
                stroke="currentColor"
                style={{ color: 'var(--fgColor-muted)' }}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
              </svg>
              <p className="font-medium" style={{ color: 'var(--fgColor-default)' }}>No logs available</p>
              <p className="text-sm mt-1">Logs will appear here during migration activities</p>
            </div>
          ) : (
            <>
              <div 
                className="space-y-1 font-mono text-sm max-h-96 overflow-y-auto rounded-lg p-4"
                style={{ backgroundColor: 'var(--bgColor-muted)' }}
              >
                {filteredLogs.map((log) => (
                  <LogEntry key={log.id} log={log} />
                ))}
              </div>
              <div className="mt-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
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
    <div 
      className="pl-4 py-3 rounded-r-lg shadow-sm"
      style={{
        borderLeft: '4px solid var(--accent-emphasis)',
        backgroundColor: 'var(--bgColor-default)'
      }}
    >
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <span 
              className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
              style={{
                backgroundColor: 'var(--accent-subtle)',
                color: 'var(--fgColor-accent)'
              }}
            >
              {event.phase}
            </span>
            <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              {formatDate(event.started_at)}
            </span>
          </div>
          <div className="text-sm mb-1" style={{ color: 'var(--fgColor-default)' }}>{event.message}</div>
          {event.error_message && (
            <div 
              className="text-sm p-2 rounded mt-2"
              style={{
                backgroundColor: 'var(--danger-subtle)',
                color: 'var(--fgColor-danger)'
              }}
            >
              <span className="font-medium">Error: </span>
              {event.error_message}
            </div>
          )}
          {event.duration_seconds !== undefined && event.duration_seconds !== null && (
            <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
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

  const getLevelStyle = (level: string) => {
    switch (level) {
      case 'ERROR': 
        return { 
          backgroundColor: 'var(--danger-subtle)', 
          color: 'var(--fgColor-danger)' 
        };
      case 'WARN': 
        return { 
          backgroundColor: 'var(--attention-subtle)', 
          color: 'var(--fgColor-attention)' 
        };
      case 'INFO': 
        return { 
          backgroundColor: 'var(--accent-subtle)', 
          color: 'var(--fgColor-accent)' 
        };
      case 'DEBUG': 
        return { 
          backgroundColor: 'var(--bgColor-muted)', 
          color: 'var(--fgColor-muted)' 
        };
      default: 
        return { 
          backgroundColor: 'var(--bgColor-muted)', 
          color: 'var(--fgColor-muted)' 
        };
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
    <div 
      className="p-2 rounded cursor-pointer transition-opacity hover:opacity-80" 
      onClick={() => setExpanded(!expanded)}
      style={{ backgroundColor: expanded ? 'var(--control-bgColor-hover)' : 'transparent' }}
    >
      <div className="flex items-start gap-2">
        {/* Timestamp */}
        <span className="whitespace-nowrap text-xs" style={{ color: 'var(--fgColor-muted)' }}>
          {new Date(log.timestamp).toLocaleTimeString()}
        </span>
        
        {/* Level Badge */}
        <span 
          className="px-2 py-0.5 rounded text-xs font-medium"
          style={getLevelStyle(log.level)}
        >
          {getLevelIcon(log.level)} {log.level}
        </span>
        
        {/* Phase & Operation */}
        <span className="whitespace-nowrap text-xs" style={{ color: 'var(--fgColor-muted)' }}>
          [{log.phase}:{log.operation}]
        </span>
        
        {/* Initiated By */}
        {log.initiated_by && (
          <span 
            className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium whitespace-nowrap"
            style={{
              backgroundColor: 'var(--accent-subtle)',
              color: 'var(--fgColor-accent)'
            }}
          >
            <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
            </svg>
            {log.initiated_by}
          </span>
        )}
        
        {/* Message */}
        <span 
          className="flex-1 text-xs"
          style={{ 
            color: log.level === 'ERROR' ? 'var(--fgColor-danger)' : 'var(--fgColor-default)',
            fontWeight: log.level === 'ERROR' ? 500 : 400
          }}
        >
          {log.message}
        </span>
      </div>
      
      {/* Expanded Details */}
      {expanded && log.details && (
        <div 
          className="mt-2 pl-4"
          style={{ borderLeft: '2px solid var(--borderColor-default)' }}
        >
          <pre 
            className="text-xs whitespace-pre-wrap break-words"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            {log.details}
          </pre>
        </div>
      )}
    </div>
  );
}

