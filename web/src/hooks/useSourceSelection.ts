import { useSourceContext } from '../contexts/SourceContext';

/**
 * Hook to manage source selection state for discovery dialogs.
 * Use this alongside DiscoverySourceSelector component.
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

