import { Link } from 'react-router-dom';
import { Label, ProgressBar } from '@primer/react';
import { AlertIcon, CheckCircleIcon } from '@primer/octicons-react';
import { Organization } from '../../types';

interface OrganizationProgressCardProps {
  organization: Organization;
}

export function OrganizationProgressCard({ organization }: OrganizationProgressCardProps) {
  const getProgressColor = (percentage: number): 'success.emphasis' | 'attention.emphasis' | 'default' => {
    if (percentage >= 80) return 'success.emphasis';
    if (percentage >= 40) return 'attention.emphasis';
    return 'default';
  };

  const hasFailures = organization.failed_count > 0;
  const progressPercentage = organization.migration_progress_percentage;

  return (
    <Link
      to={`/org/${encodeURIComponent(organization.organization)}`}
      className="block rounded-lg border transition-all hover:shadow-lg p-6"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)',
      }}
    >
      {/* Header */}
      <div className="mb-4">
        <h3 className="text-lg font-semibold mb-2 truncate" style={{ color: 'var(--fgColor-default)' }}>
          {organization.organization}
        </h3>
        <div className="flex flex-wrap gap-2">
          {organization.ado_organization && (
            <Label variant="accent" size="small">
              ADO Org: {organization.ado_organization}
            </Label>
          )}
          {organization.enterprise && (
            <Label variant="sponsors" size="small">
              Enterprise: {organization.enterprise}
            </Label>
          )}
          {organization.total_projects && (
            <Label variant="default" size="small">
              {organization.total_projects} Projects
            </Label>
          )}
        </div>
      </div>

      {/* Progress Bar */}
      <div className="mb-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
            Migration Progress
          </span>
          <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-accent)' }}>
            {progressPercentage}%
          </span>
        </div>
        <ProgressBar 
          progress={progressPercentage} 
          aria-label={`${organization.organization} ${progressPercentage}% complete`}
          bg={getProgressColor(progressPercentage)}
          barSize="small"
        />
        <div className="flex items-center justify-between mt-1">
          <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            {organization.migrated_count} of {organization.total_repos} migrated
          </span>
          {progressPercentage >= 80 && (
            <span style={{ color: 'var(--fgColor-success)' }}>
              <CheckCircleIcon size={12} />
            </span>
          )}
          {hasFailures && (
            <div className="flex items-center gap-1">
              <span style={{ color: 'var(--fgColor-attention)' }}>
                <AlertIcon size={12} />
              </span>
              <span className="text-xs" style={{ color: 'var(--fgColor-attention)' }}>
                {organization.failed_count} failed
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Status Metrics */}
      <div className="grid grid-cols-2 gap-3 mb-3 py-3 border-t" style={{ borderColor: 'var(--borderColor-default)' }}>
        <div>
          <div className="text-2xl font-semibold" style={{ color: 'var(--fgColor-success)' }}>
            {organization.migrated_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Complete</div>
        </div>
        <div>
          <div className="text-2xl font-semibold" style={{ color: 'var(--fgColor-accent)' }}>
            {organization.in_progress_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>In Progress</div>
        </div>
        <div>
          <div className="text-2xl font-semibold" style={{ color: 'var(--fgColor-danger)' }}>
            {organization.failed_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Failed</div>
        </div>
        <div>
          <div className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            {organization.pending_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Pending</div>
        </div>
      </div>

      {/* Footer */}
      <div className="text-sm hover:underline font-medium" style={{ color: 'var(--fgColor-accent)' }}>
        View details →
      </div>
    </Link>
  );
}
