import { useEffect } from 'react';
import { Button, TextInput, Flash, FormControl } from '@primer/react';
import { FormDialog } from '../common/FormDialog';
import { DiscoverySourceSelector, useSourceSelection } from '../common/DiscoverySourceSelector';
import { useSourceContext } from '../../contexts/SourceContext';

export type DiscoveryType = 'organization' | 'enterprise' | 'ado-org' | 'ado-project';

export interface DiscoveryModalProps {
  isOpen: boolean;
  /** The source type - can be derived from selected source when in All Sources mode */
  sourceType: 'github' | 'azuredevops';
  discoveryType: DiscoveryType;
  setDiscoveryType: (type: DiscoveryType | null) => void;
  organization: string;
  setOrganization: (org: string) => void;
  enterpriseSlug: string;
  setEnterpriseSlug: (slug: string) => void;
  adoOrganization: string;
  setAdoOrganization: (org: string) => void;
  adoProject: string;
  setAdoProject: (project: string) => void;
  loading: boolean;
  error: string | null;
  onStart: (sourceId?: number) => void;
  onClose: () => void;
  /** Optional: pre-selected source ID */
  selectedSourceId?: number | null;
  /** Optional: callback when source selection changes */
  onSourceChange?: (sourceId: number | null) => void;
  /** Whether "All Sources" is selected (no active source filter) */
  isAllSourcesMode?: boolean;
}

/**
 * Modal for starting repository discovery.
 * Supports GitHub (organization/enterprise) and Azure DevOps (organization/project).
 * When in "All Sources" mode, shows a source selector to pick which source to discover from.
 */
