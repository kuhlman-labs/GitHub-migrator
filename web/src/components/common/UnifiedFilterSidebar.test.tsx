import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '../../__tests__/test-utils';
import { UnifiedFilterSidebar } from './UnifiedFilterSidebar';
import type { RepositoryFilters } from '../../types';
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

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useConfig: vi.fn(() => ({
    data: { source_type: 'github' },
    isLoading: false,
  })),
  useOrganizations: vi.fn(() => ({
    data: [
      { organization: 'org1' },
      { organization: 'org2' },
      { organization: 'org3' },
    ],
    isLoading: false,
  })),
}));

// Mock the API
vi.mock('../../services/api', () => ({
  api: {
    listTeams: vi.fn().mockResolvedValue([
      { full_slug: 'org1/team1' },
      { full_slug: 'org1/team2' },
    ]),
    listADOProjects: vi.fn().mockResolvedValue([
      { name: 'project1' },
      { name: 'project2' },
    ]),
  },
}));

describe('UnifiedFilterSidebar', () => {
  const mockFilters: RepositoryFilters = {};
  const mockOnChange = vi.fn();
  const mockOnToggleCollapse = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Collapsed state', () => {
    it('should render collapsed view with filter icon', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={true}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      const expandButton = screen.getByTitle('Expand filters');
      expect(expandButton).toBeInTheDocument();
    });

    it('should call onToggleCollapse when expand button is clicked', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={true}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      const expandButton = screen.getByTitle('Expand filters');
      fireEvent.click(expandButton);
      expect(mockOnToggleCollapse).toHaveBeenCalledTimes(1);
    });

    it('should show filter count badge when filters are active', () => {
      const filtersWithSearch: RepositoryFilters = {
        search: 'test',
        status: ['pending'],
      };

      render(
        <UnifiedFilterSidebar
          filters={filtersWithSearch}
          onChange={mockOnChange}
          isCollapsed={true}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Find the badge with the filter count
      expect(screen.getByText('2')).toBeInTheDocument();
    });
  });

  describe('Expanded state', () => {
    it('should render expanded view with Filters heading', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      expect(screen.getByText('Filters')).toBeInTheDocument();
    });

    it('should call onToggleCollapse when collapse button is clicked', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      const collapseButton = screen.getByTitle('Collapse filters');
      fireEvent.click(collapseButton);
      expect(mockOnToggleCollapse).toHaveBeenCalledTimes(1);
    });

    it('should render search input when showSearch is true', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showSearch={true}
        />
      );

      expect(screen.getByPlaceholderText('Repository name...')).toBeInTheDocument();
    });

    it('should not render search input when showSearch is false', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showSearch={false}
        />
      );

      expect(screen.queryByPlaceholderText('Repository name...')).not.toBeInTheDocument();
    });
  });

  describe('Search functionality', () => {
    it('should call onChange when search input changes', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showSearch={true}
        />
      );

      const searchInput = screen.getByPlaceholderText('Repository name...');
      fireEvent.change(searchInput, { target: { value: 'test-repo' } });

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ search: 'test-repo' })
      );
    });

    it('should clear search when input is emptied', () => {
      render(
        <UnifiedFilterSidebar
          filters={{ search: 'existing' }}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showSearch={true}
        />
      );

      const searchInput = screen.getByPlaceholderText('Repository name...');
      fireEvent.change(searchInput, { target: { value: '' } });

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ search: undefined })
      );
    });
  });

  describe('Status filters', () => {
    it('should render status section when showStatus is true', async () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showStatus={true}
        />
      );

      await waitFor(() => {
        expect(screen.getByText('Status')).toBeInTheDocument();
      });
      
      // Use getAllByText since 'Pending' appears as both group header and status label
      expect(screen.getAllByText('Pending').length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText('In Progress')).toBeInTheDocument();
      // 'Complete' also appears as group header
      expect(screen.getAllByText('Complete').length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText('Failed')).toBeInTheDocument();
    });

    it('should not render status section when showStatus is false', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showStatus={false}
        />
      );

      expect(screen.queryByText('Status')).not.toBeInTheDocument();
    });

    it('should toggle status filter when checkbox is clicked', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showStatus={true}
        />
      );

      // Find status checkboxes - they're inside labels
      const pendingLabel = screen.getByText('Pending', { selector: 'span' });
      const checkbox = pendingLabel.previousElementSibling as HTMLInputElement;
      
      fireEvent.click(checkbox);

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ status: ['pending'] })
      );
    });

    it('should show checked status when filter is active', () => {
      render(
        <UnifiedFilterSidebar
          filters={{ status: ['pending', 'complete'] }}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          showStatus={true}
        />
      );

      // Find the checkbox for 'pending' status
      const pendingLabel = screen.getByText('Pending', { selector: 'span' });
      const pendingCheckbox = pendingLabel.previousElementSibling as HTMLInputElement;
      
      expect(pendingCheckbox.checked).toBe(true);
    });
  });

  describe('Complexity filters', () => {
    it('should render complexity section', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      expect(screen.getByText('Complexity')).toBeInTheDocument();
      expect(screen.getByText('simple')).toBeInTheDocument();
      expect(screen.getByText('medium')).toBeInTheDocument();
      expect(screen.getByText('complex')).toBeInTheDocument();
      expect(screen.getByText('very complex')).toBeInTheDocument();
    });

    it('should toggle complexity filter when checkbox is clicked', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      const simpleLabel = screen.getByText('simple');
      const checkbox = simpleLabel.previousElementSibling as HTMLInputElement;
      
      fireEvent.click(checkbox);

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ complexity: ['simple'] })
      );
    });

    it('should add to existing complexity filters', () => {
      render(
        <UnifiedFilterSidebar
          filters={{ complexity: ['simple'] }}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      const mediumLabel = screen.getByText('medium');
      const checkbox = mediumLabel.previousElementSibling as HTMLInputElement;
      
      fireEvent.click(checkbox);

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ complexity: ['simple', 'medium'] })
      );
    });

    it('should remove from complexity filters when unchecked', () => {
      render(
        <UnifiedFilterSidebar
          filters={{ complexity: ['simple', 'medium'] }}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      const simpleLabel = screen.getByText('simple');
      const checkbox = simpleLabel.previousElementSibling as HTMLInputElement;
      
      fireEvent.click(checkbox);

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ complexity: ['medium'] })
      );
    });
  });

  describe('Size filters', () => {
    it('should render size section', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click on Size to expand
      const sizeButton = screen.getByText('Size');
      fireEvent.click(sizeButton);

      expect(screen.getByText('Category')).toBeInTheDocument();
      expect(screen.getByText('Small (<100MB)')).toBeInTheDocument();
      expect(screen.getByText('Medium (100MB-1GB)')).toBeInTheDocument();
      expect(screen.getByText('Large (1GB-5GB)')).toBeInTheDocument();
      expect(screen.getByText('Very Large (>5GB)')).toBeInTheDocument();
    });

    it('should handle size range min input', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click on Size to expand
      const sizeButton = screen.getByText('Size');
      fireEvent.click(sizeButton);

      const minInput = screen.getByPlaceholderText('Min');
      fireEvent.change(minInput, { target: { value: '100' } });

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ min_size: 100 * 1024 * 1024 })
      );
    });

    it('should handle size range max input', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click on Size to expand
      const sizeButton = screen.getByText('Size');
      fireEvent.click(sizeButton);

      const maxInput = screen.getByPlaceholderText('Max');
      fireEvent.change(maxInput, { target: { value: '500' } });

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ max_size: 500 * 1024 * 1024 })
      );
    });

    it('should clear size when input is emptied', () => {
      render(
        <UnifiedFilterSidebar
          filters={{ min_size: 100 * 1024 * 1024 }}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click on Size to expand
      const sizeButton = screen.getByText('Size');
      fireEvent.click(sizeButton);

      const minInput = screen.getByPlaceholderText('Min');
      fireEvent.change(minInput, { target: { value: '' } });

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ min_size: undefined })
      );
    });
  });

  describe('Visibility filter', () => {
    it('should render visibility dropdown', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click to expand visibility section
      const visibilityButton = screen.getByText('Visibility');
      fireEvent.click(visibilityButton);

      // Check for options in the select
      const select = screen.getByRole('combobox');
      expect(select).toBeInTheDocument();
    });

    it('should call onChange when visibility is changed', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click to expand visibility section
      const visibilityButton = screen.getByText('Visibility');
      fireEvent.click(visibilityButton);

      const select = screen.getByRole('combobox');
      fireEvent.change(select, { target: { value: 'public' } });

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ visibility: 'public' })
      );
    });
  });

  describe('Sort By filter', () => {
    it('should render sort by dropdown', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click to expand sort section
      const sortButton = screen.getByText('Sort By');
      fireEvent.click(sortButton);

      // Should have combobox for sort
      const selects = screen.getAllByRole('combobox');
      expect(selects.length).toBeGreaterThanOrEqual(1);
    });

    it('should call onChange when sort is changed', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click to expand sort section
      const sortButton = screen.getByText('Sort By');
      fireEvent.click(sortButton);

      // Get the sort select (second combobox after visibility)
      const selects = screen.getAllByRole('combobox');
      const sortSelect = selects[selects.length - 1]; // Last combobox should be sort
      fireEvent.change(sortSelect, { target: { value: 'size' } });

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ sort_by: 'size' })
      );
    });
  });

  describe('Features section', () => {
    it('should render GitHub features for github source type', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click to expand features section
      const featuresButton = screen.getByText('Features');
      fireEvent.click(featuresButton);

      expect(screen.getByText('LFS')).toBeInTheDocument();
      expect(screen.getByText('Submodules')).toBeInTheDocument();
      expect(screen.getByText('Actions')).toBeInTheDocument();
      expect(screen.getByText('Wiki')).toBeInTheDocument();
    });

    it('should toggle feature filter when clicked', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Click to expand features section
      const featuresButton = screen.getByText('Features');
      fireEvent.click(featuresButton);

      const lfsLabel = screen.getByText('LFS');
      const checkbox = lfsLabel.previousElementSibling as HTMLInputElement;
      
      fireEvent.click(checkbox);

      expect(mockOnChange).toHaveBeenCalledWith(
        expect.objectContaining({ has_lfs: true })
      );
    });
  });

  describe('Organization section', () => {
    it('should render organization section when hideOrganization is false', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          hideOrganization={false}
        />
      );

      expect(screen.getByText('Organization')).toBeInTheDocument();
    });

    it('should not render organization section when hideOrganization is true', () => {
      render(
        <UnifiedFilterSidebar
          filters={mockFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
          hideOrganization={true}
        />
      );

      expect(screen.queryByText('Organization')).not.toBeInTheDocument();
    });
  });

  describe('Filter count', () => {
    it('should correctly count active filters', () => {
      const activeFilters: RepositoryFilters = {
        search: 'test',
        status: ['pending'],
        complexity: ['simple'],
        has_lfs: true,
        visibility: 'public',
      };

      render(
        <UnifiedFilterSidebar
          filters={activeFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Should show 5 in the badge
      expect(screen.getByText('5')).toBeInTheDocument();
    });

    it('should not show badge when no filters are active', () => {
      render(
        <UnifiedFilterSidebar
          filters={{}}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Filter heading should exist but no count badge
      expect(screen.getByText('Filters')).toBeInTheDocument();
      expect(screen.queryByText('0')).not.toBeInTheDocument();
    });

    it('should count multiple filter categories correctly', () => {
      const manyFilters: RepositoryFilters = {
        search: 'test',
        status: ['pending', 'complete'],
        organization: ['org1'],
        complexity: ['simple'],
        has_lfs: true,
        has_actions: true,
        has_wiki: true,
        visibility: 'public',
        sort_by: 'size',
      };

      render(
        <UnifiedFilterSidebar
          filters={manyFilters}
          onChange={mockOnChange}
          isCollapsed={false}
          onToggleCollapse={mockOnToggleCollapse}
        />
      );

      // Count: search(1) + status(1) + org(1) + complexity(1) + has_lfs(1) + has_actions(1) + has_wiki(1) + visibility(1) + sort_by(1) = 9
      expect(screen.getByText('9')).toBeInTheDocument();
    });
  });
});

describe('UnifiedFilterSidebar with Azure DevOps', () => {
  const mockFilters: RepositoryFilters = {};
  const mockOnChange = vi.fn();
  const mockOnToggleCollapse = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    // Mock SourceContext with ADO sources
    (SourceContextModule.useSourceContext as ReturnType<typeof vi.fn>).mockReturnValue({
      sources: [{ id: 1, name: 'ADO Source', type: 'azuredevops' }],
      activeSourceFilter: 'all',
      setActiveSourceFilter: vi.fn(),
      activeSource: null,
      isLoading: false,
      error: null,
      refetchSources: vi.fn(),
    });
  });

  it('should render ADO-specific features', async () => {
    render(
      <UnifiedFilterSidebar
        filters={mockFilters}
        onChange={mockOnChange}
        isCollapsed={false}
        onToggleCollapse={mockOnToggleCollapse}
      />
    );

    // Click to expand features section
    const featuresButton = screen.getByText('Features');
    fireEvent.click(featuresButton);

    await waitFor(() => {
      expect(screen.getByText('Git (vs TFVC)')).toBeInTheDocument();
    });
    expect(screen.getByText('Azure Boards')).toBeInTheDocument();
    expect(screen.getByText('Azure Pipelines')).toBeInTheDocument();
  });

  it('should render project section for Azure DevOps', async () => {
    render(
      <UnifiedFilterSidebar
        filters={mockFilters}
        onChange={mockOnChange}
        isCollapsed={false}
        onToggleCollapse={mockOnToggleCollapse}
        hideProject={false}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('Project')).toBeInTheDocument();
    });
  });

  it('should not render project section when hideProject is true', async () => {
    render(
      <UnifiedFilterSidebar
        filters={mockFilters}
        onChange={mockOnChange}
        isCollapsed={false}
        onToggleCollapse={mockOnToggleCollapse}
        hideProject={true}
      />
    );

    await waitFor(() => {
      expect(screen.queryByText('Project')).not.toBeInTheDocument();
    });
  });
});

