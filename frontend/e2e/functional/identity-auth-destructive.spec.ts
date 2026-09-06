import type { APIGuardianTrackerUser, WishListItem } from "../../src/types/api";
import { test, expect } from "../fixtures";
import { API_URL } from "../constants";
import { loginThroughOAuth, completeOnboarding } from "../helpers";

const privateItem: WishListItem = {
  id: "identity-fixture-a",
  itemHash: 100,
  name: "Account A private item",
  itemType: "Hand Cannon",
  rarity: "Legendary",
  icon: "",
  priority: "HIGH",
  notes: "Private note from A",
  acquisitionSources: [],
  availableNow: false,
  dateAdded: "2026-07-18T18:00:00Z",
};
const replacementItem = {
  ...privateItem,
  id: "identity-fixture-b",
  name: "Account B private item",
  notes: "Private note from B",
};
const replacementUser: APIGuardianTrackerUser = {
  id: "4611686018400000002",
  membershipId: "4611686018400000002",
  membershipType: 3,
  displayName: "Second Guardian",
  platform: "steam",
};

// Real login establishes the first session. Controlled callback/data responses
// isolate browser identity adoption from Bungie fixture account selection.
test("both tabs discard private data and late rollback after identity replacement", async ({
  page,
  context,
}) => {
  await loginThroughOAuth(page);
  await completeOnboarding(page);
  let releaseDelete!: () => void;
  let deleteStarted!: () => void;
  let deleteFinished!: () => void;
  const deletePending = new Promise<void>((resolve) => {
    releaseDelete = resolve;
  });
  const started = new Promise<void>((resolve) => {
    deleteStarted = resolve;
  });
  const finished = new Promise<void>((resolve) => {
    deleteFinished = resolve;
  });
  let releaseReplacement!: () => void;
  const replacementPending = new Promise<void>((resolve) => {
    releaseReplacement = resolve;
  });
  let replacementReads = 0;
  await page.evaluate(() =>
    localStorage.setItem("gt_done:identity-test", '["account-a-action"]'),
  );

  await context.route(`${API_URL}/api/**`, async (route) => {
    const request = route.request();
    const pathname = new URL(request.url()).pathname;
    if (pathname === "/api/auth/bungie/callback") {
      await route.fulfill({
        json: { token: "identity-replacement-b", user: replacementUser },
      });
      return;
    }
    if (
      pathname === `/api/wishlist/${privateItem.id}` &&
      request.method() === "DELETE"
    ) {
      deleteStarted();
      await deletePending;
      await route.fulfill({ status: 500, json: { error: "Delayed failure" } });
      deleteFinished();
      return;
    }
    const replacement =
      request.headers().authorization === "Bearer identity-replacement-b";
    if (pathname === "/api/wishlist") {
      if (replacement) {
        replacementReads += 1;
        await replacementPending;
      }
      await route.fulfill({
        json: [replacement ? replacementItem : privateItem],
      });
      return;
    }
    if (replacement && pathname === "/api/preferences") {
      await route.fulfill({
        json: {
          cardStyle: "framed",
          personalize: true,
          onboardedAt: "2026-07-18T18:00:00Z",
        },
      });
      return;
    }
    if (replacement && pathname === "/api/flags") {
      await route.fulfill({ json: { role: "standard", flags: [] } });
      return;
    }
    if (replacement && pathname.startsWith("/api/characters/")) {
      await route.fulfill({ json: [] });
      return;
    }
    await route.continue();
  });
  await page.goto("/wishlist");
  await expect(page.getByText('"Private note from A"')).toBeVisible();
  const second = await context.newPage();
  await second.goto("/wishlist");
  await expect(second.getByText('"Private note from A"')).toBeVisible();
  await page
    .locator(".gt-wl-item", { hasText: privateItem.name })
    .getByRole("button", { name: "Remove", exact: true })
    .click();
  await started;

  await second.evaluate(async (modulePath) => {
    const { browserSessionClient } = (await import(
      modulePath
    )) as typeof import("../../src/lib/browserSessionBrowser");
    await browserSessionClient.completeAuthorization({
      code: "fixture-b",
      state: "fixture-b-state",
    });
  }, "/src/lib/browserSessionBrowser.ts");
  await expect.poll(() => replacementReads).toBe(2);
  for (const tab of [page, second]) {
    await expect(tab.getByText('"Private note from A"')).toHaveCount(0);
    await expect(tab.getByText(privateItem.name, { exact: true })).toHaveCount(
      0,
    );
  }
  expect(
    await page.evaluate(() => localStorage.getItem("gt_done:identity-test")),
  ).toBeNull();
  releaseReplacement();
  for (const tab of [page, second]) {
    await expect(tab.getByText('"Private note from B"')).toBeVisible();
  }
  const failedResponse = page.waitForResponse(
    (response) =>
      response.url() === `${API_URL}/api/wishlist/${privateItem.id}` &&
      response.request().method() === "DELETE",
  );
  releaseDelete();
  await finished;
  await (await failedResponse).finished();
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
      }),
  );
  for (const tab of [page, second]) {
    await expect(tab.getByText('"Private note from A"')).toHaveCount(0);
    await expect(tab.getByText('"Private note from B"')).toBeVisible();
  }
  await second.close();
});

test("same-membership refresh preserves private cache and unsaved editor state", async ({
  page,
}) => {
  await loginThroughOAuth(page);
  await completeOnboarding(page);
  let reads = 0;
  await page.route(`${API_URL}/api/wishlist`, async (route) => {
    reads += 1;
    await route.fulfill({ json: [privateItem] });
  });
  await page.goto("/wishlist");
  await page.getByRole("button", { name: "Edit notes" }).click();
  const editor = page.getByRole("textbox", {
    name: `Notes for ${privateItem.name}`,
  });
  await editor.fill("Unsaved note survives rotation");
  await page.evaluate(() =>
    localStorage.setItem("gt_done:identity-test", '["account-a-action"]'),
  );
  let first = true;
  await page.route(`${API_URL}/api/auth/profile`, async (route) => {
    if (first) {
      first = false;
      await route.fulfill({ status: 401, json: { error: "Expired" } });
    } else await route.continue();
  });
  await page.evaluate(async (modulePath) => {
    const { apiFetch } = (await import(
      modulePath
    )) as typeof import("../../src/lib/api");
    await apiFetch("/api/auth/profile");
  }, "/src/lib/api.ts");
  await expect(editor).toHaveValue("Unsaved note survives rotation");
  expect(reads).toBe(1);
  expect(
    await page.evaluate(() => localStorage.getItem("gt_done:identity-test")),
  ).toBe('["account-a-action"]');
});

test("logout removes the private view from both tabs", async ({
  page,
  context,
}) => {
  await loginThroughOAuth(page);
  await completeOnboarding(page);
  await context.route(`${API_URL}/api/wishlist`, (route) =>
    route.fulfill({ json: [privateItem] }),
  );
  await page.goto("/wishlist");
  const second = await context.newPage();
  await second.goto("/wishlist");
  for (const tab of [page, second]) {
    await expect(tab.getByText('"Private note from A"')).toBeVisible();
  }
  await second.evaluate(async (modulePath) => {
    const { browserSessionClient } = (await import(
      modulePath
    )) as typeof import("../../src/lib/browserSessionBrowser");
    await browserSessionClient.end("current");
  }, "/src/lib/browserSessionBrowser.ts");
  for (const tab of [page, second]) {
    await expect(tab).toHaveURL(/\/login$/);
    await expect(tab.getByText('"Private note from A"')).toHaveCount(0);
  }
  await second.close();
});
