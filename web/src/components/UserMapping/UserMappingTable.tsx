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
  FormControl,
} from '@primer/react';
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
import { UserMapping, UserMappingStatus, ReclaimStatus } from '../../types';
import { api } from '../../services/api';
import { Pagination } from '../common/Pagination';
import { FallbackAvatar } from '../common/FallbackAvatar';
import { UserDetailPanel } from './UserDetailPanel';

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

export function UserMappingTable() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [sourceOrgFilter, setSourceOrgFilter] = useState<string>('');
  const [orgSearchFilter, setOrgSearchFilter] = useState('');
  const [page, setPage] = useState(1);
  const [editingMapping, setEditingMapping] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [importResult, setImportResult] = useState<{ created: number; updated: number; errors: number } | null>(null);
  const [selectedUser, setSelectedUser] = useState<string | null>(null);
  
  // Destination org dialog state
  const [showDestOrgDialog, setShowDestOrgDialog] = useState(false);
  const [pendingAction, setPendingAction] = useState<'fetch' | 'invite' | 'bulk_invite' | null>(null);
  const [pendingSourceLogin, setPendingSourceLogin] = useState<string | null>(null);
  const [destinationOrg, setDestinationOrg] = useState('');
  const [emuShortcode, setEmuShortcode] = useState('');
  
  // Discover dialog state
  const [showDiscoverDialog, setShowDiscoverDialog] = useState(false);
  const [discoverOrg, setDiscoverOrg] = useState('');
  
  // Delete dialog state
  const [showDeleteDialog, setShowDeleteDialog] = useState<string | null>(null);
  
  // Action result state
  const [actionResult, setActionResult] = useState<{ type: 'success' | 'danger'; message: string } | null>(null);

  const { data, isLoading, error, refetch } = useUserMappings({
    search: search || undefined,
    status: statusFilter || undefined,
    source_org: sourceOrgFilter || undefined,
    limit: ITEMS_PER_PAGE,
    offset: (page - 1) * ITEMS_PER_PAGE,
  });

  const { data: stats } = useUserMappingStats(sourceOrgFilter || undefined);
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
    } catch (err) {
      console.error('Failed to update mapping:', err);
    }
  }, [editingMapping, editValue, updateMapping]);

  const handleCancelEdit = useCallback(() => {
    setEditingMapping(null);
    setEditValue('');
  }, []);

  const handleDelete = useCallback((sourceLogin: string) => {
    setShowDeleteDialog(sourceLogin);
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!showDeleteDialog) return;
    try {
      await deleteMapping.mutateAsync(showDeleteDialog);
      setShowDeleteDialog(null);
    } catch (err) {
      console.error('Failed to delete mapping:', err);
    }
  }, [deleteMapping, showDeleteDialog]);

  const handleSkip = useCallback(async (sourceLogin: string) => {
    try {
      await updateMapping.mutateAsync({
        sourceLogin,
        updates: { mapping_status: 'skipped' as const },
      });
    } catch (err) {
      console.error('Failed to skip mapping:', err);
    }
  }, [updateMapping]);

  const handleExport = useCallback(async () => {
    try {
      const blob = await api.exportUserMappings(statusFilter || undefined);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'user-mappings.csv';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      console.error('Failed to export mappings:', err);
    }
  }, [statusFilter]);

  const handleImport = useCallback(async (file?: File) => {
    if (!file) return;
    
    try {
      const result = await api.importUserMappings(file);
      setImportResult(result);
      refetch();
      setTimeout(() => setImportResult(null), 5000);
    } catch (err) {
      console.error('Failed to import mappings:', err);
    }
  }, [refetch]);

  const handleGenerateGEI = useCallback(async () => {
    try {
      const blob = await api.generateGEICSV();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'mannequin-mappings.csv';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      console.error('Failed to generate GEI CSV:', err);
    }
  }, []);

  // Action handlers that require destination org
  const openDestOrgDialog = useCallback((action: 'fetch' | 'invite' | 'bulk_invite', sourceLogin?: string) => {
    setPendingAction(action);
    setPendingSourceLogin(sourceLogin || null);
    setShowDestOrgDialog(true);
  }, []);

  const handleConfirmDestOrg = useCallback(async () => {
    if (!destinationOrg) return;
    
    setShowDestOrgDialog(false);
    
    try {
      if (pendingAction === 'fetch') {
        // Fetch mannequins and match to destination org members
        const result = await fetchMannequins.mutateAsync({
          destinationOrg,
          emuShortcode: emuShortcode || undefined,
        });
        setActionResult({
          type: 'success',
          message: result.message,
        });
      } else if (pendingAction === 'invite' && pendingSourceLogin) {
        const result = await sendInvitation.mutateAsync({
          sourceLogin: pendingSourceLogin,
          destinationOrg,
        });
        setActionResult({
          type: result.success ? 'success' : 'danger',
          message: result.message,
        });
      } else if (pendingAction === 'bulk_invite') {
        const result = await bulkSendInvitations.mutateAsync({
          destinationOrg,
        });
        setActionResult({
          type: result.success ? 'success' : 'danger',
          message: result.message,
        });
      }
    } catch (err) {
      setActionResult({
        type: 'danger',
        message: `Action failed: ${err instanceof Error ? err.message : 'Unknown error'}`,
      });
    }
    
    setPendingAction(null);
    setPendingSourceLogin(null);
    setEmuShortcode('');
    setTimeout(() => setActionResult(null), 8000);
  }, [destinationOrg, emuShortcode, pendingAction, pendingSourceLogin, fetchMannequins, sendInvitation, bulkSendInvitations]);

  const cancelDestOrgDialog = useCallback(() => {
    setShowDestOrgDialog(false);
    setPendingAction(null);
    setPendingSourceLogin(null);
    setEmuShortcode('');
  }, []);

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
          {/* Discovery action */}
          <Button
            variant="invisible"
            onClick={() => setShowDiscoverDialog(true)}
            leadingVisual={PersonIcon}
            disabled={discoverOrgMembers.isPending}
            className="btn-bordered-invisible"
          >
            {discoverOrgMembers.isPending ? 'Discovering...' : 'Discover Org Members'}
          </Button>
          
          {/* Mannequin fetch */}
          <Button
            variant="invisible"
            onClick={() => openDestOrgDialog('fetch')}
            leadingVisual={SyncIcon}
            disabled={fetchMannequins.isPending}
            className="btn-bordered-invisible"
          >
            {fetchMannequins.isPending ? 'Fetching...' : 'Fetch Mannequins'}
          </Button>
          
          {/* Import/Export */}
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
          <Button
            variant="invisible"
            onClick={() => document.getElementById('import-csv-input')?.click()}
            leadingVisual={UploadIcon}
            className="btn-bordered-invisible"
          >
            Import
          </Button>
          <ActionMenu>
            <ActionMenu.Anchor>
              <Button
                variant="invisible"
                leadingVisual={DownloadIcon}
                trailingAction={TriangleDownIcon}
                className="btn-bordered-invisible"
              >
                Export
              </Button>
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
          
          {/* Primary action - Send Invitations (matches Teams page pattern) */}
          {invitableCount > 0 && (
            <Button
              onClick={() => openDestOrgDialog('bulk_invite')}
              leadingVisual={MailIcon}
              variant="primary"
              disabled={bulkSendInvitations.isPending}
            >
              {bulkSendInvitations.isPending ? 'Sending...' : `Send ${invitableCount} Invitation${invitableCount !== 1 ? 's' : ''}`}
            </Button>
          )}
        </div>
      </div>

      {/* Action result notification */}
      {actionResult && (
        <Flash variant={actionResult.type}>
          {actionResult.message}
        </Flash>
      )}

      {/* Import result notification */}
      {importResult && (
        <Flash variant="success">
          Import complete: {importResult.created} created, {importResult.updated} updated, {importResult.errors} errors
        </Flash>
      )}

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
              <Button
                variant="invisible"
                leadingVisual={OrganizationIcon}
                trailingAction={TriangleDownIcon}
                className="btn-bordered-invisible"
              >
                Org: {sourceOrgFilter || 'All'}
              </Button>
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
                    <ActionList.Item selected={!sourceOrgFilter} onSelect={() => handleSourceOrgFilter('')}>
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
                      selected={sourceOrgFilter === org} 
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
            <Button
              variant="invisible"
              leadingVisual={FilterIcon}
              trailingAction={TriangleDownIcon}
              className="btn-bordered-invisible"
            >
              Status: {statusFilter ? statusLabels[statusFilter as UserMappingStatus] : 'All'}
            </Button>
          </ActionMenu.Anchor>
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
              <ActionList.Item selected={statusFilter === 'reclaimed'} onSelect={() => handleStatusFilter('reclaimed')}>
                Reclaimed
              </ActionList.Item>
              <ActionList.Item selected={statusFilter === 'skipped'} onSelect={() => handleStatusFilter('skipped')}>
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
                  <td colSpan={6} className="p-8 text-center">
                    <span style={{ color: 'var(--fgColor-muted)' }}>
                      No users found. Run discovery to discover organization members.
                    </span>
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
      {showDiscoverDialog && (
        <Dialog
          title="Discover Organization Members"
          onClose={() => {
            setShowDiscoverDialog(false);
            setDiscoverOrg('');
          }}
        >
          <div className="p-4">
            <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
              Discover all members from a GitHub organization. This will fetch org members and create user mappings for mannequin matching.
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
          </div>
          <div className="flex justify-end gap-2 p-4 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
            <Button onClick={() => {
              setShowDiscoverDialog(false);
              setDiscoverOrg('');
            }}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                if (!discoverOrg.trim()) return;
                discoverOrgMembers.mutate(discoverOrg.trim(), {
                  onSuccess: (data) => {
                    setActionResult({ type: 'success', message: data.message || 'Discovery completed!' });
                    setShowDiscoverDialog(false);
                    setDiscoverOrg('');
                  },
                  onError: (error) => {
                    setActionResult({ type: 'danger', message: error instanceof Error ? error.message : 'Discovery failed' });
                  },
                });
              }}
              disabled={discoverOrgMembers.isPending || !discoverOrg.trim()}
            >
              {discoverOrgMembers.isPending ? 'Discovering...' : 'Discover'}
            </Button>
          </div>
        </Dialog>
      )}

      {/* Destination Org Dialog */}
      {showDestOrgDialog && (
        <Dialog
          title={
            pendingAction === 'fetch' ? 'Fetch Mannequins' :
            pendingAction === 'invite' ? 'Send Attribution Invitation' :
            'Send Attribution Invitations'
          }
          onClose={cancelDestOrgDialog}
        >
          <div className="p-4">
            <p className="mb-4" style={{ color: 'var(--fgColor-muted)' }}>
              {pendingAction === 'fetch' && 
                'Enter the destination GitHub organization to fetch mannequins from. Mannequins will be matched to destination org members using login, email, and name.'}
              {pendingAction === 'invite' && 
                `Send an attribution invitation for ${pendingSourceLogin} to reclaim their mannequin.`}
              {pendingAction === 'bulk_invite' && 
                `Send attribution invitations to all ${invitableCount} mapped users with mannequins.`}
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
            {pendingAction === 'fetch' && (
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
          </div>
          <div className="flex justify-end gap-2 p-4 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
            <Button onClick={cancelDestOrgDialog}>Cancel</Button>
            <Button 
              variant="primary" 
              onClick={handleConfirmDestOrg}
              disabled={!destinationOrg}
            >
              {pendingAction === 'fetch' && 'Fetch'}
              {pendingAction === 'invite' && 'Send Invitation'}
              {pendingAction === 'bulk_invite' && 'Send All'}
            </Button>
          </div>
        </Dialog>
      )}

      {/* Delete Mapping Dialog */}
      {showDeleteDialog && (
        <Dialog
          title="Delete Mapping"
          onClose={() => setShowDeleteDialog(null)}
        >
          <div className="p-4">
            <p className="mb-3" style={{ color: 'var(--fgColor-default)' }}>
              Are you sure you want to delete the mapping for <strong>{showDeleteDialog}</strong>?
            </p>
            <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              This will remove the destination mapping. The source user data will be preserved.
            </p>
          </div>
          <div className="flex justify-end gap-2 p-4 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
            <Button onClick={() => setShowDeleteDialog(null)}>Cancel</Button>
            <Button
              variant="danger"
              onClick={handleConfirmDelete}
              disabled={deleteMapping.isPending}
            >
              {deleteMapping.isPending ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </Dialog>
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
