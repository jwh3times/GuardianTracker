import { test, expect } from "../fixtures";
import { FIXTURES } from "../constants";
import { openCollectionDrawer } from "../helpers";

test("wishlist supports create, read, update, and delete", async ({ page }) => {
  const drawer = await openCollectionDrawer(page);
  const add = drawer.getByRole("button", { name: "Add to Wishlist" });
  if (await add.isVisible()) await add.click();

  await page.goto("/wishlist");
  const item = page.locator(".gt-wl-item", {
    hasText: FIXTURES.wishlistItemName,
  });
  await expect(item).toBeVisible();

  const priority = item.locator(".gt-dd").first();
  await priority.locator(".gt-fchip").click();
  await priority.getByRole("button", { name: "Urgent" }).click();
  await expect(priority.locator(".gt-fchip")).toHaveText(/Urgent/);

  await item.getByRole("button", { name: "Add notes" }).click();
  await item
    .getByRole("textbox", { name: `Notes for ${FIXTURES.wishlistItemName}` })
    .fill("E2E deterministic roll");
  await item.getByRole("button", { name: "Save" }).click();
  await expect(item.getByText('"E2E deterministic roll"')).toBeVisible();

  await item.getByRole("button", { name: "Remove" }).click();
  await expect(item).toBeHidden();
});

test("global search deep-links to the collection drawer", async ({ page }) => {
  await page.goto("/dashboard");
  const search = page.getByRole("searchbox", { name: "Search items" });
  await search.fill(FIXTURES.collectionItemName);
  const result = page
    .locator(".gt-search-menu")
    .getByRole("button", { name: new RegExp(FIXTURES.collectionItemName) });
  await expect(result).toBeVisible();
  await result.click();
  await expect(
    page.getByRole("dialog", { name: FIXTURES.collectionItemName }),
  ).toBeVisible();
});

test("This Week renders the normal Xur state", async ({ page, scenario }) => {
  await scenario.weekly("normal");
  await page.goto("/this-week");
  await expect(page.getByRole("heading", { name: "This Week" })).toBeVisible();
  await expect(page.getByText("Xûr — weekend exotics")).toBeVisible();
  await expect(page.locator(".gt-action").first()).toBeVisible();
});

test("This Week survives an empty live week from Bungie", async ({
  page,
  scenario,
}) => {
  await scenario.weekly("empty");
  await page.goto("/this-week");
  await expect(page.getByRole("heading", { name: "This Week" })).toBeVisible();

  // The empty scenario blanks Bungie's live feed: no milestones, no vendor sales.
  await expect(page.locator(".gt-milestone")).toHaveCount(0);
  await expect(page.locator(".gt-vendor-item")).toHaveCount(0);

  // Xûr's panel is driven by the weekend window, not by the sale data, and
  // E2E_FIXED_TIME is a Saturday — so he is still in town with nothing to sell.
  // "Xûr returns Friday" is unreachable under the fixed clock.
  await expect(page.getByText("Xûr — weekend exotics")).toBeVisible();

  // Recommendations come from the efficiency engine (manifest + missing items),
  // so they outlive an empty live week. An empty Bungie week is not an empty page.
  await expect(page.locator(".gt-action").first()).toBeVisible();
});
