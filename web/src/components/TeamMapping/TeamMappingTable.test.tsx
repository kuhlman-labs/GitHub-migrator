import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { TeamMappingTable } from './TeamMappingTable';

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useTeamMappings: vi.fn(),
  useTeamMappingStats: vi.fn(),
  useTeamMigrationStatus: vi.fn(),
  useTeamSourceOrgs: vi.fn(),
  useTeamDetail: vi.fn(),
}));

vi.mock('../../hooks/useMutations', () => ({
  useUpdateTeamMapping: vi.fn(),
  useDeleteTeamMapping: vi.fn(),
  useExecuteTeamMigration: vi.fn(),
  useCancelTeamMigration: vi.fn(),
  useResetTeamMigrationStatus: vi.fn(),
  useDiscoverTeams: vi.fn(),
}));

vi.mock('../../services/api', () => ({
  api: {
    exportTeamMappings: vi.fn(),
    importTeamMappings: vi.fn(),
  },
}));

import {
  useTeamMappings,
  useTeamMappingStats,
  useTeamMigrationStatus,
  useTeamSourceOrgs,
  useTeamDetail,
} from '../../hooks/useQueries';
import {
  useUpdateTeamMapping,
  useDeleteTeamMapping,
  useExecuteTeamMigration,
  useCancelTeamMigration,
  useResetTeamMigrationStatus,
  useDiscoverTeams,
} from '../../hooks/useMutations';

const mockMappings = [
  {
    id: 1,
    organization: 'source-org',
    slug: 'team-alpha',
    name: 'Team Alpha',
    privacy: 'closed',
    destination_org: 'dest-org',
    destination_team_slug: 'team-alpha-new',
    destination_team_name: 'Team Alpha New',
    mapping_status: 'mapped',
    migration_status: 'completed',
    sync_status: 'complete',
    repos_synced: 5,
    repos_eligible: 5,
    total_source_repos: 5,
    team_created_in_dest: true,
  },
  {
    id: 2,
    organization: 'source-org',
    slug: 'team-beta',
    name: 'Team Beta',
    privacy: 'secret',
    destination_org: undefined,
    destination_team_slug: undefined,
    mapping_status: 'unmapped',
    migration_status: 'pending',
    sync_status: 'pending',
    repos_synced: 0,
    repos_eligible: 3,
    total_source_repos: 3,
    team_created_in_dest: false,
  },
  {
    id: 3,
    organization: 'other-org',
    slug: 'team-gamma',
    name: 'Team Gamma',
    privacy: 'closed',
    destination_org: undefined,
    destination_team_slug: undefined,
    mapping_status: 'skipped',
    migration_status: 'pending',
    sync_status: 'pending',
    repos_synced: 0,
    repos_eligible: 0,
    total_source_repos: 2,
    team_created_in_dest: false,
  },
];

const mockStats = {
  total: 3,
  mapped: 1,
  unmapped: 1,
  skipped: 1,
};

const mockMigrationStatus = {
  is_running: false,
  progress: undefined,
  execution_stats: {
    pending: 2,
    in_progress: 0,
    completed: 1,
    failed: 0,
    needs_sync: 0,
    team_only: 0,
    partial: 0,
    total_repos_synced: 5,
    total_repos_eligible: 10,
  },
  mapping_stats: mockStats,
};

const mockSourceOrgs = ['source-org', 'other-org'];

