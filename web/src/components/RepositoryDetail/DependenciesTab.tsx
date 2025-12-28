import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { UnderlineNav, SegmentedControl, ActionMenu, ActionList } from '@primer/react';
import { DownloadIcon } from '@primer/octicons-react';
import { Button } from '../common/buttons';
import { api } from '../../services/api';
import type { DependenciesResponse, RepositoryDependency, DependentsResponse, DependentRepository } from '../../types';
import { Badge } from '../common/Badge';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { Pagination } from '../common/Pagination';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

interface DependenciesTabProps {
  fullName: string;
}

interface DependencyMetadata {
  type: string;
  path?: string;
  branch?: string;
  workflow_file?: string;
  ref?: string;
  manifest?: string;
  package_manager?: string;
  [key: string]: string | undefined;
}

interface MergedDependency extends RepositoryDependency {
  detection_methods: string[];
  all_metadata: DependencyMetadata[];
}

type ScopeFilter = 'all' | 'local' | 'external';
type ViewTab = 'depends_on' | 'depended_by';

export function DependenciesTab({ fullName }: DependenciesTabProps) {
  const { showError } = useToast();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<DependenciesResponse | null>(null);
  const [dependentsData, setDependentsData] = useState<DependentsResponse | null>(null);
  const [deduplicatedDeps, setDeduplicatedDeps] = useState<MergedDependency[]>([]);
  
  // View tab state
  const [viewTab, setViewTab] = useState<ViewTab>('depends_on');
  
  // Filter and pagination state
  const [scopeFilter, setScopeFilter] = useState<ScopeFilter>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;
  
  // Export state
  const [exporting, setExporting] = useState(false);

  // Deduplicate dependencies by dependency_full_name
  const deduplicateDependencies = (dependencies: RepositoryDependency[]): MergedDependency[] => {
    const depMap = new Map<string, MergedDependency>();

    dependencies.forEach(dep => {
      const key = dep.dependency_full_name;
      
      if (depMap.has(key)) {
        const existing = depMap.get(key)!;
        // Add this detection method
        if (!existing.detection_methods.includes(dep.dependency_type)) {
          existing.detection_methods.push(dep.dependency_type);
        }
        // Collect all metadata
        if (dep.metadata) {
          try {
            const parsed = JSON.parse(dep.metadata);
            existing.all_metadata.push({ type: dep.dependency_type, ...parsed });
          } catch {
            // Skip invalid metadata
          }
        }
        // If any instance is local, mark as local
        if (dep.is_local) {
          existing.is_local = true;
        }
      } else {
        // First occurrence of this dependency
        const metadata: DependencyMetadata[] = [];
        if (dep.metadata) {
          try {
            const parsed = JSON.parse(dep.metadata);
            metadata.push({ type: dep.dependency_type, ...parsed });
          } catch {
            // Skip invalid metadata
          }
        }
        
        depMap.set(key, {
          ...dep,
          detection_methods: [dep.dependency_type],
          all_metadata: metadata
        });
      }
    });

    return Array.from(depMap.values());
  };

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        
        // Fetch both dependencies and dependents in parallel
        const [depsResponse, dependentsResponse] = await Promise.all([
          api.getRepositoryDependencies(fullName),
          api.getRepositoryDependents(fullName)
        ]);
        
        // Deduplicate dependencies
        const deduplicated = deduplicateDependencies(depsResponse.dependencies);
        setDeduplicatedDeps(deduplicated);
        setData(depsResponse);
        setDependentsData(dependentsResponse);
      } catch (err: unknown) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to load dependencies';
        setError(errorMessage);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [fullName]);

  // Reset page when filter or view changes
  useEffect(() => {
    setCurrentPage(1);
  }, [scopeFilter, viewTab]);

  const handleExport = async (format: 'csv' | 'json') => {
    try {
      setExporting(true);
      const blob = await api.exportRepositoryDependencies(fullName, format);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${fullName.replace('/', '-')}-dependencies.${format}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      handleApiError(error, showError, 'Failed to export dependencies');
    } finally {
      setExporting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <LoadingSpinner />
      </div>
    );
  }

  if (error) {
    return (
      <div 
        className="rounded-lg p-4"
        style={{
          backgroundColor: 'var(--danger-subtle)',
          border: '1px solid var(--borderColor-danger)'
        }}
      >
        <h4 className="font-medium mb-2" style={{ color: 'var(--fgColor-danger)' }}>Error loading dependencies</h4>
        <p className="text-sm" style={{ color: 'var(--fgColor-danger)' }}>{error}</p>
      </div>
    );
  }

  // Calculate deduplicated summary
  const deduplicatedSummary = {
    total: deduplicatedDeps.length,
    local: deduplicatedDeps.filter(d => d.is_local).length,
    external: deduplicatedDeps.filter(d => !d.is_local).length,
    by_type: deduplicatedDeps.reduce((acc, dep) => {
      dep.detection_methods.forEach(method => {
        acc[method] = (acc[method] || 0) + 1;
      });
      return acc;
    }, {} as Record<string, number>)
  };

  const dependentsCount = dependentsData?.total || 0;
  const hasAnyRelationships = deduplicatedSummary.total > 0 || dependentsCount > 0;

  if (!hasAnyRelationships) {
    return (
      <div 
        className="rounded-lg p-4"
        style={{
          backgroundColor: 'var(--accent-subtle)',
          border: '1px solid var(--borderColor-accent-muted)'
        }}
      >
        <h4 className="font-medium mb-2" style={{ color: 'var(--fgColor-accent)' }}>No dependencies found</h4>
        <p className="text-sm" style={{ color: 'var(--fgColor-accent)' }}>
          This repository has no detected dependencies or dependents (submodules, workflow references, or dependency graph relationships).
        </p>
      </div>
    );
  }

  // Filter dependencies by scope
  const filteredDeps = deduplicatedDeps.filter(dep => {
    if (scopeFilter === 'local') return dep.is_local;
    if (scopeFilter === 'external') return !dep.is_local;
    return true; // 'all'
  });

  // Paginate based on current view
  const getCurrentItems = () => {
    if (viewTab === 'depends_on') {
      return filteredDeps;
    } else {
      return dependentsData?.dependents || [];
    }
  };

  const currentItems = getCurrentItems();
  const totalItems = currentItems.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedItems = currentItems.slice(startIndex, endIndex);

  return (
    <div className="space-y-6">
      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Depends On</div>
          <div className="text-2xl font-bold mt-1" style={{ color: 'var(--fgColor-default)' }}>{deduplicatedSummary.total}</div>
          {data && data.summary.total > deduplicatedSummary.total && (
            <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
              ({data.summary.total} raw detections)
            </div>
          )}
        </div>
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Depended By</div>
          <div className="text-2xl font-bold mt-1" style={{ color: 'var(--fgColor-default)' }}>
            {dependentsCount}
          </div>
          <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Repos that use this
          </div>
        </div>
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Local Dependencies</div>
          <div className="text-2xl font-bold mt-1" style={{ color: 'var(--fgColor-success)' }}>
            {deduplicatedSummary.local}
          </div>
          <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Within enterprise
          </div>
        </div>
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>External Dependencies</div>
          <div className="text-2xl font-bold mt-1" style={{ color: 'var(--fgColor-attention)' }}>
            {deduplicatedSummary.external}
          </div>
          <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Outside enterprise
          </div>
        </div>
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Detection Methods</div>
          <div className="mt-2 space-y-1">
            {Object.entries(deduplicatedSummary.by_type).map(([type, count]) => (
              <div key={type} className="text-sm flex justify-between">
                <span className="capitalize" style={{ color: 'var(--fgColor-default)' }}>{type.replace('_', ' ')}</span>
                <span className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>{count}</span>
              </div>
            ))}
            {Object.keys(deduplicatedSummary.by_type).length === 0 && (
              <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>None</div>
            )}
          </div>
        </div>
      </div>

      {/* Warning for local dependencies */}
      {(deduplicatedSummary.local > 0 || dependentsCount > 0) && (
        <div 
          className="rounded-lg p-4"
          style={{
            backgroundColor: 'var(--attention-subtle)',
            border: '1px solid var(--borderColor-attention)'
          }}
        >
          <h4 className="font-medium mb-2" style={{ color: 'var(--fgColor-attention)' }}>Local Dependencies Detected</h4>
          <p className="text-sm" style={{ color: 'var(--fgColor-attention)' }}>
            {deduplicatedSummary.local > 0 && (
              <>This repository depends on {deduplicatedSummary.local} other repository/repositories in your enterprise. </>
            )}
            {dependentsCount > 0 && (
              <>{dependentsCount} other repository/repositories depend on this one. </>
            )}
            Consider migrating related repositories in the same batch to maintain functionality.
          </p>
        </div>
      )}

      {/* Export Button with Dropdown */}
      <div className="flex justify-end">
        <ActionMenu>
          <ActionMenu.Anchor>
            <Button
              disabled={exporting}
              leadingVisual={DownloadIcon}
              variant="primary"
            >
              {exporting ? 'Exporting...' : 'Export'}
            </Button>
          </ActionMenu.Anchor>
          <ActionMenu.Overlay>
            <ActionList>
              <ActionList.Item onSelect={() => handleExport('csv')}>
                Export as CSV
              </ActionList.Item>
              <ActionList.Item onSelect={() => handleExport('json')}>
                Export as JSON
              </ActionList.Item>
            </ActionList>
          </ActionMenu.Overlay>
        </ActionMenu>
      </div>

      {/* View Tabs */}
      <div 
        className="rounded-lg shadow-sm"
        style={{ backgroundColor: 'var(--bgColor-default)' }}
      >
        <UnderlineNav aria-label="Dependency view">
          <UnderlineNav.Item
            aria-current={viewTab === 'depends_on' ? 'page' : undefined}
            onSelect={() => setViewTab('depends_on')}
            counter={deduplicatedSummary.total}
          >
            Depends On
          </UnderlineNav.Item>
          <UnderlineNav.Item
            aria-current={viewTab === 'depended_by' ? 'page' : undefined}
            onSelect={() => setViewTab('depended_by')}
            counter={dependentsCount}
          >
            Depended By
          </UnderlineNav.Item>
        </UnderlineNav>

        <div className="p-6">
          {viewTab === 'depends_on' ? (
            <>
              {/* Scope Filter for Depends On view */}
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  Dependencies ({deduplicatedSummary.total})
                </h3>
                
                <SegmentedControl
                  aria-label="Dependency scope filter"
                  onChange={(index) => {
                    const scopes: ScopeFilter[] = ['all', 'local', 'external'];
                    setScopeFilter(scopes[index]);
                  }}
                >
                  <SegmentedControl.Button selected={scopeFilter === 'all'}>
                    {`All (${deduplicatedSummary.total})`}
                  </SegmentedControl.Button>
                  <SegmentedControl.Button selected={scopeFilter === 'local'}>
                    {`Local (${deduplicatedSummary.local})`}
                  </SegmentedControl.Button>
                  <SegmentedControl.Button selected={scopeFilter === 'external'}>
                    {`External (${deduplicatedSummary.external})`}
                  </SegmentedControl.Button>
                </SegmentedControl>
              </div>
              
              {filteredDeps.length === 0 ? (
                <div className="text-center py-8" style={{ color: 'var(--fgColor-muted)' }}>
                  No {scopeFilter === 'all' ? '' : scopeFilter} dependencies found
                </div>
              ) : (
                <>
                  <div className="overflow-x-auto">
                    <table 
                      className="min-w-full divide-y"
                      style={{ borderColor: 'var(--borderColor-muted)' }}
                    >
                      <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                        <tr>
                          <th 
                            className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                            style={{ color: 'var(--fgColor-muted)' }}
                          >
                            Repository
                          </th>
                          <th 
                            className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                            style={{ color: 'var(--fgColor-muted)' }}
                          >
                            Detection Methods
                          </th>
                          <th 
                            className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                            style={{ color: 'var(--fgColor-muted)' }}
                          >
                            Scope
                          </th>
                          <th 
                            className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                            style={{ color: 'var(--fgColor-muted)' }}
                          >
                            Details
                          </th>
                        </tr>
                      </thead>
                      <tbody 
                        className="divide-y"
                        style={{ borderColor: 'var(--borderColor-muted)' }}
                      >
                        {(paginatedItems as MergedDependency[]).map((dep) => (
                          <MergedDependencyRow key={dep.id} dependency={dep} />
                        ))}
                      </tbody>
                    </table>
                  </div>
                  
                  {totalItems > pageSize && (
                    <div className="mt-4">
                      <Pagination
                        currentPage={currentPage}
                        totalItems={totalItems}
                        pageSize={pageSize}
                        onPageChange={setCurrentPage}
                      />
                    </div>
                  )}
                </>
              )}
            </>
          ) : (
            <>
              {/* Depended By view */}
              <div className="mb-4">
                <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  Repositories that depend on this one ({dependentsCount})
                </h3>
                <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                  These repositories have local dependencies on {fullName}
                </p>
              </div>
              
              {dependentsCount === 0 ? (
                <div className="text-center py-8" style={{ color: 'var(--fgColor-muted)' }}>
                  No repositories depend on this one
                </div>
              ) : (
                <>
                  <div className="overflow-x-auto">
                    <table 
                      className="min-w-full divide-y"
                      style={{ borderColor: 'var(--borderColor-muted)' }}
                    >
                      <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                        <tr>
                          <th 
                            className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                            style={{ color: 'var(--fgColor-muted)' }}
                          >
                            Repository
                          </th>
                          <th 
                            className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                            style={{ color: 'var(--fgColor-muted)' }}
                          >
                            Status
                          </th>
                          <th 
                            className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                            style={{ color: 'var(--fgColor-muted)' }}
                          >
                            Dependency Types
                          </th>
                        </tr>
                      </thead>
                      <tbody 
                        className="divide-y"
                        style={{ borderColor: 'var(--borderColor-muted)' }}
                      >
                        {(paginatedItems as DependentRepository[]).map((dep) => (
                          <DependentRow key={dep.id} dependent={dep} />
                        ))}
                      </tbody>
                    </table>
                  </div>
                  
                  {totalItems > pageSize && (
                    <div className="mt-4">
                      <Pagination
                        currentPage={currentPage}
                        totalItems={totalItems}
                        pageSize={pageSize}
                        onPageChange={setCurrentPage}
                      />
                    </div>
                  )}
                </>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

interface MergedDependencyRowProps {
  dependency: MergedDependency;
}

function MergedDependencyRow({ dependency }: MergedDependencyRowProps) {
  const getTypeColor = (type: string) => {
    switch (type) {
      case 'submodule':
        return 'blue';
      case 'workflow':
        return 'purple';
      case 'dependency_graph':
        return 'green';
      case 'package':
        return 'gray';
      default:
        return 'gray';
    }
  };

  const getTypeLabel = (type: string) => {
    return type.replace('_', ' ').replace(/\b\w/g, (l) => l.toUpperCase());
  };

  return (
    <tr className="hover:opacity-80 transition-opacity">
      <td className="px-4 py-4 whitespace-nowrap">
        <div className="text-sm font-medium">
          {dependency.is_local ? (
            <Link
              to={`/repository/${encodeURIComponent(dependency.dependency_full_name)}`}
              className="hover:underline"
              style={{ color: 'var(--fgColor-accent)' }}
            >
              {dependency.dependency_full_name}
            </Link>
          ) : (
            <a
              href={dependency.dependency_url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:underline"
              style={{ color: 'var(--fgColor-accent)' }}
            >
              {dependency.dependency_full_name}
            </a>
          )}
        </div>
        {dependency.dependency_url && (
          <div className="text-xs truncate max-w-md" style={{ color: 'var(--fgColor-muted)' }}>
            {dependency.dependency_url}
          </div>
        )}
      </td>
      <td className="px-4 py-4">
        <div className="flex flex-wrap gap-1">
          {dependency.detection_methods.map((method) => (
            <Badge key={method} color={getTypeColor(method)}>
              {getTypeLabel(method)}
            </Badge>
          ))}
        </div>
      </td>
      <td className="px-4 py-4 whitespace-nowrap">
        {dependency.is_local ? (
          <Badge color="green">Local</Badge>
        ) : (
          <Badge color="yellow">External</Badge>
        )}
      </td>
      <td className="px-4 py-4">
        <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
          {dependency.all_metadata.length > 0 ? (
            <div className="space-y-2">
              {dependency.all_metadata.map((meta, idx) => (
                <div key={idx} className="space-y-1">
                  {meta.type && dependency.all_metadata.length > 1 && (
                    <div className="text-xs font-semibold capitalize" style={{ color: 'var(--fgColor-default)' }}>
                      {meta.type.replace('_', ' ')}:
                    </div>
                  )}
                  {meta.path && (
                    <div>
                      <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>Path:</span> {meta.path}
                    </div>
                  )}
                  {meta.branch && (
                    <div>
                      <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>Branch:</span> {meta.branch}
                    </div>
                  )}
                  {meta.workflow_file && (
                    <div>
                      <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>Workflow:</span> {meta.workflow_file}
                    </div>
                  )}
                  {meta.ref && (
                    <div>
                      <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>Ref:</span> {meta.ref}
                    </div>
                  )}
                  {meta.manifest && (
                    <div>
                      <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>Manifest:</span> {meta.manifest}
                    </div>
                  )}
                  {meta.package_manager && (
                    <div>
                      <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>Manager:</span> {meta.package_manager}
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <span style={{ color: 'var(--fgColor-muted)' }}>—</span>
          )}
        </div>
      </td>
    </tr>
  );
}

interface DependentRowProps {
  dependent: DependentRepository;
}

function DependentRow({ dependent }: DependentRowProps) {
  const getTypeColor = (type: string) => {
    switch (type) {
      case 'submodule':
        return 'blue';
      case 'workflow':
        return 'purple';
      case 'dependency_graph':
        return 'green';
      case 'package':
        return 'gray';
      default:
        return 'gray';
    }
  };

  const getTypeLabel = (type: string) => {
    return type.replace('_', ' ').replace(/\b\w/g, (l) => l.toUpperCase());
  };

  const getStatusColor = (status: string) => {
    if (status === 'complete' || status === 'migration_complete') return 'green';
    if (status === 'pending') return 'gray';
    if (status.includes('failed')) return 'red';
    if (status.includes('progress') || status.includes('queued')) return 'blue';
    return 'gray';
  };

  return (
    <tr className="hover:opacity-80 transition-opacity">
      <td className="px-4 py-4 whitespace-nowrap">
        <div className="text-sm font-medium">
          <Link
            to={`/repository/${encodeURIComponent(dependent.full_name)}`}
            className="hover:underline"
            style={{ color: 'var(--fgColor-accent)' }}
          >
            {dependent.full_name}
          </Link>
        </div>
        {dependent.source_url && (
          <div className="text-xs truncate max-w-md" style={{ color: 'var(--fgColor-muted)' }}>
            {dependent.source_url}
          </div>
        )}
      </td>
      <td className="px-4 py-4 whitespace-nowrap">
        <Badge color={getStatusColor(dependent.status)}>
          {dependent.status.replace(/_/g, ' ')}
        </Badge>
      </td>
      <td className="px-4 py-4">
        <div className="flex flex-wrap gap-1">
          {dependent.dependency_types.map((type) => (
            <Badge key={type} color={getTypeColor(type)}>
              {getTypeLabel(type)}
            </Badge>
          ))}
        </div>
      </td>
    </tr>
  );
}
