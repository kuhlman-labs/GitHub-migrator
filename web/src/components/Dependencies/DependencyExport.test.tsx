import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '../../__tests__/test-utils';
import { DependencyExport } from './DependencyExport';
import type { DependencyGraphNode, DependencyGraphEdge } from '../../types';

// Mock the API
vi.mock('../../services/api', () => ({
  api: {
    exportDependencies: vi.fn(),
  },
}));

import { api } from '../../services/api';

const mockNodes: DependencyGraphNode[] = [
  { id: 'org/repo1', full_name: 'org/repo1', organization: 'org', status: 'pending', depends_on_count: 1, depended_by_count: 0 },
  { id: 'org/repo2', full_name: 'org/repo2', organization: 'org', status: 'complete', depends_on_count: 0, depended_by_count: 1 },
];

const mockEdges: DependencyGraphEdge[] = [
  { source: 'org/repo1', target: 'org/repo2', type: 'submodule' },
];

describe('DependencyExport', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders export button', () => {
    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={false}
        hasFilteredData={true}
      />
    );

    expect(screen.getByRole('button', { name: /Export/i })).toBeInTheDocument();
  });

  it('disables button when no filtered data', () => {
    render(
      <DependencyExport
        filteredNodes={[]}
        filteredEdges={[]}
        hasActiveFilters={false}
        hasFilteredData={false}
      />
    );

    expect(screen.getByRole('button', { name: /Export/i })).toBeDisabled();
  });

  it('shows export menu when clicked', () => {
    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={false}
        hasFilteredData={true}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Export/i }));

    expect(screen.getByText('Summary')).toBeInTheDocument();
    expect(screen.getByText('All Dependencies')).toBeInTheDocument();
    expect(screen.getByText('Export Summary as CSV')).toBeInTheDocument();
    expect(screen.getByText('Export Summary as JSON')).toBeInTheDocument();
    expect(screen.getByText('Export All as CSV')).toBeInTheDocument();
    expect(screen.getByText('Export All as JSON')).toBeInTheDocument();
  });

  it('shows repo count when filters are active', () => {
    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={true}
        hasFilteredData={true}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Export/i }));

    expect(screen.getByText(/Summary \(2 repos\)/)).toBeInTheDocument();
  });

  it('closes menu when clicking backdrop', () => {
    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={false}
        hasFilteredData={true}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Export/i }));
    expect(screen.getByText('Summary')).toBeInTheDocument();

    // Click the backdrop
    const backdrop = document.querySelector('.fixed.inset-0');
    fireEvent.click(backdrop!);

    expect(screen.queryByText('Export Summary as CSV')).not.toBeInTheDocument();
  });

  it('exports all as CSV when button clicked', async () => {
    const mockBlob = new Blob(['test'], { type: 'text/csv' });
    (api.exportDependencies as ReturnType<typeof vi.fn>).mockResolvedValue(mockBlob);

    // Mock URL methods
    const mockCreateObjectURL = vi.fn().mockReturnValue('blob:test');
    const mockRevokeObjectURL = vi.fn();
    global.URL.createObjectURL = mockCreateObjectURL;
    global.URL.revokeObjectURL = mockRevokeObjectURL;

    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={false}
        hasFilteredData={true}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Export/i }));
    fireEvent.click(screen.getByText('Export All as CSV'));

    await waitFor(() => {
      expect(api.exportDependencies).toHaveBeenCalledWith('csv');
    });
  });

  it('exports all as JSON when button clicked', async () => {
    const mockBlob = new Blob(['{}'], { type: 'application/json' });
    (api.exportDependencies as ReturnType<typeof vi.fn>).mockResolvedValue(mockBlob);

    global.URL.createObjectURL = vi.fn().mockReturnValue('blob:test');
    global.URL.revokeObjectURL = vi.fn();

    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={false}
        hasFilteredData={true}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Export/i }));
    fireEvent.click(screen.getByText('Export All as JSON'));

    await waitFor(() => {
      expect(api.exportDependencies).toHaveBeenCalledWith('json');
    });
  });

  it('exports filtered summary as CSV', () => {
    global.URL.createObjectURL = vi.fn().mockReturnValue('blob:test');
    global.URL.revokeObjectURL = vi.fn();

    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={true}
        hasFilteredData={true}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Export/i }));
    fireEvent.click(screen.getByText('Export Summary as CSV'));

    expect(global.URL.createObjectURL).toHaveBeenCalled();
  });

  it('exports filtered summary as JSON', () => {
    global.URL.createObjectURL = vi.fn().mockReturnValue('blob:test');
    global.URL.revokeObjectURL = vi.fn();

    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={true}
        hasFilteredData={true}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Export/i }));
    fireEvent.click(screen.getByText('Export Summary as JSON'));

    expect(global.URL.createObjectURL).toHaveBeenCalled();
  });

  it('toggles menu open and closed', () => {
    render(
      <DependencyExport
        filteredNodes={mockNodes}
        filteredEdges={mockEdges}
        hasActiveFilters={false}
        hasFilteredData={true}
      />
    );

    const exportButton = screen.getByRole('button', { name: /Export/i });

    // Open menu
    fireEvent.click(exportButton);
    expect(screen.getByText('Summary')).toBeInTheDocument();

    // Close menu by clicking button again
    fireEvent.click(exportButton);
    expect(screen.queryByText('Export Summary as CSV')).not.toBeInTheDocument();
  });
});

