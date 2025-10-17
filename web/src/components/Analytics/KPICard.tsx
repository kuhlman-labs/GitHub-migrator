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
    blue: 'text-gh-blue',
    green: 'text-gh-success',
    purple: 'text-purple-700',
    yellow: 'text-gh-warning',
  };
  
  const iconBgClasses = {
    blue: 'bg-gh-info-bg',
    green: 'bg-gh-success-bg',
    purple: 'bg-purple-100',
    yellow: 'bg-gh-warning-bg',
  };

  return (
    <div className="bg-white rounded-lg border border-gh-border-default shadow-gh-card p-6 relative group hover:border-gh-border-hover transition-colors">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <h3 className="text-xs font-semibold text-gh-text-secondary mb-2 uppercase tracking-wide">{title}</h3>
          <div className={`text-3xl font-semibold mb-1 ${colorClasses[color]}`}>
            {value}
          </div>
          {subtitle && <div className="text-sm text-gh-text-secondary">{subtitle}</div>}
        </div>
        {icon && (
          <div className={`p-3 rounded-lg ${iconBgClasses[color]}`}>
            {icon}
          </div>
        )}
      </div>
      
      {tooltip && (
        <div className="absolute top-2 right-2">
          <div className="relative group/tooltip">
            <svg 
              className="w-4 h-4 text-gh-text-muted hover:text-gh-text-secondary cursor-help" 
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
            <div className="invisible group-hover/tooltip:visible absolute right-0 top-6 w-64 p-2 bg-gh-header-bg text-white text-xs rounded shadow-lg z-10">
              {tooltip}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

