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

// Response interceptor to handle 401 errors
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Only redirect if not already on login page or auth endpoints
      const currentPath = window.location.pathname;
      if (!currentPath.includes('/login') && !currentPath.includes('/auth/')) {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

