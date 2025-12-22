import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { useTableState } from './useTableState';

// Wrapper component that provides routing context
function Wrapper({ children }: { children: React.ReactNode }) {
  return <BrowserRouter>{children}</BrowserRouter>;
}

interface TestFilters {
  status?: string;
  type?: string;
  category?: string[];
}

describe('useTableState', () => {
  describe('pagination', () => {
    it('should initialize with page 1', () => {
      const { result } = renderHook(() => useTableState(), { wrapper: Wrapper });
      expect(result.current.page).toBe(1);
    });

    it('should update page', () => {
      const { result } = renderHook(() => useTableState(), { wrapper: Wrapper });

      act(() => {
        result.current.setPage(3);
      });

      expect(result.current.page).toBe(3);
    });

    it('should calculate offset correctly', () => {
      const { result } = renderHook(() => useTableState({ pageSize: 25 }), {
        wrapper: Wrapper,
      });

      expect(result.current.offset).toBe(0);

      act(() => {
        result.current.setPage(2);
      });

      expect(result.current.offset).toBe(25);

      act(() => {
        result.current.setPage(3);
      });

      expect(result.current.offset).toBe(50);
    });

    it('should respect custom pageSize', () => {
      const { result } = renderHook(() => useTableState({ pageSize: 50 }), {
        wrapper: Wrapper,
      });

      expect(result.current.limit).toBe(50);

      act(() => {
        result.current.setPage(2);
      });

      expect(result.current.offset).toBe(50);
    });
  });

  describe('search', () => {
    it('should initialize with empty search', () => {
      const { result } = renderHook(() => useTableState(), { wrapper: Wrapper });
      expect(result.current.search).toBe('');
    });

    it('should update search', () => {
      const { result } = renderHook(() => useTableState(), { wrapper: Wrapper });

      act(() => {
        result.current.setSearch('test query');
      });

      expect(result.current.search).toBe('test query');
    });

    it('should reset page to 1 when search changes', () => {
      const { result } = renderHook(() => useTableState(), { wrapper: Wrapper });

      act(() => {
        result.current.setPage(5);
      });

      expect(result.current.page).toBe(5);

      act(() => {
        result.current.setSearch('new search');
      });

      expect(result.current.page).toBe(1);
    });
  });

  describe('filters', () => {
    it('should initialize with provided initial filters', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ initialFilters: { status: 'active' } }),
        { wrapper: Wrapper }
      );

      expect(result.current.filters).toEqual({ status: 'active' });
    });

    it('should update filters', () => {
      const { result } = renderHook(() => useTableState<TestFilters>(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.setFilters({ status: 'pending', type: 'manual' });
      });

      expect(result.current.filters).toEqual({ status: 'pending', type: 'manual' });
    });

    it('should update single filter', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ initialFilters: { status: 'active' } }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.updateFilter('type', 'manual');
      });

      expect(result.current.filters).toEqual({ status: 'active', type: 'manual' });
    });

    it('should remove filter', () => {
      const { result } = renderHook(
        () =>
          useTableState<TestFilters>({
            initialFilters: { status: 'active', type: 'manual' },
          }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.removeFilter('type');
      });

      expect(result.current.filters).toEqual({ status: 'active' });
    });

    it('should reset page to 1 when filters change', () => {
      const { result } = renderHook(() => useTableState<TestFilters>(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.setPage(5);
      });

      expect(result.current.page).toBe(5);

      act(() => {
        result.current.updateFilter('status', 'pending');
      });

      expect(result.current.page).toBe(1);
    });

    it('should support functional filter updates', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ initialFilters: { status: 'active' } }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.setFilters((prev) => ({ ...prev, type: 'manual' }));
      });

      expect(result.current.filters).toEqual({ status: 'active', type: 'manual' });
    });
  });

  describe('resetFilters', () => {
    it('should reset to initial filters', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ initialFilters: { status: 'all' } }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.setFilters({ status: 'pending', type: 'manual' });
        result.current.setSearch('test');
        result.current.setPage(5);
      });

      act(() => {
        result.current.resetFilters();
      });

      expect(result.current.filters).toEqual({ status: 'all' });
      expect(result.current.search).toBe('');
      expect(result.current.page).toBe(1);
    });
  });

  describe('activeFilterCount', () => {
    it('should count active filters', () => {
      const { result } = renderHook(() => useTableState<TestFilters>(), {
        wrapper: Wrapper,
      });

      expect(result.current.activeFilterCount).toBe(0);

      act(() => {
        result.current.setFilters({ status: 'active' });
      });

      expect(result.current.activeFilterCount).toBe(1);

      act(() => {
        result.current.updateFilter('type', 'manual');
      });

      expect(result.current.activeFilterCount).toBe(2);
    });

    it('should not count empty values as active', () => {
      const { result } = renderHook(() => useTableState<TestFilters>(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.setFilters({
          status: '',
          type: undefined,
        } as TestFilters);
      });

      expect(result.current.activeFilterCount).toBe(0);
    });

    it('should not count "all" as active', () => {
      const { result } = renderHook(() => useTableState<TestFilters>(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.setFilters({ status: 'all' });
      });

      expect(result.current.activeFilterCount).toBe(0);
    });

    it('should not count empty arrays as active', () => {
      const { result } = renderHook(() => useTableState<TestFilters>(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.setFilters({ category: [] });
      });

      expect(result.current.activeFilterCount).toBe(0);
    });

    it('should count non-empty arrays as active', () => {
      const { result } = renderHook(() => useTableState<TestFilters>(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.setFilters({ category: ['a', 'b'] });
      });

      expect(result.current.activeFilterCount).toBe(1);
    });

    it('should exclude pagination keys from count', () => {
      const { result } = renderHook(() => useTableState<TestFilters & { limit?: number; offset?: number }>(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.setFilters({ 
          status: 'active',
          limit: 25,
          offset: 0,
        });
      });

      // Only status should count, not limit/offset
      expect(result.current.activeFilterCount).toBe(1);
    });
  });

  describe('URL sync', () => {
    it('should not sync to URL when syncWithUrl is false', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ syncWithUrl: false }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.setFilters({ status: 'active' });
      });

      // URL should not contain the filter
      expect(window.location.search).toBe('');
    });

    it('should sync to URL when syncWithUrl is true', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ 
          syncWithUrl: true, 
          initialFilters: { status: '' } 
        }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.setFilters({ status: 'active' });
      });

      // URL should contain the filter
      expect(window.location.search).toContain('status=active');
    });

    it('should sync search to URL when syncWithUrl is true', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ syncWithUrl: true }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.setSearch('test query');
      });

      expect(window.location.search).toContain('search=test');
    });

    it('should sync page to URL when page > 1 and syncWithUrl is true', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ syncWithUrl: true }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.setPage(3);
      });

      expect(window.location.search).toContain('page=3');
    });

    it('should only sync specified urlKeys when provided', () => {
      const { result } = renderHook(
        () => useTableState<TestFilters>({ 
          syncWithUrl: true,
          urlKeys: ['status'],
          initialFilters: { status: '', type: '' }
        }),
        { wrapper: Wrapper }
      );

      act(() => {
        result.current.setFilters({ status: 'active', type: 'manual' });
      });

      // Only status should be in URL
      expect(window.location.search).toContain('status=active');
      expect(window.location.search).not.toContain('type=manual');
    });
  });
});

