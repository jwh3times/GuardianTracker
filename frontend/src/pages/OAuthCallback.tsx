import React, { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";

export const OAuthCallback: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login } = useAuth();

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
          "http://localhost:8081/api/auth/bungie/callback",
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
        });

        if (data.error) {
          throw new Error(
            data.error + (data.details ? ` - ${data.details}` : "")
          );
        }

        // Login with the received token and user data
        login(data.token, data.user);

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
