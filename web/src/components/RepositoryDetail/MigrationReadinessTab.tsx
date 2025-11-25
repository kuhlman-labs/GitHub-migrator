import { useState, useEffect, useRef } from 'react';
import { Button, Checkbox, TextInput, FormControl, Select, Dialog } from '@primer/react';
import { XCircleFillIcon, AlertIcon, ChevronDownIcon, InfoIcon, XIcon, CheckCircleFillIcon } from '@primer/octicons-react';
import type { Repository, Batch } from '../../types';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import { Badge } from '../common/Badge';
import { ComplexityInfoModal } from '../common/ComplexityInfoModal';
import { useUpdateRepository } from '../../hooks/useMutations';
import { formatBytes } from '../../utils/format';
import { useToast } from '../../contexts/ToastContext';

interface MigrationReadinessTabProps {
  repository: Repository;
  allBatches: Batch[];
}

export function MigrationReadinessTab({ 
  repository, 
  allBatches
}: MigrationReadinessTabProps) {
  const queryClient = useQueryClient();
  const updateRepositoryMutation = useUpdateRepository();
  const { showSuccess, showError } = useToast();
  
  // Batch assignment state - show pending and ready batches
  const batches = allBatches.filter(b => b.status === 'pending' || b.status === 'ready');
  const [selectedBatchId, setSelectedBatchId] = useState<number | null>(null);
  const [assigningBatch, setAssigningBatch] = useState(false);
  
  // Dialog state
  const [showRemoveDialog, setShowRemoveDialog] = useState(false);
  const removeButtonRef = useRef<HTMLButtonElement>(null);
  
  // Destination configuration
  
  // Helper to sanitize names for GitHub (replace spaces with hyphens)
  const sanitizeForGitHub = (name: string): string => {
    return name.replace(/\s+/g, '-');
  };
  
  // Calculate the suggested default (ignoring any saved custom destination)
  const getSuggestedDefault = () => {
    // If it's an ADO repo (has ado_project), transform to GitHub-compatible format
    if (repository.ado_project) {
      // ADO format: org/project/repo -> GitHub format: org-project/repo
      // Replace spaces with hyphens for GitHub compatibility
      const parts = repository.full_name.split('/');
      if (parts.length >= 3) {
        const [org, project, ...repoParts] = parts;
        const sanitizedOrg = sanitizeForGitHub(org);
        const sanitizedProject = sanitizeForGitHub(project);
        const sanitizedRepo = repoParts.map(sanitizeForGitHub).join('/');
        return `${sanitizedOrg}-${sanitizedProject}/${sanitizedRepo}`;
      }
    }
    
    // Default: use full_name as is
    return repository.full_name;
  };
  
  // Get the current destination (saved custom value or suggested default)
  const getDefaultDestination = () => {
    if (repository.destination_full_name) {
      return repository.destination_full_name;
    }
    return getSuggestedDefault();
  };
  
  const [destinationFullName, setDestinationFullName] = useState<string>(
    getDefaultDestination()
  );

  // Sync destinationFullName with repository data when it changes
  useEffect(() => {
      setDestinationFullName(getDefaultDestination());
  }, [repository.destination_full_name, repository.full_name, repository.ado_project]);

  // Migration options state
  const [excludeReleases, setExcludeReleases] = useState(repository.exclude_releases);
  const [savingOptions, setSavingOptions] = useState(false);
  const hasOptionsChanges = excludeReleases !== repository.exclude_releases;
  const [showMigrationOptionsInfo, setShowMigrationOptionsInfo] = useState(false);

  // Validation state
  const [expandedValidation, setExpandedValidation] = useState(false);
  
  // Determine validation status
  const hasBlockingIssues = repository.has_oversized_repository || 
    repository.has_oversized_commits || 
    repository.has_long_refs || 
    repository.has_blocking_files;
  const hasWarnings = (repository.estimated_metadata_size && repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024) || 
    repository.has_large_file_warnings;

  const canMigrate = ['pending', 'dry_run_complete', 'dry_run_failed', 'pre_migration_complete', 'migration_failed', 'rolled_back'].includes(
    repository.status
  );

  const isInActiveMigration = [
    'queued_for_migration',
    'dry_run_in_progress',
    'dry_run_queued',
    'migrating_content',
    'pre_migration',
    'archive_generating',
    'post_migration',
  ].includes(repository.status);

  const canChangeBatch = !isInActiveMigration && repository.status !== 'complete';
  
  // Find the current batch name
  const currentBatch = repository.batch_id 
    ? allBatches.find(b => b.id === repository.batch_id)
    : null;

  const handleSaveDestination = async () => {
    // Validate format
    if (!destinationFullName.includes('/')) {
      showError('Destination must be in "organization/repository" format');
      return;
    }

    try {
      await updateRepositoryMutation.mutateAsync({
        fullName: repository.full_name,
        updates: { destination_full_name: destinationFullName },
      });
      
      showSuccess('Destination saved successfully!');
    } catch (error: any) {
      console.error('Failed to save destination:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to save destination. Please try again.';
      showError(errorMessage);
    }
  };

  const handleAssignToBatch = async () => {
    if (!selectedBatchId || assigningBatch) return;

    setAssigningBatch(true);
    try {
      await api.addRepositoriesToBatch(selectedBatchId, [repository.id]);
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({ queryKey: ['repository', repository.full_name] });
      await queryClient.invalidateQueries({ queryKey: ['batches'] });
      
      showSuccess('Repository assigned to batch successfully!');
      setSelectedBatchId(null);
    } catch (error: any) {
      console.error('Failed to assign to batch:', error);
      const errorMsg = error.response?.data?.error || 'Failed to assign to batch. Please try again.';
      showError(errorMsg);
    } finally {
      setAssigningBatch(false);
    }
  };

  const handleRemoveFromBatch = () => {
    if (!repository.batch_id || assigningBatch) return;
    setShowRemoveDialog(true);
  };

  const confirmRemoveFromBatch = async () => {
    if (!repository.batch_id) return;

    setAssigningBatch(true);
    setShowRemoveDialog(false);
    try {
      await api.removeRepositoriesFromBatch(repository.batch_id, [repository.id]);
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({ queryKey: ['repository', repository.full_name] });
      await queryClient.invalidateQueries({ queryKey: ['batches'] });
      
      showSuccess('Repository removed from batch successfully!');
    } catch (error: any) {
      console.error('Failed to remove from batch:', error);
      const errorMsg = error.response?.data?.error || 'Failed to remove from batch. Please try again.';
      showError(errorMsg);
    } finally {
      setAssigningBatch(false);
    }
  };

  const handleSaveMigrationOptions = async () => {
    setSavingOptions(true);
    try {
      await api.updateRepository(repository.full_name, {
        exclude_releases: excludeReleases
      });
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({ queryKey: ['repository', repository.full_name] });
      
      showSuccess('Migration options saved successfully!');
    } catch (error: any) {
      console.error('Failed to save migration options:', error);
      const errorMsg = error.response?.data?.error || 'Failed to save migration options. Please try again.';
      showError(errorMsg);
    } finally {
      setSavingOptions(false);
    }
  };

  // Calculate complexity summary - show only non-zero contributors
  const getComplexityContributors = () => {
    const breakdown = repository.complexity_breakdown;
    const contributors: Array<{ label: string; points: number; color: string }> = [];

    // Helper to add contributor if it has points
    const addIfNonZero = (points: number | undefined, label: string, color: string) => {
      if (points && points > 0) {
        contributors.push({ label, points, color });
      }
    };

    if (breakdown) {
      // Common factors
      addIfNonZero(breakdown.size_points, 'Repository Size', 'text-blue-600');
      addIfNonZero(breakdown.large_files_points, 'Large Files', 'text-red-600');
      addIfNonZero(breakdown.activity_points, 'Activity Level', 'text-purple-600');
      
      // GitHub-specific factors
      addIfNonZero(breakdown.lfs_points, 'Git LFS', 'text-orange-600');
      addIfNonZero(breakdown.submodules_points, 'Submodules', 'text-orange-600');
      addIfNonZero(breakdown.packages_points, 'Packages', 'text-red-600');
      addIfNonZero(breakdown.environments_points, 'Environments', 'text-red-600');
      addIfNonZero(breakdown.secrets_points, 'Secrets', 'text-red-600');
      addIfNonZero(breakdown.variables_points, 'Variables', 'text-orange-600');
      addIfNonZero(breakdown.discussions_points, 'Discussions', 'text-orange-600');
      addIfNonZero(breakdown.releases_points, 'Releases', 'text-orange-600');
      addIfNonZero(breakdown.webhooks_points, 'Webhooks', 'text-yellow-600');
      addIfNonZero(breakdown.branch_protections_points, 'Branch Protections', 'text-yellow-600');
      addIfNonZero(breakdown.rulesets_points, 'Rulesets', 'text-yellow-600');
      addIfNonZero(breakdown.security_points, 'Advanced Security', 'text-yellow-600');
      addIfNonZero(breakdown.runners_points, 'Self-Hosted Runners', 'text-red-600');
      addIfNonZero(breakdown.apps_points, 'GitHub Apps', 'text-orange-600');
      addIfNonZero(breakdown.projects_points, 'Projects', 'text-orange-600');
      addIfNonZero(breakdown.public_visibility_points, 'Public Visibility', 'text-blue-600');
      addIfNonZero(breakdown.internal_visibility_points, 'Internal Visibility', 'text-yellow-600');
      addIfNonZero(breakdown.codeowners_points, 'CODEOWNERS', 'text-yellow-600');
      
      // Azure DevOps-specific factors
      addIfNonZero(breakdown.ado_tfvc_points, 'TFVC Repository (BLOCKING)', 'text-red-700');
      addIfNonZero(breakdown.ado_classic_pipeline_points, 'Classic Pipelines', 'text-red-600');
      addIfNonZero(breakdown.ado_package_feed_points, 'Package Feeds', 'text-red-600');
      addIfNonZero(breakdown.ado_service_connection_points, 'Service Connections', 'text-red-600');
      addIfNonZero(breakdown.ado_active_pipeline_points, 'Active Pipelines', 'text-red-600');
      addIfNonZero(breakdown.ado_active_boards_points, 'Active Azure Boards', 'text-red-600');
      addIfNonZero(breakdown.ado_wiki_points, 'Wiki Pages', 'text-orange-600');
      addIfNonZero(breakdown.ado_test_plan_points, 'Test Plans', 'text-orange-600');
      addIfNonZero(breakdown.ado_variable_group_points, 'Variable Groups', 'text-yellow-600');
      addIfNonZero(breakdown.ado_service_hook_points, 'Service Hooks', 'text-yellow-600');
      addIfNonZero(breakdown.ado_many_prs_points, 'Many Pull Requests', 'text-yellow-600');
      addIfNonZero(breakdown.ado_branch_policy_points, 'Branch Policies', 'text-yellow-600');
    }

    // Sort by points descending
    return contributors.sort((a, b) => b.points - a.points);
  };

  const complexityContributors = getComplexityContributors();
  const totalPoints = repository.complexity_score ?? 0;
  
  let category = 'Simple';
  
  if (totalPoints > 17) {
    category = 'Very Complex';
  } else if (totalPoints > 10) {
    category = 'Complex';
  } else if (totalPoints > 5) {
    category = 'Medium';
  }

  return (
    <div className="space-y-6">
      {/* Complexity Score Summary */}
      <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)', border: '1px solid var(--borderColor-default)' }}>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>Migration Complexity</h3>
            <ComplexityInfoModal source={repository.source as 'github' | 'azuredevops'} />
          </div>
          
          <div 
            className="rounded-lg p-4"
            style={{
              backgroundColor: 'var(--bgColor-muted)',
              border: '1px solid var(--borderColor-default)'
            }}
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  {totalPoints > 17 ? (
                    <span style={{ color: 'var(--fgColor-danger)' }}>
                      <XCircleFillIcon size={16} />
                    </span>
                  ) : totalPoints > 5 ? (
                    <span style={{ color: 'var(--fgColor-attention)' }}>
                      <AlertIcon size={16} />
                    </span>
                  ) : (
                    <span style={{ color: 'var(--fgColor-success)' }}>
                      <CheckCircleFillIcon size={16} />
                    </span>
                  )}
                  <span className="text-sm font-medium" style={{ color: 'var(--fgColor-muted)' }}>
                    {category} Migration
                  </span>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Complexity Score:</span>
                  <span 
                    className="text-3xl font-bold"
                    style={{
                      color: totalPoints > 17 ? 'var(--fgColor-danger)' :
                             totalPoints > 10 ? 'var(--fgColor-attention)' :
                             totalPoints > 5 ? 'var(--fgColor-attention)' :
                             'var(--fgColor-success)'
                    }}
                  >
                    {totalPoints}
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* Top Contributing Factors */}
          {complexityContributors.length > 0 && (
            <div className="space-y-2">
              <h4 className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>Contributing Factors:</h4>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {complexityContributors.slice(0, 8).map((contributor, idx) => (
                  <div 
                    key={idx} 
                    className="flex justify-between items-center py-2 px-3 rounded"
                    style={{
                      backgroundColor: 'var(--bgColor-muted)',
                      border: '1px solid var(--borderColor-default)'
                    }}
                  >
                    <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>{contributor.label}</span>
                    <span 
                      className="text-sm font-semibold"
                      style={{
                        color: contributor.color.includes('red') ? 'var(--fgColor-danger)' :
                               contributor.color.includes('orange') ? 'var(--fgColor-attention)' :
                               contributor.color.includes('yellow') ? 'var(--fgColor-attention)' :
                               contributor.color.includes('blue') ? 'var(--fgColor-accent)' :
                               contributor.color.includes('purple') ? 'var(--fgColor-done)' :
                               'var(--fgColor-default)'
                      }}
                    >
                      +{contributor.points}
                    </span>
                  </div>
                ))}
              </div>
              {complexityContributors.length > 8 && (
                <p className="text-xs mt-2" style={{ color: 'var(--fgColor-muted)' }}>
                  ... and {complexityContributors.length - 8} more factors
                </p>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Validation Issues - Only show if there are issues */}
      {(hasBlockingIssues || hasWarnings) && (
        <div 
          className="rounded-lg shadow-sm border-2" 
          style={{ 
            backgroundColor: 'var(--bgColor-default)', 
            borderColor: hasBlockingIssues ? 'var(--borderColor-danger)' : 'var(--borderColor-attention)'
          }}
        >
          <button
            onClick={() => setExpandedValidation(!expandedValidation)}
            className="w-full px-6 py-4 flex items-center justify-between transition-opacity hover:opacity-80"
          >
            <div className="flex items-center gap-3">
              {hasBlockingIssues ? (
                <span style={{ color: 'var(--fgColor-danger)' }}>
                  <XCircleFillIcon size={24} />
                </span>
              ) : (
                <span style={{ color: 'var(--fgColor-attention)' }}>
                  <AlertIcon size={24} />
                </span>
              )}
              <div className="text-left">
                <h3 className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  {hasBlockingIssues ? '⚠️ Validation Issues (Blocking)' : '⚠ Validation Warnings'}
                </h3>
                <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                  {hasBlockingIssues 
                    ? 'These issues must be resolved before migration' 
                    : 'Repository can migrate but has warnings to review'}
                </p>
              </div>
            </div>
            <span style={{ color: 'var(--fgColor-muted)' }}>
            <ChevronDownIcon 
              size={20}
                className={`transition-transform ${expandedValidation ? 'rotate-180' : ''}`}
            />
            </span>
          </button>
          
          {expandedValidation && (
            <div 
              className="px-6 pb-4 pt-4"
              style={{ borderTop: '1px solid var(--borderColor-default)' }}
            >
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-semibold mb-3" style={{ color: 'var(--fgColor-default)' }}>Issues Found:</h4>
                  <ul className="space-y-2 text-sm">
                    {repository.has_oversized_repository && (
                      <li className="flex items-start gap-2" style={{ color: 'var(--fgColor-danger)' }}>
                        <span className="font-bold">✗</span>
                        <span>Repository size exceeds 40 GB limit ({formatBytes(repository.total_size)})</span>
                      </li>
                    )}
                    {repository.has_blocking_files && (
                      <li className="flex items-start gap-2" style={{ color: 'var(--fgColor-danger)' }}>
                        <span className="font-bold">✗</span>
                        <span>Files larger than 400 MB detected</span>
                      </li>
                    )}
                    {repository.has_oversized_commits && (
                      <li className="flex items-start gap-2" style={{ color: 'var(--fgColor-danger)' }}>
                        <span className="font-bold">✗</span>
                        <span>Commits larger than 2 GB detected</span>
                      </li>
                    )}
                    {repository.has_long_refs && (
                      <li className="flex items-start gap-2" style={{ color: 'var(--fgColor-danger)' }}>
                        <span className="font-bold">✗</span>
                        <span>Git references longer than 255 bytes detected</span>
                      </li>
                    )}
                    {repository.estimated_metadata_size && repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024 && (
                      <li className="flex items-start gap-2" style={{ color: 'var(--fgColor-attention)' }}>
                        <span className="font-bold">⚠</span>
                        <span>Metadata size approaching 40 GB limit (est. {formatBytes(repository.estimated_metadata_size)})</span>
                      </li>
                    )}
                    {repository.has_large_file_warnings && (
                      <li className="flex items-start gap-2" style={{ color: 'var(--fgColor-attention)' }}>
                        <span className="font-bold">⚠</span>
                        <span>Large files (100-400 MB) detected - consider Git LFS</span>
                      </li>
                    )}
                  </ul>
                </div>

                <div 
                  className="p-3 rounded-lg"
                  style={{
                    backgroundColor: 'var(--accent-subtle)',
                    border: '1px solid var(--borderColor-accent-muted)'
                  }}
                >
                  <p className="text-sm" style={{ color: 'var(--fgColor-accent)' }}>
                    <span className="font-semibold">💡 Remediation: </span>
                    {hasBlockingIssues 
                      ? 'These issues must be fixed before the repository can be migrated. Consider using BFG Repo Cleaner or git-filter-repo to address large files and commits.' 
                      : 'While these warnings won\'t block migration, addressing them can improve migration success rates and reduce migration time.'}
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Migration Configuration - Hide if migration is complete */}
      {repository.status !== 'complete' && (
      <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)', border: '1px solid var(--borderColor-default)' }}>
        <h3 className="text-lg font-semibold mb-6" style={{ color: 'var(--fgColor-default)' }}>Migration Configuration</h3>

        <div className="space-y-6">
        {/* Destination Configuration */}
        {canMigrate && (
              <FormControl required>
              <FormControl.Label>Destination repository</FormControl.Label>
                  <TextInput
                    value={destinationFullName}
                    onChange={(e) => setDestinationFullName(e.target.value)}
                    placeholder="org/repo"
                onBlur={handleSaveDestination}
                    disabled={updateRepositoryMutation.isPending}
                block
                    required
                    aria-invalid={!destinationFullName.trim() ? true : undefined}
                monospace
              />
                {!destinationFullName.trim() && (
                  <FormControl.Validation variant="error">
                    Destination repository name is required
                  </FormControl.Validation>
                )}
              <FormControl.Caption>
                Destination defaults to same source organization and name but can be customized
              </FormControl.Caption>
            </FormControl>
        )}

        {/* Batch Assignment */}
        {canChangeBatch && (
            <div 
              className="pt-6"
              style={{ borderTop: '1px solid var(--borderColor-default)' }}
            >
              <FormControl>
                <FormControl.Label>Batch assignment</FormControl.Label>
            {repository.batch_id ? (
              <div className="flex items-center gap-2">
                  <div 
                    className="flex-1 px-3 py-2 rounded-md"
                    style={{
                      backgroundColor: 'var(--bgColor-muted)',
                      border: '1px solid var(--borderColor-default)'
                    }}
                  >
                    <Badge>{currentBatch?.name || `Batch #${repository.batch_id}`}</Badge>
                </div>
                <Button
                  onClick={handleRemoveFromBatch}
                  disabled={assigningBatch}
                  variant="default"
                >
                    {assigningBatch ? 'Removing...' : 'Remove'}
                </Button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                  <Select
                    value={selectedBatchId?.toString() || ''}
                  onChange={(e) => setSelectedBatchId(e.target.value ? Number(e.target.value) : null)}
                  disabled={assigningBatch}
                    block
                >
                    <Select.Option value="">Select a batch...</Select.Option>
                  {batches.map((batch) => (
                      <Select.Option key={batch.id} value={batch.id.toString()}>
                      {batch.name} ({batch.type}) - {batch.status} - {batch.repository_count} repos
                      </Select.Option>
                  ))}
                  </Select>
                <Button
                  onClick={handleAssignToBatch}
                  disabled={!selectedBatchId || assigningBatch}
                    variant="primary"
                >
                    {assigningBatch ? 'Assigning...' : 'Assign'}
                </Button>
              </div>
            )}
              <FormControl.Caption>
              {repository.batch_id
                  ? 'Repository is assigned to a batch for grouped migration'
                : batches.length === 0
                ? 'No pending or ready batches available. Create a batch first.'
                : 'Assign this repository to a batch for grouped migration'}
              </FormControl.Caption>
            </FormControl>
          </div>
        )}

        {/* Migration Options */}
          <div 
            className="pt-6"
            style={{ borderTop: '1px solid var(--borderColor-default)' }}
          >
            <div className="flex items-center justify-between mb-4">
              <div>
                <h4 className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>Migration options</h4>
                <p className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>Configure pre-migration, migration, and post-migration behavior</p>
              </div>
              <Button
                variant="invisible"
                size="small"
                leadingVisual={InfoIcon}
                onClick={() => setShowMigrationOptionsInfo(true)}
              >
                View details
              </Button>
            </div>

            <div 
              className="rounded-md p-4 space-y-3"
              style={{
                backgroundColor: 'var(--bgColor-muted)',
                border: '1px solid var(--borderColor-default)'
              }}
            >
              <FormControl>
                <div className="flex items-start gap-3">
                  <Checkbox
                    checked={excludeReleases}
                    onChange={(e) => setExcludeReleases(e.target.checked)}
                    value="exclude-releases"
                  />
                  <div className="flex-1">
                    <FormControl.Label>Exclude releases</FormControl.Label>
                    <FormControl.Caption>
                      Skip migrating releases and their associated assets. Useful for repositories with many releases or large binary assets. Release tags and commit history will still be migrated.
                    </FormControl.Caption>
                  </div>
                </div>
              </FormControl>

              {hasOptionsChanges && (
                <div className="flex gap-2 pt-3" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
                  <Button
                    onClick={handleSaveMigrationOptions}
                    disabled={savingOptions}
                    variant="primary"
                    size="small"
                  >
                    {savingOptions ? 'Saving...' : 'Save Changes'}
                  </Button>
                  <Button
                    onClick={() => setExcludeReleases(repository.exclude_releases)}
                    disabled={savingOptions}
                    variant="default"
                    size="small"
                  >
                    Reset
                  </Button>
                </div>
              )}
            </div>
          </div>

          {/* Migration Options Info Modal */}
          {showMigrationOptionsInfo && (
            <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }} onClick={() => setShowMigrationOptionsInfo(false)}>
              <div className="rounded-lg p-6 max-w-2xl w-full mx-4" style={{ backgroundColor: 'var(--bgColor-default)' }} onClick={e => e.stopPropagation()}>
                <div className="flex justify-between items-center mb-4">
                  <h3 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>Migration Options</h3>
                  <Button
                    variant="invisible"
                    onClick={() => setShowMigrationOptionsInfo(false)}
                    aria-label="Close dialog"
                  >
                    <XIcon />
                  </Button>
                </div>
                
                <div className="space-y-4">
                  <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                    Configure options that control pre-migration validation, what data is migrated, and post-migration actions.
                  </p>
                  
                  <div 
                    className="pl-4 py-2"
                    style={{ borderLeft: '4px solid var(--accent-emphasis)' }}
                  >
                    <h4 className="font-semibold text-sm mb-2" style={{ color: 'var(--fgColor-default)' }}>Exclude releases</h4>
                    <p className="text-sm mb-2" style={{ color: 'var(--fgColor-default)' }}>
                      When enabled, this option skips migrating releases and their associated assets during the migration process.
                    </p>
                    <p className="text-sm mb-2" style={{ color: 'var(--fgColor-default)' }}>
                      <strong>Use this option when:</strong>
                    </p>
                    <ul className="list-disc pl-5 text-sm space-y-1 mb-2" style={{ color: 'var(--fgColor-default)' }}>
                      <li>Your repository has many releases with large binary assets</li>
                      <li>Release assets are stored elsewhere or can be recreated</li>
                      <li>You want to significantly reduce migration time and metadata size</li>
                    </ul>
                    <div 
                      className="rounded-md p-3 mt-3"
                      style={{
                        backgroundColor: 'var(--accent-subtle)',
                        border: '1px solid var(--borderColor-accent-muted)'
                      }}
                    >
                      <p className="text-xs" style={{ color: 'var(--fgColor-accent)' }}>
                        <strong>Note:</strong> Release tags and their associated commit history will still be migrated, but release notes and assets will not be included.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
      )}

      {/* Remove from Batch Confirmation Dialog */}
      {showRemoveDialog && (
        <Dialog
          returnFocusRef={removeButtonRef}
          onDismiss={() => setShowRemoveDialog(false)}
          aria-labelledby="remove-dialog-header"
        >
          <Dialog.Header id="remove-dialog-header">
            Remove from Batch
          </Dialog.Header>
          <div style={{ padding: '16px' }}>
            <p style={{ fontSize: '14px', color: 'var(--fgColor-default)' }}>
              Are you sure you want to remove this repository from its batch?
            </p>
          </div>
          <div style={{ 
            padding: '12px 16px', 
            borderTop: '1px solid var(--borderColor-default)',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px'
          }}>
            <Button onClick={() => setShowRemoveDialog(false)}>
              Cancel
            </Button>
            <Button variant="danger" onClick={confirmRemoveFromBatch}>
              Remove
            </Button>
          </div>
        </Dialog>
      )}
    </div>
  );
}

