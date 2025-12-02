import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ProgressBar } from '@primer/react';
import { ChevronDownIcon, ChevronRightIcon, AlertIcon } from '@primer/octicons-react';
import { Organization } from '../../types';

interface ADOOrganizationCardProps {
  adoOrgName: string;
  projects: Organization[];
}

export function ADOOrganizationCard({ adoOrgName, projects }: ADOOrganizationCardProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  // Calculate aggregate stats for the ADO org
  const totalRepos = projects.reduce((sum, proj) => sum + proj.total_repos, 0);
  const totalMigrated = projects.reduce((sum, proj) => sum + proj.migrated_count, 0);
  const totalInProgress = projects.reduce((sum, proj) => sum + proj.in_progress_count, 0);
  const totalFailed = projects.reduce((sum, proj) => sum + proj.failed_count, 0);
  const totalPending = projects.reduce((sum, proj) => sum + proj.pending_count, 0);
  const overallProgress = totalRepos > 0 ? Math.floor((totalMigrated * 100) / totalRepos) : 0;

  const getProgressColor = (percentage: number): 'success.emphasis' | 'attention.emphasis' | 'default' => {
    if (percentage >= 80) return 'success.emphasis';
    if (percentage >= 40) return 'attention.emphasis';
    return 'default';
  };

  return (
    <div
      className="rounded-lg border p-6"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)',
      }}
    >
      {/* ADO Organization Header */}
      <div className="mb-4">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-2 w-full text-left hover:opacity-80 transition-opacity"
        >
          {isExpanded ? (
            <span style={{ color: 'var(--fgColor-muted)' }}>
              <ChevronDownIcon size={20} />
            </span>
          ) : (
            <span style={{ color: 'var(--fgColor-muted)' }}>
              <ChevronRightIcon size={20} />
            </span>
          )}
          <h3 className="text-xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            {adoOrgName}
          </h3>
        </button>
      </div>

      {/* Aggregate Progress Bar */}
      <div className="mb-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
            Overall Progress
          </span>
          <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-accent)' }}>
            {overallProgress}%
          </span>
        </div>
        <ProgressBar
          progress={overallProgress}
          aria-label={`${adoOrgName} ${overallProgress}% complete`}
          bg={getProgressColor(overallProgress)}
          barSize="default"
        />
        <div className="flex items-center justify-between mt-1">
          <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            {totalMigrated} of {totalRepos} repos across {projects.length} {projects.length === 1 ? 'project' : 'projects'}
          </span>
          {totalFailed > 0 && (
            <div className="flex items-center gap-1">
              <span style={{ color: 'var(--fgColor-attention)' }}>
                <AlertIcon size={12} />
              </span>
              <span className="text-xs" style={{ color: 'var(--fgColor-attention)' }}>
                {totalFailed} failed
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Aggregate Status Metrics */}
      <div className="grid grid-cols-4 gap-3 mb-4 pb-4 border-b" style={{ borderColor: 'var(--borderColor-default)' }}>
        <div className="text-center">
          <div className="text-xl font-semibold" style={{ color: 'var(--fgColor-success)' }}>
            {totalMigrated}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Complete</div>
        </div>
        <div className="text-center">
          <div className="text-xl font-semibold" style={{ color: 'var(--fgColor-accent)' }}>
            {totalInProgress}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>In Progress</div>
        </div>
        <div className="text-center">
          <div className="text-xl font-semibold" style={{ color: 'var(--fgColor-danger)' }}>
            {totalFailed}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Failed</div>
        </div>
        <div className="text-center">
          <div className="text-xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            {totalPending}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Pending</div>
        </div>
      </div>

      {/* Projects List (Collapsible) */}
      {isExpanded && (
        <div className="space-y-3">
          <div className="text-sm font-semibold mb-2" style={{ color: 'var(--fgColor-muted)' }}>
            PROJECTS ({projects.length})
          </div>
          {projects.map((project) => (
            <ProjectItem key={project.organization} project={project} />
          ))}
        </div>
      )}
    </div>
  );
}

interface ProjectItemProps {
  project: Organization;
}

function ProjectItem({ project }: ProjectItemProps) {
  const progressPercentage = project.migration_progress_percentage;
  const hasFailures = project.failed_count > 0;

  const getProgressColor = (percentage: number): 'success.emphasis' | 'attention.emphasis' | 'default' => {
    if (percentage >= 80) return 'success.emphasis';
    if (percentage >= 40) return 'attention.emphasis';
    return 'default';
  };

  return (
    <Link
      to={`/repositories?organization=${encodeURIComponent(project.organization)}`}
      className="block rounded-md border p-4 hover:shadow-md transition-all"
      style={{
        backgroundColor: 'var(--bgColor-inset)',
        borderColor: 'var(--borderColor-default)',
      }}
    >
      {/* Project Name and Total Repos */}
      <div className="flex items-start justify-between mb-3">
        <div>
          <h4 className="font-medium mb-1" style={{ color: 'var(--fgColor-default)' }}>
            {project.organization}
          </h4>
          <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            {project.total_repos} {project.total_repos === 1 ? 'repository' : 'repositories'}
          </span>
        </div>
        <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-accent)' }}>
          {progressPercentage}%
        </span>
      </div>

      {/* Progress Bar */}
      <div className="mb-3">
        <ProgressBar
          progress={progressPercentage}
          aria-label={`${project.organization} ${progressPercentage}% complete`}
          bg={getProgressColor(progressPercentage)}
          barSize="small"
        />
        <div className="flex items-center justify-between mt-1">
          <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            {project.migrated_count} migrated
          </span>
          {hasFailures && (
            <div className="flex items-center gap-1">
              <span style={{ color: 'var(--fgColor-attention)' }}>
                <AlertIcon size={12} />
              </span>
              <span className="text-xs" style={{ color: 'var(--fgColor-attention)' }}>
                {project.failed_count} failed
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Status Counts */}
      <div className="grid grid-cols-4 gap-2 text-center">
        <div>
          <div className="text-sm font-semibold" style={{ color: 'var(--fgColor-success)' }}>
            {project.migrated_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Done</div>
        </div>
        <div>
          <div className="text-sm font-semibold" style={{ color: 'var(--fgColor-accent)' }}>
            {project.in_progress_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Active</div>
        </div>
        <div>
          <div className="text-sm font-semibold" style={{ color: 'var(--fgColor-danger)' }}>
            {project.failed_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Failed</div>
        </div>
        <div>
          <div className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            {project.pending_count}
          </div>
          <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Pending</div>
        </div>
      </div>

      {/* View Link */}
      <div className="mt-2 text-xs hover:underline font-medium text-right" style={{ color: 'var(--fgColor-accent)' }}>
        View project →
      </div>
    </Link>
  );
}
