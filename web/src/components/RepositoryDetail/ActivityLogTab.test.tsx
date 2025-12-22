import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@primer/react';
import { ActivityLogTab } from './ActivityLogTab';
import { ToastProvider } from '../../contexts/ToastContext';
import { api } from '../../services/api';
import type { Repository, MigrationHistory, MigrationLog } from '../../types';

// Mock the API
vi.mock('../../services/api', () => ({
  api: {
    getMigrationHistory: vi.fn(),
    getMigrationLogs: vi.fn(),
  },
}));

// Create a wrapper with providers
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ToastProvider>{children}</ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

const mockRepository: Repository = {
  id: 1,
  full_name: 'org/repo',
  name: 'repo',
  org_name: 'org',
  source: 'github',
  status: 'pending',
  default_branch: 'main',
  source_url: 'https://github.com/org/repo',
  commit_count: 100,
  branch_count: 5,
  pr_count: 10,
  issue_count: 5,
  visibility: 'private',
  is_archived: false,
  is_fork: false,
  has_lfs: false,
  has_submodules: false,
  has_large_files: false,
  total_size: 1024000,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  complexity_score: 25,
  complexity_level: 'low',
};

const mockHistory: MigrationHistory[] = [
  {
    id: 1,
    repository_id: 1,
    phase: 'pre_migration',
    status: 'completed',
    message: 'Pre-migration checks completed',
    started_at: '2024-01-01T10:00:00Z',
    completed_at: '2024-01-01T10:05:00Z',
    duration_seconds: 300,
    metadata: '{}',
  },
  {
    id: 2,
    repository_id: 1,
    phase: 'migration',
    status: 'failed',
    message: 'Migration failed',
    error_message: 'Network timeout',
    started_at: '2024-01-01T10:10:00Z',
    completed_at: '2024-01-01T10:15:00Z',
    duration_seconds: 300,
    metadata: '{}',
  },
];

const mockLogs: MigrationLog[] = [
  {
    id: 1,
    repository_id: 1,
    level: 'INFO',
    phase: 'discovery',
    operation: 'scan',
    message: 'Starting repository scan',
    timestamp: '2024-01-01T09:00:00Z',
    details: 'Scanning for metadata',
    initiated_by: 'system',
  },
  {
    id: 2,
    repository_id: 1,
    level: 'ERROR',
    phase: 'migration',
    operation: 'transfer',
    message: 'Transfer failed',
    timestamp: '2024-01-01T10:00:00Z',
  },
];

describe('ActivityLogTab', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (api.getMigrationHistory as ReturnType<typeof vi.fn>).mockResolvedValue(mockHistory);
    (api.getMigrationLogs as ReturnType<typeof vi.fn>).mockResolvedValue({ logs: mockLogs });
  });

  it('should render with history view by default', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    expect(screen.getByText('Migration History')).toBeInTheDocument();
    expect(screen.getByText('Detailed Logs')).toBeInTheDocument();

    await waitFor(() => {
      expect(api.getMigrationHistory).toHaveBeenCalledWith(1);
    });
  });

  it('should display migration history events', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Pre-migration checks completed')).toBeInTheDocument();
      expect(screen.getByText('Migration failed')).toBeInTheDocument();
    });
  });

  it('should display error message in history events', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Network timeout')).toBeInTheDocument();
    });
  });

  it('should show empty state when no history', async () => {
    (api.getMigrationHistory as ReturnType<typeof vi.fn>).mockResolvedValue([]);

    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('No migration history yet')).toBeInTheDocument();
    });
  });

  it('should switch to logs view when clicking Detailed Logs tab', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(api.getMigrationLogs).toHaveBeenCalled();
    });
  });

  it('should display log entries in logs view', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(screen.getByText('Starting repository scan')).toBeInTheDocument();
      expect(screen.getByText('Transfer failed')).toBeInTheDocument();
    });
  });

  it('should render log level filter dropdown in logs view', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(screen.getByText('All Levels')).toBeInTheDocument();
    });
  });

  it('should render phase filter dropdown in logs view', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(screen.getByText('All Phases')).toBeInTheDocument();
    });
  });

  it('should render search input for logs', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Search logs...')).toBeInTheDocument();
    });
  });

  it('should have refresh button in logs view', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(screen.getByText('Refresh')).toBeInTheDocument();
    });
  });

  it('should filter logs by search text', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(screen.getByText('Starting repository scan')).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText('Search logs...');
    fireEvent.change(searchInput, { target: { value: 'failed' } });

    // Only the failed log should be visible
    expect(screen.getByText('Transfer failed')).toBeInTheDocument();
    expect(screen.queryByText('Starting repository scan')).not.toBeInTheDocument();
  });

  it('should show empty state when no logs available', async () => {
    (api.getMigrationLogs as ReturnType<typeof vi.fn>).mockResolvedValue({ logs: [] });

    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    const logsTab = screen.getByText('Detailed Logs');
    fireEvent.click(logsTab);

    await waitFor(() => {
      expect(screen.getByText('No logs available')).toBeInTheDocument();
    });
  });

  it('should display phase badge in history events', async () => {
    render(<ActivityLogTab repository={mockRepository} />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('pre_migration')).toBeInTheDocument();
      expect(screen.getByText('migration')).toBeInTheDocument();
    });
  });
});
