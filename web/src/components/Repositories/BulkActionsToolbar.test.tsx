import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { BulkActionsToolbar } from './BulkActionsToolbar';

// Mock useMutations
vi.mock('../../hooks/useMutations', () => ({
  useBatchUpdateRepositoryStatus: vi.fn(() => ({
    mutateAsync: vi.fn().mockResolvedValue({ updated_count: 5, failed_count: 0 }),
    isPending: false,
  })),
}));

describe('BulkActionsToolbar', () => {
  const mockOnClearSelection = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render selected count for single repository', () => {
    render(
      <BulkActionsToolbar
        selectedCount={1}
        selectedIds={[1]}
        onClearSelection={mockOnClearSelection}
      />
    );

    expect(screen.getByText('1 repository selected')).toBeInTheDocument();
  });

  it('should render selected count for multiple repositories', () => {
    render(
      <BulkActionsToolbar
        selectedCount={5}
        selectedIds={[1, 2, 3, 4, 5]}
        onClearSelection={mockOnClearSelection}
      />
    );

    expect(screen.getByText('5 repositories selected')).toBeInTheDocument();
  });

  it('should render Actions button', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    expect(screen.getByText('Actions')).toBeInTheDocument();
  });

  it('should render Clear Selection button', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    expect(screen.getByText('Clear Selection')).toBeInTheDocument();
  });

  it('should call onClearSelection when Clear Selection is clicked', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    fireEvent.click(screen.getByText('Clear Selection'));
    expect(mockOnClearSelection).toHaveBeenCalledTimes(1);
  });

  it('should show action menu when Actions button is clicked', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    fireEvent.click(screen.getByText('Actions'));

    expect(screen.getByText('Mark as Migrated')).toBeInTheDocument();
    expect(screen.getByText("Mark as Won't Migrate")).toBeInTheDocument();
    expect(screen.getByText("Unmark Won't Migrate")).toBeInTheDocument();
    expect(screen.getByText('Rollback Migration')).toBeInTheDocument();
  });

  it('should show action descriptions in menu', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    fireEvent.click(screen.getByText('Actions'));

    expect(screen.getByText('For repositories migrated outside this system')).toBeInTheDocument();
    expect(screen.getByText('Exclude from migration plans')).toBeInTheDocument();
  });

  it('should show confirmation dialog when action is selected', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    // Open menu
    fireEvent.click(screen.getByText('Actions'));
    
    // Click on Mark as Migrated
    fireEvent.click(screen.getByText('Mark as Migrated'));

    expect(screen.getByText('Mark Repositories as Migrated?')).toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
    expect(screen.getByText('Confirm')).toBeInTheDocument();
  });

  it('should show repository count in confirmation dialog', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    // Open menu
    fireEvent.click(screen.getByText('Actions'));
    
    // Click on Mark as Migrated
    fireEvent.click(screen.getByText('Mark as Migrated'));

    // Look for the combined text about repositories being affected
    expect(screen.getByText(/repositories will be/)).toBeInTheDocument();
  });

  it('should close confirmation dialog when Cancel is clicked', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    // Open menu and select action
    fireEvent.click(screen.getByText('Actions'));
    fireEvent.click(screen.getByText('Mark as Migrated'));

    // Verify dialog is open
    expect(screen.getByText('Mark Repositories as Migrated?')).toBeInTheDocument();

    // Click Cancel
    fireEvent.click(screen.getByText('Cancel'));

    // Dialog should be closed
    expect(screen.queryByText('Mark Repositories as Migrated?')).not.toBeInTheDocument();
  });

  it('should show reason input for rollback action', () => {
    render(
      <BulkActionsToolbar
        selectedCount={3}
        selectedIds={[1, 2, 3]}
        onClearSelection={mockOnClearSelection}
      />
    );

    // Open menu
    fireEvent.click(screen.getByText('Actions'));
    
    // Click on Rollback
    fireEvent.click(screen.getByText('Rollback Migration'));

    expect(screen.getByText('Reason (optional)')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g., Migration issues, incorrect destination, etc.')).toBeInTheDocument();
  });
});

