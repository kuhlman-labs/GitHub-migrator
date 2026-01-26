import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { client, setAuthEnabled } from './client';

describe('API client', () => {
  describe('configuration', () => {
    it('should have correct baseURL', () => {
      expect(client.defaults.baseURL).toBe('/api/v1');
    });

    it('should have correct timeout', () => {
      expect(client.defaults.timeout).toBe(120000);
    });

    it('should send credentials with requests', () => {
      expect(client.defaults.withCredentials).toBe(true);
    });
  });

  describe('401 interceptor', () => {
    const originalLocation = window.location;

    beforeEach(() => {
      // Reset mocks
      vi.clearAllMocks();
      
      // Enable auth for tests (unless explicitly disabled in test)
      setAuthEnabled(true);
      
      // Mock window.location
      delete (window as { location?: Location }).location;
      window.location = {
        ...originalLocation,
        href: '',
        pathname: '/dashboard',
      } as Location;
    });

    afterEach(() => {
      window.location = originalLocation;
      // Reset authEnabled state
      setAuthEnabled(false);
    });

    it('should redirect to login on 401 error', async () => {
      const error = {
        response: { status: 401 },
      };

      // Get the response interceptor
      const interceptors = client.interceptors.response as unknown as {
        handlers: Array<{ rejected?: (error: unknown) => unknown }>;
      };
      const rejectedHandler = interceptors.handlers[0]?.rejected;

      if (rejectedHandler) {
        try {
          await rejectedHandler(error);
        } catch {
          // Expected to reject
        }
        expect(window.location.href).toBe('/login');
      }
    });

    it('should not redirect when already on login page', async () => {
      window.location = {
        ...originalLocation,
        href: 'http://localhost/login',
        pathname: '/login',
      } as Location;

      const error = {
        response: { status: 401 },
      };

      const interceptors = client.interceptors.response as unknown as {
        handlers: Array<{ rejected?: (error: unknown) => unknown }>;
      };
      const rejectedHandler = interceptors.handlers[0]?.rejected;

      if (rejectedHandler) {
        try {
          await rejectedHandler(error);
        } catch {
          // Expected to reject
        }
        // Should not change href when already on login
        expect(window.location.href).toBe('http://localhost/login');
      }
    });

    it('should not redirect when on auth endpoints', async () => {
      window.location = {
        ...originalLocation,
        href: 'http://localhost/auth/callback',
        pathname: '/auth/callback',
      } as Location;

      const error = {
        response: { status: 401 },
      };

      const interceptors = client.interceptors.response as unknown as {
        handlers: Array<{ rejected?: (error: unknown) => unknown }>;
      };
      const rejectedHandler = interceptors.handlers[0]?.rejected;

      if (rejectedHandler) {
        try {
          await rejectedHandler(error);
        } catch {
          // Expected to reject
        }
        expect(window.location.href).toBe('http://localhost/auth/callback');
      }
    });

    it('should reject the error after handling', async () => {
      const error = {
        response: { status: 401 },
      };

      const interceptors = client.interceptors.response as unknown as {
        handlers: Array<{ rejected?: (error: unknown) => unknown }>;
      };
      const rejectedHandler = interceptors.handlers[0]?.rejected;

      if (rejectedHandler) {
        await expect(rejectedHandler(error)).rejects.toEqual(error);
      }
    });

    it('should pass through non-401 errors', async () => {
      const error = {
        response: { status: 500 },
      };

      const interceptors = client.interceptors.response as unknown as {
        handlers: Array<{ rejected?: (error: unknown) => unknown }>;
      };
      const rejectedHandler = interceptors.handlers[0]?.rejected;

      if (rejectedHandler) {
        await expect(rejectedHandler(error)).rejects.toEqual(error);
        // Should not redirect for 500 errors
        expect(window.location.href).not.toBe('/login');
      }
    });

    it('should not redirect on 401 when auth is disabled', async () => {
      // Disable auth for this test
      setAuthEnabled(false);

      const error = {
        response: { status: 401 },
      };

      const interceptors = client.interceptors.response as unknown as {
        handlers: Array<{ rejected?: (error: unknown) => unknown }>;
      };
      const rejectedHandler = interceptors.handlers[0]?.rejected;

      if (rejectedHandler) {
        try {
          await rejectedHandler(error);
        } catch {
          // Expected to reject
        }
        // Should not redirect when auth is disabled
        expect(window.location.href).toBe('');
      }
    });
  });
});

