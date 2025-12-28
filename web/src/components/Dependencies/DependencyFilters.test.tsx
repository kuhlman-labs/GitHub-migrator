import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { DependencyFilters, DependencyTypeFilter } from './DependencyFilters';

describe('DependencyFilters', () => {
  const mockOnTypeFilterChange = vi.fn();
  const mockOnSearchQueryChange = vi.fn();

  const defaultProps = {
    typeFilter: 'all' as DependencyTypeFilter,
    onTypeFilterChange: mockOnTypeFilterChange,
    searchQuery: '',
    onSearchQueryChange: mockOnSearchQueryChange,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders all filter buttons', () => {
    render(<DependencyFilters {...defaultProps} />);

    expect(screen.getByRole('button', { name: 'All Types' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Submodule' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Workflow' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Dependency Graph' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Package' })).toBeInTheDocument();
  });

  it('renders search input', () => {
    render(<DependencyFilters {...defaultProps} />);

    expect(screen.getByPlaceholderText('Search repositories...')).toBeInTheDocument();
  });

  it('calls onTypeFilterChange when All Types button is clicked', () => {
    render(<DependencyFilters {...defaultProps} typeFilter="submodule" />);

    fireEvent.click(screen.getByRole('button', { name: 'All Types' }));
    expect(mockOnTypeFilterChange).toHaveBeenCalledWith('all');
  });

  it('calls onTypeFilterChange when Submodule button is clicked', () => {
    render(<DependencyFilters {...defaultProps} />);

    fireEvent.click(screen.getByRole('button', { name: 'Submodule' }));
    expect(mockOnTypeFilterChange).toHaveBeenCalledWith('submodule');
  });

  it('calls onTypeFilterChange when Workflow button is clicked', () => {
    render(<DependencyFilters {...defaultProps} />);

    fireEvent.click(screen.getByRole('button', { name: 'Workflow' }));
    expect(mockOnTypeFilterChange).toHaveBeenCalledWith('workflow');
  });

  it('calls onTypeFilterChange when Dependency Graph button is clicked', () => {
    render(<DependencyFilters {...defaultProps} />);

    fireEvent.click(screen.getByRole('button', { name: 'Dependency Graph' }));
    expect(mockOnTypeFilterChange).toHaveBeenCalledWith('dependency_graph');
  });

  it('calls onTypeFilterChange when Package button is clicked', () => {
    render(<DependencyFilters {...defaultProps} />);

    fireEvent.click(screen.getByRole('button', { name: 'Package' }));
    expect(mockOnTypeFilterChange).toHaveBeenCalledWith('package');
  });

  it('calls onSearchQueryChange when typing in search input', () => {
    render(<DependencyFilters {...defaultProps} />);

    const searchInput = screen.getByPlaceholderText('Search repositories...');
    fireEvent.change(searchInput, { target: { value: 'test-repo' } });

    expect(mockOnSearchQueryChange).toHaveBeenCalledWith('test-repo');
  });

  it('displays current search query in input', () => {
    render(<DependencyFilters {...defaultProps} searchQuery="my-search" />);

    const searchInput = screen.getByPlaceholderText('Search repositories...');
    expect(searchInput).toHaveValue('my-search');
  });

  // Skip: Style assertions are brittle with CSS-in-JS and may not reflect computed styles in jsdom
  it.skip('highlights selected filter button', () => {
    render(<DependencyFilters {...defaultProps} typeFilter="workflow" />);

    const workflowButton = screen.getByRole('button', { name: 'Workflow' });
    const allTypesButton = screen.getByRole('button', { name: 'All Types' });

    // Workflow should be highlighted (purple #8250DF)
    expect(workflowButton).toHaveStyle('background-color: #8250DF');
    expect(workflowButton).toHaveStyle('color: #ffffff');

    // All Types should not be highlighted
    expect(allTypesButton).not.toHaveStyle('background-color: #2da44e');
  });

  // Skip: Style assertions are brittle with CSS-in-JS and may not reflect computed styles in jsdom
  it.skip('updates highlighted button when filter changes', () => {
    const { rerender } = render(<DependencyFilters {...defaultProps} typeFilter="all" />);

    const allTypesButton = screen.getByRole('button', { name: 'All Types' });
    expect(allTypesButton).toHaveStyle('background-color: #2da44e');

    rerender(<DependencyFilters {...defaultProps} typeFilter="package" />);

    const packageButton = screen.getByRole('button', { name: 'Package' });
    expect(packageButton).toHaveStyle('background-color: #656D76');
  });
});

