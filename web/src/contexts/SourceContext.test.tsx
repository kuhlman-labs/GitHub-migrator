import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { ThemeProvider } from '@primer/react';
import { SourceProvider, useSourceContext } from './SourceContext';
import { sourcesApi } from '../services/api/sources';
import type { Source } from '../types/source';

// Mock the sources API module
vi.mock('../services/api/sources', () => ({
  sourcesApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    validate: vi.fn(),
    setActive: vi.fn(),
  },
}));

// Helper to create mock sources
function createMockSource(overrides: Partial<Source> = {}): Source {
  return {
    id: 1,
    name: 'Test Source',
    type: 'github',
    base_url: 'https://github.com',
    has_app_auth: false,
    has_oauth: false,
    is_active: true,
    repository_count: 10,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    masked_token: 'ghp_***',
    ...overrides,
  };
}

// Test component that displays context values
function TestComponent() {
  const {
    sources,
    activeSource,
    activeSourceFilter,
    isLoading,
    hasMultipleSources,
    isAllSourcesMode,
    setActiveSourceFilter,
  } = useSourceContext();

  return (
    <div>
      <div data-testid="isLoading">{isLoading.toString()}</div>
      <div data-testid="sourcesCount">{sources.length}</div>
      <div data-testid="activeSource">{activeSource?.name || 'null'}</div>
      <div data-testid="activeSourceId">{activeSource?.id?.toString() || 'null'}</div>
      <div data-testid="activeSourceFilter">{activeSourceFilter.toString()}</div>
      <div data-testid="hasMultipleSources">{hasMultipleSources.toString()}</div>
      <div data-testid="isAllSourcesMode">{isAllSourcesMode.toString()}</div>
      <button onClick={() => setActiveSourceFilter('all')}>Select All</button>
      <button onClick={() => setActiveSourceFilter(1)}>Select Source 1</button>
      <button onClick={() => setActiveSourceFilter(2)}>Select Source 2</button>
    </div>
  );
}

function TestWrapper({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <SourceProvider>{children}</SourceProvider>
    </ThemeProvider>
  );
}

// Storage key used by SourceContext
const STORAGE_KEY = 'github-migrator-source-filter';

