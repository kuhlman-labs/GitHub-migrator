import { useEffect, useState } from 'react';
import { Spinner } from '@primer/react';

interface RefreshIndicatorProps {
  isRefreshing: boolean;
  delay?: number; // Delay in ms before showing the indicator (default: 500ms)
  subtle?: boolean; // Use a more subtle, minimal indicator
}

export const RefreshIndicator: React.FC<RefreshIndicatorProps> = ({ 
  isRefreshing, 
  delay = 500,
  subtle = true 
}) => {
  const [showIndicator, setShowIndicator] = useState(false);

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;

    if (isRefreshing) {
      // Only show indicator if refresh takes longer than the delay
      timer = setTimeout(() => {
        setShowIndicator(true);
      }, delay);
    } else {
      // Use 0ms timeout to avoid synchronous setState in effect
      timer = setTimeout(() => {
        setShowIndicator(false);
      }, 0);
    }

    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [isRefreshing, delay]);

  if (!showIndicator) return null;

  // Subtle indicator - just a small spinner in the corner
  if (subtle) {
    return (
      <div className="absolute top-4 right-4 z-10" role="status" aria-live="polite">
        <Spinner size="small" aria-label="Refreshing data" />
      </div>
    );
  }

  // Original style indicator (for when subtle = false)
  return (
    <div className="absolute top-4 right-4 z-10" role="status" aria-live="polite">
      <div className="flex items-center gap-2 bg-gh-accent-subtle text-gh-accent-fg px-3 py-1.5 rounded-full shadow-sm">
        <Spinner size="small" aria-label="Refreshing data" />
        <span className="text-sm font-medium">Updating...</span>
      </div>
    </div>
  );
};

