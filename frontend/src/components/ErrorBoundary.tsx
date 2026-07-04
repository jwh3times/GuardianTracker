import React, { Component, ErrorInfo, ReactNode } from "react";
import { Button, EmptyState } from "./primitives";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("ErrorBoundary caught an error:", error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div
          className="gt-page"
          style={{ minHeight: "100vh", display: "grid", placeItems: "center" }}
        >
          <EmptyState
            icon="lock"
            title="Something went wrong"
            body="An unexpected error occurred. Reload the page to continue."
            action={
              <Button onClick={() => window.location.reload()}>Reload</Button>
            }
            secondary={
              import.meta.env.DEV && this.state.error
                ? String(this.state.error)
                : undefined
            }
          />
        </div>
      );
    }

    return this.props.children;
  }
}
