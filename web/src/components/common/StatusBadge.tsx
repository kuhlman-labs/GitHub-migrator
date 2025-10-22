interface StatusBadgeProps {
  status: string;
  size?: 'sm' | 'md';
}

export function StatusBadge({ status, size = 'md' }: StatusBadgeProps) {
  const getStatusColor = (status: string) => {
    // Map all backend statuses to GitHub color scheme
    const colors: Record<string, string> = {
      // Pending / Ready (neutral gray)
      pending: 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default',
      ready: 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default',
      
      // In Progress (blue)
      dry_run_queued: 'bg-gh-blue text-white',
      dry_run_in_progress: 'bg-gh-blue text-white',
      pre_migration: 'bg-gh-blue text-white',
      archive_generating: 'bg-gh-blue text-white',
      queued_for_migration: 'bg-gh-blue text-white',
      migrating_content: 'bg-gh-blue text-white',
      post_migration: 'bg-gh-blue text-white',
      in_progress: 'bg-gh-blue text-white',
      
      // Complete (green)
      dry_run_complete: 'bg-gh-success text-white',
      migration_complete: 'bg-gh-success text-white',
      complete: 'bg-gh-success text-white',
      completed: 'bg-gh-success text-white',
      
      // Partial Success / Warnings (yellow/orange)
      completed_with_errors: 'bg-gh-warning text-white',
      rolled_back: 'bg-gh-warning text-white',
      
      // Failed (red)
      dry_run_failed: 'bg-gh-danger text-white',
      migration_failed: 'bg-gh-danger text-white',
      failed: 'bg-gh-danger text-white',
      
      // Cancelled (gray)
      cancelled: 'bg-gh-text-secondary text-white',
      
      // Won't Migrate (muted)
      wont_migrate: 'bg-gray-500 text-white',
    };
    return colors[status] || 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default';
  };
  
  const sizeClasses = size === 'sm' 
    ? 'text-xs px-2 h-5' 
    : 'text-xs px-3 h-6';
  
  return (
    <span className={`inline-flex items-center rounded-full font-medium ${getStatusColor(status)} ${sizeClasses}`}>
      {status.replace(/_/g, ' ')}
    </span>
  );
}

