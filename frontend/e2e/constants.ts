export const FRONTEND_URL =
  process.env.E2E_FRONTEND_URL ?? "http://127.0.0.1:5273";
export const API_URL = process.env.E2E_API_URL ?? "http://127.0.0.1:8081";
export const FAKE_URL = process.env.E2E_FAKE_URL ?? "http://127.0.0.1:8090";

export const AUTH_STATE_PATH = "playwright/.auth/user.json";
export const REFRESH_COOKIE_NAME = "guardian_refresh_token";
export const BROWSER_SESSION_KEY = "guardian_browser_session";
export const LEGACY_REFRESH_KEY = "guardian_refresh_token";

export const FIXTURES = {
  adminMembershipId: "4611686018400000001",
  collectionItemHash: "100",
  collectionItemName: "Fatebringer",
  wishlistItemName: "Fatebringer",
  catalystName: "Sunshot Catalyst",
} as const;

export const EMPTY_STORAGE_STATE = { cookies: [], origins: [] } as const;
