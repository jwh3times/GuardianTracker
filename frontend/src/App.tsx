import React, { Suspense, lazy } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { ApolloProvider } from "@apollo/client";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import { apolloClient } from "./lib/apollo";
import { Navigation } from "./components/Navigation";
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
const Login = lazy(() =>
  import("./pages/Login").then((m) => ({ default: m.Login }))
);
const OAuthCallback = lazy(() =>
  import("./pages/OAuthCallback").then((m) => ({ default: m.OAuthCallback }))
);

// Loading fallback for lazy-loaded components
const PageLoader: React.FC = () => (
  <div className="min-h-screen bg-background flex items-center justify-center">
    <LoadingSpinner size="lg" />
  </div>
);

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return <PageLoader />;
  }

  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};

const AppContent: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return <PageLoader />;
  }

  return (
    <div className="min-h-screen bg-background">
      <Suspense fallback={<PageLoader />}>
        <Routes>
          <Route
            path="/login"
            element={isAuthenticated ? <Navigate to="/dashboard" /> : <Login />}
          />

          <Route path="/auth/callback" element={<OAuthCallback />} />

          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <Navigation />
                <Dashboard />
              </ProtectedRoute>
            }
          />

          <Route
            path="/collections"
            element={
              <ProtectedRoute>
                <Navigation />
                <Collections />
              </ProtectedRoute>
            }
          />

          <Route
            path="/wishlist"
            element={
              <ProtectedRoute>
                <Navigation />
                <WishList />
              </ProtectedRoute>
            }
          />

          <Route
            path="/"
            element={<Navigate to={isAuthenticated ? "/dashboard" : "/login"} />}
          />
        </Routes>
      </Suspense>
    </div>
  );
};

function App() {
  return (
    <ErrorBoundary>
      <ApolloProvider client={apolloClient}>
        <AuthProvider>
          <ToastProvider>
            <AppContent />
          </ToastProvider>
        </AuthProvider>
      </ApolloProvider>
    </ErrorBoundary>
  );
}

export default App;
