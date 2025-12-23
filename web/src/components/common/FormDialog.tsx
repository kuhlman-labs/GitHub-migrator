import { useRef, useEffect } from 'react';
import { XIcon } from '@primer/octicons-react';
import { Button, IconButton } from './buttons';

export interface FormDialogProps {
  isOpen: boolean;
  title: string;
  submitLabel?: string;
  cancelLabel?: string;
  onSubmit: () => void;
  onCancel: () => void;
  isLoading?: boolean;
  isSubmitDisabled?: boolean;
  children: React.ReactNode;
  size?: 'small' | 'medium' | 'large';
  variant?: 'default' | 'primary' | 'danger';
}

const sizeClasses = {
  small: 'max-w-sm',
  medium: 'max-w-lg',
  large: 'max-w-2xl',
};

/**
 * A reusable form dialog component.
 * Use for dialogs that contain form fields or complex content.
 * 
 * @example
 * <FormDialog
 *   isOpen={showDiscoveryDialog}
 *   title="Start Discovery"
 *   submitLabel="Start"
 *   onSubmit={handleStartDiscovery}
 *   onCancel={() => setShowDiscoveryDialog(false)}
 *   isLoading={isDiscovering}
 *   isSubmitDisabled={!organization}
 * >
 *   <FormControl>
 *     <FormControl.Label>Organization</FormControl.Label>
 *     <TextInput value={organization} onChange={...} />
 *   </FormControl>
 * </FormDialog>
 */
export function FormDialog({
  isOpen,
  title,
  submitLabel = 'Submit',
  cancelLabel = 'Cancel',
  onSubmit,
  onCancel,
  isLoading = false,
  isSubmitDisabled = false,
  children,
  size = 'medium',
  variant = 'primary',
}: FormDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen && !isLoading) {
        onCancel();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isLoading, onCancel]);

  // Prevent body scroll when dialog is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  // Focus trap - focus first focusable element when dialog opens
  useEffect(() => {
    if (isOpen && dialogRef.current) {
      const focusableElements = dialogRef.current.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      if (focusableElements.length > 0) {
        focusableElements[0].focus();
      }
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!isSubmitDisabled && !isLoading) {
      onSubmit();
    }
  };

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-40"
        onClick={isLoading ? undefined : onCancel}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        className="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="form-dialog-title"
      >
        <div
          ref={dialogRef}
          className={`rounded-lg shadow-xl w-full ${sizeClasses[size]} max-h-[90vh] overflow-auto`}
          style={{ backgroundColor: 'var(--bgColor-default)' }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div
            className="flex items-center justify-between px-4 py-3 border-b"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <h2
              id="form-dialog-title"
              className="text-lg font-semibold"
              style={{ color: 'var(--fgColor-default)' }}
            >
              {title}
            </h2>
            <IconButton
              icon={XIcon}
              aria-label="Close"
              variant="invisible"
              onClick={onCancel}
              disabled={isLoading}
              sx={{ color: 'fg.muted' }}
            />
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit}>
            {/* Body */}
            <div className="p-4">{children}</div>

            {/* Footer */}
            <div
              className="px-4 py-3 border-t flex justify-end gap-2"
              style={{ borderColor: 'var(--borderColor-default)' }}
            >
              <Button type="button" onClick={onCancel} disabled={isLoading}>
                {cancelLabel}
              </Button>
              <Button
                type="submit"
                variant={variant}
                disabled={isLoading || isSubmitDisabled}
              >
                {isLoading ? 'Loading...' : submitLabel}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
}

