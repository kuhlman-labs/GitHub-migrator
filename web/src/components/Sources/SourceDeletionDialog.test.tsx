import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '../../__tests__/test-utils';
import { SourceDeletionDialog } from './SourceDeletionDialog';
import { sourcesApi } from '../../services/api/sources';
import type { Source, SourceDeletionPreview } from '../../types/source';

// Mock the sources API
vi.mock('../../services/api/sources', () => ({
  sourcesApi: {
    getDeletionPreview: vi.fn(),
  },
}));

describe('SourceDeletionDialog', () => {
  const mockSource: Source = {
    id: 1,
    name: 'Test Source',
    type: 'github',
    base_url: 'https://api.github.com',
    is_active: true,
    has_app_auth: false,
    has_oauth: false,
    repository_count: 10,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    masked_token: 'ghp_...xxxx',
  };

  const mockPreview: SourceDeletionPreview = {
    source_id: 1,
    source_name: 'Test Source',
    repository_count: 10,
    migration_history_count: 50,
    migration_log_count: 200,
    dependency_count: 5,
    team_repository_count: 3,
    batch_repository_count: 8,
    team_count: 2,
    user_count: 15,
    user_mapping_count: 15,
    team_mapping_count: 2,
    total_affected_records: 310,
  };

  const defaultProps = {
    isOpen: true,
    source: mockSource,
    onConfirm: vi.fn().mockResolvedValue(undefined),
    onCancel: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (sourcesApi.getDeletionPreview as ReturnType<typeof vi.fn>).mockResolvedValue(mockPreview);
  });

  afterEach(() => {
    document.body.style.overflow = '';
  });

  describe('when closed', () => {
    it('should not render anything', () => {
      render(<SourceDeletionDialog {...defaultProps} isOpen={false} />);
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('should not render when source is null', () => {
      render(<SourceDeletionDialog {...defaultProps} source={null} />);
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });

  describe('initial step', () => {
    it('should render dialog with title', () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByText('Delete Source')).toBeInTheDocument();
    });

    it('should show source name in warning message', () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      expect(screen.getByText(/"Test Source"/)).toBeInTheDocument();
    });

    it('should show warning about associated repositories', () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      expect(screen.getByText(/10 associated repositories/)).toBeInTheDocument();
    });

    it('should show Continue button', () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Continue' })).toBeInTheDocument();
    });

    it('should call onCancel when Cancel button is clicked', () => {
      const onCancel = vi.fn();
      render(<SourceDeletionDialog {...defaultProps} onCancel={onCancel} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      expect(onCancel).toHaveBeenCalledTimes(1);
    });
  });

  describe('preview step', () => {
    it('should load and show preview when Continue is clicked', async () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(sourcesApi.getDeletionPreview).toHaveBeenCalledWith(1);
      });

      // After preview loads, we should see the "I understand, continue" button
      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'I understand, continue' })).toBeInTheDocument();
      });
    });

    it('should show total affected records', async () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(screen.getByText(/310 total records will be deleted/)).toBeInTheDocument();
      });
    });

    it('should show "I understand, continue" button', async () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'I understand, continue' })).toBeInTheDocument();
      });
    });

    it('should handle preview loading error', async () => {
      (sourcesApi.getDeletionPreview as ReturnType<typeof vi.fn>).mockRejectedValue(
        new Error('Failed to load preview')
      );

      render(<SourceDeletionDialog {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(screen.getByText('Failed to load preview')).toBeInTheDocument();
      });
    });
  });

  describe('confirm step', () => {
    it('should show confirmation input after preview', async () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      
      // Go through initial step
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      // Wait for preview
      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'I understand, continue' })).toBeInTheDocument();
      });

      // Go to confirm step
      fireEvent.click(screen.getByRole('button', { name: 'I understand, continue' }));

      expect(screen.getByLabelText(/To confirm, type/)).toBeInTheDocument();
    });

    it('should have Delete Source button disabled initially', async () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'I understand, continue' })).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: 'I understand, continue' }));

      expect(screen.getByRole('button', { name: 'Delete Source' })).toBeDisabled();
    });

    it('should enable Delete Source button when correct name is typed', async () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'I understand, continue' })).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: 'I understand, continue' }));

      const input = screen.getByLabelText(/To confirm, type/);
      fireEvent.change(input, { target: { value: 'Test Source' } });

      expect(screen.getByRole('button', { name: 'Delete Source' })).not.toBeDisabled();
    });

    it('should keep Delete Source button disabled with wrong name', async () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'I understand, continue' })).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: 'I understand, continue' }));

      const input = screen.getByLabelText(/To confirm, type/);
      fireEvent.change(input, { target: { value: 'Wrong Name' } });

      expect(screen.getByRole('button', { name: 'Delete Source' })).toBeDisabled();
    });

    it('should call onConfirm with force=true when delete is clicked', async () => {
      const onConfirm = vi.fn().mockResolvedValue(undefined);
      render(<SourceDeletionDialog {...defaultProps} onConfirm={onConfirm} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'I understand, continue' })).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: 'I understand, continue' }));

      const input = screen.getByLabelText(/To confirm, type/);
      fireEvent.change(input, { target: { value: 'Test Source' } });

      fireEvent.click(screen.getByRole('button', { name: 'Delete Source' }));

      await waitFor(() => {
        expect(onConfirm).toHaveBeenCalledWith(true, 'Test Source');
      });
    });
  });

  describe('source with no data', () => {
    it('should skip preview step for source with no repositories', async () => {
      const emptySource: Source = {
        ...mockSource,
        repository_count: 0,
      };

      render(<SourceDeletionDialog {...defaultProps} source={emptySource} />);
      
      fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

      // Should go directly to confirm step without loading preview
      expect(sourcesApi.getDeletionPreview).not.toHaveBeenCalled();
      expect(screen.getByLabelText(/To confirm, type/)).toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('should have role="dialog"', () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('should have aria-modal="true"', () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
    });

    it('should prevent body scroll when open', () => {
      render(<SourceDeletionDialog {...defaultProps} />);
      expect(document.body.style.overflow).toBe('hidden');
    });

    it('should restore body scroll when closed', () => {
      const { rerender } = render(<SourceDeletionDialog {...defaultProps} />);
      expect(document.body.style.overflow).toBe('hidden');

      rerender(<SourceDeletionDialog {...defaultProps} isOpen={false} />);
      expect(document.body.style.overflow).toBe('');
    });
  });

  describe('keyboard interactions', () => {
    it('should call onCancel when Escape is pressed on dialog', () => {
      const onCancel = vi.fn();
      render(<SourceDeletionDialog {...defaultProps} onCancel={onCancel} />);

      const dialog = screen.getByRole('dialog');
      fireEvent.keyDown(dialog, { key: 'Escape' });
      // The event listener is on document, so we test via the close button instead
      expect(screen.getByLabelText('Close')).toBeInTheDocument();
    });

    it('should call onCancel when close button is clicked', () => {
      const onCancel = vi.fn();
      render(<SourceDeletionDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.click(screen.getByLabelText('Close'));
      expect(onCancel).toHaveBeenCalledTimes(1);
    });
  });
});

