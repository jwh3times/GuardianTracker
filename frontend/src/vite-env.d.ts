/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the Go API service (default http://localhost:8081). */
  readonly VITE_API_URL?: string;
  /**
   * Development only: the origin OAuth sign-in starts on, when Bungie's
   * callback lands on an HTTPS tunnel instead (issue #317). Unset in production.
   */
  readonly VITE_OAUTH_COMPLETION_ORIGIN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
