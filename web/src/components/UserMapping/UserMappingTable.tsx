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
import { Blankslate } from '@primer/react/experimental';
import { BorderedButton, SuccessButton, PrimaryButton } from '../common/buttons';
import {
  PersonIcon,
  CheckIcon,
  XIcon,
  SearchIcon,
  DownloadIcon,
  UploadIcon,
  LinkIcon,
  SyncIcon,
  MailIcon,
  OrganizationIcon,
  FilterIcon,
  TriangleDownIcon,
  PencilIcon,
  SkipIcon,
  TrashIcon,
} from '@primer/octicons-react';
import { useUserMappings, useUserMappingStats, useUserMappingSourceOrgs } from '../../hooks/useQueries';
import {
  useUpdateUserMapping,
  useDeleteUserMapping,
  useFetchMannequins,
  useSendAttributionInvitation,
  useBulkSendAttributionInvitations,
  useDiscoverOrgMembers,
} from '../../hooks/useMutations';
import { useTableState } from '../../hooks/useTableState';
import { useDialogState } from '../../hooks/useDialogState';
import { UserMapping, UserMappingStatus, ReclaimStatus } from '../../types';
import { api } from '../../services/api';
import { Pagination } from '../common/Pagination';
import { ConfirmationDialog } from '../common/ConfirmationDialog';
import { FormDialog } from '../common/FormDialog';
import { FallbackAvatar } from '../common/FallbackAvatar';
import { SourceTypeIcon } from '../common/SourceBadge';
import { DiscoverySourceSelector } from '../common/DiscoverySourceSelector';
import { useSourceSelection } from '../../hooks/useSourceSelection';
import { UserDetailPanel } from './UserDetailPanel';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';
import { useSourceContext } from '../../contexts/SourceContext';

const ITEMS_PER_PAGE = 25;

const statusColors: Record<UserMappingStatus, 'default' | 'accent' | 'success' | 'attention' | 'danger' | 'done'> = {
  unmapped: 'attention',
  mapped: 'accent',
  reclaimed: 'success',
  skipped: 'default',
};

const statusLabels: Record<UserMappingStatus, string> = {
  unmapped: 'Unmapped',
  mapped: 'Mapped',
  reclaimed: 'Reclaimed',
  skipped: 'Skipped',
};

const reclaimStatusColors: Record<ReclaimStatus, 'default' | 'accent' | 'success' | 'attention' | 'danger'> = {
  pending: 'attention',
  invited: 'accent',
  completed: 'success',
  failed: 'danger',
};

const matchReasonLabels: Record<string, string> = {
  login_exact: 'Login Match',
  login_emu: 'EMU Login Match',
  email_exact: 'Email Match',
  name_fuzzy: 'Name Match',
  // Legacy match reasons from previous implementation
  email_local_exact: 'Email Username',
  email_local_contains: 'Email Partial',
  name_login_exact: 'Name = Login',
  login_contains: 'Partial Login',
  name_contains_login: 'Name Contains Login',
};

interface UserMappingFilters extends Record<string, unknown> {
  status: string;
  sourceOrg: string;
}

