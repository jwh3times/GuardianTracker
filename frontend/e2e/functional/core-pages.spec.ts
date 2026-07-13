import { test, expect } from "../fixtures";
import { FIXTURES } from "../constants";
import { openCollectionDrawer } from "../helpers";

test("Dashboard and Collections render deterministic account data", async ({
  page,
}) => {
  await page.goto("/dashboard");
  await expect(page.getByRole("heading", { name: /Welcome,/ })).toBeVisible();
  await expect(page.getByText("Do this today")).toBeVisible();

  const drawer = await openCollectionDrawer(page);
  await expect
    .poll(() => {
      const url = new URL(page.url());
      return {
        hasNode: url.searchParams.has("node"),
        hasItem: url.searchParams.has("item"),
      };
    })
    .toEqual({ hasNode: true, hasItem: false });
  await expect(drawer.getByText(FIXTURES.collectionItemName)).toBeVisible();
  await drawer.getByRole("button", { name: "Close" }).click();
  await expect(drawer).toBeHidden();
});

test("Catalysts, Crafting, Triumphs, Settings, and Cosmetics load", async ({
  page,
}) => {
  await page.goto("/catalysts");
  await expect(
    page.getByRole("heading", { name: "Catalysts & Crafting" }),
  ).toBeVisible();
  await expect(page.getByText(FIXTURES.catalystName)).toBeVisible();

  await page.getByRole("button", { name: "Crafting Patterns" }).click();
  await expect(
    page.getByRole("button", { name: "Crafting Patterns" }),
  ).toHaveAttribute("data-on", "true");

  await page.goto("/triumphs");
  await expect(
    page.getByRole("heading", { name: "Triumphs & Seals" }),
  ).toBeVisible();
  await expect(page.locator(".gt-seal").first()).toBeVisible();

  await page.goto("/settings");
  await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Sign out all devices" }),
  ).toBeVisible();

  await page.goto("/cosmetics");
  await expect(page.getByRole("heading", { name: "Cosmetics" })).toBeVisible();
});

