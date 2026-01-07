import {
  Button,
  Flash,
  Label,
  Spinner,
} from '@primer/react';
import {
  XIcon,
  OrganizationIcon,
  PersonIcon,
  CheckIcon,
  ClockIcon,
  AlertIcon,
  MailIcon,
  LinkIcon,
} from '@primer/octicons-react';
import { useUserDetail } from '../../hooks/useQueries';
import { FallbackAvatar } from '../common/FallbackAvatar';
import { SourceBadge } from '../common/SourceBadge';
import { ReclaimStatus, UserMappingStatus } from '../../types';

interface UserDetailPanelProps {
  login: string;
  onClose: () => void;
  onEditMapping?: (login: string) => void;
}

const statusColors: Record<UserMappingStatus, 'default' | 'accent' | 'success' | 'attention' | 'danger' | 'done'> = {
  unmapped: 'attention',
  mapped: 'accent',
  reclaimed: 'success',
  skipped: 'default',
};

const reclaimStatusColors: Record<ReclaimStatus | string, 'default' | 'accent' | 'success' | 'attention' | 'danger'> = {
  pending: 'attention',
  invited: 'accent',
  completed: 'success',
  failed: 'danger',
};

const reclaimStatusIcons: Record<ReclaimStatus | string, React.ReactNode> = {
  pending: <ClockIcon size={12} />,
  invited: <MailIcon size={12} />,
  completed: <CheckIcon size={12} />,
  failed: <AlertIcon size={12} />,
};

