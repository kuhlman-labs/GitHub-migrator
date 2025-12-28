import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '../../__tests__/test-utils';
import { OrganizationSelector } from './OrganizationSelector';

describe('OrganizationSelector', () => {
  const mockOrganizations = ['org1', 'org2', 'org3', 'test-org'];
  const mockOnChange = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render placeholder when no organizations selected', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('All Organizations')).toBeInTheDocument();
  });

  it('should render custom placeholder', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
        placeholder="Select Teams"
      />
    );

    expect(screen.getByText('Select Teams')).toBeInTheDocument();
  });

  it('should show selected count when organizations are selected', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={['org1', 'org2']}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('2 selected')).toBeInTheDocument();
  });

  it('should show loading state', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
        loading={true}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('should show empty message when no organizations available', () => {
    render(
      <OrganizationSelector
        organizations={[]}
        selectedOrganizations={[]}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(screen.getByText('No organizations found')).toBeInTheDocument();
  });

  it('should show custom empty message', () => {
    render(
      <OrganizationSelector
        organizations={[]}
        selectedOrganizations={[]}
        onChange={mockOnChange}
        emptyMessage="No teams available"
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(screen.getByText('No teams available')).toBeInTheDocument();
  });

  it('should toggle dropdown when button is clicked', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
      />
    );

    // Initially closed
    expect(screen.queryByPlaceholderText('Search organizations...')).not.toBeInTheDocument();

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Now open
    expect(screen.getByPlaceholderText('Search organizations...')).toBeInTheDocument();

    // Close dropdown
    fireEvent.click(button);

    // Now closed
    expect(screen.queryByPlaceholderText('Search organizations...')).not.toBeInTheDocument();
  });

  it('should filter organizations based on search query', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Search for 'test'
    const searchInput = screen.getByPlaceholderText('Search organizations...');
    fireEvent.change(searchInput, { target: { value: 'test' } });

    // Should only show test-org
    expect(screen.getByText('test-org')).toBeInTheDocument();
    expect(screen.queryByText('org1')).not.toBeInTheDocument();
  });

  it('should use custom search placeholder', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
        searchPlaceholder="Find teams..."
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(screen.getByPlaceholderText('Find teams...')).toBeInTheDocument();
  });

  it('should call onChange when organization is toggled', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Click on org1
    const org1Option = screen.getByText('org1');
    fireEvent.click(org1Option);

    expect(mockOnChange).toHaveBeenCalledWith(['org1']);
  });

  it('should remove organization when already selected', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={['org1', 'org2']}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Click on org1 to deselect
    const org1Option = screen.getByText('org1');
    fireEvent.click(org1Option);

    expect(mockOnChange).toHaveBeenCalledWith(['org2']);
  });

  it('should select all visible organizations when Select All is clicked', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Click Select All
    const selectAllButton = screen.getByText('Select All');
    fireEvent.click(selectAllButton);

    expect(mockOnChange).toHaveBeenCalledWith(mockOrganizations);
  });

  it('should clear all selections when Clear All is clicked', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={['org1', 'org2']}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Click Clear All
    const clearAllButton = screen.getByText('Clear All');
    fireEvent.click(clearAllButton);

    expect(mockOnChange).toHaveBeenCalledWith([]);
  });

  it('should not show Clear All button when nothing is selected', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={[]}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(screen.queryByText('Clear All')).not.toBeInTheDocument();
  });

  it('should show checkbox as checked for selected organizations', () => {
    render(
      <OrganizationSelector
        organizations={mockOrganizations}
        selectedOrganizations={['org1']}
        onChange={mockOnChange}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Find org1's checkbox
    const org1Option = screen.getByRole('option', { name: /org1/i });
    const checkbox = org1Option.querySelector('input[type="checkbox"]') as HTMLInputElement;

    expect(checkbox.checked).toBe(true);
  });

  it('should use custom renderLabel for display', () => {
    const customRenderLabel = (value: string) => `Custom: ${value}`;

    render(
      <OrganizationSelector
        organizations={['team1']}
        selectedOrganizations={[]}
        onChange={mockOnChange}
        renderLabel={customRenderLabel}
      />
    );

    // Open dropdown
    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(screen.getByText('Custom: team1')).toBeInTheDocument();
  });

  it('should close dropdown when clicking outside', async () => {
    render(
      <div>
        <button data-testid="outside-button">Outside</button>
        <OrganizationSelector
          organizations={mockOrganizations}
          selectedOrganizations={[]}
          onChange={mockOnChange}
        />
      </div>
    );

    // Open dropdown
    const selectorButton = screen.getByText('All Organizations');
    fireEvent.click(selectorButton);

    // Verify it's open
    expect(screen.getByPlaceholderText('Search organizations...')).toBeInTheDocument();

    // Click outside - use mousedown as that's what the component listens for
    await waitFor(async () => {
      // Wait for the event listener to be added (10ms delay in component)
      await new Promise(resolve => setTimeout(resolve, 20));
      
      const outsideButton = screen.getByTestId('outside-button');
      fireEvent.mouseDown(outsideButton);
    });

    // Should be closed
    await waitFor(() => {
      expect(screen.queryByPlaceholderText('Search organizations...')).not.toBeInTheDocument();
    });
  });
});

