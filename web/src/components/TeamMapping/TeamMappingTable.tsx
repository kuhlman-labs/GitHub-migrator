import { useState, useCallback } from 'react';
import {
  Button,
  TextInput,
  Flash,
  ActionMenu,
  ActionList,
  Label,
  Spinner,
} from '@primer/react';
import {
  PeopleIcon,
  CheckIcon,
  XIcon,
  SearchIcon,
  DownloadIcon,
  UploadIcon,
  ArrowRightIcon,
} from '@primer/octicons-react';
import { useTeamMappings, useTeamMappingStats } from '../../hooks/useQueries';
import {
  useUpdateTeamMapping,
  useDeleteTeamMapping,
} from '../../hooks/useMutations';
import { TeamMapping, TeamMappingStatus } from '../../types';
import { api } from '../../services/api';
import { Pagination } from '../common/Pagination';

const ITEMS_PER_PAGE = 25;

const statusColors: Record<TeamMappingStatus, 'default' | 'accent' | 'success' | 'attention' | 'danger' | 'done'> = {
  unmapped: 'attention',
  mapped: 'accent',
  skipped: 'default',
};

const statusLabels: Record<TeamMappingStatus, string> = {
  unmapped: 'Unmapped',
  mapped: 'Mapped',
  skipped: 'Skipped',
};