export function UserDetailPanel({ login, onClose, onEditMapping }: UserDetailPanelProps) {
  const { data: user, isLoading, error } = useUserDetail(login);

  if (error) {
    return (
      <div
        className="fixed top-0 right-0 w-[500px] h-screen flex flex-col overflow-hidden"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderLeft: '1px solid var(--borderColor-default)',
          boxShadow: '0 0 20px rgba(0,0,0,0.2)',
          zIndex: 100,
        }}
      >
        <div
          className="p-4 flex justify-between items-center"
          style={{ borderBottom: '1px solid var(--borderColor-default)' }}
        >
          <h2 className="text-base font-semibold m-0">User Details</h2>
          <Button variant="invisible" onClick={onClose} aria-label="Close">
            <XIcon size={16} />
          </Button>
        </div>
        <div className="p-6">
          <Flash variant="danger">Failed to load user details: {error.message}</Flash>
        </div>
      </div>
    );
  }

  return (
    <div
      className="fixed top-0 right-0 w-[500px] h-screen flex flex-col overflow-hidden"
      style={{
        backgroundColor: 'var(--bgColor-default)',
        borderLeft: '1px solid var(--borderColor-default)',
        boxShadow: '0 0 20px rgba(0,0,0,0.2)',
        zIndex: 100,
      }}
    >
      {/* Header */}
      <div
        className="p-4 flex justify-between items-center"
        style={{ borderBottom: '1px solid var(--borderColor-default)' }}
      >
        <div className="flex items-center gap-3">
          {isLoading ? (
            <div className="w-10 h-10 rounded-full" style={{ backgroundColor: 'var(--bgColor-muted)' }} />
          ) : (
            <FallbackAvatar src={user?.avatar_url} login={user?.login || login} size={40} />
          )}
          <div>
            <h2 className="text-base font-semibold m-0 mb-1">
              {isLoading ? 'Loading...' : user?.name || user?.login || login}
            </h2>
            <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
              @{login}
            </span>
          </div>
        </div>
        <Button variant="invisible" onClick={onClose} aria-label="Close">
          <XIcon size={16} />
        </Button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
        {isLoading ? (
          <div className="flex justify-center py-8">
            <Spinner size="medium" />
          </div>
        ) : user ? (
          <>
            {/* Basic Info */}
            <section>
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <PersonIcon size={16} />
                User Information
              </h3>
              <div
                className="rounded-md p-3 flex flex-col gap-2"
                style={{
                  backgroundColor: 'var(--bgColor-muted)',
                  border: '1px solid var(--borderColor-default)',
                }}
              >
                {user.source_id && (
                  <div className="flex justify-between items-center text-sm">
                    <span style={{ color: 'var(--fgColor-muted)' }}>Source</span>
                    <SourceBadge sourceId={user.source_id} size="small" />
                  </div>
                )}
                {user.email && (
                  <div className="flex justify-between text-sm">
                    <span style={{ color: 'var(--fgColor-muted)' }}>Email</span>
                    <span className="font-mono">{user.email}</span>
                  </div>
                )}
                <div className="flex justify-between text-sm">
                  <span style={{ color: 'var(--fgColor-muted)' }}>Source Instance</span>
                  <span className="font-mono">{user.source_instance}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span style={{ color: 'var(--fgColor-muted)' }}>Discovered</span>
                  <span>{new Date(user.discovered_at).toLocaleDateString()}</span>
                </div>
              </div>
            </section>

            {/* Organizations */}
            <section>
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <OrganizationIcon size={16} />
                Source Organizations ({user.organizations.length})
              </h3>
              {user.organizations.length === 0 ? (
                <div
                  className="text-sm text-center py-4 rounded-md"
                  style={{
                    backgroundColor: 'var(--bgColor-muted)',
                    color: 'var(--fgColor-muted)',
                  }}
                >
                  No organization memberships found
                </div>
              ) : (
                <div className="flex flex-col gap-1">
                  {user.organizations.map((org) => (
                    <div
                      key={org.organization}
                      className="flex items-center justify-between p-2 rounded-md"
                      style={{
                        backgroundColor: 'var(--bgColor-muted)',
                        border: '1px solid var(--borderColor-default)',
                      }}
                    >
                      <div className="flex items-center gap-2">
                        <OrganizationIcon size={14} />
                        <span className="text-sm font-medium">{org.organization}</span>
                      </div>
                      <Label variant={org.role === 'admin' ? 'accent' : 'default'} size="small">
                        {org.role}
                      </Label>
                    </div>
                  ))}
                </div>
              )}
            </section>

            {/* Mapping Status */}
            <section>
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <LinkIcon size={16} />
                Mapping Status
              </h3>
              <div
                className="rounded-md p-3 flex flex-col gap-3"
                style={{
                  backgroundColor: 'var(--bgColor-muted)',
                  border: '1px solid var(--borderColor-default)',
                }}
              >
                {user.mapping ? (
                  <>
                    <div className="flex justify-between items-center">
                      <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Status</span>
                      <Label variant={statusColors[user.mapping.mapping_status as UserMappingStatus] || 'default'}>
                        {user.mapping.mapping_status}
                      </Label>
                    </div>
                    
                    {user.mapping.destination_login && (
                      <div className="flex justify-between text-sm">
                        <span style={{ color: 'var(--fgColor-muted)' }}>Destination</span>
                        <span className="font-mono">@{user.mapping.destination_login}</span>
                      </div>
                    )}
                    
                    {user.mapping.mannequin_login && (
                      <div className="flex justify-between text-sm">
                        <span style={{ color: 'var(--fgColor-muted)' }}>Mannequin</span>
                        <span className="font-mono">{user.mapping.mannequin_login}</span>
                      </div>
                    )}
                    
                    {user.mapping.reclaim_status && (
                      <div className="flex justify-between items-center">
                        <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Reclaim Status</span>
                        <Label variant={reclaimStatusColors[user.mapping.reclaim_status] || 'default'}>
                          <span className="flex items-center gap-1">
                            {reclaimStatusIcons[user.mapping.reclaim_status]}
                            {user.mapping.reclaim_status}
                          </span>
                        </Label>
                      </div>
                    )}
                    
                    {user.mapping.match_confidence != null && (
                      <div className="flex justify-between items-center">
                        <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>Match Confidence</span>
                        <div className="flex items-center gap-2">
                          <div
                            className="w-20 h-2 rounded-full overflow-hidden"
                            style={{ backgroundColor: 'var(--bgColor-neutral-muted)' }}
                          >
                            <div
                              className="h-full rounded-full"
                              style={{
                                width: `${user.mapping.match_confidence}%`,
                                backgroundColor: user.mapping.match_confidence >= 80 ? 'var(--bgColor-success-emphasis)' : 
                                                 user.mapping.match_confidence >= 60 ? 'var(--bgColor-attention-emphasis)' :
                                                 'var(--bgColor-danger-emphasis)',
                              }}
                            />
                          </div>
                          <span className="text-xs font-mono">{user.mapping.match_confidence}%</span>
                        </div>
                      </div>
                    )}
                    
                    {user.mapping.match_reason && (
                      <div className="flex justify-between text-sm">
                        <span style={{ color: 'var(--fgColor-muted)' }}>Match Reason</span>
                        <span className="font-mono text-xs">{user.mapping.match_reason}</span>
                      </div>
                    )}

                    {user.mapping.reclaim_error && (
                      <Flash variant="danger" className="mt-2">
                        {user.mapping.reclaim_error}
                      </Flash>
                    )}
                  </>
                ) : (
                  <div className="flex flex-col items-center gap-2 py-2">
                    <Label variant="attention">unmapped</Label>
                    <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                      This user has not been mapped to a destination account
                    </span>
                  </div>
                )}
                
                {onEditMapping && (
                  <Button
                    variant="default"
                    size="small"
                    onClick={() => onEditMapping(login)}
                    className="mt-2"
                  >
                    Edit Mapping
                  </Button>
                )}
              </div>
            </section>
          </>
        ) : null}
      </div>
    </div>
  );
}
