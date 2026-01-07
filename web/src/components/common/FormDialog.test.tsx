import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { FormDialog } from './FormDialog';

describe('FormDialog', () => {
  const defaultProps = {
    isOpen: true,
    title: 'Test Form',
    onSubmit: vi.fn(),
    onCancel: vi.fn(),
    children: <input data-testid="test-input" placeholder="Enter value" />,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    document.body.style.overflow = '';
  });

  describe('when closed', () => {
    it('should not render anything', () => {
      render(<FormDialog {...defaultProps} isOpen={false} />);
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });

  describe('when open', () => {
    it('should render dialog with title', () => {
      render(<FormDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByText('Test Form')).toBeInTheDocument();
    });

    it('should render children', () => {
      render(<FormDialog {...defaultProps} />);
      expect(screen.getByTestId('test-input')).toBeInTheDocument();
    });

    it('should render default button labels', () => {
      render(<FormDialog {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Submit' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    });

    it('should render custom button labels', () => {
      render(
        <FormDialog {...defaultProps} submitLabel="Save" cancelLabel="Discard" />
      );
      expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Discard' })).toBeInTheDocument();
    });
  });

  describe('form submission', () => {
    it('should call onSubmit when form is submitted', () => {
      const onSubmit = vi.fn();
      render(<FormDialog {...defaultProps} onSubmit={onSubmit} />);

      fireEvent.click(screen.getByRole('button', { name: 'Submit' }));
      expect(onSubmit).toHaveBeenCalledTimes(1);
    });

    it('should not call onSubmit when disabled', () => {
      const onSubmit = vi.fn();
      render(<FormDialog {...defaultProps} onSubmit={onSubmit} isSubmitDisabled />);

      fireEvent.click(screen.getByRole('button', { name: 'Submit' }));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('should prevent default form submission', () => {
      const onSubmit = vi.fn();
      render(<FormDialog {...defaultProps} onSubmit={onSubmit} />);

      const form = document.querySelector('form');
      const event = new Event('submit', { bubbles: true, cancelable: true });
      const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
      
      form?.dispatchEvent(event);
      expect(preventDefaultSpy).toHaveBeenCalled();
    });
  });

  describe('cancel interactions', () => {
    it('should call onCancel when cancel button is clicked', () => {
      const onCancel = vi.fn();
      render(<FormDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('should call onCancel when close button is clicked', () => {
      const onCancel = vi.fn();
      render(<FormDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.click(screen.getByLabelText('Close'));
      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('should call onCancel when backdrop is clicked', () => {
      const onCancel = vi.fn();
      render(<FormDialog {...defaultProps} onCancel={onCancel} />);

      const backdrop = document.querySelector('[aria-hidden="true"]');
      fireEvent.click(backdrop!);
      expect(onCancel).toHaveBeenCalledTimes(1);
    });

  });

  describe('loading state', () => {
    it('should show loading text on submit button', () => {
      render(<FormDialog {...defaultProps} isLoading />);
      expect(screen.getByRole('button', { name: 'Loading...' })).toBeInTheDocument();
    });

    it('should disable buttons when loading', () => {
      render(<FormDialog {...defaultProps} isLoading />);
      expect(screen.getByRole('button', { name: 'Loading...' })).toBeDisabled();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
    });

    it('should not call onSubmit when loading', () => {
      const onSubmit = vi.fn();
      render(<FormDialog {...defaultProps} onSubmit={onSubmit} isLoading />);

      fireEvent.click(screen.getByRole('button', { name: 'Loading...' }));
      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  describe('size variants', () => {
    it('should render with small size', () => {
      const { container } = render(<FormDialog {...defaultProps} size="small" />);
      expect(container.querySelector('.max-w-sm')).toBeInTheDocument();
    });

    it('should render with medium size by default', () => {
      const { container } = render(<FormDialog {...defaultProps} />);
      expect(container.querySelector('.max-w-lg')).toBeInTheDocument();
    });

    it('should render with large size', () => {
      const { container } = render(<FormDialog {...defaultProps} size="large" />);
      expect(container.querySelector('.max-w-2xl')).toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('should have role="dialog"', () => {
      render(<FormDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('should have aria-modal="true"', () => {
      render(<FormDialog {...defaultProps} />);
      expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
    });

    it('should have aria-labelledby pointing to title', () => {
      render(<FormDialog {...defaultProps} />);
      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-labelledby', 'form-dialog-title');
    });

    it('should prevent body scroll when open', () => {
      render(<FormDialog {...defaultProps} />);
      expect(document.body.style.overflow).toBe('hidden');
    });
  });
});

