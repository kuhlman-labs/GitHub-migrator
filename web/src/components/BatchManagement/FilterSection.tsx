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
    <div className="border-b border-gray-200 last:border-b-0">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between py-3 px-4 hover:bg-gray-50 transition-colors"
      >
        <span className="text-sm font-medium text-gray-900">{title}</span>
        <ChevronDownIcon
          className={`text-gray-500 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          size={16}
        />
      </button>
      {isExpanded && (
        <div className="px-4 pb-4 space-y-3">
          {children}
        </div>
      )}
    </div>
  );
}

