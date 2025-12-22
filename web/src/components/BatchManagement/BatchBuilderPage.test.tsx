import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@primer/react';
import { BatchBuilderPage } from './BatchBuilderPage';
import { ToastProvider } from '../../contexts/ToastContext';
import { api } from '../../services/api';

// Mock the API
vi.mock('../../services/api', () => ({
  api: {
    getBatch: vi.fn(),
  },
}));

// Mock BatchBuilder to simplify testing
vi.mock('./BatchBuilder', () => ({
  BatchBuilder: ({ batch, onClose, onSuccess }: { batch?: unknown; onClose: () => void; onSuccess: () => void }) => (
    <div data-testid="batch-builder">
      <div>Batch Builder Mock</div>
      {batch && <div data-testid="batch-data">Has batch data</div>}
      <button onClick={onClose}>Close</button>
      <button onClick={onSuccess}>Success</button>
    </div>
  ),
}));

// Create a wrapper with custom initial route
function createWrapper(initialEntries: string[] = ['/batches/new']) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter initialEntries={initialEntries}>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <ToastProvider>
            <Routes>
              <Route path="/batches/new" element={children} />
              <Route path="/batches/:batchId/edit" element={children} />
              <Route path="/batches" element={<div data-testid="batches-page">Batches Page</div>} />
            </Routes>
          </ToastProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('BatchBuilderPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render create new batch page', () => {
    render(<BatchBuilderPage />, { wrapper: createWrapper() });

    expect(screen.getByText('Create New Batch')).toBeInTheDocument();
    expect(screen.getByText('Select repositories and configure your migration batch')).toBeInTheDocument();
    expect(screen.getByTestId('batch-builder')).toBeInTheDocument();
  });

  it('should show loading spinner while loading batch', async () => {
    (api.getBatch as ReturnType<typeof vi.fn>).mockImplementationOnce(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );

    render(<BatchBuilderPage />, { wrapper: createWrapper(['/batches/1/edit']) });

    // Should show loading state initially
    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
  });

  it('should show error state when batch fails to load', async () => {
    (api.getBatch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error('Network error'));

    render(<BatchBuilderPage />, { wrapper: createWrapper(['/batches/1/edit']) });

    await waitFor(() => {
      expect(screen.getByText('Error')).toBeInTheDocument();
      expect(screen.getByText('Failed to load batch. Please try again.')).toBeInTheDocument();
    });

    expect(screen.getByText('Back to Batches')).toBeInTheDocument();
  });

  it('should have a cancel button', () => {
    render(<BatchBuilderPage />, { wrapper: createWrapper() });

    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('should render BatchBuilder component', () => {
    render(<BatchBuilderPage />, { wrapper: createWrapper() });

    expect(screen.getByTestId('batch-builder')).toBeInTheDocument();
    expect(screen.getByText('Batch Builder Mock')).toBeInTheDocument();
  });
});
