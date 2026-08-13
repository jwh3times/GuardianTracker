import React, { useState } from "react";
import { Brand } from "../../components/Brand";
import { Button } from "../../components/primitives";
import { Icon } from "../../components/Icon";
import type { AuthURLResponse } from "../../types/api";

export function Login() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const AUTH_SERVICE_URL =
    import.meta.env.VITE_API_URL || "http://localhost:8081";

  const handleBungieLogin = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`${AUTH_SERVICE_URL}/api/auth/bungie`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = (await response.json()) as AuthURLResponse;
      if (data.authUrl) {
        window.location.href = data.authUrl;
      } else {
        throw new Error("No authorization URL received from server");
      }
    } catch (err) {
      console.error("Error initiating Bungie OAuth:", err);
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(`Failed to start authentication: ${message}`);
      setLoading(false);
    }
  };

  return (
    <div className="gt-login">
      <div className="gt-login-bg" />
      <div className="gt-login-card">
        <Brand />
        <h1 className="gt-login-title">
          See what you're missing.
          <br />
          Chase what matters this week.
        </h1>
        <p className="gt-login-sub">
          A Destiny 2 companion that turns your sprawling collection into a
          focused plan of action.
        </p>

        {error && (
          <div
            className="gt-drawer-avail"
            style={{
              background: "var(--c-challenging-dim)",
              borderColor:
                "color-mix(in oklch, var(--c-danger) 40%, transparent)",
            }}
          >
            <Icon
              name="info"
              size="1rem"
              style={{ color: "var(--c-danger)" }}
            />
            <span>{error}</span>
          </div>
        )}

        {loading ? (
          <Button variant="primary" disabled style={{ width: "100%" }}>
            <Icon
              name="refresh"
              size="1rem"
              className="gt-fresh-icon"
              style={{ animation: "gt-spin 0.9s linear infinite" }}
            />{" "}
            Redirecting to Bungie.net…
          </Button>
        ) : (
          <Button
            variant="primary"
            icon="bungie"
            onClick={handleBungieLogin}
            style={{ width: "100%" }}
          >
            Sign in with Bungie
          </Button>
        )}

        <div className="gt-login-note mono">
          Read-only · we never modify your Destiny data
        </div>
        <ul className="gt-login-feats">
          <li>
            <Icon
              name="collections"
              size="1rem"
              style={{ color: "var(--c-signal)" }}
            />{" "}
            What's missing across every category
          </li>
          <li>
            <Icon
              name="week"
              size="1rem"
              style={{ color: "var(--c-exotic)" }}
            />{" "}
            The best things to do each week
          </li>
          <li>
            <Icon
              name="catalyst"
              size="1rem"
              style={{ color: "var(--c-legendary)" }}
            />{" "}
            Track catalysts, patterns & seals
          </li>
        </ul>
        <p className="gt-login-note" style={{ color: "var(--c-text-4)" }}>
          Guardian Tracker is not affiliated with Bungie or Destiny 2.
        </p>
      </div>
    </div>
  );
}
