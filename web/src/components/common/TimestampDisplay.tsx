import { formatTimestampWithStaleness } from '../../utils/format';

interface TimestampDisplayProps {
  timestamp: string | null | undefined;
  label?: string;
  size?: 'sm' | 'md';
  showLabel?: boolean;
}

export function TimestampDisplay({ 
  timestamp, 
  label = 'Updated', 
  size = 'md',
  showLabel = true 
}: TimestampDisplayProps) {
  if (!timestamp) {
    return null;
  }

  const { formatted, isStale, fullDate } = formatTimestampWithStaleness(timestamp, 30);
  
  const sizeClasses = size === 'sm' 
    ? 'text-xs' 
    : 'text-sm';
  
  const textColor = isStale 
    ? 'text-amber-600' 
    : 'text-gh-text-secondary';
  
  return (
    <span 
      className={`inline-flex items-center gap-1 ${sizeClasses} ${textColor}`}
      title={fullDate}
    >
      {isStale && (
        <svg 
          className="w-3.5 h-3.5" 
          fill="none" 
          stroke="currentColor" 
          viewBox="0 0 24 24"
        >
          <path 
            strokeLinecap="round" 
            strokeLinejoin="round" 
            strokeWidth={2} 
            d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" 
          />
        </svg>
      )}
      {showLabel && <span>{label}:</span>}
      <span className="font-medium">{formatted}</span>
    </span>
  );
}

