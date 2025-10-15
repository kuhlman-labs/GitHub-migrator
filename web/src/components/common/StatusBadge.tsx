interface StatusBadgeProps {
  status: string;
  size?: 'sm' | 'md';
}

export function StatusBadge({ status, size = 'md' }: StatusBadgeProps) {
  const getStatusColor = (status: string) => {
    // Map all backend statuses to simplified color categories
    const colors: Record<string, string> = {
      // Pending / Ready (gray)
      pending: 'bg-gray-100 text-gray-800',
      ready: 'bg-gray-100 text-gray-800',
      
      // In Progress (blue)
      dry_run_queued: 'bg-blue-100 text-blue-800',
      dry_run_in_progress: 'bg-blue-100 text-blue-800',
      pre_migration: 'bg-blue-100 text-blue-800',
      archive_generating: 'bg-blue-100 text-blue-800',
      queued_for_migration: 'bg-blue-100 text-blue-800',
      migrating_content: 'bg-blue-100 text-blue-800',
      post_migration: 'bg-blue-100 text-blue-800',
      in_progress: 'bg-blue-100 text-blue-800',
      
      // Complete (green)
      dry_run_complete: 'bg-green-100 text-green-800',
      migration_complete: 'bg-green-100 text-green-800',
      complete: 'bg-green-100 text-green-800',
      completed: 'bg-green-100 text-green-800',
      
      // Partial Success / Warnings (yellow)
      completed_with_errors: 'bg-yellow-100 text-yellow-800',
      rolled_back: 'bg-yellow-100 text-yellow-800',
      
      // Failed (red)
      dry_run_failed: 'bg-red-100 text-red-800',
      migration_failed: 'bg-red-100 text-red-800',
      failed: 'bg-red-100 text-red-800',
      
      // Cancelled (gray)
      cancelled: 'bg-gray-100 text-gray-800',
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
  };
  
  const sizeClasses = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-3 py-1';
  
  return (
    <span className={`inline-flex items-center rounded-full font-medium ${getStatusColor(status)} ${sizeClasses}`}>
      {status.replace(/_/g, ' ')}
    </span>
  );
}

