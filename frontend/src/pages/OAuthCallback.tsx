import React, { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { Brand } from "../components/Brand";
import { Icon } from "../components/kit";

// Storage key for OAuth state (must match Login.tsx)
const OAUTH_STATE_KEY = "oauth_state";

export const OAuthCallback: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login } = useAuth();
  const [error, setError] = useState<string | null>(null);

  const AUTH_SERVICE_URL =
    import.meta.env.VITE_AUTH_SERVICE_URL || "http://localhost:8081";

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get("code");
      const returnedState = searchParams.get("state");
      const oauthError = searchParams.get("error");
      const errorDescription = searchParams.get("error_description");

      if (oauthError) {
        console.error("OAuth error from Bungie:", oauthError, errorDescription);
        setError(`OAuth error: ${oauthError} - ${errorDescription || "Unknown error"}`);
        sessionStorage.removeItem(OAUTH_STATE_KEY);
        setTimeout(() => navigate("/login?error=oauth_error"), 3000);
        return;
      }

      if (!code) {
        console.error("No authorization code received");
        setError("No authorization code received from Bungie.");
        sessionStorage.removeItem(OAUTH_STATE_KEY);
        setTimeout(() => navigate("/login?error=no_code"), 3000);
        return;
      }

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
        setError("Security validation failed (state mismatch). Please try logging in again.");
        sessionStorage.removeItem(OAUTH_STATE_KEY);
        setTimeout(() => navigate("/login?error=csrf_mismatch"), 3000);
        return;
      }

      // Clear stored state (one-time use)
      sessionStorage.removeItem(OAUTH_STATE_KEY);

      try {
        const response = await fetch(`${AUTH_SERVICE_URL}/api/auth/bungie/callback`, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: new URLSearchParams({ code, state: returnedState }).toString(),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.error || `Authentication failed (${response.status})`);
        }

        const data = await response.json();
        if (data.error) {
          throw new Error(data.error);
        }

        login(data.token, data.refreshToken, data.user);
        navigate("/dashboard");
      } catch (err) {
        console.error("Error during OAuth callback:", err);
        const message = err instanceof Error ? err.message : "Unknown error";
        setError(`Authentication failed: ${message}`);
        setTimeout(() => navigate("/login?error=callback_failed"), 3000);
      }
    };

    handleCallback();
  }, [searchParams, navigate, login, AUTH_SERVICE_URL]);

  return (
    <div className="gt-login">
      <div className="gt-login-bg" />
      <div className="gt-login-card" style={{ textAlign: "center", alignItems: "center" }}>
        <Brand />
        {error ? (
          <>
            <div
              className="gt-empty-mark"
              style={{ "--em": "var(--c-danger)" } as React.CSSProperties & Record<string, string>}
            >
              <Icon name="info" size="3rem" stroke={1.5} />
            </div>
            <h1 className="gt-login-title" style={{ fontSize: "var(--t-xl)" }}>
              Authentication error
            </h1>
            <p className="gt-login-sub">{error}</p>
            <div className="gt-login-note mono">Redirecting to sign in…</div>
          </>
        ) : (
          <>
            <h1 className="gt-login-title" style={{ fontSize: "var(--t-xl)" }}>
              Completing sign-in…
            </h1>
            <p className="gt-login-sub">Verifying your Bungie.net authorization.</p>
            <Icon
              name="refresh"
              size="2rem"
              className="gt-fresh-icon"
              style={{ color: "var(--c-signal)", animation: "gt-spin 0.9s linear infinite" }}
            />
          </>
        )}
      </div>
    </div>
  );
};
