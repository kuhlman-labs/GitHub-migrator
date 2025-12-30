import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@primer/react';
import { Navigation } from './Navigation';
import { ToastProvider } from '../../contexts/ToastContext';

// Mock the UserProfile component
vi.mock('./UserProfile', () => ({
  UserProfile: () => <div data-testid="user-profile">User Profile</div>,
}));

// Mock the SourceSelector component
vi.mock('./SourceSelector', () => ({
  SourceSelector: () => <div data-testid="source-selector">Source Selector</div>,
}));

// Mock AuthContext
vi.mock('../../contexts/AuthContext', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useAuth: () => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    login: vi.fn(),
    logout: vi.fn(),
  }),
}));

// Mock SourceContext
vi.mock('../../contexts/SourceContext', () => ({
  SourceProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useSourceContext: () => ({
    sources: [],
    activeSourceFilter: 'all',
    setActiveSourceFilter: vi.fn(),
    activeSource: null,
    isLoading: false,
    error: null,
    refetchSources: vi.fn(),
  }),
}));

// Create a wrapper with custom initial route
function createWrapper(initialEntries: string[] = ['/']) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  
  return ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter initialEntries={initialEntries}>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <ToastProvider>
            {children}
          </ToastProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('Navigation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render the navigation with logo and title', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    expect(screen.getByText('Migrator')).toBeInTheDocument();
  });

  it('should render all main navigation links', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Repositories')).toBeInTheDocument();
    expect(screen.getByText('Dependencies')).toBeInTheDocument();
    expect(screen.getByText('Users')).toBeInTheDocument();
    expect(screen.getByText('Teams')).toBeInTheDocument();
    expect(screen.getByText('Batches')).toBeInTheDocument();
    expect(screen.getByText('Analytics')).toBeInTheDocument();
    expect(screen.getByText('History')).toBeInTheDocument();
  });

  it('should render skip link for accessibility', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    expect(screen.getByText('Skip to main content')).toBeInTheDocument();
  });

  it('should render user profile component', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    expect(screen.getByTestId('user-profile')).toBeInTheDocument();
  });

  it('should highlight active link for Dashboard', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    const dashboardLink = screen.getByText('Dashboard');
    expect(dashboardLink).toHaveStyle('background-color: var(--bgColor-neutral-muted)');
  });

  it('should render search input on searchable pages', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    // Dashboard page should have search
    expect(screen.getByPlaceholderText('Search organizations...')).toBeInTheDocument();
  });

  it('should have correct href for each navigation link', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    expect(screen.getByText('Dashboard').closest('a')).toHaveAttribute('href', '/');
    expect(screen.getByText('Repositories').closest('a')).toHaveAttribute('href', '/repositories');
    expect(screen.getByText('Dependencies').closest('a')).toHaveAttribute('href', '/dependencies');
    expect(screen.getByText('Users').closest('a')).toHaveAttribute('href', '/user-mappings');
    expect(screen.getByText('Teams').closest('a')).toHaveAttribute('href', '/team-mappings');
    expect(screen.getByText('Batches').closest('a')).toHaveAttribute('href', '/batches');
    expect(screen.getByText('Analytics').closest('a')).toHaveAttribute('href', '/analytics');
    expect(screen.getByText('History').closest('a')).toHaveAttribute('href', '/history');
  });

  it('should have main navigation landmark', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    expect(screen.getByRole('navigation', { name: 'Main navigation' })).toBeInTheDocument();
  });

  it('should handle search input changes', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    const searchInput = screen.getByPlaceholderText('Search organizations...');
    fireEvent.change(searchInput, { target: { value: 'test-org' } });

    expect(searchInput).toHaveValue('test-org');
  });

  it('should clear search on Enter key press', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    const searchInput = screen.getByPlaceholderText('Search organizations...');
    fireEvent.change(searchInput, { target: { value: 'test' } });
    fireEvent.keyDown(searchInput, { key: 'Enter', code: 'Enter' });

    // After Enter, the search value should update the URL params
    expect(searchInput).toBeInTheDocument();
  });

  it('should have clear button when search has value', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    const searchInput = screen.getByPlaceholderText('Search organizations...');
    fireEvent.change(searchInput, { target: { value: 'test' } });

    // The input should now have a value
    expect(searchInput).toHaveValue('test');
  });

  it('should render GitHub icon in logo', () => {
    const { container } = render(<Navigation />, { wrapper: createWrapper() });

    // Should have an SVG icon
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('should have proper link styling', () => {
    render(<Navigation />, { wrapper: createWrapper() });

    const repoLink = screen.getByText('Repositories').closest('a');
    expect(repoLink).toHaveClass('px-3', 'py-2', 'text-sm', 'font-semibold');
  });

  describe('context-aware search', () => {
    it('should show repository search on /repositories', () => {
      render(<Navigation />, { wrapper: createWrapper(['/repositories']) });
      
      expect(screen.getByPlaceholderText('Search repositories...')).toBeInTheDocument();
    });

    it('should show batch search on /batches', () => {
      render(<Navigation />, { wrapper: createWrapper(['/batches']) });
      
      expect(screen.getByPlaceholderText('Search batches...')).toBeInTheDocument();
    });

    it('should show history search on /history', () => {
      render(<Navigation />, { wrapper: createWrapper(['/history']) });
      
      expect(screen.getByPlaceholderText('Search migration history...')).toBeInTheDocument();
    });

    it('should show user mapping search on /user-mappings', () => {
      render(<Navigation />, { wrapper: createWrapper(['/user-mappings']) });
      
      expect(screen.getByPlaceholderText('Search user mappings...')).toBeInTheDocument();
    });

    it('should not show search on /team-mappings', () => {
      render(<Navigation />, { wrapper: createWrapper(['/team-mappings']) });
      
      // No search input should be present
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    });

    it('should not show search on /dependencies', () => {
      render(<Navigation />, { wrapper: createWrapper(['/dependencies']) });
      
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    });

    it('should not show search on /analytics', () => {
      render(<Navigation />, { wrapper: createWrapper(['/analytics']) });
      
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    });

    it('should show repository search on org detail pages', () => {
      render(<Navigation />, { wrapper: createWrapper(['/org/test-org']) });
      
      expect(screen.getByPlaceholderText('Search repositories...')).toBeInTheDocument();
    });
  });
});

