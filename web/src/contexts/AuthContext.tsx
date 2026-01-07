/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { api } from '../services/api';

interface User {
  id: number;
  login: string;
  name: string;
  email: string;
  avatar_url: string;
  roles?: string[];
}

interface AuthConfig {
  enabled: boolean;
  login_url?: string;
  authorization_rules?: {
    requires_org_membership?: boolean;
    required_orgs?: string[];
    requires_team_membership?: boolean;
    required_teams?: string[];
    requires_enterprise_admin?: boolean;
    requires_enterprise_membership?: boolean;
    enterprise?: string;
  };
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  authEnabled: boolean;
  authConfig: AuthConfig | null;
  /** Login via GitHub OAuth */
  login: () => void;
  logout: () => Promise<void>;
  refreshAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null);

  const fetchCurrentUser = useCallback(async () => {
    try {
      const userData = await api.getCurrentUser();
      setUser(userData);
    } catch {
      // Not authenticated or session expired
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch auth configuration on mount
  useEffect(() => {
    const fetchAuthConfig = async () => {
      try {
        const config = await api.getAuthConfig();
        setAuthConfig(config);
        
        // If auth is enabled, fetch current user
        if (config.enabled) {
          await fetchCurrentUser();
        } else {
          setIsLoading(false);
        }
      } catch {
        setIsLoading(false);
      }
    };

    fetchAuthConfig();
  }, [fetchCurrentUser]);

  const login = useCallback(() => {
    // Redirect to backend GitHub OAuth login
    window.location.href = '/api/v1/auth/login';
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.logout();
      setUser(null);
      // Redirect to login page after logout
      window.location.href = '/login';
    } catch {
      // Even if API call fails, clear local state and redirect
      setUser(null);
      window.location.href = '/login';
    }
  }, []);

  const refreshAuth = useCallback(async () => {
    if (authConfig?.enabled) {
      await fetchCurrentUser();
    }
  }, [authConfig?.enabled, fetchCurrentUser]);

  const value: AuthContextType = {
    user,
    isAuthenticated: user !== null,
    isLoading,
    authEnabled: authConfig?.enabled || false,
    authConfig,
    login,
    logout,
    refreshAuth,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

