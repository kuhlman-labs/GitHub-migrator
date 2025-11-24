import { InfoIcon } from '@primer/octicons-react';

interface KPICardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  color?: 'blue' | 'green' | 'purple' | 'yellow';
  icon?: React.ReactNode;
  tooltip?: string;
  onClick?: () => void;
}

export function KPICard({ title, value, subtitle, color = 'blue', icon, tooltip, onClick }: KPICardProps) {
  const colorClasses = {
    blue: 'text-blue-600',
    green: 'text-green-600',
    purple: 'text-purple-600',
    yellow: 'text-orange-600',
  };
  
  const iconBgClasses = {
    blue: 'bg-blue-50',
    green: 'bg-green-50',
    purple: 'bg-purple-50',
    yellow: 'bg-orange-50',
  };

  const borderColorClasses = {
    blue: 'border-l-blue-500',
    green: 'border-l-green-500',
    purple: 'border-l-purple-500',
    yellow: 'border-l-orange-500',
  };

  const isClickable = !!onClick;
  const baseClasses = `bg-white rounded-lg border border-gh-border-default shadow-gh-card p-6 relative group transition-all border-l-4 ${borderColorClasses[color]}`;
  const clickableClasses = isClickable 
    ? "cursor-pointer hover:border-gh-blue hover:shadow-lg" 
    : "hover:border-gh-border-hover";

  const handleClick = () => {
    if (onClick) {
      onClick();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (onClick && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault();
      onClick();
    }
  };

  return (
    <div 
      className={`${baseClasses} ${clickableClasses}`}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      role={isClickable ? 'button' : undefined}
      tabIndex={isClickable ? 0 : undefined}
      aria-label={isClickable ? `View repositories: ${title}` : undefined}
    >
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
            <InfoIcon 
              size={16}
              className="text-gh-text-muted hover:text-gh-text-secondary cursor-help" 
            />
            <div className="invisible group-hover/tooltip:visible absolute right-0 top-6 w-64 p-2 bg-gh-header-bg text-white text-xs rounded shadow-lg z-10">
              {tooltip}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

