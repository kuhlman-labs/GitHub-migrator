interface BadgeProps {
  children: React.ReactNode;
  color?: 'blue' | 'green' | 'yellow' | 'red' | 'purple' | 'gray' | 'orange' | 'pink' | 'indigo' | 'teal';
}

export function Badge({ children, color = 'gray' }: BadgeProps) {
  const colorClasses = {
    blue: 'bg-gh-info-bg text-gh-blue border border-gh-blue/20',
    green: 'bg-gh-success-bg text-gh-success border border-gh-success/20',
    yellow: 'bg-gh-warning-bg text-gh-warning border border-gh-warning/20',
    red: 'bg-gh-danger-bg text-gh-danger border border-gh-danger/20',
    purple: 'bg-purple-100 text-purple-800 border border-purple-200',
    gray: 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default',
    orange: 'bg-orange-100 text-orange-800 border border-orange-200',
    pink: 'bg-pink-100 text-pink-800 border border-pink-200',
    indigo: 'bg-indigo-100 text-indigo-800 border border-indigo-200',
    teal: 'bg-teal-100 text-teal-800 border border-teal-200',
  };
  
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${colorClasses[color]}`}>
      {children}
    </span>
  );
}

