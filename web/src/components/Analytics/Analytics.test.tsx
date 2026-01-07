import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { Analytics } from './index';
import * as useQueriesModule from '../../hooks/useQueries';
import * as SourceContextModule from '../../contexts/SourceContext';

// Mock SourceContext
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

// Mock recharts to avoid rendering issues in tests
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  BarChart: ({ children }: { children: React.ReactNode }) => <div data-testid="bar-chart">{children}</div>,
  Bar: () => null,
  PieChart: ({ children }: { children: React.ReactNode }) => <div data-testid="pie-chart">{children}</div>,
  Pie: () => null,
  Cell: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
}));

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useConfig: vi.fn(),
  useAnalytics: vi.fn(),
}));

// Mock child components
vi.mock('./FilterBar', () => ({
  FilterBar: () => <div data-testid="filter-bar">Filter Bar</div>,
}));

vi.mock('./MigrationTrendChart', () => ({
  MigrationTrendChart: () => <div data-testid="migration-trend-chart">Migration Trend Chart</div>,
}));

vi.mock('./ComplexityChart', () => ({
  ComplexityChart: () => <div data-testid="complexity-chart">Complexity Chart</div>,
}));

vi.mock('./KPICard', () => ({
  KPICard: ({ title, value, subtitle }: { title: string; value: string | number; subtitle?: string }) => (
    <div data-testid={`kpi-card-${title.toLowerCase().replace(/\s+/g, '-')}`}>
      <span>{title}</span>
      <span>{value}</span>
      {subtitle && <span>{subtitle}</span>}
    </div>
  ),
}));

