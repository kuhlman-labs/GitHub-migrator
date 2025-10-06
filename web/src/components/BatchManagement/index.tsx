import { useEffect, useState } from 'react';
import { api } from '../../services/api';
import type { Batch, Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { formatBytes, formatDate } from '../../utils/format';

export function BatchManagement() {
  const [batches, setBatches] = useState<Batch[]>([]);
  const [selectedBatch, setSelectedBatch] = useState<Batch | null>(null);
  const [batchRepositories, setBatchRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);

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

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-light text-gray-900">Batch Management</h1>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">
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
                    <Badge color="blue">{selectedBatch.type}</Badge>
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

                {selectedBatch.status === 'ready' && (
                  <button
                    onClick={() => handleStartBatch(selectedBatch.id)}
                    className="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700"
                  >
                    Start Batch Migration
                  </button>
                )}
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

