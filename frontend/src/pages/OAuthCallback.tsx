import React, { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";

export const OAuthCallback: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login } = useAuth();

  // Get auth service URL from environment (constant per deployment)
  const AUTH_SERVICE_URL = React.useMemo(
    () => process.env.REACT_APP_AUTH_SERVICE_URL || "http://localhost:8081",
    []
  );

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get("code");
      const error = searchParams.get("error");
      const errorDescription = searchParams.get("error_description");

      console.log("OAuth callback received:", {
        code: code?.substring(0, 10) + "...",
        error,
        errorDescription,
      });

      if (error) {
        console.error("OAuth error from Bungie:", error, errorDescription);
        alert(
          `OAuth error: ${error} - ${errorDescription || "Unknown error from Bungie"}`
        );
        navigate("/login?error=oauth_error");
        return;
      }

      if (!code) {
        console.error("No authorization code received");
        alert("No authorization code received from Bungie. Please try again.");
        navigate("/login?error=no_code");
        return;
      }

      try {
        console.log("Exchanging authorization code for token...");

        // Exchange code for token
        const response = await fetch(
          `${AUTH_SERVICE_URL}/api/auth/bungie/callback`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/x-www-form-urlencoded",
            },
            body: `code=${encodeURIComponent(code)}`,
          }
        );

        console.log("Token exchange response status:", response.status);

        if (!response.ok) {
          const errorText = await response.text();
          console.error("Token exchange failed:", errorText);
          throw new Error(
            `HTTP error! status: ${response.status} - ${errorText}`
          );
        }

        const data = await response.json();
        console.log("Token exchange successful:", {
          user: data.user?.displayName,
          hasRefreshToken: !!data.refreshToken,
        });

        if (data.error) {
          throw new Error(
            data.error + (data.details ? ` - ${data.details}` : "")
          );
        }

        // Login with the received tokens and user data
        login(data.token, data.refreshToken, data.user);

        alert(`Welcome, ${data.user.displayName}! Authentication successful.`);

        // Redirect to dashboard
        navigate("/dashboard");
      } catch (error) {
        console.error("Error during OAuth callback:", error);
        const errorMessage =
          error instanceof Error ? error.message : "Unknown error";
        alert(`Authentication failed: ${errorMessage}`);
        navigate("/login?error=callback_failed");
      }
    };

    handleCallback();
    // AUTH_SERVICE_URL is memoized with empty deps, so it won't change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams, navigate, login]);

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
