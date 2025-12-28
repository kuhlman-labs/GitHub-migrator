import { useState, useEffect, useMemo, useRef } from 'react';
import { Link } from 'react-router-dom';
import type { DependencyGraphNode, DependencyGraphEdge } from '../../types';
import { Badge } from '../common/Badge';
import { Pagination } from '../common/Pagination';

interface DependencyListViewProps {
  nodes: DependencyGraphNode[];
  edges: DependencyGraphEdge[];
  allNodes: DependencyGraphNode[];
  totalNodes: number;
  currentPage: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

export function DependencyListView({ nodes, edges, allNodes, totalNodes, currentPage, pageSize, onPageChange }: DependencyListViewProps) {
  const [focusedRepo, setFocusedRepo] = useState<string | null>(null);
  const [selectedRowIndex, setSelectedRowIndex] = useState<number>(-1);
  const tableRef = useRef<HTMLTableElement>(null);

  const getStatusColor = (status: string) => {
    if (status === 'complete' || status === 'migration_complete') return 'green';
    if (status === 'pending') return 'gray';
    if (status.includes('failed')) return 'red';
    if (status.includes('progress') || status.includes('queued')) return 'blue';
    return 'gray';
  };

  // Build maps for both directions: depends_on and depended_by
  const { dependsOnMap, dependedByMap } = useMemo(() => {
    const dependsOn = new Map<string, DependencyGraphEdge[]>();
    const dependedBy = new Map<string, DependencyGraphEdge[]>();
    
    edges.forEach(edge => {
      // Source depends on target
      const sourceEdges = dependsOn.get(edge.source) || [];
      sourceEdges.push(edge);
      dependsOn.set(edge.source, sourceEdges);
      
      // Target is depended by source
      const targetEdges = dependedBy.get(edge.target) || [];
      targetEdges.push({ ...edge, source: edge.target, target: edge.source });
      dependedBy.set(edge.target, targetEdges);
    });
    
    return { dependsOnMap: dependsOn, dependedByMap: dependedBy };
  }, [edges]);

  // Get the focused node details
  const focusedNodeData = useMemo(() => {
    if (!focusedRepo) return null;
    const node = allNodes.find(n => n.id === focusedRepo);
    if (!node) return null;
    
    const dependsOn = dependsOnMap.get(focusedRepo) || [];
    const dependedBy = dependedByMap.get(focusedRepo) || [];
    
    return { node, dependsOn, dependedBy };
  }, [focusedRepo, allNodes, dependsOnMap, dependedByMap]);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!tableRef.current) return;
      
      // Only handle if focus is within the component or there's an active selection
      const hasFocusInTable = tableRef.current.contains(document.activeElement);
      const hasActiveSelection = selectedRowIndex >= 0;
      
