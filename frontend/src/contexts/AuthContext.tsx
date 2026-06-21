import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { api, User } from '../lib/api';

interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, name: string, password: string) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<string | null>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));
  const [isLoading, setIsLoading] = useState(true);

  const refreshTokenFn = useCallback(async (): Promise<string | null> => {
    const storedRefreshToken = localStorage.getItem('refreshToken');
    if (!storedRefreshToken) return null;

    try {
      const response = await api.auth.refresh(storedRefreshToken);
      localStorage.setItem('token', response.access_token);
      localStorage.setItem('refreshToken', response.refresh_token);
      setToken(response.access_token);
      return response.access_token;
    } catch {
      localStorage.removeItem('token');
      localStorage.removeItem('refreshToken');
      setToken(null);
      setUser(null);
      return null;
    }
  }, []);

  useEffect(() => {
    if (token) {
      api.auth.me(token)
        .then(setUser)
        .catch(async () => {
          // Try to refresh the token
          const newToken = await refreshTokenFn();
          if (newToken) {
            api.auth.me(newToken).then(setUser).catch(() => {});
          }
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [token, refreshTokenFn]);

  // Auto-refresh token every 10 minutes
  useEffect(() => {
    if (!token) return;

    const interval = setInterval(async () => {
      await refreshTokenFn();
    }, 10 * 60 * 1000); // 10 minutes

    return () => clearInterval(interval);
  }, [token, refreshTokenFn]);

  const login = async (email: string, password: string) => {
    const response = await api.auth.login(email, password);
    localStorage.setItem('token', response.access_token);
    localStorage.setItem('refreshToken', response.refresh_token);
    setToken(response.access_token);
    const user = await api.auth.me(response.access_token);
    setUser(user);
  };

  const register = async (email: string, name: string, password: string) => {
    await api.auth.register(email, name, password);
    // Auto-login after registration
    await login(email, password);
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('refreshToken');
    setToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, token, isLoading, login, register, logout, refreshToken: refreshTokenFn }}>
      {children}
    </AuthContext.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
