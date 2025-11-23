import { useAuth } from '../../contexts/AuthContext';
import { Avatar, ActionMenu, ActionList } from '@primer/react';
import { MarkGithubIcon, SignOutIcon } from '@primer/octicons-react';

export function UserProfile() {
  const { user, logout, authEnabled, isAuthenticated } = useAuth();

  // Don't render if auth is disabled or user is not authenticated
  if (!authEnabled || !isAuthenticated || !user) {
    return null;
  }

  const handleLogout = async () => {
    await logout();
  };

  return (
    <ActionMenu>
      <ActionMenu.Anchor>
        <div className="flex items-center gap-2 cursor-pointer hover:opacity-80">
          <Avatar src={user.avatar_url} size={32} alt={user.login} />
          <span className="text-sm font-semibold text-gh-text-primary hidden md:inline">
          {user.login}
        </span>
        </div>
      </ActionMenu.Anchor>

      <ActionMenu.Overlay>
        <ActionList>
          <ActionList.Group>
            <ActionList.Item>
              <div className="flex flex-col gap-1">
                <div className="text-sm font-semibold text-gh-text-primary">
                  {user.name || user.login}
              </div>
                <div className="text-xs text-gh-text-muted">@{user.login}</div>
            {user.email && (
                  <div className="text-xs text-gh-text-muted">{user.email}</div>
            )}
          </div>
            </ActionList.Item>
          </ActionList.Group>
          
          <ActionList.Divider />

          <ActionList.Group>
            <ActionList.LinkItem
              href={`https://github.com/${user.login}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              <ActionList.LeadingVisual>
                <MarkGithubIcon size={16} />
              </ActionList.LeadingVisual>
              View GitHub Profile
            </ActionList.LinkItem>
            
            <ActionList.Item variant="danger" onSelect={handleLogout}>
              <ActionList.LeadingVisual>
                <SignOutIcon size={16} />
              </ActionList.LeadingVisual>
              Sign out
            </ActionList.Item>
          </ActionList.Group>
        </ActionList>
      </ActionMenu.Overlay>
    </ActionMenu>
  );
}

