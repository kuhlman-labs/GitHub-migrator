import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { RepositoryListItem } from './RepositoryListItem';
import type { Repository } from '../../types';

const mockRepository: Repository = {
  id: 1,
  full_name: 'org/repo1',
  name: 'repo1',
  source: 'github',
  status: 'pending',
  total_size: 1024000,
  complexity_score: 25,
  complexity_category: 'simple',
  organization: 'org',
  commit_count: 100,
  branch_count: 5,
  is_archived: false,
  is_fork: false,
  has_lfs: false,
  has_submodules: false,
} as Repository;

describe('RepositoryListItem', () => {
  it('renders the repository name', () => {
    render(
      <RepositoryListItem
        repository={mockRepository}
        selected={false}
        onToggle={vi.fn()}
      />
    );

    expect(screen.getByText('org/repo1')).toBeInTheDocument();
  });

  it('renders checkbox for selection', () => {
    render(
      <RepositoryListItem
        repository={mockRepository}
        selected={true}
        onToggle={vi.fn()}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).toBeInTheDocument();
    expect(checkbox).toBeChecked();
  });

  it('calls onToggle when clicked', async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    
    render(
      <RepositoryListItem
        repository={mockRepository}
        selected={false}
        onToggle={onToggle}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    await user.click(checkbox);

    expect(onToggle).toHaveBeenCalledWith(1);
  });

  it('is disabled when disabled prop is true', () => {
    render(
      <RepositoryListItem
        repository={mockRepository}
        selected={false}
        disabled={true}
        onToggle={vi.fn()}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).toBeDisabled();
  });
});

