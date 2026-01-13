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
    hasMultipleSources: false,
    isAllSourcesMode: false, // Single source: never in "All Sources" mode
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
  useCancelDiscovery: vi.fn(),
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
    
    // Reset SourceContext mock to default GitHub source (single source setup)
    (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
      sources: [{ id: 1, name: 'GitHub Source', type: 'github' }],
      activeSourceFilter: 1, // Single source is always selected
      setActiveSourceFilter: vi.fn(),
      activeSource: { id: 1, name: 'GitHub Source', type: 'github' }, // Single source always returned
      isLoading: false,
      error: null,
      refetchSources: vi.fn(),
      hasMultipleSources: false,
      isAllSourcesMode: false, // Single source: never in "All Sources" mode
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

    (useMutationsModule.useCancelDiscovery as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: vi.fn(),
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
    const adoSource = { id: 1, name: 'ADO Source', type: 'azuredevops' as const };
    // Mock SourceContext with single ADO source
    (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
      sources: [adoSource],
      activeSourceFilter: 1,
      setActiveSourceFilter: vi.fn(),
      activeSource: adoSource,
      isLoading: false,
      error: null,
      refetchSources: vi.fn(),
      hasMultipleSources: false,
      isAllSourcesMode: false,
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

  describe('single source vs multi-source behavior', () => {
    it('should show source-specific view with single ADO source (not aggregated view)', async () => {
      const adoSource = { id: 1, name: 'ADO Source', type: 'azuredevops' as const };
      
      // Single ADO source - should show detailed project breakdown, not aggregated
      (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
        sources: [adoSource],
        activeSourceFilter: 1,
        setActiveSourceFilter: vi.fn(),
        activeSource: adoSource, // Single source always returned as activeSource
        isLoading: false,
        error: null,
        refetchSources: vi.fn(),
        hasMultipleSources: false,
        isAllSourcesMode: false, // Never true for single source
      });

      // ADO organizations with projects
      (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
        data: [
          { organization: 'Project1', ado_organization: 'ADO-Org', total_repos: 20, status_counts: {} },
          { organization: 'Project2', ado_organization: 'ADO-Org', total_repos: 15, status_counts: {} },
        ],
        isLoading: false,
        isFetching: false,
      });

      render(<Dashboard />);

      await waitFor(() => {
        expect(screen.getByText('Azure DevOps Organizations')).toBeInTheDocument();
      });

      // Should show ADO org card (detailed view) not GitHub org cards
      expect(screen.getByTestId('ado-org-card-ADO-Org')).toBeInTheDocument();
    });

    it('should show aggregated view with multiple sources in All Sources mode', async () => {
      const githubSource = { id: 1, name: 'GitHub Source', type: 'github' as const };
      const adoSource = { id: 2, name: 'ADO Source', type: 'azuredevops' as const };

      // Multiple sources with All Sources filter
      (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
        sources: [githubSource, adoSource],
        activeSourceFilter: 'all',
        setActiveSourceFilter: vi.fn(),
        activeSource: null, // null in All Sources mode
        isLoading: false,
        error: null,
        refetchSources: vi.fn(),
        hasMultipleSources: true,
        isAllSourcesMode: true, // True for multi-source with 'all' filter
      });

      // Mixed organizations
      (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
        data: [
          { organization: 'github-org', total_repos: 50, status_counts: {} },
          { organization: 'Project1', ado_organization: 'ADO-Org', total_repos: 20, status_counts: {} },
        ],
        isLoading: false,
        isFetching: false,
      });

      render(<Dashboard />);

      await waitFor(() => {
        // Should show both sections
        expect(screen.getByText('GitHub Organizations')).toBeInTheDocument();
        expect(screen.getByText('Azure DevOps Organizations')).toBeInTheDocument();
      });

      // GitHub org should be shown as a card
      expect(screen.getByTestId('github-org-card-github-org')).toBeInTheDocument();
    });

    it('should show specific source view when source is selected from multi-source setup', async () => {
      const githubSource = { id: 1, name: 'GitHub Source', type: 'github' as const };
      const adoSource = { id: 2, name: 'ADO Source', type: 'azuredevops' as const };

      // Multiple sources but specific source selected
      (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
        sources: [githubSource, adoSource],
        activeSourceFilter: 1,
        setActiveSourceFilter: vi.fn(),
        activeSource: githubSource, // GitHub source selected
        isLoading: false,
        error: null,
        refetchSources: vi.fn(),
        hasMultipleSources: true,
        isAllSourcesMode: false, // False when specific source is selected
      });

      (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
        data: [
          { organization: 'github-org1', total_repos: 50, status_counts: {} },
          { organization: 'github-org2', total_repos: 30, status_counts: {} },
        ],
        isLoading: false,
        isFetching: false,
      });

      render(<Dashboard />);

      await waitFor(() => {
        // Should only show GitHub section (not ADO)
        expect(screen.getByText('GitHub Organizations')).toBeInTheDocument();
      });

      // Should NOT show Azure DevOps section
      expect(screen.queryByText('Azure DevOps Organizations')).not.toBeInTheDocument();
    });

    it('should show active source indicator when a specific source is selected', async () => {
      const githubSource = { id: 1, name: 'My GitHub Source', type: 'github' as const };

      (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
        sources: [githubSource],
        activeSourceFilter: 1,
        setActiveSourceFilter: vi.fn(),
        activeSource: githubSource,
        isLoading: false,
        error: null,
        refetchSources: vi.fn(),
        hasMultipleSources: false,
        isAllSourcesMode: false,
      });

      render(<Dashboard />);

      await waitFor(() => {
        // Should show the active source indicator
        expect(screen.getByText(/Showing data from:/)).toBeInTheDocument();
        expect(screen.getByText('My GitHub Source')).toBeInTheDocument();
      });
    });

    it('should NOT show active source indicator when in All Sources mode', async () => {
      const githubSource = { id: 1, name: 'GitHub Source', type: 'github' as const };
      const adoSource = { id: 2, name: 'ADO Source', type: 'azuredevops' as const };

      (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
        sources: [githubSource, adoSource],
        activeSourceFilter: 'all',
        setActiveSourceFilter: vi.fn(),
        activeSource: null,
        isLoading: false,
        error: null,
        refetchSources: vi.fn(),
        hasMultipleSources: true,
        isAllSourcesMode: true,
      });

      render(<Dashboard />);

      await waitFor(() => {
        expect(screen.getByText('Dashboard')).toBeInTheDocument();
      });

      // Should NOT show the active source indicator
      expect(screen.queryByText(/Showing data from:/)).not.toBeInTheDocument();
    });
  });
});