export function TeamMappingTable() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [sourceOrgFilter, setSourceOrgFilter] = useState<string>('');
  const [page, setPage] = useState(1);
  const [editingMapping, setEditingMapping] = useState<{ org: string; slug: string } | null>(null);
  const [editDestOrg, setEditDestOrg] = useState('');
  const [editDestSlug, setEditDestSlug] = useState('');
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<{ created: number; updated: number; errors: number } | null>(null);

  const { data, isLoading, error, refetch } = useTeamMappings({
    search: search || undefined,
    status: statusFilter || undefined,
    source_org: sourceOrgFilter || undefined,
    limit: ITEMS_PER_PAGE,
    offset: (page - 1) * ITEMS_PER_PAGE,
  });

  const { data: stats } = useTeamMappingStats();
  const updateMapping = useUpdateTeamMapping();
  const deleteMapping = useDeleteTeamMapping();

  const mappings = data?.mappings || [];
  const total = data?.total || 0;

  // Get unique source orgs from the mappings (which now contains teams directly)
  const sourceOrgs = Array.from(new Set(mappings.map(t => t.organization || t.source_org || '').filter(Boolean)));

  // Helper to get fields from either new or legacy field names
  const getOrg = (mapping: TeamMapping) => mapping.organization || mapping.source_org || '';
  const getSlug = (mapping: TeamMapping) => mapping.slug || mapping.source_team_slug || '';
  const getName = (mapping: TeamMapping) => mapping.name || mapping.source_team_name || '';

  const handleSearch = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
    setPage(1);
  }, []);

  const handleStatusFilter = useCallback((status: string) => {
    setStatusFilter(status);
    setPage(1);
  }, []);

  const handleSourceOrgFilter = useCallback((org: string) => {
    setSourceOrgFilter(org);
    setPage(1);
  }, []);

  const handleEdit = useCallback((mapping: TeamMapping) => {
    setEditingMapping({ org: getOrg(mapping), slug: getSlug(mapping) });
    setEditDestOrg(mapping.destination_org || '');
    setEditDestSlug(mapping.destination_team_slug || '');
  }, []);

  const handleSaveEdit = useCallback(async () => {
    if (!editingMapping) return;
    
    try {
      await updateMapping.mutateAsync({
        sourceOrg: editingMapping.org,
        sourceTeamSlug: editingMapping.slug,
        updates: {
          destination_org: editDestOrg || undefined,
          destination_team_slug: editDestSlug || undefined,
        },
      });
      setEditingMapping(null);
      setEditDestOrg('');
      setEditDestSlug('');
    } catch (err) {
      console.error('Failed to update mapping:', err);
    }
  }, [editingMapping, editDestOrg, editDestSlug, updateMapping]);

  const handleCancelEdit = useCallback(() => {
    setEditingMapping(null);
    setEditDestOrg('');
    setEditDestSlug('');
  }, []);

  const handleDelete = useCallback(async (sourceOrg: string, sourceTeamSlug: string) => {
    if (window.confirm(`Delete mapping for ${sourceOrg}/${sourceTeamSlug}?`)) {
      try {
        await deleteMapping.mutateAsync({ sourceOrg, sourceTeamSlug });
      } catch (err) {
        console.error('Failed to delete mapping:', err);
      }
    }
  }, [deleteMapping]);

  const handleSkip = useCallback(async (sourceOrg: string, sourceTeamSlug: string) => {
    try {
      await updateMapping.mutateAsync({
        sourceOrg,
        sourceTeamSlug,
        updates: { mapping_status: 'skipped' as const },
      });
    } catch (err) {
      console.error('Failed to skip mapping:', err);
    }
  }, [updateMapping]);

  const handleExport = useCallback(async () => {
    try {
      const blob = await api.exportTeamMappings({
        status: statusFilter || undefined,
        source_org: sourceOrgFilter || undefined,
      });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'team-mappings.csv';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      console.error('Failed to export mappings:', err);
    }
  }, [statusFilter, sourceOrgFilter]);

  const handleImport = useCallback(async (file?: File) => {
    const fileToImport = file || importFile;
    if (!fileToImport) return;
    
    try {
      const result = await api.importTeamMappings(fileToImport);
      setImportResult(result);
      setImportFile(null);
      refetch();
      // Clear the result after 5 seconds
      setTimeout(() => setImportResult(null), 5000);
    } catch (err) {
      console.error('Failed to import mappings:', err);
    }
  }, [importFile, refetch]);

  if (error) {
    return (
      <Flash variant="danger">
        Failed to load team mappings: {error.message}
      </Flash>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Header with stats */}
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-4">
          <h2 className="text-xl font-semibold flex items-center gap-2">
            <PeopleIcon size={24} />
            Team Permission Mapping
          </h2>
          {stats && (
            <div className="flex gap-2">
              <Label variant="accent">{stats.total} Total</Label>
              <Label variant="success">{stats.mapped} Mapped</Label>
              <Label variant="attention">{stats.unmapped} Unmapped</Label>
            </div>
          )}
        </div>
        <div className="flex gap-2">
          <input
            type="file"
            id="import-team-csv-input"
            accept=".csv"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) {
                handleImport(file);
              }
              e.target.value = ''; // Reset for re-upload
            }}
            className="hidden"
          />
          <Button
            onClick={() => document.getElementById('import-team-csv-input')?.click()}
            leadingVisual={UploadIcon}
          >
            Import
          </Button>
          <Button onClick={handleExport} leadingVisual={DownloadIcon}>
            Export
          </Button>
        </div>
      </div>

      {/* Import result notification */}
      {importResult && (
        <Flash variant="success">
          Import complete: {importResult.created} created, {importResult.updated} updated, {importResult.errors} errors
        </Flash>
      )}

      {/* Search and filters */}
      <div className="flex gap-4 items-center">
        <div className="flex-1 max-w-md">
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search by team name or slug..."
            value={search}
            onChange={handleSearch}
            className="w-full"
          />
        </div>
        <ActionMenu>
          <ActionMenu.Button>
            Status: {statusFilter ? statusLabels[statusFilter as TeamMappingStatus] : 'All'}
          </ActionMenu.Button>
          <ActionMenu.Overlay>
            <ActionList selectionVariant="single">
              <ActionList.Item selected={!statusFilter} onSelect={() => handleStatusFilter('')}>
                All
              </ActionList.Item>
              <ActionList.Divider />
              <ActionList.Item selected={statusFilter === 'unmapped'} onSelect={() => handleStatusFilter('unmapped')}>
                Unmapped
              </ActionList.Item>
              <ActionList.Item selected={statusFilter === 'mapped'} onSelect={() => handleStatusFilter('mapped')}>
                Mapped
              </ActionList.Item>
              <ActionList.Item selected={statusFilter === 'skipped'} onSelect={() => handleStatusFilter('skipped')}>
                Skipped
              </ActionList.Item>
            </ActionList>
          </ActionMenu.Overlay>
        </ActionMenu>
        <ActionMenu>
          <ActionMenu.Button>
            Organization: {sourceOrgFilter || 'All'}
          </ActionMenu.Button>
          <ActionMenu.Overlay>
            <ActionList selectionVariant="single">
              <ActionList.Item selected={!sourceOrgFilter} onSelect={() => handleSourceOrgFilter('')}>
                All
              </ActionList.Item>
              <ActionList.Divider />
              {sourceOrgs.map(org => (
                <ActionList.Item
                  key={org}
                  selected={sourceOrgFilter === org}
                  onSelect={() => handleSourceOrgFilter(org)}
                >
                  {org}
                </ActionList.Item>
              ))}
            </ActionList>
          </ActionMenu.Overlay>
        </ActionMenu>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="flex justify-center p-8">
          <Spinner size="large" />
        </div>
      ) : (
        <table className="w-full" style={{ borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
              <th className="text-left p-3 font-medium">Source Team</th>
              <th className="text-center p-3 font-medium w-12"></th>
              <th className="text-left p-3 font-medium">Destination Team</th>
              <th className="text-left p-3 font-medium">Status</th>
              <th className="text-right p-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {mappings.map((mapping) => {
              const org = getOrg(mapping);
              const slug = getSlug(mapping);
              const name = getName(mapping);
              return (
              <tr
                key={`${org}/${slug}`}
                style={{ borderBottom: '1px solid var(--borderColor-muted)' }}
                className="hover:opacity-80"
              >
                <td className="p-3">
                  <div>
                    <div className="font-medium">
                      <span style={{ color: 'var(--fgColor-muted)' }}>{org}/</span>
                      {slug}
                    </div>
                    {name && (
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        {name}
                      </span>
                    )}
                  </div>
                </td>
                <td className="p-3 text-center">
                  <span style={{ color: 'var(--fgColor-muted)' }}><ArrowRightIcon size={16} /></span>
                </td>
                <td className="p-3">
                  {editingMapping?.org === org && editingMapping?.slug === slug ? (
                    <div className="flex items-center gap-2">
                      <TextInput
                        value={editDestOrg}
                        onChange={(e) => setEditDestOrg(e.target.value)}
                        placeholder="dest-org"
                        size="small"
                      />
                      <span>/</span>
                      <TextInput
                        value={editDestSlug}
                        onChange={(e) => setEditDestSlug(e.target.value)}
                        placeholder="team-slug"
                        size="small"
                      />
                      <Button size="small" onClick={handleSaveEdit}>
                        <CheckIcon size={16} />
                      </Button>
                      <Button size="small" variant="invisible" onClick={handleCancelEdit}>
                        <XIcon size={16} />
                      </Button>
                    </div>
                  ) : (
                    <div>
                      {mapping.destination_org && mapping.destination_team_slug ? (
                        <div>
                          <div className="font-medium">
                            <span style={{ color: 'var(--fgColor-muted)' }}>{mapping.destination_org}/</span>
                            {mapping.destination_team_slug}
                          </div>
                          {mapping.destination_team_name && (
                            <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                              {mapping.destination_team_name}
                            </span>
                          )}
                        </div>
                      ) : (
                        <span style={{ color: 'var(--fgColor-muted)' }}>Not mapped</span>
                      )}
                    </div>
                  )}
                </td>
                <td className="p-3">
                  <Label variant={statusColors[mapping.mapping_status as TeamMappingStatus]}>
                    {statusLabels[mapping.mapping_status as TeamMappingStatus]}
                  </Label>
                </td>
                <td className="p-3 text-right">
                  <ActionMenu>
                    <ActionMenu.Button size="small" variant="invisible">
                      Actions
                    </ActionMenu.Button>
                    <ActionMenu.Overlay>
                      <ActionList>
                        <ActionList.Item onSelect={() => handleEdit(mapping)}>
                          Edit mapping
                        </ActionList.Item>
                        {mapping.mapping_status !== 'skipped' && (
                          <ActionList.Item onSelect={() => handleSkip(org, slug)}>
                            Skip team
                          </ActionList.Item>
                        )}
                        <ActionList.Divider />
                        <ActionList.Item onSelect={() => handleDelete(org, slug)}>
                          <span style={{ color: 'var(--fgColor-danger)' }}>Delete mapping</span>
                        </ActionList.Item>
                      </ActionList>
                    </ActionMenu.Overlay>
                  </ActionMenu>
                </td>
              </tr>
            );
            })}
            {mappings.length === 0 && (
              <tr>
                <td colSpan={5} className="p-8 text-center">
                  <span style={{ color: 'var(--fgColor-muted)' }}>
                    No teams found. Run discovery to discover organization teams.
                  </span>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      )}

      {/* Pagination */}
      {total > ITEMS_PER_PAGE && (
        <Pagination
          currentPage={page}
          totalItems={total}
          pageSize={ITEMS_PER_PAGE}
          onPageChange={setPage}
        />
      )}
    </div>
  );
}
