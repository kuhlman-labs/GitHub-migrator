import { useEffect, useState } from 'react';
import { api } from '../../services/api';
import type { DependenciesResponse, RepositoryDependency } from '../../types';
import { Badge } from '../common/Badge';
import { LoadingSpinner } from '../common/LoadingSpinner';

interface DependenciesTabProps {
  fullName: string;
}

export function DependenciesTab({ fullName }: DependenciesTabProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<DependenciesResponse | null>(null);

  useEffect(() => {
    const fetchDependencies = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await api.getRepositoryDependencies(fullName);
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

  return (
    <div className="space-y-6">
      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">Total Dependencies</div>
          <div className="text-2xl font-bold mt-1">{data.summary.total}</div>
        </div>
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">Local Dependencies</div>
          <div className="text-2xl font-bold mt-1 text-green-600">
            {data.summary.local}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            Within enterprise
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">External Dependencies</div>
          <div className="text-2xl font-bold mt-1 text-amber-600">
            {data.summary.external}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            Outside enterprise
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm p-4">
          <div className="text-sm text-gray-500">By Type</div>
          <div className="mt-2 space-y-1">
            {Object.entries(data.summary.by_type).map(([type, count]) => (
              <div key={type} className="text-sm flex justify-between">
                <span className="capitalize">{type.replace('_', ' ')}</span>
                <span className="font-semibold">{count}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Warning for local dependencies */}
      {data.summary.local > 0 && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <h4 className="font-medium text-yellow-800 mb-2">Local Dependencies Detected</h4>
          <p className="text-sm text-yellow-700">
            This repository depends on {data.summary.local} other repository/repositories in your enterprise. 
            Consider migrating these dependencies in the same batch to maintain functionality.
          </p>
        </div>
      )}

      {/* Dependencies List */}
      <div className="bg-white rounded-lg shadow-sm">
        <div className="p-6">
          <h3 className="text-lg font-semibold mb-4">All Dependencies</h3>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead>
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Repository
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Type
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
                {data.dependencies.map((dep) => (
                  <DependencyRow key={dep.id} dependency={dep} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}

interface DependencyRowProps {
  dependency: RepositoryDependency;
}

function DependencyRow({ dependency }: DependencyRowProps) {
  const [metadata, setMetadata] = useState<any>(null);

  useEffect(() => {
    if (dependency.metadata) {
      try {
        setMetadata(JSON.parse(dependency.metadata));
      } catch (e) {
        console.error('Failed to parse metadata:', e);
      }
    }
  }, [dependency.metadata]);

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
      <td className="px-4 py-4 whitespace-nowrap">
        <Badge color={getTypeColor(dependency.dependency_type)}>
          {getTypeLabel(dependency.dependency_type)}
        </Badge>
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
          {metadata && (
            <div className="space-y-1">
              {metadata.path && (
                <div>
                  <span className="font-medium">Path:</span> {metadata.path}
                </div>
              )}
              {metadata.branch && (
                <div>
                  <span className="font-medium">Branch:</span> {metadata.branch}
                </div>
              )}
              {metadata.workflow_file && (
                <div>
                  <span className="font-medium">Workflow:</span> {metadata.workflow_file}
                </div>
              )}
              {metadata.ref && (
                <div>
                  <span className="font-medium">Ref:</span> {metadata.ref}
                </div>
              )}
              {metadata.manifest && (
                <div>
                  <span className="font-medium">Manifest:</span> {metadata.manifest}
                </div>
              )}
              {metadata.package_manager && (
                <div>
                  <span className="font-medium">Manager:</span> {metadata.package_manager}
                </div>
              )}
            </div>
          )}
          {!metadata && <span className="text-gray-400">—</span>}
        </div>
      </td>
    </tr>
  );
}

