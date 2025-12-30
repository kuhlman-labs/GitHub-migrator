/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useEffect, useCallback, useRef, ReactNode } from 'react';
import { sourcesApi } from '../services/api/sources';
import type { Source, SourceFilter } from '../types';

interface SourceContextType {
  /** All configured sources */
  sources: Source[];
  /** Current filter state: 'all' or a specific source ID */
  activeSourceFilter: SourceFilter;
  /** Set the active source filter */
  setActiveSourceFilter: (filter: SourceFilter) => void;
  /** Get the currently selected source (null if 'all') */
  activeSource: Source | null;
  /** Whether sources are loading */
  isLoading: boolean;
  /** Error if sources failed to load */
  error: Error | null;
  /** Refresh the sources list */
  refetchSources: () => Promise<void>;
}

const SourceContext = createContext<SourceContextType | undefined>(undefined);

interface SourceProviderProps {
  children: ReactNode;
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
  const [activeSourceFilter, setActiveSourceFilter] = useState<SourceFilter>('all');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  
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

  // Get the currently selected source (null if 'all' is selected)
  const activeSource = activeSourceFilter === 'all' 
    ? null 
    : sources.find(s => s.id === activeSourceFilter) || null;

  // Handle source filter changes
  const handleSetActiveSourceFilter = useCallback((filter: SourceFilter) => {
    // Validate that the source exists if it's an ID
    if (filter !== 'all') {
      const exists = sources.some(s => s.id === filter);
      if (!exists) {
        console.warn(`Source with ID ${filter} not found, defaulting to 'all'`);
        setActiveSourceFilter('all');
        return;
      }
    }
    setActiveSourceFilter(filter);
  }, [sources]);

  // Persist filter to localStorage
  useEffect(() => {
    const savedFilter = localStorage.getItem('github-migrator-source-filter');
    if (savedFilter) {
      const parsed = savedFilter === 'all' ? 'all' : parseInt(savedFilter, 10);
      if (parsed === 'all' || !isNaN(parsed as number)) {
        setActiveSourceFilter(parsed);
      }
    }
  }, []);

  useEffect(() => {
    localStorage.setItem('github-migrator-source-filter', String(activeSourceFilter));
  }, [activeSourceFilter]);

  const value: SourceContextType = {
    sources,
    activeSourceFilter,
    setActiveSourceFilter: handleSetActiveSourceFilter,
    activeSource,
    isLoading,
    error,
    refetchSources: fetchSources,
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

  const deleteSource = useCallback(async (id: number) => {
    await sourcesApi.delete(id);
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

