import { useState } from 'react';
import { TextInput } from '@primer/react';
import { SearchIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { MigrationHistoryEntry } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { formatDate, formatDuration } from '../../utils/format';
import { useMigrationHistory } from '../../hooks/useQueries';

export function MigrationHistory() {
  const { data, isLoading, isFetching } = useMigrationHistory();
  const migrations = data?.migrations || [];
  
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
      alert(errorMessage);
    } finally {
      setExporting(false);
    }
  };

  const filteredMigrations = migrations.filter(m =>
    m.full_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (isLoading) return <LoadingSpinner />;

  return (
    <div className="max-w-7xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold text-gray-900">Migration History</h1>
        <div className="flex items-center gap-4">
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search repositories..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            style={{ width: 300 }}
          />
          <div className="flex items-center gap-2">
            <span className="text-sm text-gh-text-secondary whitespace-nowrap">Export:</span>
            <div className="flex gap-1">
              <button
                onClick={() => handleExport('csv')}
                disabled={exporting}
                className="px-3 py-1.5 text-sm font-medium text-gh-text-primary bg-white border border-gh-border-default rounded-md hover:bg-gh-neutral-bg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                title="Export migration history as CSV"
              >
                CSV
              </button>
              <button
                onClick={() => handleExport('json')}
                disabled={exporting}
                className="px-3 py-1.5 text-sm font-medium text-gh-text-primary bg-white border border-gh-border-default rounded-md hover:bg-gh-neutral-bg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                title="Export migration history as JSON"
              >
                JSON
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="mb-4 text-sm text-gray-600">
        Showing {filteredMigrations.length} of {migrations.length} migrations
      </div>

      {filteredMigrations.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-12 text-center text-gray-500">
          {migrations.length === 0 
            ? 'No migrations yet'
            : 'No migrations match your search'}
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Repository
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Started At
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Completed At
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Duration
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
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
  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case 'complete':
        return 'bg-green-100 text-green-800';
      case 'migration_failed':
        return 'bg-red-100 text-red-800';
      case 'rolled_back':
        return 'bg-orange-100 text-orange-800';
      default:
        return 'bg-gray-100 text-gray-800';
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
    <tr className="hover:bg-gray-50 transition-colors">
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex flex-col">
          <div className="text-sm font-medium text-gray-900">{migration.full_name}</div>
          {migration.destination_url && (
            <a
              href={migration.destination_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-blue-600 hover:underline"
            >
              View destination →
            </a>
          )}
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
        {migration.started_at ? formatDate(migration.started_at) : '-'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
        {migration.completed_at ? formatDate(migration.completed_at) : '-'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
        {migration.duration_seconds ? formatDuration(migration.duration_seconds) : '-'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusBadgeClass(migration.status)}`}>
          {getStatusLabel(migration.status)}
        </span>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm">
        <a
          href={`/repository/${encodeURIComponent(migration.full_name)}`}
          className="text-blue-600 hover:underline"
        >
          View Details
        </a>
      </td>
    </tr>
  );
}

