import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { DiscoveryModal, DiscoveryModalProps } from './DiscoveryModal';

describe('DiscoveryModal', () => {
  const mockOnStart = vi.fn();
  const mockOnClose = vi.fn();
  const mockSetDiscoveryType = vi.fn();
  const mockSetOrganization = vi.fn();
  const mockSetEnterpriseSlug = vi.fn();
  const mockSetAdoOrganization = vi.fn();
  const mockSetAdoProject = vi.fn();

  const defaultProps: DiscoveryModalProps = {
    isOpen: true,
    sourceType: 'github',
    discoveryType: 'organization',
    setDiscoveryType: mockSetDiscoveryType,
    organization: '',
    setOrganization: mockSetOrganization,
    enterpriseSlug: '',
    setEnterpriseSlug: mockSetEnterpriseSlug,
    adoOrganization: '',
    setAdoOrganization: mockSetAdoOrganization,
    adoProject: '',
    setAdoProject: mockSetAdoProject,
    loading: false,
    error: null,
    onStart: mockOnStart,
    onClose: mockOnClose,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders nothing when not open', () => {
    render(<DiscoveryModal {...defaultProps} isOpen={false} />);

    expect(screen.queryByText('Start Repository Discovery')).not.toBeInTheDocument();
  });

  it('renders modal when open', () => {
    render(<DiscoveryModal {...defaultProps} />);

    expect(screen.getByText('Start Repository Discovery')).toBeInTheDocument();
  });

  it('shows GitHub discovery types for github source', () => {
    render(<DiscoveryModal {...defaultProps} />);

    expect(screen.getByRole('button', { name: 'Organization' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Enterprise' })).toBeInTheDocument();
  });

  it('shows Azure DevOps discovery types for azuredevops source', () => {
    render(<DiscoveryModal {...defaultProps} sourceType="azuredevops" discoveryType="ado-org" />);

    expect(screen.getAllByRole('button', { name: 'Organization' })).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Project' })).toBeInTheDocument();
  });

  it('renders organization input for organization discovery type', () => {
    render(<DiscoveryModal {...defaultProps} discoveryType="organization" />);

    expect(screen.getByLabelText(/Organization Name/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g., your-github-org')).toBeInTheDocument();
  });

  it('renders enterprise slug input for enterprise discovery type', () => {
    render(<DiscoveryModal {...defaultProps} discoveryType="enterprise" />);

    expect(screen.getByLabelText(/Enterprise Slug/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g., your-enterprise-slug')).toBeInTheDocument();
  });

  it('renders ADO organization input for ado-org discovery type', () => {
    render(<DiscoveryModal {...defaultProps} sourceType="azuredevops" discoveryType="ado-org" />);

    expect(screen.getByLabelText(/Azure DevOps Organization/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g., your-ado-org')).toBeInTheDocument();
  });

  it('renders ADO organization and project inputs for ado-project discovery type', () => {
    render(<DiscoveryModal {...defaultProps} sourceType="azuredevops" discoveryType="ado-project" />);

    expect(screen.getByLabelText(/Azure DevOps Organization/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Project Name/i)).toBeInTheDocument();
  });

  it('calls setDiscoveryType when clicking discovery type buttons', () => {
    render(<DiscoveryModal {...defaultProps} />);

    fireEvent.click(screen.getByRole('button', { name: 'Enterprise' }));
    expect(mockSetDiscoveryType).toHaveBeenCalledWith('enterprise');
  });

  it('calls setOrganization when typing in organization input', () => {
    render(<DiscoveryModal {...defaultProps} />);

    fireEvent.change(screen.getByPlaceholderText('e.g., your-github-org'), {
      target: { value: 'my-org' },
    });
    expect(mockSetOrganization).toHaveBeenCalledWith('my-org');
  });

  it('calls onClose when clicking Cancel button', () => {
    render(<DiscoveryModal {...defaultProps} />);

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('calls onClose when clicking backdrop', () => {
    render(<DiscoveryModal {...defaultProps} />);

    const backdrop = document.querySelector('.fixed.inset-0.bg-black\\/50');
    fireEvent.click(backdrop!);
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('calls onClose when clicking close button', () => {
    render(<DiscoveryModal {...defaultProps} />);

    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('disables Start Discovery button when form is invalid', () => {
    render(<DiscoveryModal {...defaultProps} organization="" />);

    expect(screen.getByRole('button', { name: 'Start Discovery' })).toBeDisabled();
  });

  it('enables Start Discovery button when form is valid', () => {
    render(<DiscoveryModal {...defaultProps} organization="my-org" />);

    expect(screen.getByRole('button', { name: 'Start Discovery' })).not.toBeDisabled();
  });

  it('calls onStart when form is submitted', () => {
    render(<DiscoveryModal {...defaultProps} organization="my-org" />);

    fireEvent.click(screen.getByRole('button', { name: 'Start Discovery' }));
    expect(mockOnStart).toHaveBeenCalled();
  });

  it('shows loading state when loading', () => {
    render(<DiscoveryModal {...defaultProps} loading={true} organization="my-org" />);

    expect(screen.getByRole('button', { name: 'Loading...' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Loading...' })).toBeDisabled();
  });

  it('disables all inputs when loading', () => {
    render(<DiscoveryModal {...defaultProps} loading={true} />);

    expect(screen.getByPlaceholderText('e.g., your-github-org')).toBeDisabled();
  });

  it('displays error message when error is present', () => {
    render(<DiscoveryModal {...defaultProps} error="Failed to start discovery" />);

    expect(screen.getByText('Failed to start discovery')).toBeInTheDocument();
  });

  it('validates enterprise form correctly', () => {
    render(<DiscoveryModal {...defaultProps} discoveryType="enterprise" enterpriseSlug="my-enterprise" />);

    expect(screen.getByRole('button', { name: 'Start Discovery' })).not.toBeDisabled();
  });

  it('validates ado-org form correctly', () => {
    render(
      <DiscoveryModal
        {...defaultProps}
        sourceType="azuredevops"
        discoveryType="ado-org"
        adoOrganization="my-ado-org"
      />
    );

    expect(screen.getByRole('button', { name: 'Start Discovery' })).not.toBeDisabled();
  });

  it('validates ado-project form correctly', () => {
    render(
      <DiscoveryModal
        {...defaultProps}
        sourceType="azuredevops"
        discoveryType="ado-project"
        adoOrganization="my-ado-org"
        adoProject="my-project"
      />
    );

    expect(screen.getByRole('button', { name: 'Start Discovery' })).not.toBeDisabled();
  });

  it('invalidates ado-project form when project is missing', () => {
    render(
      <DiscoveryModal
        {...defaultProps}
        sourceType="azuredevops"
        discoveryType="ado-project"
        adoOrganization="my-ado-org"
        adoProject=""
      />
    );

    expect(screen.getByRole('button', { name: 'Start Discovery' })).toBeDisabled();
  });
});

