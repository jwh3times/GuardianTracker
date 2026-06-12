import React, { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { Brand } from "../components/Brand";
import { Icon } from "../components/kit";
import { API_URL } from "../lib/api";
import type { AuthTokenResponse } from "../types/api";

export const OAuthCallback: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login } = useAuth();
  const [error, setError] = useState<string | null>(null);
  // The auth code is single-use; React StrictMode double-invokes effects in dev,
  // so guard against submitting it twice.
  const submitted = useRef(false);

  useEffect(() => {
    if (submitted.current) return;
    submitted.current = true;

    const handleCallback = async () => {
      const code = searchParams.get("code");
      const returnedState = searchParams.get("state");
      const oauthError = searchParams.get("error");
      const errorDescription = searchParams.get("error_description");

      if (oauthError) {
        console.error("OAuth error from Bungie:", oauthError, errorDescription);
        setError(`OAuth error: ${oauthError} - ${errorDescription || "Unknown error"}`);
        setTimeout(() => navigate("/login?error=oauth_error"), 3000);
        return;
      }

      if (!code) {
        console.error("No authorization code received");
        setError("No authorization code received from Bungie.");
        setTimeout(() => navigate("/login?error=no_code"), 3000);
        return;
      }

      try {
        const response = await fetch(`${API_URL}/api/auth/bungie/callback`, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: new URLSearchParams({ code, state: returnedState ?? "" }).toString(),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.error || `Authentication failed (${response.status})`);
        }

        const data = (await response.json()) as AuthTokenResponse;
        if ((data as { error?: string }).error) {
          throw new Error((data as { error?: string }).error);
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
  }, [searchParams, navigate, login]);

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
