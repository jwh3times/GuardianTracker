import { test, expect } from "../fixtures";
import {
  ACCESS_TOKEN_KEY,
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
      ({ accessTokenKey, legacyRefreshKey }) => {
        localStorage.setItem(
          accessTokenKey,
          "deliberately-expired-access-token",
        );
        localStorage.removeItem(legacyRefreshKey);
      },
      {
        accessTokenKey: ACCESS_TOKEN_KEY,
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
