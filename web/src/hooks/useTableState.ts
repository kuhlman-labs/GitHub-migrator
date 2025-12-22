import { useState, useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

interface UseTableStateOptions<TFilters> {
  /** Initial filters to apply */
  initialFilters?: TFilters;
  /** Page size for pagination */
  pageSize?: number;
  /** Whether to sync filters with URL search params */
  syncWithUrl?: boolean;
  /** Keys to sync with URL (if not specified, all filter keys will be synced) */
  urlKeys?: (keyof TFilters)[];
}

interface UseTableStateReturn<TFilters> {
  /** Current page number (1-indexed) */
  page: number;
  /** Set current page */
  setPage: (page: number) => void;
  /** Search query string */
  search: string;
  /** Set search query */
  setSearch: (search: string) => void;
  /** Current filters */
  filters: TFilters;
  /** Set filters */
  setFilters: (filters: TFilters | ((prev: TFilters) => TFilters)) => void;
  /** Update a single filter */
  updateFilter: <K extends keyof TFilters>(key: K, value: TFilters[K]) => void;
  /** Remove a filter */
  removeFilter: (key: keyof TFilters) => void;
  /** Reset all filters to initial state */
  resetFilters: () => void;
  /** Get offset for pagination */
  offset: number;
  /** Page size */
  limit: number;
  /** Active filter count (excluding pagination/search) */
  activeFilterCount: number;
}

/**
 * A reusable hook for managing table state including pagination, search, and filtering.
 * 
 * @example
 * ```tsx
 * const { page, search, filters, setPage, setSearch, updateFilter } = useTableState<MyFilters>({
 *   initialFilters: { status: 'all' },
 *   pageSize: 25,
 *   syncWithUrl: true,
 * });
 * ```
 */
export function useTableState<TFilters extends Record<string, unknown> = Record<string, unknown>>(
  options: UseTableStateOptions<TFilters> = {}
): UseTableStateReturn<TFilters> {
  const {
    initialFilters = {} as TFilters,
    pageSize = 25,
    syncWithUrl = false,
    urlKeys,
  } = options;

  const [searchParams, setSearchParams] = useSearchParams();
  const [page, setPageInternal] = useState(1);
  const [search, setSearchInternal] = useState('');
  const [filters, setFiltersInternal] = useState<TFilters>(initialFilters);

  // Sync with URL on mount if enabled
  useMemo(() => {
    if (syncWithUrl) {
      const urlFilters: Partial<TFilters> = {};
      const keysToSync = urlKeys || (Object.keys(initialFilters) as (keyof TFilters)[]);
      
      keysToSync.forEach((key) => {
        const value = searchParams.get(String(key));
        if (value !== null) {
          // Handle arrays (comma-separated)
          if (value.includes(',')) {
            (urlFilters as Record<string, unknown>)[String(key)] = value.split(',');
          } else {
            (urlFilters as Record<string, unknown>)[String(key)] = value;
          }
        }
      });

      const urlSearch = searchParams.get('search');
      if (urlSearch) {
        setSearchInternal(urlSearch);
      }

      const urlPage = searchParams.get('page');
      if (urlPage) {
        setPageInternal(parseInt(urlPage, 10) || 1);
      }

      if (Object.keys(urlFilters).length > 0) {
        setFiltersInternal((prev) => ({ ...prev, ...urlFilters }));
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Only run on mount

  // Update URL when state changes
  const updateUrl = useCallback((newFilters: TFilters, newSearch: string, newPage: number) => {
    if (!syncWithUrl) return;

    const params = new URLSearchParams();
    const keysToSync = urlKeys || (Object.keys(newFilters) as (keyof TFilters)[]);

    keysToSync.forEach((key) => {
      const value = newFilters[key];
      if (value !== undefined && value !== null && value !== '' && value !== 'all') {
        if (Array.isArray(value)) {
          params.set(String(key), value.join(','));
        } else {
          params.set(String(key), String(value));
        }
      }
    });

    if (newSearch) {
      params.set('search', newSearch);
    }

    if (newPage > 1) {
      params.set('page', String(newPage));
    }

    setSearchParams(params, { replace: true });
  }, [syncWithUrl, urlKeys, setSearchParams]);

  const setPage = useCallback((newPage: number) => {
    setPageInternal(newPage);
    updateUrl(filters, search, newPage);
  }, [filters, search, updateUrl]);

  const setSearch = useCallback((newSearch: string) => {
    setSearchInternal(newSearch);
    setPageInternal(1); // Reset to first page on search
    updateUrl(filters, newSearch, 1);
  }, [filters, updateUrl]);

  const setFilters = useCallback((newFilters: TFilters | ((prev: TFilters) => TFilters)) => {
    setFiltersInternal((prev) => {
      const updated = typeof newFilters === 'function' ? newFilters(prev) : newFilters;
      updateUrl(updated, search, 1);
      setPageInternal(1); // Reset to first page on filter change
      return updated;
    });
  }, [search, updateUrl]);

  const updateFilter = useCallback(<K extends keyof TFilters>(key: K, value: TFilters[K]) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  }, [setFilters]);

  const removeFilter = useCallback((key: keyof TFilters) => {
    setFilters((prev) => {
      const updated = { ...prev };
      delete updated[key];
      return updated;
    });
  }, [setFilters]);

  const resetFilters = useCallback(() => {
    setFiltersInternal(initialFilters);
    setSearchInternal('');
    setPageInternal(1);
    updateUrl(initialFilters, '', 1);
  }, [initialFilters, updateUrl]);

  // Calculate offset for API pagination
  const offset = useMemo(() => (page - 1) * pageSize, [page, pageSize]);

  // Count active filters (excluding common non-filter fields)
  const activeFilterCount = useMemo(() => {
    const excludeKeys = ['limit', 'offset', 'sort_by', 'sort_order'];
    return Object.entries(filters).filter(([key, value]) => {
      if (excludeKeys.includes(key)) return false;
      if (value === undefined || value === null || value === '' || value === 'all') return false;
      if (Array.isArray(value) && value.length === 0) return false;
      return true;
    }).length;
  }, [filters]);

  return {
    page,
    setPage,
    search,
    setSearch,
    filters,
    setFilters,
    updateFilter,
    removeFilter,
    resetFilters,
    offset,
    limit: pageSize,
    activeFilterCount,
  };
}

export type { UseTableStateOptions, UseTableStateReturn };

