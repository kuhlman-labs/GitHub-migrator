import { useOrganizations, useBatches, useProjects } from '../../hooks/useQueries';

interface FilterBarProps {
  selectedOrganization: string;
  selectedProject: string;
  selectedBatch: string;
  onOrganizationChange: (org: string) => void;
  onProjectChange: (project: string) => void;
  onBatchChange: (batch: string) => void;
  sourceType?: 'github' | 'azuredevops';
}

export function FilterBar({
  selectedOrganization,
  selectedProject,
  selectedBatch,
  onOrganizationChange,
  onProjectChange,
  onBatchChange,
  sourceType = 'github',
}: FilterBarProps) {
  const { data: organizations } = useOrganizations();
  const { data: projects } = useProjects();
  const { data: batches } = useBatches();


  return (
    <div 
      className="rounded-lg border p-4 mb-6"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)'
      }}
    >
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex-1 min-w-[200px]">
          <label htmlFor="org-filter" className="block text-sm font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
            Organization
          </label>
          <select
            id="org-filter"
            value={selectedOrganization}
            onChange={(e) => onOrganizationChange(e.target.value)}
            className="block w-full rounded-md text-sm px-3 py-1.5"
            style={{
              backgroundColor: 'var(--control-bgColor-rest)',
              borderColor: 'var(--borderColor-default)',
              color: 'var(--fgColor-default)',
              border: '1px solid'
            }}
          >
            <option value="">All Organizations</option>
            {organizations?.map((org) => (
              <option key={org.organization} value={org.organization}>
                {org.organization} ({org.total_repos} repos)
              </option>
            ))}
          </select>
        </div>

        {/* Only show Project filter for Azure DevOps sources */}
        {sourceType === 'azuredevops' && (
          <div className="flex-1 min-w-[200px]">
            <label htmlFor="project-filter" className="block text-sm font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
              Project
            </label>
            <select
              id="project-filter"
              value={selectedProject}
              onChange={(e) => onProjectChange(e.target.value)}
              className="block w-full rounded-md text-sm px-3 py-1.5"
              style={{
                backgroundColor: 'var(--control-bgColor-rest)',
                borderColor: 'var(--borderColor-default)',
                color: 'var(--fgColor-default)',
                border: '1px solid'
              }}
            >
              <option value="">All Projects</option>
              {projects?.map((project) => (
                <option key={project.project} value={project.project}>
                  {project.project} ({project.total_repos} repos)
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="flex-1 min-w-[200px]">
          <label htmlFor="batch-filter" className="block text-sm font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
            Batch
          </label>
          <select
            id="batch-filter"
            value={selectedBatch}
            onChange={(e) => onBatchChange(e.target.value)}
            className="block w-full rounded-md text-sm px-3 py-1.5"
            style={{
              backgroundColor: 'var(--control-bgColor-rest)',
              borderColor: 'var(--borderColor-default)',
              color: 'var(--fgColor-default)',
              border: '1px solid'
            }}
          >
            <option value="">All Batches</option>
            {batches?.map((batch) => (
              <option key={batch.id} value={batch.id.toString()}>
                {batch.name} ({batch.repository_count} repos)
              </option>
            ))}
          </select>
        </div>

      </div>
    </div>
  );
}

