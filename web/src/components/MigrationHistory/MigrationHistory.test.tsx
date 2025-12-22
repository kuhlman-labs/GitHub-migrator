import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { MigrationHistory } from './index';

const mockMigrations = [
  {
    id: 1,
    full_name: 'org/repo1',
    action: 'migration',
    status: 'complete',
    dry_run: false,
    started_at: '2024-01-15T10:00:00Z',
    completed_at: '2024-01-15T10:30:00Z',
    duration_seconds: 1800,
  },
  {
    id: 2,
    full_name: 'org/repo2',
    action: 'migration',
    status: 'failed',
    dry_run: false,
    started_at: '2024-01-15T11:00:00Z',
    completed_at: '2024-01-15T11:05:00Z',
    duration_seconds: 300,
    error: 'Connection timeout',
  },
  {
    id: 3,
    full_name: 'org/repo3',
    action: 'dry_run',
    status: 'complete',
    dry_run: true,
    started_at: '2024-01-15T12:00:00Z',
    completed_at: '2024-01-15T12:10:00Z',
    duration_seconds: 600,
  },
];

vi.mock('../../hooks/useQueries', () => ({
  useMigrationHistory: () => ({
    data: { migrations: mockMigrations, total: mockMigrations.length },
    isLoading: false,
    isFetching: false,
  }),
}));

describe('MigrationHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the page header', () => {
    render(<MigrationHistory />);

    expect(screen.getByRole('heading', { name: /Migration History/i })).toBeInTheDocument();
    expect(screen.getByText('Audit trail of all migration attempts and outcomes')).toBeInTheDocument();
  });

  it('displays migration entries', () => {
    render(<MigrationHistory />);

    expect(screen.getByText('org/repo1')).toBeInTheDocument();
    expect(screen.getByText('org/repo2')).toBeInTheDocument();
    expect(screen.getByText('org/repo3')).toBeInTheDocument();
  });

  it('displays export button', () => {
    render(<MigrationHistory />);

    expect(screen.getByRole('button', { name: /Export/i })).toBeInTheDocument();
  });

  it('displays table headers', () => {
    render(<MigrationHistory />);

    expect(screen.getByText('Repository')).toBeInTheDocument();
    // Other headers may have different exact text
  });

  it('displays migration count', () => {
    render(<MigrationHistory />);

    // Should show "3 migrations" or similar
    expect(screen.getByText(/3 migrations/i)).toBeInTheDocument();
  });
});
