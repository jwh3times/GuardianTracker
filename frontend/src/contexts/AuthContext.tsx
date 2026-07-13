import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import type { APIUser } from "../types/api";
import { apiFetch } from "../lib/api";

interface AuthContextType {
  user: APIUser | null;
  token: string | null;
  login: (token: string, user: APIUser) => void;
  logout: () => void;
  logoutAll: () => void;
  isAuthenticated: boolean;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

interface AuthState {
  user: APIUser | null;
  token: string | null;
}

// Reads persisted authentication from localStorage. Runs synchronously as the
// lazy initializer for the auth state, so the correct value is available on the
// first render (no loading flash, no setState-in-effect on mount).
function readStoredAuth(): AuthState {
  const storedToken = localStorage.getItem("guardian_token");
  const storedUser = localStorage.getItem("guardian_user");

  // Refresh tokens moved to an HttpOnly cookie. Remove the legacy value
  // without attempting to bridge it into the new session mechanism.
  localStorage.removeItem("guardian_refresh_token");

  if (storedToken && storedUser) {
    try {
      return {
        token: storedToken,
        user: JSON.parse(storedUser) as APIUser,
      };
    } catch (error) {
      console.error("Error parsing stored user data:", error);
      localStorage.removeItem("guardian_token");
      localStorage.removeItem("guardian_user");
    }
  }

  return { token: null, user: null };
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [auth, setAuth] = useState<AuthState>(readStoredAuth);
  const { user, token } = auth;

  // Sync React state when apiFetch silently refreshes a token via the 401 retry path.
  // apiFetch writes the new access token and user snapshot to localStorage and
  // fires "guardian_token_refreshed".
  useEffect(() => {
    const syncFromStorage = () => {
      const newState = readStoredAuth();
      setAuth(newState);
    };
    window.addEventListener("guardian_token_refreshed", syncFromStorage);
    return () =>
      window.removeEventListener("guardian_token_refreshed", syncFromStorage);
  }, []);

  const login = (newToken: string, newUser: APIUser) => {
    setAuth({ token: newToken, user: newUser });
    localStorage.setItem("guardian_token", newToken);
    localStorage.removeItem("guardian_refresh_token");
    localStorage.setItem("guardian_user", JSON.stringify(newUser));
  };

  const clearStoredAuth = () => {
    setAuth({ token: null, user: null });
    localStorage.removeItem("guardian_token");
    localStorage.removeItem("guardian_refresh_token");
    localStorage.removeItem("guardian_user");
  };

  // Ends only this device's session; other devices stay signed in.
  const logout = () => {
    // Clear immediately, then clear again after the request settles in case an
    // expired access token triggered a successful refresh before logout.
    void apiFetch("/api/auth/logout", { method: "POST" })
      .catch(() => {})
      .finally(clearStoredAuth);
    clearStoredAuth();
  };

  // Signs out every device for the account (bumps token_version server-side).
  const logoutAll = () => {
    void apiFetch("/api/auth/logout/all", { method: "POST" })
      .catch(() => {})
      .finally(clearStoredAuth);
    clearStoredAuth();
  };

  const value: AuthContextType = {
    user,
    token,
    login,
    logout,
    logoutAll,
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
