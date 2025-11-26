import { useState } from 'react';
import { Text, Label } from '@primer/react';
import { ChevronDownIcon, ChevronRightIcon } from '@primer/octicons-react';

interface CollapsibleSectionProps {
  title: string;
  description?: string;
  isOptional?: boolean;
  defaultExpanded?: boolean;
  children: React.ReactNode;
}

export function CollapsibleSection({
  title,
  description,
  isOptional = false,
  defaultExpanded = false,
  children,
}: CollapsibleSectionProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <div
      className="border rounded-lg overflow-hidden mb-4"
      style={{ borderColor: 'var(--borderColor-default)' }}
    >
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between p-4 cursor-pointer"
        style={{
          backgroundColor: 'var(--bgColor-muted)',
          border: 'none',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = 'var(--bgColor-inset)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = 'var(--bgColor-muted)';
        }}
      >
        <div className="flex items-center gap-3">
          {expanded ? <ChevronDownIcon size={16} /> : <ChevronRightIcon size={16} />}
          <div className="text-left">
            <div className="flex items-center gap-3">
              <Text className="font-bold">{title}</Text>
              {isOptional && (
                <Label size="small" variant="accent">
                  Optional
                </Label>
              )}
            </div>
            {description && (
              <Text className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                {description}
              </Text>
            )}
          </div>
        </div>
      </button>

      {expanded && (
        <div
          className="p-4 border-t"
          style={{ borderColor: 'var(--borderColor-default)' }}
        >
          {children}
        </div>
      )}
    </div>
  );
}