describe('Analytics', () => {
  const mockAnalytics = {
    total_repositories: 100,
    migrated_count: 50,
    failed_count: 5,
    in_progress_count: 10,
    pending_count: 35,
    success_rate: 90.9,
    average_migration_time: 3600000, // 1 hour
    median_migration_time: 1800000, // 30 min
    estimated_completion_date: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
    migration_velocity: {
      repos_per_week: 12.5,
      repos_per_day: 1.8,
    },
    status_breakdown: {
      pending: 35,
      in_progress: 10,
      migration_complete: 50,
      failed: 5,
    },
    complexity_distribution: [
      { category: 'simple', count: 60 },
      { category: 'medium', count: 25 },
      { category: 'complex', count: 10 },
      { category: 'very_complex', count: 5 },
    ],
    size_distribution: [
      { category: 'small', count: 40 },
      { category: 'medium', count: 35 },
      { category: 'large', count: 20 },
      { category: 'very_large', count: 5 },
    ],
    organization_stats: [
      { organization: 'org1', total_repos: 60 },
      { organization: 'org2', total_repos: 40 },
    ],
    feature_stats: {
      total_repositories: 100,
      has_lfs: 20,
      has_actions: 45,
      has_wiki: 30,
      is_archived: 10,
    },
    migration_completion_stats: [
      { organization: 'org1', total_repos: 60, completed_count: 35, in_progress_count: 5, pending_count: 15, failed_count: 5 },
      { organization: 'org2', total_repos: 40, completed_count: 15, in_progress_count: 5, pending_count: 20, failed_count: 0 },
    ],
    migration_time_series: [],
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
    
    (useQueriesModule.useConfig as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { source_type: 'github' },
    });
    
    (useQueriesModule.useAnalytics as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockAnalytics,
      isLoading: false,
      isFetching: false,
    });
  });

  it('should render the analytics dashboard title', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Migration metrics and insights for reporting and planning')).toBeInTheDocument();
    });
  });

  it('should render the filter bar', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByTestId('filter-bar')).toBeInTheDocument();
    });
  });

  it('should render tabs for discovery and migration analytics', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      // Check for the navigation element with the tabs
      expect(screen.getByRole('navigation', { name: 'Analytics sections' })).toBeInTheDocument();
    });
  });

  it('should show discovery analytics by default', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      // Check for discovery-specific content
      expect(screen.getByText('Source environment overview to drive batch planning decisions')).toBeInTheDocument();
    });
  });

  it('should switch to migration analytics tab', async () => {
    const user = userEvent.setup();
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByText('Migration Analytics')).toBeInTheDocument();
    });
    
    await user.click(screen.getByText('Migration Analytics'));
    
    await waitFor(() => {
      expect(screen.getByText('Migration progress and performance for executive reporting')).toBeInTheDocument();
    });
  });

  it('should render KPI cards in discovery tab', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByTestId('kpi-card-total-repositories')).toBeInTheDocument();
      expect(screen.getByTestId('kpi-card-organizations')).toBeInTheDocument();
      expect(screen.getByTestId('kpi-card-high-complexity')).toBeInTheDocument();
      expect(screen.getByTestId('kpi-card-features-detected')).toBeInTheDocument();
    });
  });

  it('should render complexity chart', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByTestId('complexity-chart')).toBeInTheDocument();
    });
  });

  it('should render size distribution chart', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByText('Repository Size Distribution')).toBeInTheDocument();
    });
  });

  it('should render organization breakdown table', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByText('Organization Breakdown')).toBeInTheDocument();
      expect(screen.getByText('org1')).toBeInTheDocument();
      expect(screen.getByText('org2')).toBeInTheDocument();
    });
  });

  it('should render feature usage statistics', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByText('Feature Usage Statistics')).toBeInTheDocument();
    });
  });

  it('should render migration KPIs in migration tab', async () => {
    const user = userEvent.setup();
    render(<Analytics />);
    
    await user.click(screen.getByText('Migration Analytics'));
    
    await waitFor(() => {
      expect(screen.getByTestId('kpi-card-completion-rate')).toBeInTheDocument();
      expect(screen.getByTestId('kpi-card-migration-velocity')).toBeInTheDocument();
      expect(screen.getByTestId('kpi-card-success-rate')).toBeInTheDocument();
      expect(screen.getByTestId('kpi-card-est.-completion')).toBeInTheDocument();
    });
  });

  it('should render migration trend chart in migration tab', async () => {
    const user = userEvent.setup();
    render(<Analytics />);
    
    await user.click(screen.getByText('Migration Analytics'));
    
    await waitFor(() => {
      expect(screen.getByTestId('migration-trend-chart')).toBeInTheDocument();
    });
  });

  it('should show loading spinner when loading', async () => {
    (useQueriesModule.useAnalytics as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      isFetching: true,
    });
    
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });
  });

  it('should show no data message when analytics is empty', async () => {
    (useQueriesModule.useAnalytics as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: false,
      isFetching: false,
    });
    
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByText('No analytics data available')).toBeInTheDocument();
    });
  });

  it('should render export button', async () => {
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /export/i })).toBeInTheDocument();
    });
  });

  it('should show export menu when export button is clicked', async () => {
    const user = userEvent.setup();
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /export/i })).toBeInTheDocument();
    });
    
    await user.click(screen.getByRole('button', { name: /export/i }));
    
    await waitFor(() => {
      expect(screen.getByText('Executive Report')).toBeInTheDocument();
      expect(screen.getByText('Discovery Report')).toBeInTheDocument();
      expect(screen.getAllByText('Export as CSV').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Export as JSON').length).toBeGreaterThan(0);
    });
  });

  it('should render migration progress by organization in migration tab', async () => {
    const user = userEvent.setup();
    render(<Analytics />);
    
    await user.click(screen.getByText('Migration Analytics'));
    
    await waitFor(() => {
      expect(screen.getByText('Migration Progress by Organization')).toBeInTheDocument();
    });
  });

  it('should render performance metrics in migration tab', async () => {
    const user = userEvent.setup();
    render(<Analytics />);
    
    await user.click(screen.getByText('Migration Analytics'));
    
    await waitFor(() => {
      expect(screen.getByText('Performance Metrics')).toBeInTheDocument();
      expect(screen.getByText('Average Migration Time')).toBeInTheDocument();
      expect(screen.getByText('Median Migration Time')).toBeInTheDocument();
    });
  });

  it('should render detailed status breakdown table in migration tab', async () => {
    const user = userEvent.setup();
    render(<Analytics />);
    
    await user.click(screen.getByText('Migration Analytics'));
    
    await waitFor(() => {
      expect(screen.getByText('Detailed Status Breakdown')).toBeInTheDocument();
    });
  });

  it('should show ADO-specific labels for azuredevops source', async () => {
    const adoSource = { id: 1, name: 'ADO Source', type: 'azuredevops' as const };
    
    // Mock SourceContext with ADO sources and activeSource set
    (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
      sources: [adoSource],
      activeSourceFilter: '1',
      setActiveSourceFilter: vi.fn(),
      activeSource: adoSource,
      isLoading: false,
      error: null,
      refetchSources: vi.fn(),
    });
    
    (useQueriesModule.useAnalytics as ReturnType<typeof vi.fn>).mockReturnValue({
      data: {
        ...mockAnalytics,
        project_stats: [
          { organization: 'proj1', total_repos: 60 },
          { organization: 'proj2', total_repos: 40 },
        ],
      },
      isLoading: false,
      isFetching: false,
    });
    
    render(<Analytics />);
    
    await waitFor(() => {
      expect(screen.getByText('Project Breakdown')).toBeInTheDocument();
    });
  });
});
