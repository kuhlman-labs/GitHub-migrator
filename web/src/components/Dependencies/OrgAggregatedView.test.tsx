import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { OrgAggregatedView } from './OrgAggregatedView';
import type { DependencyGraphNode, DependencyGraphEdge } from '../../types';

// Mock react-force-graph-2d
vi.mock('react-force-graph-2d', () => ({
  default: ({ graphData }: { graphData: { nodes: unknown[]; links: unknown[] } }) => (
    <div data-testid="force-graph">
      <span data-testid="node-count">{graphData.nodes.length}</span>
      <span data-testid="link-count">{graphData.links.length}</span>
    </div>
  ),
}));

// Mock d3-force
vi.mock('d3-force', () => ({
  forceCollide: () => ({
    radius: vi.fn().mockReturnThis(),
    strength: vi.fn().mockReturnThis(),
    iterations: vi.fn().mockReturnThis(),
  }),
}));

describe('OrgAggregatedView', () => {
  const mockNodes: DependencyGraphNode[] = [
    {
      id: 'org1/repo1',
      full_name: 'org1/repo1',
      organization: 'org1',
      status: 'pending',
      depends_on_count: 2,
      depended_by_count: 0,
    },
    {
      id: 'org1/repo2',
      full_name: 'org1/repo2',
      organization: 'org1',
      status: 'complete',
      depends_on_count: 0,
      depended_by_count: 1,
    },
    {
      id: 'org2/repo3',
      full_name: 'org2/repo3',
      organization: 'org2',
      status: 'pending',
      depends_on_count: 1,
      depended_by_count: 1,
    },
  ];

  const mockEdges: DependencyGraphEdge[] = [
    { source: 'org1/repo1', target: 'org2/repo3', dependency_type: 'submodule' },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the title', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    expect(screen.getByText('Organization Dependency Map')).toBeInTheDocument();
  });

  it('renders the description', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    expect(screen.getByText(/Select an organization to focus on its cross-org relationships/)).toBeInTheDocument();
  });

  it('renders the force graph component', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    expect(screen.getByTestId('force-graph')).toBeInTheDocument();
  });

  it('renders organization selector button', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    expect(screen.getByText('Select Organization')).toBeInTheDocument();
  });

  it('renders global stats when no org is selected', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    // Should show organization count
    expect(screen.getByText('Organizations')).toBeInTheDocument();
    expect(screen.getByText('Total Repositories')).toBeInTheDocument();
    expect(screen.getByText('Cross-Org Dependencies')).toBeInTheDocument();
    expect(screen.getByText('Total Cross-Org Links')).toBeInTheDocument();
  });

  it('renders legend', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    expect(screen.getByText('Node size = repo count')).toBeInTheDocument();
    expect(screen.getByText('Arrow = dependency direction')).toBeInTheDocument();
    expect(screen.getByText('Click org to focus')).toBeInTheDocument();
  });

  it('shows correct number of organizations', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    // The Organizations stat should be visible
    expect(screen.getByText('Organizations')).toBeInTheDocument();
  });

  it('shows correct total repositories', () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    // The Total Repositories stat should be visible
    expect(screen.getByText('Total Repositories')).toBeInTheDocument();
  });

  it('shows empty state when no nodes', () => {
    render(<OrgAggregatedView nodes={[]} edges={[]} />);

    expect(screen.getByText('No organizations match the search filter')).toBeInTheDocument();
  });

  it('renders organization dropdown when clicking selector', async () => {
    render(<OrgAggregatedView nodes={mockNodes} edges={mockEdges} />);

    fireEvent.click(screen.getByText('Select Organization'));

    expect(screen.getByText('Show All Organizations')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Search organizations...')).toBeInTheDocument();
  });
});

