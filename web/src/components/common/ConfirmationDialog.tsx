import { useRef, useEffect } from 'react';
import { XIcon } from '@primer/octicons-react';
import { Button, IconButton } from './buttons';

export interface ConfirmationDialogProps {
  isOpen: boolean;
  title: string;
  message: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'default' | 'danger' | 'primary';
  onConfirm: () => void;
  onCancel: () => void;
  isLoading?: boolean;
}

/**
 * A reusable confirmation dialog component.
 * Use for simple yes/no confirmations like delete, start, retry actions.
 * 
 * @example
 * <ConfirmationDialog
 *   isOpen={showDeleteDialog}
 *   title="Delete Batch"
 *   message="Are you sure you want to delete this batch?"
 *   confirmLabel="Delete"
 *   variant="danger"
 *   onConfirm={handleDelete}
 *   onCancel={() => setShowDeleteDialog(false)}
 * />
 */
export function ConfirmationDialog({
  isOpen,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'primary',
  onConfirm,
  onCancel,
  isLoading = false,
}: ConfirmationDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const confirmButtonRef = useRef<HTMLButtonElement>(null);

  // Focus management
  useEffect(() => {
    if (isOpen && confirmButtonRef.current) {
      confirmButtonRef.current.focus();
    }
  }, [isOpen]);

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

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-50"
        onClick={isLoading ? undefined : onCancel}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        className="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirmation-dialog-title"
      >
        <div
          ref={dialogRef}
          className="rounded-lg shadow-xl max-w-md w-full"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div
            className="flex items-center justify-between px-4 py-3 border-b"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <h2
              id="confirmation-dialog-title"
              className="text-base font-semibold"
              style={{ color: 'var(--fgColor-default)' }}
            >
              {title}
            </h2>
            <IconButton
              icon={XIcon}
              aria-label="Close"
              variant="invisible"
              size="small"
              onClick={onCancel}
              disabled={isLoading}
            />
          </div>

          {/* Body */}
          <div className="p-4">
            <div
              className="text-sm"
              style={{ color: 'var(--fgColor-muted)' }}
            >
              {message}
            </div>
          </div>

          {/* Footer */}
          <div
            className="px-4 py-3 border-t flex justify-end gap-2"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <Button
              onClick={onCancel}
              disabled={isLoading}
            >
              {cancelLabel}
            </Button>
            <Button
              ref={confirmButtonRef}
              onClick={onConfirm}
              variant={variant}
              disabled={isLoading}
            >
              {isLoading ? 'Loading...' : confirmLabel}
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}

