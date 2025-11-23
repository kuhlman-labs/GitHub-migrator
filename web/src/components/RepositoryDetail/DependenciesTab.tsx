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
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <h4 className="font-medium text-red-800 mb-2">Error loading dependencies</h4>
        <p className="text-sm text-red-700">{error}</p>
      </div>
    );
  }

  if (!data || !data.dependencies || data.dependencies.length === 0) {
    return (
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h4 className="font-medium text-blue-800 mb-2">No dependencies found</h4>
        <p className="text-sm text-blue-700">
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
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">Total Dependencies</div>
          <div className="text-2xl font-bold mt-1">{deduplicatedSummary.total}</div>
          {data.summary.total > deduplicatedSummary.total && (
            <div className="text-xs text-gray-500 mt-1">
              ({data.summary.total} raw detections)
            </div>
          )}
        </div>
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">Local Dependencies</div>
          <div className="text-2xl font-bold mt-1 text-green-600">
            {deduplicatedSummary.local}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            Within enterprise
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">External Dependencies</div>
          <div className="text-2xl font-bold mt-1 text-amber-600">
            {deduplicatedSummary.external}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            Outside enterprise
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">Detection Methods</div>
          <div className="mt-2 space-y-1">
            {Object.entries(deduplicatedSummary.by_type).map(([type, count]) => (
              <div key={type} className="text-sm flex justify-between">
                <span className="capitalize">{type.replace('_', ' ')}</span>
                <span className="font-semibold">{count}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Warning for local dependencies */}
      {deduplicatedSummary.local > 0 && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <h4 className="font-medium text-yellow-800 mb-2">Local Dependencies Detected</h4>
          <p className="text-sm text-yellow-700">
            This repository depends on {deduplicatedSummary.local} other repository/repositories in your enterprise. 
            Consider migrating these dependencies in the same batch to maintain functionality.
          </p>
        </div>
      )}

      {/* Dependencies List */}
      <div className="bg-white rounded-lg shadow-sm">
        <div className="p-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg font-semibold">
              All Dependencies ({deduplicatedSummary.total})
            </h3>
            
            {/* Scope Filter */}
            <div className="flex gap-2">
              <button
                onClick={() => setScopeFilter('all')}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  scopeFilter === 'all'
                    ? 'bg-gh-blue-2 text-white'
                    : 'bg-gh-canvas-inset text-gh-text-primary hover:bg-gh-border-muted'
                }`}
              >
                All ({deduplicatedSummary.total})
              </button>
              <button
                onClick={() => setScopeFilter('local')}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  scopeFilter === 'local'
                    ? 'bg-gh-green-4 text-white'
                    : 'bg-gh-canvas-inset text-gh-text-primary hover:bg-gh-border-muted'
                }`}
              >
                Local ({deduplicatedSummary.local})
              </button>
              <button
                onClick={() => setScopeFilter('external')}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  scopeFilter === 'external'
                    ? 'bg-orange-500 text-white'
                    : 'bg-gh-canvas-inset text-gh-text-primary hover:bg-gh-border-muted'
                }`}
              >
                External ({deduplicatedSummary.external})
              </button>
            </div>
          </div>
          
          {filteredDeps.length === 0 ? (
            <div className="text-center py-8 text-gh-text-muted">
              No {scopeFilter === 'all' ? '' : scopeFilter} dependencies found
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead>
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Repository
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Detection Methods
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Scope
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Details
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
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
    <tr className="hover:bg-gray-50">
      <td className="px-4 py-4 whitespace-nowrap">
        <div className="text-sm font-medium text-gray-900">
          <a
            href={dependency.dependency_url}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-blue-600"
          >
            {dependency.dependency_full_name}
          </a>
        </div>
        {dependency.dependency_url && (
          <div className="text-xs text-gray-500 truncate max-w-md">
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
        <div className="text-sm text-gray-500">
          {dependency.all_metadata.length > 0 ? (
            <div className="space-y-2">
              {dependency.all_metadata.map((meta, idx) => (
                <div key={idx} className="space-y-1">
                  {meta.type && dependency.all_metadata.length > 1 && (
                    <div className="text-xs font-semibold text-gray-600 capitalize">
                      {meta.type.replace('_', ' ')}:
                    </div>
                  )}
                  {meta.path && (
                    <div>
                      <span className="font-medium">Path:</span> {meta.path}
                    </div>
                  )}
                  {meta.branch && (
                    <div>
                      <span className="font-medium">Branch:</span> {meta.branch}
                    </div>
                  )}
                  {meta.workflow_file && (
                    <div>
                      <span className="font-medium">Workflow:</span> {meta.workflow_file}
                    </div>
                  )}
                  {meta.ref && (
                    <div>
                      <span className="font-medium">Ref:</span> {meta.ref}
                    </div>
                  )}
                  {meta.manifest && (
                    <div>
                      <span className="font-medium">Manifest:</span> {meta.manifest}
                    </div>
                  )}
                  {meta.package_manager && (
                    <div>
                      <span className="font-medium">Manager:</span> {meta.package_manager}
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <span className="text-gray-400">—</span>
          )}
        </div>
      </td>
    </tr>
  );
}

