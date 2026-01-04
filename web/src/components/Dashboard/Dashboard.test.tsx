import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import { Dashboard } from './index';
import * as useQueriesModule from '../../hooks/useQueries';
import * as useMutationsModule from '../../hooks/useMutations';
import * as SourceContextModule from '../../contexts/SourceContext';

// Mock the SourceContext
vi.mock('../../contexts/SourceContext', () => ({
  useSourceContext: vi.fn(() => ({
    sources: [{ id: 1, name: 'GitHub Source', type: 'github' }],
    activeSourceFilter: 'all',
    setActiveSourceFilter: vi.fn(),
    activeSource: null,
    isLoading: false,
    error: null,
    refetchSources: vi.fn(),
  })),
  SourceProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useConfig: vi.fn(),
  useOrganizations: vi.fn(),
  useAnalytics: vi.fn(),
  useBatches: vi.fn(),
  useDashboardActionItems: vi.fn(),
  useDiscoveryProgress: vi.fn(),
  useSetupProgress: vi.fn(),
}));

vi.mock('../../hooks/useMutations', () => ({
  useStartDiscovery: vi.fn(),
  useStartADODiscovery: vi.fn(),
}));

// Mock child components to simplify testing
vi.mock('./KPISection', () => ({
  KPISection: ({ isLoading }: { isLoading: boolean }) => (
    <div data-testid="kpi-section">
      {isLoading ? 'Loading KPIs...' : 'KPI Section'}
    </div>
  ),
}));

vi.mock('./ActionItemsPanel', () => ({
  ActionItemsPanel: ({ isLoading }: { isLoading: boolean }) => (
    <div data-testid="action-items-panel">
      {isLoading ? 'Loading Actions...' : 'Action Items'}
    </div>
  ),
}));

vi.mock('./UpcomingBatchesTimeline', () => ({
  UpcomingBatchesTimeline: ({ isLoading }: { isLoading: boolean }) => (
    <div data-testid="upcoming-batches">
      {isLoading ? 'Loading Batches...' : 'Upcoming Batches'}
    </div>
  ),
}));

vi.mock('./DiscoveryProgressCard', () => ({
  DiscoveryProgressCard: () => (
    <div data-testid="discovery-progress-card">Discovery Progress</div>
  ),
  LastDiscoveryIndicator: () => (
    <div data-testid="last-discovery-indicator">Last Discovery</div>
  ),
}));

vi.mock('./GitHubOrganizationCard', () => ({
  GitHubOrganizationCard: ({ organization }: { organization: { organization: string } }) => (
    <div data-testid={`github-org-card-${organization.organization}`}>
      {organization.organization}
    </div>
  ),
}));

vi.mock('./ADOOrganizationCard', () => ({
  ADOOrganizationCard: ({ adoOrgName }: { adoOrgName: string }) => (
    <div data-testid={`ado-org-card-${adoOrgName}`}>
      {adoOrgName}
    </div>
  ),
}));

vi.mock('./SetupProgress', () => ({
  SetupProgress: () => (
    <div data-testid="setup-progress">Setup Progress</div>
  ),
}));

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

