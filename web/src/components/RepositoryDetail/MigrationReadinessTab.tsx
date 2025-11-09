import { useState } from 'react';
import type { Repository, Batch } from '../../types';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import { Badge } from '../common/Badge';
import { ComplexityInfoModal } from '../common/ComplexityInfoModal';
import { useUpdateRepository } from '../../hooks/useMutations';

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
  const [destinationFullName, setDestinationFullName] = useState<string>(
    repository.destination_full_name || repository.full_name
  );

  // Migration options state
  const [excludeReleases, setExcludeReleases] = useState(repository.exclude_releases);
  const [savingOptions, setSavingOptions] = useState(false);
  const hasOptionsChanges = excludeReleases !== repository.exclude_releases;

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
    } catch (error) {
      console.error('Failed to save destination:', error);
      alert('Failed to save destination. Please try again.');
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
      addIfNonZero(breakdown.size_points, 'Repository Size', 'text-blue-600');
      addIfNonZero(breakdown.large_files_points, 'Large Files', 'text-red-600');
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
      addIfNonZero(breakdown.activity_points, 'Activity Level', 'text-purple-600');
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
            💡 Scoring based on GitHub migration documentation
          </p>
          <ComplexityInfoModal />
        </div>
      </div>

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
                    setDestinationFullName(repository.destination_full_name || repository.full_name);
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
              {destinationFullName === repository.full_name 
                ? 'Using same organization as source (default)' 
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

