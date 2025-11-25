import { useAuth } from '../../contexts/AuthContext';
import { Avatar, ActionMenu, ActionList, useTheme } from '@primer/react';
import { MarkGithubIcon, SignOutIcon, SunIcon, MoonIcon } from '@primer/octicons-react';

const THEME_STORAGE_KEY = 'primer-theme-mode';

export function UserProfile() {
  const { user, logout, authEnabled, isAuthenticated } = useAuth();
  const { colorMode, setColorMode } = useTheme();

  const handleLogout = async () => {
    await logout();
  };

  const toggleTheme = () => {
    const newMode = colorMode === 'day' ? 'night' : 'day';
    setColorMode(newMode);
    localStorage.setItem(THEME_STORAGE_KEY, newMode);
    
    // Update data attributes for Primer CSS variables
    const root = document.documentElement;
    root.setAttribute('data-color-mode', newMode === 'day' ? 'light' : 'dark');
    root.setAttribute('data-light-theme', 'light');
    root.setAttribute('data-dark-theme', 'dark');
  };

  const getThemeLabel = () => {
    return colorMode === 'day' ? 'Switch to Dark' : 'Switch to Light';
  };

  const ThemeIcon = colorMode === 'day' ? MoonIcon : SunIcon;

  // If auth is disabled or not authenticated, show a dummy profile with theme toggle
  if (!authEnabled || !isAuthenticated || !user) {
    return (
      <ActionMenu>
        <ActionMenu.Anchor>
          <div className="flex items-center gap-2 cursor-pointer hover:opacity-80">
            <Avatar 
              src="https://avatars.githubusercontent.com/u/0?v=4" 
              size={32} 
              alt="Guest User" 
            />
            <span className="text-sm font-semibold hidden md:inline">Guest</span>
          </div>
        </ActionMenu.Anchor>

        <ActionMenu.Overlay sx={{ zIndex: 99999 }}>
          <ActionList>
            <ActionList.Group>
              <ActionList.Item>
                <div className="flex flex-col gap-1">
                  <div className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>Guest User</div>
                  <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>Authentication disabled</div>
                </div>
              </ActionList.Item>
            </ActionList.Group>
            
            <ActionList.Divider />

            <ActionList.Group>
              <ActionList.Item onSelect={toggleTheme}>
                <ActionList.LeadingVisual>
                  <ThemeIcon size={16} />
                </ActionList.LeadingVisual>
                {getThemeLabel()}
              </ActionList.Item>
            </ActionList.Group>
          </ActionList>
        </ActionMenu.Overlay>
      </ActionMenu>
    );
  }

  return (
    <ActionMenu>
      <ActionMenu.Anchor>
        <div className="flex items-center gap-2 cursor-pointer hover:opacity-80">
          <Avatar src={user.avatar_url} size={32} alt={user.login} />
            <span className="text-sm font-semibold hidden md:inline" style={{ color: 'var(--fgColor-default)' }}>
          {user.login}
        </span>
        </div>
      </ActionMenu.Anchor>

      <ActionMenu.Overlay sx={{ zIndex: 99999 }}>
        <ActionList>
          <ActionList.Group>
            <ActionList.Item>
              <div className="flex flex-col gap-1">
                <div className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>
                  {user.name || user.login}
              </div>
                <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>@{user.login}</div>
            {user.email && (
                  <div className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>{user.email}</div>
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
          </ActionList.Group>

          <ActionList.Divider />

          <ActionList.Group>
            <ActionList.Item onSelect={toggleTheme}>
              <ActionList.LeadingVisual>
                <ThemeIcon size={16} />
              </ActionList.LeadingVisual>
              {getThemeLabel()}
            </ActionList.Item>
            
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

