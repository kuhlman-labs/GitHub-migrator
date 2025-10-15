interface KPICardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  color?: 'blue' | 'green' | 'purple' | 'yellow';
  icon?: React.ReactNode;
  tooltip?: string;
}

export function KPICard({ title, value, subtitle, color = 'blue', icon, tooltip }: KPICardProps) {
  const colorClasses = {
    blue: 'text-blue-600 bg-blue-50',
    green: 'text-green-600 bg-green-50',
    purple: 'text-purple-600 bg-purple-50',
    yellow: 'text-yellow-600 bg-yellow-50',
  };

  return (
    <div className="bg-white rounded-lg shadow-sm p-6 relative group">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <h3 className="text-sm font-medium text-gray-600 mb-2">{title}</h3>
          <div className={`text-3xl font-light mb-1 ${colorClasses[color]}`}>
            {value}
          </div>
          {subtitle && <div className="text-sm text-gray-500">{subtitle}</div>}
        </div>
        {icon && (
          <div className={`p-3 rounded-lg ${colorClasses[color]}`}>
            {icon}
          </div>
        )}
      </div>
      
      {tooltip && (
        <div className="absolute top-2 right-2">
          <div className="relative group/tooltip">
            <svg 
              className="w-4 h-4 text-gray-400 hover:text-gray-600 cursor-help" 
              fill="none" 
              stroke="currentColor" 
              viewBox="0 0 24 24"
            >
              <path 
                strokeLinecap="round" 
                strokeLinejoin="round" 
                strokeWidth={2} 
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" 
              />
            </svg>
            <div className="invisible group-hover/tooltip:visible absolute right-0 top-6 w-64 p-2 bg-gray-900 text-white text-xs rounded shadow-lg z-10">
              {tooltip}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

