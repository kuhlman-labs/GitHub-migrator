import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { DiscoveryProgressCard, LastDiscoveryIndicator } from './DiscoveryProgressCard';
import type { DiscoveryProgress } from '../../types';

const createProgress = (overrides: Partial<DiscoveryProgress> = {}): DiscoveryProgress => ({
  status: 'in_progress',
  discovery_type: 'full',
  target: 'my-org',
  current_org: 'my-org',
  total_orgs: 1,
  processed_orgs: 0,
  total_repos: 100,
  processed_repos: 50,
  phase: 'profiling_repos',
  started_at: '2024-01-15T10:00:00Z',
  error_count: 0,
  ...overrides,
});

describe('DiscoveryProgressCard', () => {
  describe('in progress state', () => {
    it('should render progress information', () => {
      render(<DiscoveryProgressCard progress={createProgress()} />);

      expect(screen.getByText('Discovery in Progress')).toBeInTheDocument();
      expect(screen.getByText('Full')).toBeInTheDocument();
      expect(screen.getByText('50 / 100 repos')).toBeInTheDocument();
    });

    it('should show phase label', () => {
      render(<DiscoveryProgressCard progress={createProgress({ phase: 'profiling_repos' })} />);

      expect(screen.getByText('Profiling repositories...')).toBeInTheDocument();
    });

    it('should show current org for single org discovery', () => {
      render(<DiscoveryProgressCard progress={createProgress({ current_org: 'test-org' })} />);

      expect(screen.getByText('Processing: test-org')).toBeInTheDocument();
    });

    it('should show org progress for multi-org discovery', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            total_orgs: 5,
            processed_orgs: 2,
            current_org: 'org-3',
          })}
        />
      );

      expect(screen.getByText('Org 3 of 5: org-3')).toBeInTheDocument();
    });

    it('should show total orgs when current_org is empty (batch profiling)', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            total_orgs: 5,
            processed_orgs: 0,
            current_org: '', // Empty during batch profiling
            phase: 'profiling_repos',
          })}
        />
      );

      expect(screen.getByText('5 organizations')).toBeInTheDocument();
      expect(screen.getByText('Profiling repositories...')).toBeInTheDocument();
    });

    it('should show error count when errors exist', () => {
      render(<DiscoveryProgressCard progress={createProgress({ error_count: 3 })} />);

      expect(screen.getByText('3 errors encountered')).toBeInTheDocument();
    });

    it('should show singular error text for single error', () => {
      render(<DiscoveryProgressCard progress={createProgress({ error_count: 1 })} />);

      expect(screen.getByText('1 error encountered')).toBeInTheDocument();
    });
  });

  describe('completed state', () => {
    it('should render completed status', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'completed',
            processed_repos: 100,
            processed_orgs: 3,
            completed_at: '2024-01-15T10:30:00Z',
          })}
        />
      );

      expect(screen.getByText('Discovery Complete')).toBeInTheDocument();
      expect(screen.getByText(/Discovered 100 repositories across 3 organizations/)).toBeInTheDocument();
    });

    it('should show dismiss button when onDismiss provided', () => {
      const onDismiss = vi.fn();
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'completed' })}
          onDismiss={onDismiss}
        />
      );

      const dismissButton = screen.getByLabelText('Dismiss');
      expect(dismissButton).toBeInTheDocument();
      
      fireEvent.click(dismissButton);
      expect(onDismiss).toHaveBeenCalledTimes(1);
    });

    it('should use singular for single organization', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'completed',
            processed_repos: 50,
            processed_orgs: 1,
          })}
        />
      );

      // For GitHub discovery, uses "org" singular
      expect(screen.getByText(/Discovered 50 repositories across 1 org$/)).toBeInTheDocument();
    });

    it('should use project terminology for ADO discovery', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'completed',
            discovery_type: 'ado_organization',
            processed_repos: 50,
            processed_orgs: 3,
          })}
        />
      );

      // For ADO discovery, uses "projects" plural
      expect(screen.getByText(/Discovered 50 repositories across 3 projects$/)).toBeInTheDocument();
    });
  });

  describe('failed state', () => {
    it('should render failed status', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'failed',
            last_error: 'Rate limit exceeded',
          })}
        />
      );

      expect(screen.getByText('Discovery Failed')).toBeInTheDocument();
      expect(screen.getByText('Rate limit exceeded')).toBeInTheDocument();
    });

    it('should show progress before failure', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'failed',
            processed_repos: 75,
            total_repos: 100,
          })}
        />
      );

      expect(screen.getByText('Processed 75 of 100 repositories before failure')).toBeInTheDocument();
    });
  });

  describe('cancelled state', () => {
    it('should render cancelled status', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'cancelled',
            processed_repos: 50,
            total_repos: 100,
            completed_at: '2024-01-15T10:30:00Z',
          })}
        />
      );

      expect(screen.getByText('Discovery Cancelled')).toBeInTheDocument();
    });

    it('should show progress before cancellation', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'cancelled',
            processed_repos: 50,
            total_repos: 100,
          })}
        />
      );

      expect(screen.getByText(/Discovery was cancelled after processing 50 of 100 repositories/)).toBeInTheDocument();
    });

    it('should show dismiss button when onDismiss provided', () => {
      const onDismiss = vi.fn();
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'cancelled' })}
          onDismiss={onDismiss}
        />
      );

      const dismissButton = screen.getByLabelText('Dismiss');
      expect(dismissButton).toBeInTheDocument();
      
      fireEvent.click(dismissButton);
      expect(onDismiss).toHaveBeenCalledTimes(1);
    });
  });

  describe('cancelling phase', () => {
    it('should show cancelling status when phase is cancelling', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({
            status: 'in_progress',
            phase: 'cancelling',
          })}
        />
      );

      expect(screen.getByText('Cancelling Discovery...')).toBeInTheDocument();
      expect(screen.getByText('Cancelling...')).toBeInTheDocument();
    });

    it('should show cancelling status when isCancelling prop is true', () => {
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'in_progress' })}
          isCancelling={true}
        />
      );

      expect(screen.getByText('Cancelling Discovery...')).toBeInTheDocument();
    });
  });

  describe('cancel button', () => {
    it('should show cancel button when in progress and onCancel provided', () => {
      const onCancel = vi.fn();
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'in_progress' })}
          onCancel={onCancel}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      expect(cancelButton).toBeInTheDocument();
    });

    it('should call onCancel when cancel button is clicked', () => {
      const onCancel = vi.fn();
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'in_progress' })}
          onCancel={onCancel}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      fireEvent.click(cancelButton);
      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('should not show cancel button when isCancelling is true', () => {
      const onCancel = vi.fn();
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'in_progress' })}
          onCancel={onCancel}
          isCancelling={true}
        />
      );

      // When cancelling is in progress, the cancel button should be hidden
      // and the status should show "Cancelling Discovery..."
      expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
      expect(screen.getByText('Cancelling Discovery...')).toBeInTheDocument();
    });

    it('should not show cancel button when not in progress', () => {
      const onCancel = vi.fn();
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'completed' })}
          onCancel={onCancel}
        />
      );

      expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
    });

    it('should not show cancel button when phase is cancelling', () => {
      const onCancel = vi.fn();
      render(
        <DiscoveryProgressCard
          progress={createProgress({ status: 'in_progress', phase: 'cancelling' })}
          onCancel={onCancel}
        />
      );

      // Should show "Cancelling..." but not the clickable Cancel button
      expect(screen.queryByRole('button', { name: /^cancel$/i })).not.toBeInTheDocument();
    });
  });

  describe('discovery types', () => {
    it('should format full discovery type', () => {
      render(<DiscoveryProgressCard progress={createProgress({ discovery_type: 'full' })} />);
      expect(screen.getByText('Full')).toBeInTheDocument();
    });

    it('should format incremental discovery type', () => {
      render(<DiscoveryProgressCard progress={createProgress({ discovery_type: 'incremental' })} />);
      expect(screen.getByText('Incremental')).toBeInTheDocument();
    });
  });
});

