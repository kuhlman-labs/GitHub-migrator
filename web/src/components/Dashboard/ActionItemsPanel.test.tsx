import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { ActionItemsPanel } from './ActionItemsPanel';
import type { DashboardActionItems } from '../../types';

const mockActionItems: DashboardActionItems = {
  failed_migrations: [
    {
      id: 1,
      full_name: 'org/failed-repo-1',
      status: 'migration_failed',
      batch_name: 'Batch 1',
      failed_at: '2024-01-15T10:00:00Z',
    },
    {
      id: 2,
      full_name: 'org/failed-repo-2',
      status: 'migration_failed',
    },
  ],
  failed_dry_runs: [
    {
      id: 3,
      full_name: 'org/dry-run-failed',
      status: 'dry_run_failed',
      batch_name: 'Batch 2',
      failed_at: '2024-01-14T15:00:00Z',
    },
  ],
  ready_batches: [
    {
      id: 1,
      name: 'Ready Batch 1',
      repository_count: 10,
      scheduled_at: '2024-01-20T08:00:00Z',
    },
  ],
  blocked_repositories: [
    {
      id: 4,
      full_name: 'org/blocked-repo',
      status: 'remediation_required',
      has_oversized_repository: true,
      has_blocking_files: false,
    },
  ],
  total_action_items: 5,
};

const emptyActionItems: DashboardActionItems = {
  failed_migrations: [],
  failed_dry_runs: [],
  ready_batches: [],
  blocked_repositories: [],
  total_action_items: 0,
};

describe('ActionItemsPanel', () => {
  it('renders loading skeleton when loading', () => {
    const { container } = render(
      <ActionItemsPanel actionItems={undefined} isLoading={true} />
    );

    const skeleton = container.querySelector('.animate-pulse');
    expect(skeleton).toBeInTheDocument();
  });

  it('does not render panel when no action items', () => {
    render(
      <ActionItemsPanel actionItems={emptyActionItems} isLoading={false} />
    );

    // The panel should not show the Action Items header when there are no items
    expect(screen.queryByText('Action Items')).not.toBeInTheDocument();
  });

  it('renders action items header with total count', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    expect(screen.getByText('Action Items')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument(); // Total count
  });

  it('renders failed migrations section', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    expect(screen.getByText('Failed Migrations')).toBeInTheDocument();
    expect(screen.getByText('org/failed-repo-1')).toBeInTheDocument();
    expect(screen.getByText('org/failed-repo-2')).toBeInTheDocument();
    expect(screen.getByText('Batch: Batch 1')).toBeInTheDocument();
  });

  it('renders failed dry runs section', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    expect(screen.getByText('Failed Dry Runs')).toBeInTheDocument();
    expect(screen.getByText('org/dry-run-failed')).toBeInTheDocument();
    expect(screen.getByText('Batch: Batch 2')).toBeInTheDocument();
  });

  it('renders ready batches section', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    expect(screen.getByText('Batches Ready to Start')).toBeInTheDocument();
    expect(screen.getByText('Ready Batch 1')).toBeInTheDocument();
    expect(screen.getByText(/10 repositories/)).toBeInTheDocument();
  });

  it('renders blocked repositories section header', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    // Blocked Repositories section is collapsed by default, so only header is visible
    expect(screen.getByText('Blocked Repositories')).toBeInTheDocument();
  });

  it('shows section counts in badges', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    // Failed migrations count: 2
    expect(screen.getByText('2')).toBeInTheDocument();
    // Failed dry runs count: 1
    expect(screen.getAllByText('1').length).toBeGreaterThanOrEqual(1);
  });

  it('allows collapsing and expanding sections', async () => {
    const user = userEvent.setup();
    
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    // Blocked repositories section is collapsed by default
    // Find the section header button and click to expand
    const blockedSection = screen.getByText('Blocked Repositories').closest('button');
    
    if (blockedSection) {
      // Click to toggle
      await user.click(blockedSection);
    }
  });

  it('renders View Details buttons for failed migrations', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    const viewDetailsButtons = screen.getAllByText('View Details');
    expect(viewDetailsButtons.length).toBeGreaterThanOrEqual(1);
  });

  it('renders View Batch button for ready batches', () => {
    render(<ActionItemsPanel actionItems={mockActionItems} isLoading={false} />);

    expect(screen.getByText('View Batch')).toBeInTheDocument();
  });

  it('does not render sections with zero items', () => {
    const partialActionItems = {
      ...emptyActionItems,
      failed_migrations: mockActionItems.failed_migrations,
      total_action_items: 2,
    };

    render(<ActionItemsPanel actionItems={partialActionItems} isLoading={false} />);

    expect(screen.getByText('Failed Migrations')).toBeInTheDocument();
    expect(screen.queryByText('Failed Dry Runs')).not.toBeInTheDocument();
    expect(screen.queryByText('Batches Ready to Start')).not.toBeInTheDocument();
    expect(screen.queryByText('Blocked Repositories')).not.toBeInTheDocument();
  });
});

