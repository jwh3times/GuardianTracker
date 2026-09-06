import React, {
  createContext,
  useContext,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import type { APIGuardianTrackerUser } from "../types/api";
import { browserSessionClient } from "../lib/browserSessionBrowser";

interface AuthContextType {
  user: APIGuardianTrackerUser | null;
  logout: () => Promise<void>;
  logoutAll: () => Promise<void>;
  isAuthenticated: boolean;
}
const AuthContext = createContext<AuthContextType | undefined>(undefined);
const subscribe = (listener: () => void) =>
  browserSessionClient.subscribe(listener);
const getSnapshot = () => browserSessionClient.getSnapshot();
const logout = () => browserSessionClient.end("current");
const logoutAll = () => browserSessionClient.end("all");

export function AuthProvider({ children }: { children: ReactNode }) {
  const snapshot = useSyncExternalStore(subscribe, getSnapshot);
  return (
    <AuthContext.Provider
      value={{
        user: snapshot.status === "authenticated" ? snapshot.user : null,
        isAuthenticated: snapshot.status === "authenticated",
        logout,
        logoutAll,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined)
    throw new Error("useAuth must be used within an AuthProvider");
  return context;
}
export default AuthProvider;
