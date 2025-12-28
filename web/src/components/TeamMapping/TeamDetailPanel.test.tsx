import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { TeamDetailPanel } from './TeamDetailPanel';

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useTeamDetail: vi.fn(),
}));

vi.mock('../../hooks/useMutations', () => ({
  useExecuteTeamMigration: vi.fn(),
}));

import { useTeamDetail } from '../../hooks/useQueries';
import { useExecuteTeamMigration } from '../../hooks/useMutations';

const mockTeamDetail = {
  id: 1,
  organization: 'source-org',
  slug: 'team-alpha',
  name: 'Team Alpha',
  description: 'A test team for development',
  privacy: 'closed',
  discovered_at: '2024-01-01T00:00:00Z',
  members: [
    { login: 'user1', role: 'maintainer' as const },
    { login: 'user2', role: 'member' as const },
    { login: 'user3', role: 'member' as const },
  ],
  repositories: [
    { full_name: 'source-org/repo1', permission: 'admin' as const, migration_status: 'complete' },
    { full_name: 'source-org/repo2', permission: 'push' as const, migration_status: 'pending' },
    { full_name: 'source-org/repo3', permission: 'pull' as const },
  ],
  mapping: {
    destination_org: 'dest-org',
    destination_team_slug: 'team-alpha-new',
    mapping_status: 'mapped' as const,
    migration_status: 'completed' as const,
    repos_synced: 2,
    repos_eligible: 3,
    total_source_repos: 3,
    team_created_in_dest: true,
    migration_completeness: 'partial' as const,
    last_synced_at: '2024-01-15T10:00:00Z',
  },
};

const mockUnmappedTeamDetail = {
  ...mockTeamDetail,
  mapping: undefined,
};

const mockMappedNotMigratedTeamDetail = {
  ...mockTeamDetail,
  mapping: {
    destination_org: 'dest-org',
    destination_team_slug: 'team-alpha-new',
    mapping_status: 'mapped' as const,
    migration_status: 'pending' as const,
    repos_synced: 0,
    repos_eligible: 3,
    total_source_repos: 3,
    team_created_in_dest: false,
    migration_completeness: 'pending' as const,
  },
};

