import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@primer/react';
import { DependenciesTab } from './DependenciesTab';
import { ToastProvider } from '../../contexts/ToastContext';
import { api } from '../../services/api';
import type { DependenciesResponse, DependentsResponse } from '../../types';

// Mock the API
vi.mock('../../services/api', () => ({
  api: {
    getRepositoryDependencies: vi.fn(),
    getRepositoryDependents: vi.fn(),
    exportRepositoryDependencies: vi.fn(),
  },
}));

// Create a wrapper with providers
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <ToastProvider>{children}</ToastProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

const mockDependenciesResponse: DependenciesResponse = {
  dependencies: [
    {
      id: 1,
      repository_id: 1,
      dependency_full_name: 'org/dep1',
      dependency_url: 'https://github.com/org/dep1',
      dependency_type: 'submodule',
      is_local: true,
      metadata: JSON.stringify({ path: '.submodules/dep1', branch: 'main' }),
      detected_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 2,
      repository_id: 1,
      dependency_full_name: 'external/lib',
      dependency_url: 'https://github.com/external/lib',
      dependency_type: 'workflow',
      is_local: false,
      metadata: JSON.stringify({ workflow_file: '.github/workflows/ci.yml', ref: 'v1' }),
      detected_at: '2024-01-01T00:00:00Z',
    },
  ],
  summary: {
    total: 2,
    local: 1,
    external: 1,
    by_type: {
      submodule: 1,
      workflow: 1,
    },
  },
};

const mockDependentsResponse: DependentsResponse = {
  dependents: [
    {
      id: 2,
      full_name: 'org/consumer1',
      name: 'consumer1',
      status: 'pending',
      source_url: 'https://github.com/org/consumer1',
      dependency_types: ['submodule'],
    },
  ],
  total: 1,
};

describe('DependenciesTab', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (api.getRepositoryDependencies as ReturnType<typeof vi.fn>).mockResolvedValue(mockDependenciesResponse);
    (api.getRepositoryDependents as ReturnType<typeof vi.fn>).mockResolvedValue(mockDependentsResponse);
  });

  it('should show loading spinner initially', () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
  });

  it('should display local dependencies warning when present', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Local Dependencies Detected')).toBeInTheDocument();
    });
  });

  it('should have export button', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Export')).toBeInTheDocument();
    });
  });

  it('should display dependency entries', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('org/dep1')).toBeInTheDocument();
      expect(screen.getByText('external/lib')).toBeInTheDocument();
    });
  });

  it('should show filter buttons for scope', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /All/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Local/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /External/ })).toBeInTheDocument();
    });
  });

  it('should filter by local dependencies', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('org/dep1')).toBeInTheDocument();
    });

    const localButton = screen.getByRole('button', { name: /Local/ });
    fireEvent.click(localButton);

    // After filtering, only local dependencies should be visible
    expect(screen.getByText('org/dep1')).toBeInTheDocument();
    expect(screen.queryByText('external/lib')).not.toBeInTheDocument();
  });

  it('should filter by external dependencies', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('external/lib')).toBeInTheDocument();
    });

    const externalButton = screen.getByRole('button', { name: /External/ });
    fireEvent.click(externalButton);

    // After filtering, only external dependencies should be visible
    expect(screen.queryByText('org/dep1')).not.toBeInTheDocument();
    expect(screen.getByText('external/lib')).toBeInTheDocument();
  });

  it('should show error state when API fails', async () => {
    (api.getRepositoryDependencies as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error('Network error'));

    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Error loading dependencies')).toBeInTheDocument();
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  it('should show no dependencies message when empty', async () => {
    (api.getRepositoryDependencies as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      dependencies: [],
      summary: { total: 0, local: 0, external: 0, by_type: {} },
    });
    (api.getRepositoryDependents as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      dependents: [],
      total: 0,
    });

    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('No dependencies found')).toBeInTheDocument();
    });
  });

  it('should display Local badge for local dependencies', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Local')).toBeInTheDocument();
    });
  });

  it('should display External badge for external dependencies', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('External')).toBeInTheDocument();
    });
  });

  it('should show export dropdown on button click', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Export')).toBeInTheDocument();
    });

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    expect(screen.getByText('Export as CSV')).toBeInTheDocument();
    expect(screen.getByText('Export as JSON')).toBeInTheDocument();
  });

  it('should display metadata details', async () => {
    render(<DependenciesTab fullName="org/repo" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('.submodules/dep1')).toBeInTheDocument();
    });
  });
});
