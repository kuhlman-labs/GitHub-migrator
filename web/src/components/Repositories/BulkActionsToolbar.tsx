import { useState } from 'react';
import { Button, TextInput, FormControl } from '@primer/react';
import { CheckIcon, XIcon, ChevronDownIcon } from '@primer/octicons-react';
import { useBatchUpdateRepositoryStatus } from '../../hooks/useMutations';
import { useToast } from '../../contexts/ToastContext';

interface BulkActionsToolbarProps {
  selectedCount: number;
  selectedIds: number[];
  onClearSelection: () => void;
}

type ActionType = 'mark_migrated' | 'mark_wont_migrate' | 'unmark_wont_migrate' | 'rollback';

interface ActionConfig {
  label: string;
  description: string;
  confirmTitle: string;
  confirmMessage: string;
  successMessage: string;
  action: ActionType;
}

const ACTIONS: Record<ActionType, ActionConfig> = {
  mark_migrated: {
    label: 'Mark as Migrated',
    description: 'For repositories migrated outside this system',
    confirmTitle: 'Mark Repositories as Migrated?',
    confirmMessage: 'This will mark the selected repositories as complete. Use this for repositories that were migrated through another process.',
    successMessage: 'repositories marked as migrated',
    action: 'mark_migrated',
  },
  mark_wont_migrate: {
    label: "Mark as Won't Migrate",
    description: 'Exclude from migration plans',
    confirmTitle: "Mark Repositories as Won't Migrate?",
    confirmMessage: 'This will exclude the selected repositories from all migration plans. They can be unmarked later if needed.',
    successMessage: "repositories marked as won't migrate",
    action: 'mark_wont_migrate',
  },
  unmark_wont_migrate: {
    label: "Unmark Won't Migrate",
    description: 'Reset to pending status',
    confirmTitle: 'Unmark Repositories?',
    confirmMessage: "This will change the selected repositories from 'won't migrate' back to pending status.",
    successMessage: 'repositories unmarked and reset to pending',
    action: 'unmark_wont_migrate',
  },
  rollback: {
    label: 'Rollback Migration',
    description: 'Rollback completed migrations',
    confirmTitle: 'Rollback Repositories?',
    confirmMessage: 'This will rollback the selected completed migrations. Only repositories with "complete" status can be rolled back.',
    successMessage: 'repositories rolled back',
    action: 'rollback',
  },
};

