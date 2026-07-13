import { test, expect } from "../fixtures";
import { EMPTY_STORAGE_STATE } from "../constants";
import { expectCookieBackedAuth, loginThroughOAuth } from "../helpers";

test.describe("unauthenticated routing and OAuth", () => {
  test.use({ storageState: EMPTY_STORAGE_STATE });

  test("redirects a protected route to login", async ({ page }) => {
    await page.goto("/collections?item=100");
    await expect(page).toHaveURL(/\/login$/);
    await expect(
      page.getByRole("button", { name: "Sign in with Bungie" }),
    ).toBeVisible();
  });

  test("authenticates through the complete fake Bungie OAuth flow", async ({
    page,
  }) => {
    const callbackResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith("/api/auth/bungie/callback") &&
        response.request().method() === "POST",
    );
    await loginThroughOAuth(page);
    const callbackBody = (await (await callbackResponse).json()) as Record<
      string,
      unknown
    >;
    expect(callbackBody).toHaveProperty("token");
    expect(callbackBody).toHaveProperty("user");
    expect(callbackBody).not.toHaveProperty("refreshToken");
    await expectCookieBackedAuth(page);
  });
});

test("restores an authenticated protected route from storage state", async ({
  page,
}) => {
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/dashboard$/);
  await expect(page.getByRole("heading", { name: /Welcome,/ })).toBeVisible();
});
