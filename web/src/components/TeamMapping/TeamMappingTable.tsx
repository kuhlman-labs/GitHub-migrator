import { useState, useCallback } from 'react';
import {
  TextInput,
  Flash,
  ActionMenu,
  ActionList,
  Label,
  Spinner,
  FormControl,
} from '@primer/react';
import { Button, BorderedButton, SuccessButton, PrimaryButton } from '../common/buttons';
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
  OrganizationIcon,
  FilterIcon,
  TriangleDownIcon,
  PencilIcon,
  SkipIcon,
  TrashIcon,
  EyeIcon,
  RocketIcon,
} from '@primer/octicons-react';
import { useTeamMappings, useTeamMappingStats, useTeamMigrationStatus, useTeamSourceOrgs } from '../../hooks/useQueries';
import {
  useUpdateTeamMapping,
  useDeleteTeamMapping,
  useExecuteTeamMigration,
  useCancelTeamMigration,
  useResetTeamMigrationStatus,
  useDiscoverTeams,
} from '../../hooks/useMutations';
import { useTableState } from '../../hooks/useTableState';
import { useDialogState } from '../../hooks/useDialogState';
import { TeamMapping, TeamMappingStatus } from '../../types';
import { api } from '../../services/api';
import { Pagination } from '../common/Pagination';
import { ConfirmationDialog } from '../common/ConfirmationDialog';
import { FormDialog } from '../common/FormDialog';
import { TeamDetailPanel } from './TeamDetailPanel';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

const ITEMS_PER_PAGE = 25;

// Helper to get fields from either new or legacy field names
const getOrg = (mapping: TeamMapping) => mapping.organization || mapping.source_org || '';
const getSlug = (mapping: TeamMapping) => mapping.slug || mapping.source_team_slug || '';
const getName = (mapping: TeamMapping) => mapping.name || mapping.source_team_name || '';

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

// Sync status colors and labels
const syncStatusColors: Record<string, 'default' | 'accent' | 'success' | 'attention' | 'danger' | 'done'> = {
  pending: 'default',
  team_only: 'accent',
  needs_sync: 'attention',
  partial: 'attention',
  complete: 'success',
  failed: 'danger',
};

const syncStatusLabels: Record<string, string> = {
  pending: 'Pending',
  team_only: 'Team Only',
  needs_sync: 'Sync Needed',
  partial: 'Partial',
  complete: 'Complete',
  failed: 'Failed',
};

