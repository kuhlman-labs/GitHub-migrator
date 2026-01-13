/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useEffect, useCallback, useRef, useMemo, ReactNode } from 'react';
import { sourcesApi } from '../services/api/sources';
import type { Source, SourceFilter } from '../types';

const STORAGE_KEY = 'github-migrator-source-filter';

interface SourceContextType {
  /** All configured sources */
  sources: Source[];
  /** Current filter state: 'all' or a specific source ID */
  activeSourceFilter: SourceFilter;
  /** Set the active source filter */
  setActiveSourceFilter: (filter: SourceFilter) => void;
  /** Get the currently selected source (null if 'all' in multi-source mode) */
  activeSource: Source | null;
  /** Whether sources are loading */
  isLoading: boolean;
  /** Error if sources failed to load */
  error: Error | null;
  /** Refresh the sources list */
  refetchSources: () => Promise<void>;
  /** True when multiple sources are configured (source selector should be shown) */
  hasMultipleSources: boolean;
  /** True when in "All Sources" mode in a multi-source setup */
  isAllSourcesMode: boolean;
}

const SourceContext = createContext<SourceContextType | undefined>(undefined);

interface SourceProviderProps {
  children: ReactNode;
}

/**
 * Get saved source filter from localStorage
 */
function getSavedFilter(): SourceFilter {
  try {
    const savedFilter = localStorage.getItem(STORAGE_KEY);
    if (savedFilter) {
      if (savedFilter === 'all') return 'all';
      const parsed = parseInt(savedFilter, 10);
      if (!isNaN(parsed)) return parsed;
    }
  } catch {
    // localStorage might not be available
  }
  return 'all';
}

/**
 * Save source filter to localStorage
 */
function saveFilter(filter: SourceFilter): void {
  try {
    localStorage.setItem(STORAGE_KEY, String(filter));
  } catch {
    // localStorage might not be available
  }
}

/**
 * SourceProvider manages the global state for migration sources.
 * It provides:
 * - List of all configured sources
 * - Active source filter (for filtering views by source)
 * - Methods to fetch and refresh sources
 */
