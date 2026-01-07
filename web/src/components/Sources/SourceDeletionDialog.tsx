import { useState, useRef, useEffect } from 'react';
import { Flash, Spinner } from '@primer/react';
import { AlertIcon, TrashIcon, RepoIcon, PeopleIcon, OrganizationIcon, HistoryIcon, XIcon } from '@primer/octicons-react';
import { Button, IconButton } from '../common/buttons';
import type { Source, SourceDeletionPreview } from '../../types/source';
import { sourcesApi } from '../../services/api/sources';

export interface SourceDeletionDialogProps {
  isOpen: boolean;
  source: Source | null;
  onConfirm: (force: boolean, confirmName?: string) => Promise<void>;
  onCancel: () => void;
}

type DialogStep = 'initial' | 'preview' | 'confirm';

/**
 * Multi-step source deletion dialog similar to GitHub's repository deletion flow.
 * 
 * Flow:
 * 1. Initial warning - shows basic confirmation if source has no data
 * 2. Preview - shows counts of all data that will be deleted  
 * 3. Confirm - requires typing the source name to confirm cascade deletion
 */
export function SourceDeletionDialog({
  isOpen,
  source,
  onConfirm,
  onCancel,
}: SourceDeletionDialogProps) {
  const [step, setStep] = useState<DialogStep>('initial');
  const [preview, setPreview] = useState<SourceDeletionPreview | null>(null);
  const [isLoadingPreview, setIsLoadingPreview] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [confirmationText, setConfirmationText] = useState('');
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  
  const dialogRef = useRef<HTMLDivElement>(null);
  const confirmInputRef = useRef<HTMLInputElement>(null);

  // Reset state when dialog opens/closes or source changes
  useEffect(() => {
    if (isOpen && source) {
      setStep('initial');
      setPreview(null);
      setPreviewError(null);
      setConfirmationText('');
      setDeleteError(null);
      setIsDeleting(false);
    }
  }, [isOpen, source]);

  // Focus confirm input when entering confirm step
  useEffect(() => {
    if (step === 'confirm' && confirmInputRef.current) {
      confirmInputRef.current.focus();
    }
  }, [step]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen && !isDeleting) {
        onCancel();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isDeleting, onCancel]);

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

  const loadPreview = async () => {
    if (!source) return;
    
    setIsLoadingPreview(true);
    setPreviewError(null);
    
    try {
      const data = await sourcesApi.getDeletionPreview(source.id);
      setPreview(data);
      setStep('preview');
    } catch (err) {
      setPreviewError(err instanceof Error ? err.message : 'Failed to load deletion preview');
    } finally {
      setIsLoadingPreview(false);
    }
  };

  const handleInitialContinue = () => {
    // If source has no data, go straight to confirm step
    if (source && source.repository_count === 0) {
      setStep('confirm');
    } else {
      loadPreview();
    }
  };

  const handleDelete = async () => {
    if (!source) return;
    
    setIsDeleting(true);
    setDeleteError(null);
    
    try {
      const hasData = preview && preview.total_affected_records > 0;
      await onConfirm(hasData || false, hasData ? confirmationText : undefined);
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete source');
      setIsDeleting(false);
    }
  };

  const isConfirmationValid = source && confirmationText === source.name;
  const hasRelatedData = preview && preview.total_affected_records > 0;

  if (!isOpen || !source) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-50"
        onClick={isDeleting ? undefined : onCancel}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        className="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="deletion-dialog-title"
      >
        <div
          ref={dialogRef}
          className="rounded-lg shadow-xl max-w-lg w-full"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div
            className="flex items-center justify-between px-4 py-3 border-b"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <div className="flex items-center gap-2">
              <AlertIcon size={16} className="text-[var(--fgColor-danger)]" />
              <h2
                id="deletion-dialog-title"
                className="text-base font-semibold"
                style={{ color: 'var(--fgColor-default)' }}
              >
                Delete Source
              </h2>
            </div>
            <IconButton
              icon={XIcon}
              aria-label="Close"
              variant="invisible"
              size="small"
              onClick={onCancel}
              disabled={isDeleting}
            />
          </div>

          {/* Body */}
          <div className="p-4 space-y-4">
            {/* Error messages */}
            {(previewError || deleteError) && (
              <Flash variant="danger">
                {previewError || deleteError}
              </Flash>
            )}

            {/* Step: Initial Warning */}
            {step === 'initial' && (
              <>
                <Flash variant="warning">
                  <AlertIcon size={16} />
                  <span className="ml-2">This action cannot be undone.</span>
                </Flash>
                
                <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                  Are you sure you want to delete the source{' '}
                  <strong style={{ color: 'var(--fgColor-default)' }}>"{source.name}"</strong>?
                </p>

                {source.repository_count > 0 && (
                  <div
                    className="rounded-lg p-3"
                    style={{ backgroundColor: 'var(--bgColor-attention-muted)' }}
                  >
                    <p className="text-sm font-medium" style={{ color: 'var(--fgColor-attention)' }}>
                      This source has {source.repository_count} associated {source.repository_count === 1 ? 'repository' : 'repositories'}.
                    </p>
                    <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                      Deleting this source will also remove all associated data.
                    </p>
                  </div>
                )}
              </>
            )}

            {/* Loading preview */}
            {isLoadingPreview && (
              <div className="flex items-center justify-center py-8">
                <Spinner size="medium" />
                <span className="ml-2 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                  Loading deletion preview...
                </span>
              </div>
            )}

            {/* Step: Preview */}
            {step === 'preview' && preview && !isLoadingPreview && (
              <>
                <Flash variant="danger">
                  <AlertIcon size={16} />
                  <span className="ml-2">
                    Deleting this source will permanently remove all associated data.
                  </span>
                </Flash>

                <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                  Deleting <strong style={{ color: 'var(--fgColor-default)' }}>"{source.name}"</strong> will remove:
                </p>

                <div
                  className="rounded-lg border p-4 space-y-3"
                  style={{
                    backgroundColor: 'var(--bgColor-muted)',
                    borderColor: 'var(--borderColor-default)',
                  }}
                >
                  {preview.repository_count > 0 && (
                    <div className="flex items-center gap-2">
                      <RepoIcon size={16} className="text-[var(--fgColor-muted)]" />
                      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                        {preview.repository_count.toLocaleString()}
                      </span>
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        {preview.repository_count === 1 ? 'repository' : 'repositories'}
                      </span>
                    </div>
                  )}
                  
                  {preview.user_count > 0 && (
                    <div className="flex items-center gap-2">
                      <PeopleIcon size={16} className="text-[var(--fgColor-muted)]" />
                      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                        {preview.user_count.toLocaleString()}
                      </span>
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        {preview.user_count === 1 ? 'user' : 'users'}
                      </span>
                    </div>
                  )}
                  
                  {preview.team_count > 0 && (
                    <div className="flex items-center gap-2">
                      <OrganizationIcon size={16} className="text-[var(--fgColor-muted)]" />
                      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                        {preview.team_count.toLocaleString()}
                      </span>
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        {preview.team_count === 1 ? 'team' : 'teams'}
                      </span>
                    </div>
                  )}
                  
                  {preview.migration_history_count > 0 && (
                    <div className="flex items-center gap-2">
                      <HistoryIcon size={16} className="text-[var(--fgColor-muted)]" />
                      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                        {preview.migration_history_count.toLocaleString()}
                      </span>
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        migration history {preview.migration_history_count === 1 ? 'record' : 'records'}
                      </span>
                    </div>
                  )}

                  {preview.user_mapping_count > 0 && (
                    <div className="flex items-center gap-2">
                      <PeopleIcon size={16} className="text-[var(--fgColor-muted)]" />
                      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                        {preview.user_mapping_count.toLocaleString()}
                      </span>
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        user {preview.user_mapping_count === 1 ? 'mapping' : 'mappings'}
                      </span>
                    </div>
                  )}

                  {preview.team_mapping_count > 0 && (
                    <div className="flex items-center gap-2">
                      <OrganizationIcon size={16} className="text-[var(--fgColor-muted)]" />
                      <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                        {preview.team_mapping_count.toLocaleString()}
                      </span>
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                        team {preview.team_mapping_count === 1 ? 'mapping' : 'mappings'}
                      </span>
                    </div>
                  )}

                  <div
                    className="pt-2 mt-2 border-t"
                    style={{ borderColor: 'var(--borderColor-muted)' }}
                  >
                    <div className="flex items-center gap-2">
                      <TrashIcon size={16} className="text-[var(--fgColor-danger)]" />
                      <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-danger)' }}>
                        {preview.total_affected_records.toLocaleString()} total records will be deleted
                      </span>
                    </div>
                  </div>
                </div>
              </>
            )}

            {/* Step: Confirm */}
            {step === 'confirm' && (
              <>
                {hasRelatedData && (
                  <Flash variant="danger">
                    <AlertIcon size={16} />
                    <span className="ml-2">
                      This will permanently delete {preview?.total_affected_records.toLocaleString()} records.
                    </span>
                  </Flash>
                )}

                <div>
                  <label
                    htmlFor="confirm-source-name"
                    className="block text-sm font-medium mb-2"
                    style={{ color: 'var(--fgColor-default)' }}
                  >
                    To confirm, type <strong>"{source.name}"</strong> below:
                  </label>
                  <input
                    ref={confirmInputRef}
                    id="confirm-source-name"
                    type="text"
                    value={confirmationText}
                    onChange={(e) => setConfirmationText(e.target.value)}
                    placeholder={source.name}
                    disabled={isDeleting}
                    className="w-full px-3 py-2 text-sm rounded-md border focus:outline-none focus:ring-2"
                    style={{
                      backgroundColor: 'var(--bgColor-default)',
                      borderColor: confirmationText && !isConfirmationValid 
                        ? 'var(--borderColor-danger)' 
                        : 'var(--borderColor-default)',
                      color: 'var(--fgColor-default)',
                    }}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && isConfirmationValid && !isDeleting) {
                        handleDelete();
                      }
                    }}
                  />
                </div>
              </>
            )}
          </div>

          {/* Footer */}
          <div
            className="px-4 py-3 border-t flex justify-end gap-2"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <Button onClick={onCancel} disabled={isDeleting}>
              Cancel
            </Button>

            {step === 'initial' && (
              <Button
                variant="danger"
                onClick={handleInitialContinue}
                disabled={isLoadingPreview}
              >
                {isLoadingPreview ? 'Loading...' : 'Continue'}
              </Button>
            )}

            {step === 'preview' && preview && (
              <Button
                variant="danger"
                onClick={() => setStep('confirm')}
              >
                I understand, continue
              </Button>
            )}

            {step === 'confirm' && (
              <Button
                variant="danger"
                onClick={handleDelete}
                disabled={!isConfirmationValid || isDeleting}
              >
                {isDeleting ? (
                  <>
                    <Spinner size="small" />
                    <span className="ml-2">Deleting...</span>
                  </>
                ) : (
                  'Delete Source'
                )}
              </Button>
            )}
          </div>
        </div>
      </div>
    </>
  );
}

