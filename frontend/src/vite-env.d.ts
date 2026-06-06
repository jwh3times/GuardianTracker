/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_GRAPHQL_URL: string
  readonly VITE_AUTH_SERVICE_URL: string
  readonly VITE_AUTH_REDIRECT_URI: string
  readonly VITE_ENABLE_ANALYTICS: string
  readonly VITE_ENABLE_DEBUG_MODE: string
  readonly VITE_NAME: string
  readonly VITE_VERSION: string
  readonly VITE_ENVIRONMENT: string
  readonly VITE_BUNGIE_SIGNUP_URL: string
  readonly VITE_SUPPORT_EMAIL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
