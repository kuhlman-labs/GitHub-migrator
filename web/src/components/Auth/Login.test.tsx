import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ThemeProvider } from '@primer/react';
import { Login } from './Login';

// Mock the AuthContext
const mockLogin = vi.fn();
const mockUseAuth = vi.fn();

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

function TestWrapper({ children }: { children: React.ReactNode }) {
  return <ThemeProvider>{children}</ThemeProvider>;
}

describe('Login', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({
      authConfig: null,
      login: mockLogin,
    });
  });

  it('should render the login page', () => {
    render(
      <TestWrapper>
        <Login />
      </TestWrapper>
    );

    expect(screen.getByRole('heading', { name: 'GitHub Migrator' })).toBeInTheDocument();
    expect(screen.getByText('Sign in to continue')).toBeInTheDocument();
  });

  it('should render sign in button', () => {
    render(
      <TestWrapper>
        <Login />
      </TestWrapper>
    );

    expect(screen.getByRole('button', { name: /sign in with github/i })).toBeInTheDocument();
  });

  it('should call login when sign in button is clicked', () => {
    render(
      <TestWrapper>
        <Login />
      </TestWrapper>
    );

    fireEvent.click(screen.getByRole('button', { name: /sign in with github/i }));
    expect(mockLogin).toHaveBeenCalledTimes(1);
  });

  it('should show redirect info text', () => {
    render(
      <TestWrapper>
        <Login />
      </TestWrapper>
    );

    expect(screen.getByText('You will be redirected to GitHub to authenticate')).toBeInTheDocument();
  });

  describe('authorization rules', () => {
    it('should not show access requirements when no rules', () => {
      render(
        <TestWrapper>
          <Login />
        </TestWrapper>
      );

      expect(screen.queryByText('Access Requirements')).not.toBeInTheDocument();
    });

    it('should show access requirements section when rules exist', () => {
      mockUseAuth.mockReturnValue({
        authConfig: {
          authorization_rules: {
            requires_org_membership: true,
            required_orgs: ['my-org'],
          },
        },
        login: mockLogin,
      });

      render(
        <TestWrapper>
          <Login />
        </TestWrapper>
      );

      expect(screen.getByText('Access Requirements')).toBeInTheDocument();
    });

    it('should show organization membership requirement', () => {
      mockUseAuth.mockReturnValue({
        authConfig: {
          authorization_rules: {
            requires_org_membership: true,
            required_orgs: ['org1', 'org2'],
          },
        },
        login: mockLogin,
      });

      render(
        <TestWrapper>
          <Login />
        </TestWrapper>
      );

      expect(screen.getByText(/Organization member: org1, org2/)).toBeInTheDocument();
    });

    it('should show team membership requirement', () => {
      mockUseAuth.mockReturnValue({
        authConfig: {
          authorization_rules: {
            requires_team_membership: true,
            required_teams: ['team-a', 'team-b'],
          },
        },
        login: mockLogin,
      });

      render(
        <TestWrapper>
          <Login />
        </TestWrapper>
      );

      expect(screen.getByText(/Team member: team-a, team-b/)).toBeInTheDocument();
    });

    it('should show enterprise admin requirement', () => {
      mockUseAuth.mockReturnValue({
        authConfig: {
          authorization_rules: {
            requires_enterprise_admin: true,
            enterprise: 'my-enterprise',
          },
        },
        login: mockLogin,
      });

      render(
        <TestWrapper>
          <Login />
        </TestWrapper>
      );

      expect(screen.getByText(/Enterprise admin of: my-enterprise/)).toBeInTheDocument();
    });

    it('should show enterprise membership requirement', () => {
      mockUseAuth.mockReturnValue({
        authConfig: {
          authorization_rules: {
            requires_enterprise_membership: true,
            enterprise: 'my-enterprise',
          },
        },
        login: mockLogin,
      });

      render(
        <TestWrapper>
          <Login />
        </TestWrapper>
      );

      expect(screen.getByText(/Member of enterprise: my-enterprise/)).toBeInTheDocument();
    });

    it('should prefer enterprise admin over enterprise membership', () => {
      mockUseAuth.mockReturnValue({
        authConfig: {
          authorization_rules: {
            requires_enterprise_admin: true,
            requires_enterprise_membership: true,
            enterprise: 'my-enterprise',
          },
        },
        login: mockLogin,
      });

      render(
        <TestWrapper>
          <Login />
        </TestWrapper>
      );

      expect(screen.getByText(/Enterprise admin of: my-enterprise/)).toBeInTheDocument();
      // Should not show both admin and member requirement
      expect(screen.queryAllByText(/Enterprise/)).toHaveLength(1);
    });
  });
});
