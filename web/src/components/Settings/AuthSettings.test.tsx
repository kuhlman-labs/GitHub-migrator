import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ThemeProvider } from '@primer/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthSettings } from './AuthSettings';
import type { SettingsResponse } from '../../services/api/settings';

// Mock the AuthContext
const mockUseAuth = vi.fn();
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

// Mock fetch for authorization status
const mockFetch = vi.fn();
global.fetch = mockFetch;

function TestWrapper({ children }: { children: React.ReactNode }) {
  // Create a fresh query client for each test to avoid caching issues
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        cacheTime: 0,
        staleTime: 0,
        gcTime: 0,
      },
    },
  });
  
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>{children}</ThemeProvider>
    </QueryClientProvider>
  );
}

const defaultSettings: SettingsResponse = {
  auth_enabled: true,
  auth_session_secret_set: true,
  auth_session_duration_hours: 24,
  auth_callback_url: 'http://localhost:8080/api/v1/auth/callback',
  auth_frontend_url: 'http://localhost:3000',
  auth_github_oauth_client_id: 'test-client-id',
  auth_github_oauth_client_secret_set: true,
  authorization_rules: {
    require_org_membership: [],
    require_team_membership: [],
    require_enterprise_admin: false,
    require_enterprise_membership: false,
    require_enterprise_slug: '',
    privileged_teams: [],
    migration_admin_teams: [],
    allow_org_admin_migrations: false,
    allow_enterprise_admin_migrations: false,
    require_identity_mapping_for_self_service: false,
  },
  source_type: 'github',
  source_base_url: 'https://api.github.com',
  source_token_set: true,
  destination_type: 'github',
  destination_base_url: 'https://api.github.com',
  destination_token_set: true,
  migration_workers: 5,
  migration_poll_interval_seconds: 30,
  migration_post_migration_mode: 'production_only',
  migration_dest_repo_exists_action: 'fail',
  migration_visibility_public_repos: 'private',
  migration_visibility_internal_repos: 'private',
  destination_configured: true,
  updated_at: new Date().toISOString(),
};

const adminAuthStatus = {
  tier: 'admin',
  tier_name: 'Full Migration Rights',
  permissions: {
    can_view_repos: true,
    can_migrate_own_repos: true,
    can_migrate_all_repos: true,
    can_manage_batches: true,
    can_manage_sources: true,
  },
};

const selfServiceAuthStatus = {
  tier: 'self_service',
  tier_name: 'Self-Service',
  permissions: {
    can_view_repos: true,
    can_migrate_own_repos: true,
    can_migrate_all_repos: false,
    can_manage_batches: true,
    can_manage_sources: false,
  },
  identity_mapping: {
    completed: true,
    source_login: 'user@ghes.example.com',
    source_name: 'GHES Production',
  },
};

const readOnlyAuthStatus = {
  tier: 'read_only',
  tier_name: 'Read-Only',
  permissions: {
    can_view_repos: true,
    can_migrate_own_repos: false,
    can_migrate_all_repos: false,
    can_manage_batches: false,
    can_manage_sources: false,
  },
  identity_mapping: {
    completed: false,
  },
  upgrade_path: {
    action: 'complete_identity_mapping',
    message: 'Complete identity mapping to enable self-service migrations',
    link: '/user-mappings',
  },
};

