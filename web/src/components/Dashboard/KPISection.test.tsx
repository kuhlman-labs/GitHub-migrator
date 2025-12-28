import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { KPISection } from './KPISection';
import type { Analytics } from '../../types';

const mockAnalytics: Analytics = {
  total_repositories: 100,
  migrated_count: 50,
  failed_count: 5,
  in_progress_count: 10,
  pending_count: 35,
  success_rate: 90.9,
  status_breakdown: {
    pending: 35,
    complete: 50,
    failed: 5,
    in_progress: 10,
  },
  complexity_distribution: [],
  migration_time_series: [],
  migration_velocity: {
    repos_per_week: 12.5,
  },
};

describe('KPISection', () => {
  it('renders loading skeleton when isLoading is true', () => {
    const { container } = render(<KPISection analytics={undefined} isLoading={true} />);

    // Should show skeleton elements with animate-pulse class
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders loading skeleton when analytics is undefined', () => {
    const { container } = render(<KPISection analytics={undefined} isLoading={false} />);

    // Should show skeleton elements
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders all KPI cards with correct titles', () => {
    render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    expect(screen.getByText('Total Repositories')).toBeInTheDocument();
    expect(screen.getByText('Migration Progress')).toBeInTheDocument();
    expect(screen.getByText('Success Rate')).toBeInTheDocument();
    expect(screen.getByText('Active Migrations')).toBeInTheDocument();
    expect(screen.getByText('Failed Items')).toBeInTheDocument();
    expect(screen.getByText('Migration Velocity')).toBeInTheDocument();
  });

  it('displays correct total repositories count', () => {
    render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    expect(screen.getByText('100')).toBeInTheDocument();
    expect(screen.getByText('50 migrated')).toBeInTheDocument();
  });

  it('calculates and displays correct migration progress', () => {
    render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    // 50/100 = 50%
    expect(screen.getByText('50%')).toBeInTheDocument();
    expect(screen.getByText('50 of 100')).toBeInTheDocument();
  });

  it('displays correct success rate', () => {
    render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    expect(screen.getByText('90.9%')).toBeInTheDocument();
    expect(screen.getByText('of attempted migrations')).toBeInTheDocument();
  });

  it('displays correct active migrations count', () => {
    render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    expect(screen.getByText('10')).toBeInTheDocument();
    expect(screen.getByText('currently running')).toBeInTheDocument();
  });

  it('displays correct failed items count', () => {
    render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('need attention')).toBeInTheDocument();
  });

  it('displays correct migration velocity', () => {
    render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    expect(screen.getByText('12.5')).toBeInTheDocument();
    expect(screen.getByText('repos/week')).toBeInTheDocument();
  });

  it('handles zero repositories correctly', () => {
    const zeroAnalytics: Analytics = {
      ...mockAnalytics,
      total_repositories: 0,
      migrated_count: 0,
    };

    render(<KPISection analytics={zeroAnalytics} isLoading={false} />);

    // Should show 0% progress (not NaN)
    expect(screen.getByText('0%')).toBeInTheDocument();
  });

  it('handles missing optional fields gracefully', () => {
    const partialAnalytics: Analytics = {
      total_repositories: 50,
      migrated_count: 25,
      pending_count: 25,
      success_rate: 85,
      status_breakdown: { pending: 25, complete: 25 },
      complexity_distribution: [],
      migration_time_series: [],
      // missing failed_count, in_progress_count, migration_velocity
    };

    render(<KPISection analytics={partialAnalytics} isLoading={false} />);

    // Should render without errors
    expect(screen.getByText('Total Repositories')).toBeInTheDocument();
    // Velocity should show 0.0
    expect(screen.getByText('0.0')).toBeInTheDocument();
    // Failed items should exist with need attention subtitle
    expect(screen.getByText('need attention')).toBeInTheDocument();
  });

  it('renders six KPI cards', () => {
    const { container } = render(<KPISection analytics={mockAnalytics} isLoading={false} />);

    // KPICard components should be wrapped in the grid
    const gridItems = container.querySelectorAll('.grid > *');
    expect(gridItems.length).toBe(6);
  });

  it('displays repos_per_day when velocity available', () => {
    const analyticsWithVelocity: Analytics = {
      ...mockAnalytics,
      migration_velocity: {
        repos_per_week: 7,
        repos_per_day: 1,
      },
    };

    render(<KPISection analytics={analyticsWithVelocity} isLoading={false} />);

    expect(screen.getByText('7.0')).toBeInTheDocument();
  });

  it('handles 100% completion rate', () => {
    const completeAnalytics: Analytics = {
      ...mockAnalytics,
      total_repositories: 100,
      migrated_count: 100,
      pending_count: 0,
      in_progress_count: 0,
      failed_count: 0,
    };

    render(<KPISection analytics={completeAnalytics} isLoading={false} />);

    expect(screen.getByText('100%')).toBeInTheDocument();
    expect(screen.getByText('100 of 100')).toBeInTheDocument();
  });
});
