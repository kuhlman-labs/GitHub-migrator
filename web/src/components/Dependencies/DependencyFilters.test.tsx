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
});