export function SourceProvider({ children }: SourceProviderProps) {
  const [sources, setSources] = useState<Source[]>([]);
  // Initialize from localStorage synchronously to avoid flash
  const [activeSourceFilter, setActiveSourceFilter] = useState<SourceFilter>(getSavedFilter);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  
  // Track previous sources length to detect transitions (e.g., multi -> single source)
  const prevSourcesLengthRef = useRef<number>(0);
  
  // Track if component is mounted to prevent state updates after unmount
  const isMountedRef = useRef(true);

  // Fetch sources on mount
  const fetchSources = useCallback(async () => {
    try {
      if (isMountedRef.current) {
        setIsLoading(true);
        setError(null);
      }
      const data = await sourcesApi.list();
      if (isMountedRef.current) {
        setSources(data);
      }
    } catch (err) {
      if (isMountedRef.current) {
        setError(err instanceof Error ? err : new Error('Failed to fetch sources'));
        console.error('Failed to fetch sources:', err);
      }
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    isMountedRef.current = true;
    fetchSources();
    
    return () => {
      isMountedRef.current = false;
    };
  }, [fetchSources]);

  // Validate and adjust filter whenever sources change
  // This handles:
  // 1. Initial load: validate saved filter against available sources
  // 2. Source deletion: reset filter if selected source no longer exists
  // 3. Single source setup: auto-select the only available source
  // 4. Transition from multi to single source: auto-select the remaining source
  useEffect(() => {
    if (isLoading || sources.length === 0) {
      return;
    }
    
    const prevLength = prevSourcesLengthRef.current;
    prevSourcesLengthRef.current = sources.length;
    
    // Case 1: Single source setup - always select it
    // This handles both initial load with single source AND transition from multi to single
    if (sources.length === 1) {
      const singleSource = sources[0];
      if (activeSourceFilter !== singleSource.id) {
        setActiveSourceFilter(singleSource.id);
        saveFilter(singleSource.id);
      }
      return;
    }
    
    // Case 2: Multi-source setup - validate that selected source still exists
    // This runs on every sources change, not just initial load
    if (activeSourceFilter !== 'all') {
      const exists = sources.some(s => s.id === activeSourceFilter);
      if (!exists) {
        // Selected source was deleted - reset to 'all'
        console.warn(`Source ID ${activeSourceFilter} no longer exists, resetting to 'all'`);
        setActiveSourceFilter('all');
        saveFilter('all');
        return;
      }
    }
    
    // Case 3: Transition from single to multi-source (source was added)
    // If user was on single source and now has multiple, keep their selection
    // (no action needed - the selected source ID is still valid)
    if (prevLength === 1 && sources.length > 1) {
      // Just log for debugging - no state change needed
      // The previously auto-selected source remains selected
    }
  }, [isLoading, sources, activeSourceFilter]);

  // Get the currently selected source
  // When there's exactly one source, always return it (regardless of filter)
  // This ensures single-source setups always behave as if that source is selected
  const activeSource = useMemo(() => {
    // Single source: always return it (no "All Sources" mode)
    if (sources.length === 1) {
      return sources[0];
    }
    // Multiple sources: respect the filter
    if (activeSourceFilter === 'all') {
      return null;
    }
    return sources.find(s => s.id === activeSourceFilter) || null;
  }, [sources, activeSourceFilter]);

  // Handle source filter changes - validates and persists
  const handleSetActiveSourceFilter = useCallback((filter: SourceFilter) => {
    // Validate that the source exists if it's an ID
    if (filter !== 'all') {
      const exists = sources.some(s => s.id === filter);
      if (!exists) {
        console.warn(`Source with ID ${filter} not found, defaulting to 'all'`);
        setActiveSourceFilter('all');
        saveFilter('all');
        return;
      }
    }
    setActiveSourceFilter(filter);
    saveFilter(filter);
  }, [sources]);

  // Derived values for source mode
  const hasMultipleSources = sources.length > 1;
  // "All Sources" mode is only possible when there are multiple sources AND filter is 'all'
  const isAllSourcesMode = hasMultipleSources && activeSourceFilter === 'all';

  const value: SourceContextType = {
    sources,
    activeSourceFilter,
    setActiveSourceFilter: handleSetActiveSourceFilter,
    activeSource,
    isLoading,
    error,
    refetchSources: fetchSources,
    hasMultipleSources,
    isAllSourcesMode,
  };

  return (
    <SourceContext.Provider value={value}>
      {children}
    </SourceContext.Provider>
  );
}

/**
 * Hook to access source context.
 * Must be used within a SourceProvider.
 */
export function useSourceContext(): SourceContextType {
  const context = useContext(SourceContext);
  if (context === undefined) {
    throw new Error('useSourceContext must be used within a SourceProvider');
  }
  return context;
}

/**
 * Hook for individual source operations.
 * Provides methods for CRUD operations on sources.
 */
export function useSources() {
  const { sources, refetchSources, isLoading, error } = useSourceContext();

  const createSource = useCallback(async (data: Parameters<typeof sourcesApi.create>[0]) => {
    const source = await sourcesApi.create(data);
    await refetchSources();
    return source;
  }, [refetchSources]);

  const updateSource = useCallback(async (id: number, data: Parameters<typeof sourcesApi.update>[1]) => {
    const source = await sourcesApi.update(id, data);
    await refetchSources();
    return source;
  }, [refetchSources]);

  const deleteSource = useCallback(async (id: number, options?: { force?: boolean; confirm?: string }) => {
    await sourcesApi.delete(id, options);
    await refetchSources();
  }, [refetchSources]);

  const validateSource = useCallback(async (data: Parameters<typeof sourcesApi.validate>[0]) => {
    return sourcesApi.validate(data);
  }, []);

  const setSourceActive = useCallback(async (id: number, isActive: boolean) => {
    const result = await sourcesApi.setActive(id, isActive);
    await refetchSources();
    return result;
  }, [refetchSources]);

  return {
    sources,
    isLoading,
    error,
    refetchSources,
    createSource,
    updateSource,
    deleteSource,
    validateSource,
    setSourceActive,
  };
}

