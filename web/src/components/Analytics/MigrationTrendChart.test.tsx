import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { MigrationTrendChart } from './MigrationTrendChart';
import type { MigrationTimeSeriesPoint } from '../../types';

// Mock recharts to avoid rendering issues in tests
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div data-testid="responsive-container">{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div data-testid="area-chart">{children}</div>,
  Area: () => <div data-testid="area" />,
  XAxis: () => <div data-testid="x-axis" />,
  YAxis: () => <div data-testid="y-axis" />,
  CartesianGrid: () => <div data-testid="cartesian-grid" />,
  Tooltip: () => <div data-testid="tooltip" />,
}));

describe('MigrationTrendChart', () => {
  const mockData: MigrationTimeSeriesPoint[] = [
    { date: '2024-01-01', count: 5 },
    { date: '2024-01-02', count: 8 },
    { date: '2024-01-03', count: 3 },
    { date: '2024-01-04', count: 12 },
    { date: '2024-01-05', count: 7 },
  ];

  it('renders the chart title', () => {
    render(<MigrationTrendChart data={mockData} />);

    expect(screen.getByText('Migration Trend (Last 30 Days)')).toBeInTheDocument();
  });

  it('renders description', () => {
    render(<MigrationTrendChart data={mockData} />);

    expect(screen.getByText(/Daily migration activity showing velocity trends/)).toBeInTheDocument();
  });

  it('renders the area chart', () => {
    render(<MigrationTrendChart data={mockData} />);

    expect(screen.getByTestId('area-chart')).toBeInTheDocument();
    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
  });

  it('shows empty state when no data', () => {
    render(<MigrationTrendChart data={[]} />);

    expect(screen.getByText('No migration data available for the selected period')).toBeInTheDocument();
    expect(screen.queryByTestId('area-chart')).not.toBeInTheDocument();
  });

  it('shows empty state when data is undefined', () => {
    render(<MigrationTrendChart data={undefined as unknown as MigrationTimeSeriesPoint[]} />);

    expect(screen.getByText('No migration data available for the selected period')).toBeInTheDocument();
  });

  it('still renders title in empty state', () => {
    render(<MigrationTrendChart data={[]} />);

    expect(screen.getByText('Migration Trend (Last 30 Days)')).toBeInTheDocument();
    expect(screen.getByText(/Daily migration activity/)).toBeInTheDocument();
  });
});

