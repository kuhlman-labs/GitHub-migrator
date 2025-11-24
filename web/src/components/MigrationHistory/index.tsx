import { useState } from 'react';
import { TextInput } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { SearchIcon, HistoryIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { MigrationHistoryEntry } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { useToast } from '../../contexts/ToastContext';
import { formatDate, formatDuration } from '../../utils/format';
import { useMigrationHistory } from '../../hooks/useQueries';

export function MigrationHistory() {
  const { data, isLoading, isFetching } = useMigrationHistory();
  const migrations = data?.migrations || [];
  const { showError } = useToast();
  
  const [searchTerm, setSearchTerm] = useState('');
  const [exporting, setExporting] = useState(false);

  const handleExport = async (format: 'csv' | 'json') => {
    setExporting(true);
    try {
      const blob = await api.exportMigrationHistory(format);
      
      // Create download link
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `migration_history.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (error: any) {
      console.error('Failed to export migration history:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to export migration history. Please try again.';
      showError(errorMessage);
    } finally {
      setExporting(false);
    }
  };

  const filteredMigrations = migrations.filter(m =>
    m.full_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (isLoading) return <LoadingSpinner />;

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>Migration History</h1>
        <div className="flex items-center gap-4">
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search repositories..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            style={{ width: 300 }}
          />
          <div className="flex items-center gap-2">
            <span className="text-sm whitespace-nowrap" style={{ color: 'var(--fgColor-muted)' }}>Export:</span>
            <div className="flex gap-1">
              <button
                onClick={() => handleExport('csv')}
                disabled={exporting}
                className="px-3 py-1.5 text-sm font-medium border rounded-md disabled:opacity-50 disabled:cursor-not-allowed transition-colors cursor-pointer"
                style={{
                  backgroundColor: 'var(--control-bgColor-rest)',
                  borderColor: 'var(--borderColor-default)',
                  color: 'var(--fgColor-default)'
                }}
                onMouseEnter={(e) => {
                  if (!e.currentTarget.disabled) {
                    e.currentTarget.style.backgroundColor = 'var(--control-bgColor-hover)';
                  }
                }}
                onMouseLeave={(e) => {
                  if (!e.currentTarget.disabled) {
                    e.currentTarget.style.backgroundColor = 'var(--control-bgColor-rest)';
                  }
                }}
                title="Export migration history as CSV"
              >
                CSV
              </button>
              <button
                onClick={() => handleExport('json')}
                disabled={exporting}
                className="px-3 py-1.5 text-sm font-medium border rounded-md disabled:opacity-50 disabled:cursor-not-allowed transition-colors cursor-pointer"
                style={{
                  backgroundColor: 'var(--control-bgColor-rest)',
                  borderColor: 'var(--borderColor-default)',
                  color: 'var(--fgColor-default)'
                }}
                onMouseEnter={(e) => {
                  if (!e.currentTarget.disabled) {
                    e.currentTarget.style.backgroundColor = 'var(--control-bgColor-hover)';
                  }
                }}
                onMouseLeave={(e) => {
                  if (!e.currentTarget.disabled) {
                    e.currentTarget.style.backgroundColor = 'var(--control-bgColor-rest)';
                  }
                }}
                title="Export migration history as JSON"
              >
                JSON
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="mb-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
        Showing {filteredMigrations.length} of {migrations.length} migrations
      </div>

      {filteredMigrations.length === 0 ? (
        <Blankslate border>
          <Blankslate.Visual>
            <HistoryIcon size={48} />
          </Blankslate.Visual>
          <Blankslate.Heading>
            {migrations.length === 0 ? 'No migration history yet' : 'No migrations match your search'}
          </Blankslate.Heading>
          <Blankslate.Description>
            {migrations.length === 0 
              ? 'Once you start migrating repositories, their migration history will appear here.'
              : 'Try adjusting your search term to find migrations.'}
          </Blankslate.Description>
          {migrations.length === 0 && (
            <Blankslate.PrimaryAction href="/">
              Go to Dashboard
            </Blankslate.PrimaryAction>
          )}
        </Blankslate>
      ) : (
        <div className="rounded-lg shadow-sm overflow-hidden" style={{ backgroundColor: 'var(--bgColor-default)' }}>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y" style={{ borderColor: 'var(--borderColor-muted)' }}>
              <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Repository
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Started At
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Completed At
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Duration
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y" style={{ backgroundColor: 'var(--bgColor-default)', borderColor: 'var(--borderColor-muted)' }}>
                {filteredMigrations.map((migration) => (
                  <MigrationRow key={migration.id} migration={migration} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function MigrationRow({ migration }: { migration: MigrationHistoryEntry }) {
  const getStatusBadgeStyles = (status: string) => {
    switch (status) {
      case 'complete':
        return { backgroundColor: 'var(--success-subtle)', color: 'var(--fgColor-success)' };
      case 'migration_failed':
        return { backgroundColor: 'var(--danger-subtle)', color: 'var(--fgColor-danger)' };
      case 'rolled_back':
        return { backgroundColor: 'var(--attention-subtle)', color: 'var(--fgColor-attention)' };
      default:
        return { backgroundColor: 'var(--bgColor-muted)', color: 'var(--fgColor-muted)' };
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'complete':
        return 'Complete';
      case 'migration_failed':
        return 'Failed';
      case 'rolled_back':
        return 'Rolled Back';
      default:
        return status;
    }
  };

  return (
    <tr className="hover:opacity-80 transition-opacity">
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex flex-col">
          <div className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>{migration.full_name}</div>
          {migration.destination_url && (
            <a
              href={migration.destination_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs hover:underline"
              style={{ color: 'var(--fgColor-accent)' }}
            >
              View destination →
            </a>
          )}
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-muted)' }}>
        {migration.started_at ? formatDate(migration.started_at) : '-'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-muted)' }}>
        {migration.completed_at ? formatDate(migration.completed_at) : '-'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-muted)' }}>
        {migration.duration_seconds ? formatDuration(migration.duration_seconds) : '-'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <span 
          className="px-2 py-1 text-xs font-medium rounded-full"
          style={getStatusBadgeStyles(migration.status)}
        >
          {getStatusLabel(migration.status)}
        </span>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm">
        <a
          href={`/repository/${encodeURIComponent(migration.full_name)}`}
          className="hover:underline"
          style={{ color: 'var(--fgColor-accent)' }}
        >
          View Details
        </a>
      </td>
    </tr>
  );
}

