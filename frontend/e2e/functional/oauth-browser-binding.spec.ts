import type { Page } from "@playwright/test";
import { test, expect } from "../fixtures";
import { API_URL, EMPTY_STORAGE_STATE } from "../constants";
import {
  expectCookieBackedAuth,
  loginThroughOAuth,
  readBrowserAuth,
} from "../helpers";

// Stop before redemption so the same fresh provider response can be presented
// to independent browser cookie jars. The fake provider still issues the code.
async function pendingCallback(page: Page) {
  const authorization = await page.evaluate(async (apiUrl) => {
    const response = await fetch(`${apiUrl}/api/auth/bungie`, {
      credentials: "include",
    });
    if (!response.ok) throw new Error("Authorization start failed");
    return (await response.json()) as { authUrl: string };
  }, API_URL);
  const response = await page.request.get(authorization.authUrl, {
    maxRedirects: 0,
  });
  expect(response.status()).toBe(302);
  const location = response.headers().location;
  expect(location).toBeTruthy();
  return location;
}

async function rejectedCallback(page: Page, callback: string) {
  const response = page.waitForResponse(
    (result) =>
      result.url().endsWith("/api/auth/bungie/callback") &&
      result.request().method() === "POST",
  );
  await page.goto(callback);
  expect((await response).status()).toBe(400);
  await expect(
    page.getByRole("heading", { name: "Authentication error" }),
  ).toBeVisible();
  expect((await readBrowserAuth(page)).token).toBeNull();
}

test.describe("OAuth browser transaction binding", () => {
  test.use({ storageState: EMPTY_STORAGE_STATE });

  test("rejects another browser's callback and leaves both transactions usable", async ({
    page,
    browser,
  }) => {
    await page.goto("/login");
    const callback = await pendingCallback(page);
    const otherContext = await browser.newContext({
      storageState: EMPTY_STORAGE_STATE,
    });
    try {
      const otherPage = await otherContext.newPage();
      // An unsolicited callback has no transaction cookie at all.
      await rejectedCallback(otherPage, callback);
      const otherCallback = await pendingCallback(otherPage);
      // Starting an unrelated flow must not make the foreign state acceptable.
      await rejectedCallback(otherPage, callback);
      await otherPage.goto(otherCallback);
      await expect(otherPage).toHaveURL(/\/dashboard$/);
      await expectCookieBackedAuth(otherPage);
      // The original browser's pending transaction still succeeds. Backend
      // tests separately assert rejection happens before provider exchange;
      // this fixture intentionally uses a reusable authorization code.
      await page.goto(callback);
      await expect(page).toHaveURL(/\/dashboard$/);
      await expectCookieBackedAuth(page);
    } finally {
      await otherContext.close();
    }
  });

  test("accepts only the latest start and consumes its cookie across tabs", async ({
    page,
    context,
  }) => {
    await page.goto("/login");
    const first = await pendingCallback(page);
    const latest = await pendingCallback(page);
    await rejectedCallback(page, first);
    const callbackTab = await context.newPage();
    await callbackTab.goto(latest);
    await expect(callbackTab).toHaveURL(/\/dashboard$/);
    await expectCookieBackedAuth(callbackTab);
    const cookies = await context.cookies(API_URL);
    expect(
      cookies.some((cookie) => cookie.name.includes("oauth_transaction")),
    ).toBe(false);
  });

  test("reconnects Bungie without replacing the Guardian session", async ({
    page,
    context,
  }) => {
    await loginThroughOAuth(page);
    const originalAuth = await readBrowserAuth(page);
    const originalCookie = await expectCookieBackedAuth(page);
    await page.goto("/reauthorize");
    const reconnect = page.waitForResponse(
      (response) =>
        response.url().endsWith("/api/auth/bungie/reconnect") &&
        response.request().method() === "POST",
    );
    await page.getByRole("button", { name: "Reconnect Bungie" }).click();
    expect((await reconnect).status()).toBe(204);
    await expect(page).toHaveURL(/\/dashboard$/);
    expect(await readBrowserAuth(page)).toEqual(originalAuth);
    expect((await expectCookieBackedAuth(page))?.value).toBe(
      originalCookie?.value,
    );
    expect(
      (await context.cookies(API_URL)).some((cookie) =>
        cookie.name.includes("oauth_transaction"),
      ),
    ).toBe(false);
  });
});