export function BulkActionsToolbar({ selectedCount, selectedIds, onClearSelection }: BulkActionsToolbarProps) {
  const [showActionMenu, setShowActionMenu] = useState(false);
  const [confirmDialog, setConfirmDialog] = useState<ActionConfig | null>(null);
  const [rollbackReason, setRollbackReason] = useState('');
  const batchUpdateMutation = useBatchUpdateRepositoryStatus();
  const { showToast } = useToast();

  const handleActionClick = (actionConfig: ActionConfig) => {
    setShowActionMenu(false);
    setConfirmDialog(actionConfig);
  };

  const handleConfirmAction = async () => {
    if (!confirmDialog) return;

    try {
      const result = await batchUpdateMutation.mutateAsync({
        repositoryIds: selectedIds,
        action: confirmDialog.action,
        reason: confirmDialog.action === 'rollback' ? rollbackReason : undefined,
      });

      setConfirmDialog(null);
      setRollbackReason('');
      onClearSelection();

      // Show success/partial success message
      if (result.failed_count === 0) {
        showToast(`Successfully updated ${result.updated_count} ${confirmDialog.successMessage}`, 'success');
      } else if (result.updated_count > 0) {
        showToast(
          `Partially completed: ${result.updated_count} of ${selectedCount} ${confirmDialog.successMessage}. ${result.failed_count} failed.`,
          'warning'
        );
      } else {
        showToast(`Failed to update repositories: ${result.errors?.[0] || 'Unknown error'}`, 'danger');
      }
    } catch (error: unknown) {
      setConfirmDialog(null);
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      showToast(
        `Failed to update repositories: ${err.response?.data?.error || err.message || 'Unknown error'}`,
        'danger'
      );
    }
  };

  return (
    <>
      {/* Floating Toolbar */}
      <div
        className="fixed bottom-8 left-1/2 transform -translate-x-1/2 z-50 shadow-2xl rounded-lg border animate-slide-up"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderColor: 'var(--borderColor-default)',
          boxShadow: 'var(--shadow-floating-large)',
          maxWidth: '90vw',
        }}
      >
        <div className="flex items-center gap-4 px-6 py-4">
          <div className="flex items-center gap-2">
            <span style={{ color: 'var(--fgColor-accent)' }}>
              <CheckIcon size={20} />
            </span>
            <span className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>
              {selectedCount} {selectedCount === 1 ? 'repository' : 'repositories'} selected
            </span>
          </div>

          <div className="h-6 w-px" style={{ backgroundColor: 'var(--borderColor-default)' }} />

          {/* Actions Dropdown */}
          <div className="relative">
            <Button
              onClick={() => setShowActionMenu(!showActionMenu)}
              trailingVisual={ChevronDownIcon}
              variant="primary"
              disabled={batchUpdateMutation.isPending}
            >
              Actions
            </Button>

            {showActionMenu && (
              <>
                {/* Backdrop */}
                <div
                  className="fixed inset-0 z-10"
                  onClick={() => setShowActionMenu(false)}
                />
                {/* Menu */}
                <div
                  className="absolute bottom-full mb-2 left-0 w-72 rounded-lg shadow-lg z-20"
                  style={{
                    backgroundColor: 'var(--bgColor-default)',
                    border: '1px solid var(--borderColor-default)',
                    boxShadow: 'var(--shadow-floating-large)',
                  }}
                >
                  <div className="py-1">
                    {Object.values(ACTIONS).map((actionConfig) => (
                      <button
                        key={actionConfig.action}
                        onClick={() => handleActionClick(actionConfig)}
                        className="w-full text-left px-4 py-3 transition-colors hover:bg-[var(--control-bgColor-hover)]"
                        style={{ color: 'var(--fgColor-default)' }}
                      >
                        <div className="font-medium">{actionConfig.label}</div>
                        <div className="text-xs mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                          {actionConfig.description}
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>

          <Button onClick={onClearSelection} variant="invisible" leadingVisual={XIcon}>
            Clear Selection
          </Button>
        </div>
      </div>

      {/* Confirmation Dialog */}
      {confirmDialog && (
        <>
          {/* Backdrop */}
          <div 
            className="fixed inset-0 bg-black/50 z-50"
            onClick={() => {
              if (!batchUpdateMutation.isPending) {
                setConfirmDialog(null);
                setRollbackReason('');
              }
            }}
          />
          
          {/* Dialog */}
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div 
              className="rounded-lg shadow-xl max-w-md w-full"
              style={{ backgroundColor: 'var(--bgColor-default)' }}
              onClick={(e) => e.stopPropagation()}
            >
              <div className="px-4 py-3 border-b border-gh-border-default">
                <h3 className="text-base font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  {confirmDialog.confirmTitle}
                </h3>
              </div>
              
              <div className="p-4">
                <p className="text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
                  {confirmDialog.confirmMessage}
                </p>
                <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
                  <strong>{selectedCount}</strong> {selectedCount === 1 ? 'repository' : 'repositories'} will be
                  affected.
                </p>
                {confirmDialog.action === 'rollback' && (
                  <FormControl>
                    <FormControl.Label>Reason (optional)</FormControl.Label>
                    <TextInput
                      value={rollbackReason}
                      onChange={(e) => setRollbackReason(e.target.value)}
                      placeholder="e.g., Migration issues, incorrect destination, etc."
                      disabled={batchUpdateMutation.isPending}
                      block
                    />
                    <FormControl.Caption>
                      Provide a reason for the rollback to help with tracking and auditing.
                    </FormControl.Caption>
                  </FormControl>
                )}
              </div>
              
              <div className="px-4 py-3 border-t border-gh-border-default flex justify-end gap-2">
                <Button
                  onClick={() => {
                    setConfirmDialog(null);
                    setRollbackReason('');
                  }}
                  disabled={batchUpdateMutation.isPending}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleConfirmAction}
                  variant="primary"
                  disabled={batchUpdateMutation.isPending}
                >
                  {batchUpdateMutation.isPending ? 'Processing...' : 'Confirm'}
                </Button>
              </div>
            </div>
          </div>
        </>
      )}
    </>
  );
}
