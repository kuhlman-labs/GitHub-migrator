import { useState, useCallback } from 'react';
import {
  Button,
  TextInput,
  Flash,
  ActionMenu,
  ActionList,
  Label,
  Avatar,
  Spinner,
} from '@primer/react';
import {
  PersonIcon,
  CheckIcon,
  XIcon,
  SearchIcon,
  DownloadIcon,
  UploadIcon,
  LinkIcon,
} from '@primer/octicons-react';
import { useUserMappings, useUserMappingStats } from '../../hooks/useQueries';
import {
  useUpdateUserMapping,
  useDeleteUserMapping,
} from '../../hooks/useMutations';
import { UserMapping, UserMappingStatus } from '../../types';
import { api } from '../../services/api';
import { Pagination } from '../common/Pagination';

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

export function UserMappingTable() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [page, setPage] = useState(1);
  const [editingMapping, setEditingMapping] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<{ created: number; updated: number; errors: number } | null>(null);

  const { data, isLoading, error, refetch } = useUserMappings({
    search: search || undefined,
    status: statusFilter || undefined,
    limit: ITEMS_PER_PAGE,
    offset: (page - 1) * ITEMS_PER_PAGE,
  });

  const { data: stats } = useUserMappingStats();
  const updateMapping = useUpdateUserMapping();
  const deleteMapping = useDeleteUserMapping();

  const mappings = data?.mappings || [];
  const total = data?.total || 0;

  const handleSearch = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
    setPage(1);
  }, []);

  const handleStatusFilter = useCallback((status: string) => {
    setStatusFilter(status);
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
        updates: { destination_login: editValue || undefined },
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

  const handleDelete = useCallback(async (sourceLogin: string) => {
    if (window.confirm(`Delete mapping for ${sourceLogin}?`)) {
      try {
        await deleteMapping.mutateAsync(sourceLogin);
      } catch (err) {
        console.error('Failed to delete mapping:', err);
      }
    }
  }, [deleteMapping]);

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
    const fileToImport = file || importFile;
    if (!fileToImport) return;
    
    try {
      const result = await api.importUserMappings(fileToImport);
      setImportResult(result);
      setImportFile(null);
      refetch();
      // Clear the result after 5 seconds
      setTimeout(() => setImportResult(null), 5000);
    } catch (err) {
      console.error('Failed to import mappings:', err);
    }
  }, [importFile, refetch]);

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
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-4">
          <h2 className="text-xl font-semibold flex items-center gap-2">
            <PersonIcon size={24} />
            User Identity Mapping
          </h2>
          {stats && (
            <div className="flex gap-2">
              <Label variant="accent">{stats.total} Total</Label>
              <Label variant="success">{stats.mapped} Mapped</Label>
              <Label variant="attention">{stats.unmapped} Unmapped</Label>
              {stats.reclaimed > 0 && <Label variant="done">{stats.reclaimed} Reclaimed</Label>}
            </div>
          )}
        </div>
        <div className="flex gap-2">
          <input
            type="file"
            id="import-csv-input"
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
            onClick={() => document.getElementById('import-csv-input')?.click()}
            leadingVisual={UploadIcon}
          >
            Import
          </Button>
          <ActionMenu>
            <ActionMenu.Button leadingVisual={DownloadIcon}>
              Export
            </ActionMenu.Button>
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
            placeholder="Search by login, email, or name..."
            value={search}
            onChange={handleSearch}
            className="w-full"
          />
        </div>
        <ActionMenu>
          <ActionMenu.Button>
            Status: {statusFilter ? statusLabels[statusFilter as UserMappingStatus] : 'All'}
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
        <table className="w-full" style={{ borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
              <th className="text-left p-3 font-medium">Source User</th>
              <th className="text-left p-3 font-medium">Destination User</th>
              <th className="text-left p-3 font-medium">Status</th>
              <th className="text-left p-3 font-medium">Mannequin</th>
              <th className="text-right p-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {mappings.map((mapping) => {
              const login = getLogin(mapping);
              const email = getEmail(mapping);
              return (
              <tr
                key={login}
                style={{ borderBottom: '1px solid var(--borderColor-muted)' }}
                className="hover:opacity-80"
              >
                <td className="p-3">
                  <div className="flex items-center gap-2">
                    <Avatar src={mapping.avatar_url || `https://github.com/${login}.png`} size={24} />
                    <div>
                      <div className="font-medium">{login}</div>
                      {email && (
                        <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                          {email}
                        </span>
                      )}
                    </div>
                  </div>
                </td>
                <td className="p-3">
                  {editingMapping === login ? (
                    <div className="flex items-center gap-2">
                      <TextInput
                        value={editValue}
                        onChange={(e) => setEditValue(e.target.value)}
                        placeholder="destination-username"
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
                    <div className="flex items-center gap-2">
                      {mapping.destination_login ? (
                        <>
                          <Avatar src={`https://github.com/${mapping.destination_login}.png`} size={24} />
                          <span>{mapping.destination_login}</span>
                          <span style={{ color: 'var(--fgColor-muted)' }}><LinkIcon size={16} /></span>
                        </>
                      ) : (
                        <span style={{ color: 'var(--fgColor-muted)' }}>Not mapped</span>
                      )}
                    </div>
                  )}
                </td>
                <td className="p-3">
                  <Label variant={statusColors[mapping.mapping_status as UserMappingStatus]}>
                    {statusLabels[mapping.mapping_status as UserMappingStatus]}
                  </Label>
                </td>
                <td className="p-3">
                  {mapping.mannequin_login ? (
                    <div>
                      <span className="text-sm">{mapping.mannequin_login}</span>
                      {mapping.reclaim_status && (
                        <Label variant="secondary" className="ml-2">
                          {mapping.reclaim_status}
                        </Label>
                      )}
                    </div>
                  ) : (
                    <span style={{ color: 'var(--fgColor-muted)' }}>—</span>
                  )}
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
                          <ActionList.Item onSelect={() => handleSkip(login)}>
                            Skip user
                          </ActionList.Item>
                        )}
                        <ActionList.Divider />
                        <ActionList.Item onSelect={() => handleDelete(login)}>
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
                    No users found. Run discovery to discover organization members.
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
