import { useCallback, useEffect, useState } from 'react';
import { api } from '../../services/api';
import type { Batch, Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { formatBytes, formatDate } from '../../utils/format';

export function BatchManagement() {
  const [batches, setBatches] = useState<Batch[]>([]);
  const [selectedBatch, setSelectedBatch] = useState<Batch | null>(null);
  const [batchRepositories, setBatchRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showRetryModal, setShowRetryModal] = useState(false);

  useEffect(() => {
    loadBatches();
    // Poll for updates every 15 seconds
    const interval = setInterval(loadBatches, 15000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (selectedBatch) {
      loadBatchRepositories(selectedBatch.id);
    }
  }, [selectedBatch]);

  const loadBatches = async () => {
    try {
      const data = await api.listBatches();
      setBatches(data);
    } catch (error) {
      console.error('Failed to load batches:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadBatchRepositories = async (batchId: number) => {
    try {
      const data = await api.listRepositories({ batch_id: batchId });
      setBatchRepositories(data);
    } catch (error) {
      console.error('Failed to load batch repositories:', error);
    }
  };

  const handleStartBatch = async (batchId: number) => {
    if (!confirm('Are you sure you want to start migration for this entire batch?')) {
      return;
    }

    try {
      const response = await api.startBatch(batchId);
      alert(`Started migration for ${response.count} repositories`);
      await loadBatches();
      if (selectedBatch?.id === batchId) {
        await loadBatchRepositories(batchId);
      }
    } catch (error) {
      console.error('Failed to start batch:', error);
      alert('Failed to start batch migration');
    }
  };

  const handleBatchCreated = async () => {
    setShowCreateModal(false);
    await loadBatches();
  };

  const handleBatchUpdated = async () => {
    setShowEditModal(false);
    await loadBatches();
    if (selectedBatch) {
      const updated = await api.getBatch(selectedBatch.id);
      setSelectedBatch(updated);
      await loadBatchRepositories(selectedBatch.id);
    }
  };

  const handleRetryComplete = async () => {
    setShowRetryModal(false);
    if (selectedBatch) {
      await loadBatchRepositories(selectedBatch.id);
    }
  };

  const failedRepos = batchRepositories.filter(
    (r) => r.status === 'migration_failed' || r.status === 'dry_run_failed'
  );

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-light text-gray-900">Batch Management</h1>
        <button
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700"
        >
          Create New Batch
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Batch List */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg shadow-sm p-4">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Batches</h2>
            {loading ? (
              <LoadingSpinner />
            ) : batches.length === 0 ? (
              <div className="text-center py-8 text-gray-500">No batches found</div>
            ) : (
              <div className="space-y-2">
                {batches.map((batch) => (
                  <BatchCard
                    key={batch.id}
                    batch={batch}
                    isSelected={selectedBatch?.id === batch.id}
                    onClick={() => setSelectedBatch(batch)}
                    onStart={() => handleStartBatch(batch.id)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Batch Detail */}
        <div className="lg:col-span-2">
          {selectedBatch ? (
            <div className="bg-white rounded-lg shadow-sm p-6">
              <div className="flex justify-between items-start mb-6">
                <div>
                  <h2 className="text-2xl font-medium text-gray-900">{selectedBatch.name}</h2>
                  <p className="text-gray-600 mt-1">{selectedBatch.description}</p>
                  <div className="flex gap-3 mt-3">
                    <StatusBadge status={selectedBatch.status} />
                    <span className="text-sm text-gray-600">
                      {selectedBatch.repository_count} repositories
                    </span>
                  </div>
                  {selectedBatch.scheduled_at && (
                    <div className="text-sm text-gray-600 mt-2">
                      Scheduled: {formatDate(selectedBatch.scheduled_at)}
                    </div>
                  )}
                </div>

                <div className="flex gap-2">
                  {selectedBatch.status === 'ready' && (
                    <>
                      <button
                        onClick={() => setShowEditModal(true)}
                        className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50"
                      >
                        Edit Batch
                      </button>
                      <button
                        onClick={() => handleStartBatch(selectedBatch.id)}
                        className="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700"
                      >
                        Start Batch Migration
                      </button>
                    </>
                  )}
                  {failedRepos.length > 0 && (
                    <button
                      onClick={() => setShowRetryModal(true)}
                      className="px-4 py-2 bg-yellow-600 text-white rounded-lg text-sm font-medium hover:bg-yellow-700"
                    >
                      Retry Failed ({failedRepos.length})
                    </button>
                  )}
                </div>
              </div>

              {/* Repositories in Batch */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-4">Repositories</h3>
                {batchRepositories.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">No repositories in this batch</div>
                ) : (
                  <div className="space-y-2">
                    {batchRepositories.map((repo) => (
                      <RepositoryListItem key={repo.id} repository={repo} />
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="bg-white rounded-lg shadow-sm p-6 text-center text-gray-500">
              Select a batch to view details
            </div>
          )}
        </div>
      </div>

      {/* Modals */}
      {showCreateModal && (
        <CreateBatchModal onClose={() => setShowCreateModal(false)} onSuccess={handleBatchCreated} />
      )}

      {showEditModal && selectedBatch && (
        <EditBatchModal
          batch={selectedBatch}
          onClose={() => setShowEditModal(false)}
          onSuccess={handleBatchUpdated}
        />
      )}

      {showRetryModal && selectedBatch && (
        <RetryModal
          batch={selectedBatch}
          failedRepos={failedRepos}
          onClose={() => setShowRetryModal(false)}
          onSuccess={handleRetryComplete}
        />
      )}
    </div>
  );
}

interface BatchCardProps {
  batch: Batch;
  isSelected: boolean;
  onClick: () => void;
  onStart: () => void;
}

function BatchCard({ batch, isSelected, onClick, onStart }: BatchCardProps) {
  return (
    <div
      className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
        isSelected
          ? 'border-blue-500 bg-blue-50'
          : 'border-gray-200 hover:border-gray-300'
      }`}
      onClick={onClick}
    >
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <h3 className="font-medium text-gray-900">{batch.name}</h3>
          <div className="flex gap-2 mt-2">
            <StatusBadge status={batch.status} size="sm" />
            <span className="text-xs text-gray-600">{batch.repository_count} repos</span>
          </div>
        </div>
        {batch.status === 'ready' && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onStart();
            }}
            className="text-sm px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Start
          </button>
        )}
      </div>
    </div>
  );
}

function RepositoryListItem({ repository }: { repository: Repository }) {
  return (
    <div className="flex justify-between items-center p-3 border border-gray-200 rounded-lg hover:bg-gray-50">
      <div>
        <div className="font-medium text-gray-900">{repository.full_name}</div>
        <div className="text-sm text-gray-600">
          {formatBytes(repository.total_size)} • {repository.branch_count} branches
        </div>
      </div>
      <StatusBadge status={repository.status} size="sm" />
    </div>
  );
}

interface CreateBatchModalProps {
  onClose: () => void;
  onSuccess: () => void;
}

function CreateBatchModal({ onClose, onSuccess }: CreateBatchModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [scheduledAt, setScheduledAt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Batch name is required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await api.createBatch({
        name: name.trim(),
        description: description.trim() || undefined,
        type: 'batch', // Default type
        scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
      });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create batch');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="flex justify-between items-center p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">Create New Batch</h2>
          <button
            onClick={onClose}
            disabled={loading}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6">
          <div className="space-y-4">
            <div>
              <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
                Batch Name *
              </label>
              <input
                id="name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Pilot, Wave 1, Wave 2, Q1 Migration"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
                required
              />
            </div>

            <div>
              <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Optional description"
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
              />
            </div>

            <div>
              <label htmlFor="scheduledAt" className="block text-sm font-medium text-gray-700 mb-1">
                Scheduled Date (Optional)
              </label>
              <input
                id="scheduledAt"
                type="datetime-local"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
              />
            </div>
          </div>

          {error && (
            <div className="mt-4 bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-3 mt-6">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? 'Creating...' : 'Create Batch'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

interface EditBatchModalProps {
  batch: Batch;
  onClose: () => void;
  onSuccess: () => void;
}

function EditBatchModal({ batch, onClose, onSuccess }: EditBatchModalProps) {
  const [name, setName] = useState(batch.name);
  const [description, setDescription] = useState(batch.description || '');
  const [scheduledAt, setScheduledAt] = useState(
    batch.scheduled_at ? new Date(batch.scheduled_at).toISOString().slice(0, 16) : ''
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Repository browser state
  const [showRepoSection, setShowRepoSection] = useState(false);
  const [availableRepos, setAvailableRepos] = useState<Repository[]>([]);
  const [repoSearch, setRepoSearch] = useState('');
  const [repoLoading, setRepoLoading] = useState(false);
  const [selectedRepoIds, setSelectedRepoIds] = useState<Set<number>>(new Set());
  const [currentRepos, setCurrentRepos] = useState<Repository[]>([]);

  const loadAvailableRepos = useCallback(async () => {
    setRepoLoading(true);
    try {
      const repos = await api.listRepositories({
        available_for_batch: true,
        search: repoSearch || undefined,
        limit: 100,
      });
      setAvailableRepos(repos);
    } catch (err) {
      console.error('Failed to load available repositories:', err);
    } finally {
      setRepoLoading(false);
    }
  }, [repoSearch]);

  const loadCurrentRepos = useCallback(async () => {
    try {
      const repos = await api.listRepositories({ batch_id: batch.id });
      setCurrentRepos(repos);
    } catch (err) {
      console.error('Failed to load current repositories:', err);
    }
  }, [batch.id]);

  useEffect(() => {
    if (showRepoSection) {
      loadAvailableRepos();
      loadCurrentRepos();
    }
  }, [showRepoSection, loadAvailableRepos, loadCurrentRepos]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Batch name is required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await api.updateBatch(batch.id, {
        name: name.trim(),
        description: description.trim() || undefined,
        scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
      });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update batch');
    } finally {
      setLoading(false);
    }
  };

  const handleAddRepos = async () => {
    if (selectedRepoIds.size === 0) return;

    setLoading(true);
    try {
      await api.addRepositoriesToBatch(batch.id, Array.from(selectedRepoIds));
      setSelectedRepoIds(new Set());
      await loadAvailableRepos();
      await loadCurrentRepos();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add repositories');
    } finally {
      setLoading(false);
    }
  };

  const handleRemoveRepos = async () => {
    if (selectedRepoIds.size === 0) return;

    setLoading(true);
    try {
      await api.removeRepositoriesFromBatch(batch.id, Array.from(selectedRepoIds));
      setSelectedRepoIds(new Set());
      await loadAvailableRepos();
      await loadCurrentRepos();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove repositories');
    } finally {
      setLoading(false);
    }
  };

  const toggleRepoSelection = (repoId: number) => {
    const newSelection = new Set(selectedRepoIds);
    if (newSelection.has(repoId)) {
      newSelection.delete(repoId);
    } else {
      newSelection.add(repoId);
    }
    setSelectedRepoIds(newSelection);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 overflow-y-auto">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 my-8">
        <div className="flex justify-between items-center p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">Edit Batch</h2>
          <button
            onClick={onClose}
            disabled={loading}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6">
          <div className="space-y-4">
            <div>
              <label htmlFor="edit-name" className="block text-sm font-medium text-gray-700 mb-1">
                Batch Name *
              </label>
              <input
                id="edit-name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Pilot, Wave 1, Wave 2, Q1 Migration"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
                required
              />
            </div>

            <div>
              <label htmlFor="edit-description" className="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <textarea
                id="edit-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
              />
            </div>

            <div>
              <label htmlFor="edit-scheduledAt" className="block text-sm font-medium text-gray-700 mb-1">
                Scheduled Date (Optional)
              </label>
              <input
                id="edit-scheduledAt"
                type="datetime-local"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
              />
            </div>

            {/* Repository Management Section */}
            <div className="border-t pt-4 mt-4">
              <button
                type="button"
                onClick={() => setShowRepoSection(!showRepoSection)}
                className="flex items-center justify-between w-full text-left"
              >
                <h3 className="text-lg font-medium text-gray-900">
                  Manage Repositories ({currentRepos.length})
                </h3>
                <svg
                  className={`w-5 h-5 transform transition-transform ${showRepoSection ? 'rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {showRepoSection && (
                <div className="mt-4 space-y-4">
                  <div>
                    <h4 className="text-sm font-medium text-gray-700 mb-2">Current Repositories</h4>
                    <div className="max-h-48 overflow-y-auto border border-gray-200 rounded-lg">
                      {currentRepos.length === 0 ? (
                        <div className="p-4 text-sm text-gray-500 text-center">No repositories assigned</div>
                      ) : (
                        <div className="divide-y">
                          {currentRepos.map((repo) => (
                            <label key={repo.id} className="flex items-center p-3 hover:bg-gray-50 cursor-pointer">
                              <input
                                type="checkbox"
                                checked={selectedRepoIds.has(repo.id)}
                                onChange={() => toggleRepoSelection(repo.id)}
                                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                              />
                              <div className="ml-3 flex-1">
                                <div className="text-sm font-medium text-gray-900">{repo.full_name}</div>
                                <div className="text-xs text-gray-500">
                                  <StatusBadge status={repo.status} size="sm" />
                                </div>
                              </div>
                            </label>
                          ))}
                        </div>
                      )}
                    </div>
                    {currentRepos.length > 0 && (
                      <button
                        type="button"
                        onClick={handleRemoveRepos}
                        disabled={selectedRepoIds.size === 0 || loading}
                        className="mt-2 px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
                      >
                        Remove Selected ({selectedRepoIds.size})
                      </button>
                    )}
                  </div>

                  <div>
                    <h4 className="text-sm font-medium text-gray-700 mb-2">Available Repositories</h4>
                    <input
                      type="text"
                      placeholder="Search repositories..."
                      value={repoSearch}
                      onChange={(e) => setRepoSearch(e.target.value)}
                      className="w-full px-3 py-2 mb-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <div className="max-h-48 overflow-y-auto border border-gray-200 rounded-lg">
                      {repoLoading ? (
                        <div className="p-4 text-center"><LoadingSpinner /></div>
                      ) : availableRepos.length === 0 ? (
                        <div className="p-4 text-sm text-gray-500 text-center">No available repositories</div>
                      ) : (
                        <div className="divide-y">
                          {availableRepos
                            .filter((r) => !currentRepos.some((cr) => cr.id === r.id))
                            .map((repo) => (
                              <label key={repo.id} className="flex items-center p-3 hover:bg-gray-50 cursor-pointer">
                                <input
                                  type="checkbox"
                                  checked={selectedRepoIds.has(repo.id)}
                                  onChange={() => toggleRepoSelection(repo.id)}
                                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                                />
                                <div className="ml-3 flex-1">
                                  <div className="text-sm font-medium text-gray-900">{repo.full_name}</div>
                                  <div className="text-xs text-gray-500">
                                    <StatusBadge status={repo.status} size="sm" />
                                    {repo.batch_id && <span className="ml-2">• Batch #{repo.batch_id}</span>}
                                  </div>
                                </div>
                              </label>
                            ))}
                        </div>
                      )}
                    </div>
                    {availableRepos.filter((r) => !currentRepos.some((cr) => cr.id === r.id)).length > 0 && (
                      <button
                        type="button"
                        onClick={handleAddRepos}
                        disabled={selectedRepoIds.size === 0 || loading}
                        className="mt-2 px-3 py-1 text-sm bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                      >
                        Add Selected ({selectedRepoIds.size})
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>

          {error && (
            <div className="mt-4 bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-3 mt-6">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? 'Updating...' : 'Update Batch'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

interface RetryModalProps {
  batch: Batch;
  failedRepos: Repository[];
  onClose: () => void;
  onSuccess: () => void;
}

function RetryModal({ batch, failedRepos, onClose, onSuccess }: RetryModalProps) {
  const [selectedRepoIds, setSelectedRepoIds] = useState<Set<number>>(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleRetry = async (retryAll: boolean) => {
    setLoading(true);
    setError(null);

    try {
      await api.retryBatchFailures(
        batch.id,
        retryAll ? undefined : Array.from(selectedRepoIds)
      );
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retry repositories');
    } finally {
      setLoading(false);
    }
  };

  const toggleRepoSelection = (repoId: number) => {
    const newSelection = new Set(selectedRepoIds);
    if (newSelection.has(repoId)) {
      newSelection.delete(repoId);
    } else {
      newSelection.add(repoId);
    }
    setSelectedRepoIds(newSelection);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4">
        <div className="flex justify-between items-center p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">Retry Failed Repositories</h2>
          <button
            onClick={onClose}
            disabled={loading}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-6">
          <p className="text-sm text-gray-600 mb-4">
            Select repositories to retry or retry all failed repositories in this batch.
          </p>

          <div className="max-h-64 overflow-y-auto border border-gray-200 rounded-lg mb-4">
            {failedRepos.map((repo) => (
              <label key={repo.id} className="flex items-center p-3 hover:bg-gray-50 cursor-pointer border-b last:border-0">
                <input
                  type="checkbox"
                  checked={selectedRepoIds.has(repo.id)}
                  onChange={() => toggleRepoSelection(repo.id)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  disabled={loading}
                />
                <div className="ml-3 flex-1">
                  <div className="text-sm font-medium text-gray-900">{repo.full_name}</div>
                  <div className="text-xs text-gray-500">
                    <StatusBadge status={repo.status} size="sm" />
                  </div>
                </div>
              </label>
            ))}
          </div>

          {error && (
            <div className="mb-4 bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              onClick={() => handleRetry(false)}
              disabled={selectedRepoIds.size === 0 || loading}
              className="px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 disabled:opacity-50"
            >
              Retry Selected ({selectedRepoIds.size})
            </button>
            <button
              onClick={() => handleRetry(true)}
              disabled={loading}
              className="px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 disabled:opacity-50"
            >
              Retry All ({failedRepos.length})
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
