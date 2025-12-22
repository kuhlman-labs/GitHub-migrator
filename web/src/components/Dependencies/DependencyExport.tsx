import { useState } from 'react';
import { Button } from '@primer/react';
import { DownloadIcon, ChevronDownIcon } from '@primer/octicons-react';
import { api } from '../../services/api';
import type { DependencyGraphNode, DependencyGraphEdge } from '../../types';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

interface DependencyExportProps {
  filteredNodes: DependencyGraphNode[];
  filteredEdges: DependencyGraphEdge[];
  hasActiveFilters: boolean;
  hasFilteredData: boolean;
}

export function DependencyExport({
  filteredNodes,
  filteredEdges,
  hasActiveFilters,
  hasFilteredData,
}: DependencyExportProps) {
  const { showError } = useToast();
  const [exporting, setExporting] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);

  // Export all dependencies from API (no filters applied)
  const handleExportAll = async (format: 'csv' | 'json') => {
    setShowExportMenu(false);
    try {
      setExporting(true);
      const blob = await api.exportDependencies(format);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `dependencies-all.${format}`;
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
    setShowExportMenu(false);
    
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
      filename = 'dependencies-summary.csv';
    } else {
      content = JSON.stringify(exportRows, null, 2);
      mimeType = 'application/json';
      filename = 'dependencies-summary.json';
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
    <div className="relative">
      <Button
        variant="invisible"
        onClick={() => setShowExportMenu(!showExportMenu)}
        disabled={exporting || !hasFilteredData}
        leadingVisual={DownloadIcon}
        trailingVisual={ChevronDownIcon}
        className="btn-bordered-invisible"
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
            className="absolute right-0 mt-2 w-56 rounded-lg shadow-lg z-20"
            style={{
              backgroundColor: 'var(--bgColor-default)',
              border: '1px solid var(--borderColor-default)',
              boxShadow: 'var(--shadow-floating-large)'
            }}
          >
            <div className="py-1">
              {/* Summary Export Section */}
              <div className="px-4 py-1.5 text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                Summary {hasActiveFilters && `(${filteredNodes.length} repos)`}
              </div>
              <button
                onClick={() => handleExportFiltered('csv')}
                className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                style={{ color: 'var(--fgColor-default)' }}
              >
                Export Summary as CSV
              </button>
              <button
                onClick={() => handleExportFiltered('json')}
                className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                style={{ color: 'var(--fgColor-default)' }}
              >
                Export Summary as JSON
              </button>
              
              {/* Divider */}
              <div className="my-1 border-t" style={{ borderColor: 'var(--borderColor-muted)' }} />
              
              {/* Full Export Section */}
              <div className="px-4 py-1.5 text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                All Dependencies
              </div>
              <button
                onClick={() => handleExportAll('csv')}
                className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                style={{ color: 'var(--fgColor-default)' }}
              >
                Export All as CSV
              </button>
              <button
                onClick={() => handleExportAll('json')}
                className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)]"
                style={{ color: 'var(--fgColor-default)' }}
              >
                Export All as JSON
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