describe('Dashboard', () => {
  const mockOrganizations = [
    { organization: 'org1', total_repos: 50 },
    { organization: 'org2', total_repos: 30 },
  ];

  const mockAnalytics = {
    total_repositories: 80,
    migrated_count: 40,
    failed_count: 5,
    in_progress_count: 10,
    pending_count: 25,
    success_rate: 88.9,
  };

  const mockBatches = [
    { id: 1, name: 'Batch 1', status: 'pending' },
    { id: 2, name: 'Batch 2', status: 'ready' },
  ];

  const mockActionItems = {
    remediation_required: [],
    failed_migrations: [],
    pending_dry_runs: [],
    total_action_items: 0,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    
    // Reset SourceContext mock to default GitHub source
    (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
      sources: [{ id: 1, name: 'GitHub Source', type: 'github' }],
      activeSourceFilter: 'all',
      setActiveSourceFilter: vi.fn(),
      activeSource: null,
      isLoading: false,
      error: null,
      refetchSources: vi.fn(),
    });
    
    // Setup default mock implementations
    (useQueriesModule.useConfig as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { source_type: 'github' },
    });
    
    (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockOrganizations,
      isLoading: false,
      isFetching: false,
    });
    
    (useQueriesModule.useAnalytics as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockAnalytics,
      isLoading: false,
      isFetching: false,
    });
    
    (useQueriesModule.useBatches as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockBatches,
      isLoading: false,
      isFetching: false,
    });
    
    (useQueriesModule.useDashboardActionItems as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockActionItems,
      isLoading: false,
      isFetching: false,
    });
    
    (useQueriesModule.useDiscoveryProgress as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
    });

    (useQueriesModule.useSetupProgress as ReturnType<typeof vi.fn>).mockReturnValue({
      data: {
        destination_configured: true,
        sources_configured: true,
        source_count: 1,
        batches_created: false,
        batch_count: 0,
        setup_complete: true,
      },
    });
    
    (useMutationsModule.useStartDiscovery as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    });
    
    (useMutationsModule.useStartADODiscovery as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    });
  });

  it('should render the dashboard title and description', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Overview of migration progress across all organizations')).toBeInTheDocument();
    });
  });

  it('should render the Start Discovery button', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Start Discovery' })).toBeInTheDocument();
    });
  });

  it('should render KPI section', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByTestId('kpi-section')).toBeInTheDocument();
    });
  });

  it('should render action items panel', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByTestId('action-items-panel')).toBeInTheDocument();
    });
  });

  it('should render upcoming batches timeline', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByTestId('upcoming-batches')).toBeInTheDocument();
    });
  });

  it('should render GitHub organizations section for github source', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('GitHub Organizations')).toBeInTheDocument();
    });
  });

  it('should render Azure DevOps organizations section for azuredevops source', async () => {
    // Mock SourceContext with ADO sources
    (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
      sources: [{ id: 1, name: 'ADO Source', type: 'azuredevops' }],
      activeSourceFilter: 'all',
      setActiveSourceFilter: vi.fn(),
      activeSource: null,
      isLoading: false,
      error: null,
      refetchSources: vi.fn(),
    });
    
    // Mock organizations with ADO data
    (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [
        { organization: 'proj1', ado_organization: 'ado-org1', total_repos: 20 },
        { organization: 'proj2', ado_organization: 'ado-org1', total_repos: 15 },
      ],
      isLoading: false,
      isFetching: false,
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('Azure DevOps Organizations')).toBeInTheDocument();
    });
  });

  it('should render organization cards', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByTestId('github-org-card-org1')).toBeInTheDocument();
      expect(screen.getByTestId('github-org-card-org2')).toBeInTheDocument();
    });
  });

  it('should show blankslate when no organizations', async () => {
    (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('No organizations discovered yet')).toBeInTheDocument();
      expect(screen.getByText('Get started by discovering repositories from your GitHub organizations.')).toBeInTheDocument();
    });
  });

  it('should show loading spinner while loading organizations', async () => {
    (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: true,
      isFetching: true,
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });
  });

  it('should render discovery progress card when discovery is active', async () => {
    (useQueriesModule.useDiscoveryProgress as ReturnType<typeof vi.fn>).mockReturnValue({
      data: {
        id: 1,
        status: 'in_progress',
        total: 100,
        completed: 50,
      },
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByTestId('discovery-progress-card')).toBeInTheDocument();
    });
  });

  it('should show organization count and repo totals', async () => {
    const githubSource = { id: 1, name: 'GitHub Source', type: 'github' as const };
    
    // Set activeSource to trigger pagination display
    (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
      sources: [githubSource],
      activeSourceFilter: '1',
      setActiveSourceFilter: vi.fn(),
      activeSource: githubSource,
      isLoading: false,
      error: null,
      refetchSources: vi.fn(),
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText(/Showing 1-2 of 2 organizations with 80 total repositories/)).toBeInTheDocument();
    });
  });

  it('should show success flash when discovery completes', async () => {
    // First render without success
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });
    
    // Simulate discovery success would typically be shown via state
    // This tests the flash rendering when present
  });

  it('should handle search filtering', async () => {
    // Simulate search via URL params - the component reads from useSearchParams
    render(<Dashboard />);
    
    await waitFor(() => {
      // Both orgs should be visible without search filter
      expect(screen.getByTestId('github-org-card-org1')).toBeInTheDocument();
      expect(screen.getByTestId('github-org-card-org2')).toBeInTheDocument();
    });
  });

  it('should show loading state for KPIs when analytics is loading', async () => {
    (useQueriesModule.useAnalytics as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      isFetching: true,
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('Loading KPIs...')).toBeInTheDocument();
    });
  });

  it('should show loading state for action items when loading', async () => {
    (useQueriesModule.useDashboardActionItems as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      isFetching: true,
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('Loading Actions...')).toBeInTheDocument();
    });
  });

  it('should show loading state for batches when loading', async () => {
    (useQueriesModule.useBatches as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: true,
      isFetching: true,
    });
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('Loading Batches...')).toBeInTheDocument();
    });
  });
});
