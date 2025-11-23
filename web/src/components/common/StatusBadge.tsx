import { Label } from '@primer/react';

interface StatusBadgeProps {
  status: string;
  size?: 'small' | 'large';
}

export function StatusBadge({ status, size = 'large' }: StatusBadgeProps) {
  const getStatusVariant = (status: string): 'default' | 'primary' | 'secondary' | 'accent' | 'success' | 'attention' | 'severe' | 'danger' | 'done' | 'sponsors' => {
    // Map all backend statuses to Primer Label variants
    const statusMap: Record<string, 'default' | 'primary' | 'secondary' | 'accent' | 'success' | 'attention' | 'severe' | 'danger' | 'done' | 'sponsors'> = {
      // Pending / Ready (neutral)
      pending: 'default',
      ready: 'default',
      
      // Requires Attention (attention/warning)
      remediation_required: 'attention',
      
      // In Progress (accent/blue)
      dry_run_queued: 'accent',
      dry_run_in_progress: 'accent',
      pre_migration: 'accent',
      archive_generating: 'accent',
      queued_for_migration: 'accent',
      migrating_content: 'accent',
      post_migration: 'accent',
      in_progress: 'accent',
      
      // Complete (success/green)
      dry_run_complete: 'success',
      migration_complete: 'success',
      complete: 'success',
      completed: 'success',
      
      // Partial Success / Warnings (attention)
      completed_with_errors: 'attention',
      rolled_back: 'attention',
      
      // Failed (danger/red)
      dry_run_failed: 'danger',
      migration_failed: 'danger',
      failed: 'danger',
      
      // Cancelled (secondary)
      cancelled: 'secondary',
      
      // Won't Migrate (secondary)
      wont_migrate: 'secondary',
    };
    return statusMap[status] || 'default';
  };
  
  return (
    <Label variant={getStatusVariant(status)} size={size}>
      {status.replace(/_/g, ' ')}
    </Label>
  );
}

