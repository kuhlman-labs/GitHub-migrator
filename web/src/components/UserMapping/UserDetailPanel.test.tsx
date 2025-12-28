import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { UserDetailPanel } from './UserDetailPanel';

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useUserDetail: vi.fn(),
}));

import { useUserDetail } from '../../hooks/useQueries';

const mockUserDetail = {
  login: 'johndoe',
  name: 'John Doe',
  email: 'john@example.com',
  avatar_url: 'https://github.com/johndoe.png',
  source_instance: 'github.example.com',
  discovered_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-15T00:00:00Z',
  stats: {
    commit_count: 100,
    issue_count: 25,
    pr_count: 50,
    comment_count: 200,
    repository_count: 10,
  },
  organizations: [
    { id: 1, user_login: 'johndoe', organization: 'org-1', role: 'admin' as const, discovered_at: '2024-01-01' },
    { id: 2, user_login: 'johndoe', organization: 'org-2', role: 'member' as const, discovered_at: '2024-01-01' },
  ],
  mapping: {
    destination_login: 'johndoe-new',
    destination_email: 'john.doe@company.com',
    mapping_status: 'mapped' as const,
    mannequin_id: 'mq-123',
    mannequin_login: 'mona-johndoe',
    reclaim_status: 'invited' as const,
    match_confidence: 95,
    match_reason: 'email_exact',
  },
};

const mockUnmappedUserDetail = {
  ...mockUserDetail,
  mapping: undefined,
};

const mockUserWithReclaimError = {
  ...mockUserDetail,
  mapping: {
    ...mockUserDetail.mapping,
    reclaim_status: 'failed' as const,
    reclaim_error: 'User email not verified',
  },
};

describe('UserDetailPanel', () => {
  const mockOnClose = vi.fn();
  const mockOnEditMapping = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    (useUserDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockUserDetail,
      isLoading: false,
      error: null,
    });
  });

  it('renders loading state', () => {
    (useUserDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
    });

    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders error state', () => {
    (useUserDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error('Network error'),
    });

    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText(/Failed to load user details/)).toBeInTheDocument();
    expect(screen.getByText(/Network error/)).toBeInTheDocument();
  });

  it('renders user header with name and login', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('John Doe')).toBeInTheDocument();
    expect(screen.getByText('@johndoe')).toBeInTheDocument();
  });

  it('renders user information section', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('User Information')).toBeInTheDocument();
    expect(screen.getByText('Email')).toBeInTheDocument();
    expect(screen.getByText('john@example.com')).toBeInTheDocument();
    expect(screen.getByText('Source Instance')).toBeInTheDocument();
    expect(screen.getByText('github.example.com')).toBeInTheDocument();
  });

  it('renders organizations section', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText(/Source Organizations/)).toBeInTheDocument();
    expect(screen.getByText('org-1')).toBeInTheDocument();
    expect(screen.getByText('org-2')).toBeInTheDocument();
    expect(screen.getByText('admin')).toBeInTheDocument();
  });

  it('renders mapping status section when mapped', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Mapping Status')).toBeInTheDocument();
    expect(screen.getByText('mapped')).toBeInTheDocument();
    expect(screen.getByText('@johndoe-new')).toBeInTheDocument();
    expect(screen.getByText('mona-johndoe')).toBeInTheDocument();
  });

  it('renders reclaim status when present', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Reclaim Status')).toBeInTheDocument();
    expect(screen.getByText('invited')).toBeInTheDocument();
  });

  it('renders match confidence', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Match Confidence')).toBeInTheDocument();
    expect(screen.getByText('95%')).toBeInTheDocument();
  });

  it('renders match reason', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Match Reason')).toBeInTheDocument();
    expect(screen.getByText('email_exact')).toBeInTheDocument();
  });

  it('renders unmapped state when no mapping', () => {
    (useUserDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockUnmappedUserDetail,
      isLoading: false,
      error: null,
    });

    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
        onEditMapping={mockOnEditMapping}
      />
    );

    expect(screen.getByText('unmapped')).toBeInTheDocument();
    expect(screen.getByText(/This user has not been mapped/)).toBeInTheDocument();
  });

  it('renders reclaim error when present', () => {
    (useUserDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockUserWithReclaimError,
      isLoading: false,
      error: null,
    });

    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('User email not verified')).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', async () => {
    const user = userEvent.setup();

    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    const closeButtons = screen.getAllByRole('button', { name: 'Close' });
    await user.click(closeButtons[0]);

    expect(mockOnClose).toHaveBeenCalled();
  });

  it('calls onEditMapping when Edit Mapping button is clicked', async () => {
    const user = userEvent.setup();

    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
        onEditMapping={mockOnEditMapping}
      />
    );

    const editButton = screen.getByRole('button', { name: 'Edit Mapping' });
    await user.click(editButton);

    expect(mockOnEditMapping).toHaveBeenCalledWith('johndoe');
  });

  it('does not render Edit Mapping button when onEditMapping is not provided', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.queryByRole('button', { name: 'Edit Mapping' })).not.toBeInTheDocument();
  });

  it('shows no organizations message when empty', () => {
    (useUserDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: {
        ...mockUserDetail,
        organizations: [],
      },
      isLoading: false,
      error: null,
    });

    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText(/No organization memberships found/)).toBeInTheDocument();
  });

  it('renders destination login with @ prefix', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('@johndoe-new')).toBeInTheDocument();
  });

  it('renders mannequin login', () => {
    render(
      <UserDetailPanel
        login="johndoe"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Mannequin')).toBeInTheDocument();
    expect(screen.getByText('mona-johndoe')).toBeInTheDocument();
  });
});

