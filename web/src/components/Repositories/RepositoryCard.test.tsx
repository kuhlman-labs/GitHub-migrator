import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { RepositoryCard } from './RepositoryCard';
import type { Repository } from '../../types';

const createRepository = (overrides: Partial<Repository> = {}): Repository => ({
  id: 1,
  full_name: 'my-org/my-repo',
  name: 'my-repo',
  source: 'github',
  status: 'pending',
  total_size: 1024000,
  commit_count: 100,
  has_lfs: false,
  has_submodules: false,
  has_large_files: false,
  branch_count: 5,
  visibility: 'private',
  is_archived: false,
  is_fork: false,
  created_at: '2023-01-01T00:00:00Z',
  updated_at: '2024-01-15T10:30:00Z',
  ...overrides,
});

describe('RepositoryCard', () => {
  describe('basic rendering', () => {
    it('should render repository name', () => {
      render(<RepositoryCard repository={createRepository()} />);

      expect(screen.getByText('my-org/my-repo')).toBeInTheDocument();
    });

    it('should render status badge', () => {
      render(<RepositoryCard repository={createRepository({ status: 'pending' })} />);

      expect(screen.getByText('pending')).toBeInTheDocument();
    });

    it('should render size', () => {
      render(<RepositoryCard repository={createRepository({ total_size: 1024000 })} />);

      expect(screen.getByText(/Size:/)).toBeInTheDocument();
    });

    it('should render branch count', () => {
      render(<RepositoryCard repository={createRepository({ branch_count: 5 })} />);

      expect(screen.getByText('Branches: 5')).toBeInTheDocument();
    });

    it('should link to repository detail page', () => {
      render(<RepositoryCard repository={createRepository()} />);

      const link = screen.getByRole('link');
      expect(link).toHaveAttribute('href', '/repository/my-org%2Fmy-repo');
    });
  });

  describe('ADO repository display', () => {
    it('should display ADO project and repo name for ADO repos', () => {
      render(
        <RepositoryCard
          repository={createRepository({
            full_name: 'ado-org/MyProject/my-repo',
            ado_project: 'MyProject',
          })}
        />
      );

      expect(screen.getByText('MyProject/my-repo')).toBeInTheDocument();
      expect(screen.getByText('ado-org')).toBeInTheDocument();
    });
  });

  describe('feature badges', () => {
    it('should render Archived badge when archived', () => {
      render(<RepositoryCard repository={createRepository({ is_archived: true })} />);

      expect(screen.getByText('Archived')).toBeInTheDocument();
    });

    it('should render Fork badge when fork', () => {
      render(<RepositoryCard repository={createRepository({ is_fork: true })} />);

      expect(screen.getByText('Fork')).toBeInTheDocument();
    });

    it('should render LFS badge when has LFS', () => {
      render(<RepositoryCard repository={createRepository({ has_lfs: true })} />);

      expect(screen.getByText('LFS')).toBeInTheDocument();
    });

    it('should render Submodules badge when has submodules', () => {
      render(<RepositoryCard repository={createRepository({ has_submodules: true })} />);

      expect(screen.getByText('Submodules')).toBeInTheDocument();
    });

    it('should render Large Files badge when has large files', () => {
      render(<RepositoryCard repository={createRepository({ has_large_files: true })} />);

      expect(screen.getByText('Large Files')).toBeInTheDocument();
    });

    it('should render Actions badge when has actions', () => {
      render(<RepositoryCard repository={createRepository({ has_actions: true })} />);

      expect(screen.getByText('Actions')).toBeInTheDocument();
    });

    it('should render Packages badge when has packages', () => {
      render(<RepositoryCard repository={createRepository({ has_packages: true })} />);

      expect(screen.getByText('Packages')).toBeInTheDocument();
    });

    it('should render Protected badge when has branch protections', () => {
      render(<RepositoryCard repository={createRepository({ branch_protections: 2 })} />);

      expect(screen.getByText('Protected')).toBeInTheDocument();
    });

    it('should render Public badge for public repos', () => {
      render(<RepositoryCard repository={createRepository({ visibility: 'public' })} />);

      expect(screen.getByText('Public')).toBeInTheDocument();
    });

    it('should render Internal badge for internal repos', () => {
      render(<RepositoryCard repository={createRepository({ visibility: 'internal' })} />);

      expect(screen.getByText('Internal')).toBeInTheDocument();
    });

    it('should not render visibility badge for private repos', () => {
      render(<RepositoryCard repository={createRepository({ visibility: 'private' })} />);

      expect(screen.queryByText('Private')).not.toBeInTheDocument();
    });
  });

  describe('selection mode', () => {
    it('should render checkbox in selection mode', () => {
      render(
        <RepositoryCard
          repository={createRepository()}
          selectionMode={true}
          onToggleSelect={() => {}}
        />
      );

      expect(screen.getByRole('checkbox')).toBeInTheDocument();
    });

    it('should show checkbox as checked when selected', () => {
      render(
        <RepositoryCard
          repository={createRepository()}
          selectionMode={true}
          selected={true}
          onToggleSelect={() => {}}
        />
      );

      expect(screen.getByRole('checkbox')).toBeChecked();
    });

    it('should call onToggleSelect when checkbox is changed', () => {
      const onToggleSelect = vi.fn();
      render(
        <RepositoryCard
          repository={createRepository({ id: 123 })}
          selectionMode={true}
          onToggleSelect={onToggleSelect}
        />
      );

      fireEvent.click(screen.getByRole('checkbox'));
      expect(onToggleSelect).toHaveBeenCalledWith(123);
    });

    it('should call onToggleSelect when card is clicked in selection mode', () => {
      const onToggleSelect = vi.fn();
      render(
        <RepositoryCard
          repository={createRepository({ id: 456 })}
          selectionMode={true}
          onToggleSelect={onToggleSelect}
        />
      );

      // Click the card container, not the checkbox
      const card = screen.getByText('my-org/my-repo').closest('div');
      fireEvent.click(card!);

      expect(onToggleSelect).toHaveBeenCalledWith(456);
    });

    it('should not render as link in selection mode', () => {
      render(
        <RepositoryCard
          repository={createRepository()}
          selectionMode={true}
          onToggleSelect={() => {}}
        />
      );

      expect(screen.queryByRole('link')).not.toBeInTheDocument();
    });
  });

  describe('timestamps', () => {
    it('should render discovery timestamp when available', () => {
      render(
        <RepositoryCard
          repository={createRepository({
            last_discovery_at: '2024-01-15T10:00:00Z',
          })}
        />
      );

      expect(screen.getByText('Discovered:')).toBeInTheDocument();
    });

    it('should render dry run timestamp when available', () => {
      render(
        <RepositoryCard
          repository={createRepository({
            last_dry_run_at: '2024-01-16T14:00:00Z',
          })}
        />
      );

      expect(screen.getByText('Dry run:')).toBeInTheDocument();
    });
  });
});

