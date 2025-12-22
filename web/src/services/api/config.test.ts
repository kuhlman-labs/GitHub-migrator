import { describe, it, expect, vi, beforeEach } from 'vitest';
import { configApi } from './config';
import { client } from './client';

// Mock the axios client
vi.mock('./client', () => ({
  client: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('configApi', () => {
  const mockClient = client as unknown as {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getConfig', () => {
    it('should fetch application configuration', async () => {
      const mockData = { source_type: 'github', auth_enabled: true };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await configApi.getConfig();

      expect(mockClient.get).toHaveBeenCalledWith('/config');
      expect(result).toEqual(mockData);
    });
  });

  describe('getAuthConfig', () => {
    it('should fetch auth configuration', async () => {
      const mockData = {
        enabled: true,
        login_url: '/auth/login',
        authorization_rules: {
          requires_org_membership: true,
          required_orgs: ['my-org'],
        },
      };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await configApi.getAuthConfig();

      expect(mockClient.get).toHaveBeenCalledWith('/auth/config');
      expect(result).toEqual(mockData);
    });
  });

  describe('getCurrentUser', () => {
    it('should fetch current user info', async () => {
      const mockUser = {
        id: 1,
        login: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        avatar_url: 'https://example.com/avatar.png',
        roles: ['admin'],
      };
      mockClient.get.mockResolvedValue({ data: mockUser });

      const result = await configApi.getCurrentUser();

      expect(mockClient.get).toHaveBeenCalledWith('/auth/user');
      expect(result).toEqual(mockUser);
    });
  });

  describe('logout', () => {
    it('should call logout endpoint', async () => {
      mockClient.post.mockResolvedValue({});

      await configApi.logout();

      expect(mockClient.post).toHaveBeenCalledWith('/auth/logout');
    });
  });

  describe('refreshToken', () => {
    it('should call refresh token endpoint', async () => {
      mockClient.post.mockResolvedValue({});

      await configApi.refreshToken();

      expect(mockClient.post).toHaveBeenCalledWith('/auth/refresh');
    });
  });

  describe('getSetupStatus', () => {
    it('should fetch setup status', async () => {
      const mockData = { is_configured: true, needs_migration: false };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await configApi.getSetupStatus();

      expect(mockClient.get).toHaveBeenCalledWith('/setup/status');
      expect(result).toEqual(mockData);
    });
  });

  describe('validateSourceConnection', () => {
    it('should validate source connection', async () => {
      const sourceConfig = {
        type: 'github' as const,
        base_url: 'https://api.github.com',
        token: 'test-token',
      };
      const mockResult = { valid: true, message: 'Connection successful' };
      mockClient.post.mockResolvedValue({ data: mockResult });

      const result = await configApi.validateSourceConnection(sourceConfig);

      expect(mockClient.post).toHaveBeenCalledWith('/setup/validate-source', sourceConfig);
      expect(result).toEqual(mockResult);
    });
  });

  describe('validateDestinationConnection', () => {
    it('should validate destination connection', async () => {
      const destConfig = {
        base_url: 'https://api.github.com',
        token: 'dest-token',
      };
      const mockResult = { valid: true, message: 'Connection successful' };
      mockClient.post.mockResolvedValue({ data: mockResult });

      const result = await configApi.validateDestinationConnection(destConfig);

      expect(mockClient.post).toHaveBeenCalledWith('/setup/validate-destination', destConfig);
      expect(result).toEqual(mockResult);
    });
  });

  describe('validateDatabaseConnection', () => {
    it('should validate database connection', async () => {
      const dbConfig = {
        type: 'postgres' as const,
        dsn: 'postgres://localhost/db',
      };
      const mockResult = { valid: true, message: 'Database connected' };
      mockClient.post.mockResolvedValue({ data: mockResult });

      const result = await configApi.validateDatabaseConnection(dbConfig);

      expect(mockClient.post).toHaveBeenCalledWith('/setup/validate-database', dbConfig);
      expect(result).toEqual(mockResult);
    });
  });

  describe('applySetup', () => {
    it('should apply setup configuration', async () => {
      const setupConfig = {
        source: {
          type: 'github' as const,
          base_url: 'https://api.github.com',
          token: 'source-token',
        },
        destination: {
          base_url: 'https://api.github.com',
          token: 'dest-token',
        },
        database: {
          type: 'sqlite' as const,
          dsn: './data/migrator.db',
        },
        server: {
          port: 8080,
        },
        migration: {
          workers: 5,
        },
      };
      mockClient.post.mockResolvedValue({ data: { success: true } });

      await configApi.applySetup(setupConfig);

      expect(mockClient.post).toHaveBeenCalledWith('/setup/apply', setupConfig);
    });
  });
});