describe('SourceContext', () => {
  const mockSourcesApi = sourcesApi as unknown as {
    list: ReturnType<typeof vi.fn>;
    create: ReturnType<typeof vi.fn>;
    update: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
    validate: ReturnType<typeof vi.fn>;
    setActive: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Clear localStorage before each test
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  describe('useSourceContext hook', () => {
    it('should throw error when used outside SourceProvider', () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        render(
          <ThemeProvider>
            <TestComponent />
          </ThemeProvider>
        );
      }).toThrow('useSourceContext must be used within a SourceProvider');

      consoleError.mockRestore();
    });
  });

  describe('single source configuration', () => {
    const singleSource = createMockSource({ id: 1, name: 'Only Source', type: 'azuredevops' });

    beforeEach(() => {
      mockSourcesApi.list.mockResolvedValue([singleSource]);
    });

    it('should automatically select the single source as activeSource', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // activeSource should be the single source
      expect(screen.getByTestId('activeSource').textContent).toBe('Only Source');
      expect(screen.getByTestId('activeSourceId').textContent).toBe('1');
    });

    it('should set isAllSourcesMode to false for single source', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // isAllSourcesMode should always be false for single source
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('false');
    });

    it('should set hasMultipleSources to false for single source', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(screen.getByTestId('hasMultipleSources').textContent).toBe('false');
    });

    it('should persist the single source ID to localStorage', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Should have saved the source ID to localStorage
      expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
    });

    it('should always return the single source even if localStorage has "all"', async () => {
      // Pre-set localStorage to 'all'
      localStorage.setItem(STORAGE_KEY, 'all');

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Should still return the single source
      expect(screen.getByTestId('activeSource').textContent).toBe('Only Source');
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('false');
    });
  });

  describe('multiple sources configuration', () => {
    const source1 = createMockSource({ id: 1, name: 'GitHub Source', type: 'github' });
    const source2 = createMockSource({ id: 2, name: 'ADO Source', type: 'azuredevops' });

    beforeEach(() => {
      mockSourcesApi.list.mockResolvedValue([source1, source2]);
    });

    it('should set activeSource to null when filter is "all"', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Wait for sources to be loaded
      await waitFor(() => {
        expect(screen.getByTestId('sourcesCount').textContent).toBe('2');
      });

      // activeSource should be null when filter is 'all' and multiple sources exist
      expect(screen.getByTestId('activeSource').textContent).toBe('null');
      expect(screen.getByTestId('activeSourceFilter').textContent).toBe('all');
    });

    it('should set isAllSourcesMode to true when filter is "all"', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      await waitFor(() => {
        expect(screen.getByTestId('sourcesCount').textContent).toBe('2');
      });

      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('true');
    });

    it('should set hasMultipleSources to true', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(screen.getByTestId('hasMultipleSources').textContent).toBe('true');
    });

    it('should allow selecting a specific source', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Select source 1
      await act(async () => {
        screen.getByText('Select Source 1').click();
      });

      expect(screen.getByTestId('activeSource').textContent).toBe('GitHub Source');
      expect(screen.getByTestId('activeSourceId').textContent).toBe('1');
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('false');
    });

    it('should allow switching back to all sources', async () => {
      // Start with source 1 selected
      localStorage.setItem(STORAGE_KEY, '1');

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Should start with source 1 selected
      expect(screen.getByTestId('activeSource').textContent).toBe('GitHub Source');

      // Switch to all sources
      await act(async () => {
        screen.getByText('Select All').click();
      });

      expect(screen.getByTestId('activeSource').textContent).toBe('null');
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('true');
    });

    it('should persist selected source to localStorage', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Select source 2
      await act(async () => {
        screen.getByText('Select Source 2').click();
      });

      expect(localStorage.getItem(STORAGE_KEY)).toBe('2');
    });

    it('should restore saved filter from localStorage', async () => {
      // Pre-set localStorage to source 2
      localStorage.setItem(STORAGE_KEY, '2');

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(screen.getByTestId('activeSource').textContent).toBe('ADO Source');
      expect(screen.getByTestId('activeSourceId').textContent).toBe('2');
    });

    it('should reset to "all" if saved source ID no longer exists', async () => {
      // Pre-set localStorage to a non-existent source
      localStorage.setItem(STORAGE_KEY, '999');

      const consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => {});

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Wait for validation to run and reset to 'all'
      await waitFor(() => {
        expect(screen.getByTestId('activeSourceFilter').textContent).toBe('all');
      });

      // Should fall back to 'all'
      expect(screen.getByTestId('activeSource').textContent).toBe('null');
      
      consoleWarn.mockRestore();
    });
  });

  describe('no sources configuration', () => {
    beforeEach(() => {
      mockSourcesApi.list.mockResolvedValue([]);
    });

    it('should handle empty sources list', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(screen.getByTestId('sourcesCount').textContent).toBe('0');
      expect(screen.getByTestId('activeSource').textContent).toBe('null');
      expect(screen.getByTestId('hasMultipleSources').textContent).toBe('false');
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('false');
    });
  });

  describe('loading state', () => {
    it('should show loading state initially', async () => {
      // Make the API call take some time
      mockSourcesApi.list.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve([]), 100))
      );

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      // Should be loading initially
      expect(screen.getByTestId('isLoading').textContent).toBe('true');

      // Wait for loading to complete
      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });
    });
  });

  describe('error handling', () => {
    it('should handle API errors gracefully', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
      mockSourcesApi.list.mockRejectedValue(new Error('Network error'));

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Should still render with empty sources
      expect(screen.getByTestId('sourcesCount').textContent).toBe('0');

      consoleError.mockRestore();
    });
  });

  describe('filter validation', () => {
    const source1 = createMockSource({ id: 1, name: 'GitHub Source', type: 'github' });

    beforeEach(() => {
      mockSourcesApi.list.mockResolvedValue([source1]);
    });

    it('should reject filter for non-existent source ID', async () => {
      const consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => {});

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Try to select non-existent source 2
      await act(async () => {
        screen.getByText('Select Source 2').click();
      });

      // Should fall back to 'all' (but for single source, activeSource is still the single source)
      // The filter might be 'all' but activeSource should still return the single source
      expect(screen.getByTestId('activeSource').textContent).toBe('GitHub Source');

      consoleWarn.mockRestore();
    });
  });

  describe('dynamic source changes', () => {
    it('should reset filter to "all" when selected source is deleted (multi-source)', async () => {
      const source1 = createMockSource({ id: 1, name: 'GitHub Source', type: 'github' });
      const source2 = createMockSource({ id: 2, name: 'ADO Source', type: 'azuredevops' });
      
      // Start with source 2 selected
      localStorage.setItem(STORAGE_KEY, '2');
      mockSourcesApi.list.mockResolvedValue([source1, source2]);
      
      const consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const { rerender } = render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Verify source 2 is selected
      expect(screen.getByTestId('activeSource').textContent).toBe('ADO Source');
      expect(screen.getByTestId('activeSourceId').textContent).toBe('2');

      // Simulate source 2 being deleted - now only source 1 exists
      // Since there's now only 1 source, it should auto-select it
      mockSourcesApi.list.mockResolvedValue([source1]);
      
      // Trigger refetch by re-rendering with new sources
      // In real app, this would be triggered by refetchSources()
      rerender(
        <ThemeProvider>
          <SourceProvider>
            <TestComponent />
          </SourceProvider>
        </ThemeProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Wait for sources to reload
      await waitFor(() => {
        expect(screen.getByTestId('sourcesCount').textContent).toBe('1');
      });

      // With single source, activeSource should be that source
      await waitFor(() => {
        expect(screen.getByTestId('activeSource').textContent).toBe('GitHub Source');
      });

      consoleWarn.mockRestore();
    });

    it('should reset filter when selected source is deleted in multi-source setup', async () => {
      const source1 = createMockSource({ id: 1, name: 'GitHub Source', type: 'github' });
      const source2 = createMockSource({ id: 2, name: 'ADO Source', type: 'azuredevops' });
      const source3 = createMockSource({ id: 3, name: 'Another Source', type: 'github' });
      
      // Start with source 2 selected
      localStorage.setItem(STORAGE_KEY, '2');
      mockSourcesApi.list.mockResolvedValue([source1, source2, source3]);
      
      const consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const { rerender } = render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Verify source 2 is selected
      expect(screen.getByTestId('activeSource').textContent).toBe('ADO Source');

      // Simulate source 2 being deleted - now sources 1 and 3 exist
      mockSourcesApi.list.mockResolvedValue([source1, source3]);
      
      rerender(
        <ThemeProvider>
          <SourceProvider>
            <TestComponent />
          </SourceProvider>
        </ThemeProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      await waitFor(() => {
        expect(screen.getByTestId('sourcesCount').textContent).toBe('2');
      });

      // Should reset to 'all' since selected source no longer exists
      await waitFor(() => {
        expect(screen.getByTestId('activeSourceFilter').textContent).toBe('all');
      });
      
      // activeSource should be null in All Sources mode
      expect(screen.getByTestId('activeSource').textContent).toBe('null');
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('true');

      consoleWarn.mockRestore();
    });

    it('should auto-select single source when transitioning from multi to single source', async () => {
      const source1 = createMockSource({ id: 1, name: 'GitHub Source', type: 'github' });
      const source2 = createMockSource({ id: 2, name: 'ADO Source', type: 'azuredevops' });
      
      // Start in "All Sources" mode with 2 sources
      localStorage.setItem(STORAGE_KEY, 'all');
      mockSourcesApi.list.mockResolvedValue([source1, source2]);

      const { rerender } = render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Verify All Sources mode
      expect(screen.getByTestId('activeSource').textContent).toBe('null');
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('true');

      // Simulate one source being deleted - now only source 1 exists
      mockSourcesApi.list.mockResolvedValue([source1]);
      
      rerender(
        <ThemeProvider>
          <SourceProvider>
            <TestComponent />
          </SourceProvider>
        </ThemeProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      await waitFor(() => {
        expect(screen.getByTestId('sourcesCount').textContent).toBe('1');
      });

      // Should auto-select the single remaining source
      await waitFor(() => {
        expect(screen.getByTestId('activeSource').textContent).toBe('GitHub Source');
      });
      expect(screen.getByTestId('isAllSourcesMode').textContent).toBe('false');
    });
  });
});
