import { useNavigate } from 'react-router-dom';
import { UnderlineNav } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { PackageIcon } from '@primer/octicons-react';
import type { Batch } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { Pagination } from '../common/Pagination';
import { BatchCard } from './BatchCard';

type BatchTab = 'active' | 'completed';

interface BatchListPanelProps {
  batches: Batch[];
  loading: boolean;
  activeTab: BatchTab;
  onTabChange: (tab: BatchTab) => void;
  selectedBatchId: number | null;
  onSelectBatch: (batchId: number) => void;
  onStartBatch: (batchId: number) => void;
  searchTerm?: string;
  currentPage: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

export function BatchListPanel({
  batches,
  loading,
  activeTab,
  onTabChange,
  selectedBatchId,
  onSelectBatch,
  onStartBatch,
  searchTerm = '',
  currentPage,
  pageSize,
  onPageChange,
}: BatchListPanelProps) {
  const navigate = useNavigate();

  // Filter batches by tab
  const activeBatches = batches.filter((b) => 
    ['pending', 'ready', 'in_progress', 'scheduled'].includes(b.status)
  );
  const completedBatches = batches.filter((b) => 
    ['completed', 'completed_with_errors', 'failed', 'cancelled'].includes(b.status)
  );

  const filteredBatches = activeTab === 'active' ? activeBatches : completedBatches;

  // Apply search filter
  const searchFiltered = searchTerm
    ? filteredBatches.filter((b) =>
        b.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (b.description && b.description.toLowerCase().includes(searchTerm.toLowerCase()))
      )
    : filteredBatches;

  // Paginate
  const totalItems = searchFiltered.length;
  const startIndex = (currentPage - 1) * pageSize;
  const paginatedBatches = searchFiltered.slice(startIndex, startIndex + pageSize);

  return (
    <div 
      className="rounded-lg border"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)'
      }}
    >
      {/* Tabs */}
      <UnderlineNav aria-label="Batch tabs">
        <UnderlineNav.Item
          aria-current={activeTab === 'active' ? 'page' : undefined}
          onSelect={() => onTabChange('active')}
        >
          Active ({activeBatches.length})
        </UnderlineNav.Item>
        <UnderlineNav.Item
          aria-current={activeTab === 'completed' ? 'page' : undefined}
          onSelect={() => onTabChange('completed')}
        >
          Completed ({completedBatches.length})
        </UnderlineNav.Item>
      </UnderlineNav>

      {/* Batch List */}
      <div className="p-4">
        {loading ? (
          <LoadingSpinner />
        ) : paginatedBatches.length === 0 ? (
          <Blankslate>
            <Blankslate.Visual>
              <PackageIcon size={48} />
            </Blankslate.Visual>
            <Blankslate.Heading>
              {searchTerm ? 'No batches match your search' : `No ${activeTab} batches`}
            </Blankslate.Heading>
            <Blankslate.Description>
              {searchTerm 
                ? 'Try a different search term to find batches.'
                : activeTab === 'active'
                ? 'Create a batch to group repositories for migration.'
                : 'Completed batches will appear here once migrations finish.'}
            </Blankslate.Description>
            {!searchTerm && activeTab === 'active' && (
              <Blankslate.PrimaryAction onClick={() => navigate('/batches/new')}>
                Create New Batch
              </Blankslate.PrimaryAction>
            )}
          </Blankslate>
        ) : (
          <>
            <div className="space-y-2 mb-4">
              {paginatedBatches.map((batch) => (
                <BatchCard
                  key={batch.id}
                  batch={batch}
                  isSelected={selectedBatchId === batch.id}
                  onClick={() => onSelectBatch(batch.id)}
                  onStart={() => onStartBatch(batch.id)}
                />
              ))}
            </div>
            {totalItems > pageSize && (
              <Pagination
                currentPage={currentPage}
                totalItems={totalItems}
                pageSize={pageSize}
                onPageChange={onPageChange}
              />
            )}
          </>
        )}
      </div>
    </div>
  );
}

