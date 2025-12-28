import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { StatusBadge } from './StatusBadge';

describe('StatusBadge', () => {
  describe('text formatting', () => {
    it('should replace underscores with spaces', () => {
      render(<StatusBadge status="dry_run_complete" />);
      expect(screen.getByText('dry run complete')).toBeInTheDocument();
    });

    it('should render single word status', () => {
      render(<StatusBadge status="pending" />);
      expect(screen.getByText('pending')).toBeInTheDocument();
    });

    it('should handle status with multiple underscores', () => {
      render(<StatusBadge status="dry_run_in_progress" />);
      expect(screen.getByText('dry run in progress')).toBeInTheDocument();
    });
  });

  describe('status variants', () => {
    it('should render pending status', () => {
      render(<StatusBadge status="pending" />);
      expect(screen.getByText('pending')).toBeInTheDocument();
    });

    it('should render ready status', () => {
      render(<StatusBadge status="ready" />);
      expect(screen.getByText('ready')).toBeInTheDocument();
    });

    it('should render remediation_required status', () => {
      render(<StatusBadge status="remediation_required" />);
      expect(screen.getByText('remediation required')).toBeInTheDocument();
    });

    it('should render in-progress statuses', () => {
      const inProgressStatuses = [
        'dry_run_queued',
        'dry_run_in_progress',
        'pre_migration',
        'archive_generating',
        'queued_for_migration',
        'migrating_content',
        'post_migration',
        'in_progress',
      ];

      inProgressStatuses.forEach((status) => {
        const { unmount } = render(<StatusBadge status={status} />);
        expect(screen.getByText(status.replace(/_/g, ' '))).toBeInTheDocument();
        unmount();
      });
    });

    it('should render complete statuses', () => {
      const completeStatuses = [
        'dry_run_complete',
        'migration_complete',
        'complete',
        'completed',
      ];

      completeStatuses.forEach((status) => {
        const { unmount } = render(<StatusBadge status={status} />);
        expect(screen.getByText(status.replace(/_/g, ' '))).toBeInTheDocument();
        unmount();
      });
    });

    it('should render failed statuses', () => {
      const failedStatuses = ['dry_run_failed', 'migration_failed', 'failed'];

      failedStatuses.forEach((status) => {
        const { unmount } = render(<StatusBadge status={status} />);
        expect(screen.getByText(status.replace(/_/g, ' '))).toBeInTheDocument();
        unmount();
      });
    });

    it('should render cancelled status', () => {
      render(<StatusBadge status="cancelled" />);
      expect(screen.getByText('cancelled')).toBeInTheDocument();
    });

    it('should render wont_migrate status', () => {
      render(<StatusBadge status="wont_migrate" />);
      expect(screen.getByText('wont migrate')).toBeInTheDocument();
    });

    it('should render completed_with_errors status', () => {
      render(<StatusBadge status="completed_with_errors" />);
      expect(screen.getByText('completed with errors')).toBeInTheDocument();
    });

    it('should render rolled_back status', () => {
      render(<StatusBadge status="rolled_back" />);
      expect(screen.getByText('rolled back')).toBeInTheDocument();
    });
  });

  describe('size variants', () => {
    it('should render with large size by default', () => {
      render(<StatusBadge status="pending" />);
      expect(screen.getByText('pending')).toBeInTheDocument();
    });

    it('should render with small size', () => {
      render(<StatusBadge status="pending" size="small" />);
      expect(screen.getByText('pending')).toBeInTheDocument();
    });

    it('should render with large size explicitly', () => {
      render(<StatusBadge status="pending" size="large" />);
      expect(screen.getByText('pending')).toBeInTheDocument();
    });
  });

  describe('unknown status', () => {
    it('should render unknown status with default variant', () => {
      render(<StatusBadge status="unknown_status" />);
      expect(screen.getByText('unknown status')).toBeInTheDocument();
    });
  });
});

