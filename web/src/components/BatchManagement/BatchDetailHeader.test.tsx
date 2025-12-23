import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { BatchDetailHeader } from './BatchDetailHeader';
import type { Batch, Repository } from '../../types';

describe('BatchDetailHeader', () => {
  const mockOnEdit = vi.fn();
  const mockOnDelete = vi.fn();
  const mockOnDryRun = vi.fn();
  const mockOnStart = vi.fn();
  const mockOnRetryFailed = vi.fn();

  const baseBatch: Batch = {
    id: 1,
    name: 'Test Batch',
    description: 'A test batch description',
    status: 'pending',
    repository_count: 5,
    created_at: '2024-01-01T10:00:00Z',
  };

  const baseRepositories: Repository[] = [
    { id: 1, name: 'repo1', full_name: 'org/repo1', status: 'pending' } as Repository,
    { id: 2, name: 'repo2', full_name: 'org/repo2', status: 'pending' } as Repository,
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render batch name', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('Test Batch')).toBeInTheDocument();
  });

  it('should render batch description', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('A test batch description')).toBeInTheDocument();
  });

  it('should render repository count', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('5 repositories')).toBeInTheDocument();
  });

  it('should render Edit and Delete buttons', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('Edit')).toBeInTheDocument();
    expect(screen.getByText('Delete')).toBeInTheDocument();
  });

  it('should call onEdit when Edit button is clicked', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    fireEvent.click(screen.getByText('Edit'));
    expect(mockOnEdit).toHaveBeenCalledWith(baseBatch);
  });

  it('should call onDelete when Delete button is clicked', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    fireEvent.click(screen.getByText('Delete'));
    expect(mockOnDelete).toHaveBeenCalledWith(baseBatch);
  });

  it('should show Dry Run button when there are pending repos', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('Dry Run')).toBeInTheDocument();
  });

  it('should show Start Migration button for ready batch', () => {
    const readyBatch = { ...baseBatch, status: 'ready' as const };

    render(
      <BatchDetailHeader
        batch={readyBatch}
        batchRepositories={[
          { id: 1, name: 'repo1', full_name: 'org/repo1', status: 'dry_run_complete' } as Repository,
        ]}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('Start Migration')).toBeInTheDocument();
  });

  it('should call onStart when Start Migration is clicked for ready batch', () => {
    const readyBatch = { ...baseBatch, status: 'ready' as const };

    render(
      <BatchDetailHeader
        batch={readyBatch}
        batchRepositories={[
          { id: 1, name: 'repo1', full_name: 'org/repo1', status: 'dry_run_complete' } as Repository,
        ]}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    fireEvent.click(screen.getByText('Start Migration'));
    expect(mockOnStart).toHaveBeenCalledWith(1);
  });

  it('should show Retry Failed button when there are failed repos', () => {
    const failedRepos: Repository[] = [
      { id: 1, name: 'repo1', full_name: 'org/repo1', status: 'migration_failed' } as Repository,
      { id: 2, name: 'repo2', full_name: 'org/repo2', status: 'dry_run_failed' } as Repository,
    ];

    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={failedRepos}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('Retry Failed (2)')).toBeInTheDocument();
  });

  it('should call onRetryFailed when Retry Failed is clicked', () => {
    const failedRepos: Repository[] = [
      { id: 1, name: 'repo1', full_name: 'org/repo1', status: 'migration_failed' } as Repository,
    ];

    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={failedRepos}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    fireEvent.click(screen.getByText('Retry Failed (1)'));
    expect(mockOnRetryFailed).toHaveBeenCalledTimes(1);
  });

  it('should show destination org in Migration Settings when set', () => {
    const batchWithDestination = { ...baseBatch, destination_org: 'target-org' };

    render(
      <BatchDetailHeader
        batch={batchWithDestination}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('Migration Settings')).toBeInTheDocument();
    expect(screen.getByText('Default Destination:')).toBeInTheDocument();
    expect(screen.getByText('target-org')).toBeInTheDocument();
  });

  it('should show Schedule & Timeline section', () => {
    render(
      <BatchDetailHeader
        batch={baseBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    expect(screen.getByText('Timeline')).toBeInTheDocument();
  });

  it('should not show action bar when batch is in progress', () => {
    const inProgressBatch = { ...baseBatch, status: 'in_progress' as const };

    render(
      <BatchDetailHeader
        batch={inProgressBatch}
        batchRepositories={baseRepositories}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    // Dry Run and other action buttons should not be visible when in progress
    expect(screen.queryByText('Dry Run')).not.toBeInTheDocument();
  });

  it('should not show action bar when batch is complete', () => {
    const completeBatch = { ...baseBatch, status: 'complete' as const };

    render(
      <BatchDetailHeader
        batch={completeBatch}
        batchRepositories={[]}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
        onDryRun={mockOnDryRun}
        onStart={mockOnStart}
        onRetryFailed={mockOnRetryFailed}
      />
    );

    // Action buttons should not be visible when complete
    expect(screen.queryByText('Dry Run')).not.toBeInTheDocument();
    expect(screen.queryByText('Start Migration')).not.toBeInTheDocument();
  });
});

