import { useState, useCallback } from 'react';
import { ActionMenu, ActionList, TextInput } from '@primer/react';
import { OrganizationIcon, RepoIcon, PackageIcon, SearchIcon, TriangleDownIcon } from '@primer/octicons-react';
import { FilterDropdownButton } from '../common/buttons';
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

  // Search state for each dropdown
  const [orgSearch, setOrgSearch] = useState('');
  const [projectSearch, setProjectSearch] = useState('');
  const [batchSearch, setBatchSearch] = useState('');

  // Filter functions
  const filteredOrgs = organizations?.filter((org) =>
    org.organization.toLowerCase().includes(orgSearch.toLowerCase())
  ) || [];

  const filteredProjects = projects?.filter((project) =>
    project.project.toLowerCase().includes(projectSearch.toLowerCase())
  ) || [];

  const filteredBatches = batches?.filter((batch) =>
    batch.name.toLowerCase().includes(batchSearch.toLowerCase())
  ) || [];

  // Get display text for buttons
  const getOrgButtonText = useCallback(() => {
    if (!selectedOrganization) return 'All Organizations';
    const org = organizations?.find((o) => o.organization === selectedOrganization);
    return org ? `${org.organization} (${org.total_repos})` : selectedOrganization;
  }, [selectedOrganization, organizations]);

  const getProjectButtonText = useCallback(() => {
    if (!selectedProject) return 'All Projects';
    const project = projects?.find((p) => p.project === selectedProject);
    return project ? `${project.project} (${project.total_repos})` : selectedProject;
  }, [selectedProject, projects]);

  const getBatchButtonText = useCallback(() => {
    if (!selectedBatch) return 'All Batches';
    const batch = batches?.find((b) => b.id.toString() === selectedBatch);
    return batch ? `${batch.name} (${batch.repository_count})` : selectedBatch;
  }, [selectedBatch, batches]);

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
        {/* Organization Filter */}
        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
            Organization
          </label>
          <ActionMenu onOpenChange={(open) => { if (!open) setOrgSearch(''); }}>
            <ActionMenu.Anchor>
              <FilterDropdownButton
                leadingVisual={OrganizationIcon}
                trailingAction={TriangleDownIcon}
              >
                <span className="truncate">{getOrgButtonText()}</span>
              </FilterDropdownButton>
            </ActionMenu.Anchor>
            <ActionMenu.Overlay width="large">
              <div className="p-2" style={{ borderBottom: '1px solid var(--borderColor-muted)' }}>
                <TextInput
                  placeholder="Search organizations..."
                  value={orgSearch}
                  onChange={(e) => setOrgSearch(e.target.value)}
                  leadingVisual={SearchIcon}
                  size="small"
                  block
                  onClick={(e) => e.stopPropagation()}
                  onKeyDown={(e) => e.stopPropagation()}
                />
              </div>
              <ActionList selectionVariant="single" style={{ maxHeight: '300px', overflowY: 'auto' }}>
                {!orgSearch && (
                  <>
                    <ActionList.Item
                      selected={!selectedOrganization}
                      onSelect={() => onOrganizationChange('')}
                    >
                      All Organizations
                    </ActionList.Item>
                    {organizations && organizations.length > 0 && <ActionList.Divider />}
                  </>
                )}
                {filteredOrgs.map((org) => (
                  <ActionList.Item
                    key={org.organization}
                    selected={selectedOrganization === org.organization}
                    onSelect={() => onOrganizationChange(org.organization)}
                  >
                    <ActionList.LeadingVisual>
                      <OrganizationIcon />
                    </ActionList.LeadingVisual>
                    {org.organization}
                    <ActionList.TrailingVisual>
                      <span style={{ color: 'var(--fgColor-muted)' }}>{org.total_repos} repos</span>
                    </ActionList.TrailingVisual>
                  </ActionList.Item>
                ))}
                {filteredOrgs.length === 0 && orgSearch && (
                  <ActionList.Item disabled>No matching organizations</ActionList.Item>
                )}
              </ActionList>
            </ActionMenu.Overlay>
          </ActionMenu>
        </div>

        {/* Project Filter - Only for Azure DevOps */}
        {sourceType === 'azuredevops' && (
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
              Project
            </label>
            <ActionMenu onOpenChange={(open) => { if (!open) setProjectSearch(''); }}>
              <ActionMenu.Anchor>
                <FilterDropdownButton
                  leadingVisual={RepoIcon}
                  trailingAction={TriangleDownIcon}
                >
                  <span className="truncate">{getProjectButtonText()}</span>
                </FilterDropdownButton>
              </ActionMenu.Anchor>
              <ActionMenu.Overlay width="large">
                <div className="p-2" style={{ borderBottom: '1px solid var(--borderColor-muted)' }}>
                  <TextInput
                    placeholder="Search projects..."
                    value={projectSearch}
                    onChange={(e) => setProjectSearch(e.target.value)}
                    leadingVisual={SearchIcon}
                    size="small"
                    block
                    onClick={(e) => e.stopPropagation()}
                    onKeyDown={(e) => e.stopPropagation()}
                  />
                </div>
                <ActionList selectionVariant="single" style={{ maxHeight: '300px', overflowY: 'auto' }}>
                  {!projectSearch && (
                    <>
                      <ActionList.Item
                        selected={!selectedProject}
                        onSelect={() => onProjectChange('')}
                      >
                        All Projects
                      </ActionList.Item>
                      {projects && projects.length > 0 && <ActionList.Divider />}
                    </>
                  )}
                  {filteredProjects.map((project) => (
                    <ActionList.Item
                      key={project.project}
                      selected={selectedProject === project.project}
                      onSelect={() => onProjectChange(project.project)}
                    >
                      <ActionList.LeadingVisual>
                        <RepoIcon />
                      </ActionList.LeadingVisual>
                      {project.project}
                      <ActionList.TrailingVisual>
                        <span style={{ color: 'var(--fgColor-muted)' }}>{project.total_repos} repos</span>
                      </ActionList.TrailingVisual>
                    </ActionList.Item>
                  ))}
                  {filteredProjects.length === 0 && projectSearch && (
                    <ActionList.Item disabled>No matching projects</ActionList.Item>
                  )}
                </ActionList>
              </ActionMenu.Overlay>
            </ActionMenu>
          </div>
        )}

        {/* Batch Filter */}
        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
            Batch
          </label>
          <ActionMenu onOpenChange={(open) => { if (!open) setBatchSearch(''); }}>
            <ActionMenu.Anchor>
              <FilterDropdownButton
                leadingVisual={PackageIcon}
                trailingAction={TriangleDownIcon}
              >
                <span className="truncate">{getBatchButtonText()}</span>
              </FilterDropdownButton>
            </ActionMenu.Anchor>
            <ActionMenu.Overlay width="large">
              <div className="p-2" style={{ borderBottom: '1px solid var(--borderColor-muted)' }}>
                <TextInput
                  placeholder="Search batches..."
                  value={batchSearch}
                  onChange={(e) => setBatchSearch(e.target.value)}
                  leadingVisual={SearchIcon}
                  size="small"
                  block
                  onClick={(e) => e.stopPropagation()}
                  onKeyDown={(e) => e.stopPropagation()}
                />
              </div>
              <ActionList selectionVariant="single" style={{ maxHeight: '300px', overflowY: 'auto' }}>
                {!batchSearch && (
                  <>
                    <ActionList.Item
                      selected={!selectedBatch}
                      onSelect={() => onBatchChange('')}
                    >
                      All Batches
                    </ActionList.Item>
                    {batches && batches.length > 0 && <ActionList.Divider />}
                  </>
                )}
                {filteredBatches.map((batch) => (
                  <ActionList.Item
                    key={batch.id}
                    selected={selectedBatch === batch.id.toString()}
                    onSelect={() => onBatchChange(batch.id.toString())}
                  >
                    <ActionList.LeadingVisual>
                      <PackageIcon />
                    </ActionList.LeadingVisual>
                    {batch.name}
                    <ActionList.TrailingVisual>
                      <span style={{ color: 'var(--fgColor-muted)' }}>{batch.repository_count} repos</span>
                    </ActionList.TrailingVisual>
                  </ActionList.Item>
                ))}
                {filteredBatches.length === 0 && batchSearch && (
                  <ActionList.Item disabled>No matching batches</ActionList.Item>
                )}
              </ActionList>
            </ActionMenu.Overlay>
          </ActionMenu>
        </div>
      </div>
    </div>
  );
}
