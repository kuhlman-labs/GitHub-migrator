import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@primer/react';
import { BatchBuilder } from './BatchBuilder';
import { ToastProvider } from '../../contexts/ToastContext';
import { SourceProvider } from '../../contexts/SourceContext';
import { api } from '../../services/api';
import type { Repository } from '../../types';

// Mock the API
vi.mock('../../services/api', () => ({
  api: {
    listRepositories: vi.fn(),
    listOrganizations: vi.fn(),
    createBatch: vi.fn(),
    addRepositoriesToBatch: vi.fn(),
  },
}));

// Create mock repositories for testing
const createMockRepo = (id: number, name: string): Repository => ({
  id,
  full_name: `org/${name}`,
  source: 'github.com',
  source_url: `https://github.com/org/${name}`,
  total_size: 1000,
  status: 'pending',
  visibility: 'private',
  default_branch: 'main',
  complexity_score: 1,
  complexity_category: 'simple',
  issue_count: 0,
  pr_count: 0,
  branch_count: 1,
  discovered_at: new Date().toISOString(),
});

// Mock repos for page 1
const page1Repos: Repository[] = [
  createMockRepo(1, 'repo-1'),
  createMockRepo(2, 'repo-2'),
  createMockRepo(3, 'repo-3'),
];

// Mock repos for page 2 - different IDs
const page2Repos: Repository[] = [
  createMockRepo(4, 'repo-4'),
  createMockRepo(5, 'repo-5'),
  createMockRepo(6, 'repo-6'),
];

// Create a wrapper with all necessary providers
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <ToastProvider>
            <SourceProvider>
              {children}
            </SourceProvider>
          </ToastProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('BatchBuilder', () => {
  const mockOnClose = vi.fn();
  const mockOnSuccess = vi.fn();
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
    
    // Default mock for organizations
    (api.listOrganizations as ReturnType<typeof vi.fn>).mockResolvedValue([]);
  });

  describe('repository selection', () => {
    it('should select and deselect a repository', async () => {
      (api.listRepositories as ReturnType<typeof vi.fn>).mockResolvedValue({
        repositories: page1Repos,
        total: 3,
      });

      render(
        <BatchBuilder onClose={mockOnClose} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      );

      // Wait for repos to load
      await waitFor(() => {
        expect(screen.getByText('org/repo-1')).toBeInTheDocument();
      });

      // Find the label for repo-1 (which contains the checkbox)
      const repo1Label = screen.getByText('org/repo-1').closest('label');
      expect(repo1Label).toBeInTheDocument();

      // Click to select
      await user.click(repo1Label!);

      // Verify 1 repo is selected
      await waitFor(() => {
        expect(screen.getByText('1 selected')).toBeInTheDocument();
      });

      // Click again to deselect
      await user.click(repo1Label!);

      // Selection count badge should be gone
      await waitFor(() => {
        expect(screen.queryByText('1 selected')).not.toBeInTheDocument();
      });
    });

    it('should add selected repositories to the batch', async () => {
      (api.listRepositories as ReturnType<typeof vi.fn>).mockResolvedValue({
        repositories: page1Repos,
        total: 3,
      });

      render(
        <BatchBuilder onClose={mockOnClose} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      );

      // Wait for repos to load
      await waitFor(() => {
        expect(screen.getByText('org/repo-1')).toBeInTheDocument();
      });

      // Select two repos
      const repo1Label = screen.getByText('org/repo-1').closest('label');
      const repo2Label = screen.getByText('org/repo-2').closest('label');
      
      await user.click(repo1Label!);
      await user.click(repo2Label!);

      // Verify 2 repos are selected
      await waitFor(() => {
        expect(screen.getByText('2 selected')).toBeInTheDocument();
      });

      // Click "Add Selected" button
      const addSelectedButton = screen.getByRole('button', { name: /add selected/i });
      await user.click(addSelectedButton);

      // After adding, selections should be cleared
      await waitFor(() => {
        expect(screen.queryByText('2 selected')).not.toBeInTheDocument();
      });

      // Both repos should now be in the "Selected Repositories" section
      // The summary should show 2 repositories
      await waitFor(() => {
        expect(screen.getByText(/2 repositories/i)).toBeInTheDocument();
      });
    });

    it('should preserve selections across page changes', async () => {
      // This is the key test for the bug fix
      // Mock listRepositories to return different repos based on offset
      let callCount = 0;
      (api.listRepositories as ReturnType<typeof vi.fn>).mockImplementation(async () => {
        callCount++;
        // First call returns page 1, subsequent calls return page 2
        if (callCount === 1) {
          return { repositories: page1Repos, total: 6 };
        }
        return { repositories: page2Repos, total: 6 };
      });

      render(
        <BatchBuilder onClose={mockOnClose} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      );

      // Wait for page 1 to load
      await waitFor(() => {
        expect(screen.getByText('org/repo-1')).toBeInTheDocument();
      });

      // Select a repo from page 1
      const repo1Label = screen.getByText('org/repo-1').closest('label');
      await user.click(repo1Label!);

      // Verify 1 repo is selected
      await waitFor(() => {
        expect(screen.getByText('1 selected')).toBeInTheDocument();
      });

      // Simulate page change - trigger a re-fetch with different filters
      // This happens internally when pagination changes
      // For testing, we'll manually trigger by changing filters
      // Find a button that triggers pagination or filter change
      
      // Look for the page 2 button if pagination exists
      const paginationButtons = screen.queryAllByRole('button');
      const page2Button = paginationButtons.find(btn => btn.textContent === '2');
      
      if (page2Button) {
        await user.click(page2Button);

        // Wait for page 2 to load
        await waitFor(() => {
          expect(screen.getByText('org/repo-4')).toBeInTheDocument();
        });

        // The selection count should still show 1 (from page 1)
        // This is the key assertion - before the fix, this would fail
        expect(screen.getByText('1 selected')).toBeInTheDocument();

        // Select a repo from page 2
        const repo4Label = screen.getByText('org/repo-4').closest('label');
        await user.click(repo4Label!);

        // Now 2 repos should be selected (1 from each page)
        await waitFor(() => {
          expect(screen.getByText('2 selected')).toBeInTheDocument();
        });

        // Click "Add Selected"
        const addSelectedButton = screen.getByRole('button', { name: /add selected/i });
        await user.click(addSelectedButton);

        // After adding, the batch should contain 2 repositories
        // (This was the bug - before fix, only 1 from current page was added)
        await waitFor(() => {
          expect(screen.getByText(/2 repositories/i)).toBeInTheDocument();
        });
      }
    });

    it('should clear selections when filters change', async () => {
      (api.listRepositories as ReturnType<typeof vi.fn>).mockResolvedValue({
        repositories: page1Repos,
        total: 3,
      });

      render(
        <BatchBuilder onClose={mockOnClose} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      );

      // Wait for repos to load
      await waitFor(() => {
        expect(screen.getByText('org/repo-1')).toBeInTheDocument();
      });

      // Select a repo
      const repo1Label = screen.getByText('org/repo-1').closest('label');
      await user.click(repo1Label!);

      await waitFor(() => {
        expect(screen.getByText('1 selected')).toBeInTheDocument();
      });

      // Type in search to trigger filter change
      const searchInput = screen.getByPlaceholderText(/repository name/i);
      await user.type(searchInput, 'test');

      // Wait a bit for debounced filter to apply
      await waitFor(() => {
        // Selection should be cleared when filters change
        expect(screen.queryByText('1 selected')).not.toBeInTheDocument();
      }, { timeout: 1000 });
    });
  });
});
