import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { BatchMetadataForm } from './BatchMetadataForm';

describe('BatchMetadataForm', () => {
  const mockSetBatchName = vi.fn();
  const mockSetBatchDescription = vi.fn();
  const mockSetScheduledAt = vi.fn();
  const mockOnMigrationSettingsChange = vi.fn();
  const mockSetShowMigrationSettings = vi.fn();
  const mockOnSave = vi.fn();
  const mockOnClose = vi.fn();

  const defaultProps = {
    batchName: '',
    setBatchName: mockSetBatchName,
    batchDescription: '',
    setBatchDescription: mockSetBatchDescription,
    scheduledAt: '',
    setScheduledAt: mockSetScheduledAt,
    migrationSettings: {
      destinationOrg: '',
      migrationAPI: 'GEI' as const,
      excludeReleases: false,
      excludeAttachments: false,
    },
    onMigrationSettingsChange: mockOnMigrationSettingsChange,
    showMigrationSettings: false,
    setShowMigrationSettings: mockSetShowMigrationSettings,
    organizations: ['org1', 'org2'],
    loading: false,
    isEditMode: false,
    currentBatchReposCount: 5,
    error: null,
    onSave: mockOnSave,
    onClose: mockOnClose,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render batch name input', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    expect(screen.getByPlaceholderText('e.g., Wave 1, Q1 Migration')).toBeInTheDocument();
  });

  it('should render batch description textarea', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    expect(screen.getByPlaceholderText('Optional description')).toBeInTheDocument();
  });

  it('should call setBatchName when name input changes', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    const input = screen.getByPlaceholderText('e.g., Wave 1, Q1 Migration');
    fireEvent.change(input, { target: { value: 'New Batch' } });

    expect(mockSetBatchName).toHaveBeenCalledWith('New Batch');
  });

  it('should call setBatchDescription when description changes', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    const textarea = screen.getByPlaceholderText('Optional description');
    fireEvent.change(textarea, { target: { value: 'A new description' } });

    expect(mockSetBatchDescription).toHaveBeenCalledWith('A new description');
  });

  it('should render Migration Settings toggle button', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    expect(screen.getByText('Migration Settings')).toBeInTheDocument();
  });

  it('should toggle migration settings when button is clicked', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    const toggleButton = screen.getByText('Migration Settings').closest('button')!;
    fireEvent.click(toggleButton);

    expect(mockSetShowMigrationSettings).toHaveBeenCalledWith(true);
  });

  it('should show migration settings when showMigrationSettings is true', () => {
    render(<BatchMetadataForm {...defaultProps} showMigrationSettings={true} />);

    expect(screen.getByText('Destination Organization')).toBeInTheDocument();
    expect(screen.getByText('Migration API')).toBeInTheDocument();
    expect(screen.getByText('Exclude Releases')).toBeInTheDocument();
    expect(screen.getByText('Exclude Attachments')).toBeInTheDocument();
  });

  it('should call onMigrationSettingsChange when destination org changes', () => {
    render(<BatchMetadataForm {...defaultProps} showMigrationSettings={true} />);

    const input = screen.getByPlaceholderText('Leave blank to use source org');
    fireEvent.change(input, { target: { value: 'new-org' } });

    expect(mockOnMigrationSettingsChange).toHaveBeenCalledWith({ destinationOrg: 'new-org' });
  });

  it('should call onMigrationSettingsChange when excludeReleases is toggled', () => {
    render(<BatchMetadataForm {...defaultProps} showMigrationSettings={true} />);

    const checkbox = screen.getByRole('checkbox', { name: /Exclude Releases/i });
    fireEvent.click(checkbox);

    expect(mockOnMigrationSettingsChange).toHaveBeenCalledWith({ excludeReleases: true });
  });

  it('should render scheduled date input', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    expect(screen.getByText('Scheduled Date (Optional)')).toBeInTheDocument();
  });

  it('should render Cancel button', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('should call onClose when Cancel is clicked', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    fireEvent.click(screen.getByText('Cancel'));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('should render Create Batch button when not in edit mode', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    expect(screen.getByText('Create Batch')).toBeInTheDocument();
  });

  it('should render Update Batch button when in edit mode', () => {
    render(<BatchMetadataForm {...defaultProps} isEditMode={true} />);

    expect(screen.getByText('Update Batch')).toBeInTheDocument();
  });

  it('should render Create & Start button when not in edit mode', () => {
    render(<BatchMetadataForm {...defaultProps} />);

    expect(screen.getByText('Create & Start')).toBeInTheDocument();
  });

  it('should not render Create & Start button when in edit mode', () => {
    render(<BatchMetadataForm {...defaultProps} isEditMode={true} />);

    expect(screen.queryByText('Create & Start')).not.toBeInTheDocument();
  });

  it('should call onSave(false) when Create Batch is clicked', () => {
    render(<BatchMetadataForm {...defaultProps} batchName="Test" />);

    fireEvent.click(screen.getByText('Create Batch'));
    expect(mockOnSave).toHaveBeenCalledWith(false);
  });

  it('should call onSave(true) when Create & Start is clicked', () => {
    render(<BatchMetadataForm {...defaultProps} batchName="Test" />);

    fireEvent.click(screen.getByText('Create & Start'));
    expect(mockOnSave).toHaveBeenCalledWith(true);
  });

  it('should disable buttons when loading', () => {
    render(<BatchMetadataForm {...defaultProps} batchName="Test" loading={true} />);

    expect(screen.getByText('Saving...')).toBeInTheDocument();
    expect(screen.getByText('Starting...')).toBeInTheDocument();
  });

  it('should disable buttons when batch name is empty', () => {
    render(<BatchMetadataForm {...defaultProps} batchName="" />);

    const createButton = screen.getByRole('button', { name: 'Create Batch' });
    expect(createButton).toBeDisabled();
  });

  it('should disable buttons when no repos in batch', () => {
    render(<BatchMetadataForm {...defaultProps} batchName="Test" currentBatchReposCount={0} />);

    const createButton = screen.getByRole('button', { name: 'Create Batch' });
    expect(createButton).toBeDisabled();
  });

  it('should display error message when error is set', () => {
    render(<BatchMetadataForm {...defaultProps} error="Something went wrong" />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('should show configured count badge when settings are configured', () => {
    const settings = {
      destinationOrg: 'my-org',
      migrationAPI: 'GEI' as const,
      excludeReleases: true,
      excludeAttachments: false,
    };

    render(<BatchMetadataForm {...defaultProps} migrationSettings={settings} />);

    expect(screen.getByText('2 configured')).toBeInTheDocument();
  });
});

