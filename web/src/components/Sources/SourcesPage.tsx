import { useState } from 'react';
import { Flash, Spinner } from '@primer/react';
import { PlusIcon, AlertIcon } from '@primer/octicons-react';
import { useSources } from '../../contexts/SourceContext';
import { SourceCard } from './SourceCard';
import { SourceForm } from './SourceForm';
import { SourceDeletionDialog } from './SourceDeletionDialog';
import { FormDialog } from '../common/FormDialog';
import { Button } from '../common/buttons';
import type { Source, CreateSourceRequest, UpdateSourceRequest } from '../../types';

/**
 * Sources management page.
 * Displays all configured sources and allows CRUD operations.
 */
export function SourcesPage() {
  const {
    sources,
    isLoading,
    error,
    createSource,
    updateSource,
    deleteSource,
    validateSource,
    refetchSources,
  } = useSources();

  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<Source | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [deleteDialog, setDeleteDialog] = useState<{ isOpen: boolean; source: Source | null }>({
    isOpen: false,
    source: null,
  });

  const [validationResult, setValidationResult] = useState<{ sourceId: number; success: boolean; message: string } | null>(null);

  const handleOpenCreate = () => {
    setEditingSource(null);
    setFormError(null);
    setIsFormOpen(true);
  };

  const handleOpenEdit = (source: Source) => {
    setEditingSource(source);
    setFormError(null);
    setIsFormOpen(true);
  };

  const handleCloseForm = () => {
    setIsFormOpen(false);
    setEditingSource(null);
    setFormError(null);
  };

  const handleSubmit = async (data: CreateSourceRequest | UpdateSourceRequest) => {
    setIsSubmitting(true);
    setFormError(null);

    try {
      if (editingSource) {
        await updateSource(editingSource.id, data as UpdateSourceRequest);
      } else {
        await createSource(data as CreateSourceRequest);
      }
      handleCloseForm();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save source');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenDelete = (source: Source) => {
    setDeleteDialog({ isOpen: true, source });
  };

  const handleCloseDelete = () => {
    setDeleteDialog({ isOpen: false, source: null });
  };

  const handleConfirmDelete = async (force: boolean, confirmName?: string) => {
    if (!deleteDialog.source) return;

    await deleteSource(deleteDialog.source.id, force ? { force, confirm: confirmName } : undefined);
    handleCloseDelete();
  };

  const handleValidate = async (source: Source) => {
    // Validation started for source: source.id
    setValidationResult(null);

    try {
      const result = await validateSource({ source_id: source.id });
      setValidationResult({
        sourceId: source.id,
        success: result.valid,
        message: result.valid ? 'Connection successful!' : (result.error || 'Connection failed'),
      });
    } catch (err) {
      setValidationResult({
        sourceId: source.id,
        success: false,
        message: err instanceof Error ? err.message : 'Validation failed',
      });
    } finally {
      // Validation completed
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Spinner size="large" />
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: 'var(--fgColor-default)' }}>
            Migration Sources
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Configure and manage your GitHub and Azure DevOps sources
          </p>
        </div>
        <Button variant="primary" onClick={handleOpenCreate}>
          <PlusIcon /> Add Source
        </Button>
      </div>

      {/* Error state */}
      {error && (
        <Flash variant="danger" className="mb-4">
          <AlertIcon /> Failed to load sources: {error.message}
          <Button variant="invisible" size="small" onClick={refetchSources} className="ml-2">
            Retry
          </Button>
        </Flash>
      )}

      {/* Validation result */}
      {validationResult && (
        <Flash 
          variant={validationResult.success ? 'success' : 'danger'} 
          className="mb-4"
        >
          {validationResult.message}
        </Flash>
      )}

      {/* Empty state */}
      {sources.length === 0 ? (
        <div 
          className="rounded-lg border p-12 text-center"
          style={{ 
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <h2 className="text-lg font-semibold mb-2" style={{ color: 'var(--fgColor-default)' }}>
            No sources configured
          </h2>
          <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
            Add a GitHub or Azure DevOps source to start discovering repositories for migration.
          </p>
          <Button variant="primary" onClick={handleOpenCreate}>
            <PlusIcon /> Add Your First Source
          </Button>
        </div>
      ) : (
        /* Sources Grid */
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {sources.map((source) => (
            <SourceCard
              key={source.id}
              source={source}
              onEdit={handleOpenEdit}
              onDelete={handleOpenDelete}
              onValidate={handleValidate}
            />
          ))}
        </div>
      )}

      {/* Create/Edit Dialog */}
      <FormDialog
        isOpen={isFormOpen}
        title={editingSource ? 'Edit Source' : 'Add New Source'}
        submitLabel={editingSource ? 'Save Changes' : 'Create Source'}
        onSubmit={() => {}} // Form handles its own submission
        onCancel={handleCloseForm}
        isLoading={isSubmitting}
        size="large"
        hideFooter
      >
        {formError && (
          <Flash variant="danger" className="mb-4">
            {formError}
          </Flash>
        )}
        <SourceForm
          source={editingSource}
          onSubmit={handleSubmit}
          onCancel={handleCloseForm}
          isSubmitting={isSubmitting}
        />
      </FormDialog>

      {/* Delete Confirmation Dialog */}
      <SourceDeletionDialog
        isOpen={deleteDialog.isOpen}
        source={deleteDialog.source}
        onConfirm={handleConfirmDelete}
        onCancel={handleCloseDelete}
      />
    </div>
  );
}

