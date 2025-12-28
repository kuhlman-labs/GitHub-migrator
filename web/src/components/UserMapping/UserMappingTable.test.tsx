import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { UserMappingTable } from './UserMappingTable';

// Mock the hooks
vi.mock('../../hooks/useQueries', () => ({
  useUserMappings: vi.fn(),
  useUserMappingStats: vi.fn(),
  useUserMappingSourceOrgs: vi.fn(),
  useUserDetail: vi.fn(),
}));

vi.mock('../../hooks/useMutations', () => ({
  useUpdateUserMapping: vi.fn(),
  useDeleteUserMapping: vi.fn(),
  useFetchMannequins: vi.fn(),
  useSendAttributionInvitation: vi.fn(),
  useBulkSendAttributionInvitations: vi.fn(),
  useDiscoverOrgMembers: vi.fn(),
}));

vi.mock('../../services/api', () => ({
  api: {
    exportUserMappings: vi.fn(),
    importUserMappings: vi.fn(),
    generateGEICSV: vi.fn(),
  },
}));

import {
  useUserMappings,
  useUserMappingStats,
  useUserMappingSourceOrgs,
  useUserDetail,
} from '../../hooks/useQueries';
import {
  useUpdateUserMapping,
  useDeleteUserMapping,
  useFetchMannequins,
  useSendAttributionInvitation,
  useBulkSendAttributionInvitations,
  useDiscoverOrgMembers,
} from '../../hooks/useMutations';

const mockMappings = [
  {
    id: 1,
    login: 'johndoe',
    name: 'John Doe',
    email: 'john@example.com',
    avatar_url: 'https://github.com/johndoe.png',
    source_instance: 'github.example.com',
    source_org: 'org-1',
    destination_login: 'johndoe-new',
    mapping_status: 'mapped',
    mannequin_id: 'mq-123',
    mannequin_login: 'mona-johndoe',
    reclaim_status: 'pending',
    match_confidence: 95,
    match_reason: 'email_exact',
  },
  {
    id: 2,
    login: 'janedoe',
    name: 'Jane Doe',
    email: 'jane@example.com',
    source_instance: 'github.example.com',
    source_org: 'org-1',
    mapping_status: 'unmapped',
  },
  {
    id: 3,
    login: 'bobsmith',
    name: 'Bob Smith',
    email: 'bob@example.com',
    source_instance: 'github.example.com',
    source_org: 'org-2',
    destination_login: 'bob-reclaimed',
    mapping_status: 'reclaimed',
    reclaim_status: 'completed',
  },
  {
    id: 4,
    login: 'skippeduser',
    name: 'Skipped User',
    source_instance: 'github.example.com',
    mapping_status: 'skipped',
  },
];

const mockStats = {
  total: 4,
  mapped: 1,
  unmapped: 1,
  skipped: 1,
  reclaimed: 1,
  pending_reclaim: 1,
  invitable: 1,
};

const mockSourceOrgs = {
  organizations: ['org-1', 'org-2'],
};