describe('LastDiscoveryIndicator', () => {
  it('should render last discovery info', () => {
    render(
      <LastDiscoveryIndicator
        progress={createProgress({
          status: 'completed',
          processed_repos: 150,
          completed_at: new Date().toISOString(),
        })}
      />
    );

    expect(screen.getByText(/Last discovery: 150 repos/)).toBeInTheDocument();
  });

  it('should call onExpand when clicked', () => {
    const onExpand = vi.fn();
    render(
      <LastDiscoveryIndicator
        progress={createProgress({ status: 'completed' })}
        onExpand={onExpand}
      />
    );

    fireEvent.click(screen.getByRole('button'));
    expect(onExpand).toHaveBeenCalledTimes(1);
  });

  it('should call onExpand when Enter is pressed', () => {
    const onExpand = vi.fn();
    render(
      <LastDiscoveryIndicator
        progress={createProgress({ status: 'completed' })}
        onExpand={onExpand}
      />
    );

    fireEvent.keyDown(screen.getByRole('button'), { key: 'Enter' });
    expect(onExpand).toHaveBeenCalledTimes(1);
  });

  it('should format relative time as just now', () => {
    render(
      <LastDiscoveryIndicator
        progress={createProgress({
          status: 'completed',
          completed_at: new Date().toISOString(),
        })}
      />
    );

    expect(screen.getByText(/just now/)).toBeInTheDocument();
  });
});

