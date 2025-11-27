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
  const colorStyles = {
    blue: 'var(--fgColor-accent)',
    green: 'var(--fgColor-success)',
    purple: 'var(--fgColor-done)',
    yellow: 'var(--fgColor-attention)',
  };
  
  const iconBgStyles = {
    blue: 'var(--bgColor-accent-muted)',
    green: 'var(--bgColor-success-muted)',
    purple: 'var(--bgColor-done-muted)',
    yellow: 'var(--bgColor-attention-muted)',
  };

  const borderColorStyles = {
    blue: 'var(--borderColor-accent-emphasis)',
    green: 'var(--borderColor-success-emphasis)',
    purple: 'var(--borderColor-done-emphasis)',
    yellow: 'var(--borderColor-attention-emphasis)',
  };

  const isClickable = !!onClick;
  const baseClasses = `rounded-lg border p-6 relative group transition-all border-l-4`;
  const clickableClasses = isClickable 
    ? "cursor-pointer hover:shadow-lg" 
    : "";

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
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        borderLeftColor: borderColorStyles[color],
        boxShadow: 'var(--shadow-resting-small)'
      }}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      role={isClickable ? 'button' : undefined}
      tabIndex={isClickable ? 0 : undefined}
      aria-label={isClickable ? `View repositories: ${title}` : undefined}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <h3 className="text-xs font-semibold mb-2 uppercase tracking-wide" style={{ color: 'var(--fgColor-muted)' }}>{title}</h3>
          <div className="text-3xl font-semibold mb-1" style={{ color: colorStyles[color] }}>
            {value}
          </div>
          {subtitle && <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>{subtitle}</div>}
        </div>
        {icon && (
          <div className="p-3 rounded-lg" style={{ backgroundColor: iconBgStyles[color] }}>
            {icon}
          </div>
        )}
      </div>
      
      {tooltip && (
        <div className="absolute top-2 right-2">
          <div className="relative group/tooltip">
            <span className="cursor-help" style={{ color: 'var(--fgColor-muted)' }}>
              <InfoIcon size={16} />
            </span>
            <div 
              className="invisible group-hover/tooltip:visible absolute right-0 top-6 w-64 p-3 text-xs rounded-md shadow-lg z-10 border"
              style={{
                backgroundColor: 'var(--overlay-bgColor)',
                color: 'var(--fgColor-default)',
                borderColor: 'var(--borderColor-default)'
              }}
            >
              {tooltip}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

