import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { ActiveFilterPills } from './ActiveFilterPills';
import type { RepositoryFilters } from '../../types';

describe('ActiveFilterPills', () => {
  const mockOnRemoveFilter = vi.fn();
  const mockOnClearAll = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should return null when no filters are active', () => {
    render(
      <ActiveFilterPills
        filters={{}}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    // Should not render the "Active filters:" label
    expect(screen.queryByText('Active filters:')).not.toBeInTheDocument();
  });

  it('should render organization filter pill', () => {
    const filters: RepositoryFilters = { organization: 'my-org' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Organization:')).toBeInTheDocument();
    expect(screen.getByText('my-org')).toBeInTheDocument();
  });

  it('should show count when multiple organizations selected', () => {
    const filters: RepositoryFilters = { organization: ['org1', 'org2', 'org3'] };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('3 selected')).toBeInTheDocument();
  });

  it('should render search filter pill', () => {
    const filters: RepositoryFilters = { search: 'test-repo' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Search:')).toBeInTheDocument();
    expect(screen.getByText('test-repo')).toBeInTheDocument();
  });

  it('should render size category filter pill', () => {
    const filters: RepositoryFilters = { size_category: 'small' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Size:')).toBeInTheDocument();
    expect(screen.getByText('small')).toBeInTheDocument();
  });

  it('should render complexity filter pill', () => {
    const filters: RepositoryFilters = { complexity: 'simple' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Complexity:')).toBeInTheDocument();
    expect(screen.getByText('simple')).toBeInTheDocument();
  });

  it('should render size range filter pill', () => {
    const filters: RepositoryFilters = {
      min_size: 100 * 1024 * 1024, // 100MB
      max_size: 500 * 1024 * 1024, // 500MB
    };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Size Range:')).toBeInTheDocument();
    expect(screen.getByText('100-500 MB')).toBeInTheDocument();
  });

  it('should show infinity for max when only min is set', () => {
    const filters: RepositoryFilters = { min_size: 100 * 1024 * 1024 };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('100-∞ MB')).toBeInTheDocument();
  });

  it('should render feature flag filter pills', () => {
    const filters: RepositoryFilters = { has_lfs: true, has_actions: true };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('LFS')).toBeInTheDocument();
    expect(screen.getByText('Actions')).toBeInTheDocument();
  });

  it('should render sort filter pill when not default', () => {
    const filters: RepositoryFilters = { sort_by: 'size' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Sort:')).toBeInTheDocument();
    expect(screen.getByText('size')).toBeInTheDocument();
  });

  it('should not render sort pill when sort is name (default)', () => {
    const filters: RepositoryFilters = { sort_by: 'name' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    // Should not render the "Active filters:" label when no active filters
    expect(screen.queryByText('Active filters:')).not.toBeInTheDocument();
  });

  it('should call onRemoveFilter when a pill is clicked', () => {
    const filters: RepositoryFilters = { search: 'test' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    const pill = screen.getByText('test');
    fireEvent.click(pill.closest('button')!);

    expect(mockOnRemoveFilter).toHaveBeenCalledWith('search');
  });

  it('should show Clear all button when multiple filters are active', () => {
    const filters: RepositoryFilters = { search: 'test', has_lfs: true };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Clear all')).toBeInTheDocument();
  });

  it('should not show Clear all button when only one filter is active', () => {
    const filters: RepositoryFilters = { search: 'test' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.queryByText('Clear all')).not.toBeInTheDocument();
  });

  it('should call onClearAll when Clear all button is clicked', () => {
    const filters: RepositoryFilters = { search: 'test', has_lfs: true };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    const clearAllButton = screen.getByText('Clear all');
    fireEvent.click(clearAllButton);

    expect(mockOnClearAll).toHaveBeenCalledTimes(1);
  });

  it('should render "Active filters:" label', () => {
    const filters: RepositoryFilters = { search: 'test' };

    render(
      <ActiveFilterPills
        filters={filters}
        onRemoveFilter={mockOnRemoveFilter}
        onClearAll={mockOnClearAll}
      />
    );

    expect(screen.getByText('Active filters:')).toBeInTheDocument();
  });
});

