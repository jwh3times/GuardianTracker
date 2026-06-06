import React, { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";

// Storage key for OAuth state (must match Login.tsx)
const OAUTH_STATE_KEY = "oauth_state";

export const OAuthCallback: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login } = useAuth();
  const [error, setError] = useState<string | null>(null);

  // Get auth service URL from environment
  const AUTH_SERVICE_URL =
    import.meta.env.VITE_AUTH_SERVICE_URL || "http://localhost:8081";

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get("code");
      const returnedState = searchParams.get("state");
      const oauthError = searchParams.get("error");
      const errorDescription = searchParams.get("error_description");

      // Handle OAuth errors from Bungie
      if (oauthError) {
        console.error("OAuth error from Bungie:", oauthError, errorDescription);
        setError(
          `OAuth error: ${oauthError} - ${errorDescription || "Unknown error"}`
        );
        sessionStorage.removeItem(OAUTH_STATE_KEY);
        setTimeout(() => navigate("/login?error=oauth_error"), 3000);
        return;
      }

      // Validate authorization code
      if (!code) {
        console.error("No authorization code received");
        setError("No authorization code received from Bungie.");
        sessionStorage.removeItem(OAUTH_STATE_KEY);
        setTimeout(() => navigate("/login?error=no_code"), 3000);
        return;
      }

      // Validate CSRF state
      const storedState = sessionStorage.getItem(OAUTH_STATE_KEY);
      if (!storedState || !returnedState) {
        console.error("Missing CSRF state");
        setError("Security validation failed. Please try logging in again.");
        sessionStorage.removeItem(OAUTH_STATE_KEY);
        setTimeout(() => navigate("/login?error=csrf_missing"), 3000);
        return;
      }

      if (storedState !== returnedState) {
        console.error("CSRF state mismatch");
        setError(
          "Security validation failed (state mismatch). Please try logging in again."
        );
        sessionStorage.removeItem(OAUTH_STATE_KEY);
        setTimeout(() => navigate("/login?error=csrf_mismatch"), 3000);
        return;
      }

      // Clear stored state (one-time use)
      sessionStorage.removeItem(OAUTH_STATE_KEY);

      try {
        // Exchange code for token, including state for server-side validation
        const response = await fetch(
          `${AUTH_SERVICE_URL}/api/auth/bungie/callback`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/x-www-form-urlencoded",
            },
            body: new URLSearchParams({
              code: code,
              state: returnedState,
            }).toString(),
          }
        );

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Authentication failed (${response.status})`
          );
        }

        const data = await response.json();

        if (data.error) {
          throw new Error(data.error);
        }

        // Login with the received tokens and user data
        login(data.token, data.refreshToken, data.user);

        // Redirect to dashboard
        navigate("/dashboard");
      } catch (err) {
        console.error("Error during OAuth callback:", err);
        const errorMessage =
          err instanceof Error ? err.message : "Unknown error";
        setError(`Authentication failed: ${errorMessage}`);
        setTimeout(() => navigate("/login?error=callback_failed"), 3000);
      }
    };

    handleCallback();
  }, [searchParams, navigate, login, AUTH_SERVICE_URL]);

  if (error) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center max-w-md p-6">
          <div className="text-destructive mb-4">
            <svg
              className="mx-auto h-12 w-12"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
          </div>
          <h2 className="text-lg font-semibold text-foreground mb-2">
            Authentication Error
          </h2>
          <p className="text-muted-foreground mb-4">{error}</p>
          <p className="text-sm text-muted-foreground">
            Redirecting to login...
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="text-center">
        <LoadingSpinner size="lg" />
        <p className="mt-4 text-muted-foreground">
          Completing Bungie authentication...
        </p>
      </div>
    </div>
  );
};
