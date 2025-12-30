/**
 * MSW request handlers for API mocking in tests.
 */
import { http, HttpResponse } from 'msw';

// Base URL for API requests
const API_BASE = '/api/v1';

// Mock data
export const mockAnalytics = {
  total_repositories: 100,
  migrated_count: 50,
  failed_count: 5,
  in_progress_count: 10,
  pending_count: 35,
  success_rate: 90.9,
  status_breakdown: {
    pending: 35,
    complete: 50,
    failed: 5,
    in_progress: 10,
  },
};

export const mockBatches = [
  {
    id: 1,
    name: 'Batch 1',
    description: 'First migration batch',
    type: 'manual',
    repository_count: 10,
    status: 'pending',
    created_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'Batch 2',
    description: 'Second migration batch',
    type: 'manual',
    repository_count: 5,
    status: 'ready',
    created_at: '2024-01-02T00:00:00Z',
  },
];

export const mockRepositories = [
  {
    id: 1,
    full_name: 'org/repo1',
    name: 'repo1',
    source: 'github',
    status: 'pending',
    total_size: 1024000,
    commit_count: 100,
    branch_count: 5,
    visibility: 'private',
    is_archived: false,
    is_fork: false,
    has_lfs: false,
    has_submodules: false,
    has_large_files: false,
  },
  {
    id: 2,
    full_name: 'org/repo2',
    name: 'repo2',
    source: 'github',
    status: 'complete',
    total_size: 2048000,
    commit_count: 200,
    branch_count: 10,
    visibility: 'public',
    is_archived: false,
    is_fork: true,
    has_lfs: true,
    has_submodules: false,
    has_large_files: false,
  },
];

export const mockOrganizations = [
  { name: 'org1', repository_count: 50 },
  { name: 'org2', repository_count: 30 },
];

export const mockUser = {
  id: 1,
  login: 'testuser',
  name: 'Test User',
  email: 'test@example.com',
  avatar_url: 'https://github.com/testuser.png',
  roles: ['admin'],
};

export const mockConfig = {
  source_type: 'github',
  auth_enabled: true,
};

export const mockAuthConfig = {
  enabled: true,
  login_url: '/auth/github/login',
};

// Request handlers
export const handlers = [
  // Analytics endpoints
  http.get(`${API_BASE}/analytics/summary`, () => {
    return HttpResponse.json(mockAnalytics);
  }),

  // Batch endpoints
  http.get(`${API_BASE}/batches`, () => {
    return HttpResponse.json(mockBatches);
  }),

  http.get(`${API_BASE}/batches/:id`, ({ params }) => {
    const batch = mockBatches.find((b) => b.id === Number(params.id));
    if (batch) {
      return HttpResponse.json(batch);
    }
    return new HttpResponse(null, { status: 404 });
  }),

  // Repository endpoints
  http.get(`${API_BASE}/repositories`, () => {
    return HttpResponse.json({
      repositories: mockRepositories,
      total: mockRepositories.length,
    });
  }),

  http.get(`${API_BASE}/repositories/:id`, ({ params }) => {
    const repo = mockRepositories.find((r) => r.id === Number(params.id));
    if (repo) {
      return HttpResponse.json({ repository: repo, history: [] });
    }
    return new HttpResponse(null, { status: 404 });
  }),

  // Organization endpoints
  http.get(`${API_BASE}/organizations`, () => {
    return HttpResponse.json(mockOrganizations);
  }),

  http.get(`${API_BASE}/organizations/list`, () => {
    return HttpResponse.json(['org1', 'org2']);
  }),

  // Auth endpoints
  http.get(`${API_BASE}/auth/config`, () => {
    return HttpResponse.json(mockAuthConfig);
  }),

  http.get(`${API_BASE}/auth/user`, () => {
    return HttpResponse.json(mockUser);
  }),

  http.post(`${API_BASE}/auth/logout`, () => {
    return new HttpResponse(null, { status: 200 });
  }),

  // Config endpoint
  http.get(`${API_BASE}/config`, () => {
    return HttpResponse.json(mockConfig);
  }),

  // Discovery endpoints
  http.get(`${API_BASE}/discovery/progress`, () => {
    return HttpResponse.json({ status: 'none' });
  }),

  http.get(`${API_BASE}/discovery/status`, () => {
    return HttpResponse.json({ status: 'idle' });
  }),

  // Dashboard action items
  http.get(`${API_BASE}/dashboard/action-items`, () => {
    return HttpResponse.json({
      remediation_required: [],
      failed_migrations: [],
      pending_dry_runs: [],
      total_action_items: 0,
    });
  }),

  // Teams endpoint
  http.get(`${API_BASE}/teams`, () => {
    return HttpResponse.json([]);
  }),
];

