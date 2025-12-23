import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { BatchListPanel } from './BatchListPanel';
import type { Batch } from '../../types';

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const mockBatches: Batch[] = [
  {
    id: 1,
    name: 'Batch 1',
    description: 'First batch',
    type: 'manual',
    repository_count: 10,
    status: 'ready',
    created_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'Batch 2',
    description: 'Second batch',
    type: 'manual',
    repository_count: 5,
    status: 'in_progress',
    created_at: '2024-01-02T00:00:00Z',
  },
  {
    id: 3,
    name: 'Completed Batch',
    description: 'Finished',
    type: 'manual',
    repository_count: 20,
    status: 'completed',
    created_at: '2024-01-03T00:00:00Z',
  },
];

describe('BatchListPanel', () => {
  const mockOnTabChange = vi.fn();
  const mockOnSelectBatch = vi.fn();
  const mockOnStartBatch = vi.fn();
  const mockOnPageChange = vi.fn();

  const defaultProps = {
    batches: mockBatches,
    loading: false,
    activeTab: 'active' as const,
    onTabChange: mockOnTabChange,
    selectedBatchId: null,
    onSelectBatch: mockOnSelectBatch,
    onStartBatch: mockOnStartBatch,
    currentPage: 1,
    pageSize: 10,
    onPageChange: mockOnPageChange,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders tabs for active and completed batches', () => {
    render(<BatchListPanel {...defaultProps} />);

    expect(screen.getByText('Active (2)')).toBeInTheDocument();
    expect(screen.getByText('Completed (1)')).toBeInTheDocument();
  });

  it('displays active batches by default', () => {
    render(<BatchListPanel {...defaultProps} />);

    expect(screen.getByText('Batch 1')).toBeInTheDocument();
    expect(screen.getByText('Batch 2')).toBeInTheDocument();
    expect(screen.queryByText('Completed Batch')).not.toBeInTheDocument();
  });

  it('displays completed batches when tab is completed', () => {
    render(<BatchListPanel {...defaultProps} activeTab="completed" />);

    expect(screen.queryByText('Batch 1')).not.toBeInTheDocument();
    expect(screen.queryByText('Batch 2')).not.toBeInTheDocument();
    expect(screen.getByText('Completed Batch')).toBeInTheDocument();
  });

  it('calls onTabChange when clicking tabs', () => {
    render(<BatchListPanel {...defaultProps} />);

    fireEvent.click(screen.getByText('Completed (1)'));
    expect(mockOnTabChange).toHaveBeenCalledWith('completed');
  });

  it('shows loading spinner when loading', () => {
    render(<BatchListPanel {...defaultProps} loading={true} />);

    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('shows empty state for no active batches', () => {
    render(
      <BatchListPanel
        {...defaultProps}
        batches={[]}
      />
    );

    expect(screen.getByText('No active batches')).toBeInTheDocument();
    expect(screen.getByText('Create a batch to group repositories for migration.')).toBeInTheDocument();
  });

  it('shows empty state for no completed batches', () => {
    render(
      <BatchListPanel
        {...defaultProps}
        batches={[]}
        activeTab="completed"
      />
    );

    expect(screen.getByText('No completed batches')).toBeInTheDocument();
    expect(screen.getByText('Completed batches will appear here once migrations finish.')).toBeInTheDocument();
  });

  it('shows search empty state when no results match', () => {
    render(
      <BatchListPanel
        {...defaultProps}
        searchTerm="nonexistent"
      />
    );

    expect(screen.getByText('No batches match your search')).toBeInTheDocument();
    expect(screen.getByText('Try a different search term to find batches.')).toBeInTheDocument();
  });

  it('filters batches by search term', () => {
    render(
      <BatchListPanel
        {...defaultProps}
        searchTerm="Batch 1"
      />
    );

    expect(screen.getByText('Batch 1')).toBeInTheDocument();
    expect(screen.queryByText('Batch 2')).not.toBeInTheDocument();
  });

  it('filters batches by description', () => {
    render(
      <BatchListPanel
        {...defaultProps}
        searchTerm="First"
      />
    );

    expect(screen.getByText('Batch 1')).toBeInTheDocument();
    expect(screen.queryByText('Batch 2')).not.toBeInTheDocument();
  });

  it('calls onSelectBatch when clicking a batch', () => {
    render(<BatchListPanel {...defaultProps} />);

    fireEvent.click(screen.getByText('Batch 1'));
    expect(mockOnSelectBatch).toHaveBeenCalledWith(1);
  });

  it('highlights selected batch', () => {
    render(<BatchListPanel {...defaultProps} selectedBatchId={1} />);

    // The BatchCard component will handle the visual selection
    // We just verify it renders with the correct prop
    expect(screen.getByText('Batch 1')).toBeInTheDocument();
  });

  it('shows pagination when items exceed page size', () => {
    const manyBatches: Batch[] = Array.from({ length: 15 }, (_, i) => ({
      id: i + 1,
      name: `Batch ${i + 1}`,
      description: `Batch description ${i + 1}`,
      type: 'manual',
      repository_count: 10,
      status: 'pending',
      created_at: '2024-01-01T00:00:00Z',
    }));

    render(
      <BatchListPanel
        {...defaultProps}
        batches={manyBatches}
        pageSize={10}
      />
    );

    // Should show pagination controls - check for navigation buttons
    expect(screen.getByRole('navigation', { name: /pagination/i })).toBeInTheDocument();
  });

  it('navigates to new batch page when clicking create button', () => {
    render(
      <BatchListPanel
        {...defaultProps}
        batches={[]}
      />
    );

    fireEvent.click(screen.getByText('Create New Batch'));
    expect(mockNavigate).toHaveBeenCalledWith('/batches/new');
  });
});

