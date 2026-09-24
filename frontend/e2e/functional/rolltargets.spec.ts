import { test, expect } from "../fixtures";
import type { Page } from "@playwright/test";
import { FIXTURES } from "../constants";
import { openCollectionDrawer } from "../helpers";

async function deleteAllRollTargets(page: Page) {
  await page.getByRole("button", { name: "Delete all…" }).click();
  await page.getByRole("button", { name: "Yes, delete all" }).click();
  await expect(page.getByText("Save the rolls you're chasing")).toBeVisible();
}

/**
 * Roll targets (slice 5b), behind the `god-roll` flag. The auth-setup fixture
 * user is admin (`ADMIN_MEMBERSHIP_IDS` in playwright.config.ts), which
 * bypasses the flag's alpha-tier gate.
 *
 * The fake-Bungie fixture serves the owned Fatebringer through the same
 * `102,201,205,305` profile-component read as production. Its current socket
 * uses an enhanced Arrowhead Brake plug, proving that owned rolls resolve base
 * and enhanced variants to the same display name before matching.
 *
 * One test walks the whole lifecycle (empty state -> import -> notes ->
 * delete) rather than several independent ones: roll targets persist in
 * Postgres for the life of the webServer, so a saved-then-re-imported
 * identical roll would report "already saved" instead of "imported" on a
 * second test, and the suite runs with `workers: 1` (fixtures mutate shared
 * state) so tests in this file already execute in file order.
 */
test("Roll targets: wanted, unmatched, and unwanted matches through the real stack", async ({
  page,
}) => {
  const collectionDrawer = await openCollectionDrawer(page);
  const addToWishlist = collectionDrawer.getByRole("button", {
    name: "Add to Wishlist",
  });
  // The broader fixture deliberately leaves Fatebringer's collectible missing;
  // the wish list makes this target eligible for the default page filter.
  if (await addToWishlist.isVisible()) {
    await addToWishlist.click();
    await expect(
      collectionDrawer.getByRole("button", { name: "On wishlist" }),
    ).toBeVisible();
  }

  await page.goto("/rolls");
  await expect(
    page.getByRole("heading", { name: "Roll targets" }),
  ).toBeVisible();
  await expect(page.getByText("Save the rolls you're chasing")).toBeVisible();

  await page
    .getByLabel("Paste a DIM-format wish list")
    .fill(
      [
        `dimwishlist:item=${FIXTURES.collectionItemHash}&perks=11000,11002`,
        `dimwishlist:item=${FIXTURES.collectionItemHash}&perks=11001,11002`,
        `dimwishlist:item=-${FIXTURES.collectionItemHash}&perks=11000,11002`,
      ].join("\n"),
    );
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByText("3 imported")).toBeVisible();

  const wanted = page.locator('section[aria-labelledby="rt-wanted-title"]');
  await expect(wanted.getByText(FIXTURES.collectionItemName)).toBeVisible();
  await expect(wanted.getByText("Copy 1")).toBeVisible();
  await expect(wanted.getByText("Arrowhead Brake").first()).toBeVisible();
  await expect(wanted.getByText("Explosive Payload").first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Retry" })).toHaveCount(0);

  const chasing = page.locator('section[aria-labelledby="rt-chasing-title"]');
  await expect(chasing.getByText("Corkscrew Rifling")).toBeVisible();
  await expect(
    chasing.getByText("Your best copy has 1 of 2 target perks."),
  ).toBeVisible();

  const unwanted = page.locator('section[aria-labelledby="rt-unwanted-title"]');
  const unwantedToggle = unwanted.getByRole("button", {
    name: "Matches a roll you marked unwanted (1)",
  });
  await expect(unwantedToggle).toHaveAttribute("aria-expanded", "false");
  await expect(unwanted.locator(".gt-rt-copy")).toHaveCount(0);
  await unwantedToggle.click();
  await expect(unwanted.getByText("Copy 1")).toBeVisible();
  await expect(unwanted.getByText("Arrowhead Brake").first()).toBeVisible();
  await expect(unwanted.getByText("Explosive Payload").first()).toBeVisible();

  const row = wanted
    .locator(".gt-rt-row")
    .filter({ hasText: FIXTURES.collectionItemName })
    .first();
  await expect(row).toBeVisible();
  await expect(row.getByText("Arrowhead Brake").first()).toBeVisible();
  await expect(row.getByText("Explosive Payload").first()).toBeVisible();

  await row.getByRole("button", { name: "Add notes" }).click();
  await row.getByRole("textbox", { name: /Notes for/ }).fill("E2E roll note");
  await row.getByRole("button", { name: "Save" }).click();
  await expect(row.getByText('"E2E roll note"')).toBeVisible();

  await deleteAllRollTargets(page);

  await page.goto("/wishlist");
  const wishListItem = page.locator(".gt-wl-item", {
    hasText: FIXTURES.wishlistItemName,
  });
  await wishListItem.getByRole("button", { name: "Remove" }).click();
  await expect(wishListItem).toBeHidden();
});

/**
 * Roll targets slice 6 (#371): the item detail drawer's per-weapon
 * roll-target section. Uses a distinct perk pair from the test above so this
 * test's import cannot collide with — and report "already saved" instead of
 * "imported" against — a target the previous test already deleted.
 *
 * The owned copy carries Explosive Payload but not this target's Corkscrew
 * Rifling, so the successful match report places it under "Still chasing" and
 * names it as the one-of-two best copy.
 */
test("Roll targets: item drawer section shows an unmatched target as still chasing", async ({
  page,
}) => {
  await page.goto("/rolls");
  await page
    .getByLabel("Paste a DIM-format wish list")
    .fill(`dimwishlist:item=${FIXTURES.collectionItemHash}&perks=11001,11002`);
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByText("1 imported")).toBeVisible();

  const drawer = await openCollectionDrawer(page);
  await expect(
    drawer.getByRole("heading", { name: /Your roll targets/ }),
  ).toBeVisible();
  const chasingRow = drawer
    .locator(".gt-rt-row")
    .filter({ hasText: "Corkscrew Rifling" });
  await expect(drawer.getByText("Still chasing")).toBeVisible();
  await expect(chasingRow).toBeVisible();
  await expect(chasingRow.getByText("Corkscrew Rifling")).toBeVisible();
  await expect(
    chasingRow.getByText("Your best copy has 1 of 2 target perks."),
  ).toBeVisible();
  await expect(drawer.getByText("Match status unknown")).toHaveCount(0);
  await expect(drawer.getByRole("button", { name: "Retry" })).toHaveCount(0);
  await expect(
    drawer.getByRole("link", { name: "Manage roll targets" }),
  ).toBeVisible();

  // Clean up so later runs against the same webServer don't see a stale
  // target for this weapon.
  await drawer.getByRole("button", { name: "Close" }).click();
  await page.goto("/rolls");
  await deleteAllRollTargets(page);
});
