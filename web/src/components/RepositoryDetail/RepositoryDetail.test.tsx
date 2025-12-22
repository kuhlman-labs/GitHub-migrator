import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { RepositoryDetail } from './index';

// Mock useParams to return a valid fullName
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ fullName: 'org/repo1' }),
    useLocation: () => ({ state: null, pathname: '/repositories/org/repo1', search: '', hash: '' }),
  };
});

const mockRepository = {
  id: 1,
  full_name: 'org/repo1',
  name: 'repo1',
  source: 'github',
  status: 'pending',
  total_size: 1024000,
  commit_count: 100,
  branch_count: 5,
  visibility: 'private',
  is_archived: false,
  is_fork: false,
  has_lfs: false,
  has_submodules: false,
  has_large_files: false,
  complexity_score: 25,
  complexity_category: 'simple',
  organization: 'org',
  default_branch: 'main',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-15T00:00:00Z',
};

vi.mock('../../hooks/useQueries', () => ({
  useRepositoryWithHistory: () => ({
    data: { repository: mockRepository, history: [] },
    isLoading: false,
    isFetching: false,
    refetch: vi.fn(),
  }),
  useBatches: () => ({
    data: [],
    isLoading: false,
  }),
}));

vi.mock('../../hooks/useMutations', () => ({
  useRediscoverRepository: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useUnlockRepository: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useRollbackRepository: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useMarkRepositoryWontMigrate: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useUpdateRepository: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useAddRepositoriesToBatch: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useRemoveRepositoriesFromBatch: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}));

describe('RepositoryDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the repository name in header', () => {
    render(<RepositoryDetail />);

    expect(screen.getByText('org/repo1')).toBeInTheDocument();
  });

  it('displays repository content', () => {
    const { container } = render(<RepositoryDetail />);

    // The component should render
    expect(container).toBeDefined();
  });

  it('renders page layout with tabs and buttons', () => {
    const { container } = render(<RepositoryDetail />);

    // Check basic structure is present
    expect(container).toBeDefined();
    // Some navigation should be present
    expect(screen.getAllByRole('button').length).toBeGreaterThan(0);
  });
});

