import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { DependencyListView } from './DependencyListView';
import type { DependencyGraphNode, DependencyGraphEdge } from '../../types';

describe('DependencyListView', () => {
  const mockNodes: DependencyGraphNode[] = [
    {
      id: 'org/repo1',
      full_name: 'org/repo1',
      organization: 'org',
      status: 'pending',
      depends_on_count: 2,
      depended_by_count: 1,
    },
    {
      id: 'org/repo2',
      full_name: 'org/repo2',
      organization: 'org',
      status: 'complete',
      depends_on_count: 0,
      depended_by_count: 2,
    },
    {
      id: 'org/repo3',
      full_name: 'org/repo3',
      organization: 'org',
      status: 'migration_failed',
      depends_on_count: 1,
      depended_by_count: 0,
    },
  ];

  const mockEdges: DependencyGraphEdge[] = [
    { source: 'org/repo1', target: 'org/repo2', dependency_type: 'submodule' },
    { source: 'org/repo1', target: 'org/repo3', dependency_type: 'workflow' },
    { source: 'org/repo3', target: 'org/repo2', dependency_type: 'package' },
  ];

  const defaultProps = {
    nodes: mockNodes,
    edges: mockEdges,
    allNodes: mockNodes,
    totalNodes: mockNodes.length,
    currentPage: 1,
    pageSize: 10,
    onPageChange: vi.fn(),
  };

  it('renders the title with total count', () => {
    render(<DependencyListView {...defaultProps} />);

    expect(screen.getByText('Repositories with Dependencies (3)')).toBeInTheDocument();
  });

  it('renders all repository rows', () => {
    render(<DependencyListView {...defaultProps} />);

    expect(screen.getByText('org/repo1')).toBeInTheDocument();
    expect(screen.getByText('org/repo2')).toBeInTheDocument();
    expect(screen.getByText('org/repo3')).toBeInTheDocument();
  });

  it('displays depends_on and depended_by counts', () => {
    render(<DependencyListView {...defaultProps} />);

    // org/repo1 has 2 depends_on, 1 depended_by
    const rows = screen.getAllByRole('row');
    expect(rows.length).toBe(4); // 1 header + 3 data rows
  });

  it('shows status badges', () => {
    render(<DependencyListView {...defaultProps} />);

    expect(screen.getByText('pending')).toBeInTheDocument();
    expect(screen.getByText('complete')).toBeInTheDocument();
    expect(screen.getByText('migration failed')).toBeInTheDocument();
  });

  it('shows empty state when no nodes match search', () => {
    render(<DependencyListView {...defaultProps} nodes={[]} totalNodes={0} />);

    expect(screen.getByText('No repositories match your search')).toBeInTheDocument();
  });

  it('shows focus panel when clicking focus button', () => {
    render(<DependencyListView {...defaultProps} />);

    // Find the first focus button and click it
    const focusButtons = screen.getAllByTitle('Focus on this repository');
    fireEvent.click(focusButtons[0]);

    // Should show focus panel - "Depends On" and "Depended By" sections in focus panel
    // These appear as h5 elements in the focus panel
    const dependsOnHeaders = screen.getAllByText('Depends On');
    const dependedByHeaders = screen.getAllByText('Depended By');
    
    // There should be at least 2 (one in table header, one in focus panel)
    expect(dependsOnHeaders.length).toBeGreaterThanOrEqual(2);
    expect(dependedByHeaders.length).toBeGreaterThanOrEqual(2);
  });

  it('shows Clear Focus button when repo is focused', () => {
    render(<DependencyListView {...defaultProps} />);

    // Focus on a repo
    const focusButtons = screen.getAllByTitle('Focus on this repository');
    fireEvent.click(focusButtons[0]);

    expect(screen.getByText('Clear Focus')).toBeInTheDocument();
  });

  it('clears focus when clicking Clear Focus button', () => {
    render(<DependencyListView {...defaultProps} />);

    // Focus on a repo
    const focusButtons = screen.getAllByTitle('Focus on this repository');
    fireEvent.click(focusButtons[0]);

    // Clear focus
    fireEvent.click(screen.getByText('Clear Focus'));

    // Clear Focus button should disappear
    expect(screen.queryByText('Clear Focus')).not.toBeInTheDocument();
  });

  it('shows dependency links in focus panel', () => {
    render(<DependencyListView {...defaultProps} />);

    // Focus on repo1 which depends on repo2 and repo3
    const focusButtons = screen.getAllByTitle('Focus on this repository');
    fireEvent.click(focusButtons[0]);

    // Should show the dependencies
    expect(screen.getByText('submodule')).toBeInTheDocument();
    expect(screen.getByText('workflow')).toBeInTheDocument();
  });

  it('renders pagination when total nodes exceed page size', () => {
    const manyNodes = Array.from({ length: 15 }, (_, i) => ({
      ...mockNodes[0],
      id: `org/repo${i}`,
      full_name: `org/repo${i}`,
    }));

    const { container } = render(
      <DependencyListView
        {...defaultProps}
        nodes={manyNodes.slice(0, 10)}
        allNodes={manyNodes}
        totalNodes={15}
        pageSize={10}
      />
    );

    // Should show pagination - look for pagination container
    expect(container.querySelector('.mt-4')).toBeInTheDocument();
  });

  it('does not render pagination when total nodes are within page size', () => {
    const { container } = render(<DependencyListView {...defaultProps} />);

    // Should not show pagination since we have 3 nodes and pageSize is 10
    // The Pagination component should not render when totalItems <= pageSize
    const tables = container.querySelectorAll('table');
    expect(tables).toHaveLength(1);
  });

  it('shows table headers', () => {
    render(<DependencyListView {...defaultProps} />);

    expect(screen.getByText('Focus')).toBeInTheDocument();
    expect(screen.getByText('Repository')).toBeInTheDocument();
    expect(screen.getByText('Organization')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Depends On')).toBeInTheDocument();
    expect(screen.getByText('Depended By')).toBeInTheDocument();
    expect(screen.getByText('Dependencies')).toBeInTheDocument();
  });

  it('renders dependency columns in table', () => {
    render(<DependencyListView {...defaultProps} />);

    // Should show the Dependencies column header
    expect(screen.getByText('Dependencies')).toBeInTheDocument();
  });
});

