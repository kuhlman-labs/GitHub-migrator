import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { BatchSummaryPanel } from './BatchSummaryPanel';
import type { Repository } from '../../types';

const mockRepositories: Repository[] = [
  {
    id: 1,
    full_name: 'org/repo1',
    name: 'repo1',
    source: 'github',
    status: 'pending',
    total_size: 1024000,
    complexity_category: 'simple',
    organization: 'org',
  },
  {
    id: 2,
    full_name: 'org/repo2',
    name: 'repo2',
    source: 'github',
    status: 'complete',
    total_size: 2048000,
    complexity_category: 'medium',
    organization: 'org',
  },
] as Repository[];

describe('BatchSummaryPanel', () => {
  const defaultProps = {
    currentBatchRepos: mockRepositories,
    groupedRepos: { org: mockRepositories },
    totalSize: 3072000,
    onRemoveRepo: vi.fn(),
    onClearAll: vi.fn(),
  };

  it('renders the panel', () => {
    const { container } = render(<BatchSummaryPanel {...defaultProps} />);

    expect(container).toBeDefined();
  });

  it('displays repository count', () => {
    render(<BatchSummaryPanel {...defaultProps} />);

    // Should show 2 repositories
    expect(screen.getByText(/2 repositor/i)).toBeInTheDocument();
  });

  it('displays content', () => {
    const { container } = render(<BatchSummaryPanel {...defaultProps} />);

    // Should have content
    expect(container.textContent?.length || 0).toBeGreaterThan(0);
  });

  it('shows clear all button', () => {
    render(<BatchSummaryPanel {...defaultProps} />);

    expect(screen.getByRole('button', { name: /Clear All/i })).toBeInTheDocument();
  });

  it('calls onClearAll when clear button is clicked', async () => {
    const onClearAll = vi.fn();
    render(<BatchSummaryPanel {...defaultProps} onClearAll={onClearAll} />);

    const clearButton = screen.getByRole('button', { name: /Clear All/i });
    clearButton.click();

    expect(onClearAll).toHaveBeenCalled();
  });
});

