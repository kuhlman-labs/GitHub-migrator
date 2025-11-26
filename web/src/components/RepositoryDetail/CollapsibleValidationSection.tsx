import { ChevronDownIcon } from '@primer/octicons-react';

interface CollapsibleValidationSectionProps {
  id: string;
  title: string;
  status: 'blocking' | 'warning' | 'passed';
  expanded: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}

export function CollapsibleValidationSection({
  id,
  title,
  status,
  expanded,
  onToggle,
  children
}: CollapsibleValidationSectionProps) {
  const statusConfig = {
    blocking: { 
      color: 'red',
      borderColor: 'border-red-200',
      bgColor: 'bg-red-50',
      hoverColor: 'hover:bg-red-100',
      textColor: 'text-red-900',
      iconColor: 'text-red-600'
    },
    warning: { 
      color: 'yellow',
      borderColor: 'border-yellow-200',
      bgColor: 'bg-yellow-50',
      hoverColor: 'hover:bg-yellow-100',
      textColor: 'text-yellow-900',
      iconColor: 'text-yellow-600'
    },
    passed: { 
      color: 'green',
      borderColor: 'border-green-200',
      bgColor: 'bg-green-50',
      hoverColor: 'hover:bg-green-100',
      textColor: 'text-green-900',
      iconColor: 'text-green-600'
    }
  };
  
  const config = statusConfig[status];
  
  const renderIcon = () => {
    if (status === 'blocking') {
      return (
        <svg className={`w-5 h-5 ${config.iconColor} flex-shrink-0`} fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
        </svg>
      );
    } else if (status === 'warning') {
      return (
        <svg className={`w-5 h-5 ${config.iconColor} flex-shrink-0`} fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
        </svg>
      );
    } else {
      return (
        <svg className={`w-5 h-5 ${config.iconColor} flex-shrink-0`} fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    }
  };
  
  return (
    <div className={`border ${config.borderColor} rounded-lg mb-4`}>
      <button
        onClick={onToggle}
        className={`w-full flex items-center justify-between p-4 ${config.bgColor} ${config.hoverColor} transition-colors rounded-t-lg`}
        aria-expanded={expanded}
        aria-controls={`section-${id}`}
      >
        <div className="flex items-center gap-3">
          {renderIcon()}
          <h3 className={`text-lg font-medium ${config.textColor}`}>{title}</h3>
        </div>
        <ChevronDownIcon 
          className={`text-gray-500 transition-transform ${expanded ? 'rotate-180' : ''}`}
          size={16}
        />
      </button>
      
      {expanded && (
        <div id={`section-${id}`} className="p-4 border-t border-gray-200">
          {children}
        </div>
      )}
    </div>
  );
}

