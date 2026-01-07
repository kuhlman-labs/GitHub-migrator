import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { BatchRepositoryItem } from './BatchRepositoryItem';
import type { Repository, Batch } from '../../types';

const mockRepository: Repository = {
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
};

const mockBatch: Batch = {
  id: 1,
  name: 'Test Batch',
  description: 'Test',
  type: 'manual',
  repository_count: 10,
  status: 'ready',
  created_at: '2024-01-01T00:00:00Z',
  destination_org: 'dest-org',
};

describe('BatchRepositoryItem', () => {
  it('renders repository full name', () => {
    render(<BatchRepositoryItem repository={mockRepository} />);

    // Repository name appears twice (once as title, once as default destination)
    const repoNames = screen.getAllByText('org/repo1');
    expect(repoNames.length).toBeGreaterThan(0);
  });

  it('renders repository size and branch count', () => {
    render(<BatchRepositoryItem repository={mockRepository} />);

    expect(screen.getByText(/1000 KB/)).toBeInTheDocument();
    expect(screen.getByText(/5 branches/)).toBeInTheDocument();
  });

  it('renders status badge', () => {
    render(<BatchRepositoryItem repository={mockRepository} />);

    // Status badge uses StatusBadge component which renders the label
    expect(screen.getByRole('link')).toBeInTheDocument();
  });

  it('shows default destination label when no custom destination', () => {
    render(<BatchRepositoryItem repository={mockRepository} />);

    expect(screen.getByText('Default')).toBeInTheDocument();
  });

  it('shows custom destination when set differently from source', () => {
    const repoWithDestination: Repository = {
      ...mockRepository,
      destination_full_name: 'other-org/different-name',
    };

    render(<BatchRepositoryItem repository={repoWithDestination} />);

    expect(screen.getByText('other-org/different-name')).toBeInTheDocument();
    expect(screen.getByText('Custom')).toBeInTheDocument();
  });

  it('shows batch default when destination matches batch org', () => {
    const repoWithBatchDestination: Repository = {
      ...mockRepository,
      destination_full_name: 'dest-org/repo1',
    };

    render(
      <BatchRepositoryItem
        repository={repoWithBatchDestination}
        batch={mockBatch}
      />
    );

    expect(screen.getByText('dest-org/repo1')).toBeInTheDocument();
    expect(screen.getByText('Batch Default')).toBeInTheDocument();
  });

  it('shows batch default destination when repo has no custom destination', () => {
    render(
      <BatchRepositoryItem
        repository={mockRepository}
        batch={mockBatch}
      />
    );

    expect(screen.getByText('dest-org/repo1')).toBeInTheDocument();
    expect(screen.getByText('Batch Default')).toBeInTheDocument();
  });

  it('shows retry button for failed migrations', () => {
    const failedRepo: Repository = {
      ...mockRepository,
      status: 'migration_failed',
    };

    render(
      <BatchRepositoryItem
        repository={failedRepo}
        onRetry={vi.fn()}
      />
    );

    expect(screen.getByRole('button', { name: 'Retry Migration' })).toBeInTheDocument();
  });

  it('shows re-run button for dry run failures', () => {
    const failedRepo: Repository = {
      ...mockRepository,
      status: 'dry_run_failed',
    };

    render(
      <BatchRepositoryItem
        repository={failedRepo}
        onRetry={vi.fn()}
      />
    );

    expect(screen.getByRole('button', { name: 'Re-run Dry Run' })).toBeInTheDocument();
  });

  it('calls onRetry when retry button is clicked', () => {
    const mockOnRetry = vi.fn();
    const failedRepo: Repository = {
      ...mockRepository,
      status: 'migration_failed',
    };

    render(
      <BatchRepositoryItem
        repository={failedRepo}
        onRetry={mockOnRetry}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Retry Migration' }));
    expect(mockOnRetry).toHaveBeenCalled();
  });

  it('does not show retry button when onRetry is not provided', () => {
    const failedRepo: Repository = {
      ...mockRepository,
      status: 'migration_failed',
    };

    render(<BatchRepositoryItem repository={failedRepo} />);

    expect(screen.queryByRole('button', { name: 'Retry Migration' })).not.toBeInTheDocument();
  });

  it('does not show retry button for non-failed status', () => {
    render(
      <BatchRepositoryItem
        repository={mockRepository}
        onRetry={vi.fn()}
      />
    );

    expect(screen.queryByRole('button', { name: 'Retry Migration' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Re-run Dry Run' })).not.toBeInTheDocument();
  });

  it('links to repository detail page', () => {
    render(
      <BatchRepositoryItem
        repository={mockRepository}
        batchId={1}
        batchName="Test Batch"
      />
    );

    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', '/repository/org%2Frepo1');
  });

  describe('ADO repository naming', () => {
    const mockADORepository: Repository = {
      ...mockRepository,
      id: 2,
      full_name: 'brettkuhlman/DevOps/Terraform',
      name: 'Terraform',
      source: 'azuredevops',
      ado_project: 'DevOps',
    };

    it('shows batch default with project-repo pattern for ADO repos', () => {
      render(
        <BatchRepositoryItem
          repository={mockADORepository}
          batch={mockBatch}
        />
      );

      // ADO repos should use project-repo pattern: dest-org/DevOps-Terraform
      expect(screen.getByText('dest-org/DevOps-Terraform')).toBeInTheDocument();
      expect(screen.getByText('Batch Default')).toBeInTheDocument();
    });

    it('shows custom destination when set for ADO repo', () => {
      const adoWithCustomDestination: Repository = {
        ...mockADORepository,
        destination_full_name: 'custom-org/my-terraform',
      };

      render(
        <BatchRepositoryItem
          repository={adoWithCustomDestination}
          batch={mockBatch}
        />
      );

      expect(screen.getByText('custom-org/my-terraform')).toBeInTheDocument();
      expect(screen.getByText('Custom')).toBeInTheDocument();
    });

    it('shows batch default when ADO repo destination matches computed default', () => {
      const adoWithBatchDefault: Repository = {
        ...mockADORepository,
        destination_full_name: 'dest-org/DevOps-Terraform',
      };

      render(
        <BatchRepositoryItem
          repository={adoWithBatchDefault}
          batch={mockBatch}
        />
      );

      expect(screen.getByText('dest-org/DevOps-Terraform')).toBeInTheDocument();
      expect(screen.getByText('Batch Default')).toBeInTheDocument();
    });
  });
});

