import { useEffect, useState } from 'react';

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
    let timer: NodeJS.Timeout;

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
        <div className="flex items-center gap-1.5 text-gray-400">
          <svg 
            className="animate-spin h-3.5 w-3.5" 
            xmlns="http://www.w3.org/2000/svg" 
            fill="none" 
            viewBox="0 0 24 24"
          >
            <circle 
              className="opacity-25" 
              cx="12" 
              cy="12" 
              r="10" 
              stroke="currentColor" 
              strokeWidth="4"
            />
            <path 
              className="opacity-75" 
              fill="currentColor" 
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
        </div>
      </div>
    );
  }

  // Original style indicator (for when subtle = false)
  return (
    <div className="absolute top-4 right-4 z-10">
      <div className="flex items-center gap-2 bg-blue-50 text-blue-600 px-3 py-1.5 rounded-full shadow-sm">
        <svg 
          className="animate-spin h-4 w-4" 
          xmlns="http://www.w3.org/2000/svg" 
          fill="none" 
          viewBox="0 0 24 24"
        >
          <circle 
            className="opacity-25" 
            cx="12" 
            cy="12" 
            r="10" 
            stroke="currentColor" 
            strokeWidth="4"
          />
          <path 
            className="opacity-75" 
            fill="currentColor" 
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
        <span className="text-sm font-medium">Updating...</span>
      </div>
    </div>
  );
};