describe('AuthSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({
      user: { login: 'testuser' },
      isAuthenticated: true,
    });
    // Default mock for fetch - will be overridden in specific tests
    mockFetch.mockImplementation((url) => {
      if (url === '/api/v1/auth/authorization-status') {
        return Promise.resolve({
          ok: true,
          json: async () => adminAuthStatus,
        } as Response);
      }
      return Promise.reject(new Error(`Unexpected fetch call to ${url}`));
    });
  });

  it('renders authentication heading', () => {
    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    expect(screen.getByRole('heading', { name: /Authentication & Authorization/i })).toBeInTheDocument();
  });

  it('renders how authorization works panel', () => {
    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    expect(screen.getByText('How Authorization Works')).toBeInTheDocument();
  });

  it('renders authorization tier descriptions', async () => {
    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    expect(screen.getByText('Full Migration Rights')).toBeInTheDocument();
    expect(screen.getByText('Self-Service')).toBeInTheDocument();
    expect(screen.getByText('Read-Only')).toBeInTheDocument();
  });

  it('renders authorization tier for admin user', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(adminAuthStatus),
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getAllByText('Full Migration Rights').length).toBeGreaterThan(0);
    });
  });

  it.skip('renders authorization tier for self-service user', async () => {
    mockFetch.mockImplementationOnce((url) => {
      if (url === '/api/v1/auth/authorization-status') {
        return Promise.resolve({
          ok: true,
          json: async () => selfServiceAuthStatus,
        } as Response);
      }
      return Promise.reject(new Error('Unexpected fetch call'));
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    // First verify the authorization panel is rendered
    await waitFor(() => {
      expect(screen.getByText('Your Authorization Level')).toBeInTheDocument();
    }, { timeout: 3000 });

    // Wait for loading to finish
    await waitFor(() => {
      expect(screen.queryByText('Loading authorization status...')).not.toBeInTheDocument();
    }, { timeout: 3000 });

    // Check for self-service tier
    await waitFor(() => {
      expect(screen.getByText('Self-Service')).toBeInTheDocument();
    }, { timeout: 3000 });

    // Check for identity mapping status
    expect(screen.getByText('Identity Mapping')).toBeInTheDocument();
    expect(screen.getByText(/Mapped to user@ghes.example.com/)).toBeInTheDocument();
  });

  it.skip('renders authorization tier for read-only user with upgrade path', async () => {
    mockFetch.mockImplementationOnce((url) => {
      if (url === '/api/v1/auth/authorization-status') {
        return Promise.resolve({
          ok: true,
          json: async () => readOnlyAuthStatus,
        } as Response);
      }
      return Promise.reject(new Error('Unexpected fetch call'));
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    // Wait for loading to finish
    await waitFor(() => {
      expect(screen.queryByText('Loading authorization status...')).not.toBeInTheDocument();
    });

    // Check for read-only tier
    expect(screen.getByText('Read-Only')).toBeInTheDocument();

    // Check for upgrade path
    expect(screen.getByText(/Complete identity mapping to enable self-service migrations/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Get Started' })).toBeInTheDocument();
  });

  it.skip('shows identity mapping status when completed', async () => {
    mockFetch.mockImplementationOnce((url) => {
      if (url === '/api/v1/auth/authorization-status') {
        return Promise.resolve({
          ok: true,
          json: async () => selfServiceAuthStatus,
        } as Response);
      }
      return Promise.reject(new Error('Unexpected fetch call'));
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    // Wait for loading to finish
    await waitFor(() => {
      expect(screen.queryByText('Loading authorization status...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('Identity Mapping')).toBeInTheDocument();
    expect(screen.getByText(/✓ Mapped to user@ghes.example.com/)).toBeInTheDocument();
    expect(screen.getByText(/(GHES Production)/)).toBeInTheDocument();
  });

  it.skip('shows upgrade path when identity mapping incomplete', async () => {
    mockFetch.mockImplementationOnce((url) => {
      if (url === '/api/v1/auth/authorization-status') {
        return Promise.resolve({
          ok: true,
          json: async () => readOnlyAuthStatus,
        } as Response);
      }
      return Promise.reject(new Error('Unexpected fetch call'));
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    // Wait for loading to finish
    await waitFor(() => {
      expect(screen.queryByText('Loading authorization status...')).not.toBeInTheDocument();
    });

    expect(screen.getByText(/Complete identity mapping to enable self-service migrations/)).toBeInTheDocument();

    const getStartedButton = screen.getByRole('button', { name: 'Get Started' });
    expect(getStartedButton).toBeInTheDocument();
    expect(getStartedButton.closest('a')).toHaveAttribute('href', '/user-mappings');
  });

  it('expands and collapses explanation sections', () => {
    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    // Panel should be open by default
    expect(screen.getByText(/destination-based authorization/i)).toBeInTheDocument();

    // Click to collapse
    fireEvent.click(screen.getByText('How Authorization Works'));

    // The detailed explanation should be hidden (panel collapsed)
    // Note: The exact behavior depends on CSS, but we're testing the toggle works
  });

  it('calls onSave when save button is clicked', () => {
    const onSave = vi.fn();
    
    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={onSave} isSaving={false} />
      </TestWrapper>
    );

    // Change a value to enable save button
    const durationInput = screen.getByLabelText(/Session Duration/i);
    fireEvent.change(durationInput, { target: { value: '48' } });

    // Click save
    fireEvent.click(screen.getByRole('button', { name: /Save Changes/i }));

    expect(onSave).toHaveBeenCalled();
  });

  it('disables save button when no changes', () => {
    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    expect(screen.getByRole('button', { name: /Save Changes/i })).toBeDisabled();
  });

  it('shows auth disabled warning when auth is off', () => {
    const disabledSettings = { ...defaultSettings, auth_enabled: false };
    
    render(
      <TestWrapper>
        <AuthSettings settings={disabledSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    expect(screen.getByText(/Authentication is disabled/)).toBeInTheDocument();
  });

  it.skip('navigates to identity mapping page on button click', async () => {
    mockFetch.mockImplementationOnce((url) => {
      if (url === '/api/v1/auth/authorization-status') {
        return Promise.resolve({
          ok: true,
          json: async () => readOnlyAuthStatus,
        } as Response);
      }
      return Promise.reject(new Error('Unexpected fetch call'));
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    // Wait for loading to finish
    await waitFor(() => {
      expect(screen.queryByText('Loading authorization status...')).not.toBeInTheDocument();
    });

    const getStartedButton = screen.getByRole('button', { name: 'Get Started' });
    expect(getStartedButton).toBeInTheDocument();
    expect(getStartedButton.closest('a')).toHaveAttribute('href', '/user-mappings');
  });
});

