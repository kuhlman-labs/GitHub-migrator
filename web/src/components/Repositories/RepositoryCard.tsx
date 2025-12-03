import { Link } from 'react-router-dom';
import { Checkbox } from '@primer/react';
import type { Repository } from '../../types';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { TimestampDisplay } from '../common/TimestampDisplay';
import { formatBytes } from '../../utils/format';

interface RepositoryCardProps {
  repository: Repository;
  selectionMode?: boolean;
  selected?: boolean;
  onToggleSelect?: (id: number) => void;
}

export function RepositoryCard({ 
  repository, 
  selectionMode = false, 
  selected = false,
  onToggleSelect 
}: RepositoryCardProps) {
  // For ADO repos, parse the full_name (org/project/repo) to show project/repo as title
  const getDisplayInfo = () => {
    if (repository.ado_project) {
      // ADO full_name format: organization/project/reponame
      const parts = repository.full_name.split('/');
      if (parts.length >= 3) {
        const adoOrg = parts[0];
        // Join remaining parts as project/repo (handles repos with slashes in name)
        const projectAndRepo = parts.slice(1).join('/');
        return {
          title: projectAndRepo,
          subtitle: adoOrg,
          isAdo: true
        };
      }
    }
    return {
      title: repository.full_name,
      subtitle: null,
      isAdo: false
    };
  };

  const displayInfo = getDisplayInfo();

  const handleCheckboxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.preventDefault();
    e.stopPropagation();
    if (onToggleSelect) {
      onToggleSelect(repository.id);
    }
  };

  const handleCardClick = (e: React.MouseEvent) => {
    if (selectionMode && onToggleSelect) {
      e.preventDefault();
      onToggleSelect(repository.id);
    }
  };

  const cardContent = (
    <>
      {selectionMode && (
        <div 
          className="absolute top-4 right-4 z-10"
          onClick={(e) => e.stopPropagation()}
        >
          <Checkbox
            checked={selected}
            onChange={handleCheckboxChange}
            aria-label={`Select ${repository.full_name}`}
          />
        </div>
      )}
      
      {/* For ADO: show org as subtitle above the title */}
      {displayInfo.subtitle && (
        <div className="text-xs mb-1 truncate" style={{ color: 'var(--fgColor-muted)' }}>
          {displayInfo.subtitle}
        </div>
      )}
      <h3 className="text-base font-semibold mb-3 truncate pr-8" style={{ color: 'var(--fgColor-default)' }}>
        {displayInfo.title}
      </h3>
      <div className="mb-3 flex items-center justify-between">
        <StatusBadge status={repository.status} size="small" />
      </div>
      <div className="space-y-1.5 text-sm mb-3" style={{ color: 'var(--fgColor-muted)' }}>
        <div>Size: {formatBytes(repository.total_size)}</div>
        <div>Branches: {repository.branch_count}</div>
      </div>
      
      {/* Timestamps */}
      <div 
        className="space-y-1 mb-3 pt-3"
        style={{ borderTop: '1px solid var(--borderColor-default)' }}
      >
        {repository.last_discovery_at && (
          <TimestampDisplay 
            timestamp={repository.last_discovery_at} 
            label="Discovered"
            size="sm"
          />
        )}
        {repository.last_dry_run_at && (
          <TimestampDisplay 
            timestamp={repository.last_dry_run_at} 
            label="Dry run"
            size="sm"
          />
        )}
      </div>

      <div className="flex gap-1.5 flex-wrap">
        {repository.is_archived && <Badge color="gray">Archived</Badge>}
        {repository.is_fork && <Badge color="purple">Fork</Badge>}
        {repository.has_lfs && <Badge color="blue">LFS</Badge>}
        {repository.has_submodules && <Badge color="purple">Submodules</Badge>}
        {repository.has_large_files && <Badge color="orange">Large Files</Badge>}
        {repository.has_actions && <Badge color="green">Actions</Badge>}
        {repository.has_packages && <Badge color="orange">Packages</Badge>}
        {repository.has_wiki && <Badge color="yellow">Wiki</Badge>}
        {repository.has_pages && <Badge color="pink">Pages</Badge>}
        {repository.has_discussions && <Badge color="indigo">Discussions</Badge>}
        {repository.has_projects && <Badge color="teal">Projects</Badge>}
        {repository.branch_protections > 0 && <Badge color="red">Protected</Badge>}
        {repository.has_rulesets && <Badge color="red">Rulesets</Badge>}
        {repository.has_code_scanning && <Badge color="green">Code Scanning</Badge>}
        {repository.has_dependabot && <Badge color="green">Dependabot</Badge>}
        {repository.has_secret_scanning && <Badge color="green">Secret Scanning</Badge>}
        {repository.has_codeowners && <Badge color="blue">CODEOWNERS</Badge>}
        {repository.has_self_hosted_runners && <Badge color="purple">Self-Hosted</Badge>}
        {repository.visibility === 'public' && <Badge color="blue">Public</Badge>}
        {repository.visibility === 'internal' && <Badge color="yellow">Internal</Badge>}
        {repository.has_release_assets && <Badge color="pink">Releases</Badge>}
      </div>
    </>
  );

  if (selectionMode) {
    return (
      <div
        onClick={handleCardClick}
        className="relative rounded-lg transition-all cursor-pointer p-6"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderWidth: selected ? '2px' : '1px',
          borderStyle: 'solid',
          borderColor: selected ? 'var(--borderColor-accent-emphasis)' : 'var(--borderColor-default)',
          boxShadow: 'var(--shadow-resting-small)',
        }}
      >
        {cardContent}
      </div>
    );
  }

  return (
    <Link
      to={`/repository/${encodeURIComponent(repository.full_name)}`}
      className="relative rounded-lg border transition-opacity hover:opacity-80 p-6 block"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: 'var(--borderColor-default)',
        boxShadow: 'var(--shadow-resting-small)'
      }}
    >
      {cardContent}
    </Link>
  );
}