const syncStatusDescriptions: Record<string, string> = {
  pending: 'Not yet migrated',
  team_only: 'Team created, no repos to sync',
  needs_sync: 'Team created, repos need sync',
  partial: 'Some repo permissions synced',
  complete: 'All repo permissions synced',
  failed: 'Migration failed',
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
    needs_sync: number;
    team_only: number;
    partial: number;
    total_repos_synced: number;
    total_repos_eligible: number;
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
            {executionStats.needs_sync > 0 && (
              <Label variant="attention">{executionStats.needs_sync} Need Sync</Label>
            )}
            {executionStats.team_only > 0 && (
              <Label variant="accent">{executionStats.team_only} Team Only</Label>
            )}
            {executionStats.partial > 0 && (
              <Label variant="attention">{executionStats.partial} Partial</Label>
            )}
            <Label variant="done">
              {executionStats.total_repos_synced}/{executionStats.total_repos_eligible} Repo Permissions
            </Label>
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

interface TeamMappingFilters extends Record<string, unknown> {
  status: string;
  sourceOrg: string;
}

export function TeamMappingTable() {
  const { showError } = useToast();
  
  // Use shared table state hook for pagination, search, and filtering
  const { page, search, filters, setPage, setSearch, updateFilter, offset, limit } = useTableState<TeamMappingFilters>({
    initialFilters: { status: '', sourceOrg: '' },
    pageSize: ITEMS_PER_PAGE,
  });
  
  const [orgSearchFilter, setOrgSearchFilter] = useState('');
  const [editingMapping, setEditingMapping] = useState<{ org: string; slug: string } | null>(null);
  const [editDestOrg, setEditDestOrg] = useState('');
  const [editDestSlug, setEditDestSlug] = useState('');
  const [importResult, setImportResult] = useState<{ created: number; updated: number; errors: number } | null>(null);
  const [selectedTeam, setSelectedTeam] = useState<{ org: string; slug: string } | null>(null);
  const [dryRun, setDryRun] = useState(false);
  const [discoverOrg, setDiscoverOrg] = useState('');
  const [actionResult, setActionResult] = useState<{ type: 'success' | 'danger'; message: string } | null>(null);
  
  // Dialog state using shared hooks
  const executeDialog = useDialogState();
  const singleTeamDialog = useDialogState<{ org: string; slug: string }>();
  const discoverDialog = useDialogState();
  const resetDialog = useDialogState();
  const deleteDialog = useDialogState<{ org: string; slug: string }>();

  const { data, isLoading, error, refetch } = useTeamMappings({
    search: search || undefined,
    status: filters.status || undefined,
    source_org: filters.sourceOrg || undefined,
    limit,
    offset,
  });

  const { data: stats } = useTeamMappingStats(filters.sourceOrg || undefined);
  const { data: migrationStatus } = useTeamMigrationStatus();
  const { data: sourceOrgsData } = useTeamSourceOrgs();
  const sourceOrgs = sourceOrgsData || [];
  const updateMapping = useUpdateTeamMapping();
  const deleteMapping = useDeleteTeamMapping();
  const executeMigration = useExecuteTeamMigration();
  const cancelMigration = useCancelTeamMigration();
  const resetMigration = useResetTeamMigrationStatus();
  const discoverTeams = useDiscoverTeams();

  const mappings = data?.mappings || [];
  const total = data?.total || 0;

  const handleSearch = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
  }, [setSearch]);

  const handleStatusFilter = useCallback((status: string) => {
    updateFilter('status', status);
  }, [updateFilter]);

  const handleSourceOrgFilter = useCallback((org: string) => {
    updateFilter('sourceOrg', org);
  }, [updateFilter]);

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
    } catch {
      // Update failed, mutation will show error
    }
  }, [editingMapping, editDestOrg, editDestSlug, updateMapping]);

  const handleCancelEdit = useCallback(() => {
    setEditingMapping(null);
    setEditDestOrg('');
    setEditDestSlug('');
  }, []);

  const handleDelete = useCallback((sourceOrg: string, sourceTeamSlug: string) => {
    deleteDialog.open({ org: sourceOrg, slug: sourceTeamSlug });
  }, [deleteDialog]);

  const handleConfirmDelete = useCallback(async () => {
    if (!deleteDialog.data) return;
    try {
      await deleteMapping.mutateAsync({ sourceOrg: deleteDialog.data.org, sourceTeamSlug: deleteDialog.data.slug });
      deleteDialog.close();
    } catch {
      // Delete failed, mutation will show error
    }
  }, [deleteMapping, deleteDialog]);

  const handleSkip = useCallback(async (sourceOrg: string, sourceTeamSlug: string) => {
    try {
      await updateMapping.mutateAsync({
        sourceOrg,
        sourceTeamSlug,
        updates: { mapping_status: 'skipped' as const },
      });
    } catch {
      // Skip failed, mutation will show error
    }
  }, [updateMapping]);

  const handleExport = useCallback(async () => {
    try {
      const blob = await api.exportTeamMappings({
        status: filters.status || undefined,
        source_org: filters.sourceOrg || undefined,
      });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'team-mappings.csv';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      handleApiError(error, showError, 'Failed to export mappings');
    }
  }, [filters.status, filters.sourceOrg, showError]);

  const handleImport = useCallback(async (file: File) => {
    try {
      const result = await api.importTeamMappings(file);
      setImportResult(result);
      refetch();
      setTimeout(() => setImportResult(null), 5000);
    } catch (error) {
      handleApiError(error, showError, 'Failed to import mappings');
    }
  }, [refetch, showError]);

  const handleExecute = useCallback(async () => {
    try {
      await executeMigration.mutateAsync({
        source_org: filters.sourceOrg || undefined,
        dry_run: dryRun,
      });
      executeDialog.close();
    } catch {
      // Execute migration failed, mutation will show error
    }
  }, [executeMigration, filters.sourceOrg, dryRun, executeDialog]);

  const handleCancel = useCallback(async () => {
    try {
      await cancelMigration.mutateAsync();
    } catch {
      // Cancel failed, mutation will show error
    }
  }, [cancelMigration]);

  const handleReset = useCallback(() => {
    resetDialog.open();
  }, [resetDialog]);

  const handleConfirmReset = useCallback(async () => {
    try {
      await resetMigration.mutateAsync(filters.sourceOrg || undefined);
      refetch();
      resetDialog.close();
    } catch {
      // Reset failed, mutation will show error
    }
  }, [resetMigration, filters.sourceOrg, refetch, resetDialog]);

  const handleMigrateSingleTeam = useCallback((sourceOrg: string, sourceTeamSlug: string) => {
    singleTeamDialog.open({ org: sourceOrg, slug: sourceTeamSlug });
  }, [singleTeamDialog]);

  const handleConfirmSingleTeamMigration = useCallback(async () => {
    if (!singleTeamDialog.data) return;
    
    try {
      await executeMigration.mutateAsync({
        source_org: singleTeamDialog.data.org,
        source_team_slug: singleTeamDialog.data.slug,
        dry_run: false,
      });
      singleTeamDialog.close();
    } catch {
      // Single team migration failed, mutation will show error
    }
  }, [executeMigration, singleTeamDialog]);

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
      <div className="flex justify-between items-start flex-wrap gap-4">
        <div>
          <div className="flex items-center gap-4">
            <h2 className="text-2xl font-semibold flex items-center gap-2" style={{ color: 'var(--fgColor-default)' }}>
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
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Map source teams to destination GitHub teams for permission sync
          </p>
        </div>
        <div className="flex gap-2">
          {/* Data Management - Import/Export */}
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
          <BorderedButton
            onClick={() => document.getElementById('import-team-csv-input')?.click()}
            leadingVisual={UploadIcon}
          >
            Import
          </BorderedButton>
          <BorderedButton
            onClick={handleExport}
            leadingVisual={DownloadIcon}
          >
            Export
          </BorderedButton>
          
          {/* Discovery/Setup action */}
          <PrimaryButton
            onClick={() => discoverDialog.open()}
            leadingVisual={PeopleIcon}
            disabled={discoverTeams.isPending}
          >
            {discoverTeams.isPending ? 'Discovering...' : 'Discover Teams'}
          </PrimaryButton>
          
          {/* Primary action - Migrate Teams */}
          <SuccessButton
            onClick={() => executeDialog.open()}
            disabled={!hasMappedTeams || isRunning}
            leadingVisual={PlayIcon}
          >
            Migrate Teams
          </SuccessButton>
        </div>
      </div>

      {/* Import result notification */}
      {importResult && (
        <Flash variant="success">
          Import complete: {importResult.created} created, {importResult.updated} updated, {importResult.errors} errors
        </Flash>
      )}

      {/* Action result notification */}
      {actionResult && (
        <Flash variant={actionResult.type}>
          <span className="flex items-center justify-between">
            {actionResult.message}
            <Button
              variant="invisible"
              size="small"
              onClick={() => setActionResult(null)}
              className="ml-2"
            >
              Dismiss
            </Button>
          </span>
        </Flash>
      )}

      {/* Search and filters */}
      <div className="flex gap-4 items-center flex-wrap">
        <div className="flex-1 min-w-[200px] max-w-md">
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search by team name or slug..."
            value={search}
            onChange={handleSearch}
            className="w-full"
          />
        </div>
        
        {/* Source Org Filter - matches Users page order */}
        {sourceOrgs.length > 0 && (
          <ActionMenu onOpenChange={(open) => { if (!open) setOrgSearchFilter(''); }}>
            <ActionMenu.Anchor>
              <BorderedButton
                leadingVisual={OrganizationIcon}
                trailingAction={TriangleDownIcon}
              >
                Org: {filters.sourceOrg || 'All'}
              </BorderedButton>
            </ActionMenu.Anchor>
            <ActionMenu.Overlay>
              <div className="p-2" style={{ borderBottom: '1px solid var(--borderColor-muted)' }}>
                <TextInput
                  placeholder="Search organizations..."
                  value={orgSearchFilter}
                  onChange={(e) => setOrgSearchFilter(e.target.value)}
                  leadingVisual={SearchIcon}
                  size="small"
                  block
                  onClick={(e) => e.stopPropagation()}
                  onKeyDown={(e) => e.stopPropagation()}
                />
              </div>
              <ActionList selectionVariant="single" style={{ maxHeight: '300px', overflowY: 'auto' }}>
                {!orgSearchFilter && (
                  <>
                    <ActionList.Item selected={!filters.sourceOrg} onSelect={() => handleSourceOrgFilter('')}>
                      All Organizations
                    </ActionList.Item>
                    <ActionList.Divider />
                  </>
                )}
                {sourceOrgs
                  .filter(org => org.toLowerCase().includes(orgSearchFilter.toLowerCase()))
                  .map(org => (
                    <ActionList.Item
                      key={org}
                      selected={filters.sourceOrg === org}
                      onSelect={() => handleSourceOrgFilter(org)}
                    >
                      {org}
                    </ActionList.Item>
                  ))}
                {sourceOrgs.filter(org => org.toLowerCase().includes(orgSearchFilter.toLowerCase())).length === 0 && (
                  <ActionList.Item disabled>No matching organizations</ActionList.Item>
                )}
              </ActionList>
            </ActionMenu.Overlay>
          </ActionMenu>
        )}
        
        {/* Status Filter */}
        <ActionMenu>
          <ActionMenu.Anchor>
            <BorderedButton
              leadingVisual={FilterIcon}
              trailingAction={TriangleDownIcon}
            >
              Status: {filters.status ? statusLabels[filters.status as TeamMappingStatus] : 'All'}
            </BorderedButton>
          </ActionMenu.Anchor>
          <ActionMenu.Overlay>
            <ActionList selectionVariant="single">
              <ActionList.Item selected={!filters.status} onSelect={() => handleStatusFilter('')}>
                All
              </ActionList.Item>
              <ActionList.Divider />
              <ActionList.Item selected={filters.status === 'unmapped'} onSelect={() => handleStatusFilter('unmapped')}>
                Unmapped
              </ActionList.Item>
              <ActionList.Item selected={filters.status === 'mapped'} onSelect={() => handleStatusFilter('mapped')}>
                Mapped
              </ActionList.Item>
              <ActionList.Item selected={filters.status === 'skipped'} onSelect={() => handleStatusFilter('skipped')}>
                Skipped
              </ActionList.Item>
            </ActionList>
          </ActionMenu.Overlay>
        </ActionMenu>
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
              <th className="text-left p-3 font-medium">Sync Status</th>
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
                    <div 
                      className="flex items-center gap-2"
                      onClick={(e) => e.stopPropagation()}
                    >
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
                      <button
                        type="button"
                        onClick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          handleSaveEdit();
                        }}
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          padding: '4px 8px',
                          backgroundColor: 'var(--bgColor-success-emphasis)',
                          color: 'white',
                          border: 'none',
                          borderRadius: '6px',
                          cursor: 'pointer',
                        }}
                      >
                        <CheckIcon size={16} />
                      </button>
                      <button
                        type="button"
                        onClick={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          handleCancelEdit();
                        }}
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          padding: '4px 8px',
                          backgroundColor: 'transparent',
                          color: 'var(--fgColor-muted)',
                          border: 'none',
                          borderRadius: '6px',
                          cursor: 'pointer',
                        }}
                      >
                        <XIcon size={16} />
                      </button>
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
                <td className="p-3">
                  <div className="flex items-center gap-2">
                    <Label 
                      variant={syncStatusColors[mapping.sync_status] || 'default'}
                      title={syncStatusDescriptions[mapping.sync_status] || ''}
                    >
                      {syncStatusLabels[mapping.sync_status] || 'Pending'}
                    </Label>
                    {mapping.repos_eligible > 0 && (
                      <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                        {mapping.repos_synced}/{mapping.repos_eligible}
                      </span>
                    )}
                  </div>
                </td>
                <td className="p-3 pr-4 text-right w-24" onClick={(e) => e.stopPropagation()}>
                  <ActionMenu>
                    <ActionMenu.Button size="small" variant="invisible">
                      Actions
                    </ActionMenu.Button>
                    <ActionMenu.Overlay>
                      <ActionList>
                        <ActionList.Item onSelect={() => handleTeamClick(org, slug)}>
                          <ActionList.LeadingVisual>
                            <EyeIcon size={16} />
                          </ActionList.LeadingVisual>
                          View details
                        </ActionList.Item>
                        <ActionList.Item onSelect={() => handleEdit(mapping)}>
                          <ActionList.LeadingVisual>
                            <PencilIcon size={16} />
                          </ActionList.LeadingVisual>
                          Edit mapping
                        </ActionList.Item>
                        {mapping.mapping_status === 'mapped' && (
                          <ActionList.Item onSelect={() => handleMigrateSingleTeam(org, slug)}>
                            <ActionList.LeadingVisual>
                              <span style={{ color: 'var(--fgColor-success)' }}><RocketIcon size={16} /></span>
                            </ActionList.LeadingVisual>
                            <span style={{ color: 'var(--fgColor-success)' }}>Migrate team</span>
                          </ActionList.Item>
                        )}
                        {mapping.mapping_status !== 'skipped' && (
                          <ActionList.Item onSelect={() => handleSkip(org, slug)}>
                            <ActionList.LeadingVisual>
                              <SkipIcon size={16} />
                            </ActionList.LeadingVisual>
                            Skip team
                          </ActionList.Item>
                        )}
                        <ActionList.Divider />
                        <ActionList.Item onSelect={() => handleDelete(org, slug)}>
                          <ActionList.LeadingVisual>
                            <span style={{ color: 'var(--fgColor-danger)' }}><TrashIcon size={16} /></span>
                          </ActionList.LeadingVisual>
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
                <td colSpan={6} className="p-8 text-center">
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
      <FormDialog
        isOpen={executeDialog.isOpen}
        title="Migrate Teams"
        submitLabel={executeMigration.isPending ? 'Starting...' : dryRun ? 'Start Dry Run' : 'Start Migration'}
        onSubmit={handleExecute}
        onCancel={executeDialog.close}
        isLoading={executeMigration.isPending}
      >
        <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
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

        {filters.sourceOrg && (
          <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
            Only teams from <strong>{filters.sourceOrg}</strong> will be processed.
          </p>
        )}
      </FormDialog>

      {/* Single Team Migration Dialog */}
      {singleTeamDialog.data && (
        <ConfirmationDialog
          isOpen={singleTeamDialog.isOpen}
          title="Migrate Team"
          message={
            <>
              <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
                This will create the team <strong style={{ color: 'var(--fgColor-default)' }}>{singleTeamDialog.data.org}/{singleTeamDialog.data.slug}</strong> in the destination organization and apply repository permissions.
              </p>

              <Flash variant="warning">
                <div className="flex items-start gap-2">
                  <AlertIcon size={16} className="flex-shrink-0 mt-0.5" />
                  <div>
                    <strong>EMU/IdP Notice:</strong> The team will be created <strong>without members</strong>. 
                    If your destination is an EMU environment, manage team membership through your Identity Provider (IdP/SCIM).
                  </div>
                </div>
              </Flash>
            </>
          }
          confirmLabel="Migrate Team"
          onConfirm={handleConfirmSingleTeamMigration}
          onCancel={singleTeamDialog.close}
          isLoading={executeMigration.isPending}
        />
      )}

      {/* Discover Teams Dialog */}
      <FormDialog
        isOpen={discoverDialog.isOpen}
        title="Discover Teams"
        submitLabel={discoverTeams.isPending ? 'Discovering...' : 'Discover'}
        onSubmit={() => {
          if (!discoverOrg.trim()) return;
          discoverTeams.mutate(discoverOrg.trim(), {
            onSuccess: (data) => {
              setActionResult({ type: 'success', message: data.message || 'Discovery completed!' });
              discoverDialog.close();
              setDiscoverOrg('');
            },
            onError: (error) => {
              setActionResult({ type: 'danger', message: error instanceof Error ? error.message : 'Discovery failed' });
            },
          });
        }}
        onCancel={() => {
          discoverDialog.close();
          setDiscoverOrg('');
        }}
        isLoading={discoverTeams.isPending}
        isSubmitDisabled={!discoverOrg.trim()}
      >
        <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
          Discover all teams from a GitHub organization. This will fetch teams, their members, and create team mappings.
        </p>
        <FormControl>
          <FormControl.Label>Source Organization</FormControl.Label>
          <TextInput
            value={discoverOrg}
            onChange={(e) => setDiscoverOrg(e.target.value)}
            placeholder="e.g., my-org"
            block
          />
          <FormControl.Caption>
            Enter the source GitHub organization name
          </FormControl.Caption>
        </FormControl>
      </FormDialog>

      {/* Delete Mapping Dialog */}
      {deleteDialog.data && (
        <ConfirmationDialog
          isOpen={deleteDialog.isOpen}
          title="Delete Mapping"
          message={
            <>
              <p className="mb-3" style={{ color: 'var(--fgColor-default)' }}>
                Are you sure you want to delete the mapping for{' '}
                <strong>{deleteDialog.data.org}/{deleteDialog.data.slug}</strong>?
              </p>
              <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                This will remove the destination mapping. The source team data will be preserved.
              </p>
            </>
          }
          confirmLabel="Delete"
          variant="danger"
          onConfirm={handleConfirmDelete}
          onCancel={deleteDialog.close}
          isLoading={deleteMapping.isPending}
        />
      )}

      {/* Reset Migration Status Dialog */}
      <ConfirmationDialog
        isOpen={resetDialog.isOpen}
        title="Reset Migration Status"
        message={
          <>
            <p className="mb-3" style={{ color: 'var(--fgColor-default)' }}>
              Are you sure you want to reset all team migration statuses to pending?
            </p>
            <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              This will allow you to re-run the team migration. Teams that have already been created in the destination will be skipped, but repository permissions will be re-applied.
            </p>
            {filters.sourceOrg && (
              <p className="text-sm mt-3" style={{ color: 'var(--fgColor-accent)' }}>
                <strong>Note:</strong> Only teams from <strong>{filters.sourceOrg}</strong> will be reset.
              </p>
            )}
          </>
        }
        confirmLabel="Reset Status"
        variant="danger"
        onConfirm={handleConfirmReset}
        onCancel={resetDialog.close}
        isLoading={resetMigration.isPending}
      />
    </div>
  );
}