describe('TeamDetailPanel', () => {
  const mockOnClose = vi.fn();
  const mockOnEditMapping = vi.fn();
  const mockOnMigrationStarted = vi.fn();
  const mockRefetch = vi.fn();
  const mockMutateAsync = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    
    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockTeamDetail,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });
    
    (useExecuteTeamMigration as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });
  });

  it('renders loading state', () => {
    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders error state', () => {
    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error('Network error'),
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText(/Failed to load team details/)).toBeInTheDocument();
    expect(screen.getByText(/Network error/)).toBeInTheDocument();
  });

  it('renders team header with name and org/slug', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Team Alpha')).toBeInTheDocument();
    // There are multiple elements with this text, use getAllByText
    const orgSlugElements = screen.getAllByText(/source-org\/team-alpha/);
    expect(orgSlugElements.length).toBeGreaterThanOrEqual(1);
  });

  it('renders team info badges', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('closed')).toBeInTheDocument();
    expect(screen.getByText('3 members')).toBeInTheDocument();
    expect(screen.getByText('3 repos')).toBeInTheDocument();
  });

  it('renders team description', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('A test team for development')).toBeInTheDocument();
  });

  it('renders destination mapping when mapped', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText(/dest-org\/team-alpha-new/)).toBeInTheDocument();
    // The mapping status is shown in a label
    const mappedLabels = screen.getAllByText('mapped');
    expect(mappedLabels.length).toBeGreaterThanOrEqual(1);
  });

  it('renders "No mapping configured" when unmapped', () => {
    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockUnmappedTeamDetail,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
        onEditMapping={mockOnEditMapping}
      />
    );

    expect(screen.getByText('No mapping configured')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create Mapping' })).toBeInTheDocument();
  });

  it('renders repo sync progress when team is created in destination', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Repo Permissions Synced')).toBeInTheDocument();
    expect(screen.getByText(/2 \/ 3/)).toBeInTheDocument();
  });

  it('renders members tab by default', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('user1')).toBeInTheDocument();
    expect(screen.getByText('user2')).toBeInTheDocument();
    expect(screen.getByText('user3')).toBeInTheDocument();
    expect(screen.getByText('maintainer')).toBeInTheDocument();
  });

  it('switches to repositories tab when clicked', async () => {
    const user = userEvent.setup();
    
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    // Click on repositories tab
    const repoTab = screen.getByText(/Repositories/);
    await user.click(repoTab);

    // Check that repositories are displayed
    expect(screen.getByText('source-org/repo1')).toBeInTheDocument();
    expect(screen.getByText('source-org/repo2')).toBeInTheDocument();
    expect(screen.getByText('source-org/repo3')).toBeInTheDocument();
    expect(screen.getByText('admin')).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', async () => {
    const user = userEvent.setup();
    
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    const closeButton = screen.getByRole('button', { name: 'Close' });
    await user.click(closeButton);

    expect(mockOnClose).toHaveBeenCalled();
  });

  it('calls onEditMapping when Edit Mapping button is clicked', async () => {
    const user = userEvent.setup();
    
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
        onEditMapping={mockOnEditMapping}
      />
    );

    const editButton = screen.getByRole('button', { name: 'Edit Mapping' });
    await user.click(editButton);

    expect(mockOnEditMapping).toHaveBeenCalledWith('source-org', 'team-alpha');
  });

  it('shows Migrate Team button when mapping exists but not migrated', async () => {
    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockMappedNotMigratedTeamDetail,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
        onMigrationStarted={mockOnMigrationStarted}
      />
    );

    expect(screen.getByRole('button', { name: /Migrate Team/ })).toBeInTheDocument();
  });

  it('shows Sync Permissions button when team needs sync', async () => {
    const needsSyncTeam = {
      ...mockTeamDetail,
      mapping: {
        ...mockTeamDetail.mapping,
        sync_status: 'needs_sync' as const,
      },
    };

    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: needsSyncTeam,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByRole('button', { name: /Sync Permissions/ })).toBeInTheDocument();
  });

  it('triggers migration when Migrate Team button is clicked', async () => {
    const user = userEvent.setup();
    
    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockMappedNotMigratedTeamDetail,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    mockMutateAsync.mockResolvedValue({});

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
        onMigrationStarted={mockOnMigrationStarted}
      />
    );

    const migrateButton = screen.getByRole('button', { name: /Migrate Team/ });
    await user.click(migrateButton);

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        source_org: 'source-org',
        source_team_slug: 'team-alpha',
        dry_run: false,
      });
    });
  });

  it('shows error message when mapping has error', () => {
    const teamWithError = {
      ...mockTeamDetail,
      mapping: {
        ...mockTeamDetail.mapping,
        error_message: 'Failed to create team: permission denied',
      },
    };

    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: teamWithError,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Failed to create team: permission denied')).toBeInTheDocument();
  });

  it('shows last synced timestamp when available', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText(/Last synced:/)).toBeInTheDocument();
  });

  it('shows empty members message when no members', () => {
    const teamNoMembers = {
      ...mockTeamDetail,
      members: [],
    };

    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: teamNoMembers,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('No members discovered')).toBeInTheDocument();
  });

  it('shows empty repositories message when no repos', async () => {
    const user = userEvent.setup();
    
    const teamNoRepos = {
      ...mockTeamDetail,
      repositories: [],
    };

    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: teamNoRepos,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    // Switch to repositories tab
    const repoTab = screen.getByText(/Repositories/);
    await user.click(repoTab);

    expect(screen.getByText('No repositories discovered')).toBeInTheDocument();
  });

  it('renders completeness badge and description', () => {
    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Partial')).toBeInTheDocument();
    expect(screen.getByText('Some repo permissions synced')).toBeInTheDocument();
  });

  it('renders "Not mapped" when destination is not set', () => {
    const unmappedDestTeam = {
      ...mockTeamDetail,
      mapping: {
        ...mockTeamDetail.mapping,
        destination_org: undefined,
        destination_team_slug: undefined,
        mapping_status: 'unmapped' as const,
      },
    };

    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: unmappedDestTeam,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <TeamDetailPanel
        org="source-org"
        teamSlug="team-alpha"
        onClose={mockOnClose}
      />
    );

    expect(screen.getByText('Not mapped')).toBeInTheDocument();
  });
});