export function UserMappingTable() {
  const { showError, showSuccess } = useToast();
  const { activeSource } = useSourceContext();
  const { isAllSourcesMode } = useSourceSelection();
  
  // Use shared table state hook for pagination, search, and filtering
  const { page, search, filters, setPage, setSearch, updateFilter, offset, limit } = useTableState<UserMappingFilters>({
    initialFilters: { status: '', sourceOrg: '' },
    pageSize: ITEMS_PER_PAGE,
  });
  
  const [orgSearchFilter, setOrgSearchFilter] = useState('');
  const [editingMapping, setEditingMapping] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [selectedUser, setSelectedUser] = useState<string | null>(null);
  
  // Dialog state using shared hooks
  const deleteDialog = useDialogState<string>(); // stores the source login to delete
  const discoverDialog = useDialogState();
  const destOrgDialog = useDialogState<{ action: 'fetch' | 'invite' | 'bulk_invite' | 'generate_gei'; sourceLogin?: string }>();
  
  // Form state for dialogs
  const [destinationOrg, setDestinationOrg] = useState('');
  const [emuShortcode, setEmuShortcode] = useState('');
  const [discoverOrg, setDiscoverOrg] = useState('');
  const [discoverSourceId, setDiscoverSourceId] = useState<number | null>(null);
  
  

  const { data, isLoading, error, refetch } = useUserMappings({
    search: search || undefined,
    status: filters.status || undefined,
    source_org: filters.sourceOrg || undefined,
    source_id: activeSource?.id,
    limit,
    offset,
  });

  const { data: stats } = useUserMappingStats(filters.sourceOrg || undefined, activeSource?.id);
  const { data: sourceOrgsData } = useUserMappingSourceOrgs();
  const updateMapping = useUpdateUserMapping();
  const deleteMapping = useDeleteUserMapping();
  const fetchMannequins = useFetchMannequins();
  const sendInvitation = useSendAttributionInvitation();
  const bulkSendInvitations = useBulkSendAttributionInvitations();
  const discoverOrgMembers = useDiscoverOrgMembers();

  const mappings = data?.mappings || [];
  const total = data?.total || 0;
  const sourceOrgs = sourceOrgsData?.organizations || [];

  const handleSearch = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
  }, [setSearch]);

  const handleStatusFilter = useCallback((status: string) => {
    updateFilter('status', status);
  }, [updateFilter]);

  const handleSourceOrgFilter = useCallback((org: string) => {
    updateFilter('sourceOrg', org);
  }, [updateFilter]);

  // Helper to get fields from either new or legacy field names
  const getLogin = (mapping: UserMapping) => mapping.login || mapping.source_login || '';
  const getEmail = (mapping: UserMapping) => mapping.email || mapping.source_email;

  const handleEdit = useCallback((mapping: UserMapping) => {
    setEditingMapping(getLogin(mapping));
    setEditValue(mapping.destination_login || '');
  }, []);

  const handleSaveEdit = useCallback(async () => {
    if (!editingMapping) return;
    
    try {
      await updateMapping.mutateAsync({
        sourceLogin: editingMapping,
        updates: { 
          destination_login: editValue || undefined,
          mapping_status: editValue ? 'mapped' : 'unmapped',
        },
      });
      setEditingMapping(null);
      setEditValue('');
    } catch {
      // Update failed, mutation will show error
    }
  }, [editingMapping, editValue, updateMapping]);

  const handleCancelEdit = useCallback(() => {
    setEditingMapping(null);
    setEditValue('');
  }, []);

  const handleDelete = useCallback((sourceLogin: string) => {
    deleteDialog.open(sourceLogin);
  }, [deleteDialog]);

  const handleConfirmDelete = useCallback(async () => {
    if (!deleteDialog.data) return;
    try {
      await deleteMapping.mutateAsync(deleteDialog.data);
      deleteDialog.close();
    } catch {
      // Delete failed, mutation will show error
    }
  }, [deleteMapping, deleteDialog]);

  const handleSkip = useCallback(async (sourceLogin: string) => {
    try {
      await updateMapping.mutateAsync({
        sourceLogin,
        updates: { mapping_status: 'skipped' as const },
      });
    } catch {
      // Skip failed, mutation will show error
    }
  }, [updateMapping]);

  const handleExport = useCallback(async () => {
    try {
      const blob = await api.exportUserMappings(filters.status || undefined, activeSource?.id);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const sourceSuffix = activeSource ? `_${activeSource.name.replace(/\s+/g, '_')}` : '';
      a.download = `user-mappings${sourceSuffix}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      handleApiError(error, showError, 'Failed to export mappings');
    }
  }, [filters.status, activeSource, showError]);

  const handleImport = useCallback(async (file?: File) => {
    if (!file) return;
    
    try {
      const result = await api.importUserMappings(file);
      showSuccess(`Import complete: ${result.created} created, ${result.updated} updated, ${result.errors} errors`);
      refetch();
    } catch (error) {
      handleApiError(error, showError, 'Failed to import mappings');
    }
  }, [refetch, showError, showSuccess]);

  const handleGenerateGEI = useCallback(() => {
    destOrgDialog.open({ action: 'generate_gei' });
  }, [destOrgDialog]);

  // Action handlers that require destination org
  const openDestOrgDialog = useCallback((action: 'fetch' | 'invite' | 'bulk_invite' | 'generate_gei', sourceLogin?: string) => {
    destOrgDialog.open({ action, sourceLogin });
  }, [destOrgDialog]);

  const handleConfirmDestOrg = useCallback(async () => {
    if (!destinationOrg || !destOrgDialog.data) return;
    
    const { action, sourceLogin } = destOrgDialog.data;
    destOrgDialog.close();
    
    try {
      if (action === 'fetch') {
        // Fetch mannequins and match to destination org members
        const result = await fetchMannequins.mutateAsync({
          destinationOrg,
          emuShortcode: emuShortcode || undefined,
        });
        showSuccess(result.message);
      } else if (action === 'invite' && sourceLogin) {
        const result = await sendInvitation.mutateAsync({
          sourceLogin,
          destinationOrg,
        });
        if (result.success) {
          showSuccess(result.message);
        } else {
          showError(result.message);
        }
      } else if (action === 'bulk_invite') {
        const result = await bulkSendInvitations.mutateAsync({
          destinationOrg,
        });
        if (result.success) {
          showSuccess(result.message);
        } else {
          showError(result.message);
        }
      } else if (action === 'generate_gei') {
        // Generate GEI CSV filtered by destination org
        const blob = await api.generateGEICSV(undefined, destinationOrg);
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `mannequin-mappings-${destinationOrg}.csv`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        showSuccess(`GEI CSV generated for ${destinationOrg}`);
      }
    } catch (err) {
      showError(`Action failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
    
    setEmuShortcode('');
  }, [destinationOrg, emuShortcode, destOrgDialog, fetchMannequins, sendInvitation, bulkSendInvitations, showSuccess, showError]);

  const cancelDestOrgDialog = useCallback(() => {
    destOrgDialog.close();
    setEmuShortcode('');
  }, [destOrgDialog]);

  // Get invitable count from stats (not from paginated mappings)
  const invitableCount = stats?.invitable || 0;

  if (error) {
    return (
      <Flash variant="danger">
        Failed to load user mappings: {error.message}
      </Flash>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Header with stats */}
      <div className="flex justify-between items-start flex-wrap gap-4">
        <div>
          <div className="flex items-center gap-4 flex-wrap">
            <h2 className="text-2xl font-semibold flex items-center gap-2" style={{ color: 'var(--fgColor-default)' }}>
              <PersonIcon size={24} />
              User Identity Mapping
            </h2>
            {stats && (
              <div className="flex gap-2 flex-wrap">
                <Label variant="accent">{stats.total} Total</Label>
                <Label variant="success">{stats.mapped} Mapped</Label>
                <Label variant="attention">{stats.unmapped} Unmapped</Label>
                {stats.reclaimed > 0 && <Label variant="done">{stats.reclaimed} Reclaimed</Label>}
              </div>
            )}
          </div>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Map source identities to destination GitHub users for attribution
          </p>
        </div>
        <div className="flex gap-2 flex-wrap">
          {/* Data Management - Import/Export */}
          <input
            type="file"
            id="import-csv-input"
            accept=".csv"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) handleImport(file);
              e.target.value = '';
            }}
            className="hidden"
          />
          <BorderedButton
            onClick={() => document.getElementById('import-csv-input')?.click()}
            leadingVisual={UploadIcon}
          >
            Import
          </BorderedButton>
          <ActionMenu>
            <ActionMenu.Anchor>
              <BorderedButton
                leadingVisual={DownloadIcon}
                trailingAction={TriangleDownIcon}
              >
                Export
              </BorderedButton>
            </ActionMenu.Anchor>
            <ActionMenu.Overlay>
              <ActionList>
                <ActionList.Item onSelect={handleExport}>
                  Export All Mappings (CSV)
                </ActionList.Item>
                <ActionList.Item onSelect={handleGenerateGEI}>
                  Generate GEI Reclaim CSV
                </ActionList.Item>
              </ActionList>
            </ActionMenu.Overlay>
          </ActionMenu>
          
          {/* Discovery/Setup actions */}
          <PrimaryButton
            onClick={() => discoverDialog.open()}
            leadingVisual={PersonIcon}
            disabled={discoverOrgMembers.isPending}
          >
            {discoverOrgMembers.isPending ? 'Discovering...' : 'Discover Org Members'}
          </PrimaryButton>
          <PrimaryButton
            onClick={() => openDestOrgDialog('fetch')}
            leadingVisual={SyncIcon}
            disabled={fetchMannequins.isPending}
          >
            {fetchMannequins.isPending ? 'Fetching...' : 'Fetch Mannequins'}
          </PrimaryButton>
          
          {/* Primary action - Send Invitations */}
          {invitableCount > 0 && (
            <SuccessButton
              onClick={() => openDestOrgDialog('bulk_invite')}
              leadingVisual={MailIcon}
              disabled={bulkSendInvitations.isPending}
            >
              {bulkSendInvitations.isPending ? 'Sending...' : `Send ${invitableCount} Invitation${invitableCount !== 1 ? 's' : ''}`}
            </SuccessButton>
          )}
        </div>
      </div>

      {/* Search and filters */}
      <div className="flex gap-4 items-center flex-wrap">
        <div className="flex-1 min-w-[200px] max-w-md">
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search by login, email, or name..."
            value={search}
            onChange={handleSearch}
            className="w-full"
          />
        </div>
        
        {/* Source Org Filter */}
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
                  .filter((org) => org.toLowerCase().includes(orgSearchFilter.toLowerCase()))
                  .map((org) => (
                    <ActionList.Item 
                      key={org} 
                      selected={filters.sourceOrg === org} 
                      onSelect={() => handleSourceOrgFilter(org)}
                    >
                      {org}
                    </ActionList.Item>
                  ))}
                {sourceOrgs.filter((org) => org.toLowerCase().includes(orgSearchFilter.toLowerCase())).length === 0 && (
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
              Status: {filters.status ? statusLabels[filters.status as UserMappingStatus] : 'All'}
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
              <ActionList.Item selected={filters.status === 'reclaimed'} onSelect={() => handleStatusFilter('reclaimed')}>
                Reclaimed
              </ActionList.Item>
              <ActionList.Item selected={filters.status === 'skipped'} onSelect={() => handleStatusFilter('skipped')}>
                Skipped
              </ActionList.Item>
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
        <div className="overflow-x-auto">
          <table className="w-full" style={{ borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
                <th className="text-left p-3 font-medium" style={{ color: 'var(--fgColor-muted)' }}>Source User</th>
                <th className="text-left p-3 font-medium" style={{ color: 'var(--fgColor-muted)' }}>Destination User</th>
                <th className="text-left p-3 font-medium" style={{ color: 'var(--fgColor-muted)' }}>Status</th>
                <th className="text-left p-3 font-medium" style={{ color: 'var(--fgColor-muted)' }}>Mannequin</th>
                <th className="text-left p-3 font-medium" style={{ color: 'var(--fgColor-muted)' }}>Match</th>
                <th className="text-right p-3 font-medium" style={{ color: 'var(--fgColor-muted)' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {mappings.map((mapping) => {
                const login = getLogin(mapping);
                const email = getEmail(mapping);
                const canSendInvitation = mapping.mapping_status === 'mapped' && 
                  mapping.mannequin_id && 
                  mapping.destination_login &&
                  (!mapping.reclaim_status || mapping.reclaim_status === 'pending' || mapping.reclaim_status === 'failed');
                
                return (
                  <tr
                    key={login}
                    style={{ 
                      borderBottom: '1px solid var(--borderColor-muted)',
                      cursor: 'pointer',
                      backgroundColor: selectedUser === login ? 'var(--bgColor-accent-muted)' : 'transparent',
                    }}
                    className="hover:opacity-80"
                    onClick={() => setSelectedUser(login)}
                  >
                    <td className="p-3">
                      <div className="flex items-center gap-2">
                        {mapping.source_id && (
                          <SourceTypeIcon sourceId={mapping.source_id} size={14} />
                        )}
                        <FallbackAvatar src={mapping.avatar_url} login={login} size={24} />
                        <div>
                          <div className="font-medium" style={{ color: 'var(--fgColor-default)' }}>{login}</div>
                          {email && (
                            <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                              {email}
                            </span>
                          )}
                          {mapping.source_org && (
                            <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                              <OrganizationIcon size={12} className="mr-1" />
                              {mapping.source_org}
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="p-3">
                      {editingMapping === login ? (
                        <div 
                          className="flex items-center gap-2" 
                          onClick={(e) => e.stopPropagation()}
                        >
                          <TextInput
                            value={editValue}
                            onChange={(e) => setEditValue(e.target.value)}
                            placeholder="destination-username"
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
                        <div className="flex items-center gap-2">
                          {mapping.destination_login ? (
                            <>
                              <FallbackAvatar login={mapping.destination_login} size={24} />
                              <span style={{ color: 'var(--fgColor-default)' }}>{mapping.destination_login}</span>
                              <span style={{ color: 'var(--fgColor-muted)' }}><LinkIcon size={16} /></span>
                            </>
                          ) : (
                            <span style={{ color: 'var(--fgColor-muted)' }}>Not mapped</span>
                          )}
                        </div>
                      )}
                    </td>
                    <td className="p-3">
                      <div className="flex flex-col gap-1">
                        <Label variant={statusColors[mapping.mapping_status as UserMappingStatus]}>
                          {statusLabels[mapping.mapping_status as UserMappingStatus]}
                        </Label>
                        {mapping.reclaim_status && (
                          <Label variant={reclaimStatusColors[mapping.reclaim_status]}>
                            {mapping.reclaim_status}
                          </Label>
                        )}
                      </div>
                    </td>
                    <td className="p-3">
                      {mapping.mannequin_login ? (
                        <div>
                          <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>
                            {mapping.mannequin_login}
                          </span>
                        </div>
                      ) : (
                        <span style={{ color: 'var(--fgColor-muted)' }}>—</span>
                      )}
                    </td>
                    <td className="p-3">
                      {mapping.match_confidence && mapping.match_reason ? (
                        <div className="flex flex-col gap-1">
                          <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>
                            {mapping.match_confidence}%
                          </span>
                          <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                            {matchReasonLabels[mapping.match_reason] || mapping.match_reason}
                          </span>
                        </div>
                      ) : (
                        <span style={{ color: 'var(--fgColor-muted)' }}>—</span>
                      )}
                    </td>
                    <td className="p-3 text-right" onClick={(e) => e.stopPropagation()}>
                      <ActionMenu>
                        <ActionMenu.Button size="small" variant="invisible">
                          Actions
                        </ActionMenu.Button>
                        <ActionMenu.Overlay>
                          <ActionList>
                            <ActionList.Item onSelect={() => handleEdit(mapping)}>
                              <ActionList.LeadingVisual>
                                <PencilIcon size={16} />
                              </ActionList.LeadingVisual>
                              Edit mapping
                            </ActionList.Item>
                            {mapping.mapping_status !== 'skipped' && (
                              <ActionList.Item onSelect={() => handleSkip(login)}>
                                <ActionList.LeadingVisual>
                                  <SkipIcon size={16} />
                                </ActionList.LeadingVisual>
                                Skip user
                              </ActionList.Item>
                            )}
                            {canSendInvitation && (
                              <ActionList.Item onSelect={() => openDestOrgDialog('invite', login)}>
                                <ActionList.LeadingVisual>
                                  <MailIcon size={16} />
                                </ActionList.LeadingVisual>
                                Send invitation
                              </ActionList.Item>
                            )}
                            <ActionList.Divider />
                            <ActionList.Item onSelect={() => handleDelete(login)}>
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
                  <td colSpan={6} className="p-8">
                    <Blankslate>
                      <Blankslate.Visual>
                        <PersonIcon size={48} />
                      </Blankslate.Visual>
                      <Blankslate.Heading>No users found</Blankslate.Heading>
                      <Blankslate.Description>
                        {search || filters.status || filters.sourceOrg
                          ? 'Try adjusting your search or filters to find users.'
                          : 'No users have been discovered yet. Start by discovering organization members.'}
                      </Blankslate.Description>
                      {!search && !filters.status && !filters.sourceOrg && (
                        <Blankslate.PrimaryAction onClick={() => discoverDialog.open()}>
                          Discover Org Members
                        </Blankslate.PrimaryAction>
                      )}
                    </Blankslate>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
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

      {/* Discover Org Members Dialog */}
      <FormDialog
        isOpen={discoverDialog.isOpen}
        title="Discover Organization Members"
        submitLabel={discoverOrgMembers.isPending ? 'Discovering...' : 'Discover'}
        onSubmit={() => {
          // Validate source selection in All Sources mode
          if (isAllSourcesMode && !discoverSourceId) {
            showError('Please select a source');
            return;
          }
          if (!discoverOrg.trim()) return;
          
          discoverOrgMembers.mutate({ 
            organization: discoverOrg.trim(),
            source_id: discoverSourceId ?? activeSource?.id,
          }, {
            onSuccess: (data) => {
              showSuccess(data.message || 'Discovery completed!');
              discoverDialog.close();
              setDiscoverOrg('');
              setDiscoverSourceId(null);
            },
            onError: (error) => {
              showError(error instanceof Error ? error.message : 'Discovery failed');
            },
          });
        }}
        onCancel={() => {
          discoverDialog.close();
          setDiscoverOrg('');
          setDiscoverSourceId(null);
        }}
        isLoading={discoverOrgMembers.isPending}
        isSubmitDisabled={!discoverOrg.trim() || (isAllSourcesMode && !discoverSourceId)}
      >
        <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
          Discover all members from an organization. This will fetch org members and create user mappings for mannequin matching.
        </p>
        
        {/* Source Selection */}
        <DiscoverySourceSelector
          selectedSourceId={discoverSourceId}
          onSourceChange={(sourceId, source) => {
            setDiscoverSourceId(sourceId);
            // Pre-populate organization from source config
            if (source?.organization) {
              setDiscoverOrg(source.organization);
            } else {
              setDiscoverOrg('');
            }
          }}
          required={isAllSourcesMode}
          disabled={discoverOrgMembers.isPending}
          label="Select Source"
          defaultCaption="Select which source to discover members from."
        />
        
        {/* Only show organization field when source is selected (in All Sources mode) or always (in single source mode) */}
        {(!isAllSourcesMode || discoverSourceId) && (
          <FormControl>
            <FormControl.Label>Source Organization</FormControl.Label>
            <TextInput
              value={discoverOrg}
              onChange={(e) => setDiscoverOrg(e.target.value)}
              placeholder="e.g., my-org"
              block
            />
            <FormControl.Caption>
              Enter the source organization name to discover members from
            </FormControl.Caption>
          </FormControl>
        )}
        
        {/* Prompt to select a source first in All Sources mode */}
        {isAllSourcesMode && !discoverSourceId && (
          <div className="text-center py-4" style={{ color: 'var(--fgColor-muted)' }}>
            <p className="text-sm">Please select a source to configure discovery options.</p>
          </div>
        )}
      </FormDialog>

      {/* Destination Org Dialog */}
      {destOrgDialog.data && (
        <FormDialog
          isOpen={destOrgDialog.isOpen}
          title={
            destOrgDialog.data.action === 'fetch' ? 'Fetch Mannequins' :
            destOrgDialog.data.action === 'invite' ? 'Send Attribution Invitation' :
            destOrgDialog.data.action === 'generate_gei' ? 'Generate GEI Reclaim CSV' :
            'Send Attribution Invitations'
          }
          submitLabel={
            destOrgDialog.data.action === 'fetch' ? 'Fetch' :
            destOrgDialog.data.action === 'invite' ? 'Send Invitation' :
            destOrgDialog.data.action === 'generate_gei' ? 'Generate CSV' :
            'Send All'
          }
          onSubmit={handleConfirmDestOrg}
          onCancel={cancelDestOrgDialog}
          isSubmitDisabled={!destinationOrg}
        >
          <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
            {destOrgDialog.data.action === 'fetch' && 
              'Enter the destination GitHub organization to fetch mannequins from. Mannequins will be matched to destination org members using login, email, and name.'}
            {destOrgDialog.data.action === 'invite' && 
              `Send an attribution invitation for ${destOrgDialog.data.sourceLogin} to reclaim their mannequin.`}
            {destOrgDialog.data.action === 'bulk_invite' && 
              `Send attribution invitations to all ${invitableCount} mapped users with mannequins.`}
            {destOrgDialog.data.action === 'generate_gei' && 
              'Enter the destination organization to generate a GEI-compatible CSV for mannequin reclaim. The CSV will only include mannequins from this organization.'}
          </p>
          <FormControl>
            <FormControl.Label>Destination Organization</FormControl.Label>
            <TextInput
              value={destinationOrg}
              onChange={(e) => setDestinationOrg(e.target.value)}
              placeholder="e.g., my-org"
              block
            />
            <FormControl.Caption>
              The GitHub organization where migrations were imported
            </FormControl.Caption>
          </FormControl>
          
          {/* EMU Shortcode - only show for fetch action */}
          {destOrgDialog.data.action === 'fetch' && (
            <FormControl className="mt-4">
              <FormControl.Label>EMU Shortcode (Optional)</FormControl.Label>
              <TextInput
                value={emuShortcode}
                onChange={(e) => setEmuShortcode(e.target.value)}
                placeholder="e.g., fabrikam"
                block
              />
              <FormControl.Caption>
                For EMU migrations where usernames differ. If source is "jsmith" and destination is "jsmith_fabrikam", enter "fabrikam".
              </FormControl.Caption>
            </FormControl>
          )}
        </FormDialog>
      )}

      {/* Delete Mapping Dialog */}
      {deleteDialog.data && (
        <ConfirmationDialog
          isOpen={deleteDialog.isOpen}
          title="Delete Mapping"
          message={
            <>
              <p className="mb-3" style={{ color: 'var(--fgColor-default)' }}>
                Are you sure you want to delete the mapping for <strong>{deleteDialog.data}</strong>?
              </p>
              <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                This will remove the destination mapping. The source user data will be preserved.
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

      {/* User Detail Panel */}
      {selectedUser && (
        <UserDetailPanel
          login={selectedUser}
          onClose={() => setSelectedUser(null)}
          onEditMapping={(login) => {
            // Find the mapping and trigger edit
            const mapping = mappings.find(m => (m.login || m.source_login) === login);
            if (mapping) {
              handleEdit(mapping);
            }
            setSelectedUser(null);
          }}
        />
      )}
    </div>
  );
}
