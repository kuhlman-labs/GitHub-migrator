import { useNavigate, useParams } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { api } from '../../services/api';
import type { Batch } from '../../types';
import { BatchBuilder } from './BatchBuilder';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { ErrorBoundary } from '../common/ErrorBoundary';
import { useToast } from '../../contexts/ToastContext';
import { handleApiError } from '../../utils/errorHandler';

export function BatchBuilderPage() {
  const { showError } = useToast();
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batchId]);

  const loadBatch = async () => {
    if (!batchId) return;

    setLoading(true);
    setError(null);

    try {
      const batchData = await api.getBatch(parseInt(batchId, 10));
      setBatch(batchData);
    } catch (err) {
      handleApiError(err, showError, 'Failed to load batch');
      setError('Failed to load batch. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    navigate('/batches');
  };

  const handleSuccess = () => {
    // Navigate with state to trigger immediate refresh
    navigate('/batches', { state: { refreshData: true } });
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
        <LoadingSpinner />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen p-8" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
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
    <div 
      className="flex flex-col -mx-4 sm:-mx-6 lg:-mx-8 -my-8"
      style={{ 
        backgroundColor: 'var(--bgColor-muted)',
        height: 'calc(100vh - 4rem)' // 4rem for navigation only, negative margins cancel PageLayout padding
      }}
    >
      <div 
        className="border-b shadow-sm flex-shrink-0"
        style={{
          borderColor: 'var(--borderColor-default)',
          backgroundColor: 'var(--bgColor-default)'
        }}
      >
        <div className="max-w-full px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold" style={{ color: 'var(--fgColor-default)' }}>
                {isEditing ? 'Edit Batch' : 'Create New Batch'}
              </h1>
              <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                {isEditing
                  ? 'Modify repositories and batch settings'
                  : 'Select repositories and configure your migration batch'}
              </p>
            </div>
            <button
              onClick={handleClose}
              className="px-4 py-2 border rounded-lg transition-colors"
              style={{
                color: 'var(--fgColor-default)',
                borderColor: 'var(--borderColor-default)',
                backgroundColor: 'var(--control-bgColor-rest)'
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-hidden">
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