      if (!hasFocusInTable && !hasActiveSelection) return;
      
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setSelectedRowIndex(prev => Math.min(prev + 1, nodes.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setSelectedRowIndex(prev => Math.max(prev - 1, 0));
          break;
        case 'Enter':
          if (selectedRowIndex >= 0 && selectedRowIndex < nodes.length) {
            e.preventDefault();
            setFocusedRepo(nodes[selectedRowIndex].id);
          }
          break;
        case 'Escape':
          e.preventDefault();
          setFocusedRepo(null);
          setSelectedRowIndex(-1);
          // Keep focus on table so user can continue keyboard navigation
          tableRef.current?.focus();
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [nodes, selectedRowIndex]);

  // Reset selected row when nodes change (e.g., pagination)
  useEffect(() => {
    // Use setTimeout to avoid synchronous setState in effect body
    const timer = setTimeout(() => {
      setSelectedRowIndex(-1);
    }, 0);
    return () => clearTimeout(timer);
  }, [nodes]);

  // Scroll selected row into view
  useEffect(() => {
    if (selectedRowIndex >= 0 && tableRef.current) {
      const rows = tableRef.current.querySelectorAll('tbody tr');
      rows[selectedRowIndex]?.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedRowIndex]);

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            Repositories with Dependencies ({totalNodes})
          </h3>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Click the focus button to explore a repository's connections. Use arrow keys to navigate, Enter to focus, Escape to clear.
          </p>
        </div>
        {focusedRepo && (
          <button
            onClick={() => setFocusedRepo(null)}
            className="px-3 py-1.5 rounded text-sm font-medium transition-colors"
            style={{ 
              backgroundColor: 'var(--accent-subtle)',
              color: 'var(--fgColor-accent)',
              border: '1px solid var(--borderColor-accent-muted)'
            }}
          >
            Clear Focus
          </button>
        )}
      </div>

      {/* Focus Mode Panel */}
      {focusedNodeData && (
        <div 
          className="mb-4 rounded-lg p-4"
          style={{ 
            backgroundColor: 'var(--accent-subtle)',
            border: '1px solid var(--borderColor-accent-muted)'
          }}
        >
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-3">
              <h4 className="text-lg font-semibold" style={{ color: 'var(--fgColor-accent)' }}>
                {focusedNodeData.node.full_name}
              </h4>
              <Badge color={getStatusColor(focusedNodeData.node.status)}>
                {focusedNodeData.node.status.replace(/_/g, ' ')}
              </Badge>
            </div>
            <Link
              to={`/repository/${encodeURIComponent(focusedNodeData.node.full_name)}`}
              className="text-sm px-3 py-1 rounded"
              style={{ 
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-accent)'
              }}
            >
              View Repository →
            </Link>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Depends On */}
            <div 
              className="rounded-lg p-3"
              style={{ backgroundColor: 'var(--bgColor-default)' }}
            >
              <h5 className="text-sm font-medium mb-2 flex items-center gap-2" style={{ color: 'var(--fgColor-default)' }}>
                <span>Depends On</span>
                <Badge color="blue">{focusedNodeData.dependsOn.length}</Badge>
              </h5>
              {focusedNodeData.dependsOn.length === 0 ? (
                <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>No dependencies</p>
              ) : (
                <div className="space-y-1 max-h-40 overflow-y-auto">
                  {focusedNodeData.dependsOn.map((edge, idx) => (
                    <div 
                      key={idx}
                      className="flex items-center justify-between text-sm py-1 px-2 rounded hover:bg-[var(--bgColor-muted)]"
                    >
                      <button
                        onClick={() => setFocusedRepo(edge.target)}
                        className="text-left hover:underline truncate"
                        style={{ color: 'var(--fgColor-accent)' }}
                      >
                        {edge.target}
                      </button>
                      <span 
                        className="text-xs px-1.5 py-0.5 rounded ml-2 flex-shrink-0"
                        style={{ 
                          backgroundColor: 'var(--bgColor-muted)',
                          color: 'var(--fgColor-muted)'
                        }}
                      >
                        {edge.dependency_type}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Depended By */}
            <div 
              className="rounded-lg p-3"
              style={{ backgroundColor: 'var(--bgColor-default)' }}
            >
              <h5 className="text-sm font-medium mb-2 flex items-center gap-2" style={{ color: 'var(--fgColor-default)' }}>
                <span>Depended By</span>
                <Badge color="green">{focusedNodeData.dependedBy.length}</Badge>
              </h5>
              {focusedNodeData.dependedBy.length === 0 ? (
                <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>No dependents</p>
              ) : (
                <div className="space-y-1 max-h-40 overflow-y-auto">
                  {focusedNodeData.dependedBy.map((edge, idx) => (
                    <div 
                      key={idx}
                      className="flex items-center justify-between text-sm py-1 px-2 rounded hover:bg-[var(--bgColor-muted)]"
                    >
                      <button
                        onClick={() => setFocusedRepo(edge.target)}
                        className="text-left hover:underline truncate"
                        style={{ color: 'var(--fgColor-accent)' }}
                      >
                        {edge.target}
                      </button>
                      <span 
                        className="text-xs px-1.5 py-0.5 rounded ml-2 flex-shrink-0"
                        style={{ 
                          backgroundColor: 'var(--bgColor-muted)',
                          color: 'var(--fgColor-muted)'
                        }}
                      >
                        {edge.dependency_type}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {nodes.length === 0 ? (
        <div className="text-center py-8" style={{ color: 'var(--fgColor-muted)' }}>
          No repositories match your search
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table 
              ref={tableRef}
              className="min-w-full divide-y"
              style={{ borderColor: 'var(--borderColor-muted)' }}
              tabIndex={0}
            >
              <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                <tr>
                  <th 
                    className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider"
                    style={{ color: 'var(--fgColor-muted)', width: '40px' }}
                  >
                    Focus
                  </th>
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
                {nodes.map((node, index) => {
                  const nodeEdges = dependsOnMap.get(node.id) || [];
                  const isSelected = selectedRowIndex === index;
                  const isFocused = focusedRepo === node.id;
                  
                  return (
                    <tr 
                      key={node.id} 
                      className="transition-all"
                      style={{
                        backgroundColor: isFocused 
                          ? 'var(--accent-subtle)' 
                          : isSelected 
                            ? 'var(--bgColor-muted)' 
                            : undefined,
                        outline: isSelected ? '2px solid var(--borderColor-accent-muted)' : undefined,
                        outlineOffset: '-2px'
                      }}
                      onClick={() => setSelectedRowIndex(index)}
                    >
                      <td className="px-4 py-4 whitespace-nowrap">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setFocusedRepo(focusedRepo === node.id ? null : node.id);
                          }}
                          className="p-1 rounded transition-colors"
                          style={{ 
                            backgroundColor: isFocused ? 'var(--fgColor-accent)' : 'var(--control-bgColor-rest)',
                            color: isFocused ? 'white' : 'var(--fgColor-muted)'
                          }}
                          title={isFocused ? 'Clear focus' : 'Focus on this repository'}
                        >
                          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l2.817 2.817a.75.75 0 11-1.06 1.06l-2.817-2.816A6 6 0 012 8z" />
                          </svg>
                        </button>
                      </td>
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
                            <button
                              key={idx}
                              onClick={(e) => {
                                e.stopPropagation();
                                setFocusedRepo(edge.target);
                              }}
                              className="text-xs px-2 py-1 rounded hover:opacity-80"
                              style={{ 
                                backgroundColor: 'var(--bgColor-muted)',
                                color: 'var(--fgColor-accent)'
                              }}
                            >
                              {edge.target.split('/').pop()}
                            </button>
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

