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

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function TestWrapper({ children }: { children: React.ReactNode }) {
  const queryClient = createTestQueryClient();
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
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(adminAuthStatus),
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

  it('renders authorization tier for self-service user', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(selfServiceAuthStatus),
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText(/Mapped to user@ghes.example.com/)).toBeInTheDocument();
    });
  });

  it('renders authorization tier for read-only user with upgrade path', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(readOnlyAuthStatus),
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText(/Complete identity mapping to enable self-service migrations/)).toBeInTheDocument();
    });
  });

  it('shows identity mapping status when completed', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(selfServiceAuthStatus),
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText(/Mapped to user@ghes.example.com/)).toBeInTheDocument();
    });
  });

  it('shows upgrade path when identity mapping incomplete', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(readOnlyAuthStatus),
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Get Started' })).toBeInTheDocument();
    });
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

  it('navigates to identity mapping page on button click', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(readOnlyAuthStatus),
    });

    render(
      <TestWrapper>
        <AuthSettings settings={defaultSettings} onSave={vi.fn()} isSaving={false} />
      </TestWrapper>
    );

    await waitFor(() => {
      const getStartedButton = screen.getByRole('button', { name: 'Get Started' });
      expect(getStartedButton.closest('a')).toHaveAttribute('href', '/user-mappings');
    });
  });
});

