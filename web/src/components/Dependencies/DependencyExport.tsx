import { useState } from 'react';
import { ActionMenu, ActionList } from '@primer/react';
import { DownloadIcon, TriangleDownIcon } from '@primer/octicons-react';
import { BorderedButton } from '../common/buttons';
import { api } from '../../services/api';
import type { DependencyGraphNode, DependencyGraphEdge } from '../../types';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

interface DependencyExportProps {
  filteredNodes: DependencyGraphNode[];
  filteredEdges: DependencyGraphEdge[];
  hasActiveFilters: boolean;
  hasFilteredData: boolean;
  sourceId?: number;
  sourceName?: string;
}

export function DependencyExport({
  filteredNodes,
  filteredEdges,
  hasActiveFilters,
  hasFilteredData,
  sourceId,
  sourceName,
}: DependencyExportProps) {
  const { showError } = useToast();
  const [exporting, setExporting] = useState(false);
  const sourceSuffix = sourceName ? `_${sourceName.replace(/\s+/g, '_')}` : '';

  // Export all dependencies from API (filtered by source if selected)
  const handleExportAll = async (format: 'csv' | 'json') => {
    try {
      setExporting(true);
      const blob = await api.exportDependencies(format, { source_id: sourceId });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `dependencies-all${sourceSuffix}.${format}`;
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

  // Export current filtered view (client-side) - one row per repository
  const handleExportFiltered = (format: 'csv' | 'json') => {
    // Build maps for dependencies (what each repo depends on) and dependents (what depends on each repo)
    const dependsOnMap = new Map<string, string[]>();
    const dependedByMap = new Map<string, string[]>();
    
    // Process all edges to build both maps
    filteredEdges.forEach(edge => {
      // edge.source depends on edge.target
      const deps = dependsOnMap.get(edge.source) || [];
      deps.push(edge.target);
      dependsOnMap.set(edge.source, deps);
      
      // edge.target is depended on by edge.source
      const dependents = dependedByMap.get(edge.target) || [];
      dependents.push(edge.source);
      dependedByMap.set(edge.target, dependents);
    });
    
    // Create one row per repository with aggregated dependencies
    // Use computed counts from filteredEdges to match the listed dependencies
    const exportRows = filteredNodes.map(node => {
      const dependencies = dependsOnMap.get(node.id) || [];
      const dependedBy = dependedByMap.get(node.id) || [];
      
      return {
        repository: node.full_name,
        organization: node.organization,
        status: node.status,
        depends_on_count: dependencies.length,
        depended_by_count: dependedBy.length,
        dependencies: dependencies.join('; '),
        depended_by: dependedBy.join('; ')
      };
    });

    let content: string;
    let mimeType: string;
    let filename: string;

    if (format === 'csv') {
      // Helper to escape CSV fields - double quotes must be escaped as ""
      const escapeCSV = (value: string) => `"${value.replace(/"/g, '""')}"`;
      
      const headers = ['repository', 'organization', 'status', 'depends_on_count', 'depended_by_count', 'dependencies', 'depended_by'];
      const csvRows = [headers.join(',')];
      exportRows.forEach(row => {
        csvRows.push([
          escapeCSV(row.repository),
          escapeCSV(row.organization),
          escapeCSV(row.status),
          row.depends_on_count,
          row.depended_by_count,
          escapeCSV(row.dependencies),
          escapeCSV(row.depended_by)
        ].join(','));
      });
      content = csvRows.join('\n');
      mimeType = 'text/csv';
      filename = `dependencies-summary${sourceSuffix}.csv`;
    } else {
      content = JSON.stringify(exportRows, null, 2);
      mimeType = 'application/json';
      filename = `dependencies-summary${sourceSuffix}.json`;
    }

    const blob = new Blob([content], { type: mimeType });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  };

  return (
    <ActionMenu>
      <ActionMenu.Anchor>
        <BorderedButton
          disabled={exporting || !hasFilteredData}
          leadingVisual={DownloadIcon}
          trailingAction={TriangleDownIcon}
        >
          {exporting ? 'Exporting...' : 'Export'}
        </BorderedButton>
      </ActionMenu.Anchor>
      <ActionMenu.Overlay>
        <ActionList>
          <ActionList.Group title={`Summary ${hasActiveFilters ? `(${filteredNodes.length} repos)` : ''}`}>
            <ActionList.Item onSelect={() => handleExportFiltered('csv')}>
              Export Summary as CSV
            </ActionList.Item>
            <ActionList.Item onSelect={() => handleExportFiltered('json')}>
              Export Summary as JSON
            </ActionList.Item>
          </ActionList.Group>
          <ActionList.Divider />
          <ActionList.Group title="All Dependencies">
            <ActionList.Item onSelect={() => handleExportAll('csv')}>
              Export All as CSV
            </ActionList.Item>
            <ActionList.Item onSelect={() => handleExportAll('json')}>
              Export All as JSON
            </ActionList.Item>
          </ActionList.Group>
        </ActionList>
      </ActionMenu.Overlay>
    </ActionMenu>
  );
}

