import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { ConfirmationDialog } from './ConfirmationDialog';

describe('ConfirmationDialog', () => {
  const defaultProps = {
    isOpen: true,
    title: 'Confirm Action',
    message: 'Are you sure you want to proceed?',
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Reset body overflow style
    document.body.style.overflow = '';
  });

  describe('when closed', () => {
    it('should not render anything', () => {
      render(<ConfirmationDialog {...defaultProps} isOpen={false} />);
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });

  describe('when open', () => {
    it('should render dialog with title', () => {
      render(<ConfirmationDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByText('Confirm Action')).toBeInTheDocument();
    });

    it('should render message', () => {
      render(<ConfirmationDialog {...defaultProps} />);
      expect(screen.getByText('Are you sure you want to proceed?')).toBeInTheDocument();
    });

    it('should render default button labels', () => {
      render(<ConfirmationDialog {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Confirm' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    });

    it('should render custom button labels', () => {
      render(
        <ConfirmationDialog
          {...defaultProps}
          confirmLabel="Delete"
          cancelLabel="Keep"
        />
      );
      expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Keep' })).toBeInTheDocument();
    });

    it('should render message as ReactNode', () => {
      render(
        <ConfirmationDialog
          {...defaultProps}
          message={<span data-testid="custom-message">Custom content</span>}
        />
      );
      expect(screen.getByTestId('custom-message')).toBeInTheDocument();
    });
  });

  describe('interactions', () => {
    it('should call onConfirm when confirm button is clicked', () => {
      const onConfirm = vi.fn();
      render(<ConfirmationDialog {...defaultProps} onConfirm={onConfirm} />);

      fireEvent.click(screen.getByRole('button', { name: 'Confirm' }));
      expect(onConfirm).toHaveBeenCalledTimes(1);
    });

    it('should call onCancel when cancel button is clicked', () => {
      const onCancel = vi.fn();
      render(<ConfirmationDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('should call onCancel when close button is clicked', () => {
      const onCancel = vi.fn();
      render(<ConfirmationDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.click(screen.getByLabelText('Close'));
      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('should call onCancel when backdrop is clicked', () => {
      const onCancel = vi.fn();
      render(<ConfirmationDialog {...defaultProps} onCancel={onCancel} />);

      // The backdrop has aria-hidden="true"
      const backdrop = document.querySelector('[aria-hidden="true"]');
      expect(backdrop).toBeInTheDocument();
      fireEvent.click(backdrop!);
      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    // Skip: popover polyfill interferes with Escape key in jsdom
    it.skip('should call onCancel when Escape key is pressed', () => {
      const onCancel = vi.fn();
      render(<ConfirmationDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.keyDown(document, { key: 'Escape' });
      expect(onCancel).toHaveBeenCalledTimes(1);
    });
  });

  describe('loading state', () => {
    it('should show loading text on confirm button', () => {
      render(<ConfirmationDialog {...defaultProps} isLoading={true} />);
      expect(screen.getByRole('button', { name: 'Loading...' })).toBeInTheDocument();
    });

    it('should disable buttons when loading', () => {
      render(<ConfirmationDialog {...defaultProps} isLoading={true} />);
      expect(screen.getByRole('button', { name: 'Loading...' })).toBeDisabled();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
    });

    it('should disable close button when loading', () => {
      render(<ConfirmationDialog {...defaultProps} isLoading={true} />);
      expect(screen.getByLabelText('Close')).toBeDisabled();
    });

    // Skip: popover polyfill interferes with Escape key in jsdom
    it.skip('should not call onCancel on Escape when loading', () => {
      const onCancel = vi.fn();
      render(<ConfirmationDialog {...defaultProps} onCancel={onCancel} isLoading={true} />);

      fireEvent.keyDown(document, { key: 'Escape' });
      expect(onCancel).not.toHaveBeenCalled();
    });

    it('should not call onCancel on backdrop click when loading', () => {
      const onCancel = vi.fn();
      render(<ConfirmationDialog {...defaultProps} onCancel={onCancel} isLoading={true} />);

      const backdrop = document.querySelector('[aria-hidden="true"]');
      fireEvent.click(backdrop!);
      expect(onCancel).not.toHaveBeenCalled();
    });
  });

  describe('accessibility', () => {
    it('should have role="dialog"', () => {
      render(<ConfirmationDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('should have aria-modal="true"', () => {
      render(<ConfirmationDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
    });

    it('should have aria-labelledby pointing to title', () => {
      render(<ConfirmationDialog {...defaultProps} />);
      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-labelledby', 'confirmation-dialog-title');
      
      const title = document.getElementById('confirmation-dialog-title');
      expect(title).toHaveTextContent('Confirm Action');
    });

    it('should prevent body scroll when open', () => {
      render(<ConfirmationDialog {...defaultProps} />);
      expect(document.body.style.overflow).toBe('hidden');
    });

    it('should restore body scroll when closed', () => {
      const { rerender } = render(<ConfirmationDialog {...defaultProps} />);
      expect(document.body.style.overflow).toBe('hidden');

      rerender(<ConfirmationDialog {...defaultProps} isOpen={false} />);
      expect(document.body.style.overflow).toBe('');
    });
  });
});

