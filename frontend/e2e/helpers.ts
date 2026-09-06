import { expect, type APIRequestContext, type Page } from "@playwright/test";
import {
  BROWSER_SESSION_KEY,
  API_URL,
  FAKE_URL,
  FIXTURES,
  LEGACY_REFRESH_KEY,
  REFRESH_COOKIE_NAME,
} from "./constants";

export type WeeklyScenario = "normal" | "empty";

export async function setWeeklyScenario(
  request: APIRequestContext,
  weekly: WeeklyScenario,
) {
  const response = await request.put(`${FAKE_URL}/__e2e/scenario`, {
    data: { weekly },
  });
  expect(response.ok(), await response.text()).toBe(true);
}

export async function resetScenario(request: APIRequestContext) {
  const response = await request.delete(`${FAKE_URL}/__e2e/scenario`);
  expect(response.ok(), await response.text()).toBe(true);
}

export async function loginThroughOAuth(page: Page) {
  await page.goto("/login");
  await page.getByRole("button", { name: "Sign in with Bungie" }).click();

  // The hermetic fake authorizes immediately. Keeping this optional click makes
  // the helper compatible with a deliberately interactive fake consent screen.
  const consent = page.getByRole("button", {
    name: /authorize|approve|allow/i,
  });
  await consent.click({ timeout: 1_000 }).catch(() => undefined);

  await expect(page).toHaveURL(/\/dashboard(?:[?#]|$)/, { timeout: 30_000 });
  await expect(page.getByRole("heading", { name: /Welcome,/ })).toBeVisible();
}

// A fresh fixture account has no onboarded_at preference, so OnboardingTour
// mounts a full-screen scrim that intercepts pointer events on every route.
// Completing it persists onboarded_at server-side, so the whole run stays clear.
// The e2e Postgres container outlives a single run, so a repeat run may find the
// tour already dismissed — treat its absence as success, not failure.
export async function completeOnboarding(page: Page) {
  const skip = page.getByRole("button", { name: "Skip tour" });
  try {
    await skip.waitFor({ state: "visible", timeout: 10_000 });
  } catch {
    return;
  }
  await skip.click();
  await expect(page.locator(".gt-tour-scrim")).toHaveCount(0);
  await expect(page.locator(".gt-tour-layer")).toHaveCount(0);
}

export async function readBrowserAuth(page: Page) {
  return page.evaluate(
    ({ sessionKey, legacyRefreshKey }) => {
      const envelope = JSON.parse(localStorage.getItem(sessionKey) ?? "null");
      const projection = envelope?.projection;
      return {
        token:
          projection?.status === "authenticated"
            ? projection.accessToken
            : null,
        user:
          projection?.status === "authenticated"
            ? JSON.stringify(projection.user)
            : null,
        legacyRefresh: localStorage.getItem(legacyRefreshKey),
      };
    },
    {
      sessionKey: BROWSER_SESSION_KEY,
      legacyRefreshKey: LEGACY_REFRESH_KEY,
    },
  );
}

export async function expectCookieBackedAuth(page: Page) {
  const storage = await readBrowserAuth(page);
  expect(storage.token).toBeTruthy();
  expect(storage.user).toBeTruthy();
  expect(storage.legacyRefresh).toBeNull();

  const user = JSON.parse(storage.user ?? "{}") as { membershipId?: string };
  expect(user.membershipId).toBe(FIXTURES.adminMembershipId);

  const cookies = await page.context().cookies(`${API_URL}/api/auth/refresh`);
  const refresh = cookies.find((cookie) => cookie.name === REFRESH_COOKIE_NAME);
  expect(refresh).toBeDefined();
  expect(refresh).toMatchObject({
    httpOnly: true,
    sameSite: "Lax",
    path: "/api/auth",
    secure: false,
  });
  expect(refresh?.value).toBeTruthy();
  return refresh;
}

export async function waitForSearchIndex(page: Page) {
  const { token } = await readBrowserAuth(page);
  expect(token).toBeTruthy();

  await expect
    .poll(
      async () => {
        const response = await page.request.get(
          `${API_URL}/api/items/search?q=${encodeURIComponent(FIXTURES.collectionItemName)}&limit=5`,
          { headers: { Authorization: `Bearer ${token}` } },
        );
        if (!response.ok()) return false;
        const results = (await response.json()) as Array<{ name?: string }>;
        return results.some(
          (item) => item.name === FIXTURES.collectionItemName,
        );
      },
      {
        message: "wait for the asynchronous manifest search index",
        timeout: 120_000,
        intervals: [250, 500, 1_000, 2_000],
      },
    )
    .toBe(true);
}

export async function blockThirdPartyAssets(page: Page) {
  await page.route("**/*", async (route) => {
    const request = route.request();
    const resourceType = request.resourceType();
    if (!new Set(["font", "image", "media", "stylesheet"]).has(resourceType)) {
      await route.continue();
      return;
    }

    const hostname = new URL(request.url()).hostname;
    if (hostname === "127.0.0.1" || hostname === "localhost") {
      await route.continue();
      return;
    }
    await route.abort("blockedbyclient");
  });
}

export function dynamicTextMasks(page: Page) {
  return [
    page.locator(".gt-chip-time"),
    page.locator(".gt-fresh"),
    page.locator(".gt-userrow-active"),
    page.locator("time"),
  ];
}

export async function openCollectionDrawer(page: Page) {
  await page.goto(`/collections?item=${FIXTURES.collectionItemHash}`);
  const drawer = page.getByRole("dialog", {
    name: FIXTURES.collectionItemName,
  });
  await expect(drawer).toBeVisible();
  return drawer;
}
