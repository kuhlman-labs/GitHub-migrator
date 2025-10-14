interface RefreshIndicatorProps {
  isRefreshing: boolean;
}

export const RefreshIndicator: React.FC<RefreshIndicatorProps> = ({ isRefreshing }) => {
  if (!isRefreshing) return null;

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

