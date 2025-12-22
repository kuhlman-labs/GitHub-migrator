import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { ComplexityChart } from './ComplexityChart';
import type { ComplexityDistribution } from '../../types';

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// Mock recharts to avoid rendering issues in tests
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div data-testid="responsive-container">{children}</div>,
  BarChart: ({ children }: { children: React.ReactNode }) => <div data-testid="bar-chart">{children}</div>,
  Bar: () => <div data-testid="bar" />,
  XAxis: () => <div data-testid="x-axis" />,
  YAxis: () => <div data-testid="y-axis" />,
  CartesianGrid: () => <div data-testid="cartesian-grid" />,
  Tooltip: () => <div data-testid="tooltip" />,
  Cell: () => <div data-testid="cell" />,
}));

describe('ComplexityChart', () => {
  const mockData: ComplexityDistribution[] = [
    { category: 'simple', count: 50 },
    { category: 'medium', count: 30 },
    { category: 'complex', count: 15 },
    { category: 'very_complex', count: 5 },
  ];

  beforeEach(() => {
    mockNavigate.mockClear();
  });

  it('renders the chart title', () => {
    render(<ComplexityChart data={mockData} />);

    expect(screen.getByText('Repository Complexity Distribution')).toBeInTheDocument();
  });

  it('renders description for GitHub source', () => {
    render(<ComplexityChart data={mockData} source="github" />);

    expect(screen.getByText(/GitHub migration complexity factors/)).toBeInTheDocument();
  });

  it('renders description for Azure DevOps source', () => {
    render(<ComplexityChart data={mockData} source="azuredevops" />);

    expect(screen.getByText(/ADO → GitHub migration complexity factors/)).toBeInTheDocument();
  });

  it('renders description for all sources', () => {
    render(<ComplexityChart data={mockData} source="all" />);

    expect(screen.getByText(/Scoring varies by source/)).toBeInTheDocument();
  });

  it('renders the bar chart', () => {
    render(<ComplexityChart data={mockData} />);

    expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
  });

  it('renders legend items for all complexity levels', () => {
    render(<ComplexityChart data={mockData} />);

    expect(screen.getByText(/Simple: 50/)).toBeInTheDocument();
    expect(screen.getByText(/Medium: 30/)).toBeInTheDocument();
    expect(screen.getByText(/Complex: 15/)).toBeInTheDocument();
    expect(screen.getByText(/Very Complex: 5/)).toBeInTheDocument();
  });

  it('handles empty data', () => {
    render(<ComplexityChart data={[]} />);

    // Should still render the chart component
    expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
  });

  it('handles partial data', () => {
    const partialData: ComplexityDistribution[] = [
      { category: 'simple', count: 25 },
      { category: 'complex', count: 10 },
    ];

    render(<ComplexityChart data={partialData} />);

    // Should render the chart component
    expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
  });

  it('renders legend buttons', () => {
    const { container } = render(<ComplexityChart data={mockData} />);

    // Should have buttons for each complexity level
    const buttons = container.querySelectorAll('button');
    expect(buttons.length).toBeGreaterThan(0);
  });

  it('renders complexity info modal trigger', () => {
    render(<ComplexityChart data={mockData} />);

    // The ComplexityInfoModal trigger button should be rendered
    expect(screen.getByText('How is complexity calculated?')).toBeInTheDocument();
  });
});

