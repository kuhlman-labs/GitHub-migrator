import { useEffect } from 'react';
import { Button, TextInput, Flash, FormControl, Select } from '@primer/react';
import { FormDialog } from '../common/FormDialog';
import { useSourceContext } from '../../contexts/SourceContext';
import { SourceBadge } from '../common/SourceBadge';

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
  
  // Get the selected source object
  const selectedSource = selectedSourceId ? sources.find(s => s.id === selectedSourceId) : null;
  
  // Determine effective source type from selected source or prop
  const effectiveSourceType = selectedSource?.type || sourceType;
  
  // In "All Sources" mode, show all active sources
  // Otherwise, filter sources by the current source type
  const availableSources = isAllSourcesMode 
    ? sources.filter(s => s.is_active)
    : sources.filter(s => 
        s.is_active && (
          (sourceType === 'github' && s.type === 'github') ||
          (sourceType === 'azuredevops' && s.type === 'azuredevops')
        )
      );
  
  // In All Sources mode, a source must be selected
  const sourceSelected = !isAllSourcesMode || selectedSourceId != null;
  
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

  const handleSourceChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    const newSourceId = value ? parseInt(value, 10) : null;
    onSourceChange?.(newSourceId);
    
    // When source changes, reset discovery type to match the new source type
    // and pre-populate enterprise/organization fields if available in source config
    if (newSourceId) {
      const newSource = sources.find(s => s.id === newSourceId);
      if (newSource) {
        // Pre-populate fields from source configuration
        if (newSource.type === 'azuredevops') {
          // ADO: pre-populate organization (required field)
          setAdoOrganization(newSource.organization || '');
        } else if (newSource.type === 'github') {
          // GitHub: only pre-populate enterprise slug (organization is not a source-level config)
          setEnterpriseSlug(newSource.enterprise_slug || '');
          // Clear organization field for fresh input
          setOrganization('');
        }
        
        // Reset discovery type to match the new source type
        if (newSource.type === 'github' && (discoveryType === 'ado-org' || discoveryType === 'ado-project')) {
          setDiscoveryType('organization');
        } else if (newSource.type === 'azuredevops' && (discoveryType === 'organization' || discoveryType === 'enterprise')) {
          setDiscoveryType('ado-org');
        }
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

      {/* Source Selection - Required in All Sources mode, optional otherwise */}
      {isAllSourcesMode ? (
        <FormControl className="mb-3" required>
          <FormControl.Label>Select Source</FormControl.Label>
          <Select 
            value={selectedSourceId?.toString() || ''} 
            onChange={handleSourceChange}
            disabled={loading}
          >
            <Select.Option value="">Choose a source...</Select.Option>
            {availableSources.map(source => (
              <Select.Option key={source.id} value={source.id.toString()}>
                {source.name} ({source.repository_count} repos)
              </Select.Option>
            ))}
          </Select>
          <FormControl.Caption>
            Select which source to discover repositories from.
          </FormControl.Caption>
        </FormControl>
      ) : availableSources.length > 1 && onSourceChange ? (
        <FormControl className="mb-3">
          <FormControl.Label>Associate with Source</FormControl.Label>
          <Select 
            value={selectedSourceId?.toString() || ''} 
            onChange={handleSourceChange}
            disabled={loading}
          >
            <Select.Option value="">Default (use current config)</Select.Option>
            {availableSources.map(source => (
              <Select.Option key={source.id} value={source.id.toString()}>
                {source.name} ({source.repository_count} repos)
              </Select.Option>
            ))}
          </Select>
          <FormControl.Caption>
            Discovered repositories will be associated with this source.
          </FormControl.Caption>
        </FormControl>
      ) : null}
      
      {/* Show selected source info in All Sources mode */}
      {isAllSourcesMode && selectedSource && (
        <div className="mb-3 p-3 rounded-md" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
          <div className="flex items-center gap-2">
            <SourceBadge sourceType={selectedSource.type} sourceName={selectedSource.name} size="small" />
            <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              {selectedSource.type === 'github' ? 'GitHub' : 'Azure DevOps'} source
            </span>
          </div>
        </div>
      )}

      {/* Only show discovery type options when a source is selected (in All Sources mode) or always (in single source mode) */}
      {(!isAllSourcesMode || selectedSourceId) && (
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
      {(!isAllSourcesMode || selectedSourceId) && (
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
      {isAllSourcesMode && !selectedSourceId && (
        <div className="text-center py-4" style={{ color: 'var(--fgColor-muted)' }}>
          <p className="text-sm">Please select a source to configure discovery options.</p>
        </div>
      )}
    </FormDialog>
  );
}