export function DiscoveryModal({
  isOpen,
  sourceType,
  discoveryType,
  setDiscoveryType,
  organization,
  setOrganization,
  enterpriseSlug,
  setEnterpriseSlug,
  adoOrganization,
  setAdoOrganization,
  adoProject,
  setAdoProject,
  loading,
  error,
  onStart,
  onClose,
  selectedSourceId,
  onSourceChange,
  isAllSourcesMode = false,
}: DiscoveryModalProps) {
  const { sources } = useSourceContext();
  const { isAllSourcesMode: contextIsAllSourcesMode } = useSourceSelection();
  
  // Use prop if provided, otherwise fall back to context
  const effectiveIsAllSourcesMode = isAllSourcesMode ?? contextIsAllSourcesMode;
  
  // Get the selected source object
  const selectedSource = selectedSourceId ? sources.find(s => s.id === selectedSourceId) : null;
  
  // Determine effective source type from selected source or prop
  const effectiveSourceType = selectedSource?.type || sourceType;
  
  // In All Sources mode, a source must be selected
  const sourceSelected = !effectiveIsAllSourcesMode || selectedSourceId != null;
  
  const isFormValid = sourceSelected && (
    (discoveryType === 'organization' && organization.trim()) ||
    (discoveryType === 'enterprise' && enterpriseSlug.trim()) ||
    (discoveryType === 'ado-org' && adoOrganization.trim()) ||
    (discoveryType === 'ado-project' && adoOrganization.trim() && adoProject.trim())
  );

  // Pre-populate organization/enterprise fields when modal opens or source changes
  useEffect(() => {
    if (isOpen && selectedSource) {
      // Only pre-populate if the field is currently empty to avoid overwriting user input
      if (selectedSource.type === 'azuredevops' && selectedSource.organization && !adoOrganization) {
        setAdoOrganization(selectedSource.organization);
      } else if (selectedSource.type === 'github') {
        // For GitHub, only pre-populate enterprise slug (not organization)
        if (selectedSource.enterprise_slug && !enterpriseSlug) {
          setEnterpriseSlug(selectedSource.enterprise_slug);
        }
      }
    }
  }, [isOpen, selectedSource, adoOrganization, enterpriseSlug, setAdoOrganization, setEnterpriseSlug]);

  // Handle source change from DiscoverySourceSelector
  const handleSourceChange = (sourceId: number | null, source: import('../../types').Source | null) => {
    onSourceChange?.(sourceId);
    
    if (source) {
      // Pre-populate fields from source configuration
      if (source.type === 'azuredevops') {
        setAdoOrganization(source.organization || '');
      } else if (source.type === 'github') {
        setEnterpriseSlug(source.enterprise_slug || '');
        setOrganization('');
      }
      
      // Reset discovery type to match the new source type
      if (source.type === 'github' && (discoveryType === 'ado-org' || discoveryType === 'ado-project')) {
        setDiscoveryType('enterprise');
      } else if (source.type === 'azuredevops' && (discoveryType === 'organization' || discoveryType === 'enterprise')) {
        setDiscoveryType('ado-org');
      }
    } else {
      // Clear all fields when no source is selected
      setOrganization('');
      setEnterpriseSlug('');
      setAdoOrganization('');
      setAdoProject('');
    }
  };
  
  const handleStart = () => {
    onStart(selectedSourceId ?? undefined);
  };

  return (
    <FormDialog
      isOpen={isOpen}
      title="Start Repository Discovery"
      submitLabel={loading ? 'Starting...' : 'Start Discovery'}
      onSubmit={handleStart}
      onCancel={onClose}
      isLoading={loading}
      isSubmitDisabled={!isFormValid}
      size="large"
    >
      {error && (
        <Flash variant="danger" className="mb-3">
          {error}
        </Flash>
      )}

      {/* Source Selection */}
      <DiscoverySourceSelector
        selectedSourceId={selectedSourceId ?? null}
        onSourceChange={handleSourceChange}
        required={effectiveIsAllSourcesMode}
        disabled={loading}
        label="Select Source"
        defaultCaption="Select which source to discover repositories from."
        showRepoCount={true}
      />

      {/* Only show discovery type options when a source is selected (in All Sources mode) or always (in single source mode) */}
      {(!effectiveIsAllSourcesMode || selectedSourceId) && (
        <FormControl className="mb-3">
          <FormControl.Label>Discovery Type</FormControl.Label>
          <div className="flex gap-2">
            {effectiveSourceType === 'github' ? (
              <>
                <Button
                  type="button"
                  variant={discoveryType === 'organization' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('organization')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Organization
                </Button>
                <Button
                  type="button"
                  variant={discoveryType === 'enterprise' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('enterprise')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Enterprise
                </Button>
              </>
            ) : (
              <>
                <Button
                  type="button"
                  variant={discoveryType === 'ado-org' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('ado-org')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Organization
                </Button>
                <Button
                  type="button"
                  variant={discoveryType === 'ado-project' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('ado-project')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Project
                </Button>
              </>
            )}
          </div>
        </FormControl>
      )}

      {/* Only show input fields when a source is selected (in All Sources mode) or always (in single source mode) */}
      {(!effectiveIsAllSourcesMode || selectedSourceId) && (
        <>
          {discoveryType === 'organization' && (
            <FormControl className="mb-3" required>
              <FormControl.Label>Organization Name</FormControl.Label>
              <TextInput
                value={organization}
                onChange={(e) => setOrganization(e.target.value)}
                placeholder="e.g., your-github-org"
                disabled={loading}
                autoFocus
                required
              />
              <FormControl.Caption>
                Enter the GitHub organization name to discover all repositories.
              </FormControl.Caption>
            </FormControl>
          )}

          {discoveryType === 'enterprise' && (
            <FormControl className="mb-3" required>
              <FormControl.Label>Enterprise Slug</FormControl.Label>
              <TextInput
                value={enterpriseSlug}
                onChange={(e) => setEnterpriseSlug(e.target.value)}
                placeholder="e.g., your-enterprise-slug"
                disabled={loading}
                autoFocus
                required
              />
              <FormControl.Caption>
                Enter the GitHub Enterprise slug to discover repositories across all
                organizations.
              </FormControl.Caption>
            </FormControl>
          )}

          {discoveryType === 'ado-org' && (
            <FormControl className="mb-3" required>
              <FormControl.Label>Azure DevOps Organization</FormControl.Label>
              <TextInput
                value={adoOrganization}
                onChange={(e) => setAdoOrganization(e.target.value)}
                placeholder="e.g., your-ado-org"
                disabled={loading}
                autoFocus
                required
              />
              <FormControl.Caption>
                Discover all projects and repositories in this Azure DevOps organization.
              </FormControl.Caption>
            </FormControl>
          )}

          {discoveryType === 'ado-project' && (
            <div className="space-y-3 mb-3">
              <FormControl required>
                <FormControl.Label>Azure DevOps Organization</FormControl.Label>
                <TextInput
                  value={adoOrganization}
                  onChange={(e) => setAdoOrganization(e.target.value)}
                  placeholder="e.g., your-ado-org"
                  disabled={loading}
                  required
                />
              </FormControl>
              <FormControl required>
                <FormControl.Label>Project Name</FormControl.Label>
                <TextInput
                  value={adoProject}
                  onChange={(e) => setAdoProject(e.target.value)}
                  placeholder="e.g., your-project"
                  disabled={loading}
                  autoFocus
                  required
                />
                <FormControl.Caption>
                  Discover repositories in a specific Azure DevOps project.
                </FormControl.Caption>
              </FormControl>
            </div>
          )}
        </>
      )}
      
      {/* Prompt to select a source first in All Sources mode */}
      {effectiveIsAllSourcesMode && !selectedSourceId && (
        <div className="text-center py-4" style={{ color: 'var(--fgColor-muted)' }}>
          <p className="text-sm">Please select a source to configure discovery options.</p>
        </div>
      )}
    </FormDialog>
  );
}

