interface StatusBadgeProps {
  status: string;
  size?: 'sm' | 'md';
}

export function StatusBadge({ status, size = 'md' }: StatusBadgeProps) {
  const getStatusColor = (status: string) => {
    const normalizedStatus = status.toLowerCase().replace(/_/g, ' ');
    
    if (normalizedStatus.includes('complete')) return 'bg-green-100 text-green-800';
    if (normalizedStatus.includes('failed')) return 'bg-red-100 text-red-800';
    if (normalizedStatus.includes('progress') || normalizedStatus.includes('migrating')) {
      return 'bg-blue-100 text-blue-800';
    }
    if (normalizedStatus === 'pending') return 'bg-gray-100 text-gray-800';
    if (normalizedStatus.includes('queued')) return 'bg-yellow-100 text-yellow-800';
    if (normalizedStatus.includes('dry run')) return 'bg-purple-100 text-purple-800';
    
    return 'bg-gray-100 text-gray-800';
  };
  
  const sizeClasses = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-3 py-1';
  
  return (
    <span className={`inline-flex items-center rounded-full font-medium ${getStatusColor(status)} ${sizeClasses}`}>
      {status.replace(/_/g, ' ')}
    </span>
  );
}

