import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import { Dependencies } from './index';
import { api } from '../../services/api';

// Mock the API
vi.mock('../../services/api', () => ({
  api: {
    getDependencyGraph: vi.fn(),
  },
}));

describe('Dependencies', () => {
  const mockDependencyData = {
    nodes: [
      { id: 'org/repo1', label: 'org/repo1' },
      { id: 'org/repo2', label: 'org/repo2' },
    ],
    edges: [
      { source: 'org/repo1', target: 'org/repo2', type: 'package' },
    ],
    total_repos: 2,
    total_dependencies: 1,
    dependency_types: { package: 1 },
    organization_summary: [
      { organization: 'org', total_repos: 2, repos_with_deps: 1, total_deps: 1 },
    ],
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (api.getDependencyGraph as ReturnType<typeof vi.fn>).mockResolvedValue(mockDependencyData);
  });

  it('should show loading spinner initially', () => {
    render(<Dependencies />);
    
    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
  });

  it('should fetch dependency data', async () => {
    render(<Dependencies />);
    
    await waitFor(() => {
      expect(api.getDependencyGraph).toHaveBeenCalled();
    });
  });

  it('should show error message on API failure', async () => {
    (api.getDependencyGraph as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'));
    
    render(<Dependencies />);
    
    await waitFor(() => {
      expect(screen.getByText('API Error')).toBeInTheDocument();
    });
  });

  it('should call API with filter when type changes', async () => {
    render(<Dependencies />);
    
    await waitFor(() => {
      expect(api.getDependencyGraph).toHaveBeenCalledTimes(1);
    });
  });
});
