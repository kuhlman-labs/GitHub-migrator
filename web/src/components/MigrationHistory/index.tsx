import { useEffect, useState } from 'react';
import { api } from '../../services/api';
import type { MigrationHistoryEntry } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { formatDate, formatDuration } from '../../utils/format';

export function MigrationHistory() {
  const [migrations, setMigrations] = useState<MigrationHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    loadMigrationHistory();
  }, []);

  const loadMigrationHistory = async () => {
    setLoading(true);
    try {
      const data = await api.getMigrationHistoryList();
      setMigrations(data.migrations || []);
    } catch (error) {
      console.error('Failed to load migration history:', error);
    } finally {
      setLoading(false);
    }
  };

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
    } catch (error) {
      console.error('Failed to export migration history:', error);
      alert('Failed to export migration history. Please try again.');
    } finally {
      setExporting(false);
    }
  };

  const filteredMigrations = migrations.filter(m =>
    m.full_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (loading) return <LoadingSpinner />;

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-light text-gray-900">Migration History</h1>
        <div className="flex gap-4">
          <input
            type="text"
            placeholder="Search repositories..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <button
            onClick={() => handleExport('csv')}
            disabled={exporting}
            className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {exporting ? 'Exporting...' : 'Export CSV'}
          </button>
          <button
            onClick={() => handleExport('json')}
            disabled={exporting}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {exporting ? 'Exporting...' : 'Export JSON'}
          </button>
        </div>
      </div>

      <div className="mb-4 text-sm text-gray-600">
        Showing {filteredMigrations.length} of {migrations.length} completed migrations
      </div>

      {filteredMigrations.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-12 text-center text-gray-500">
          {migrations.length === 0 
            ? 'No completed migrations yet'
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
        <span className="px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800">
          {migration.status}
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

