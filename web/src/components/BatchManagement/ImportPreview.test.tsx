import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { ImportPreview, ValidationGroup } from './ImportPreview';
import type { Repository } from '../../types';

const mockValidRepo: Repository = {
  id: 1,
  full_name: 'org/repo1',
  name: 'repo1',
  source: 'github',
  status: 'pending',
  total_size: 1024,
  commit_count: 10,
  branch_count: 2,
  visibility: 'private',
  is_archived: false,
  is_fork: false,
  has_lfs: false,
  has_submodules: false,
  has_large_files: false,
};

const mockAlreadyInBatchRepo: Repository = {
  ...mockValidRepo,
  id: 2,
  full_name: 'org/repo2',
  batch_id: 5,
};

describe('ImportPreview', () => {
  const mockOnConfirm = vi.fn();
  const mockOnCancel = vi.fn();

  const defaultValidationResult: ValidationGroup = {
    valid: [mockValidRepo],
    alreadyInBatch: [],
    notFound: [],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the preview dialog', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText('Import Preview')).toBeInTheDocument();
  });

  it('displays total count of repositories', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText(/Reviewed 1 repositories from file/)).toBeInTheDocument();
  });

  it('displays valid repositories section', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText(/Valid & Available/)).toBeInTheDocument();
    expect(screen.getByText('org/repo1')).toBeInTheDocument();
  });

  it('displays already in batch section when present', () => {
    const validationResult: ValidationGroup = {
      valid: [mockValidRepo],
      alreadyInBatch: [mockAlreadyInBatchRepo],
      notFound: [],
    };

    render(
      <ImportPreview
        validationResult={validationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText(/Already in a Batch/)).toBeInTheDocument();
    expect(screen.getByText('org/repo2')).toBeInTheDocument();
    expect(screen.getByText(/Batch ID: 5/)).toBeInTheDocument();
  });

  it('displays not found section when present', () => {
    const validationResult: ValidationGroup = {
      valid: [],
      alreadyInBatch: [],
      notFound: [{ full_name: 'org/missing-repo' }],
    };

    render(
      <ImportPreview
        validationResult={validationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText(/Not Found/)).toBeInTheDocument();
    expect(screen.getByText('org/missing-repo')).toBeInTheDocument();
    expect(screen.getByText('Repository not found in database')).toBeInTheDocument();
  });

  it('shows empty state when no repositories', () => {
    const validationResult: ValidationGroup = {
      valid: [],
      alreadyInBatch: [],
      notFound: [],
    };

    render(
      <ImportPreview
        validationResult={validationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText('No repositories found in the import file')).toBeInTheDocument();
  });

  it('selects all valid repositories by default', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).toBeChecked();
  });

  it('toggles repository selection when clicking checkbox', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();

    fireEvent.click(checkbox);
    expect(checkbox).toBeChecked();
  });

  it('shows correct selected count', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText('1 of 1 repositories selected')).toBeInTheDocument();
  });

  it('updates selected count when deselecting', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);

    expect(screen.getByText('0 of 1 repositories selected')).toBeInTheDocument();
  });

  it('calls onCancel when Cancel button is clicked', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(mockOnCancel).toHaveBeenCalled();
  });

  it('calls onConfirm with selected repositories', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Add.*Repositories/ }));
    expect(mockOnConfirm).toHaveBeenCalledWith([mockValidRepo]);
  });

  it('disables Add button when no repositories are selected', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);

    expect(screen.getByRole('button', { name: /Add.*Repositories/ })).toBeDisabled();
  });

  it('has Select All/Deselect All toggle', () => {
    render(
      <ImportPreview
        validationResult={defaultValidationResult}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    // When all selected, shows "Deselect All"
    expect(screen.getByText('Deselect All')).toBeInTheDocument();

    // Deselect the repo
    fireEvent.click(screen.getByRole('checkbox'));

    // Now shows "Select All"
    expect(screen.getByText('Select All')).toBeInTheDocument();
  });

  it('toggles all when clicking Select All/Deselect All', () => {
    const multipleValid: ValidationGroup = {
      valid: [
        mockValidRepo,
        { ...mockValidRepo, id: 3, full_name: 'org/repo3' },
      ],
      alreadyInBatch: [],
      notFound: [],
    };

    render(
      <ImportPreview
        validationResult={multipleValid}
        onConfirm={mockOnConfirm}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText('2 of 2 repositories selected')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Deselect All'));

    expect(screen.getByText('0 of 2 repositories selected')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Select All'));

    expect(screen.getByText('2 of 2 repositories selected')).toBeInTheDocument();
  });
});

