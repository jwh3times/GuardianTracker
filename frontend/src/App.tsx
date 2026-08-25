import React, { Suspense, lazy } from "react";
import { Routes, Route, Navigate, Outlet, useNavigate } from "react-router";
import { useAuth } from "./contexts/AuthContext";
import { useFlags } from "./contexts/FlagsContext";
import { AppProviders, AuthedProviders } from "./contexts/AppProviders";
import { AppShell } from "./components/AppShell";
import { LockedFeature } from "./features/admin/AdminKit";
import { LoadingSpinner } from "./components/LoadingSpinner";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { OnboardingTour } from "./features/onboarding/OnboardingTour";

// Lazy-loaded pages for code splitting
const Dashboard = lazy(() =>
  import("./features/dashboard/Dashboard").then((m) => ({
    default: m.Dashboard,
  })),
);
const Collections = lazy(() =>
  import("./features/collections/Collections").then((m) => ({
    default: m.Collections,
  })),
);
const Cosmetics = lazy(() =>
  import("./features/cosmetics/Cosmetics").then((m) => ({
    default: m.Cosmetics,
  })),
);
const WishList = lazy(() =>
  import("./features/wishlist/WishList").then((m) => ({
    default: m.WishList,
  })),
);
const ThisWeek = lazy(() =>
  import("./features/weekly/ThisWeek").then((m) => ({
    default: m.ThisWeek,
  })),
);
const Catalysts = lazy(() =>
  import("./features/collections/Catalysts").then((m) => ({
    default: m.Catalysts,
  })),
);
const Triumphs = lazy(() =>
  import("./features/collections/Triumphs").then((m) => ({
    default: m.Triumphs,
  })),
);
const Settings = lazy(() =>
  import("./features/settings/Settings").then((m) => ({
    default: m.Settings,
  })),
);
const Admin = lazy(() =>
  import("./features/admin/Admin").then((m) => ({ default: m.Admin })),
);
const Login = lazy(() =>
  import("./features/auth/Login").then((m) => ({ default: m.Login })),
);
const OAuthCallback = lazy(() =>
  import("./features/auth/OAuthCallback").then((m) => ({
    default: m.OAuthCallback,
  })),
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
    <AuthedProviders>
      <AppShell>
        <Suspense fallback={<PageLoader />}>
          <Outlet />
        </Suspense>
        <OnboardingTour />
      </AppShell>
    </AuthedProviders>
  );
};

const ReauthorizeRoute: React.FC = () => {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? (
    <Login mode="reauthorize" />
  ) : (
    <Navigate to="/login" replace />
  );
};

// AdminRoute guards the admin console; the server enforces RequireAdmin regardless.
const AdminRoute: React.FC<{ children: React.ReactElement }> = ({
  children,
}) => {
  const { isAdmin, isLoading } = useFlags();
  if (isLoading) return <PageLoader />;
  if (!isAdmin) return <Navigate to="/dashboard" replace />;
  return children;
};

// FlaggedRoute gates a page on a feature flag: hidden flags redirect home, locked
// flags show the upsell, accessible flags render the page (port of resolvePage).
const FlaggedRoute: React.FC<{
  flagKey: string;
  children: React.ReactElement;
}> = ({ flagKey, children }) => {
  const { flagState } = useFlags();
  const navigate = useNavigate();
  const st = flagState(flagKey);
  if (!st.enabled) return <Navigate to="/dashboard" replace />;
  if (st.locked && st.flag) {
    return (
      <LockedFeature
        flag={st.flag}
        onChangeTier={() => navigate("/settings")}
        onBack={() => navigate("/dashboard")}
      />
    );
  }
  return children;
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
        <Route path="/reauthorize" element={<ReauthorizeRoute />} />
        <Route path="/auth/callback" element={<OAuthCallback />} />

        <Route element={<ProtectedLayout />}>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/collections" element={<Collections />} />
          <Route
            path="/cosmetics"
            element={
              <FlaggedRoute flagKey="cosmetics">
                <Cosmetics />
              </FlaggedRoute>
            }
          />
          <Route path="/wishlist" element={<WishList />} />
          <Route
            path="/this-week"
            element={
              <FlaggedRoute flagKey="weekly-planner">
                <ThisWeek />
              </FlaggedRoute>
            }
          />
          <Route
            path="/catalysts"
            element={
              <FlaggedRoute flagKey="catalysts-crafting">
                <Catalysts />
              </FlaggedRoute>
            }
          />
          <Route
            path="/triumphs"
            element={
              <FlaggedRoute flagKey="triumphs-seals">
                <Triumphs />
              </FlaggedRoute>
            }
          />
          <Route path="/settings" element={<Settings />} />
          <Route
            path="/admin"
            element={
              <AdminRoute>
                <Admin />
              </AdminRoute>
            }
          />
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
  // ErrorBoundary stays outside AppProviders on purpose — see the note there.
  return (
    <ErrorBoundary>
      <AppProviders>
        <AppContent />
      </AppProviders>
    </ErrorBoundary>
  );
}

export default App;
