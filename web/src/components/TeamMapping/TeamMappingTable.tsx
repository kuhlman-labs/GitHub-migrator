import { useState, useCallback } from 'react';
import {
  Button,
  TextInput,
  Flash,
  ActionMenu,
  ActionList,
  Label,
  Spinner,
  Dialog,
} from '@primer/react';
import {
  PeopleIcon,
  CheckIcon,
  XIcon,
  SearchIcon,
  DownloadIcon,
  UploadIcon,
  ArrowRightIcon,
  PlayIcon,
  AlertIcon,
  InfoIcon,
  ClockIcon,
} from '@primer/octicons-react';
import { useTeamMappings, useTeamMappingStats, useTeamMigrationStatus } from '../../hooks/useQueries';
import {
  useUpdateTeamMapping,
  useDeleteTeamMapping,
  useExecuteTeamMigration,
  useCancelTeamMigration,
  useResetTeamMigrationStatus,
} from '../../hooks/useMutations';
import { TeamMapping, TeamMappingStatus } from '../../types';
import { api } from '../../services/api';
import { Pagination } from '../common/Pagination';
import { TeamDetailPanel } from './TeamDetailPanel';

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

// Migration progress component
function MigrationProgress({
  isRunning,
  progress,
  executionStats,
  onCancel,
  onReset,
}: {
  isRunning: boolean;
  progress?: {
    total_teams: number;
    processed_teams: number;
    created_teams: number;
    skipped_teams: number;
    failed_teams: number;
    total_repos_synced: number;
    current_team?: string;
    status: string;
    errors?: string[];
  };
  executionStats?: {
    pending: number;
    in_progress: number;
    completed: number;
    failed: number;
    total_repos_synced: number;
  };
  onCancel: () => void;
  onReset: () => void;
}) {
  if (!isRunning && !progress && !executionStats?.completed && !executionStats?.failed) {
    return null;
  }

  const percentComplete = progress && progress.total_teams > 0
    ? Math.round((progress.processed_teams / progress.total_teams) * 100)
    : 0;

  return (
    <div
      className="p-4 rounded-lg mb-4"
      style={{
        backgroundColor: 'var(--bgColor-muted)',
        border: `1px solid ${isRunning ? 'var(--borderColor-accent-emphasis)' : 'var(--borderColor-default)'}`,
      }}
    >
      <div className="flex justify-between items-start mb-2">
        <div>
          <div className="font-semibold flex items-center gap-2">
            {isRunning ? (
              <>
                <Spinner size="small" /> Migration In Progress
              </>
            ) : progress?.status === 'completed' || progress?.status === 'completed_with_errors' ? (
              <>
                <CheckIcon size={16} /> Migration Complete
              </>
            ) : (
              <>
                <ClockIcon size={16} /> Migration Status
              </>
            )}
          </div>
          {progress?.current_team && (
            <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              Processing: {progress.current_team}
            </span>
          )}
        </div>
        <div className="flex gap-2">
          {isRunning ? (
            <Button size="small" variant="danger" onClick={onCancel}>
              Cancel
            </Button>
          ) : (
            <Button size="small" variant="invisible" onClick={onReset}>
              Reset Status
            </Button>
          )}
        </div>
      </div>

      {isRunning && progress && (
        <div className="mb-4">
          <div className="flex justify-between mb-1">
            <span className="text-sm">
              {progress.processed_teams} of {progress.total_teams} teams processed
            </span>
            <span className="text-sm">{percentComplete}%</span>
          </div>
          <div
            className="h-2 rounded-full overflow-hidden"
            style={{ backgroundColor: 'var(--bgColor-neutral-muted)' }}
          >
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${percentComplete}%`,
                backgroundColor: 'var(--bgColor-accent-emphasis)',
              }}
            />
          </div>
        </div>
      )}

      <div className="flex gap-3 flex-wrap">
        {progress ? (
          <>
            <Label variant="success">{progress.created_teams} Created</Label>
            <Label variant="default">{progress.skipped_teams} Skipped (existed)</Label>
            {progress.failed_teams > 0 && (
              <Label variant="danger">{progress.failed_teams} Failed</Label>
            )}
            <Label variant="accent">{progress.total_repos_synced} Repo Permissions</Label>
          </>
        ) : executionStats && (
          <>
            <Label variant="success">{executionStats.completed} Completed</Label>
            <Label variant="default">{executionStats.pending} Pending</Label>
            {executionStats.failed > 0 && (
              <Label variant="danger">{executionStats.failed} Failed</Label>
            )}
            <Label variant="accent">{executionStats.total_repos_synced} Repo Permissions</Label>
          </>
        )}
      </div>

      {progress?.errors && progress.errors.length > 0 && (
        <div className="mt-4">
          <span className="font-semibold text-sm" style={{ color: 'var(--fgColor-danger)' }}>
            Errors:
          </span>
          <ul
            className="m-0 mt-1 pl-4 text-xs max-h-24 overflow-auto"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            {progress.errors.slice(0, 10).map((err, i) => (
              <li key={i}>{err}</li>
            ))}
            {progress.errors.length > 10 && (
              <li>... and {progress.errors.length - 10} more errors</li>
            )}
          </ul>
        </div>
      )}
    </div>
  );
}

export function TeamMappingTable() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [sourceOrgFilter, setSourceOrgFilter] = useState<string>('');
  const [page, setPage] = useState(1);
  const [editingMapping, setEditingMapping] = useState<{ org: string; slug: string } | null>(null);
  const [editDestOrg, setEditDestOrg] = useState('');
  const [editDestSlug, setEditDestSlug] = useState('');
  const [importResult, setImportResult] = useState<{ created: number; updated: number; errors: number } | null>(null);
  const [selectedTeam, setSelectedTeam] = useState<{ org: string; slug: string } | null>(null);
  const [showExecuteDialog, setShowExecuteDialog] = useState(false);
  const [showSingleTeamDialog, setShowSingleTeamDialog] = useState<{ org: string; slug: string } | null>(null);
  const [dryRun, setDryRun] = useState(false);

  const { data, isLoading, error, refetch } = useTeamMappings({
    search: search || undefined,
    status: statusFilter || undefined,
    source_org: sourceOrgFilter || undefined,
    limit: ITEMS_PER_PAGE,
    offset: (page - 1) * ITEMS_PER_PAGE,
  });

  const { data: stats } = useTeamMappingStats();
  const { data: migrationStatus } = useTeamMigrationStatus();
  const updateMapping = useUpdateTeamMapping();
  const deleteMapping = useDeleteTeamMapping();
  const executeMigration = useExecuteTeamMigration();
  const cancelMigration = useCancelTeamMigration();
  const resetMigration = useResetTeamMigrationStatus();

  const mappings = data?.mappings || [];
  const total = data?.total || 0;

  // Get unique source orgs from the mappings
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

  const handleImport = useCallback(async (file: File) => {
    try {
      const result = await api.importTeamMappings(file);
      setImportResult(result);
      refetch();
      setTimeout(() => setImportResult(null), 5000);
    } catch (err) {
      console.error('Failed to import mappings:', err);
    }
  }, [refetch]);

  const handleExecute = useCallback(async () => {
    try {
      await executeMigration.mutateAsync({
        source_org: sourceOrgFilter || undefined,
        dry_run: dryRun,
      });
      setShowExecuteDialog(false);
    } catch (err) {
      console.error('Failed to execute migration:', err);
    }
  }, [executeMigration, sourceOrgFilter, dryRun]);

  const handleCancel = useCallback(async () => {
    try {
      await cancelMigration.mutateAsync();
    } catch (err) {
      console.error('Failed to cancel migration:', err);
    }
  }, [cancelMigration]);

  const handleReset = useCallback(async () => {
    if (window.confirm('Reset all team migration statuses to pending? This will allow you to re-run the migration.')) {
      try {
        await resetMigration.mutateAsync(sourceOrgFilter || undefined);
        refetch();
      } catch (err) {
        console.error('Failed to reset migration status:', err);
      }
    }
  }, [resetMigration, sourceOrgFilter, refetch]);

  const handleMigrateSingleTeam = useCallback((sourceOrg: string, sourceTeamSlug: string) => {
    setShowSingleTeamDialog({ org: sourceOrg, slug: sourceTeamSlug });
  }, []);

  const handleConfirmSingleTeamMigration = useCallback(async () => {
    if (!showSingleTeamDialog) return;
    
    try {
      await executeMigration.mutateAsync({
        source_org: showSingleTeamDialog.org,
        source_team_slug: showSingleTeamDialog.slug,
        dry_run: false,
      });
      setShowSingleTeamDialog(null);
    } catch (err) {
      console.error('Failed to migrate team:', err);
    }
  }, [executeMigration, showSingleTeamDialog]);

  const handleTeamClick = useCallback((org: string, slug: string) => {
    setSelectedTeam({ org, slug });
  }, []);

  if (error) {
    return (
      <Flash variant="danger">
        Failed to load team mappings: {error.message}
      </Flash>
    );
  }

  const hasMappedTeams = (stats?.mapped || 0) > 0;
  const isRunning = migrationStatus?.is_running || false;

  return (
    <div className="flex flex-col gap-4">
      {/* Migration Progress */}
      <MigrationProgress
        isRunning={isRunning}
        progress={migrationStatus?.progress}
        executionStats={migrationStatus?.execution_stats}
        onCancel={handleCancel}
        onReset={handleReset}
      />

      {/* Header with stats */}
      <div className="flex justify-between items-center flex-wrap gap-4">
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
              e.target.value = '';
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
          <button
            onClick={() => setShowExecuteDialog(true)}
            disabled={!hasMappedTeams || isRunning}
            className="px-4 py-2 rounded-md text-sm font-medium border-0 transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            style={{ 
              backgroundColor: (!hasMappedTeams || isRunning) ? '#94d3a2' : '#1a7f37',
              color: '#ffffff',
              fontWeight: 600
            }}
            onMouseEnter={(e) => {
              if (hasMappedTeams && !isRunning) {
                e.currentTarget.style.backgroundColor = '#2da44e';
              }
            }}
            onMouseLeave={(e) => {
              if (hasMappedTeams && !isRunning) {
                e.currentTarget.style.backgroundColor = '#1a7f37';
              }
            }}
          >
            <PlayIcon size={16} />
            Migrate Teams
          </button>
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
        {sourceOrgs.length > 0 && (
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
        )}
      </div>

      {/* Info banner for unmapped teams */}
      {stats && stats.unmapped > 0 && (
        <div
          className="flex items-center gap-3 px-4 py-3 rounded-md text-sm"
          style={{
            backgroundColor: 'var(--bgColor-attention-muted)',
            border: '1px solid var(--borderColor-attention-muted)',
          }}
        >
          <InfoIcon size={16} className="flex-shrink-0" />
          <span>
            <strong>{stats.unmapped} teams</strong> need mapping. Click a team row to view details, or use Export → Edit → Import for bulk updates.
          </span>
        </div>
      )}

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
              <th className="text-right p-3 pr-4 font-medium w-24">Actions</th>
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
                style={{ borderBottom: '1px solid var(--borderColor-muted)', cursor: 'pointer' }}
                className="hover:opacity-80"
                onClick={() => handleTeamClick(org, slug)}
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
                <td className="p-3" onClick={(e) => e.stopPropagation()}>
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
                <td className="p-3 pr-4 text-right w-24" onClick={(e) => e.stopPropagation()}>
                  <ActionMenu>
                    <ActionMenu.Button size="small" variant="invisible">
                      Actions
                    </ActionMenu.Button>
                    <ActionMenu.Overlay>
                      <ActionList>
                        <ActionList.Item onSelect={() => handleTeamClick(org, slug)}>
                          View details
                        </ActionList.Item>
                        <ActionList.Item onSelect={() => handleEdit(mapping)}>
                          Edit mapping
                        </ActionList.Item>
                        {mapping.mapping_status === 'mapped' && (
                          <ActionList.Item onSelect={() => handleMigrateSingleTeam(org, slug)}>
                            <span style={{ color: 'var(--fgColor-success)' }}>Migrate team</span>
                          </ActionList.Item>
                        )}
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

      {/* Team Detail Panel */}
      {selectedTeam && (
        <TeamDetailPanel
          org={selectedTeam.org}
          teamSlug={selectedTeam.slug}
          onClose={() => setSelectedTeam(null)}
          onEditMapping={(org, slug) => {
            setSelectedTeam(null);
            const mapping = mappings.find(m => getOrg(m) === org && getSlug(m) === slug);
            if (mapping) {
              handleEdit(mapping);
            }
          }}
        />
      )}

      {/* Execute Migration Dialog */}
      {showExecuteDialog && (
        <Dialog
          title="Migrate Teams"
          onClose={() => setShowExecuteDialog(false)}
        >
          <div className="p-4">
            <p className="mb-4">
              This will create teams in the destination organization and apply repository permissions for all mapped teams.
            </p>

            <Flash variant="warning" className="mb-4">
              <div className="flex items-start gap-2">
                <AlertIcon size={16} className="flex-shrink-0 mt-0.5" />
                <div>
                  <strong>EMU/IdP Notice:</strong> Teams are created <strong>without members</strong>. 
                  If your destination is an EMU environment, manage team membership through your Identity Provider (IdP/SCIM). 
                  Repository permissions will be applied to teams.
                </div>
              </div>
            </Flash>

            <div className="mb-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={dryRun}
                  onChange={(e) => setDryRun(e.target.checked)}
                />
                <span>Dry run (preview changes without applying)</span>
              </label>
            </div>

            {sourceOrgFilter && (
              <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
                Only teams from <strong>{sourceOrgFilter}</strong> will be processed.
              </p>
            )}

            <div className="flex justify-end gap-2">
              <Button onClick={() => setShowExecuteDialog(false)}>Cancel</Button>
              <Button
                variant="primary"
                onClick={handleExecute}
                disabled={executeMigration.isPending}
              >
                {executeMigration.isPending ? (
                  <>
                    <Spinner size="small" /> Starting...
                  </>
                ) : dryRun ? (
                  'Start Dry Run'
                ) : (
                  'Start Migration'
                )}
              </Button>
            </div>
          </div>
        </Dialog>
      )}

      {/* Single Team Migration Dialog */}
      {showSingleTeamDialog && (
        <Dialog
          title="Migrate Team"
          onClose={() => setShowSingleTeamDialog(null)}
        >
          <div className="p-4">
            <p className="mb-4">
              This will create the team <strong>{showSingleTeamDialog.org}/{showSingleTeamDialog.slug}</strong> in the destination organization and apply repository permissions.
            </p>

            <Flash variant="warning" className="mb-4">
              <div className="flex items-start gap-2">
                <AlertIcon size={16} className="flex-shrink-0 mt-0.5" />
                <div>
                  <strong>EMU/IdP Notice:</strong> The team will be created <strong>without members</strong>. 
                  If your destination is an EMU environment, manage team membership through your Identity Provider (IdP/SCIM).
                </div>
              </div>
            </Flash>

            <div className="flex justify-end gap-2">
              <Button onClick={() => setShowSingleTeamDialog(null)}>Cancel</Button>
              <Button
                variant="primary"
                onClick={handleConfirmSingleTeamMigration}
                disabled={executeMigration.isPending}
              >
                {executeMigration.isPending ? (
                  <>
                    <Spinner size="small" /> Migrating...
                  </>
                ) : (
                  'Migrate Team'
                )}
              </Button>
            </div>
          </div>
        </Dialog>
      )}
    </div>
  );
}
