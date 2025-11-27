import { useEffect, useState } from 'react';
import { api } from '../../services/api';
import type { DependenciesResponse, RepositoryDependency } from '../../types';
import { Badge } from '../common/Badge';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { Pagination } from '../common/Pagination';

interface DependenciesTabProps {
  fullName: string;
}

interface MergedDependency extends RepositoryDependency {
  detection_methods: string[];
  all_metadata: any[];
}

type ScopeFilter = 'all' | 'local' | 'external';

export function DependenciesTab({ fullName }: DependenciesTabProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<DependenciesResponse | null>(null);
  const [deduplicatedDeps, setDeduplicatedDeps] = useState<MergedDependency[]>([]);
  
  // Filter and pagination state
  const [scopeFilter, setScopeFilter] = useState<ScopeFilter>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;

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
          } catch (e) {
            // Skip invalid metadata
          }
        }
        // If any instance is local, mark as local
        if (dep.is_local) {
          existing.is_local = true;
        }
      } else {
        // First occurrence of this dependency
        const metadata = [];
        if (dep.metadata) {
          try {
            const parsed = JSON.parse(dep.metadata);
            metadata.push({ type: dep.dependency_type, ...parsed });
          } catch (e) {
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
    const fetchDependencies = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await api.getRepositoryDependencies(fullName);
        
        // Deduplicate dependencies
        const deduplicated = deduplicateDependencies(response.dependencies);
        setDeduplicatedDeps(deduplicated);
        setData(response);
      } catch (err: any) {
        console.error('Failed to fetch dependencies:', err);
        setError(err?.response?.data?.message || 'Failed to load dependencies');
      } finally {
        setLoading(false);
      }
    };

    fetchDependencies();
  }, [fullName]);

  // Reset page when filter changes
  useEffect(() => {
    setCurrentPage(1);
  }, [scopeFilter]);

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

  if (!data || !data.dependencies || data.dependencies.length === 0) {
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
          This repository has no detected dependencies (submodules, workflow references, or dependency graph relationships).
        </p>
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

  // Filter dependencies by scope
  const filteredDeps = deduplicatedDeps.filter(dep => {
    if (scopeFilter === 'local') return dep.is_local;
    if (scopeFilter === 'external') return !dep.is_local;
    return true; // 'all'
  });

  // Paginate filtered dependencies
  const totalItems = filteredDeps.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedDeps = filteredDeps.slice(startIndex, endIndex);

  return (
    <div className="space-y-6">
      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Total Dependencies</div>
          <div className="text-2xl font-bold mt-1" style={{ color: 'var(--fgColor-default)' }}>{deduplicatedSummary.total}</div>
          {data.summary.total > deduplicatedSummary.total && (
            <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
              ({data.summary.total} raw detections)
            </div>
          )}
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
          </div>
        </div>
      </div>

      {/* Warning for local dependencies */}
      {deduplicatedSummary.local > 0 && (
        <div 
          className="rounded-lg p-4"
          style={{
            backgroundColor: 'var(--attention-subtle)',
            border: '1px solid var(--borderColor-attention)'
          }}
        >
          <h4 className="font-medium mb-2" style={{ color: 'var(--fgColor-attention)' }}>Local Dependencies Detected</h4>
          <p className="text-sm" style={{ color: 'var(--fgColor-attention)' }}>
            This repository depends on {deduplicatedSummary.local} other repository/repositories in your enterprise. 
            Consider migrating these dependencies in the same batch to maintain functionality.
          </p>
        </div>
      )}

      {/* Dependencies List */}
      <div 
        className="rounded-lg shadow-sm"
        style={{ backgroundColor: 'var(--bgColor-default)' }}
      >
        <div className="p-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
              All Dependencies ({deduplicatedSummary.total})
            </h3>
            
            {/* Scope Filter */}
            <div className="flex gap-2">
              <button
                onClick={() => setScopeFilter('all')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: scopeFilter === 'all' ? '#2da44e' : 'var(--control-bgColor-rest)',
                  color: scopeFilter === 'all' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                All ({deduplicatedSummary.total})
              </button>
              <button
                onClick={() => setScopeFilter('local')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: scopeFilter === 'local' ? '#2da44e' : 'var(--control-bgColor-rest)',
                  color: scopeFilter === 'local' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                Local ({deduplicatedSummary.local})
              </button>
              <button
                onClick={() => setScopeFilter('external')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: scopeFilter === 'external' ? '#fb8500' : 'var(--control-bgColor-rest)',
                  color: scopeFilter === 'external' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                External ({deduplicatedSummary.external})
              </button>
            </div>
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
                    {paginatedDeps.map((dep) => (
                      <MergedDependencyRow key={dep.id} dependency={dep} />
                    ))}
                  </tbody>
                </table>
              </div>
              
              {/* Pagination */}
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
          <a
            href={dependency.dependency_url}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:underline"
            style={{ color: 'var(--fgColor-accent)' }}
          >
            {dependency.dependency_full_name}
          </a>
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

