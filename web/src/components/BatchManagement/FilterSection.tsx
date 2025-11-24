import { useState, ReactNode } from 'react';
import { ChevronDownIcon } from '@primer/octicons-react';

interface FilterSectionProps {
  title: string;
  children: ReactNode;
  defaultExpanded?: boolean;
}

export function FilterSection({ title, children, defaultExpanded = true }: FilterSectionProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  return (
    <div className="last:border-b-0" style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between py-3 px-4 transition-opacity hover:opacity-80"
      >
        <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>{title}</span>
        <span style={{ color: 'var(--fgColor-muted)' }}>
        <ChevronDownIcon
            className={`transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          size={16}
        />
        </span>
      </button>
      {isExpanded && (
        <div className="px-4 pb-4 space-y-3">
          {children}
        </div>
      )}
    </div>
  );
}

