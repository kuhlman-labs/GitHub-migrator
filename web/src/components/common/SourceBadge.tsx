import { Tooltip } from '@primer/react';
import { MarkGithubIcon } from '@primer/octicons-react';
import { useState } from 'react';
import { useSourceContext } from '../../contexts/SourceContext';
import type { SourceType } from '../../types';

/**
 * Azure DevOps icon component using the official SVG.
 * Uses a unique gradient ID to avoid conflicts with other instances.
 */
function AzureDevOpsIcon({ size = 16, id = 'badge' }: { size?: number; id?: string }) {
  return (
    <svg 
      width={size} 
      height={size} 
      viewBox="0 0 18 18" 
      xmlns="http://www.w3.org/2000/svg"
      style={{ display: 'inline-block', verticalAlign: 'text-bottom', flexShrink: 0 }}
    >
      <defs>
        <linearGradient id={`ado-gradient-${id}`} x1="9" y1="16.97" x2="9" y2="1.03" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#0078d4"/>
          <stop offset="0.16" stopColor="#1380da"/>
          <stop offset="0.53" stopColor="#3c91e5"/>
          <stop offset="0.82" stopColor="#559cec"/>
          <stop offset="1" stopColor="#5ea0ef"/>
        </linearGradient>
      </defs>
      <path 
        d="M17,4v9.74l-4,3.28-6.2-2.26V17L3.29,12.41l10.23.8V4.44Zm-3.41.49L7.85,1V3.29L2.58,4.84,1,6.87v4.61l2.26,1V6.57Z" 
        fill={`url(#ado-gradient-${id})`}
      />
    </svg>
  );
}

/**
 * Get the appropriate icon for a source type with size support.
 */
function SourceIcon({ type, size = 16, id }: { type: SourceType; size?: number; id?: string }) {
  return type === 'github' 
    ? <MarkGithubIcon size={size} /> 
    : <AzureDevOpsIcon size={size} id={id} />;
}

interface SourceBadgeProps {
  /** Source ID to look up from context */
  sourceId?: number;
  /** Directly provide source name (used when sourceId lookup fails or for external sources) */
  sourceName?: string;
  /** Directly provide source type */
  sourceType?: SourceType;
  /** Size variant */
  size?: 'small' | 'medium' | 'large';
  /** Show only icon (useful for compact views) */
  iconOnly?: boolean;
  /** Additional CSS class */
  className?: string;
  /** Show tooltip with source details */
  showTooltip?: boolean;
}

const SIZE_CONFIG = {
  small: {
    iconSize: 12,
    fontSize: 'text-xs',
    padding: 'px-1.5 py-0.5',
    gap: 'gap-1',
  },
  medium: {
    iconSize: 14,
    fontSize: 'text-sm',
    padding: 'px-2 py-1',
    gap: 'gap-1.5',
  },
  large: {
    iconSize: 16,
    fontSize: 'text-base',
    padding: 'px-2.5 py-1.5',
    gap: 'gap-2',
  },
};

/**
 * SourceBadge displays a visual indicator for the source of a repository,
 * team, user, or other entity.
 * 
 * It can look up source details from context using sourceId, or display
 * directly provided source information.
 * 
 * @example
 * // Using source ID (looks up from context)
 * <SourceBadge sourceId={repository.source_id} size="small" />
 * 
 * @example
 * // Direct source info
 * <SourceBadge sourceName="Production GitHub" sourceType="github" />
 * 
 * @example
 * // Icon only with tooltip
 * <SourceBadge sourceId={1} iconOnly showTooltip />
 */
export function SourceBadge({
  sourceId,
  sourceName,
  sourceType,
  size = 'small',
  iconOnly = false,
  className = '',
  showTooltip = true,
}: SourceBadgeProps) {
  const { sources } = useSourceContext();
  
  // Look up source from context if sourceId is provided
  const source = sourceId ? sources.find(s => s.id === sourceId) : null;
  
  // Determine display values
  const displayName = source?.name || sourceName || 'Unknown Source';
  const displayType: SourceType = source?.type || sourceType || 'github';
  const displayUrl = source?.base_url;
  
  // Get size configuration
  const config = SIZE_CONFIG[size];
  
  // Generate unique ID for SVG gradient (generated once on mount)
  const [uniqueId] = useState(
    () => `badge-${sourceId || 'static'}-${Math.random().toString(36).substr(2, 9)}`
  );
  
  // Badge background color based on source type
  const bgColor = displayType === 'github' 
    ? 'var(--bgColor-neutral-muted)' 
    : 'rgba(0, 120, 212, 0.1)';
  
  const tooltipText = iconOnly 
    ? displayName 
    : displayUrl 
      ? `${displayName} (${displayUrl})`
      : displayName;
  
  const badgeContent = (
    <>
      <SourceIcon type={displayType} size={config.iconSize} id={uniqueId} />
      {!iconOnly && (
        <span className={`${config.fontSize} font-medium truncate max-w-[150px]`}>
          {displayName}
        </span>
      )}
    </>
  );
  
  // Wrap in tooltip if enabled - Tooltip requires interactive element
  if (showTooltip) {
    return (
      <Tooltip text={tooltipText} direction="s">
        <button
          type="button"
          className={`inline-flex items-center rounded-full ${config.padding} ${config.gap} ${className}`}
          style={{ 
            backgroundColor: bgColor,
            color: 'var(--fgColor-default)',
            flexShrink: 0,
            border: 'none',
            cursor: 'default',
          }}
          tabIndex={-1}
          aria-label={tooltipText}
        >
          {badgeContent}
        </button>
      </Tooltip>
    );
  }
  
  return (
    <span
      className={`inline-flex items-center rounded-full ${config.padding} ${config.gap} ${className}`}
      style={{ 
        backgroundColor: bgColor,
        color: 'var(--fgColor-default)',
        flexShrink: 0,
      }}
    >
      {badgeContent}
    </span>
  );
}

/**
 * Simplified source icon component for inline use.
 * Does not include badge styling, just the icon with optional tooltip.
 */
export function SourceTypeIcon({
  sourceId,
  sourceType,
  size = 14,
  showTooltip = true,
}: {
  sourceId?: number;
  sourceType?: SourceType;
  size?: number;
  showTooltip?: boolean;
}) {
  const { sources } = useSourceContext();
  const source = sourceId ? sources.find(s => s.id === sourceId) : null;
  const type: SourceType = source?.type || sourceType || 'github';
  const name = source?.name || (type === 'github' ? 'GitHub' : 'Azure DevOps');
  
  const icon = <SourceIcon type={type} size={size} id={`icon-${sourceId || 'static'}`} />;
  
  if (showTooltip) {
    return (
      <Tooltip text={name} direction="s">
        <button
          type="button"
          className="inline-flex items-center"
          style={{ 
            background: 'none', 
            border: 'none', 
            padding: 0, 
            cursor: 'default',
            color: 'inherit',
          }}
          tabIndex={-1}
          aria-label={name}
        >
          {icon}
        </button>
      </Tooltip>
    );
  }
  
  return icon;
}

export { AzureDevOpsIcon, SourceIcon };

