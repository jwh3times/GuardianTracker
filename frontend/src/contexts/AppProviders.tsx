import React, { useMemo, useSyncExternalStore } from "react";
import { browserSessionClient } from "../lib/browserSessionBrowser";
import { createApplicationIdentity } from "../lib/applicationIdentity";
import { IdentityContext } from "./IdentityMutation";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "./AuthContext";
import { PreferencesProvider } from "./PreferencesContext";
import { FlagsProvider } from "./FlagsContext";
import { CharacterProvider } from "./CharacterContext";
import { ToastProvider } from "../components/Toast";
import { queryClient as defaultQueryClient } from "../lib/api";

/**
 * The context tower every page renders inside, in one place.
 *
 * The ordering is not arbitrary and the providers are not interchangeable:
 *
 * - `PreferencesProvider` reads `useAuth`, so it must sit inside `AuthProvider`.
 * - An identity boundary replaces the QueryClient and remounts the provider tree.
 *   Same-membership refresh retains both; late mutation work stays with its old client.
 *
 * Deliberately NOT included:
 *
 * - **The router.** `index.tsx` mounts `BrowserRouter`; tests mount
 *   `MemoryRouter`, and some need real `Routes`/`Route` matching. Baking a
 *   router in would force a choice on both.
 * - **`ErrorBoundary`.** It stays outermost in `App.tsx`. Inside a test tower it
 *   would swallow render errors and turn a failed assertion into a fallback
 *   render — including the "hook throws outside its provider" assertions.
 */
export function AppProviders({
  children,
  client = defaultQueryClient,
}: {
  children: React.ReactNode;
  /** Defaults to the app singleton. Tests pass a fresh, retry-disabled client. */
  client?: QueryClient;
}) {
  const identities = useMemo(
    () => createApplicationIdentity(browserSessionClient, client),
    [client],
  );
  const scope = useSyncExternalStore(
    identities.subscribe,
    identities.getSnapshot,
  );
  return (
    <IdentityContext.Provider value={scope} key={scope.revision}>
      <QueryClientProvider client={scope.client}>
        <AuthProvider>
          <PreferencesProvider
            resetLocal={scope.revision > 0}
            isCurrent={scope.isCurrent}
          >
            <ToastProvider>{children}</ToastProvider>
          </PreferencesProvider>
        </AuthProvider>
      </QueryClientProvider>
    </IdentityContext.Provider>
  );
}

/**
 * The second half of the tower, mounted only below the authentication gate.
 *
 * Kept separate from {@link AppProviders} because the split is load-bearing:
 * both of these fetch on mount for a signed-in user, and neither is available
 * to a page rendered above the gate (Login, the OAuth callback). Hoisting a page
 * out of `ProtectedLayout` silently loses them, which is exactly why this is a
 * named module rather than an inline pair of tags.
 *
 * `CharacterProvider` reads `useAuth` and issues a query, so it must sit inside
 * `AppProviders`.
 */
export function AuthedProviders({ children }: { children: React.ReactNode }) {
  return (
    <FlagsProvider>
      <CharacterProvider>{children}</CharacterProvider>
    </FlagsProvider>
  );
}
