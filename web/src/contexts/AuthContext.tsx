import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
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

  // Fetch auth configuration on mount
  useEffect(() => {
    const fetchAuthConfig = async () => {
      try {
        const config = await api.getAuthConfig();
        setAuthConfig(config);
        
        // If auth is enabled, try to get current user
        if (config.enabled) {
          await fetchCurrentUser();
        } else {
          setIsLoading(false);
        }
      } catch (error) {
        console.error('Failed to fetch auth config:', error);
        setIsLoading(false);
      }
    };

    fetchAuthConfig();
  }, []);

  const fetchCurrentUser = async () => {
    try {
      const userData = await api.getCurrentUser();
      setUser(userData);
    } catch (error) {
      // Not authenticated or session expired
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  };

  const login = () => {
    // Redirect to backend OAuth login
    window.location.href = '/api/v1/auth/login';
  };

  const logout = async () => {
    try {
      await api.logout();
      setUser(null);
      // Redirect to login page after logout
      window.location.href = '/login';
    } catch (error) {
      console.error('Logout failed:', error);
      // Even if API call fails, clear local state and redirect
      setUser(null);
      window.location.href = '/login';
    }
  };

  const refreshAuth = async () => {
    if (authConfig?.enabled) {
      await fetchCurrentUser();
    }
  };

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

