import { test, expect } from "../fixtures";
import {
  BROWSER_SESSION_KEY,
  API_URL,
  LEGACY_REFRESH_KEY,
  REFRESH_COOKIE_NAME,
} from "../constants";
import {
  expectCookieBackedAuth,
  loginThroughOAuth,
  readBrowserAuth,
} from "../helpers";

test.describe.serial("cookie-backed session lifecycle", () => {
  test("automatically rotates the refresh cookie after an access-token 401", async ({
    page,
  }) => {
    await loginThroughOAuth(page);
    const before = await expectCookieBackedAuth(page);

    await page.evaluate(
      ({ sessionKey, legacyRefreshKey }) => {
        const envelope = JSON.parse(localStorage.getItem(sessionKey)!);
        envelope.projection.accessToken = "deliberately-expired-access-token";
        localStorage.setItem(sessionKey, JSON.stringify(envelope));
        localStorage.removeItem(legacyRefreshKey);
      },
      {
        sessionKey: BROWSER_SESSION_KEY,
        legacyRefreshKey: LEGACY_REFRESH_KEY,
      },
    );

    const refreshResponse = page.waitForResponse(
      (response) =>
        response.url() === `${API_URL}/api/auth/refresh` &&
        response.request().method() === "POST",
    );
    await page.goto("/collections");
    const response = await refreshResponse;
    expect(response.ok()).toBe(true);
    const refreshBody = (await response.json()) as Record<string, unknown>;
    expect(refreshBody).toHaveProperty("token");
    expect(refreshBody).toHaveProperty("user");
    expect(refreshBody).not.toHaveProperty("refreshToken");

    await expect
      .poll(async () => (await readBrowserAuth(page)).token)
      .not.toBe("deliberately-expired-access-token");
    const after = await expectCookieBackedAuth(page);
    expect(after?.value).not.toBe(before?.value);
  });

  test("two pages share one refresh rotation and replacement", async ({
    page,
    context,
  }) => {
    await loginThroughOAuth(page);
    const before = await expectCookieBackedAuth(page);
    const second = await context.newPage();
    await second.goto("/dashboard");
    await expect(
      second.getByRole("heading", { name: /Welcome,/ }),
    ).toBeVisible();

    let release!: () => void;
    const bothRequested = new Promise<void>((resolve) => {
      release = resolve;
    });
    let initialRequests = 0;
    let refreshes = 0;
    const retryTokens: string[] = [];
    context.on("request", (request) => {
      if (request.url() === `${API_URL}/api/auth/refresh`) refreshes += 1;
    });
    for (const tab of [page, second]) {
      let first = true;
      await tab.route(`${API_URL}/api/auth/profile`, async (route) => {
        if (first) {
          first = false;
          initialRequests += 1;
          if (initialRequests === 2) release();
          await bothRequested;
          await route.fulfill({ status: 401, json: { error: "Expired" } });
          return;
        }
        retryTokens.push(route.request().headers().authorization);
        await route.continue();
      });
    }
    await Promise.all(
      [page, second].map((tab) =>
        tab.evaluate(async (modulePath) => {
          const { apiFetch } = await import(modulePath);
          await apiFetch("/api/auth/profile");
        }, "/src/lib/api.ts"),
      ),
    );

    expect(refreshes).toBe(1);
    expect(retryTokens).toHaveLength(2);
    expect(retryTokens[0]).toBe(retryTokens[1]);
    const after = await expectCookieBackedAuth(page);
    expect(after?.value).not.toBe(before?.value);
    expect(retryTokens[0]).toBe(
      `Bearer ${(await readBrowserAuth(page)).token}`,
    );
    await second.close();
  });

  test("canceling a read preserves another page's refresh rotation", async ({
    page,
    context,
  }) => {
    await loginThroughOAuth(page);
    const before = await expectCookieBackedAuth(page);
    const second = await context.newPage();
    await second.goto("/dashboard");
    await expect(
      second.getByRole("heading", { name: /Welcome,/ }),
    ).toBeVisible();

    let releaseRefresh!: () => void;
    const heldRefresh = new Promise<void>((resolve) => {
      releaseRefresh = resolve;
    });
    let enteredRefresh!: () => void;
    const refreshStarted = new Promise<void>((resolve) => {
      enteredRefresh = resolve;
    });
    let enteredFollower!: () => void;
    const followerStarted = new Promise<void>((resolve) => {
      enteredFollower = resolve;
    });
    let refreshes = 0;
    const retries = [0, 0];
    await context.route(`${API_URL}/api/auth/refresh`, async (route) => {
      refreshes += 1;
      enteredRefresh();
      await heldRefresh;
      await route.continue();
    });
    for (const [index, tab] of [page, second].entries()) {
      let first = true;
      await tab.route(`${API_URL}/api/auth/profile`, async (route) => {
        if (first) {
          first = false;
          await route.fulfill({ status: 401, json: { error: "Expired" } });
          if (index === 1) enteredFollower();
          return;
        }
        retries[index] += 1;
        await route.continue();
      });
    }

    try {
      const abandoned = page.evaluate(async (modulePath) => {
        const { apiFetch } = (await import(
          modulePath
        )) as typeof import("../../src/lib/api");
        const controller = new AbortController();
        window.addEventListener("abort-test-read", () => controller.abort(), {
          once: true,
        });
        try {
          await apiFetch("/api/auth/profile", { signal: controller.signal });
          return "completed";
        } catch (error) {
          return error instanceof Error ? error.name : "unknown";
        }
      }, "/src/lib/api.ts");
      await refreshStarted;
      const surviving = second.evaluate(async (modulePath) => {
        const { apiFetch } = (await import(
          modulePath
        )) as typeof import("../../src/lib/api");
        await apiFetch("/api/auth/profile");
      }, "/src/lib/api.ts");
      await followerStarted;
      await page.evaluate(() =>
        window.dispatchEvent(new Event("abort-test-read")),
      );
      // This must settle before the rotating refresh is allowed to complete.
      expect(await abandoned).toBe("AbortError");
      expect(refreshes).toBe(1);
      releaseRefresh();
      await surviving;

      expect(refreshes).toBe(1);
      expect(retries).toEqual([0, 1]);
      const after = await expectCookieBackedAuth(page);
      expect(after?.value).not.toBe(before?.value);
      expect((await readBrowserAuth(second)).token).toBe(
        (await readBrowserAuth(page)).token,
      );
    } finally {
      releaseRefresh();
      await second.close();
    }
  });

  test("sign out expires the current refresh cookie", async ({ page }) => {
    await loginThroughOAuth(page);
    await expectCookieBackedAuth(page);
    await page.goto("/settings");
    await page.getByRole("button", { name: "Sign out", exact: true }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect
      .poll(async () =>
        (await page.context().cookies(`${API_URL}/api/auth/refresh`)).some(
          (cookie) => cookie.name === REFRESH_COOKIE_NAME,
        ),
      )
      .toBe(false);
  });

  test("sign out all devices expires the refresh cookie and clears browser auth", async ({
    page,
  }) => {
    await loginThroughOAuth(page);
    await expectCookieBackedAuth(page);
    await page.goto("/settings");
    await page.getByRole("button", { name: "Sign out all devices" }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect
      .poll(async () =>
        (await page.context().cookies(`${API_URL}/api/auth/refresh`)).some(
          (cookie) => cookie.name === REFRESH_COOKIE_NAME,
        ),
      )
      .toBe(false);
    expect(await readBrowserAuth(page)).toEqual({
      token: null,
      user: null,
      legacyRefresh: null,
    });
  });
});
