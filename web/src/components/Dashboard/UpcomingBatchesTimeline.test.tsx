import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { UpcomingBatchesTimeline } from './UpcomingBatchesTimeline';
import type { Batch } from '../../types';

const mockBatches: Batch[] = [
  {
    id: 1,
    name: 'Batch 1',
    description: 'First batch',
    type: 'manual',
    status: 'ready',
    repository_count: 10,
    created_at: '2024-01-01T00:00:00Z',
    scheduled_at: '2024-01-20T08:00:00Z',
  },
  {
    id: 2,
    name: 'Batch 2',
    description: 'Second batch',
    type: 'scheduled',
    status: 'pending',
    repository_count: 5,
    created_at: '2024-01-02T00:00:00Z',
    scheduled_at: '2024-01-25T10:00:00Z',
  },
];

describe('UpcomingBatchesTimeline', () => {
  it('renders the timeline', () => {
    const { container } = render(<UpcomingBatchesTimeline batches={mockBatches} />);

    expect(container).toBeDefined();
  });

  it('shows batch names', () => {
    render(<UpcomingBatchesTimeline batches={mockBatches} />);

    expect(screen.getByText('Batch 1')).toBeInTheDocument();
    expect(screen.getByText('Batch 2')).toBeInTheDocument();
  });

  it('shows repository counts', () => {
    render(<UpcomingBatchesTimeline batches={mockBatches} />);

    expect(screen.getByText(/10 repo/i)).toBeInTheDocument();
    expect(screen.getByText(/5 repo/i)).toBeInTheDocument();
  });

  it('handles empty batches', () => {
    const { container } = render(<UpcomingBatchesTimeline batches={[]} />);

    expect(container.textContent).toBe('');
  });
});

