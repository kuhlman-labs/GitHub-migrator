import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '../../__tests__/test-utils';
import { ImportDialog } from './ImportDialog';

// Mock the parseImportFile function
vi.mock('../../utils/import', () => ({
  parseImportFile: vi.fn(),
}));

import { parseImportFile } from '../../utils/import';

describe('ImportDialog', () => {
  const mockOnImport = vi.fn();
  const mockOnCancel = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the dialog with title', () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    expect(screen.getByText('Import Repositories from File')).toBeInTheDocument();
  });

  it('renders file format information', () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    expect(screen.getByText('Required column:')).toBeInTheDocument();
    // Check for the code elements containing the column names
    const codeElements = document.querySelectorAll('code');
    const codeTexts = Array.from(codeElements).map(el => el.textContent);
    expect(codeTexts.some(text => text?.includes('repository'))).toBe(true);
    expect(codeTexts.some(text => text?.includes('full_name'))).toBe(true);
  });

  it('renders Choose File button', () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    expect(screen.getByRole('button', { name: /Choose File/i })).toBeInTheDocument();
  });

  it('disables Import button when no file is selected', () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled();
  });

  it('calls onCancel when Cancel button is clicked', () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(mockOnCancel).toHaveBeenCalled();
  });

  it('shows error when Import is clicked without file', () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    // The Import button should be disabled without a file
    const importButton = screen.getByRole('button', { name: 'Import' });
    expect(importButton).toBeDisabled();
  });

  it('displays selected file name after file selection', async () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['test content'], 'repositories.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText('repositories.csv')).toBeInTheDocument();
    });
  });

  it('shows file size after selection', async () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const content = 'repository\norg/repo1\norg/repo2';
    const file = new File([content], 'repositories.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText(/KB$/)).toBeInTheDocument();
    });
  });

  it('enables Import button after file selection', async () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['test'], 'repositories.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Import' })).not.toBeDisabled();
    });
  });

  it('shows different file icon for CSV', async () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['test'], 'repositories.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText('📄')).toBeInTheDocument();
    });
  });

  it('shows different file icon for Excel', async () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['test'], 'repositories.xlsx', { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText('📊')).toBeInTheDocument();
    });
  });

  it('shows different file icon for JSON', async () => {
    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['[]'], 'repositories.json', { type: 'application/json' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText('📋')).toBeInTheDocument();
    });
  });

  it('calls onImport with parsed result on successful parse', async () => {
    const mockResult = {
      success: true,
      rows: [{ full_name: 'org/repo1' }],
      errors: [],
    };
    (parseImportFile as ReturnType<typeof vi.fn>).mockResolvedValue(mockResult);

    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['test'], 'repositories.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Import' })).not.toBeDisabled();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Import' }));

    await waitFor(() => {
      expect(mockOnImport).toHaveBeenCalledWith(mockResult);
    });
  });

  it('displays error when parse fails', async () => {
    const mockResult = {
      success: false,
      rows: [],
      errors: ['Invalid file format'],
    };
    (parseImportFile as ReturnType<typeof vi.fn>).mockResolvedValue(mockResult);

    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['test'], 'repositories.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Import' }));
    });

    await waitFor(() => {
      expect(screen.getByText('Invalid file format')).toBeInTheDocument();
    });
  });

  it('displays error when no repositories found', async () => {
    const mockResult = {
      success: true,
      rows: [],
      errors: [],
    };
    (parseImportFile as ReturnType<typeof vi.fn>).mockResolvedValue(mockResult);

    render(<ImportDialog onImport={mockOnImport} onCancel={mockOnCancel} />);

    const file = new File(['test'], 'repositories.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Import' }));
    });

    await waitFor(() => {
      expect(screen.getByText('No repositories found in file')).toBeInTheDocument();
    });
  });
});

