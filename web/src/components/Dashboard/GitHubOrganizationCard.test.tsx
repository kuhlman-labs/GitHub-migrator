import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { GitHubOrganizationCard } from './GitHubOrganizationCard';
import type { Organization } from '../../types';

const mockOrganization: Organization = {
  organization: 'test-org',
  total_repos: 50,
  migrated_count: 25,
  in_progress_count: 5,
  failed_count: 2,
  pending_count: 18,
  migration_progress_percentage: 50,
};

describe('GitHubOrganizationCard', () => {
  it('renders organization name', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.getByText('test-org')).toBeInTheDocument();
  });

  it('displays total repository count', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.getByText('50')).toBeInTheDocument();
    expect(screen.getByText('Repositories')).toBeInTheDocument();
  });

  it('displays singular "Repository" for single repo', () => {
    const singleRepoOrg: Organization = {
      ...mockOrganization,
      total_repos: 1,
    };

    render(<GitHubOrganizationCard organization={singleRepoOrg} />);

    expect(screen.getByText('Repository')).toBeInTheDocument();
  });

  it('displays migration progress percentage', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.getByText('50%')).toBeInTheDocument();
    expect(screen.getByText('25 of 50 migrated')).toBeInTheDocument();
  });

  it('displays status counts', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.getByText('Complete')).toBeInTheDocument();
    expect(screen.getByText('In Progress')).toBeInTheDocument();
    expect(screen.getByText('Failed')).toBeInTheDocument();
    expect(screen.getByText('Pending')).toBeInTheDocument();
  });

  it('shows failure indicator when there are failures', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.getByText('2 failed')).toBeInTheDocument();
  });

  it('does not show failure indicator when no failures', () => {
    const noFailuresOrg: Organization = {
      ...mockOrganization,
      failed_count: 0,
    };

    render(<GitHubOrganizationCard organization={noFailuresOrg} />);

    expect(screen.queryByText(/failed/)).not.toBeInTheDocument();
  });

  it('displays enterprise label when enterprise is set', () => {
    const enterpriseOrg: Organization = {
      ...mockOrganization,
      enterprise: 'MyEnterprise',
    };

    render(<GitHubOrganizationCard organization={enterpriseOrg} />);

    expect(screen.getByText('Enterprise: MyEnterprise')).toBeInTheDocument();
  });

  it('does not display enterprise label when not set', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.queryByText(/Enterprise:/)).not.toBeInTheDocument();
  });

  it('links to organization repositories', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', '/repositories?organization=test-org');
  });

  it('displays view organization link', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.getByText('View organization →')).toBeInTheDocument();
  });

  it('has progress bar with correct aria-label', () => {
    render(<GitHubOrganizationCard organization={mockOrganization} />);

    expect(screen.getByRole('progressbar')).toHaveAttribute(
      'aria-label',
      'test-org 50% complete'
    );
  });
});

