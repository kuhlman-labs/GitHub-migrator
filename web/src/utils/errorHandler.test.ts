import { describe, it, expect, vi } from 'vitest';
import {
  extractErrorMessage,
  handleApiError,
  createErrorHandler,
  withErrorHandler,
} from './errorHandler';

// Helper to create mock Axios errors
function createAxiosError(
  status?: number,
  data?: { error?: string; message?: string },
  code?: string
) {
  return {
    isAxiosError: true,
    response: status
      ? {
          status,
          data,
        }
      : undefined,
    code,
    message: 'Request failed',
  };
}

describe('extractErrorMessage', () => {
  describe('Axios errors with API response', () => {
    it('should extract error field from API response', () => {
      const error = createAxiosError(400, { error: 'Invalid request data' });
      expect(extractErrorMessage(error)).toBe('Invalid request data');
    });

    it('should extract message field from API response', () => {
      const error = createAxiosError(400, { message: 'Something went wrong' });
      expect(extractErrorMessage(error)).toBe('Something went wrong');
    });

    it('should prefer error over message field', () => {
      const error = createAxiosError(400, {
        error: 'Primary error',
        message: 'Secondary message',
      });
      expect(extractErrorMessage(error)).toBe('Primary error');
    });
  });

  describe('Axios errors with HTTP status codes', () => {
    it('should handle 401 Unauthorized', () => {
      const error = createAxiosError(401);
      expect(extractErrorMessage(error)).toBe('Unauthorized. Please log in again.');
    });

    it('should handle 403 Forbidden', () => {
      const error = createAxiosError(403);
      expect(extractErrorMessage(error)).toBe(
        'You do not have permission to perform this action.'
      );
    });

    it('should handle 404 Not Found', () => {
      const error = createAxiosError(404);
      expect(extractErrorMessage(error)).toBe('The requested resource was not found.');
    });

    it('should handle 422 Unprocessable Entity', () => {
      const error = createAxiosError(422);
      expect(extractErrorMessage(error)).toBe('The request contains invalid data.');
    });

    it('should handle 429 Too Many Requests', () => {
      const error = createAxiosError(429);
      expect(extractErrorMessage(error)).toBe('Too many requests. Please try again later.');
    });

    it('should handle 500 Internal Server Error', () => {
      const error = createAxiosError(500);
      expect(extractErrorMessage(error)).toBe(
        'An internal server error occurred. Please try again later.'
      );
    });

    it('should handle 502 Bad Gateway', () => {
      const error = createAxiosError(502);
      expect(extractErrorMessage(error)).toBe(
        'The server is temporarily unavailable. Please try again later.'
      );
    });

    it('should handle 503 Service Unavailable', () => {
      const error = createAxiosError(503);
      expect(extractErrorMessage(error)).toBe(
        'The server is temporarily unavailable. Please try again later.'
      );
    });
  });

  describe('Axios errors with error codes', () => {
    it('should handle network error', () => {
      const error = createAxiosError(undefined, undefined, 'ERR_NETWORK');
      expect(extractErrorMessage(error)).toBe(
        'Network error. Please check your connection and try again.'
      );
    });

    it('should handle timeout error', () => {
      const error = createAxiosError(undefined, undefined, 'ECONNABORTED');
      expect(extractErrorMessage(error)).toBe('Request timed out. Please try again.');
    });
  });

  describe('Standard Error objects', () => {
    it('should extract message from Error', () => {
      const error = new Error('Something went wrong');
      expect(extractErrorMessage(error)).toBe('Something went wrong');
    });

    it('should extract message from TypeError', () => {
      const error = new TypeError('Type mismatch');
      expect(extractErrorMessage(error)).toBe('Type mismatch');
    });
  });

  describe('String errors', () => {
    it('should return string errors as-is', () => {
      expect(extractErrorMessage('Something went wrong')).toBe('Something went wrong');
    });
  });

  describe('Unknown error types', () => {
    it('should return fallback for null', () => {
      expect(extractErrorMessage(null)).toBe('An unexpected error occurred. Please try again.');
    });

    it('should return fallback for undefined', () => {
      expect(extractErrorMessage(undefined)).toBe(
        'An unexpected error occurred. Please try again.'
      );
    });

    it('should return fallback for objects without message', () => {
      expect(extractErrorMessage({ foo: 'bar' })).toBe(
        'An unexpected error occurred. Please try again.'
      );
    });
  });
});

describe('handleApiError', () => {
  it('should call showToast with extracted message', () => {
    const showToast = vi.fn();
    const error = new Error('Test error');

    handleApiError(error, showToast);

    expect(showToast).toHaveBeenCalledWith('Test error');
  });

  it('should prepend context to message', () => {
    const showToast = vi.fn();
    const error = new Error('Test error');

    handleApiError(error, showToast, 'Failed to save');

    expect(showToast).toHaveBeenCalledWith('Failed to save: Test error');
  });

  it('should handle Axios errors', () => {
    const showToast = vi.fn();
    const error = createAxiosError(400, { error: 'Invalid data' });

    handleApiError(error, showToast, 'Validation failed');

    expect(showToast).toHaveBeenCalledWith('Validation failed: Invalid data');
  });
});

describe('createErrorHandler', () => {
  it('should return a function that handles errors', () => {
    const showToast = vi.fn();
    const handleError = createErrorHandler(showToast);

    expect(typeof handleError).toBe('function');

    handleError(new Error('Test error'));
    expect(showToast).toHaveBeenCalledWith('Test error');
  });

  it('should support context in returned function', () => {
    const showToast = vi.fn();
    const handleError = createErrorHandler(showToast);

    handleError(new Error('Test error'), 'Operation failed');
    expect(showToast).toHaveBeenCalledWith('Operation failed: Test error');
  });
});

describe('withErrorHandler', () => {
  it('should call the wrapped function and return result on success', async () => {
    const showToast = vi.fn();
    const asyncFn = vi.fn().mockResolvedValue('success');
    const wrapped = withErrorHandler(asyncFn, showToast);

    const result = await wrapped('arg1', 'arg2');

    expect(asyncFn).toHaveBeenCalledWith('arg1', 'arg2');
    expect(result).toBe('success');
    expect(showToast).not.toHaveBeenCalled();
  });

  it('should handle errors and return undefined', async () => {
    const showToast = vi.fn();
    const asyncFn = vi.fn().mockRejectedValue(new Error('Async error'));
    const wrapped = withErrorHandler(asyncFn, showToast);

    const result = await wrapped();

    expect(result).toBeUndefined();
    expect(showToast).toHaveBeenCalledWith('Async error');
  });

  it('should include context in error message', async () => {
    const showToast = vi.fn();
    const asyncFn = vi.fn().mockRejectedValue(new Error('Async error'));
    const wrapped = withErrorHandler(asyncFn, showToast, 'Save failed');

    await wrapped();

    expect(showToast).toHaveBeenCalledWith('Save failed: Async error');
  });

  it('should pass all arguments to wrapped function', async () => {
    const showToast = vi.fn();
    const asyncFn = vi.fn().mockResolvedValue('done');
    const wrapped = withErrorHandler(asyncFn, showToast);

    await wrapped(1, 'two', { three: 3 });

    expect(asyncFn).toHaveBeenCalledWith(1, 'two', { three: 3 });
  });
});

