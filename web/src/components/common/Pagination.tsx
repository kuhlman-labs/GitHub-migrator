import { Pagination as PrimerPagination } from '@primer/react';

interface PaginationProps {
  currentPage: number;
  totalItems: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

export function Pagination({ currentPage, totalItems, pageSize, onPageChange }: PaginationProps) {
  const totalPages = Math.ceil(totalItems / pageSize);
  
  if (totalPages <= 1) {
    return null;
  }

  const startItem = (currentPage - 1) * pageSize + 1;
  const endItem = Math.min(currentPage * pageSize, totalItems);

  return (
    <div 
      className="flex items-center justify-between px-4 py-3 rounded-b-lg"
      style={{
        borderTop: '1px solid var(--borderColor-default)',
        backgroundColor: 'var(--bgColor-default)'
      }}
    >
      <div className="hidden sm:block">
          <p className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
            Showing <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>{startItem}</span> to{' '}
            <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>{endItem}</span> of{' '}
            <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>{totalItems}</span> results
          </p>
        </div>
      
        <div>
        <PrimerPagination
          pageCount={totalPages}
          currentPage={currentPage}
          onPageChange={(_e, page) => onPageChange(page)}
          showPages={{ narrow: false }}
        />
      </div>
    </div>
  );
}

