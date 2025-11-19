import { useOrganizations, useBatches, useProjects } from '../../hooks/useQueries';
import { api } from '../../services/api';

interface FilterBarProps {
  selectedOrganization: string;
  selectedProject: string;
  selectedBatch: string;
  onOrganizationChange: (org: string) => void;
  onProjectChange: (project: string) => void;
  onBatchChange: (batch: string) => void;
}

export function FilterBar({
  selectedOrganization,
  selectedProject,
  selectedBatch,
  onOrganizationChange,
  onProjectChange,
  onBatchChange,
}: FilterBarProps) {
  const { data: organizations } = useOrganizations();
  const { data: projects } = useProjects();
  const { data: batches } = useBatches();

  const handleExecutiveReportExport = async (format: 'csv' | 'json') => {
    try {
      const filters = {
        organization: selectedOrganization || undefined,
        project: selectedProject || undefined,
        batch_id: selectedBatch || undefined,
      };
      const blob = await api.exportExecutiveReport(format, filters);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `executive-migration-report.${format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Failed to export executive report:', error);
    }
  };

  return (
    <div className="bg-white rounded-lg border border-gh-border-default shadow-gh-card p-4 mb-6">
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex-1 min-w-[200px]">
          <label htmlFor="org-filter" className="block text-sm font-semibold text-gh-text-primary mb-1">
            Organization
          </label>
          <select
            id="org-filter"
            value={selectedOrganization}
            onChange={(e) => onOrganizationChange(e.target.value)}
            className="block w-full rounded-md border-gh-border-default text-sm px-3 py-1.5"
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
          <label htmlFor="project-filter" className="block text-sm font-semibold text-gh-text-primary mb-1">
            Project
          </label>
          <select
            id="project-filter"
            value={selectedProject}
            onChange={(e) => onProjectChange(e.target.value)}
            className="block w-full rounded-md border-gh-border-default text-sm px-3 py-1.5"
          >
            <option value="">All Projects</option>
            {projects?.map((project) => (
              <option key={project.project} value={project.project}>
                {project.project} ({project.total_repos} repos)
              </option>
            ))}
          </select>
        </div>

        <div className="flex-1 min-w-[200px]">
          <label htmlFor="batch-filter" className="block text-sm font-semibold text-gh-text-primary mb-1">
            Batch
          </label>
          <select
            id="batch-filter"
            value={selectedBatch}
            onChange={(e) => onBatchChange(e.target.value)}
            className="block w-full rounded-md border-gh-border-default text-sm px-3 py-1.5"
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
            onClick={() => handleExecutiveReportExport('csv')}
            className="px-6 py-2 text-sm font-medium text-gh-text-primary bg-white border border-gh-border-default rounded-md hover:bg-gh-neutral-bg transition-colors"
            title="Export executive report as CSV"
          >
            Export CSV
          </button>
          <button
            onClick={() => handleExecutiveReportExport('json')}
            className="px-6 py-2 text-sm font-medium text-gh-text-primary bg-white border border-gh-border-default rounded-md hover:bg-gh-neutral-bg transition-colors"
            title="Export executive report as JSON"
          >
            Export JSON
          </button>
        </div>
      </div>
    </div>
  );
}

