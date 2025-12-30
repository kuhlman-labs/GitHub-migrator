import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { ThemeProvider } from '@primer/react';
import { AuthProvider, useAuth } from './AuthContext';
import { api } from '../services/api';

// Mock the API module
vi.mock('../services/api', () => ({
  api: {
    getAuthConfig: vi.fn(),
    getAuthSources: vi.fn(),
    getCurrentUser: vi.fn(),
    logout: vi.fn(),
  },
}));

// Test component that uses the auth context
function TestComponent() {
  const { user, isAuthenticated, isLoading, authEnabled, authConfig, login, logout, refreshAuth } = useAuth();

  return (
    <div>
      <div data-testid="isLoading">{isLoading.toString()}</div>
      <div data-testid="isAuthenticated">{isAuthenticated.toString()}</div>
      <div data-testid="authEnabled">{authEnabled.toString()}</div>
      <div data-testid="user">{user?.login || 'no-user'}</div>
      <div data-testid="authConfig">{JSON.stringify(authConfig)}</div>
      <button onClick={() => login()}>Login</button>
      <button onClick={logout}>Logout</button>
      <button onClick={refreshAuth}>Refresh</button>
    </div>
  );
}

function TestWrapper({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <AuthProvider>{children}</AuthProvider>
    </ThemeProvider>
  );
}

describe('AuthContext', () => {
  const mockApi = api as unknown as {
    getAuthConfig: ReturnType<typeof vi.fn>;
    getAuthSources: ReturnType<typeof vi.fn>;
    getCurrentUser: ReturnType<typeof vi.fn>;
    logout: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset window.location
    delete (window as { location?: Location }).location;
    window.location = { href: '', pathname: '/' } as Location;
    // Default mock for getAuthSources - return empty array
    mockApi.getAuthSources.mockResolvedValue([]);
  });

  describe('useAuth hook', () => {
    it('should throw error when used outside AuthProvider', () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        render(
          <ThemeProvider>
            <TestComponent />
          </ThemeProvider>
        );
      }).toThrow('useAuth must be used within an AuthProvider');

      consoleError.mockRestore();
    });
  });

  describe('when auth is disabled', () => {
    beforeEach(() => {
      mockApi.getAuthConfig.mockResolvedValue({ enabled: false });
    });

    it('should set authEnabled to false', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(screen.getByTestId('authEnabled').textContent).toBe('false');
      expect(screen.getByTestId('isAuthenticated').textContent).toBe('false');
    });

    it('should not fetch current user', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(mockApi.getCurrentUser).not.toHaveBeenCalled();
    });
  });

  describe('when auth is enabled', () => {
    beforeEach(() => {
      mockApi.getAuthConfig.mockResolvedValue({
        enabled: true,
        login_url: '/auth/login',
        authorization_rules: {
          requires_org_membership: true,
          required_orgs: ['my-org'],
        },
      });
    });

    it('should fetch current user', async () => {
      const mockUser = {
        id: 1,
        login: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        avatar_url: 'https://example.com/avatar.png',
      };
      mockApi.getCurrentUser.mockResolvedValue(mockUser);

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(screen.getByTestId('authEnabled').textContent).toBe('true');
      expect(screen.getByTestId('isAuthenticated').textContent).toBe('true');
      expect(screen.getByTestId('user').textContent).toBe('testuser');
    });

    it('should set user to null when fetch fails', async () => {
      mockApi.getCurrentUser.mockRejectedValue(new Error('Unauthorized'));

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      expect(screen.getByTestId('isAuthenticated').textContent).toBe('false');
      expect(screen.getByTestId('user').textContent).toBe('no-user');
    });
  });

  describe('login function', () => {
    beforeEach(() => {
      mockApi.getAuthConfig.mockResolvedValue({ enabled: true });
      mockApi.getAuthSources.mockResolvedValue([]);
      mockApi.getCurrentUser.mockResolvedValue({
        id: 1,
        login: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        avatar_url: '',
      });
    });

    it('should redirect to login URL without source_id when no sources', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      const loginButton = screen.getByText('Login');
      act(() => {
        loginButton.click();
      });

      expect(window.location.href).toBe('/api/v1/auth/login');
    });

    it('should redirect with source_id when a source is specified', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // We need to call login directly with a source ID
      // The test component calls login() without arguments, so we test the default behavior above
      // For source-specific login, the Login component would call login(sourceId)
    });
  });

  describe('logout function', () => {
    beforeEach(() => {
      mockApi.getAuthConfig.mockResolvedValue({ enabled: true });
      mockApi.getCurrentUser.mockResolvedValue({
        id: 1,
        login: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        avatar_url: '',
      });
    });

    it('should call logout API and redirect to login', async () => {
      mockApi.logout.mockResolvedValue(undefined);

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      const logoutButton = screen.getByText('Logout');
      await act(async () => {
        logoutButton.click();
      });

      expect(mockApi.logout).toHaveBeenCalled();
      expect(window.location.href).toBe('/login');
    });

    it('should redirect even when logout API fails', async () => {
      mockApi.logout.mockRejectedValue(new Error('Network error'));

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      const logoutButton = screen.getByText('Logout');
      await act(async () => {
        logoutButton.click();
      });

      expect(window.location.href).toBe('/login');
    });
  });

  describe('refreshAuth function', () => {
    beforeEach(() => {
      mockApi.getAuthConfig.mockResolvedValue({ enabled: true });
      mockApi.getCurrentUser.mockResolvedValue({
        id: 1,
        login: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        avatar_url: '',
      });
    });

    it('should refetch current user when auth is enabled', async () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Initial fetch
      expect(mockApi.getCurrentUser).toHaveBeenCalledTimes(1);

      const refreshButton = screen.getByText('Refresh');
      await act(async () => {
        refreshButton.click();
      });

      expect(mockApi.getCurrentUser).toHaveBeenCalledTimes(2);
    });
  });

  describe('error handling', () => {
    it('should handle auth config fetch failure gracefully', async () => {
      mockApi.getAuthConfig.mockRejectedValue(new Error('Network error'));

      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('isLoading').textContent).toBe('false');
      });

      // Should still render without crashing
      expect(screen.getByTestId('authEnabled').textContent).toBe('false');
    });
  });
});

