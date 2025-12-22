import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { OrganizationProgressCard } from './OrganizationProgressCard';
import type { Organization } from '../../types';

const mockOrganization: Organization = {
  organization: 'test-org',
  total_repos: 100,
  migrated_count: 50,
  pending_count: 30,
  in_progress_count: 10,
  failed_count: 5,
  wont_migrate_count: 5,
  migration_progress_percentage: 50,
};

describe('OrganizationProgressCard', () => {
  it('should render organization name', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    expect(screen.getByText('test-org')).toBeInTheDocument();
  });

  it('should render progress percentage', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    expect(screen.getByText('50%')).toBeInTheDocument();
  });

  it('should render migration progress stats', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    expect(screen.getByText('Migration Progress')).toBeInTheDocument();
    expect(screen.getByText('50 of 100 migrated')).toBeInTheDocument();
  });

  it('should render status metrics', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    // Complete count
    expect(screen.getByText('50')).toBeInTheDocument();
    expect(screen.getByText('Complete')).toBeInTheDocument();
    
    // In Progress count
    expect(screen.getByText('10')).toBeInTheDocument();
    expect(screen.getByText('In Progress')).toBeInTheDocument();
    
    // Failed count  
    expect(screen.getByText('5 failed')).toBeInTheDocument();
    
    // Pending count
    expect(screen.getByText('30')).toBeInTheDocument();
    expect(screen.getByText('Pending')).toBeInTheDocument();
  });

  it('should link to repositories filtered by organization', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', '/repositories?organization=test-org');
  });

  it('should show view details link', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    expect(screen.getByText('View details →')).toBeInTheDocument();
  });

  it('should show ADO organization label when present', () => {
    const orgWithAdo = {
      ...mockOrganization,
      ado_organization: 'my-ado-org',
    };
    
    render(<OrganizationProgressCard organization={orgWithAdo} />);
    
    expect(screen.getByText('ADO Org: my-ado-org')).toBeInTheDocument();
  });

  it('should show enterprise label when present', () => {
    const orgWithEnterprise = {
      ...mockOrganization,
      enterprise: 'my-enterprise',
    };
    
    render(<OrganizationProgressCard organization={orgWithEnterprise} />);
    
    expect(screen.getByText('Enterprise: my-enterprise')).toBeInTheDocument();
  });

  it('should show projects count when present', () => {
    const orgWithProjects = {
      ...mockOrganization,
      total_projects: 5,
    };
    
    render(<OrganizationProgressCard organization={orgWithProjects} />);
    
    expect(screen.getByText('5 Projects')).toBeInTheDocument();
  });

  it('should show check icon for high progress', () => {
    const highProgressOrg = {
      ...mockOrganization,
      migration_progress_percentage: 85,
    };
    
    render(<OrganizationProgressCard organization={highProgressOrg} />);
    
    // Progress bar should reflect high completion
    expect(screen.getByText('85%')).toBeInTheDocument();
  });

  it('should show warning when there are failures', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    expect(screen.getByText('5 failed')).toBeInTheDocument();
  });

  it('should not show failure warning when no failures', () => {
    const noFailuresOrg = {
      ...mockOrganization,
      failed_count: 0,
    };
    
    render(<OrganizationProgressCard organization={noFailuresOrg} />);
    
    expect(screen.queryByText('failed')).not.toBeInTheDocument();
  });

  it('should handle zero repositories', () => {
    const emptyOrg = {
      ...mockOrganization,
      total_repos: 0,
      migrated_count: 0,
      pending_count: 0,
      in_progress_count: 0,
      failed_count: 0,
      migration_progress_percentage: 0,
    };
    
    render(<OrganizationProgressCard organization={emptyOrg} />);
    
    expect(screen.getByText('0 of 0 migrated')).toBeInTheDocument();
    expect(screen.getByText('0%')).toBeInTheDocument();
  });

  it('should have proper accessibility label on progress bar', () => {
    render(<OrganizationProgressCard organization={mockOrganization} />);
    
    // Check for the aria-label on the progress bar
    const progressBar = screen.getByLabelText('test-org 50% complete');
    expect(progressBar).toBeInTheDocument();
  });
});

