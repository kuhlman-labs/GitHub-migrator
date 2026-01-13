import { useSourceContext } from '../contexts/SourceContext';

/**
 * Hook to manage source selection state for discovery dialogs.
 * Use this alongside DiscoverySourceSelector component.
 */
export function useSourceSelection() {
  const { activeSource, sources, isAllSourcesMode, hasMultipleSources } = useSourceContext();
  const activeSources = sources.filter(s => s.is_active);
  
  return {
    isAllSourcesMode,
    hasMultipleSources,
    activeSources,
    activeSource,
    sources,
  };
}

