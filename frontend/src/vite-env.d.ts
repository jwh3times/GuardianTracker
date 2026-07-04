/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the Go API service (default http://localhost:8081). */
  readonly VITE_API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