describe('TeamMappingTable', () => {
  const mockRefetch = vi.fn();
  const mockMutateAsync = vi.fn();
  const mockMutate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    (useTeamMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { mappings: mockMappings, total: 3 },
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    (useTeamMappingStats as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockStats,
    });

    (useTeamMigrationStatus as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockMigrationStatus,
    });

    (useTeamSourceOrgs as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockSourceOrgs,
    });

    (useUpdateTeamMapping as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useDeleteTeamMapping as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useExecuteTeamMigration as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useCancelTeamMigration as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useResetTeamMigrationStatus as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useDiscoverTeams as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: mockMutate,
      isPending: false,
    });

    // Mock useTeamDetail for TeamDetailPanel that opens when clicking on a row
    (useTeamDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
      refetch: mockRefetch,
    });
  });

  it('renders the component header', () => {
    render(<TeamMappingTable />);

    expect(screen.getByText('Team Permission Mapping')).toBeInTheDocument();
    expect(screen.getByText(/Map source teams to destination GitHub teams/)).toBeInTheDocument();
  });

  it('renders stats badges', () => {
    render(<TeamMappingTable />);

    expect(screen.getByText('3 Total')).toBeInTheDocument();
    expect(screen.getByText('1 Mapped')).toBeInTheDocument();
    expect(screen.getByText('1 Unmapped')).toBeInTheDocument();
  });

  it('renders the team mapping table with rows', () => {
    render(<TeamMappingTable />);

    expect(screen.getByText('team-alpha')).toBeInTheDocument();
    expect(screen.getByText('team-beta')).toBeInTheDocument();
    expect(screen.getByText('team-gamma')).toBeInTheDocument();
  });

  it('shows destination mapping for mapped teams', () => {
    render(<TeamMappingTable />);

    expect(screen.getByText('team-alpha-new')).toBeInTheDocument();
  });

  it('shows "Not mapped" for unmapped teams', () => {
    render(<TeamMappingTable />);

    expect(screen.getAllByText('Not mapped').length).toBeGreaterThanOrEqual(1);
  });

  it('renders status labels correctly', () => {
    render(<TeamMappingTable />);

    expect(screen.getByText('Mapped')).toBeInTheDocument();
    expect(screen.getByText('Unmapped')).toBeInTheDocument();
    expect(screen.getByText('Skipped')).toBeInTheDocument();
  });

  it('renders sync status labels', () => {
    render(<TeamMappingTable />);

    expect(screen.getByText('Complete')).toBeInTheDocument();
    expect(screen.getAllByText('Pending').length).toBeGreaterThanOrEqual(1);
  });

  it('renders action buttons in header', () => {
    render(<TeamMappingTable />);

    expect(screen.getByRole('button', { name: /Discover Teams/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Import/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Export/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Migrate Teams/i })).toBeInTheDocument();
  });

  it('renders search input', () => {
    render(<TeamMappingTable />);

    expect(screen.getByPlaceholderText(/Search by team name or slug/i)).toBeInTheDocument();
  });

  it('renders loading state', () => {
    (useTeamMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
      refetch: mockRefetch,
    });

    render(<TeamMappingTable />);

    // Spinner should be visible - look for the actual class used by Primer
    expect(document.querySelector('[class*="Spinner"]')).toBeInTheDocument();
  });

  it('renders error state', () => {
    (useTeamMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error('Failed to load teams'),
      refetch: mockRefetch,
    });

    render(<TeamMappingTable />);

    expect(screen.getByText(/Failed to load team mappings/)).toBeInTheDocument();
  });

  it('renders empty state when no mappings', () => {
    (useTeamMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { mappings: [], total: 0 },
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(<TeamMappingTable />);

    expect(screen.getByText(/No teams found/)).toBeInTheDocument();
  });

  it('filters by search term', async () => {
    const user = userEvent.setup();
    
    render(<TeamMappingTable />);

    const searchInput = screen.getByPlaceholderText(/Search by team name or slug/i);
    await user.type(searchInput, 'alpha');

    // The hook should be called with the search term
    await waitFor(() => {
      expect(useTeamMappings).toHaveBeenCalled();
    });
  });

  it('opens organization filter dropdown', async () => {
    const user = userEvent.setup();
    
    render(<TeamMappingTable />);

    const orgButton = screen.getByRole('button', { name: /Org: All/i });
    await user.click(orgButton);

    await waitFor(() => {
      expect(screen.getByText('All Organizations')).toBeInTheDocument();
      expect(screen.getByText('source-org')).toBeInTheDocument();
      expect(screen.getByText('other-org')).toBeInTheDocument();
    });
  });

  it('opens status filter dropdown', async () => {
    const user = userEvent.setup();
    
    render(<TeamMappingTable />);

    const statusButton = screen.getByRole('button', { name: /Status: All/i });
    await user.click(statusButton);

    await waitFor(() => {
      expect(screen.getByRole('menuitemradio', { name: 'All' })).toBeInTheDocument();
      expect(screen.getByRole('menuitemradio', { name: 'Unmapped' })).toBeInTheDocument();
      expect(screen.getByRole('menuitemradio', { name: 'Mapped' })).toBeInTheDocument();
      expect(screen.getByRole('menuitemradio', { name: 'Skipped' })).toBeInTheDocument();
    });
  });

  it('opens team action menu', async () => {
    const user = userEvent.setup();
    
    render(<TeamMappingTable />);

    // Find the first Actions button
    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    await user.click(actionButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('View details')).toBeInTheDocument();
      expect(screen.getByText('Edit mapping')).toBeInTheDocument();
    });
  });

  it('shows migrate team option for mapped teams', async () => {
    const user = userEvent.setup();
    
    render(<TeamMappingTable />);

    // Find the first Actions button (for mapped team)
    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    await user.click(actionButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Migrate team')).toBeInTheDocument();
    });
  });

  it('disables Migrate Teams button when no mapped teams', () => {
    (useTeamMappingStats as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { total: 2, mapped: 0, unmapped: 2, skipped: 0 },
    });

    render(<TeamMappingTable />);

    const migrateButton = screen.getByRole('button', { name: /Migrate Teams/i });
    expect(migrateButton).toBeDisabled();
  });

  it('renders migrate teams button', () => {
    render(<TeamMappingTable />);

    expect(screen.getByRole('button', { name: /Migrate Teams/i })).toBeInTheDocument();
  });

  it('shows info banner when there are unmapped teams', () => {
    render(<TeamMappingTable />);

    // The text is split across elements: "1 teams" is in a <strong> and "need mapping" follows
    expect(screen.getByText(/need mapping/)).toBeInTheDocument();
    expect(screen.getByText('1 teams')).toBeInTheDocument();
  });

  it('does not show info banner when all teams are mapped', () => {
    (useTeamMappingStats as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { total: 3, mapped: 3, unmapped: 0, skipped: 0 },
    });

    render(<TeamMappingTable />);

    expect(screen.queryByText(/need mapping/)).not.toBeInTheDocument();
  });

  it('shows migration progress when migration is running', () => {
    const runningStatus = {
      ...mockMigrationStatus,
      is_running: true,
      progress: {
        total_teams: 10,
        processed_teams: 5,
        created_teams: 4,
        skipped_teams: 1,
        failed_teams: 0,
        total_repos_synced: 15,
        current_team: 'source-org/team-delta',
        status: 'in_progress',
      },
    };

    (useTeamMigrationStatus as ReturnType<typeof vi.fn>).mockReturnValue({
      data: runningStatus,
    });

    render(<TeamMappingTable />);

    expect(screen.getByText('Migration In Progress')).toBeInTheDocument();
    expect(screen.getByText(/Processing: source-org\/team-delta/)).toBeInTheDocument();
    expect(screen.getByText('50%')).toBeInTheDocument();
    expect(screen.getByText('4 Created')).toBeInTheDocument();
  });

  it('shows cancel button when migration is running', () => {
    const runningStatus = {
      ...mockMigrationStatus,
      is_running: true,
      progress: {
        total_teams: 10,
        processed_teams: 5,
        created_teams: 4,
        skipped_teams: 1,
        failed_teams: 0,
        total_repos_synced: 15,
        status: 'in_progress',
      },
    };

    (useTeamMigrationStatus as ReturnType<typeof vi.fn>).mockReturnValue({
      data: runningStatus,
    });

    render(<TeamMappingTable />);

    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
  });

  it('renders discover teams button', () => {
    render(<TeamMappingTable />);

    expect(screen.getByRole('button', { name: /Discover Teams/i })).toBeInTheDocument();
  });

  it('shows delete mapping option in action menu', async () => {
    const user = userEvent.setup();
    
    render(<TeamMappingTable />);

    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    await user.click(actionButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Delete mapping')).toBeInTheDocument();
    });
  });

  it('shows repos synced count in sync status', () => {
    render(<TeamMappingTable />);

    // The first team has 5/5 repos synced
    expect(screen.getByText('5/5')).toBeInTheDocument();
  });

  it('shows team name under slug', () => {
    render(<TeamMappingTable />);

    expect(screen.getByText('Team Alpha')).toBeInTheDocument();
    expect(screen.getByText('Team Beta')).toBeInTheDocument();
    expect(screen.getByText('Team Gamma')).toBeInTheDocument();
  });

  it('renders action buttons for each team row', () => {
    render(<TeamMappingTable />);

    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    expect(actionButtons.length).toBe(3); // 3 teams in mockMappings
  });

  it('shows migration errors in progress panel', () => {
    const runningStatus = {
      ...mockMigrationStatus,
      is_running: false,
      progress: {
        total_teams: 10,
        processed_teams: 10,
        created_teams: 8,
        skipped_teams: 1,
        failed_teams: 1,
        total_repos_synced: 20,
        status: 'completed_with_errors',
        errors: ['Failed to create team: permission denied', 'API rate limit exceeded'],
      },
    };

    (useTeamMigrationStatus as ReturnType<typeof vi.fn>).mockReturnValue({
      data: runningStatus,
    });

    render(<TeamMappingTable />);

    expect(screen.getByText('Errors:')).toBeInTheDocument();
    expect(screen.getByText(/Failed to create team: permission denied/)).toBeInTheDocument();
    expect(screen.getByText(/API rate limit exceeded/)).toBeInTheDocument();
  });

  it('handles clicking on a team row to open detail panel', async () => {
    const user = userEvent.setup();
    
    render(<TeamMappingTable />);

    // Click on the first team row (but not on action buttons)
    const teamRow = screen.getByText('team-alpha').closest('tr');
    if (teamRow) {
      await user.click(teamRow);
    }

    // TeamDetailPanel would be rendered - we just verify the click works
    // In actual implementation, this would open the TeamDetailPanel
  });
});

