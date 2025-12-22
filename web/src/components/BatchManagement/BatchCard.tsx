import { CalendarIcon } from '@primer/octicons-react';
import type { Batch } from '../../types';
import { StatusBadge } from '../common/StatusBadge';
import { formatDate } from '../../utils/format';

export interface BatchCardProps {
  batch: Batch;
  isSelected: boolean;
  onClick: () => void;
  onStart: () => void;
}

export function BatchCard({ batch, isSelected, onClick, onStart }: BatchCardProps) {
  return (
    <div
      className="p-4 rounded-lg border-2 cursor-pointer transition-all"
      style={
        isSelected
          ? { borderColor: 'var(--accent-emphasis)', backgroundColor: 'var(--accent-subtle)' }
          : { borderColor: 'var(--borderColor-default)', backgroundColor: 'var(--bgColor-default)' }
      }
      onClick={onClick}
    >
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <h3 className="font-medium" style={{ color: 'var(--fgColor-default)' }}>
            {batch.name}
          </h3>
          <div className="flex gap-2 mt-2">
            <StatusBadge status={batch.status} size="small" />
            <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
              {batch.repository_count} repos
            </span>
          </div>
          {batch.scheduled_at && (
            <div
              className="mt-1.5 text-xs flex items-center gap-1"
              style={{ color: 'var(--fgColor-accent)' }}
            >
              <CalendarIcon size={12} />
              {formatDate(batch.scheduled_at)}
            </div>
          )}
        </div>
        {batch.status === 'ready' && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onStart();
            }}
            className="text-sm px-3 py-1 rounded border-0 transition-all cursor-pointer"
            style={{
              backgroundColor: 'var(--bgColor-success-emphasis)',
              color: 'var(--fgColor-onEmphasis)',
              fontWeight: 500,
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.opacity = '0.9';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.opacity = '1';
            }}
          >
            Start
          </button>
        )}
        {batch.status === 'pending' && (
          <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            Dry run needed
          </span>
        )}
      </div>
    </div>
  );
}

