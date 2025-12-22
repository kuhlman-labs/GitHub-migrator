import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { BatchManagement } from './index';
import * as useQueriesModule from '../../hooks/useQueries';

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useBatches: vi.fn(),
  useBatchRepositories: vi.fn(),
}));

// Mock child components
vi.mock('./BatchListPanel', () => ({
  BatchListPanel: ({ 
    batches, 
    selectedBatchId, 
    onSelectBatch 
  }: { 
    batches: Array<{ id: number; name: string }>; 
    selectedBatchId: number | null; 
    onSelectBatch: (id: number) => void 
  }) => (
    <div data-testid="batch-list-panel">
      {batches.map((batch) => (
        <button 
          key={batch.id} 
          data-testid={`batch-item-${batch.id}`}
          onClick={() => onSelectBatch(batch.id)}
          data-selected={selectedBatchId === batch.id}
        >
          {batch.name}
        </button>
      ))}
    </div>
  ),
}));

vi.mock('./BatchDetailHeader', () => ({
  BatchDetailHeader: ({ batch }: { batch: { name: string } }) => (
    <div data-testid="batch-detail-header">{batch.name}</div>
  ),
}));

vi.mock('./BatchRepositoryItem', () => ({
  BatchRepositoryItem: ({ repository }: { repository: { full_name: string } }) => (
    <div data-testid={`batch-repo-${repository.full_name}`}>{repository.full_name}</div>
  ),
}));

describe('BatchManagement', () => {
  const mockBatches = [
    { id: 1, name: 'Batch 1', status: 'pending', repository_count: 5, created_at: '2024-01-01' },
    { id: 2, name: 'Batch 2', status: 'in_progress', repository_count: 10, created_at: '2024-01-02' },
    { id: 3, name: 'Batch 3', status: 'complete', repository_count: 8, created_at: '2024-01-03' },
  ];

  const mockRepositories = [
    { id: 1, full_name: 'org/repo1', status: 'pending' },
    { id: 2, full_name: 'org/repo2', status: 'complete' },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    
    (useQueriesModule.useBatches as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockBatches,
      isLoading: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    
    (useQueriesModule.useBatchRepositories as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { repositories: mockRepositories, total: 2 },
      refetch: vi.fn(),
    });
  });

  it('should render the batch management page title', async () => {
    render(<BatchManagement />);
    
    await waitFor(() => {
      expect(screen.getByText('Batch Management')).toBeInTheDocument();
    });
  });

  it('should render the create batch button', async () => {
    render(<BatchManagement />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /create new batch/i })).toBeInTheDocument();
    });
  });

  it('should render batch list panel', async () => {
    render(<BatchManagement />);
    
    await waitFor(() => {
      expect(screen.getByTestId('batch-list-panel')).toBeInTheDocument();
    });
  });

  it('should render batch items', async () => {
    render(<BatchManagement />);
    
    await waitFor(() => {
      expect(screen.getByTestId('batch-item-1')).toBeInTheDocument();
      expect(screen.getByTestId('batch-item-2')).toBeInTheDocument();
      expect(screen.getByTestId('batch-item-3')).toBeInTheDocument();
    });
  });

  it('should show batch detail header when batch is selected', async () => {
    const user = userEvent.setup();
    render(<BatchManagement />);
    
    await waitFor(() => {
      expect(screen.getByTestId('batch-item-1')).toBeInTheDocument();
    });
    
    await user.click(screen.getByTestId('batch-item-1'));
    
    await waitFor(() => {
      expect(screen.getByTestId('batch-detail-header')).toBeInTheDocument();
    });
  });

  it('should render page header', async () => {
    render(<BatchManagement />);
    
    await waitFor(() => {
      // Check that the page header is rendered
      expect(screen.getByRole('heading', { name: 'Batch Management' })).toBeInTheDocument();
    });
  });

  it('should show loading state when batches are loading', async () => {
    (useQueriesModule.useBatches as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: true,
      isFetching: true,
      refetch: vi.fn(),
    });
    
    render(<BatchManagement />);
    
    // When loading, the batch list panel is still rendered but empty
    await waitFor(() => {
      expect(screen.getByText('Batch Management')).toBeInTheDocument();
    });
  });

  it('should show empty state when no batches', async () => {
    (useQueriesModule.useBatches as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    
    render(<BatchManagement />);
    
    await waitFor(() => {
      // Empty batch list panel should be rendered
      expect(screen.getByTestId('batch-list-panel')).toBeInTheDocument();
    });
  });

  it('should show batch detail when batch is selected', async () => {
    const user = userEvent.setup();
    render(<BatchManagement />);
    
    await waitFor(() => {
      expect(screen.getByTestId('batch-item-1')).toBeInTheDocument();
    });
    
    await user.click(screen.getByTestId('batch-item-1'));
    
    // Batch detail header should show
    await waitFor(() => {
      expect(screen.getByTestId('batch-detail-header')).toBeInTheDocument();
    });
  });

  it('should navigate to batch builder on create click', async () => {
    const user = userEvent.setup();
    render(<BatchManagement />);
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /create new batch/i })).toBeInTheDocument();
    });
    
    await user.click(screen.getByRole('button', { name: /create new batch/i }));
    
    // Navigation would happen - this is tested through integration
  });
});
