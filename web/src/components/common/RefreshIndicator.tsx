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
    let timer: number | undefined;

    if (isRefreshing) {
      // Only show indicator if refresh takes longer than the delay
      timer = setTimeout(() => {
        setShowIndicator(true);
      }, delay);
    } else {
      setShowIndicator(false);
    }

    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [isRefreshing, delay]);

  if (!showIndicator) return null;

  // Subtle indicator - just a small spinner in the corner
  if (subtle) {
    return (
      <div className="absolute top-4 right-4 z-10">
        <Spinner size="small" />
      </div>
    );
  }

  // Original style indicator (for when subtle = false)
  return (
    <div className="absolute top-4 right-4 z-10">
      <div className="flex items-center gap-2 bg-blue-50 text-blue-600 px-3 py-1.5 rounded-full shadow-sm">
        <Spinner size="small" />
        <span className="text-sm font-medium">Updating...</span>
      </div>
    </div>
  );
};

