import AxeBuilder from "@axe-core/playwright";
import type { Page, TestInfo } from "@playwright/test";
import { test, expect } from "./fixtures";
import { EMPTY_STORAGE_STATE, FIXTURES } from "./constants";
import { openCollectionDrawer } from "./helpers";

type Impact = "minor" | "moderate" | "serious" | "critical" | null;

async function scanPage(page: Page, testInfo: TestInfo, label: string) {
  const results = await new AxeBuilder({ page })
    // WCAG 2.2 inherits the 2.0/2.1 A/AA criteria; axe labels new 2.2
    // checks separately, so all generations are intentionally included.
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();

  const blocking = results.violations.filter((violation) =>
    (["serious", "critical"] satisfies Impact[]).includes(violation.impact),
  );
  const advisory = results.violations.filter((violation) =>
    (["minor", "moderate", null] satisfies Impact[]).includes(violation.impact),
  );

  await testInfo.attach(`axe-${label}-advisories`, {
    body: JSON.stringify(advisory, null, 2),
    contentType: "application/json",
  });
  if (advisory.length > 0) {
    testInfo.annotations.push({
      type: "axe-advisory",
      description: `${label}: ${advisory.length} moderate/minor finding(s); see attachment`,
    });
  }

  expect(
    blocking,
    `${label} has critical/serious axe violations:\n${JSON.stringify(blocking, null, 2)}`,
  ).toEqual([]);
}

test.describe("unauthenticated accessibility", () => {
  test.use({ storageState: EMPTY_STORAGE_STATE });

  test("Login is WCAG 2.2 A/AA clean and keyboard reachable", async ({
    page,
  }, testInfo) => {
    await page.goto("/login");
    const signIn = page.getByRole("button", { name: "Sign in with Bungie" });
    await expect(signIn).toBeVisible();

    await page.keyboard.press("Tab");
    await expect(signIn).toBeFocused();
    await scanPage(page, testInfo, "login");
  });
});

// Axe results must not depend on which functional test happened to run last.
// The scenario lives in the fake-Bungie process and outlives a single test, so
// every authenticated scan pins it to the fully-populated week.
test.beforeEach(async ({ scenario }) => {
  await scenario.weekly("normal");
});

const authenticatedPages = [
  ["dashboard", "/dashboard", /Welcome,/],
  ["collections", "/collections", "Collections"],
  ["cosmetics", "/cosmetics", "Cosmetics"],
  ["wishlist", "/wishlist", "Wishlist"],
  ["this-week", "/this-week", "This Week"],
  ["catalysts", "/catalysts", "Catalysts & Crafting"],
  ["triumphs", "/triumphs", "Triumphs & Seals"],
  ["settings", "/settings", "Settings"],
  ["admin", "/admin", /Admin Console/],
] as const;

for (const [label, path, heading] of authenticatedPages) {
  test(`${label} is WCAG 2.2 A/AA clean`, async ({ page }, testInfo) => {
    await page.goto(path);
    await expect(page.getByRole("heading", { name: heading })).toBeVisible();
    await scanPage(page, testInfo, label);
  });
}

test("Crafting is WCAG 2.2 A/AA clean", async ({ page }, testInfo) => {
  await page.goto("/catalysts");
  const crafting = page.getByRole("button", { name: "Crafting Patterns" });
  await crafting.click();
  await expect(crafting).toHaveAttribute("data-on", "true");
  await scanPage(page, testInfo, "crafting");
});

test("the item drawer has a name, keyboard close, focus, and no blocking axe findings", async ({
  page,
}, testInfo) => {
  const drawer = await openCollectionDrawer(page);
  const close = drawer.getByRole("button", { name: "Close" });
  await close.focus();
  await expect(close).toBeFocused();
  await expect(drawer).toHaveAccessibleName(FIXTURES.collectionItemName);
  await scanPage(page, testInfo, "item-drawer");

  await page.keyboard.press("Escape");
  await expect(drawer).toBeHidden();
});

test("keyboard disclosures and reduced-motion preference are active", async ({
  page,
}) => {
  await page.goto("/triumphs");
  const seal = page.locator(".gt-seal").first();
  const disclosure = seal.locator(".gt-seal-head");
  await expect(seal).toHaveAttribute("data-expanded", "true");
  await disclosure.focus();
  await expect(disclosure).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(seal).toHaveAttribute("data-expanded", "false");
  await page.keyboard.press("Enter");
  await expect(seal).toHaveAttribute("data-expanded", "true");
  if ((await disclosure.getAttribute("aria-expanded")) !== null) {
    await expect(disclosure).toHaveAttribute("aria-expanded", "true");
  }

  expect(
    await page.evaluate(
      () => window.matchMedia("(prefers-reduced-motion: reduce)").matches,
    ),
  ).toBe(true);
});

test("the Collections category search has an accessible name and no blocking axe findings while active", async ({
  page,
}, testInfo) => {
  await page.goto("/collections?node=10");
  const search = page.getByRole("searchbox", {
    name: "Search this category…",
  });
  await expect(search).toBeVisible();
  await expect(search).toHaveAccessibleName("Search this category…");

  await search.focus();
  await expect(search).toBeFocused();
  await search.fill("Fate");
  await scanPage(page, testInfo, "collections-search-active");
});

// Mirrors the item-drawer scan above: opens an interactive disclosure by
// keyboard, confirms it has an accessible name, then scans the expanded state.
test("an expanded triumph disclosure has an accessible name and no blocking axe findings", async ({
  page,
}, testInfo) => {
  await page.goto("/triumphs");
  const toggle = page.getByRole("button", { name: /Tested Resolve/ });
  await expect(toggle).toBeVisible();
  await expect(toggle).toHaveAccessibleName(/Tested Resolve/);

  await toggle.focus();
  await expect(toggle).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(toggle).toHaveAttribute("aria-expanded", "true");

  await scanPage(page, testInfo, "triumph-disclosure-expanded");
});
