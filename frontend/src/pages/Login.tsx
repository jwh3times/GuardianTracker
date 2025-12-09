import React, { useState } from "react";
import { Button } from "../components/ui/Button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/Card";
import { Loader2 } from "lucide-react";

// Storage key for OAuth state (CSRF protection)
const OAUTH_STATE_KEY = "oauth_state";

export function Login() {
  const [bungieLoading, setBungieLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Get auth service URL from environment
  const AUTH_SERVICE_URL =
    process.env.REACT_APP_AUTH_SERVICE_URL || "http://localhost:8081";

  // Bungie OAuth login
  const handleBungieLogin = async () => {
    try {
      setBungieLoading(true);
      setError(null);

      const response = await fetch(`${AUTH_SERVICE_URL}/api/auth/bungie`);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      if (data.authUrl && data.state) {
        // Store the state in sessionStorage for CSRF validation
        sessionStorage.setItem(OAUTH_STATE_KEY, data.state);

        // Redirect to Bungie OAuth
        window.location.href = data.authUrl;
      } else {
        throw new Error("No authorization URL received from server");
      }
    } catch (err) {
      console.error("Error initiating Bungie OAuth:", err);
      const errorMessage =
        err instanceof Error ? err.message : "Unknown error";
      setError(`Failed to start authentication: ${errorMessage}`);
      setBungieLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <Card className="bg-background/95 backdrop-blur">
          <CardHeader className="text-center space-y-4">
            <div className="mx-auto w-16 h-16 bg-gradient-to-br from-destiny-exotic to-destiny-legendary rounded-full flex items-center justify-center">
              <span className="text-2xl font-bold text-white">GT</span>
            </div>
            <CardTitle className="text-2xl font-bold">
              Guardian Tracker
            </CardTitle>
            <CardDescription className="text-base">
              Sign in with your Bungie.net account
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Error Display */}
            {error && (
              <div className="p-3 bg-destructive/10 border border-destructive/20 rounded-md">
                <p className="text-sm text-destructive">{error}</p>
              </div>
            )}

            {/* Bungie OAuth Login */}
            <div className="space-y-3">
              <Button
                type="button"
                className="w-full bg-gradient-to-r from-destiny-exotic to-destiny-legendary hover:from-destiny-legendary hover:to-destiny-exotic text-lg py-6"
                onClick={handleBungieLogin}
                disabled={bungieLoading}
              >
                {bungieLoading ? (
                  <>
                    <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                    Redirecting to Bungie...
                  </>
                ) : (
                  "Login with Bungie.net"
                )}
              </Button>
              <p className="text-sm text-muted-foreground text-center">
                Secure OAuth authentication with your Bungie.net account.
                <br />
                Your Guardian data will be automatically synced.
              </p>
            </div>

            <div className="mt-6 text-center">
              <p className="text-sm text-muted-foreground">
                Don't have a Bungie.net account?{" "}
                <a
                  href="https://www.bungie.net/en/User/SignUp"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-purple-600 hover:underline"
                >
                  Create one here
                </a>
              </p>
            </div>
          </CardContent>
        </Card>

        <div className="mt-8 text-center text-sm text-muted-foreground">
          <p>Guardian Tracker is not affiliated with Bungie or Destiny 2.</p>
        </div>
      </div>
    </div>
  );
}
