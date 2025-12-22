import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { ADOOrganizationCard } from './ADOOrganizationCard';
import type { Organization } from '../../types';

const mockProjects: Organization[] = [
  {
    organization: 'project1',
    ado_organization: 'ado-org',
    total_repos: 30,
    migrated_count: 15,
    in_progress_count: 3,
    failed_count: 2,
    pending_count: 10,
    migration_progress_percentage: 50,
  },
  {
    organization: 'project2',
    ado_organization: 'ado-org',
    total_repos: 20,
    migrated_count: 18,
    in_progress_count: 1,
    failed_count: 0,
    pending_count: 1,
    migration_progress_percentage: 90,
  },
];

describe('ADOOrganizationCard', () => {
  it('renders ADO organization name', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    expect(screen.getByText('ado-org')).toBeInTheDocument();
  });

  it('displays aggregate statistics', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    // Total migrated: 15 + 18 = 33
    expect(screen.getByText('33')).toBeInTheDocument();
    // Overall progress section
    expect(screen.getByText('Overall Progress')).toBeInTheDocument();
  });

  it('displays project count in subheader', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    // 33 of 50 repos across 2 projects
    expect(screen.getByText(/2 projects/)).toBeInTheDocument();
  });

  it('shows failure indicator when there are failures', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    // Total failed: 2 + 0 = 2 (may appear multiple times in aggregate and project level)
    const failedIndicators = screen.getAllByText('2 failed');
    expect(failedIndicators.length).toBeGreaterThanOrEqual(1);
  });

  it('displays projects section header', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    expect(screen.getByText('PROJECTS (2)')).toBeInTheDocument();
  });

  it('displays project names', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    expect(screen.getByText('project1')).toBeInTheDocument();
    expect(screen.getByText('project2')).toBeInTheDocument();
  });

  it('can collapse and expand projects', async () => {
    const user = userEvent.setup();
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    // Initially expanded
    expect(screen.getByText('PROJECTS (2)')).toBeInTheDocument();

    // Click to collapse
    const collapseButton = screen.getByRole('button', { name: /Collapse projects/i });
    await user.click(collapseButton);

    // Projects section should be hidden
    expect(screen.queryByText('PROJECTS (2)')).not.toBeInTheDocument();

    // Click to expand
    const expandButton = screen.getByRole('button', { name: /Expand projects/i });
    await user.click(expandButton);

    // Projects section should be visible again
    expect(screen.getByText('PROJECTS (2)')).toBeInTheDocument();
  });

  it('shows progress bar with correct aria-label', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    // Overall progress: 33/50 = 66%
    expect(screen.getByRole('progressbar', { name: 'ado-org 66% complete' })).toBeInTheDocument();
  });

  it('displays status breakdown', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    // Multiple elements may have these texts
    expect(screen.getAllByText('Complete').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('In Progress').length).toBeGreaterThanOrEqual(1);
  });

  it('handles single project correctly', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={[mockProjects[0]]} />);

    expect(screen.getByText(/1 project$/)).toBeInTheDocument();
    expect(screen.getByText('PROJECTS (1)')).toBeInTheDocument();
  });

  it('displays view project link for each project', () => {
    render(<ADOOrganizationCard adoOrgName="ado-org" projects={mockProjects} />);

    const viewLinks = screen.getAllByText('View project →');
    expect(viewLinks.length).toBe(2);
  });
});

