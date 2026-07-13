import { fileURLToPath } from "node:url";
import path from "node:path";
import { defineConfig, devices, type WebServerConfig } from "@playwright/test";

const frontendDir = path.dirname(fileURLToPath(import.meta.url));
const backendDir = path.resolve(frontendDir, "../backend/api-service");
const isCI = Boolean(process.env.CI);
const useExternalServers = process.env.E2E_EXTERNAL_SERVERS === "1";

const FRONTEND_URL = process.env.E2E_FRONTEND_URL ?? "http://127.0.0.1:5273";
const API_URL = process.env.E2E_API_URL ?? "http://127.0.0.1:8081";
const FAKE_URL = process.env.E2E_FAKE_URL ?? "http://127.0.0.1:8090";

const inheritedEnvironment = Object.fromEntries(
  Object.entries(process.env).filter(
    (entry): entry is [string, string] => entry[1] !== undefined,
  ),
);

const apiEnvironment: Record<string, string> = {
  ...inheritedEnvironment,
  GO_ENV: "development",
  LOG_LEVEL: "error",
  LOG_FORMAT: "text",
  BUNGIE_API_KEY: "e2e",
  BUNGIE_CLIENT_ID: "e2e-client",
  BUNGIE_CLIENT_SECRET: "e2e-client-secret",
  BUNGIE_AUTHORIZE_URL: `${FAKE_URL}/en/OAuth/Authorize`,
  BUNGIE_TOKEN_URL: `${FAKE_URL}/platform/app/oauth/token/`,
  BUNGIE_API_BASE_URL: `${FAKE_URL}/Platform`,
  BUNGIE_CDN_BASE_URL: FAKE_URL,
  AUTH_REDIRECT_URI: `${FRONTEND_URL}/auth/callback`,
  BUNGIE_REDIRECT_URI: `${FRONTEND_URL}/auth/callback`,
  CORS_ALLOWED_ORIGINS: FRONTEND_URL,
  FRONTEND_URL,
  DATABASE_URL:
    process.env.E2E_DATABASE_URL ??
    "postgres://guardian_app:guardian_dev_password@127.0.0.1:5534/guardian_tracker?sslmode=disable",
  JWT_SECRET: "e2e-jwt-secret-that-is-at-least-32-bytes-long",
  TOKEN_ENCRYPTION_KEY: "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=",
  TOKEN_ENCRYPTION_KEY_VERSION: "1",
  ADMIN_MEMBERSHIP_IDS: "4611686018400000001",
  CACHE_ENABLED: "false",
  E2E_FIXED_TIME: "2026-07-18T18:00:00Z",
  MANIFEST_DB_PATH: path.join(backendDir, ".e2e", "manifest.sqlite"),
  // Playwright starts every webServer in parallel, so the API's cold-start
  // manifest download can beat fake-Bungie to the port. That download retries
  // three times with no delay, then defers to the background updater — which
  // idles for MANIFEST_CHECK_INTERVAL (1h in production). A short interval lets
  // the API recover within the readiness probe instead of serving 503 forever.
  MANIFEST_CHECK_INTERVAL: "5s",
};

const webServer: WebServerConfig[] | undefined = useExternalServers
  ? undefined
  : [
      {
        command: process.env.E2E_FAKE_COMMAND ?? "go run ./cmd/fake-bungie",
        cwd: backendDir,
        env: {
          ...inheritedEnvironment,
          FAKE_BUNGIE_REDIRECT_URI: `${FRONTEND_URL}/auth/callback`,
          FAKE_BUNGIE_CLIENT_ID: "e2e-client",
        },
        url: `${FAKE_URL}/health`,
        reuseExistingServer: !isCI,
        timeout: 120_000,
      },
      {
        command: process.env.E2E_API_COMMAND ?? "go run .",
        cwd: backendDir,
        env: apiEnvironment,
        url: `${API_URL}/ready`,
        reuseExistingServer: !isCI,
        timeout: 120_000,
      },
      {
        command:
          process.env.E2E_WEB_COMMAND ??
          "npm run dev -- --host 127.0.0.1 --port 5273",
        cwd: frontendDir,
        env: {
          ...inheritedEnvironment,
          VITE_API_URL: API_URL,
        },
        url: FRONTEND_URL,
        reuseExistingServer: !isCI,
        timeout: 120_000,
      },
    ];

export default defineConfig({
  testDir: "./e2e",
  outputDir: "./test-results",
  snapshotPathTemplate: "{testDir}/__screenshots__/{arg}{ext}",
  fullyParallel: false,
  workers: 1,
  retries: isCI ? 1 : 0,
  forbidOnly: isCI,
  reporter: isCI
    ? [
        ["github"],
        ["html", { outputFolder: "playwright-report", open: "never" }],
      ]
    : [
        ["list"],
        ["html", { outputFolder: "playwright-report", open: "never" }],
      ],
  timeout: 45_000,
  expect: {
    timeout: 10_000,
    toHaveScreenshot: {
      animations: "disabled",
      maxDiffPixels: 100,
    },
  },
  use: {
    ...devices["Desktop Chrome"],
    baseURL: FRONTEND_URL,
    viewport: { width: 1440, height: 900 },
    timezoneId: "UTC",
    reducedMotion: "reduce",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  webServer,
  projects: [
    {
      name: "setup",
      testMatch: /auth\.setup\.ts/,
    },
    {
      name: "functional",
      testDir: "./e2e/functional",
      testIgnore: /auth-destructive\.spec\.ts/,
      dependencies: ["setup"],
      use: { storageState: "playwright/.auth/user.json" },
    },
    {
      name: "accessibility",
      testMatch: /accessibility\.spec\.ts/,
      dependencies: ["setup"],
      use: { storageState: "playwright/.auth/user.json" },
    },
    {
      name: "visual",
      testMatch: /visual\.spec\.ts/,
      dependencies: ["setup"],
      use: { storageState: "playwright/.auth/user.json" },
    },
    {
      name: "auth-destructive",
      testMatch: /auth-destructive\.spec\.ts/,
      dependencies: ["functional", "accessibility"],
      use: { storageState: { cookies: [], origins: [] } },
    },
  ],
});
