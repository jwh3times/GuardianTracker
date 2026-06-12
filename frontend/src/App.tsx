import React, { Suspense, lazy } from "react";
import { Routes, Route, Navigate, Outlet } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import { CharacterProvider } from "./contexts/CharacterContext";
import { PreferencesProvider } from "./contexts/PreferencesContext";
import { queryClient } from "./lib/api";
import { AppShell } from "./components/AppShell";
import { LoadingSpinner } from "./components/ui/LoadingSpinner";
import { ToastProvider } from "./components/ui/Toast";
import { ErrorBoundary } from "./components/ErrorBoundary";

// Lazy-loaded pages for code splitting
const Dashboard = lazy(() =>
  import("./pages/Dashboard").then((m) => ({ default: m.Dashboard }))
);
const Collections = lazy(() =>
  import("./pages/Collections").then((m) => ({ default: m.Collections }))
);
const WishList = lazy(() =>
  import("./pages/WishList").then((m) => ({ default: m.WishList }))
);
const ThisWeek = lazy(() =>
  import("./pages/ThisWeek").then((m) => ({ default: m.ThisWeek }))
);
const Catalysts = lazy(() =>
  import("./pages/Catalysts").then((m) => ({ default: m.Catalysts }))
);
const Triumphs = lazy(() =>
  import("./pages/Triumphs").then((m) => ({ default: m.Triumphs }))
);
const Settings = lazy(() =>
  import("./pages/Settings").then((m) => ({ default: m.Settings }))
);
const Login = lazy(() =>
  import("./pages/Login").then((m) => ({ default: m.Login }))
);
const OAuthCallback = lazy(() =>
  import("./pages/OAuthCallback").then((m) => ({ default: m.OAuthCallback }))
);

const PageLoader: React.FC = () => (
  <div
    style={{
      minHeight: "100vh",
      display: "grid",
      placeItems: "center",
      background: "var(--c-bg)",
    }}
  >
    <LoadingSpinner size="lg" />
  </div>
);

const ProtectedLayout: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return <PageLoader />;
  }
  if (!isAuthenticated) {
    return <Navigate to="/login" />;
  }

  return (
    <CharacterProvider>
      <AppShell>
        <Suspense fallback={<PageLoader />}>
          <Outlet />
        </Suspense>
      </AppShell>
    </CharacterProvider>
  );
};

const AppContent: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return <PageLoader />;
  }

  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route
          path="/login"
          element={isAuthenticated ? <Navigate to="/dashboard" /> : <Login />}
        />
        <Route path="/auth/callback" element={<OAuthCallback />} />

        <Route element={<ProtectedLayout />}>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/collections" element={<Collections />} />
          <Route path="/wishlist" element={<WishList />} />
          <Route path="/this-week" element={<ThisWeek />} />
          <Route path="/catalysts" element={<Catalysts />} />
          <Route path="/triumphs" element={<Triumphs />} />
          <Route path="/settings" element={<Settings />} />
        </Route>

        <Route
          path="/"
          element={<Navigate to={isAuthenticated ? "/dashboard" : "/login"} />}
        />
        <Route
          path="*"
          element={<Navigate to={isAuthenticated ? "/dashboard" : "/login"} />}
        />
      </Routes>
    </Suspense>
  );
};

function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <PreferencesProvider>
            <ToastProvider>
              <AppContent />
            </ToastProvider>
          </PreferencesProvider>
        </AuthProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;
