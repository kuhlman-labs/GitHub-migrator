import { useState, useEffect } from 'react';
import type { Repository, Batch } from '../../types';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import { Badge } from '../common/Badge';
import { ComplexityInfoModal } from '../common/ComplexityInfoModal';
import { useUpdateRepository } from '../../hooks/useMutations';
import { formatBytes } from '../../utils/format';

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
  
  // Batch assignment state - show pending and ready batches
  const batches = allBatches.filter(b => b.status === 'pending' || b.status === 'ready');
  const [selectedBatchId, setSelectedBatchId] = useState<number | null>(null);
  const [assigningBatch, setAssigningBatch] = useState(false);
  
  // Destination configuration
  const [editingDestination, setEditingDestination] = useState(false);
  
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

  // Sync destinationFullName with repository data when it changes (but not while editing)
  useEffect(() => {
    if (!editingDestination) {
      setDestinationFullName(getDefaultDestination());
    }
  }, [repository.destination_full_name, repository.full_name, repository.ado_project, editingDestination]);

  // Migration options state
  const [excludeReleases, setExcludeReleases] = useState(repository.exclude_releases);
  const [savingOptions, setSavingOptions] = useState(false);
  const hasOptionsChanges = excludeReleases !== repository.exclude_releases;

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
      alert('Destination must be in "organization/repository" format');
      return;
    }

    try {
      await updateRepositoryMutation.mutateAsync({
        fullName: repository.full_name,
        updates: { destination_full_name: destinationFullName },
      });
      
      setEditingDestination(false);
    } catch (error: any) {
      console.error('Failed to save destination:', error);
      const errorMessage = error.response?.data?.error || error.message || 'Failed to save destination. Please try again.';
      alert(errorMessage);
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
      
      alert('Repository assigned to batch successfully!');
      setSelectedBatchId(null);
    } catch (error: any) {
      console.error('Failed to assign to batch:', error);
      const errorMsg = error.response?.data?.error || 'Failed to assign to batch. Please try again.';
      alert(errorMsg);
    } finally {
      setAssigningBatch(false);
    }
  };

  const handleRemoveFromBatch = async () => {
    if (!repository.batch_id || assigningBatch) return;

    if (!confirm('Are you sure you want to remove this repository from its batch?')) {
      return;
    }

    setAssigningBatch(true);
    try {
      await api.removeRepositoriesFromBatch(repository.batch_id, [repository.id]);
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({ queryKey: ['repository', repository.full_name] });
      await queryClient.invalidateQueries({ queryKey: ['batches'] });
      
      alert('Repository removed from batch successfully!');
    } catch (error: any) {
      console.error('Failed to remove from batch:', error);
      const errorMsg = error.response?.data?.error || 'Failed to remove from batch. Please try again.';
      alert(errorMsg);
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
      
      alert('Migration options saved successfully!');
    } catch (error: any) {
      console.error('Failed to save migration options:', error);
      const errorMsg = error.response?.data?.error || 'Failed to save migration options. Please try again.';
      alert(errorMsg);
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
  let categoryColor = 'text-green-600';
  let categoryBg = 'bg-green-50';
  if (totalPoints > 17) {
    category = 'Very Complex';
    categoryColor = 'text-red-600';
    categoryBg = 'bg-red-50';
  } else if (totalPoints > 10) {
    category = 'Complex';
    categoryColor = 'text-orange-600';
    categoryBg = 'bg-orange-50';
  } else if (totalPoints > 5) {
    category = 'Medium';
    categoryColor = 'text-yellow-600';
    categoryBg = 'bg-yellow-50';
  }

  return (
    <div className="space-y-6">
      {/* Complexity Score Summary */}
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h3 className="text-lg font-semibold mb-4">Migration Complexity</h3>
        
        <div className={`mb-4 p-4 ${categoryBg} rounded-lg border-l-4 ${categoryColor.replace('text-', 'border-')}`}>
          <div className="flex justify-between items-center mb-2">
            <span className="text-sm font-medium text-gray-700">Total Complexity Score</span>
            <span className={`text-3xl font-bold ${categoryColor}`}>{totalPoints}</span>
          </div>
          <div className="text-sm">
            <span className="font-medium text-gray-700">Category: </span>
            <span className={`font-semibold ${categoryColor}`}>{category}</span>
          </div>
        </div>

        {/* Top Contributing Factors */}
        {complexityContributors.length > 0 && (
          <div className="space-y-2 mb-4">
            <h4 className="text-sm font-medium text-gray-700">Contributing Factors:</h4>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {complexityContributors.slice(0, 8).map((contributor, idx) => (
                <div key={idx} className="flex justify-between items-center py-2 px-3 bg-gray-50 rounded border border-gray-200">
                  <span className="text-sm text-gray-900">{contributor.label}</span>
                  <span className={`text-sm font-semibold ${contributor.color}`}>
                    +{contributor.points}
                  </span>
                </div>
              ))}
            </div>
            {complexityContributors.length > 8 && (
              <p className="text-xs text-gray-500 mt-2">
                ... and {complexityContributors.length - 8} more factors
              </p>
            )}
          </div>
        )}

        <div className="pt-3 border-t border-gray-200 flex items-center justify-between">
          <p className="text-xs text-blue-700">
            💡 {repository.source === 'azuredevops' ? 
              'Scoring based on ADO → GitHub migration complexity factors' :
              'Scoring based on GitHub migration documentation'}
          </p>
          <ComplexityInfoModal source={repository.source as 'github' | 'azuredevops'} />
        </div>
      </div>

      {/* Validation Issues - Only show if there are issues */}
      {(hasBlockingIssues || hasWarnings) && (
        <div className="bg-white rounded-lg shadow-sm border-2 border-red-200">
          <button
            onClick={() => setExpandedValidation(!expandedValidation)}
            className="w-full px-6 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center gap-3">
              {hasBlockingIssues ? (
                <svg className="w-6 h-6 text-red-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                </svg>
              ) : (
                <svg className="w-6 h-6 text-yellow-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
              )}
              <div className="text-left">
                <h3 className="font-semibold text-gray-900">
                  {hasBlockingIssues ? '⚠️ Validation Issues (Blocking)' : '⚠ Validation Warnings'}
                </h3>
                <p className="text-sm text-gray-600">
                  {hasBlockingIssues 
                    ? 'These issues must be resolved before migration' 
                    : 'Repository can migrate but has warnings to review'}
                </p>
              </div>
            </div>
            <svg 
              className={`w-5 h-5 text-gray-400 transition-transform ${expandedValidation ? 'rotate-180' : ''}`}
              fill="none" 
              viewBox="0 0 24 24" 
              strokeWidth={2} 
              stroke="currentColor"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
          
          {expandedValidation && (
            <div className="px-6 pb-4 border-t border-gray-200 pt-4">
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Issues Found:</h4>
                  <ul className="space-y-2 text-sm">
                    {repository.has_oversized_repository && (
                      <li className="flex items-start gap-2 text-red-700">
                        <span className="text-red-600 font-bold">✗</span>
                        <span>Repository size exceeds 40 GB limit ({formatBytes(repository.total_size)})</span>
                      </li>
                    )}
                    {repository.has_blocking_files && (
                      <li className="flex items-start gap-2 text-red-700">
                        <span className="text-red-600 font-bold">✗</span>
                        <span>Files larger than 400 MB detected</span>
                      </li>
                    )}
                    {repository.has_oversized_commits && (
                      <li className="flex items-start gap-2 text-red-700">
                        <span className="text-red-600 font-bold">✗</span>
                        <span>Commits larger than 2 GB detected</span>
                      </li>
                    )}
                    {repository.has_long_refs && (
                      <li className="flex items-start gap-2 text-red-700">
                        <span className="text-red-600 font-bold">✗</span>
                        <span>Git references longer than 255 bytes detected</span>
                      </li>
                    )}
                    {repository.estimated_metadata_size && repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024 && (
                      <li className="flex items-start gap-2 text-yellow-700">
                        <span className="text-yellow-600 font-bold">⚠</span>
                        <span>Metadata size approaching 40 GB limit (est. {formatBytes(repository.estimated_metadata_size)})</span>
                      </li>
                    )}
                    {repository.has_large_file_warnings && (
                      <li className="flex items-start gap-2 text-yellow-700">
                        <span className="text-yellow-600 font-bold">⚠</span>
                        <span>Large files (100-400 MB) detected - consider Git LFS</span>
                      </li>
                    )}
                  </ul>
                </div>

                <div className="p-3 bg-blue-50 rounded-lg border border-blue-200">
                  <p className="text-sm text-blue-800">
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
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h3 className="text-lg font-semibold mb-4">Migration Configuration</h3>

        {/* Destination Configuration */}
        {canMigrate && (
          <div className="mb-4 p-4 bg-gray-50 rounded-lg">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Destination (where to migrate)
            </label>
            {editingDestination ? (
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={destinationFullName}
                  onChange={(e) => setDestinationFullName(e.target.value)}
                  placeholder="org/repo"
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  disabled={updateRepositoryMutation.isPending}
                />
                <button
                  onClick={handleSaveDestination}
                  disabled={updateRepositoryMutation.isPending}
                  className="px-3 py-1.5 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50"
                >
                  {updateRepositoryMutation.isPending ? 'Saving...' : 'Save'}
                </button>
                <button
                  onClick={() => {
                    setEditingDestination(false);
                    // Reset to the saved/default value using the same logic
                    setDestinationFullName(getDefaultDestination());
                  }}
                  disabled={updateRepositoryMutation.isPending}
                  className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
                >
                  Cancel
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <code className="flex-1 px-3 py-2 bg-white border border-gray-200 rounded-md text-sm text-gray-900">
                  {destinationFullName}
                </code>
                <button
                  onClick={() => setEditingDestination(true)}
                  className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
                >
                  Edit
                </button>
              </div>
            )}
            <p className="mt-1 text-xs text-gray-500">
              {destinationFullName === getSuggestedDefault()
                ? repository.ado_project 
                  ? 'Suggested default preserving ADO org and project (spaces replaced with hyphens)' 
                  : 'Suggested default using same organization as source'
                : repository.ado_project
                  ? 'Using custom destination'
                : 'Using custom destination organization'}
            </p>
          </div>
        )}

        {/* Batch Assignment */}
        {canChangeBatch && (
          <div className="p-4 bg-gray-50 rounded-lg">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Batch Assignment
            </label>
            {repository.batch_id ? (
              <div className="flex items-center gap-2">
                <div className="flex-1 px-3 py-2 bg-white border border-gray-200 rounded-md text-sm">
                  <Badge color="blue">{currentBatch?.name || `Batch #${repository.batch_id}`}</Badge>
                </div>
                <button
                  onClick={handleRemoveFromBatch}
                  disabled={assigningBatch}
                  className="px-3 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
                >
                  {assigningBatch ? 'Removing...' : 'Remove from Batch'}
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <select
                  value={selectedBatchId || ''}
                  onChange={(e) => setSelectedBatchId(e.target.value ? Number(e.target.value) : null)}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  disabled={assigningBatch}
                >
                  <option value="">Select a batch...</option>
                  {batches.map((batch) => (
                    <option key={batch.id} value={batch.id}>
                      {batch.name} ({batch.type}) - {batch.status} - {batch.repository_count} repos
                    </option>
                  ))}
                </select>
                <button
                  onClick={handleAssignToBatch}
                  disabled={!selectedBatchId || assigningBatch}
                  className="px-3 py-1.5 bg-gh-success text-white rounded-md text-sm font-medium hover:bg-gh-success-hover disabled:opacity-50"
                >
                  {assigningBatch ? 'Assigning...' : 'Assign to Batch'}
                </button>
              </div>
            )}
            <p className="mt-1 text-xs text-gray-500">
              {repository.batch_id
                ? 'Repository is assigned to a batch'
                : batches.length === 0
                ? 'No pending or ready batches available. Create a batch first.'
                : 'Assign this repository to a batch for grouped migration'}
            </p>
          </div>
        )}

        {/* Migration Options */}
        <div className="mt-4 pt-4 border-t border-gray-200">
          <h4 className="text-sm font-semibold text-gray-900 mb-2">Migration Options</h4>
          <p className="text-gray-600 text-sm mb-3">
            Configure what data to include or exclude from the migration.
          </p>
          
          <div className="p-3 bg-gray-50 rounded-lg border border-gray-200">
            <label className="flex items-start cursor-pointer">
              <input
                type="checkbox"
                checked={excludeReleases}
                onChange={(e) => setExcludeReleases(e.target.checked)}
                className="mt-1 mr-3 h-4 w-4"
              />
              <div className="flex-1">
                <div className="font-medium text-gray-900">Exclude Releases</div>
                <div className="text-sm text-gray-600 mt-1">
                  Skip migrating releases and their assets. This can significantly reduce metadata size for repositories with large release assets.
                </div>
              </div>
            </label>
          </div>

          {hasOptionsChanges && (
            <div className="flex gap-2 mt-3">
              <button
                onClick={handleSaveMigrationOptions}
                disabled={savingOptions}
                className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
              >
                {savingOptions ? 'Saving...' : 'Save Options'}
              </button>
              <button
                onClick={() => setExcludeReleases(repository.exclude_releases)}
                disabled={savingOptions}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
              >
                Reset
              </button>
            </div>
          )}
        </div>
      </div>
      )}
    </div>
  );
}

