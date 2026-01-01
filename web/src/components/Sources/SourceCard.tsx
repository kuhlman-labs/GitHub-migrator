import { Label, IconButton, RelativeTime } from '@primer/react';
import { 
  MarkGithubIcon, 
  CheckCircleIcon, 
  PencilIcon,
  TrashIcon,
  RepoIcon,
  SyncIcon,
} from '@primer/octicons-react';
import type { Source } from '../../types';

/**
 * Azure DevOps icon component using the official SVG
 */
function AzureDevOpsIcon({ size = 24 }: { size?: number }) {
  return (
    <svg 
      width={size} 
      height={size} 
      viewBox="0 0 18 18" 
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <linearGradient id="ado-gradient" x1="9" y1="16.97" x2="9" y2="1.03" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#0078d4"/>
          <stop offset="0.16" stopColor="#1380da"/>
          <stop offset="0.53" stopColor="#3c91e5"/>
          <stop offset="0.82" stopColor="#559cec"/>
          <stop offset="1" stopColor="#5ea0ef"/>
        </linearGradient>
      </defs>
      <path 
        d="M17,4v9.74l-4,3.28-6.2-2.26V17L3.29,12.41l10.23.8V4.44Zm-3.41.49L7.85,1V3.29L2.58,4.84,1,6.87v4.61l2.26,1V6.57Z" 
        fill="url(#ado-gradient)"
      />
    </svg>
  );
}

interface SourceCardProps {
  source: Source;
  onEdit: (source: Source) => void;
  onDelete: (source: Source) => void;
  onValidate: (source: Source) => void;
}

/**
 * Card component displaying a single migration source.
 */
export function SourceCard({ source, onEdit, onDelete, onValidate }: SourceCardProps) {
  const typeLabel = source.type === 'github' ? 'GitHub' : 'Azure DevOps';
  const typeVariant = source.type === 'github' ? 'accent' : 'done';

  return (
    <div
      className="rounded-lg border p-5 transition-all hover:shadow-md"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderColor: source.is_active ? 'var(--borderColor-default)' : 'var(--borderColor-muted)',
        opacity: source.is_active ? 1 : 0.7,
      }}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className="p-2 rounded-lg"
            style={{ backgroundColor: 'var(--bgColor-muted)' }}
          >
            {source.type === 'github' ? (
              <MarkGithubIcon size={24} />
            ) : (
              <AzureDevOpsIcon size={24} />
            )}
          </div>
          <div>
            <h3 
              className="text-lg font-semibold"
              style={{ color: 'var(--fgColor-default)' }}
            >
              {source.name}
            </h3>
            <div className="flex items-center gap-2 mt-1">
              <Label variant={typeVariant} size="small">
                {typeLabel}
              </Label>
              {!source.is_active && (
                <Label variant="secondary" size="small">
                  Inactive
                </Label>
              )}
              {source.has_app_auth && (
                <Label variant="success" size="small">
                  App Auth
                </Label>
              )}
            </div>
          </div>
        </div>
        
        {/* Actions */}
        <div className="flex items-center gap-1">
          <IconButton
            aria-label="Validate connection"
            icon={SyncIcon}
            variant="invisible"
            size="small"
            onClick={() => onValidate(source)}
          />
          <IconButton
            aria-label="Edit source"
            icon={PencilIcon}
            variant="invisible"
            size="small"
            onClick={() => onEdit(source)}
          />
          <IconButton
            aria-label="Delete source"
            icon={TrashIcon}
            variant="invisible"
            size="small"
            onClick={() => onDelete(source)}
          />
        </div>
      </div>

      {/* URL */}
      <div className="mb-4">
        <div className="text-xs mb-1" style={{ color: 'var(--fgColor-muted)' }}>
          Base URL
        </div>
        <div 
          className="text-sm font-mono truncate"
          style={{ color: 'var(--fgColor-default)' }}
          title={source.base_url}
        >
          {source.base_url}
        </div>
        {source.organization && (
          <div className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Organization: <span style={{ color: 'var(--fgColor-default)' }}>{source.organization}</span>
          </div>
        )}
      </div>

      {/* Stats */}
      <div 
        className="flex items-center justify-between pt-4 border-t"
        style={{ borderColor: 'var(--borderColor-muted)' }}
      >
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <RepoIcon size={16} className="text-muted" />
            <span className="text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
              {source.repository_count}
            </span>
            <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
              repos
            </span>
          </div>
        </div>
        
        <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
          {source.last_sync_at ? (
            <>
              Last synced <RelativeTime datetime={source.last_sync_at} />
            </>
          ) : (
            'Never synced'
          )}
        </div>
      </div>

      {/* Token indicator */}
      <div className="flex items-center gap-2 mt-3 text-xs" style={{ color: 'var(--fgColor-muted)' }}>
        <CheckCircleIcon size={12} className="text-success" />
        Token: {source.masked_token}
      </div>
    </div>
  );
}

