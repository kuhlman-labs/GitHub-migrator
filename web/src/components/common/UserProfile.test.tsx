import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { UserProfile } from './UserProfile';
import * as AuthContext from '../../contexts/AuthContext';

// Mock useAuth
vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

// Mock FallbackAvatar
vi.mock('./FallbackAvatar', () => ({
  FallbackAvatar: ({ login }: { login: string }) => (
    <div data-testid="fallback-avatar">{login}</div>
  ),
}));

describe('UserProfile', () => {
  const mockLogout = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('when auth is disabled', () => {
    beforeEach(() => {
      vi.mocked(AuthContext.useAuth).mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        authEnabled: false,
        loginUrl: null,
        logout: mockLogout,
      });
    });

    it('should render guest user', () => {
      render(<UserProfile />);

      // Guest appears in both avatar and span
      const guestElements = screen.getAllByText('Guest');
      expect(guestElements.length).toBeGreaterThanOrEqual(1);
      expect(screen.getByTestId('fallback-avatar')).toBeInTheDocument();
    });

    it('should show guest menu when clicked', () => {
      render(<UserProfile />);

      // Click on the avatar/name to open menu
      const triggers = screen.getAllByText('Guest');
      fireEvent.click(triggers[0]);

      expect(screen.getByText('Guest User')).toBeInTheDocument();
      expect(screen.getByText('Authentication disabled')).toBeInTheDocument();
    });
  });

  describe('when authenticated', () => {
    beforeEach(() => {
      vi.mocked(AuthContext.useAuth).mockReturnValue({
        user: {
          id: 1,
          login: 'testuser',
          name: 'Test User',
          email: 'test@example.com',
          avatar_url: 'https://example.com/avatar.png',
          roles: ['admin'],
        },
        isLoading: false,
        isAuthenticated: true,
        authEnabled: true,
        loginUrl: '/login',
        logout: mockLogout,
      });
    });

    it('should render user login', () => {
      render(<UserProfile />);

      // testuser appears in both avatar and span
      expect(screen.getAllByText('testuser').length).toBeGreaterThanOrEqual(1);
    });

    it('should show user menu when clicked', () => {
      render(<UserProfile />);

      // Click on the avatar/name to open menu
      const triggers = screen.getAllByText('testuser');
      fireEvent.click(triggers[0]);

      expect(screen.getByText('Test User')).toBeInTheDocument();
      expect(screen.getByText('@testuser')).toBeInTheDocument();
      expect(screen.getByText('test@example.com')).toBeInTheDocument();
    });

    it('should have link to GitHub profile', () => {
      render(<UserProfile />);

      // Click on the avatar/name to open menu
      const triggers = screen.getAllByText('testuser');
      fireEvent.click(triggers[0]);

      expect(screen.getByText('View GitHub Profile')).toBeInTheDocument();
    });

    it('should have sign out option', () => {
      render(<UserProfile />);

      // Click on the avatar/name to open menu
      const triggers = screen.getAllByText('testuser');
      fireEvent.click(triggers[0]);

      expect(screen.getByText('Sign out')).toBeInTheDocument();
    });

    it('should have theme toggle', () => {
      render(<UserProfile />);

      // Click on the avatar/name to open menu
      const triggers = screen.getAllByText('testuser');
      fireEvent.click(triggers[0]);

      // Should show theme toggle option
      expect(screen.getByText(/Switch to/)).toBeInTheDocument();
    });
  });

  describe('when user has no name', () => {
    beforeEach(() => {
      vi.mocked(AuthContext.useAuth).mockReturnValue({
        user: {
          id: 1,
          login: 'noname',
          name: '',
          email: '',
          avatar_url: '',
          roles: [],
        },
        isLoading: false,
        isAuthenticated: true,
        authEnabled: true,
        loginUrl: '/login',
        logout: mockLogout,
      });
    });

    it('should show login as fallback for name', () => {
      render(<UserProfile />);

      // Click on the avatar/name to open menu
      const triggers = screen.getAllByText('noname');
      fireEvent.click(triggers[0]);

      // Should show login multiple times (in avatar, in header, once in menu header, once as @username)
      expect(screen.getAllByText(/noname/).length).toBeGreaterThanOrEqual(2);
    });

    it('should not show email when empty', () => {
      render(<UserProfile />);

      // Click on the avatar/name to open menu
      const triggers = screen.getAllByText('noname');
      fireEvent.click(triggers[0]);

      // No email displayed in menu (should not have an email with @ and a dot after)
      const emailItems = screen.queryAllByText(/@[\w.-]+\./);
      expect(emailItems.length).toBe(0);
    });
  });
});

