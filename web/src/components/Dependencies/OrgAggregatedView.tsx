import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import ForceGraph2D, { ForceGraphMethods, NodeObject, LinkObject } from 'react-force-graph-2d';
import { Link } from 'react-router-dom';
import { CircleIcon, ArrowRightIcon, SearchIcon, ChevronDownIcon } from '@primer/octicons-react';
import type { DependencyGraphNode, DependencyGraphEdge } from '../../types';
import { Badge } from '../common/Badge';
import { Pagination } from '../common/Pagination';
import * as d3 from 'd3-force';

interface OrgAggregatedViewProps {
  nodes: DependencyGraphNode[];
  edges: DependencyGraphEdge[];
}

interface OrgNode extends NodeObject {
  id: string;
  name: string;
  repoCount: number;
  totalDependsOn: number;
  totalDependedBy: number;
  isExpanded: boolean;
  isSearchMatch: boolean;
  color: string;
}

interface OrgLink extends LinkObject {
  source: string | OrgNode;
  target: string | OrgNode;
  dependencyCount: number;
  hasBidirectional: boolean;  // True if there's also a reverse link
}

interface OrgStats {
  repoCount: number;
  totalDependsOn: number;
  totalDependedBy: number;
  repos: DependencyGraphNode[];
}

// Color palette for organizations
const ORG_COLORS = [
  '#0969DA', '#8250DF', '#1a7f37', '#bf3989', '#cf222e',
  '#9a6700', '#0550ae', '#6639ba', '#116329', '#a40e26',
  '#953800', '#0349b4', '#5e4db2', '#1b7c83', '#8c1c13',
];

