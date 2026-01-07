import { useState } from 'react';
import { Heading, Text, Flash, Spinner } from '@primer/react';
import { PlusIcon, RepoIcon, AlertIcon } from '@primer/octicons-react';
import { useSources } from '../../contexts/SourceContext';
import { SourceCard } from '../Sources/SourceCard';
import { SourceForm } from '../Sources/SourceForm';
import { SourceDeletionDialog } from '../Sources/SourceDeletionDialog';
import { FormDialog } from '../common/FormDialog';
import { PrimaryButton, Button } from '../common/buttons';
import type { Source, CreateSourceRequest, UpdateSourceRequest } from '../../types';

/**
 * Sources configuration panel for the Settings page.
 * Allows managing migration sources (GitHub/Azure DevOps) inline with other settings.
 */
interface SourcesSettingsProps {
  readOnly?: boolean;
}

export function SourcesSettings({ readOnly = false }: SourcesSettingsProps) {
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
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner size="large" />
      </div>
    );
  }

  if (error) {
    return (
      <Flash variant="danger">
        <AlertIcon /> Failed to load sources: {error.message}
        <Button variant="invisible" size="small" onClick={refetchSources} className="ml-2">
          Retry
        </Button>
      </Flash>
    );
  }

  return (
    <div className="max-w-4xl">
      <div className="flex items-center justify-between mb-2">
        <Heading as="h2" className="text-lg">Migration Sources</Heading>
        {!readOnly && (
        <PrimaryButton onClick={handleOpenCreate} leadingVisual={PlusIcon}>
          Add Source
        </PrimaryButton>
        )}
      </div>
      <Text className="block mb-6" style={{ color: 'var(--fgColor-muted)' }}>
        Configure GitHub or Azure DevOps sources to discover repositories for migration.
      </Text>

      {/* Validation result */}
      {validationResult && (
        <Flash 
          variant={validationResult.success ? 'success' : 'danger'} 
          className="mb-4"
        >
          {validationResult.message}
        </Flash>
      )}

      {/* Sources List */}
      {sources.length === 0 ? (
        <div 
          className="rounded-lg border p-12 text-center"
          style={{ 
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <div className="flex justify-center mb-4" style={{ color: 'var(--fgColor-muted)' }}>
            <RepoIcon size={40} />
          </div>
          <h3 className="text-lg font-semibold mb-2" style={{ color: 'var(--fgColor-default)' }}>
            No sources configured
          </h3>
          <p className="text-sm mb-6" style={{ color: 'var(--fgColor-muted)' }}>
            Add a GitHub or Azure DevOps source to start discovering repositories for migration.
          </p>
          <div className="flex justify-center">
            <PrimaryButton onClick={handleOpenCreate} leadingVisual={PlusIcon}>
              Add Your First Source
            </PrimaryButton>
          </div>
        </div>
      ) : (
        <div className="grid gap-4 grid-cols-1 lg:grid-cols-2">
          {sources.map((source) => (
            <SourceCard
              key={source.id}
              source={source}
              onEdit={handleOpenEdit}
              onDelete={handleOpenDelete}
              onValidate={handleValidate}
              readOnly={readOnly}
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