test("an admin feature flag can be changed and restored", async ({ page }) => {
  await page.goto("/admin");
  await expect(
    page.getByRole("heading", { name: /Admin Console/ }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Feature Flags" }).click();

  const toggle = page.getByRole("switch").first();
  await expect(toggle).toBeVisible();
  const original = await toggle.getAttribute("aria-checked");
  expect(original).not.toBeNull();
  const changed = original === "true" ? "false" : "true";

  try {
    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-checked", changed);
  } finally {
    if ((await toggle.getAttribute("aria-checked")) !== original) {
      await toggle.click();
    }
    await expect(toggle).toHaveAttribute("aria-checked", original ?? "false");
  }
});

test("Collections composes category, filter, view, and deep-link URL state", async ({
  page,
}) => {
  await page.goto(
    `/collections?node=11&avail=1&item=${FIXTURES.collectionItemHash}`,
  );
  const drawer = page.getByRole("dialog", {
    name: FIXTURES.collectionItemName,
  });
  await expect(drawer).toBeVisible();
  await expect
    .poll(() => {
      const params = new URL(page.url()).searchParams;
      return {
        node: params.get("node"),
        avail: params.get("avail"),
        item: params.get("item"),
      };
    })
    .toEqual({ node: "11", avail: "1", item: null });

  await drawer.getByRole("button", { name: "Close" }).click();
  await expect(
    page.getByRole("button", { name: "Available now" }),
  ).toHaveAttribute("data-on", "true");

  const firstCategory = page.locator('[role="treeitem"] .gt-tree-main').first();
  await firstCategory.click();
  await expect
    .poll(() => new URL(page.url()).searchParams.has("node"))
    .toBe(true);

  await page.getByRole("button", { name: "List" }).click();
  await expect(page.getByRole("button", { name: "List" })).toHaveAttribute(
    "data-on",
    "true",
  );
  await page.getByRole("button", { name: "Hide farm-only" }).click();
  await expect(
    page.getByRole("button", { name: "Hide farm-only" }),
  ).toHaveAttribute("data-on", "true");
});

// The Weapons category (node 10) aggregates leaves from three child nodes:
// Fatebringer (missing, Legendary), Midnight Coup (collected, Legendary),
// Gjallarhorn (missing, Exotic), and Bold Endings (missing, Legendary). The
// default "missing only" filter hides Midnight Coup, leaving three cards.
// Item names render as plain text in the grid (only the list/compact
// densities carry a per-item accessible name), so cards are asserted by name
// text scoped to the grid rather than by role.
test("Collections category search filters the grid and updates the URL", async ({
  page,
}) => {
  await page.goto("/collections?node=10");
  await expect(
    page.getByRole("heading", { name: "Collections" }),
  ).toBeVisible();
  const grid = page.locator(".gt-itemgrid");
  await expect(grid.getByText("Fatebringer", { exact: true })).toBeVisible();
  await expect(grid.getByText("Gjallarhorn", { exact: true })).toBeVisible();
  await expect(grid.getByText("Bold Endings", { exact: true })).toBeVisible();

  const search = page.getByRole("searchbox", {
    name: "Search this category…",
  });
  await search.fill("Fate");

  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("Fate");
  await expect(grid.getByText("Fatebringer", { exact: true })).toBeVisible();
  await expect(grid.getByText("Gjallarhorn", { exact: true })).toBeHidden();
  await expect(grid.getByText("Bold Endings", { exact: true })).toBeHidden();
  await expect(page.locator(".gt-coll-resultcount")).toHaveText("1 shown");
});

test("Collections search composes with an active rarity filter as an intersection, not a union", async ({
  page,
}) => {
  await page.goto("/collections?node=10");
  const grid = page.locator(".gt-itemgrid");
  await expect(grid.getByText("Gjallarhorn", { exact: true })).toBeVisible();

  // The Rarity filter dropdown and the Sort dropdown (default "rarity") both
  // render a button literally labelled "Rarity" — the filter control is the
  // first one in DOM order.
  await page
    .getByRole("button", { name: "Rarity", exact: true })
    .first()
    .click();
  await page.getByRole("button", { name: "Legendary" }).click();
  await expect(grid.getByText("Fatebringer", { exact: true })).toBeVisible();
  await expect(grid.getByText("Bold Endings", { exact: true })).toBeVisible();
  await expect(grid.getByText("Gjallarhorn", { exact: true })).toBeHidden();

  // "Gjall" matches only the Exotic Gjallarhorn, which the active Legendary
  // filter excludes. A union-based (buggy) composition would still surface
  // either the two Legendary cards (filter match) or Gjallarhorn (search
  // match); a correct intersection surfaces neither.
  await page
    .getByRole("searchbox", { name: "Search this category…" })
    .fill("Gjall");

  await expect(page.getByText('No items match "Gjall"')).toBeVisible();
  await expect(grid.getByText("Fatebringer", { exact: true })).toBeHidden();
  await expect(grid.getByText("Bold Endings", { exact: true })).toBeHidden();
  await expect(grid.getByText("Gjallarhorn", { exact: true })).toBeHidden();
});

test("a no-match category search names the term, and Clear filters restores the grid", async ({
  page,
}) => {
  await page.goto("/collections?node=10");
  const grid = page.locator(".gt-itemgrid");
  const search = page.getByRole("searchbox", {
    name: "Search this category…",
  });
  await search.fill("zzz-not-a-real-item");

  await expect(
    page.getByText('No items match "zzz-not-a-real-item"'),
  ).toBeVisible();

  await page.getByRole("button", { name: "Clear filters" }).click();

  await expect(search).toHaveValue("");
  await expect(grid.getByText("Fatebringer", { exact: true })).toBeVisible();
  await expect(grid.getByText("Gjallarhorn", { exact: true })).toBeVisible();
  await expect(grid.getByText("Bold Endings", { exact: true })).toBeVisible();
  await expect
    .poll(() => new URL(page.url()).searchParams.has("q"))
    .toBe(false);
});

test("typing in the category search does not push a history entry per keystroke", async ({
  page,
}) => {
  await page.goto("/dashboard");
  await expect(page.getByRole("heading", { name: /Welcome,/ })).toBeVisible();

  await page.goto("/collections?node=10");
  const search = page.getByRole("searchbox", {
    name: "Search this category…",
  });
  // Three discrete state changes standing in for three keystrokes, each
  // exercised via `fill` (rather than dispatched key events) so the
  // assertion isn't sensitive to real-keyboard-event timing.
  await search.fill("a");
  await search.fill("ab");
  await search.fill("abc");

  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("abc");

  // Every change used replace navigation, so one Back from the fully-typed
  // state must return directly to the prior full navigation (Dashboard), not
  // step through "a", "ab" as intermediate history entries.
  await page.goBack();
  await expect(page).toHaveURL(/\/dashboard(?:[?#]|$)/);
});

// Record 9301 ("Prestige Worldwide") is redeemed (state 1) but carries a
// deliberately stale, incomplete objective payload — the redeemed state must
// win. Record 9302 ("Tested Resolve") is still in progress and carries three
// raw objectives: one visible, one with visibility omitted (must render), and
// one explicitly hidden (must never render). Both hang off the Dredgen seal,
// which auto-expands on page load.
test("Triumphs disclosure shows redeemed-authoritative progress over a stale objective", async ({
  page,
}) => {
  await page.goto("/triumphs");
  await expect(page.getByText("Dredgen")).toBeVisible();

  const toggle = page.getByRole("button", { name: /Prestige Worldwide/ });
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect(toggle).toContainText("1/1");
  await expect(page.getByText("Triumph complete")).toBeHidden();

  await toggle.click();

  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByText("Triumph complete")).toBeVisible();
});

test("Triumphs multi-objective disclosure expands via keyboard and never renders an explicitly hidden objective", async ({
  page,
}) => {
  await page.goto("/triumphs");
  const toggle = page.getByRole("button", { name: /Tested Resolve/ });
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect(toggle).toContainText("1/2");
  await expect(page.getByText("Matches won")).toBeHidden();
  await expect(page.getByText("Opponents defeated")).toBeHidden();
  await expect(page.getByText("Hidden objective")).toBeHidden();

  await toggle.focus();
  await expect(toggle).toBeFocused();
  await page.keyboard.press("Enter");

  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByText("Matches won")).toBeVisible();
  await expect(page.getByText("Opponents defeated")).toBeVisible();
  await expect(page.getByText("Hidden objective")).toBeHidden();
});
