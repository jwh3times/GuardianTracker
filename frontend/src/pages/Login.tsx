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

export function Login() {
  const [bungieLoading, setBungieLoading] = useState(false);

  // Bungie OAuth login
  const handleBungieLogin = async () => {
    try {
      setBungieLoading(true);
      console.log("Starting Bungie OAuth flow...");

      const response = await fetch("http://localhost:8081/api/auth/bungie");
      console.log("Response status:", response.status);
      console.log("Response headers:", response.headers);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      console.log("Response data:", data);

      if (data.authUrl) {
        console.log("Redirecting to:", data.authUrl);
        window.location.href = data.authUrl;
      } else {
        throw new Error("No authorization URL received from server");
      }
    } catch (error) {
      console.error("Error initiating Bungie OAuth:", error);
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      alert(`Failed to start Bungie authentication: ${errorMessage}`);
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
                  "🚀 Login with Bungie.net"
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
