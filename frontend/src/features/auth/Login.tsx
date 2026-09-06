import React, { useState } from "react";
import { Brand } from "../../components/Brand";
import { Button } from "../../components/primitives";
import { Icon } from "../../components/Icon";
import { browserSessionClient } from "../../lib/browserSessionBrowser";
import {
  currentReturnPath,
  markBungieReconnect,
} from "../../lib/bungieReauthorization";

interface LoginProps {
  mode?: "login" | "reauthorize";
}

export function Login({ mode = "login" }: LoginProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleBungieLogin = async () => {
    try {
      setLoading(true);
      setError(null);

      if (mode === "reauthorize") {
        markBungieReconnect(currentReturnPath());
      }

      const data = await browserSessionClient.beginAuthorization();
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
          {mode === "reauthorize" ? (
            <>Reconnect Bungie to continue.</>
          ) : (
            <>
              See what you're missing.
              <br />
              Chase what matters this week.
            </>
          )}
        </h1>
        <p className="gt-login-sub">
          {mode === "reauthorize"
            ? "Your Guardian Tracker session is still active. Bungie authorization expired and needs to be renewed."
            : "A Destiny 2 companion that turns your sprawling collection into a focused plan of action."}
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
            onClick={() => void handleBungieLogin()}
            style={{ width: "100%" }}
          >
            {mode === "reauthorize"
              ? "Reconnect Bungie"
              : "Sign in with Bungie"}
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
