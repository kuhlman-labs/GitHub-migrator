import type { AxiosError } from 'axios';

/**
 * Standard error response structure from the API
 */
interface ApiErrorResponse {
  error?: string;
  message?: string;
  details?: Record<string, unknown>;
}

/**
 * Extracts a human-readable error message from various error types.
 * Handles Axios errors, standard JS errors, and unknown error types.
 * 
 * @param error - The error to extract a message from
 * @returns A human-readable error message
 */
export function extractErrorMessage(error: unknown): string {
  // Handle Axios errors with API response
  if (isAxiosError(error)) {
    const data = error.response?.data as ApiErrorResponse | undefined;
    
    // API returned an error message
    if (data?.error) {
      return data.error;
    }
    if (data?.message) {
      return data.message;
    }
    
    // Handle specific HTTP status codes
    if (error.response?.status === 401) {
      return 'Unauthorized. Please log in again.';
    }
    if (error.response?.status === 403) {
      return 'You do not have permission to perform this action.';
    }
    if (error.response?.status === 404) {
      return 'The requested resource was not found.';
    }
    if (error.response?.status === 422) {
      return 'The request contains invalid data.';
    }
    if (error.response?.status === 429) {
      return 'Too many requests. Please try again later.';
    }
    if (error.response?.status === 500) {
      return 'An internal server error occurred. Please try again later.';
    }
    if (error.response?.status === 502 || error.response?.status === 503) {
      return 'The server is temporarily unavailable. Please try again later.';
    }
    
    // Network error (no response)
    if (error.code === 'ERR_NETWORK') {
      return 'Network error. Please check your connection and try again.';
    }
    
    // Request timed out
    if (error.code === 'ECONNABORTED') {
      return 'Request timed out. Please try again.';
    }
    
    // Fallback to Axios message
    if (error.message) {
      return error.message;
    }
  }
  
  // Handle standard Error objects
  if (error instanceof Error) {
    return error.message;
  }
  
  // Handle string errors
  if (typeof error === 'string') {
    return error;
  }
  
  // Fallback for unknown error types
  return 'An unexpected error occurred. Please try again.';
}

/**
 * Type guard to check if an error is an Axios error
 */
function isAxiosError(error: unknown): error is AxiosError<ApiErrorResponse> {
  return (
    typeof error === 'object' &&
    error !== null &&
    'isAxiosError' in error &&
    (error as AxiosError).isAxiosError === true
  );
}

/**
 * Handles an API error by extracting the message and calling a callback.
 * Useful for standardized error handling in components.
 * 
 * @param error - The error to handle
 * @param showToast - Callback to show an error toast (e.g., from ToastContext)
 * @param context - Optional context string to prepend to the error message
 * 
 * @example
 * ```tsx
 * const { showError } = useToast();
 * 
 * try {
 *   await api.deleteItem(id);
 * } catch (error) {
 *   handleApiError(error, showError, 'Failed to delete item');
 * }
 * ```
 */
export function handleApiError(
  error: unknown,
  showToast: (message: string) => void,
  context?: string
): void {
  const message = extractErrorMessage(error);
  const fullMessage = context ? `${context}: ${message}` : message;
  showToast(fullMessage);
}

/**
 * Creates a standardized error handler bound to a toast function.
 * Useful for creating component-level error handlers.
 * 
 * @param showToast - Callback to show an error toast
 * @returns A function that handles errors with optional context
 * 
 * @example
 * ```tsx
 * const { showError } = useToast();
 * const handleError = createErrorHandler(showError);
 * 
 * try {
 *   await api.deleteItem(id);
 * } catch (error) {
 *   handleError(error, 'Delete failed');
 * }
 * ```
 */
export function createErrorHandler(
  showToast: (message: string) => void
): (error: unknown, context?: string) => void {
  return (error: unknown, context?: string) => {
    handleApiError(error, showToast, context);
  };
}

/**
 * Wraps an async function with error handling.
 * Useful for mutation callbacks that need consistent error handling.
 * 
 * @param fn - The async function to wrap
 * @param showToast - Callback to show an error toast
 * @param context - Optional context for error messages
 * @returns A wrapped function that handles errors
 * 
 * @example
 * ```tsx
 * const { showError } = useToast();
 * 
 * const handleDelete = withErrorHandler(
 *   async (id: number) => {
 *     await api.deleteItem(id);
 *     showSuccess('Item deleted!');
 *   },
 *   showError,
 *   'Delete failed'
 * );
 * 
 * <Button onClick={() => handleDelete(item.id)}>Delete</Button>
 * ```
 */
export function withErrorHandler<TArgs extends unknown[], TReturn>(
  fn: (...args: TArgs) => Promise<TReturn>,
  showToast: (message: string) => void,
  context?: string
): (...args: TArgs) => Promise<TReturn | undefined> {
  return async (...args: TArgs): Promise<TReturn | undefined> => {
    try {
      return await fn(...args);
    } catch (error) {
      handleApiError(error, showToast, context);
      return undefined;
    }
  };
}

export type { ApiErrorResponse };

