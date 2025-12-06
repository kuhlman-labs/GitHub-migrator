import { useState, useEffect, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { TextInput, UnderlineNav, Button } from '@primer/react';
import { SearchIcon, DownloadIcon, AlertIcon, ChevronDownIcon } from '@primer/octicons-react';
import { Sankey, Tooltip, ResponsiveContainer } from 'recharts';
import { api } from '../../services/api';
import type { DependencyGraphResponse, DependencyGraphNode, DependencyGraphEdge } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Badge } from '../common/Badge';
import { Pagination } from '../common/Pagination';

type ViewMode = 'list' | 'graph';
type DependencyTypeFilter = 'all' | 'submodule' | 'workflow' | 'dependency_graph' | 'package';

export function Dependencies() {
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<DependencyGraphResponse | null>(null);
  
  // View and filter state
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [typeFilter, setTypeFilter] = useState<DependencyTypeFilter>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 25;
  
  // Export state
  const [exporting, setExporting] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);

  const fetchData = async (isRefresh = false) => {
    try {
      if (isRefresh) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError(null);
      
      const params = typeFilter !== 'all' ? { dependency_type: typeFilter } : undefined;
      const response = await api.getDependencyGraph(params);
      setData(response);
    } catch (err: unknown) {
      console.error('Failed to fetch dependency graph:', err);
      const errorMessage = err instanceof Error ? err.message : 'Failed to load dependency graph';
      setError(errorMessage);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [typeFilter]);

  // Reset page when search changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, typeFilter]);

  const handleExport = async (format: 'csv' | 'json') => {
    setShowExportMenu(false);
    try {
      setExporting(true);
      const params = typeFilter !== 'all' ? { dependency_type: typeFilter } : undefined;
      const blob = await api.exportDependencies(format, params);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `dependencies-export.${format}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Failed to export dependencies:', err);
    } finally {
      setExporting(false);
    }
  };

  // Filter and search nodes
  const filteredNodes = useMemo(() => {
    if (!data?.nodes) return [];
    
    let nodes = data.nodes;
    
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      nodes = nodes.filter(node => 
        node.full_name.toLowerCase().includes(query) ||
        node.organization.toLowerCase().includes(query)
      );
    }
    
    return nodes.sort((a, b) => {
      // Sort by total relationships (depended_by + depends_on) descending
      const aTotal = a.depended_by_count + a.depends_on_count;
      const bTotal = b.depended_by_count + b.depends_on_count;
      return bTotal - aTotal;
    });
  }, [data?.nodes, searchQuery]);

  // Filter edges based on nodes
  const filteredEdges = useMemo(() => {
    if (!data?.edges) return [];
    
    if (!searchQuery) return data.edges;
    
    const nodeIds = new Set(filteredNodes.map(n => n.id));
    return data.edges.filter(edge => 
      nodeIds.has(edge.source) || nodeIds.has(edge.target)
    );
  }, [data?.edges, filteredNodes, searchQuery]);

  // Paginate nodes for list view
  const paginatedNodes = useMemo(() => {
    const startIndex = (currentPage - 1) * pageSize;
    return filteredNodes.slice(startIndex, startIndex + pageSize);
  }, [filteredNodes, currentPage, pageSize]);

  // Calculate type distribution from edges
  const typeDistribution = useMemo(() => {
    if (!data?.edges) return {};
    return data.edges.reduce((acc, edge) => {
      acc[edge.dependency_type] = (acc[edge.dependency_type] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);
  }, [data?.edges]);

  // Find most depended repos
  const mostDependedRepos = useMemo(() => {
    if (!data?.nodes) return [];
    return [...data.nodes]
      .sort((a, b) => b.depended_by_count - a.depended_by_count)
      .slice(0, 5);
  }, [data?.nodes]);

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <LoadingSpinner />
      </div>
    );
  }

  if (error) {
    return (
      <div 
        className="rounded-lg p-6"
        style={{
          backgroundColor: 'var(--danger-subtle)',
          border: '1px solid var(--borderColor-danger)'
        }}
      >
        <h4 className="font-medium mb-2" style={{ color: 'var(--fgColor-danger)' }}>Error loading dependency graph</h4>
        <p className="text-sm" style={{ color: 'var(--fgColor-danger)' }}>{error}</p>
      </div>
    );
  }

  const stats = data?.stats || {
    total_repos_with_dependencies: 0,
    total_local_dependencies: 0,
    circular_dependency_count: 0
  };

  const hasData = stats.total_repos_with_dependencies > 0;

  return (
    <div className="relative space-y-6">
      <RefreshIndicator isRefreshing={refreshing} />
      
      {/* Header */}
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-light" style={{ color: 'var(--fgColor-default)' }}>
            Dependency Explorer
          </h1>
          <p className="text-sm mt-2" style={{ color: 'var(--fgColor-muted)' }}>
            Visualize and analyze local dependencies between repositories for migration batch planning
          </p>
        </div>
        
        {/* Export Button with Dropdown */}
        <div className="relative">
          <Button
            onClick={() => setShowExportMenu(!showExportMenu)}
            disabled={exporting || !hasData}
            leadingVisual={DownloadIcon}
            trailingVisual={ChevronDownIcon}
            variant="primary"
          >
            Export
          </Button>
          {showExportMenu && (
            <>
              {/* Backdrop to close menu when clicking outside */}
              <div 
                className="fixed inset-0 z-10" 
                onClick={() => setShowExportMenu(false)}
              />
              {/* Dropdown menu */}
              <div 
                className="absolute right-0 mt-2 w-48 rounded-lg shadow-lg z-20"
                style={{
                  backgroundColor: 'var(--bgColor-default)',
                  border: '1px solid var(--borderColor-default)',
                  boxShadow: 'var(--shadow-floating-large)'
                }}
              >
                <div className="py-1">
                  <button
                    onClick={() => handleExport('csv')}
                    className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                    style={{ color: 'var(--fgColor-default)' }}
                  >
                    Export as CSV
                  </button>
                  <button
                    onClick={() => handleExport('json')}
                    className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                    style={{ color: 'var(--fgColor-default)' }}
                  >
                    Export as JSON
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Repos with Dependencies</div>
          <div className="text-2xl font-bold mt-1" style={{ color: 'var(--fgColor-default)' }}>
            {stats.total_repos_with_dependencies}
          </div>
          <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Connected repositories
          </div>
        </div>
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Local Dependencies</div>
          <div className="text-2xl font-bold mt-1" style={{ color: 'var(--fgColor-success)' }}>
            {stats.total_local_dependencies}
          </div>
          <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Internal relationships
          </div>
        </div>
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Circular Dependencies</div>
          <div className="text-2xl font-bold mt-1" style={{ color: stats.circular_dependency_count > 0 ? 'var(--fgColor-attention)' : 'var(--fgColor-default)' }}>
            {stats.circular_dependency_count}
          </div>
          <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Bidirectional relationships
          </div>
        </div>
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>By Type</div>
          <div className="mt-2 space-y-1">
            {Object.entries(typeDistribution).map(([type, count]) => (
              <div key={type} className="text-sm flex justify-between">
                <span className="capitalize" style={{ color: 'var(--fgColor-default)' }}>{type.replace('_', ' ')}</span>
                <span className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>{count}</span>
              </div>
            ))}
            {Object.keys(typeDistribution).length === 0 && (
              <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>None</div>
            )}
          </div>
        </div>
      </div>

      {/* Circular Dependencies Warning */}
      {stats.circular_dependency_count > 0 && (
        <div 
          className="rounded-lg p-4 flex gap-3"
          style={{
            backgroundColor: 'var(--attention-subtle)',
            border: '1px solid var(--borderColor-attention)'
          }}
        >
          <div className="flex-shrink-0 mt-0.5" style={{ color: 'var(--fgColor-attention)' }}>
            <AlertIcon size={20} />
          </div>
          <div>
            <h4 className="font-medium mb-1" style={{ color: 'var(--fgColor-attention)' }}>
              Circular Dependencies Detected
            </h4>
            <p className="text-sm" style={{ color: 'var(--fgColor-attention)' }}>
              {stats.circular_dependency_count} pair(s) of repositories have bidirectional dependencies. 
              These should be migrated together in the same batch to avoid broken references.
            </p>
          </div>
        </div>
      )}

      {/* Most Depended Repos */}
      {mostDependedRepos.length > 0 && (
        <div 
          className="rounded-lg shadow-sm p-4"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
        >
          <h3 className="text-lg font-semibold mb-3" style={{ color: 'var(--fgColor-default)' }}>
            Most Depended Repositories
          </h3>
          <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
            These repositories are dependencies for many others. Migrate them first or include them in early batches.
          </p>
          <div className="flex flex-wrap gap-3">
            {mostDependedRepos.map(repo => (
              <Link
                key={repo.id}
                to={`/repository/${encodeURIComponent(repo.full_name)}`}
                className="flex items-center gap-2 px-3 py-2 rounded-lg hover:opacity-80 transition-opacity"
                style={{ 
                  backgroundColor: 'var(--bgColor-muted)',
                  color: 'var(--fgColor-accent)'
                }}
              >
                <span className="font-medium">{repo.full_name}</span>
                <Badge color="blue">{repo.depended_by_count} dependents</Badge>
              </Link>
            ))}
          </div>
        </div>
      )}

      {!hasData ? (
        <div 
          className="rounded-lg p-6"
          style={{
            backgroundColor: 'var(--accent-subtle)',
            border: '1px solid var(--borderColor-accent-muted)'
          }}
        >
          <h4 className="font-medium mb-2" style={{ color: 'var(--fgColor-accent)' }}>No Local Dependencies Found</h4>
          <p className="text-sm" style={{ color: 'var(--fgColor-accent)' }}>
            No local dependencies have been detected between repositories in your enterprise. 
            Dependencies are discovered during repository profiling (submodules, workflow references, dependency graph).
          </p>
        </div>
      ) : (
        <>
          {/* Filters and Search */}
          <div className="flex flex-wrap gap-4 items-center justify-between">
            <div className="flex gap-2">
              <button
                onClick={() => setTypeFilter('all')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: typeFilter === 'all' ? '#2da44e' : 'var(--control-bgColor-rest)',
                  color: typeFilter === 'all' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                All Types
              </button>
              <button
                onClick={() => setTypeFilter('submodule')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: typeFilter === 'submodule' ? '#0969DA' : 'var(--control-bgColor-rest)',
                  color: typeFilter === 'submodule' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                Submodule
              </button>
              <button
                onClick={() => setTypeFilter('workflow')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: typeFilter === 'workflow' ? '#8250DF' : 'var(--control-bgColor-rest)',
                  color: typeFilter === 'workflow' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                Workflow
              </button>
              <button
                onClick={() => setTypeFilter('dependency_graph')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: typeFilter === 'dependency_graph' ? '#1a7f37' : 'var(--control-bgColor-rest)',
                  color: typeFilter === 'dependency_graph' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                Dependency Graph
              </button>
              <button
                onClick={() => setTypeFilter('package')}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-opacity hover:opacity-80 cursor-pointer border-0"
                style={{
                  backgroundColor: typeFilter === 'package' ? '#656D76' : 'var(--control-bgColor-rest)',
                  color: typeFilter === 'package' ? '#ffffff' : 'var(--fgColor-default)'
                }}
              >
                Package
              </button>
            </div>
            
            <TextInput
              leadingVisual={SearchIcon}
              placeholder="Search repositories..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              style={{ width: 300 }}
            />
          </div>

          {/* View Tabs */}
          <div 
            className="rounded-lg shadow-sm"
            style={{ backgroundColor: 'var(--bgColor-default)' }}
          >
            <UnderlineNav aria-label="Dependency view mode">
              <UnderlineNav.Item
                aria-current={viewMode === 'list' ? 'page' : undefined}
                onSelect={() => setViewMode('list')}
              >
                List View
              </UnderlineNav.Item>
              <UnderlineNav.Item
                aria-current={viewMode === 'graph' ? 'page' : undefined}
                onSelect={() => setViewMode('graph')}
              >
                Graph View
              </UnderlineNav.Item>
            </UnderlineNav>

            <div className="p-6">
              {viewMode === 'list' ? (
                <DependencyListView 
                  nodes={paginatedNodes}
                  edges={filteredEdges}
                  totalNodes={filteredNodes.length}
                  currentPage={currentPage}
                  pageSize={pageSize}
                  onPageChange={setCurrentPage}
                />
              ) : (
                <DependencyGraphView 
                  nodes={filteredNodes}
                  edges={filteredEdges}
                />
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

interface DependencyListViewProps {
  nodes: DependencyGraphNode[];
  edges: DependencyGraphEdge[];
  totalNodes: number;
  currentPage: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

function DependencyListView({ nodes, edges, totalNodes, currentPage, pageSize, onPageChange }: DependencyListViewProps) {
  const getStatusColor = (status: string) => {
    if (status === 'complete' || status === 'migration_complete') return 'green';
    if (status === 'pending') return 'gray';
    if (status.includes('failed')) return 'red';
    if (status.includes('progress') || status.includes('queued')) return 'blue';
    return 'gray';
  };

  // Build a map of edges for quick lookup
  const edgeMap = useMemo(() => {
    const map = new Map<string, DependencyGraphEdge[]>();
    edges.forEach(edge => {
      const sourceEdges = map.get(edge.source) || [];
      sourceEdges.push(edge);
      map.set(edge.source, sourceEdges);
    });
    return map;
  }, [edges]);

  return (
    <div>
      <div className="mb-4">
        <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
          Repositories with Dependencies ({totalNodes})
        </h3>
        <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
          Click on a repository to see its full dependency details
        </p>
      </div>

      {nodes.length === 0 ? (
        <div className="text-center py-8" style={{ color: 'var(--fgColor-muted)' }}>
          No repositories match your search
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
                    Organization
                  </th>
                  <th 
                    className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                    style={{ color: 'var(--fgColor-muted)' }}
                  >
                    Status
                  </th>
                  <th 
                    className="px-4 py-3 text-center text-xs font-medium uppercase tracking-wider"
                    style={{ color: 'var(--fgColor-muted)' }}
                  >
                    Depends On
                  </th>
                  <th 
                    className="px-4 py-3 text-center text-xs font-medium uppercase tracking-wider"
                    style={{ color: 'var(--fgColor-muted)' }}
                  >
                    Depended By
                  </th>
                  <th 
                    className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                    style={{ color: 'var(--fgColor-muted)' }}
                  >
                    Dependencies
                  </th>
                </tr>
              </thead>
              <tbody 
                className="divide-y"
                style={{ borderColor: 'var(--borderColor-muted)' }}
              >
                {nodes.map((node) => {
                  const nodeEdges = edgeMap.get(node.id) || [];
                  return (
                    <tr key={node.id} className="hover:opacity-80 transition-opacity">
                      <td className="px-4 py-4 whitespace-nowrap">
                        <Link
                          to={`/repository/${encodeURIComponent(node.full_name)}`}
                          className="text-sm font-medium hover:underline"
                          style={{ color: 'var(--fgColor-accent)' }}
                        >
                          {node.full_name}
                        </Link>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        {node.organization}
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap">
                        <Badge color={getStatusColor(node.status)}>
                          {node.status.replace(/_/g, ' ')}
                        </Badge>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-center">
                        <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                          {node.depends_on_count}
                        </span>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-center">
                        <span className="text-sm font-semibold" style={{ color: node.depended_by_count > 0 ? 'var(--fgColor-success)' : 'var(--fgColor-default)' }}>
                          {node.depended_by_count}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex flex-wrap gap-1 max-w-md">
                          {nodeEdges.slice(0, 3).map((edge, idx) => (
                            <Link
                              key={idx}
                              to={`/repository/${encodeURIComponent(edge.target)}`}
                              className="text-xs px-2 py-1 rounded hover:opacity-80"
                              style={{ 
                                backgroundColor: 'var(--bgColor-muted)',
                                color: 'var(--fgColor-accent)'
                              }}
                            >
                              {edge.target.split('/').pop()}
                            </Link>
                          ))}
                          {nodeEdges.length > 3 && (
                            <span className="text-xs px-2 py-1" style={{ color: 'var(--fgColor-muted)' }}>
                              +{nodeEdges.length - 3} more
                            </span>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {totalNodes > pageSize && (
            <div className="mt-4">
              <Pagination
                currentPage={currentPage}
                totalItems={totalNodes}
                pageSize={pageSize}
                onPageChange={onPageChange}
              />
            </div>
          )}
        </>
      )}
    </div>
  );
}

interface DependencyGraphViewProps {
  nodes: DependencyGraphNode[];
  edges: DependencyGraphEdge[];
}

function DependencyGraphView({ nodes, edges }: DependencyGraphViewProps) {
  // Convert to Sankey format
  const sankeyData = useMemo(() => {
    if (nodes.length === 0 || edges.length === 0) return null;

    // Create node index map
    const nodeIndexMap = new Map<string, number>();
    const sankeyNodes = nodes.map((node, index) => {
      nodeIndexMap.set(node.id, index);
      return { name: node.full_name };
    });

    // Create links with proper indices
    const sankeyLinks = edges
      .filter(edge => nodeIndexMap.has(edge.source) && nodeIndexMap.has(edge.target))
      .map(edge => ({
        source: nodeIndexMap.get(edge.source)!,
        target: nodeIndexMap.get(edge.target)!,
        value: 1,
        type: edge.dependency_type
      }));

    // Only return if we have valid links
    if (sankeyLinks.length === 0) return null;

    return { nodes: sankeyNodes, links: sankeyLinks };
  }, [nodes, edges]);

  if (!sankeyData) {
    return (
      <div className="text-center py-12" style={{ color: 'var(--fgColor-muted)' }}>
        <p className="mb-2">Not enough data to display graph visualization</p>
        <p className="text-sm">Try the List View for a detailed breakdown of dependencies</p>
      </div>
    );
  }

  // For large graphs, show a message and suggest list view
  if (nodes.length > 50) {
    return (
      <div className="text-center py-12" style={{ color: 'var(--fgColor-muted)' }}>
        <p className="mb-2">Graph visualization works best with fewer repositories</p>
        <p className="text-sm mb-4">You have {nodes.length} repositories with dependencies</p>
        <p className="text-sm">Use the search filter to narrow down the view, or use List View for full details</p>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-4">
        <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
          Dependency Flow
        </h3>
        <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
          Visualizing how repositories depend on each other (source → target)
        </p>
      </div>

      <div style={{ width: '100%', height: 500 }}>
        <ResponsiveContainer>
          <Sankey
            data={sankeyData}
            node={{
              fill: '#0969DA',
              opacity: 0.8
            }}
            link={{
              stroke: '#656D76',
              opacity: 0.3
            }}
            nodePadding={50}
            margin={{ top: 20, right: 200, bottom: 20, left: 200 }}
          >
            <Tooltip
              contentStyle={{
                backgroundColor: 'rgba(27, 31, 36, 0.95)',
                border: '1px solid rgba(255, 255, 255, 0.1)',
                borderRadius: '6px',
                color: '#ffffff',
                padding: '8px 12px'
              }}
            />
          </Sankey>
        </ResponsiveContainer>
      </div>

      <div className="mt-4 text-center">
        <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
          Tip: Use the List View for detailed information and to navigate to specific repositories
        </p>
      </div>
    </div>
  );
}
