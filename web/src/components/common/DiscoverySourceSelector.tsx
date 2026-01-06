import { useEffect } from 'react';
import { FormControl, Select } from '@primer/react';
import { useSourceContext } from '../../contexts/SourceContext';
import type { Source } from '../../types';

export interface DiscoverySourceSelectorProps {
  /** Currently selected source ID */
  selectedSourceId: number | null;
  /** Callback when source selection changes */
  onSourceChange: (sourceId: number | null, source: Source | null) => void;
  /** Whether a source selection is required (true in "All Sources" mode) */
  required?: boolean;
  /** Whether the selector is disabled */
  disabled?: boolean;
  /** Custom label for the form control */
  label?: string;
  /** Custom caption when no source is selected */
  defaultCaption?: string;
  /** Filter sources by type (optional) */
  filterByType?: 'github' | 'azuredevops';
  /** Show repository count in caption (for repository discovery) */
  showRepoCount?: boolean;
}

/**
 * Reusable source selector component for discovery dialogs.
 * Handles source selection, auto-selection when only one source available,
 * and displays dynamic caption with source type info.
 */
export function DiscoverySourceSelector({
  selectedSourceId,
  onSourceChange,
  required = false,
  disabled = false,
  label = 'Select Source',
  defaultCaption = 'Select which source to use.',
  filterByType,
  showRepoCount = false,
}: DiscoverySourceSelectorProps) {
  const { sources, activeSource } = useSourceContext();
  
  // Filter to active sources, optionally by type
  const availableSources = sources.filter(s => 
    s.is_active && (!filterByType || s.type === filterByType)
  );
  
  // Determine if we're in "All Sources" mode (no active source selected)
  const isAllSourcesMode = !activeSource;
  
  // Get the selected source object
  const selectedSource = selectedSourceId 
    ? sources.find(s => s.id === selectedSourceId) 
    : null;
  
  // Auto-select source when:
  // - Required mode (All Sources) and only one source available
  // - Or when there's an active source selected in the header
  useEffect(() => {
    if (selectedSourceId) return; // Already selected
    
    if (activeSource) {
      // Use the active source from header
      onSourceChange(activeSource.id, activeSource);
    } else if (required && availableSources.length === 1) {
      // Auto-select if only one source available
      const onlySource = availableSources[0];
      onSourceChange(onlySource.id, onlySource);
    }
  }, [activeSource, availableSources, required, selectedSourceId, onSourceChange]);
  
  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    const newSourceId = value ? parseInt(value, 10) : null;
    const newSource = newSourceId ? sources.find(s => s.id === newSourceId) || null : null;
    onSourceChange(newSourceId, newSource);
  };
  
  // Generate dynamic caption
  const getCaption = () => {
    if (selectedSource) {
      const typeLabel = selectedSource.type === 'github' ? 'GitHub' : 'Azure DevOps';
      if (showRepoCount) {
        return `${typeLabel} source with ${selectedSource.repository_count} repositories`;
      }
      return `${typeLabel} source`;
    }
    return defaultCaption;
  };
  
  // Don't render if not in All Sources mode and only one source available
  // (will be auto-selected via useEffect)
  if (!isAllSourcesMode && availableSources.length <= 1) {
    return null;
  }
  
  return (
    <FormControl className="mb-3" required={required}>
      <FormControl.Label>{label}</FormControl.Label>
      <Select 
        value={selectedSourceId?.toString() || ''} 
        onChange={handleChange}
        disabled={disabled}
      >
        <Select.Option value="">
          {required ? 'Choose a source...' : 'Default (use current config)'}
        </Select.Option>
        {availableSources.map(source => (
          <Select.Option key={source.id} value={source.id.toString()}>
            {source.name} ({source.type === 'github' ? 'GitHub' : 'Azure DevOps'})
          </Select.Option>
        ))}
      </Select>
      <FormControl.Caption>
        {getCaption()}
      </FormControl.Caption>
    </FormControl>
  );
}

/**
 * Hook to manage source selection state for discovery dialogs.
 * Use this alongside SourceSelector component.
 */
export function useSourceSelection() {
  const { activeSource, sources } = useSourceContext();
  const isAllSourcesMode = !activeSource;
  const activeSources = sources.filter(s => s.is_active);
  
  return {
    isAllSourcesMode,
    activeSources,
    activeSource,
    sources,
  };
}
