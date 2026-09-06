import React, { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate, useSearchParams } from "react-router";
import { useAuth } from "../../contexts/AuthContext";
import { Brand } from "../../components/Brand";
import { Icon } from "../../components/Icon";
import { apiFetch } from "../../lib/api";
import {
  bungieReconnectReturnTo,
  clearBungieReconnect,
  hasBungieReconnectIntent,
} from "../../lib/bungieReauthorization";
import { browserSessionClient } from "../../lib/browserSessionBrowser";

export const OAuthCallback: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const { isAuthenticated } = useAuth();
  const [error, setError] = useState<string | null>(null);
  // The auth code is single-use; React StrictMode double-invokes effects in dev,
  // so guard against submitting it twice.
  const submitted = useRef(false);

  useEffect(() => {
    if (submitted.current) return;
    submitted.current = true;

    const handleCallback = async () => {
      const reconnect = hasBungieReconnectIntent() && isAuthenticated;
      if (hasBungieReconnectIntent() && !isAuthenticated) {
        // The app session disappeared while the user was at Bungie. The same
        // authorization code can complete a normal login instead.
        clearBungieReconnect();
      }
      const code = searchParams.get("code");
      const returnedState = searchParams.get("state");
      const oauthError = searchParams.get("error");
      const errorDescription = searchParams.get("error_description");

      if (oauthError) {
        console.error("OAuth error from Bungie:", oauthError, errorDescription);
        setError(
          `OAuth error: ${oauthError} - ${errorDescription || "Unknown error"}`,
        );
        setTimeout(() => {
          void navigate(
            reconnect ? "/reauthorize" : "/login?error=oauth_error",
          );
        }, 3000);
        return;
      }

      if (!code) {
        console.error("No authorization code received");
        setError("No authorization code received from Bungie.");
        setTimeout(() => {
          void navigate(reconnect ? "/reauthorize" : "/login?error=no_code");
        }, 3000);
        return;
      }

      try {
        const formBody = new URLSearchParams({
          code,
          state: returnedState ?? "",
        }).toString();

        if (reconnect) {
          const returnTo = bungieReconnectReturnTo();
          await apiFetch<void>("/api/auth/bungie/reconnect", {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: formBody,
          });
          await queryClient.invalidateQueries();
          clearBungieReconnect();
          void navigate(returnTo, { replace: true });
          return;
        }

        await browserSessionClient.completeAuthorization({
          code,
          state: returnedState ?? "",
        });
        void navigate("/dashboard");
      } catch (err) {
        console.error("Error during OAuth callback:", err);
        const message = err instanceof Error ? err.message : "Unknown error";
        setError(`Authentication failed: ${message}`);
        setTimeout(() => {
          void navigate(
            reconnect ? "/reauthorize" : "/login?error=callback_failed",
          );
        }, 3000);
      }
    };

    void handleCallback();
  }, [searchParams, navigate, isAuthenticated, queryClient]);

  return (
    <div className="gt-login">
      <div className="gt-login-bg" />
      <div
        className="gt-login-card"
        style={{ textAlign: "center", alignItems: "center" }}
      >
        <Brand />
        {error ? (
          <>
            <div
              className="gt-empty-mark"
              style={
                { "--em": "var(--c-danger)" } as React.CSSProperties &
                  Record<string, string>
              }
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
              {hasBungieReconnectIntent() && isAuthenticated
                ? "Reconnecting Bungie…"
                : "Completing sign-in…"}
            </h1>
            <p className="gt-login-sub">
              Verifying your Bungie.net authorization.
            </p>
            <Icon
              name="refresh"
              size="2rem"
              className="gt-fresh-icon"
              style={{
                color: "var(--c-signal)",
                animation: "gt-spin 0.9s linear infinite",
              }}
            />
          </>
        )}
      </div>
    </div>
  );
};
