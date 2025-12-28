import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { RepositoryGroup } from './RepositoryGroup';
import type { Repository } from '../../types';

const mockRepositories: Repository[] = [
  {
    id: 1,
    full_name: 'org/repo1',
    name: 'repo1',
    source: 'github',
    status: 'pending',
    total_size: 1024000,
    complexity_score: 25,
    complexity_category: 'simple',
  },
  {
    id: 2,
    full_name: 'org/repo2',
    name: 'repo2',
    source: 'github',
    status: 'pending',
    total_size: 2048000,
    complexity_score: 55,
    complexity_category: 'medium',
  },
] as Repository[];

describe('RepositoryGroup', () => {
  const defaultProps = {
    organization: 'test-org',
    repositories: mockRepositories,
    selectedIds: new Set<number>(),
    onToggle: vi.fn(),
    onToggleAll: vi.fn(),
  };

  it('renders the group header', () => {
    render(<RepositoryGroup {...defaultProps} />);

    expect(screen.getByText('test-org')).toBeInTheDocument();
  });

  it('shows content', () => {
    const { container } = render(<RepositoryGroup {...defaultProps} />);

    expect(container.textContent?.length || 0).toBeGreaterThan(0);
  });

  it('renders all repositories in the group', () => {
    render(<RepositoryGroup {...defaultProps} />);

    expect(screen.getByText('org/repo1')).toBeInTheDocument();
    expect(screen.getByText('org/repo2')).toBeInTheDocument();
  });
});

