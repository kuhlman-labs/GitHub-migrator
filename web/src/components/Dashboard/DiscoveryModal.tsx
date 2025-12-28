import { Button, TextInput, Flash, FormControl } from '@primer/react';
import { FormDialog } from '../common/FormDialog';

export type DiscoveryType = 'organization' | 'enterprise' | 'ado-org' | 'ado-project';

export interface DiscoveryModalProps {
  isOpen: boolean;
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
  onStart: () => void;
  onClose: () => void;
}

/**
 * Modal for starting repository discovery.
 * Supports GitHub (organization/enterprise) and Azure DevOps (organization/project).
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
}: DiscoveryModalProps) {
  const isFormValid =
    (discoveryType === 'organization' && organization.trim()) ||
    (discoveryType === 'enterprise' && enterpriseSlug.trim()) ||
    (discoveryType === 'ado-org' && adoOrganization.trim()) ||
    (discoveryType === 'ado-project' && adoOrganization.trim() && adoProject.trim());

  return (
    <FormDialog
      isOpen={isOpen}
      title="Start Repository Discovery"
      submitLabel={loading ? 'Starting...' : 'Start Discovery'}
      onSubmit={onStart}
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

      <FormControl className="mb-3">
        <FormControl.Label>Discovery Type</FormControl.Label>
        <div className="flex gap-2">
          {sourceType === 'github' ? (
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
    </FormDialog>
  );
}

