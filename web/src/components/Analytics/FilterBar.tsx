import { useOrganizations, useBatches } from '../../hooks/useQueries';
import { api } from '../../services/api';

interface FilterBarProps {
  selectedOrganization: string;
  selectedBatch: string;
  onOrganizationChange: (org: string) => void;
  onBatchChange: (batch: string) => void;
}

export function FilterBar({
  selectedOrganization,
  selectedBatch,
  onOrganizationChange,
  onBatchChange,
}: FilterBarProps) {
  const { data: organizations } = useOrganizations();
  const { data: batches } = useBatches();

  const handleExport = async (format: 'csv' | 'json') => {
    try {
      const blob = await api.exportMigrationHistory(format);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `migration-analytics.${format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Failed to export data:', error);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-sm p-4 mb-6">
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex-1 min-w-[200px]">
          <label htmlFor="org-filter" className="block text-sm font-medium text-gray-700 mb-1">
            Organization
          </label>
          <select
            id="org-filter"
            value={selectedOrganization}
            onChange={(e) => onOrganizationChange(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          >
            <option value="">All Organizations</option>
            {organizations?.map((org) => (
              <option key={org.organization} value={org.organization}>
                {org.organization} ({org.total_repos} repos)
              </option>
            ))}
          </select>
        </div>

        <div className="flex-1 min-w-[200px]">
          <label htmlFor="batch-filter" className="block text-sm font-medium text-gray-700 mb-1">
            Batch
          </label>
          <select
            id="batch-filter"
            value={selectedBatch}
            onChange={(e) => onBatchChange(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          >
            <option value="">All Batches</option>
            {batches?.map((batch) => (
              <option key={batch.id} value={batch.id.toString()}>
                {batch.name} ({batch.repository_count} repos)
              </option>
            ))}
          </select>
        </div>

        <div className="flex gap-2 items-end">
          <button
            onClick={() => handleExport('csv')}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            Export CSV
          </button>
          <button
            onClick={() => handleExport('json')}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            Export JSON
          </button>
        </div>
      </div>
    </div>
  );
}

