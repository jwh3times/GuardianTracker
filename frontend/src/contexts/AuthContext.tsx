import React, {
  createContext,
  useContext,
  useState,
  ReactNode,
} from "react";

interface User {
  id: string;
  displayName: string;
  membershipType: number;
  membershipId: string;
  platform?: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  refreshToken: string | null;
  login: (token: string, refreshToken: string, user: User) => void;
  logout: () => void;
  refreshAccessToken: () => Promise<boolean>;
  isAuthenticated: boolean;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

interface AuthState {
  user: User | null;
  token: string | null;
  refreshToken: string | null;
}

// Reads persisted authentication from localStorage. Runs synchronously as the
// lazy initializer for the auth state, so the correct value is available on the
// first render (no loading flash, no setState-in-effect on mount).
function readStoredAuth(): AuthState {
  const storedToken = localStorage.getItem("guardian_token");
  const storedRefreshToken = localStorage.getItem("guardian_refresh_token");
  const storedUser = localStorage.getItem("guardian_user");

  if (storedToken && storedUser) {
    try {
      return {
        token: storedToken,
        refreshToken: storedRefreshToken,
        user: JSON.parse(storedUser) as User,
      };
    } catch (error) {
      console.error("Error parsing stored user data:", error);
      // Clear invalid data
      localStorage.removeItem("guardian_token");
      localStorage.removeItem("guardian_refresh_token");
      localStorage.removeItem("guardian_user");
    }
  }

  return { token: null, refreshToken: null, user: null };
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [auth, setAuth] = useState<AuthState>(readStoredAuth);
  const { user, token, refreshToken } = auth;

  // Get auth service URL from environment
  const AUTH_SERVICE_URL =
    import.meta.env.VITE_AUTH_SERVICE_URL || "http://localhost:8081";

  const login = (newToken: string, newRefreshToken: string, newUser: User) => {
    setAuth({ token: newToken, refreshToken: newRefreshToken, user: newUser });
    localStorage.setItem("guardian_token", newToken);
    localStorage.setItem("guardian_refresh_token", newRefreshToken);
    localStorage.setItem("guardian_user", JSON.stringify(newUser));
  };

  const logout = () => {
    setAuth({ token: null, refreshToken: null, user: null });
    localStorage.removeItem("guardian_token");
    localStorage.removeItem("guardian_refresh_token");
    localStorage.removeItem("guardian_user");
  };

  const refreshAccessToken = async (): Promise<boolean> => {
    if (!refreshToken) {
      console.error("No refresh token available");
      return false;
    }

    try {
      const response = await fetch(`${AUTH_SERVICE_URL}/api/auth/refresh`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ refreshToken }),
      });

      if (!response.ok) {
        console.error("Failed to refresh token");
        logout();
        return false;
      }

      const data = await response.json();

      // Update tokens and user data
      setAuth({
        token: data.token,
        refreshToken: data.refreshToken,
        user: data.user,
      });
      localStorage.setItem("guardian_token", data.token);
      localStorage.setItem("guardian_refresh_token", data.refreshToken);
      localStorage.setItem("guardian_user", JSON.stringify(data.user));

      console.log("Successfully refreshed access token");
      return true;
    } catch (error) {
      console.error("Error refreshing token:", error);
      logout();
      return false;
    }
  };

  const value: AuthContextType = {
    user,
    token,
    refreshToken,
    login,
    logout,
    refreshAccessToken,
    isAuthenticated: !!token && !!user,
    // Auth is hydrated synchronously from localStorage during render, so it is
    // always resolved by the time consumers read it.
    isLoading: false,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};

export default AuthProvider;
