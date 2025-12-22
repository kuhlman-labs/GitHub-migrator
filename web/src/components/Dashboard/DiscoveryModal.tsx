import { Button, TextInput, Flash, FormControl } from '@primer/react';
import { XIcon } from '@primer/octicons-react';

export type DiscoveryType = 'organization' | 'enterprise' | 'ado-org' | 'ado-project';

export interface DiscoveryModalProps {
  isOpen: boolean;
  sourceType: 'github' | 'azuredevops';
  discoveryType: DiscoveryType;
  setDiscoveryType: (type: DiscoveryType) => void;
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
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onStart();
  };

  const isFormValid =
    (discoveryType === 'organization' && organization.trim()) ||
    (discoveryType === 'enterprise' && enterpriseSlug.trim()) ||
    (discoveryType === 'ado-org' && adoOrganization.trim()) ||
    (discoveryType === 'ado-project' && adoOrganization.trim() && adoProject.trim());

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop overlay */}
      <div
        className="fixed inset-0 bg-black/50 z-40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div
        className="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="discovery-modal-title"
      >
        <div
          className="rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-auto"
          style={{ backgroundColor: 'var(--bgColor-default)' }}
          onClick={(e) => e.stopPropagation()}
        >
          <div
            className="flex items-center justify-between p-4 border-b"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <h2
              id="discovery-modal-title"
              className="text-xl font-semibold"
              style={{ color: 'var(--fgColor-default)' }}
            >
              Start Repository Discovery
            </h2>
            <button
              onClick={onClose}
              className="p-1 rounded transition-colors hover:bg-[var(--control-bgColor-hover)]"
              style={{ color: 'var(--fgColor-muted)' }}
              aria-label="Close"
            >
              <XIcon size={20} />
            </button>
          </div>

          <form onSubmit={handleSubmit} className="p-4">
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

            <div
              className="flex justify-end gap-2 pt-4 border-t"
              style={{ borderColor: 'var(--borderColor-default)' }}
            >
              <Button type="button" onClick={onClose} disabled={loading}>
                Cancel
              </Button>
              <Button type="submit" variant="primary" disabled={loading || !isFormValid}>
                {loading ? 'Starting...' : 'Start Discovery'}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
}