export function OrgAggregatedView({ nodes, edges }: OrgAggregatedViewProps) {
  const graphRef = useRef<ForceGraphMethods<OrgNode, OrgLink> | undefined>(undefined);
  const containerRef = useRef<HTMLDivElement>(null);
  
  const [selectedOrg, setSelectedOrg] = useState<string | null>(null);
  const [hoveredOrg, setHoveredOrg] = useState<string | null>(null);
  const [dimensions, setDimensions] = useState({ width: 800, height: 500 });
  const [showOrgDropdown, setShowOrgDropdown] = useState(false);
  const [orgSearchFilter, setOrgSearchFilter] = useState('');
  const [repoPage, setRepoPage] = useState(1);
  const repoPageSize = 25;

  // Track container size
  useEffect(() => {
    const updateDimensions = () => {
      if (containerRef.current) {
        const { width } = containerRef.current.getBoundingClientRect();
        setDimensions({ width, height: 500 });
      }
    };
    
    updateDimensions();
    window.addEventListener('resize', updateDimensions);
    return () => window.removeEventListener('resize', updateDimensions);
  }, []);

  // Aggregate nodes by organization
  const orgStats = useMemo(() => {
    const stats = new Map<string, OrgStats>();
    
    nodes.forEach(node => {
      const org = node.organization;
      if (!stats.has(org)) {
        stats.set(org, {
          repoCount: 0,
          totalDependsOn: 0,
          totalDependedBy: 0,
          repos: [],
        });
      }
      const orgStat = stats.get(org)!;
      orgStat.repoCount++;
      orgStat.totalDependsOn += node.depends_on_count;
      orgStat.totalDependedBy += node.depended_by_count;
      orgStat.repos.push(node);
    });
    
    return stats;
  }, [nodes]);

  // Build org-to-org edges
  const orgEdges = useMemo(() => {
    const edgeMap = new Map<string, number>();
    
    // Build a map of repo ID to org
    const repoToOrg = new Map<string, string>();
    nodes.forEach(node => {
      repoToOrg.set(node.id, node.organization);
    });
    
    edges.forEach(edge => {
      const sourceOrg = repoToOrg.get(edge.source);
      const targetOrg = repoToOrg.get(edge.target);
      
      if (sourceOrg && targetOrg && sourceOrg !== targetOrg) {
        const key = `${sourceOrg}|${targetOrg}`;
        edgeMap.set(key, (edgeMap.get(key) || 0) + 1);
      }
    });
    
    return edgeMap;
  }, [nodes, edges]);

  // Build color map
  const orgColorMap = useMemo(() => {
    const orgs = [...orgStats.keys()].sort();
    const map = new Map<string, string>();
    orgs.forEach((org, i) => {
      map.set(org, ORG_COLORS[i % ORG_COLORS.length]);
    });
    return map;
  }, [orgStats]);

  // Get orgs connected to a specific org (from all edges, not just filtered)
  const getConnectedOrgsFromAllEdges = useCallback((orgId: string): Set<string> => {
    const connected = new Set<string>();
    [...orgEdges.keys()].forEach(edgeKey => {
      const [source, target] = edgeKey.split('|');
      if (source === orgId) connected.add(target);
      if (target === orgId) connected.add(source);
    });
    return connected;
  }, [orgEdges]);

  // Sorted list of orgs for the dropdown
  const sortedOrgList = useMemo(() => {
    return [...orgStats.entries()]
      .sort((a, b) => b[1].repoCount - a[1].repoCount)
      .map(([name, stats]) => ({ name, repoCount: stats.repoCount }));
  }, [orgStats]);

  // Transform to force graph format
  const graphData = useMemo(() => {
    // Start with all orgs
    let allOrgsToShow = new Set([...orgStats.keys()]);
    
    // If an org is selected, filter to show only that org and its connections
    if (selectedOrg) {
      const selectedOrgConnections = getConnectedOrgsFromAllEdges(selectedOrg);
      allOrgsToShow = new Set([selectedOrg, ...selectedOrgConnections]);
    }
    
    const graphNodes: OrgNode[] = [...orgStats.entries()]
      .filter(([orgName]) => allOrgsToShow.has(orgName))
      .map(([orgName, stats]) => ({
        id: orgName,
        name: orgName,
        repoCount: stats.repoCount,
        totalDependsOn: stats.totalDependsOn,
        totalDependedBy: stats.totalDependedBy,
        isExpanded: selectedOrg === orgName,
        isSearchMatch: selectedOrg ? orgName === selectedOrg : true,
        color: orgColorMap.get(orgName) || '#0969DA',
      }));

    const nodeIds = new Set(graphNodes.map(n => n.id));
    
    const graphLinks: OrgLink[] = [...orgEdges.entries()]
      .filter(([key]) => {
        const [source, target] = key.split('|');
        return nodeIds.has(source) && nodeIds.has(target);
      })
      .map(([key, count]) => {
        const [source, target] = key.split('|');
        // Check if there's a reverse link (bidirectional)
        const reverseKey = `${target}|${source}`;
        const hasBidirectional = orgEdges.has(reverseKey);
        return {
          source,
          target,
          dependencyCount: count,
          hasBidirectional,
        };
      });

    return { nodes: graphNodes, links: graphLinks };
  }, [orgStats, orgEdges, selectedOrg, orgColorMap, getConnectedOrgsFromAllEdges]);

  // Get connected orgs for highlighting
  const getConnectedOrgs = useCallback((orgId: string): Set<string> => {
    const connected = new Set<string>();
    graphData.links.forEach(link => {
      const sourceId = typeof link.source === 'object' ? link.source.id : link.source;
      const targetId = typeof link.target === 'object' ? link.target.id : link.target;
      if (sourceId === orgId) connected.add(targetId);
      if (targetId === orgId) connected.add(sourceId);
    });
    return connected;
  }, [graphData.links]);

  // Check if we're in filtered mode (org is selected)
  const isFilteredMode = useMemo(() => {
    return selectedOrg !== null;
  }, [selectedOrg]);

  // Node paint function
  const paintNode = useCallback((node: OrgNode, ctx: CanvasRenderingContext2D, globalScale: number) => {
    // Check for undefined/null coordinates (0 is a valid coordinate)
    if (node.x == null || node.y == null) return;
    
    const fontSize = Math.max(14 / globalScale, 5);
    const nodeSize = Math.max(15, Math.min(50, 15 + Math.sqrt(node.repoCount) * 5));
    
    // Determine opacity
    let opacity = 1;
    const connectedOrgs = selectedOrg ? getConnectedOrgs(selectedOrg) : new Set<string>();
    if (selectedOrg && selectedOrg !== node.id && !connectedOrgs.has(node.id)) {
      opacity = 0.2;
    }
    if (hoveredOrg && hoveredOrg !== node.id) {
      const hoveredConnected = getConnectedOrgs(hoveredOrg);
      if (!hoveredConnected.has(node.id)) {
        opacity = Math.min(opacity, 0.4);
      }
    }
    // Dim non-matching orgs in filtered mode (they're just shown for context)
    if (isFilteredMode && !node.isSearchMatch && !hoveredOrg) {
      opacity = 0.5;
    }

    ctx.globalAlpha = opacity;

    // Extract coordinates (we know they're defined from the check above)
    const x = node.x;
    const y = node.y;

    // Draw node circle
    ctx.beginPath();
    ctx.arc(x, y, nodeSize, 0, 2 * Math.PI);
    ctx.fillStyle = node.color;
    ctx.fill();
    
    // Draw border for selected or search-matched orgs
    if (selectedOrg === node.id) {
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 4 / globalScale;
      ctx.stroke();
    } else if (isFilteredMode && node.isSearchMatch) {
      // Highlight border for search-matched orgs
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 3 / globalScale;
      ctx.stroke();
    }

    // Draw org name
    ctx.font = `bold ${fontSize}px 'Mona Sans', sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillStyle = '#ffffff';
    ctx.fillText(node.name, x, y);
    
    // Draw repo count badge
    const badgeFontSize = Math.max(10 / globalScale, 4);
    ctx.font = `${badgeFontSize}px 'Mona Sans', sans-serif`;
    ctx.fillStyle = 'rgba(255,255,255,0.8)';
    ctx.fillText(`${node.repoCount} repos`, x, y + nodeSize + badgeFontSize + 2);

    ctx.globalAlpha = 1;
  }, [selectedOrg, hoveredOrg, getConnectedOrgs, isFilteredMode]);

  // Link paint function
  const paintLink = useCallback((link: OrgLink, ctx: CanvasRenderingContext2D, globalScale: number) => {
    const source = link.source as OrgNode;
    const target = link.target as OrgNode;
    
    // Check for undefined/null explicitly (0 is a valid coordinate)
    if (source.x == null || source.y == null || target.x == null || target.y == null) return;

    const sourceId = source.id;
    const targetId = target.id;
    
    let opacity = 0.4;
    let lineWidth = Math.max(1, Math.min(6, link.dependencyCount / 15 + 1));
    
    if (selectedOrg) {
      if (sourceId === selectedOrg || targetId === selectedOrg) {
        opacity = 0.9;
        lineWidth = Math.max(lineWidth, 2);
      } else {
        opacity = 0.1;
      }
    }
    if (hoveredOrg && (sourceId === hoveredOrg || targetId === hoveredOrg)) {
      opacity = Math.max(opacity, 0.7);
    }

    // Extract coordinates (we know they're defined from the check above)
    const sx = source.x;
    const sy = source.y;
    const tx = target.x;
    const ty = target.y;

    const dx = tx - sx;
    const dy = ty - sy;
    const distance = Math.sqrt(dx * dx + dy * dy);
    const angle = Math.atan2(dy, dx);
    
    // For bidirectional links, curve the line slightly to separate them
    const curvature = link.hasBidirectional ? 0.15 : 0;
    
    // Calculate perpendicular offset for curve
    const perpX = -dy / distance * curvature * distance;
    const perpY = dx / distance * curvature * distance;
    
    // Control point for quadratic curve
    const cpX = (sx + tx) / 2 + perpX;
    const cpY = (sy + ty) / 2 + perpY;
    
    // Draw curved or straight link
    ctx.beginPath();
    ctx.moveTo(sx, sy);
    if (link.hasBidirectional) {
      ctx.quadraticCurveTo(cpX, cpY, tx, ty);
    } else {
      ctx.lineTo(tx, ty);
    }
    ctx.strokeStyle = `rgba(150, 150, 150, ${opacity})`;
    ctx.lineWidth = lineWidth / globalScale;
    ctx.stroke();

    // Calculate position along the curve for arrow and label
    // For bidirectional, place at different positions to avoid overlap
    const labelT = link.hasBidirectional ? 0.35 : 0.5;
    const arrowT = 0.75;
    
    // Get point on curve at parameter t
    const getPointOnCurve = (t: number) => {
      if (link.hasBidirectional) {
        // Quadratic bezier: (1-t)²*P0 + 2(1-t)t*CP + t²*P1
        const mt = 1 - t;
        return {
          x: mt * mt * sx + 2 * mt * t * cpX + t * t * tx,
          y: mt * mt * sy + 2 * mt * t * cpY + t * t * ty,
        };
      } else {
        return {
          x: sx + dx * t,
          y: sy + dy * t,
        };
      }
    };
    
    // Get tangent angle at point on curve
    const getTangentAngle = (t: number) => {
      if (link.hasBidirectional) {
        // Derivative of quadratic bezier
        const mt = 1 - t;
        const tangentX = 2 * mt * (cpX - sx) + 2 * t * (tx - cpX);
        const tangentY = 2 * mt * (cpY - sy) + 2 * t * (ty - cpY);
        return Math.atan2(tangentY, tangentX);
      } else {
        return angle;
      }
    };

    // Draw arrow to show direction
    if (globalScale > 0.3 && opacity > 0.15) {
      const arrowPoint = getPointOnCurve(arrowT);
      const arrowAngle = getTangentAngle(arrowT);
      const arrowSize = Math.max(8 / globalScale, 4);
      
      ctx.beginPath();
      ctx.moveTo(arrowPoint.x, arrowPoint.y);
      ctx.lineTo(
        arrowPoint.x - arrowSize * Math.cos(arrowAngle - Math.PI / 7),
        arrowPoint.y - arrowSize * Math.sin(arrowAngle - Math.PI / 7)
      );
      ctx.moveTo(arrowPoint.x, arrowPoint.y);
      ctx.lineTo(
        arrowPoint.x - arrowSize * Math.cos(arrowAngle + Math.PI / 7),
        arrowPoint.y - arrowSize * Math.sin(arrowAngle + Math.PI / 7)
      );
      ctx.strokeStyle = `rgba(200, 200, 200, ${opacity})`;
      ctx.lineWidth = Math.max(2 / globalScale, 1);
      ctx.stroke();
    }

    // Draw dependency count with background for visibility
    if (globalScale > 0.35 && opacity > 0.15) {
      const labelPoint = getPointOnCurve(labelT);
      const labelFontSize = Math.max(11 / globalScale, 5);
      const label = String(link.dependencyCount);
      
      ctx.font = `bold ${labelFontSize}px 'Mona Sans', sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      
      // Measure text for background
      const textMetrics = ctx.measureText(label);
      const padding = 4 / globalScale;
      const bgWidth = textMetrics.width + padding * 2;
      const bgHeight = labelFontSize + padding * 1.5;
      
      // Draw background pill
      ctx.fillStyle = `rgba(13, 17, 23, ${Math.min(opacity + 0.4, 0.95)})`;
      ctx.beginPath();
      ctx.roundRect(labelPoint.x - bgWidth / 2, labelPoint.y - bgHeight / 2, bgWidth, bgHeight, 3 / globalScale);
      ctx.fill();
      
      // Draw border
      ctx.strokeStyle = `rgba(100, 100, 100, ${opacity * 0.6})`;
      ctx.lineWidth = 1 / globalScale;
      ctx.stroke();
      
      // Draw text
      ctx.fillStyle = `rgba(255, 255, 255, ${Math.min(opacity + 0.3, 1)})`;
      ctx.fillText(label, labelPoint.x, labelPoint.y);
    }
  }, [selectedOrg, hoveredOrg]);

  // Configure forces and zoom to fit when graph data changes
  useEffect(() => {
    if (graphRef.current && graphData.nodes.length > 0) {
      const fg = graphRef.current;
      
      // Configure charge force for stronger repulsion
      const chargeForce = fg.d3Force('charge') as d3.ForceManyBody<OrgNode> | undefined;
      if (chargeForce) {
        chargeForce.strength(-800);
      }
      
      // Configure link force for longer distances
      const linkForce = fg.d3Force('link') as d3.ForceLink<OrgNode, OrgLink> | undefined;
      if (linkForce) {
        linkForce.distance(180);
      }
      
      // Add collision force to prevent node overlap
      fg.d3Force('collision', d3.forceCollide<OrgNode>()
        .radius((node) => Math.max(15, Math.min(50, 15 + Math.sqrt(node.repoCount) * 5)) + 30)
        .strength(1)
        .iterations(3)
      );
      
      // Reheat the simulation to apply the new forces
      fg.d3ReheatSimulation();
      
      setTimeout(() => {
        graphRef.current?.zoomToFit(400, 60);
      }, 600);
    }
  }, [graphData]);

  // Get selected org details
  const selectedOrgData = selectedOrg ? orgStats.get(selectedOrg) : null;

  // Sort repos in selected org by dependent count
  const sortedRepos = useMemo(() => {
    if (!selectedOrgData) return [];
    return [...selectedOrgData.repos]
      .sort((a, b) => b.depended_by_count - a.depended_by_count);
  }, [selectedOrgData]);

  // Paginated repos
  const paginatedRepos = useMemo(() => {
    const startIndex = (repoPage - 1) * repoPageSize;
    return sortedRepos.slice(startIndex, startIndex + repoPageSize);
  }, [sortedRepos, repoPage, repoPageSize]);

  const totalRepoPages = Math.ceil(sortedRepos.length / repoPageSize);

  // Reset repo page when selected org changes
  useEffect(() => {
    setRepoPage(1);
  }, [selectedOrg]);

  return (
    <div className="space-y-4">
      {/* Description */}
      <div className="mb-4 flex items-start justify-between gap-4">
        <div className="flex-1">
          <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            Organization Dependency Map
          </h3>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            {selectedOrg 
              ? `Showing ${selectedOrg} and its cross-org dependencies.`
              : 'Select an organization to focus on its cross-org relationships.'}
          </p>
        </div>
        
        {/* Org Selector */}
        <div className="relative">
          <button
            onClick={() => setShowOrgDropdown(!showOrgDropdown)}
            className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors"
            style={{ 
              backgroundColor: selectedOrg ? 'var(--accent-subtle)' : 'var(--control-bgColor-rest)',
              color: selectedOrg ? 'var(--fgColor-accent)' : 'var(--fgColor-default)',
              border: selectedOrg ? '1px solid var(--borderColor-accent-muted)' : '1px solid var(--borderColor-default)'
            }}
          >
            {selectedOrg ? (
              <>
                <span 
                  className="w-3 h-3 rounded-full" 
                  style={{ backgroundColor: orgColorMap.get(selectedOrg) }}
                />
                {selectedOrg}
              </>
            ) : (
              'Select Organization'
            )}
            <ChevronDownIcon size={16} />
          </button>
          
          {showOrgDropdown && (
            <>
              {/* Backdrop */}
              <div 
                className="fixed inset-0 z-20" 
                onClick={() => {
                  setShowOrgDropdown(false);
                  setOrgSearchFilter('');
                }}
              />
              
              {/* Dropdown */}
              <div 
                className="absolute right-0 mt-1 w-72 rounded-lg shadow-lg z-30 overflow-hidden"
                style={{
                  backgroundColor: 'var(--bgColor-default)',
                  border: '1px solid var(--borderColor-default)',
                }}
              >
                {/* Search input */}
                <div className="p-2 border-b" style={{ borderColor: 'var(--borderColor-muted)' }}>
                  <div className="relative">
                    <span className="absolute left-2 top-1/2 -translate-y-1/2" style={{ color: 'var(--fgColor-muted)' }}>
                      <SearchIcon size={14} />
                    </span>
                    <input
                      type="text"
                      placeholder="Search organizations..."
                      value={orgSearchFilter}
                      onChange={(e) => setOrgSearchFilter(e.target.value)}
                      className="w-full pl-7 pr-3 py-1.5 text-sm rounded"
                      style={{
                        backgroundColor: 'var(--bgColor-muted)',
                        color: 'var(--fgColor-default)',
                        border: '1px solid var(--borderColor-muted)'
                      }}
                      autoFocus
                    />
                  </div>
                </div>
                
                {/* Show All option */}
                <button
                  onClick={() => {
                    setSelectedOrg(null);
                    setShowOrgDropdown(false);
                    setOrgSearchFilter('');
                  }}
                  className="w-full px-3 py-2 text-left text-sm flex items-center gap-2 transition-colors hover:bg-[var(--bgColor-muted)]"
                  style={{ 
                    color: !selectedOrg ? 'var(--fgColor-accent)' : 'var(--fgColor-default)',
                    fontWeight: !selectedOrg ? 600 : 400
                  }}
                >
                  Show All Organizations
                </button>
                
                <div className="border-t" style={{ borderColor: 'var(--borderColor-muted)' }} />
                
                {/* Org list */}
                <div className="max-h-64 overflow-y-auto">
                  {sortedOrgList
                    .filter(org => org.name.toLowerCase().includes(orgSearchFilter.toLowerCase()))
                    .map(org => (
                      <button
                        key={org.name}
                        onClick={() => {
                          setSelectedOrg(org.name);
                          setShowOrgDropdown(false);
                          setOrgSearchFilter('');
                        }}
                        className="w-full px-3 py-2 text-left text-sm flex items-center justify-between gap-2 transition-colors hover:bg-[var(--bgColor-muted)]"
                        style={{ 
                          color: selectedOrg === org.name ? 'var(--fgColor-accent)' : 'var(--fgColor-default)',
                          fontWeight: selectedOrg === org.name ? 600 : 400
                        }}
                      >
                        <div className="flex items-center gap-2">
                          <span 
                            className="w-3 h-3 rounded-full flex-shrink-0" 
                            style={{ backgroundColor: orgColorMap.get(org.name) }}
                          />
                          <span className="truncate">{org.name}</span>
                        </div>
                        <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                          {org.repoCount} repos
                        </span>
                      </button>
                    ))}
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Graph Container */}
      <div 
        ref={containerRef}
        className="rounded-lg relative"
        style={{ 
          backgroundColor: 'rgba(13, 17, 23, 0.5)',
          border: '1px solid var(--borderColor-muted)',
          height: '500px'
        }}
      >
        {/* Legend */}
        <div 
          className="absolute top-3 left-3 z-10 rounded-lg p-2 text-xs"
          style={{ 
            backgroundColor: 'rgba(13, 17, 23, 0.9)',
            border: '1px solid var(--borderColor-muted)'
          }}
        >
          <div className="flex items-center gap-1.5" style={{ color: 'var(--fgColor-muted)' }}>
            <CircleIcon size={12} />
            <span>Node size = repo count</span>
          </div>
          <div className="flex items-center gap-1.5 mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            <ArrowRightIcon size={12} />
            <span>Arrow = dependency direction</span>
          </div>
          <div className="flex items-center gap-1.5 mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            <SearchIcon size={12} />
            <span>Click org to focus</span>
          </div>
          {selectedOrg && (
            <button
              onClick={() => setSelectedOrg(null)}
              className="mt-2 w-full px-2 py-1 rounded text-xs"
              style={{ 
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-default)'
              }}
            >
              Clear Selection
            </button>
          )}
        </div>

        {graphData.nodes.length === 0 ? (
          <div className="flex items-center justify-center h-full" style={{ color: 'var(--fgColor-muted)' }}>
            No organizations match the search filter
          </div>
        ) : (
          <ForceGraph2D
            ref={graphRef}
            graphData={graphData}
            width={dimensions.width}
            height={dimensions.height}
            nodeCanvasObject={paintNode}
            linkCanvasObject={paintLink}
            nodePointerAreaPaint={(node, color, ctx) => {
              const nodeSize = Math.max(15, Math.min(50, 15 + Math.sqrt((node as OrgNode).repoCount) * 5));
              ctx.beginPath();
              ctx.arc(node.x!, node.y!, nodeSize + 10, 0, 2 * Math.PI);
              ctx.fillStyle = color;
              ctx.fill();
            }}
            onNodeClick={(node) => setSelectedOrg(selectedOrg === node.id ? null : node.id as string)}
            onNodeHover={(node) => setHoveredOrg(node ? node.id as string : null)}
            onBackgroundClick={() => setSelectedOrg(null)}
            backgroundColor="transparent"
            cooldownTicks={100}
            d3AlphaDecay={0.02}
            d3VelocityDecay={0.3}
            warmupTicks={50}
            linkDirectionalArrowLength={0}
          />
        )}
      </div>

      {/* Org Stats Summary */}
      <div 
        className="grid grid-cols-2 md:grid-cols-4 gap-3"
        style={{ color: 'var(--fgColor-muted)' }}
      >
        {selectedOrg && selectedOrgData ? (
          // Stats for selected org
          <>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-default)' }}>
                {graphData.nodes.length - 1}
              </div>
              <div className="text-xs">Connected Organizations</div>
            </div>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-default)' }}>
                {selectedOrgData.repoCount.toLocaleString()}
              </div>
              <div className="text-xs">Repositories in {selectedOrg}</div>
            </div>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-accent)' }}>
                {selectedOrgData.totalDependsOn.toLocaleString()}
              </div>
              <div className="text-xs">Outgoing Dependencies</div>
            </div>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-success)' }}>
                {selectedOrgData.totalDependedBy.toLocaleString()}
              </div>
              <div className="text-xs">Incoming Dependents</div>
            </div>
          </>
        ) : (
          // Global stats
          <>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-default)' }}>
                {orgStats.size}
              </div>
              <div className="text-xs">Organizations</div>
            </div>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-default)' }}>
                {nodes.length.toLocaleString()}
              </div>
              <div className="text-xs">Total Repositories</div>
            </div>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-accent)' }}>
                {orgEdges.size}
              </div>
              <div className="text-xs">Cross-Org Dependencies</div>
            </div>
            <div className="text-center p-3 rounded-lg" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <div className="text-2xl font-bold" style={{ color: 'var(--fgColor-success)' }}>
                {[...orgEdges.values()].reduce((a, b) => a + b, 0).toLocaleString()}
              </div>
              <div className="text-xs">Total Cross-Org Links</div>
            </div>
          </>
        )}
      </div>

      {/* Selected Organization Details */}
      {selectedOrg && selectedOrgData && (
        <div 
          className="rounded-lg p-4"
          style={{ 
            backgroundColor: 'var(--bgColor-muted)',
            border: `2px solid ${orgColorMap.get(selectedOrg)}`
          }}
        >
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div 
                className="w-4 h-4 rounded-full"
                style={{ backgroundColor: orgColorMap.get(selectedOrg) }}
              />
              <h4 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                {selectedOrg}
              </h4>
              <Badge color="blue">{selectedOrgData.repoCount} repositories</Badge>
            </div>
            <button
              onClick={() => setSelectedOrg(null)}
              className="px-3 py-1 rounded text-sm"
              style={{ 
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-muted)'
              }}
            >
              Close
            </button>
          </div>

          <div className="grid grid-cols-2 gap-4 mb-4 text-sm">
            <div>
              <span style={{ color: 'var(--fgColor-muted)' }}>Total dependencies (outgoing): </span>
              <span className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                {selectedOrgData.totalDependsOn}
              </span>
            </div>
            <div>
              <span style={{ color: 'var(--fgColor-muted)' }}>Total dependents (incoming): </span>
              <span className="font-semibold" style={{ color: 'var(--fgColor-success)' }}>
                {selectedOrgData.totalDependedBy}
              </span>
            </div>
          </div>

          {/* Repository List */}
          <div className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <h5 className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                Repositories ({sortedRepos.length})
              </h5>
              {totalRepoPages > 1 && (
                <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                  Page {repoPage} of {totalRepoPages}
                </span>
              )}
            </div>
            <div 
              className="rounded-lg"
              style={{ backgroundColor: 'var(--bgColor-default)' }}
            >
              {sortedRepos.length === 0 ? (
                <div className="p-4 text-center text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                  No repositories in this organization
                </div>
              ) : (
                <>
                  <table className="w-full text-sm">
                    <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                      <tr>
                        <th className="text-left px-3 py-2 font-medium" style={{ color: 'var(--fgColor-muted)' }}>
                          Repository
                        </th>
                        <th className="text-center px-3 py-2 font-medium" style={{ color: 'var(--fgColor-muted)' }}>
                          Depends On
                        </th>
                        <th className="text-center px-3 py-2 font-medium" style={{ color: 'var(--fgColor-muted)' }}>
                          Depended By
                        </th>
                        <th className="text-left px-3 py-2 font-medium" style={{ color: 'var(--fgColor-muted)' }}>
                          Status
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {paginatedRepos.map(repo => (
                        <tr 
                          key={repo.id}
                          className="border-t hover:bg-[var(--bgColor-muted)]"
                          style={{ borderColor: 'var(--borderColor-muted)' }}
                        >
                          <td className="px-3 py-2">
                            <Link
                              to={`/repository/${encodeURIComponent(repo.full_name)}`}
                              className="hover:underline"
                              style={{ color: 'var(--fgColor-accent)' }}
                            >
                              {repo.full_name.split('/').pop()}
                            </Link>
                          </td>
                          <td className="text-center px-3 py-2" style={{ color: 'var(--fgColor-default)' }}>
                            {repo.depends_on_count}
                          </td>
                          <td className="text-center px-3 py-2" style={{ color: repo.depended_by_count > 0 ? 'var(--fgColor-success)' : 'var(--fgColor-default)' }}>
                            {repo.depended_by_count}
                          </td>
                          <td className="px-3 py-2">
                            <Badge color={repo.status === 'pending' ? 'gray' : repo.status.includes('complete') ? 'green' : 'blue'}>
                              {repo.status.replace(/_/g, ' ')}
                            </Badge>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  
                  {/* Pagination */}
                  {totalRepoPages > 1 && (
                    <div className="border-t p-3" style={{ borderColor: 'var(--borderColor-muted)' }}>
                      <Pagination
                        currentPage={repoPage}
                        onPageChange={setRepoPage}
                        totalItems={sortedRepos.length}
                        pageSize={repoPageSize}
                      />
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

