import { RepoIcon, CheckCircleIcon, XCircleIcon, SyncIcon, AlertIcon, RocketIcon } from '@primer/octicons-react';
import { KPICard } from '../Analytics/KPICard';
import { Analytics } from '../../types';
import { getRepositoriesUrl } from '../../utils/filters';
import { useNavigate } from 'react-router-dom';

interface KPISectionProps {
  analytics: Analytics | undefined;
  isLoading: boolean;
}

export function KPISection({ analytics, isLoading }: KPISectionProps) {
  const navigate = useNavigate();

  if (isLoading || !analytics) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {[...Array(6)].map((_, i) => (
          <div
            key={i}
            className="rounded-lg border p-6 animate-pulse"
            style={{
              backgroundColor: 'var(--bgColor-default)',
              borderColor: 'var(--borderColor-default)',
            }}
          >
            <div className="h-4 bg-gray-300 rounded w-1/2 mb-4"></div>
            <div className="h-8 bg-gray-300 rounded w-3/4"></div>
          </div>
        ))}
      </div>
    );
  }

  const completionRate = analytics.total_repositories > 0
    ? Math.round((analytics.migrated_count / analytics.total_repositories) * 100)
    : 0;

  const failedCount = analytics.failed_count || 0;
  const inProgressCount = analytics.in_progress_count || 0;
  const successRate = analytics.success_rate || 0;
  const velocityPerWeek = analytics.migration_velocity?.repos_per_week || 0;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4 mb-8">
      <KPICard
        title="Total Repositories"
        value={analytics.total_repositories}
        subtitle={`${analytics.migrated_count} migrated`}
        color="blue"
        icon={<span style={{ color: 'var(--fgColor-accent)' }}><RepoIcon size={20} /></span>}
        tooltip="Total number of repositories discovered across all organizations"
        onClick={() => navigate('/repositories')}
      />

      <KPICard
        title="Migration Progress"
        value={`${completionRate}%`}
        subtitle={`${analytics.migrated_count} of ${analytics.total_repositories}`}
        color="green"
        icon={<span style={{ color: 'var(--fgColor-success)' }}><CheckCircleIcon size={20} /></span>}
        tooltip="Percentage of repositories successfully migrated"
        onClick={() => navigate(getRepositoriesUrl({ status: ['complete', 'migration_complete'] }))}
      />

      <KPICard
        title="Success Rate"
        value={`${successRate.toFixed(1)}%`}
        subtitle="of attempted migrations"
        color="purple"
        icon={<span style={{ color: 'var(--fgColor-done)' }}><RocketIcon size={20} /></span>}
        tooltip="Percentage of successful migrations vs. failed migrations"
      />

      <KPICard
        title="Active Migrations"
        value={inProgressCount}
        subtitle="currently running"
        color="blue"
        icon={<span style={{ color: 'var(--fgColor-accent)' }}><SyncIcon size={20} /></span>}
        tooltip="Number of repositories currently being migrated"
        onClick={() => navigate(getRepositoriesUrl({ 
          status: ['queued_for_migration', 'migrating_content', 'pre_migration', 'archive_generating', 'post_migration'] 
        }))}
      />

      <KPICard
        title="Failed Items"
        value={failedCount}
        subtitle="need attention"
        color="yellow"
        icon={<span style={{ color: 'var(--fgColor-attention)' }}><AlertIcon size={20} /></span>}
        tooltip="Repositories with failed migrations or dry runs requiring attention"
        onClick={() => navigate(getRepositoriesUrl({ 
          status: ['migration_failed', 'dry_run_failed', 'rolled_back'] 
        }))}
      />

      <KPICard
        title="Migration Velocity"
        value={velocityPerWeek.toFixed(1)}
        subtitle="repos/week"
        color="green"
        icon={<span style={{ color: 'var(--fgColor-success)' }}><XCircleIcon size={20} /></span>}
        tooltip="Average number of repositories migrated per week over the last 30 days"
      />
    </div>
  );
}