describe('UserMappingTable', () => {
  const mockRefetch = vi.fn();
  const mockMutateAsync = vi.fn();
  const mockMutate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    (useUserMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { mappings: mockMappings, total: 4 },
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    (useUserMappingStats as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockStats,
    });

    (useUserMappingSourceOrgs as ReturnType<typeof vi.fn>).mockReturnValue({
      data: mockSourceOrgs,
    });

    (useUpdateUserMapping as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useDeleteUserMapping as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useFetchMannequins as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useSendAttributionInvitation as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useBulkSendAttributionInvitations as ReturnType<typeof vi.fn>).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    });

    (useDiscoverOrgMembers as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: mockMutate,
      isPending: false,
    });

    // Mock useUserDetail for when UserDetailPanel is opened
    (useUserDetail as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
    });
  });

  it('renders the component header', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('User Identity Mapping')).toBeInTheDocument();
    expect(screen.getByText(/Map source identities to destination GitHub users/)).toBeInTheDocument();
  });

  it('renders stats badges', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('4 Total')).toBeInTheDocument();
    expect(screen.getByText('1 Mapped')).toBeInTheDocument();
    expect(screen.getByText('1 Unmapped')).toBeInTheDocument();
    expect(screen.getByText('1 Reclaimed')).toBeInTheDocument();
  });

  it('renders the user mapping table with rows', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('johndoe')).toBeInTheDocument();
    expect(screen.getByText('janedoe')).toBeInTheDocument();
    expect(screen.getByText('bobsmith')).toBeInTheDocument();
    expect(screen.getByText('skippeduser')).toBeInTheDocument();
  });

  it('shows destination login for mapped users', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('johndoe-new')).toBeInTheDocument();
  });

  it('shows "Not mapped" for unmapped users', () => {
    render(<UserMappingTable />);

    expect(screen.getAllByText('Not mapped').length).toBeGreaterThanOrEqual(1);
  });

  it('renders status labels correctly', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('Mapped')).toBeInTheDocument();
    expect(screen.getByText('Unmapped')).toBeInTheDocument();
    expect(screen.getByText('Reclaimed')).toBeInTheDocument();
    expect(screen.getByText('Skipped')).toBeInTheDocument();
  });

  it('renders mannequin login when present', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('mona-johndoe')).toBeInTheDocument();
  });

  it('renders match confidence when present', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('95%')).toBeInTheDocument();
  });

  it('renders action buttons in header', () => {
    render(<UserMappingTable />);

    expect(screen.getByRole('button', { name: /Discover Org Members/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Fetch Mannequins/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Import/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Export/i })).toBeInTheDocument();
  });

  it('renders search input', () => {
    render(<UserMappingTable />);

    expect(screen.getByPlaceholderText(/Search by login, email, or name/i)).toBeInTheDocument();
  });

  it('renders loading state', () => {
    (useUserMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
      refetch: mockRefetch,
    });

    render(<UserMappingTable />);

    expect(document.querySelector('[class*="Spinner"]')).toBeInTheDocument();
  });

  it('renders error state', () => {
    (useUserMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error('Failed to load users'),
      refetch: mockRefetch,
    });

    render(<UserMappingTable />);

    expect(screen.getByText(/Failed to load user mappings/)).toBeInTheDocument();
  });

  it('renders empty state when no mappings', () => {
    (useUserMappings as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { mappings: [], total: 0 },
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(<UserMappingTable />);

    expect(screen.getByText(/No users found/)).toBeInTheDocument();
  });

  it('filters by search term', async () => {
    const user = userEvent.setup();

    render(<UserMappingTable />);

    const searchInput = screen.getByPlaceholderText(/Search by login, email, or name/i);
    await user.type(searchInput, 'john');

    await waitFor(() => {
      expect(useUserMappings).toHaveBeenCalled();
    });
  });

  it('renders organization filter button', () => {
    render(<UserMappingTable />);

    // The filter button should be present when sourceOrgs are available
    expect(screen.getByRole('button', { name: /Org: All/i })).toBeInTheDocument();
  });

  it('opens status filter dropdown', async () => {
    const user = userEvent.setup();

    render(<UserMappingTable />);

    const statusButton = screen.getByRole('button', { name: /Status: All/i });
    await user.click(statusButton);

    await waitFor(() => {
      expect(screen.getByRole('menuitemradio', { name: 'All' })).toBeInTheDocument();
      expect(screen.getByRole('menuitemradio', { name: 'Unmapped' })).toBeInTheDocument();
      expect(screen.getByRole('menuitemradio', { name: 'Mapped' })).toBeInTheDocument();
      expect(screen.getByRole('menuitemradio', { name: 'Reclaimed' })).toBeInTheDocument();
      expect(screen.getByRole('menuitemradio', { name: 'Skipped' })).toBeInTheDocument();
    });
  });

  it('opens user action menu', async () => {
    const user = userEvent.setup();

    render(<UserMappingTable />);

    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    await user.click(actionButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Edit mapping')).toBeInTheDocument();
      expect(screen.getByText('Delete mapping')).toBeInTheDocument();
    });
  });

  it('shows skip user option for non-skipped users', async () => {
    const user = userEvent.setup();

    render(<UserMappingTable />);

    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    await user.click(actionButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Skip user')).toBeInTheDocument();
    });
  });

  it('shows send invitation button when there are invitable users', () => {
    render(<UserMappingTable />);

    expect(screen.getByRole('button', { name: /Send 1 Invitation/i })).toBeInTheDocument();
  });

  it('does not show send invitation button when no invitable users', () => {
    (useUserMappingStats as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { ...mockStats, invitable: 0 },
    });

    render(<UserMappingTable />);

    expect(screen.queryByRole('button', { name: /Send.*Invitation/i })).not.toBeInTheDocument();
  });

  it('renders fetch mannequins button', () => {
    render(<UserMappingTable />);

    expect(screen.getByRole('button', { name: /Fetch Mannequins/i })).toBeInTheDocument();
  });

  it('renders discover org members button', () => {
    render(<UserMappingTable />);

    expect(screen.getByRole('button', { name: /Discover Org Members/i })).toBeInTheDocument();
  });

  it('shows delete mapping option in action menu', async () => {
    const user = userEvent.setup();

    render(<UserMappingTable />);

    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    await user.click(actionButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Delete mapping')).toBeInTheDocument();
    });
  });

  it('shows user email when present', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('john@example.com')).toBeInTheDocument();
    expect(screen.getByText('jane@example.com')).toBeInTheDocument();
  });

  it('shows source org when present', () => {
    render(<UserMappingTable />);

    // Multiple users have org-1
    const orgElements = screen.getAllByText('org-1');
    expect(orgElements.length).toBeGreaterThanOrEqual(1);
  });

  it('renders reclaim status badge when present', () => {
    render(<UserMappingTable />);

    expect(screen.getByText('pending')).toBeInTheDocument();
    expect(screen.getByText('completed')).toBeInTheDocument();
  });

  it('renders export button', () => {
    render(<UserMappingTable />);

    expect(screen.getByRole('button', { name: /Export/i })).toBeInTheDocument();
  });

  it('renders action buttons for each user row', () => {
    render(<UserMappingTable />);

    // There should be action buttons for each user
    const actionButtons = screen.getAllByRole('button', { name: /Actions/i });
    expect(actionButtons.length).toBe(4); // 4 users in mockMappings
  });

});

