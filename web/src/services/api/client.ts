/**
 * Axios client configuration and interceptors.
 * Shared by all API modules.
 */
import axios from 'axios';

export const client = axios.create({
  baseURL: '/api/v1',
  timeout: 120000, // 120 seconds for long operations like mannequin fetching
  withCredentials: true, // Send cookies with requests
});

// Track whether auth is enabled (set by AuthContext after fetching config)
let authEnabled = false;

export function setAuthEnabled(enabled: boolean) {
  authEnabled = enabled;
}

export function getAuthEnabled() {
  return authEnabled;
}

// Response interceptor to handle 401 errors
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Only redirect if auth is enabled and not already on login page or auth endpoints
      const currentPath = window.location.pathname;
      if (authEnabled && !currentPath.includes('/login') && !currentPath.includes('/auth/')) {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

