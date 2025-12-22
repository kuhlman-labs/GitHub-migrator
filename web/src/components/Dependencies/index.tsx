import { useState, useEffect, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { UnderlineNav } from '@primer/react';
import { AlertIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { DependencyGraphResponse } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Badge } from '../common/Badge';
import { OrgAggregatedView } from './OrgAggregatedView';
import { DependencyListView } from './DependencyListView';
import { DependencyFilters, DependencyTypeFilter } from './DependencyFilters';
import { DependencyExport } from './DependencyExport';

type ViewMode = 'list' | 'org';

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
  const [showCircularOnly, setShowCircularOnly] = useState(false);
  const pageSize = 25;

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

  // Reset page when search or filter changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, typeFilter, showCircularOnly]);

  // Compute repos that have circular dependencies (bidirectional relationships)
  const circularDependencyRepos = useMemo(() => {
    if (!data?.edges) return new Set<string>();
    
    // Build edge set for quick lookup
    const edgeSet = new Set<string>();
    data.edges.forEach(edge => {
      edgeSet.add(`${edge.source}|${edge.target}`);
    });
    
    // Find repos involved in circular dependencies
    const circularRepos = new Set<string>();
    data.edges.forEach(edge => {
      const reverseKey = `${edge.target}|${edge.source}`;
      if (edgeSet.has(reverseKey)) {
        circularRepos.add(edge.source);
        circularRepos.add(edge.target);
      }
    });
    
    return circularRepos;
  }, [data?.edges]);

  // Check if any filters are active
  const hasActiveFilters = searchQuery !== '' || showCircularOnly || typeFilter !== 'all';

  // Filter and search nodes
  const filteredNodes = useMemo(() => {
    if (!data?.nodes) return [];
    
    let nodes = data.nodes;
    
    // Filter by circular dependencies if enabled
    if (showCircularOnly) {
      nodes = nodes.filter(node => circularDependencyRepos.has(node.id));
    }
    
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
  }, [data?.nodes, searchQuery, showCircularOnly, circularDependencyRepos]);

  // Filter edges based on filtered nodes (respects all filters: search, circular, type)
  const filteredEdges = useMemo(() => {
    if (!data?.edges) return [];
    
    // If no filters are active, return all edges
    if (!searchQuery && !showCircularOnly) return data.edges;
    
    // Filter edges to include those where at least one endpoint is in filtered nodes
    // This preserves visibility of relationships to external repositories
    const nodeIds = new Set(filteredNodes.map(n => n.id));
    return data.edges.filter(edge => 
      nodeIds.has(edge.source) || nodeIds.has(edge.target)
    );
  }, [data?.edges, filteredNodes, searchQuery, showCircularOnly]);

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

  // hasData indicates if the current filter has data
  const hasFilteredData = stats.total_repos_with_dependencies > 0;
  
  // Check if there's any dependency data at all (used to show the global empty state)
  // We consider data exists if we have nodes or edges, or if we're filtering (meaning unfiltered might have data)
  const hasAnyData = hasFilteredData || typeFilter !== 'all';

  return (
    <div className="relative space-y-6">
      <RefreshIndicator isRefreshing={refreshing} />
      
      {/* Header */}
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            Dependency Explorer
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Visualize and analyze local dependencies between repositories for migration batch planning
          </p>
        </div>
        
        {/* Export Button with Dropdown */}
        <DependencyExport
          filteredNodes={filteredNodes}
          filteredEdges={filteredEdges}
          hasActiveFilters={hasActiveFilters}
          hasFilteredData={hasFilteredData}
        />
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
        <button 
          onClick={() => circularDependencyRepos.size > 0 && setShowCircularOnly(!showCircularOnly)}
          className={`rounded-lg shadow-sm p-4 text-left w-full transition-all ${circularDependencyRepos.size > 0 ? 'cursor-pointer hover:ring-2 hover:ring-[var(--borderColor-attention)]' : ''}`}
          style={{ 
            backgroundColor: showCircularOnly ? 'var(--attention-subtle)' : 'var(--bgColor-default)',
            border: showCircularOnly ? '2px solid var(--borderColor-attention)' : '2px solid transparent'
          }}
          disabled={circularDependencyRepos.size === 0}
        >
          <div className="text-sm flex items-center gap-2" style={{ color: 'var(--fgColor-muted)' }}>
            Circular Dependencies
            {circularDependencyRepos.size > 0 && (
              <span className="text-xs px-1.5 py-0.5 rounded" style={{ 
                backgroundColor: showCircularOnly ? 'var(--fgColor-attention)' : 'var(--attention-subtle)', 
                color: showCircularOnly ? 'var(--bgColor-default)' : 'var(--fgColor-attention)' 
              }}>
                {showCircularOnly ? 'Filtered' : 'Click to filter'}
              </span>
            )}
          </div>
          <div className="text-2xl font-bold mt-1" style={{ color: circularDependencyRepos.size > 0 ? 'var(--fgColor-attention)' : 'var(--fgColor-default)' }}>
            {circularDependencyRepos.size}
          </div>
          <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Repos in {stats.circular_dependency_count} bidirectional relationship{stats.circular_dependency_count !== 1 ? 's' : ''}
          </div>
        </button>
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

      {/* Show global empty state only when there's no data at all */}
      {!hasAnyData ? (
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
          {/* Circular Dependencies Filter Indicator */}
          {showCircularOnly && (
            <div 
              className="rounded-lg p-3 flex items-center justify-between"
              style={{
                backgroundColor: 'var(--attention-subtle)',
                border: '1px solid var(--borderColor-attention)'
              }}
            >
              <div className="flex items-center gap-2">
                <span style={{ color: 'var(--fgColor-attention)' }}><AlertIcon size={16} /></span>
                <span className="text-sm font-medium" style={{ color: 'var(--fgColor-attention)' }}>
                  Showing {circularDependencyRepos.size} repositories with circular dependencies
                </span>
                <span className="text-sm" style={{ color: 'var(--fgColor-attention)' }}>
                  — These should be migrated together in the same batch to avoid broken references.
                </span>
              </div>
              <button
                onClick={() => setShowCircularOnly(false)}
                className="px-3 py-1 rounded text-sm font-medium transition-opacity hover:opacity-80"
                style={{
                  backgroundColor: 'var(--fgColor-attention)',
                  color: 'var(--bgColor-default)'
                }}
              >
                Clear Filter
              </button>
            </div>
          )}

          {/* Filters and Search */}
          <DependencyFilters
            typeFilter={typeFilter}
            onTypeFilterChange={setTypeFilter}
            searchQuery={searchQuery}
            onSearchQueryChange={setSearchQuery}
          />

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
                aria-current={viewMode === 'org' ? 'page' : undefined}
                onSelect={() => setViewMode('org')}
              >
                Organization View
              </UnderlineNav.Item>
            </UnderlineNav>

            <div className="p-6">
              {/* Show filter-specific empty state when current filter has no results */}
              {!hasFilteredData ? (
                <div className="text-center py-8">
                  <p className="text-lg mb-2" style={{ color: 'var(--fgColor-default)' }}>
                    No {typeFilter === 'dependency_graph' ? 'Dependency Graph' : typeFilter.charAt(0).toUpperCase() + typeFilter.slice(1)} Dependencies Found
                  </p>
                  <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                    No local dependencies of type "{typeFilter.replace('_', ' ')}" have been detected.
                    Try selecting a different dependency type or click "All Types" to see all dependencies.
                  </p>
                </div>
              ) : viewMode === 'list' ? (
                <DependencyListView 
                  nodes={paginatedNodes}
                  edges={filteredEdges}
                  allNodes={data?.nodes || []}
                  totalNodes={filteredNodes.length}
                  currentPage={currentPage}
                  pageSize={pageSize}
                  onPageChange={setCurrentPage}
                />
              ) : (
                <OrgAggregatedView 
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
