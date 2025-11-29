import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Button } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { HistoryIcon, DownloadIcon, ChevronDownIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { MigrationHistoryEntry } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Pagination } from '../common/Pagination';
import { useToast } from '../../contexts/ToastContext';
import { formatDate, formatDuration } from '../../utils/format';
import { useMigrationHistory } from '../../hooks/useQueries';

export function MigrationHistory() {
  const { data, isLoading, isFetching } = useMigrationHistory();
  const migrations = data?.migrations || [];
  const { showError } = useToast();
  const [searchParams] = useSearchParams();
  
  const searchTerm = searchParams.get('search') || '';
  const [exporting, setExporting] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 50;

  const handleExport = async (format: 'csv' | 'json') => {
    setShowExportMenu(false);
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

  // Reset to page 1 when search term changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchTerm]);

  // Calculate pagination
  const totalPages = Math.ceil(filteredMigrations.length / pageSize);
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedMigrations = filteredMigrations.slice(startIndex, endIndex);

  if (isLoading) return <LoadingSpinner />;

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>Migration History</h1>
        <div className="flex items-center gap-4">
          {/* Export Button with Dropdown */}
          <div className="relative">
            <Button
              onClick={() => setShowExportMenu(!showExportMenu)}
              disabled={exporting || migrations.length === 0}
              leadingVisual={DownloadIcon}
              trailingVisual={ChevronDownIcon}
              variant="primary"
            >
              Export
            </Button>
            {showExportMenu && (
              <>
                {/* Backdrop to close menu when clicking outside */}
                <div 
                  className="fixed inset-0 z-10" 
                  onClick={() => setShowExportMenu(false)}
                />
                {/* Dropdown menu */}
                <div 
                  className="absolute right-0 mt-2 w-48 rounded-lg shadow-lg z-20"
                  style={{
                    backgroundColor: 'var(--bgColor-default)',
                    border: '1px solid var(--borderColor-default)',
                    boxShadow: 'var(--shadow-floating-large)'
                  }}
                >
                  <div className="py-1">
                    <button
                      onClick={() => handleExport('csv')}
                      className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                      style={{ color: 'var(--fgColor-default)' }}
                    >
                      Export as CSV
                    </button>
                    <button
                      onClick={() => handleExport('json')}
                      className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                      style={{ color: 'var(--fgColor-default)' }}
                    >
                      Export as JSON
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="mb-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
        Showing {startIndex + 1}-{Math.min(endIndex, filteredMigrations.length)} of {filteredMigrations.length} migrations
        {searchTerm && ` (filtered from ${migrations.length} total)`}
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
                {paginatedMigrations.map((migration) => (
                  <MigrationRow key={migration.id} migration={migration} />
                ))}
              </tbody>
            </table>
          </div>
          
          {/* Pagination */}
          {totalPages > 1 && (
            <Pagination
              currentPage={currentPage}
              totalItems={filteredMigrations.length}
              pageSize={pageSize}
              onPageChange={setCurrentPage}
            />
          )}
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

