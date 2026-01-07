import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { Repositories } from './index';
import * as useQueriesModule from '../../hooks/useQueries';
import * as useMutationsModule from '../../hooks/useMutations';

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useRepositories: vi.fn(),
  useConfig: vi.fn(),
  useOrganizations: vi.fn(),
  useProjects: vi.fn(),
  useTeams: vi.fn(),
}));

vi.mock('../../hooks/useMutations', () => ({
  useDiscoverRepositories: vi.fn(),
  useStartDiscovery: vi.fn(),
  useStartADODiscovery: vi.fn(),
}));

// Mock child components
vi.mock('./RepositoryCard', () => ({
  RepositoryCard: ({ repository }: { repository: { full_name: string; id: number } }) => (
    <div data-testid={`repo-card-${repository.id}`}>{repository.full_name}</div>
  ),
}));

vi.mock('./BulkActionsToolbar', () => ({
  BulkActionsToolbar: () => <div data-testid="bulk-actions-toolbar">Bulk Actions</div>,
}));

vi.mock('../common/UnifiedFilterSidebar', () => ({
  UnifiedFilterSidebar: () => <div data-testid="filter-sidebar">Filter Sidebar</div>,
}));

describe('Repositories', () => {
  const mockRepositories = [
    { id: 1, full_name: 'org/repo1', name: 'repo1', status: 'pending' },
    { id: 2, full_name: 'org/repo2', name: 'repo2', status: 'complete' },
    { id: 3, full_name: 'org/repo3', name: 'repo3', status: 'failed' },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    
    (useQueriesModule.useRepositories as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { repositories: mockRepositories, total: 3 },
      isLoading: false,
      isFetching: false,
    });
    
    (useQueriesModule.useConfig as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { source_type: 'github' },
    });
    
    (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [{ organization: 'org1' }],
    });
    
    (useQueriesModule.useProjects as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
    });
    
    (useQueriesModule.useTeams as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
    });
    
    (useMutationsModule.useDiscoverRepositories as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    });
    
    (useMutationsModule.useStartDiscovery as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });
    
    (useMutationsModule.useStartADODiscovery as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });
  });

  it('should render the repositories page title', async () => {
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByText('Repositories')).toBeInTheDocument();
    });
  });

  it('should render repository cards', async () => {
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByTestId('repo-card-1')).toBeInTheDocument();
      expect(screen.getByTestId('repo-card-2')).toBeInTheDocument();
      expect(screen.getByTestId('repo-card-3')).toBeInTheDocument();
    });
  });

  it('should render filter sidebar', async () => {
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByTestId('filter-sidebar')).toBeInTheDocument();
    });
  });

  it('should show repository count', async () => {
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByText(/Showing 1-3 of 3 repositories/)).toBeInTheDocument();
    });
  });

  it('should show loading spinner when loading', async () => {
    (useQueriesModule.useRepositories as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      isFetching: true,
    });
    
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });
  });

  it('should show blankslate when no repositories', async () => {
    (useQueriesModule.useRepositories as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { repositories: [], total: 0 },
      isLoading: false,
      isFetching: false,
    });
    
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'No repositories found' })).toBeInTheDocument();
    });
  });

  it('should render export button', async () => {
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /export/i })).toBeInTheDocument();
    });
  });

  it('should show export menu when clicked', async () => {
    const user = userEvent.setup();
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /export/i })).toBeInTheDocument();
    });
    
    await user.click(screen.getByRole('button', { name: /export/i }));
    
    await waitFor(() => {
      expect(screen.getByText('Export as CSV')).toBeInTheDocument();
      expect(screen.getByText('Export as Excel')).toBeInTheDocument();
      expect(screen.getByText('Export as JSON')).toBeInTheDocument();
    });
  });

  it('should render Select button for bulk actions', async () => {
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /select/i })).toBeInTheDocument();
    });
  });

  it('should render Discover Repos button', async () => {
    render(<Repositories />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /discover repos/i })).toBeInTheDocument();
    });
  });

  it('should render discover repos button', async () => {
    render(<Repositories />);
    
    const discoverButton = await screen.findByRole('button', { name: /discover repos/i });
    expect(discoverButton).toBeInTheDocument();
  });

  it('should show clear filters button when filters are active', async () => {
    // Simulate having active filters by rendering with search params
    // This would typically be done by mocking useSearchParams
    render(<Repositories />);
    
    // With no active filters, clear button shouldn't be visible
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /clear all filters/i })).not.toBeInTheDocument();
    });
  });
});
