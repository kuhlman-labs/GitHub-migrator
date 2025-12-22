import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { FilterBar } from './FilterBar';
import * as useQueriesModule from '../../hooks/useQueries';

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useOrganizations: vi.fn(),
  useBatches: vi.fn(),
  useProjects: vi.fn(),
}));

describe('FilterBar', () => {
  const mockOrganizations = [
    { organization: 'org1', total_repos: 50 },
    { organization: 'org2', total_repos: 30 },
    { organization: 'test-org', total_repos: 20 },
  ];

  const mockBatches = [
    { id: 1, name: 'Batch 1', repository_count: 10 },
    { id: 2, name: 'Batch 2', repository_count: 5 },
  ];

  const mockProjects = [
    { project: 'project1', total_repos: 25 },
    { project: 'project2', total_repos: 15 },
  ];

  const defaultProps = {
    selectedOrganization: '',
    selectedProject: '',
    selectedBatch: '',
    onOrganizationChange: vi.fn(),
    onProjectChange: vi.fn(),
    onBatchChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    
    (useQueriesModule.useOrganizations as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockOrganizations,
    });
    
    (useQueriesModule.useBatches as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockBatches,
    });
    
    (useQueriesModule.useProjects as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockProjects,
    });
  });

  it('should render organization filter', () => {
    render(<FilterBar {...defaultProps} />);
    
    expect(screen.getByText('Organization')).toBeInTheDocument();
    expect(screen.getByText('All Organizations')).toBeInTheDocument();
  });

  it('should render batch filter', () => {
    render(<FilterBar {...defaultProps} />);
    
    expect(screen.getByText('Batch')).toBeInTheDocument();
    expect(screen.getByText('All Batches')).toBeInTheDocument();
  });

  it('should show project filter for azuredevops source type', () => {
    render(<FilterBar {...defaultProps} sourceType="azuredevops" />);
    
    expect(screen.getByText('Project')).toBeInTheDocument();
    expect(screen.getByText('All Projects')).toBeInTheDocument();
  });

  it('should not show project filter for github source type', () => {
    render(<FilterBar {...defaultProps} sourceType="github" />);
    
    expect(screen.queryByText('Project')).not.toBeInTheDocument();
  });

  it('should display selected organization', () => {
    render(<FilterBar {...defaultProps} selectedOrganization="org1" />);
    
    expect(screen.getByText('org1 (50)')).toBeInTheDocument();
  });

  it('should display selected batch', () => {
    render(<FilterBar {...defaultProps} selectedBatch="1" />);
    
    expect(screen.getByText('Batch 1 (10)')).toBeInTheDocument();
  });

  it('should display selected project for azuredevops', () => {
    render(<FilterBar {...defaultProps} sourceType="azuredevops" selectedProject="project1" />);
    
    expect(screen.getByText('project1 (25)')).toBeInTheDocument();
  });

  it('should open organization dropdown when clicked', async () => {
    const user = userEvent.setup();
    render(<FilterBar {...defaultProps} />);
    
    await user.click(screen.getByText('All Organizations'));
    
    await waitFor(() => {
      expect(screen.getByPlaceholderText('Search organizations...')).toBeInTheDocument();
    });
  });

  it('should open batch dropdown when clicked', async () => {
    const user = userEvent.setup();
    render(<FilterBar {...defaultProps} />);
    
    await user.click(screen.getByText('All Batches'));
    
    await waitFor(() => {
      expect(screen.getByPlaceholderText('Search batches...')).toBeInTheDocument();
    });
  });

  it('should call onOrganizationChange when organization is selected', async () => {
    const user = userEvent.setup();
    const onOrganizationChange = vi.fn();
    render(<FilterBar {...defaultProps} onOrganizationChange={onOrganizationChange} />);
    
    await user.click(screen.getByText('All Organizations'));
    
    await waitFor(() => {
      expect(screen.getByText('org1')).toBeInTheDocument();
    });
    
    await user.click(screen.getByText('org1'));
    
    expect(onOrganizationChange).toHaveBeenCalledWith('org1');
  });

  it('should call onBatchChange when batch is selected', async () => {
    const user = userEvent.setup();
    const onBatchChange = vi.fn();
    render(<FilterBar {...defaultProps} onBatchChange={onBatchChange} />);
    
    await user.click(screen.getByText('All Batches'));
    
    await waitFor(() => {
      expect(screen.getByText('Batch 1')).toBeInTheDocument();
    });
    
    await user.click(screen.getByText('Batch 1'));
    
    expect(onBatchChange).toHaveBeenCalledWith('1');
  });

  it('should show both organization and batch filters', () => {
    render(
      <FilterBar 
        {...defaultProps} 
        selectedOrganization="org1"
        selectedBatch="1"
      />
    );
    
    expect(screen.getByText('org1 (50)')).toBeInTheDocument();
    expect(screen.getByText('Batch 1 (10)')).toBeInTheDocument();
  });

  it('should use correct default text when no filters are selected', () => {
    render(<FilterBar {...defaultProps} />);
    
    expect(screen.getByText('All Organizations')).toBeInTheDocument();
    expect(screen.getByText('All Batches')).toBeInTheDocument();
  });

  it('should handle missing organization gracefully', () => {
    render(
      <FilterBar 
        {...defaultProps} 
        selectedOrganization="unknown-org"
      />
    );
    
    // When org is not found in list, it shows the raw value
    expect(screen.getByText('unknown-org')).toBeInTheDocument();
  });
});
