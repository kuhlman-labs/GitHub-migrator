import { useNavigate, useParams } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { api } from '../../services/api';
import type { Batch } from '../../types';
import { BatchBuilder } from './BatchBuilder';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { ErrorBoundary } from '../common/ErrorBoundary';

export function BatchBuilderPage() {
  const navigate = useNavigate();
  const { batchId } = useParams<{ batchId: string }>();
  const [batch, setBatch] = useState<Batch | undefined>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isEditing = !!batchId;

  useEffect(() => {
    if (batchId) {
      loadBatch();
    }
  }, [batchId]);

  const loadBatch = async () => {
    if (!batchId) return;

    setLoading(true);
    setError(null);

    try {
      const batchData = await api.getBatch(parseInt(batchId, 10));
      console.log('Raw batch data from API:', batchData);
      console.log('Batch structure check - has batch property?', 'batch' in (batchData as any));
      setBatch(batchData);
    } catch (err) {
      console.error('Failed to load batch:', err);
      setError('Failed to load batch. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    navigate('/batches');
  };

  const handleSuccess = () => {
    navigate('/batches');
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <LoadingSpinner />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 p-8">
        <div className="max-w-4xl mx-auto">
          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-red-900 mb-2">Error</h2>
            <p className="text-sm text-red-700 mb-4">{error}</p>
            <button
              onClick={() => navigate('/batches')}
              className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
            >
              Back to Batches
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-[calc(100vh-4rem)] bg-gray-50 flex flex-col overflow-hidden">
      <div className="border-b border-gray-200 bg-white shadow-sm flex-shrink-0">
        <div className="max-w-full px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                {isEditing ? 'Edit Batch' : 'Create New Batch'}
              </h1>
              <p className="text-sm text-gray-600 mt-1">
                {isEditing
                  ? 'Modify repositories and batch settings'
                  : 'Select repositories and configure your migration batch'}
              </p>
            </div>
            <button
              onClick={handleClose}
              className="px-4 py-2 text-gray-700 hover:text-gray-900 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-hidden">
        <ErrorBoundary>
          <BatchBuilder
            batch={batch}
            onClose={handleClose}
            onSuccess={handleSuccess}
          />
        </ErrorBoundary>
      </div>
    </div>
  );
}

