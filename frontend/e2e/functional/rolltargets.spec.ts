import { test, expect } from "../fixtures";
import { FIXTURES } from "../constants";
import { openCollectionDrawer } from "../helpers";

/**
 * Roll targets (slice 5b), behind the `god-roll` flag. The auth-setup fixture
 * user is admin (`ADMIN_MEMBERSHIP_IDS` in playwright.config.ts), which
 * bypasses the flag's alpha-tier gate.
 *
 * The fake-Bungie fixture's `/Profile` handler only recognises a fixed set of
 * `components` query strings (`800`, `200`, `200,205,300`, `204`, `900`); the
 * owned-roll read this feature needs uses `102,201,205,305`, which is not one
 * of them. Extending the fixture to serve instanced-item socket data is a
 * separate, larger change, so the match report reliably fails here — which is
 * itself a real, worth-covering state: once a target exists, it still renders
 * neutrally rather than the page crashing or showing nothing.
 *
 * One test walks the whole lifecycle (empty state -> import -> notes ->
 * delete) rather than several independent ones: roll targets persist in
 * Postgres for the life of the webServer, so a saved-then-re-imported
 * identical roll would report "already saved" instead of "imported" on a
 * second test, and the suite runs with `workers: 1` (fixtures mutate shared
 * state) so tests in this file already execute in file order.
 */
test("Roll targets: empty state, DIM import, management, and delete", async ({
  page,
}) => {
  await page.goto("/rolls");
  await expect(
    page.getByRole("heading", { name: "Roll targets" }),
  ).toBeVisible();
  await expect(page.getByText("Save the rolls you're chasing")).toBeVisible();

  await page
    .getByLabel("Paste a DIM-format wish list")
    .fill(`dimwishlist:item=${FIXTURES.collectionItemHash}&perks=11000,11002`);
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByText("1 imported")).toBeVisible();

  // The match report cannot succeed against this fixture (see above), so the
  // imported target renders in the neutral "match status unknown" list
  // rather than as "Still chasing", with a Retry banner.
  await expect(page.getByRole("button", { name: "Retry" })).toBeVisible();
  await expect(page.getByText("Match status unknown").first()).toBeVisible();

  const row = page
    .locator(".gt-rt-row")
    .filter({ hasText: FIXTURES.collectionItemName })
    .first();
  await expect(row).toBeVisible();
  await expect(row.getByText("Arrowhead Brake")).toBeVisible();
  await expect(row.getByText("Explosive Payload")).toBeVisible();

  await row.getByRole("button", { name: "Add notes" }).click();
  await row.getByRole("textbox", { name: /Notes for/ }).fill("E2E roll note");
  await row.getByRole("button", { name: "Save" }).click();
  await expect(row.getByText('"E2E roll note"')).toBeVisible();

  await row.getByRole("button", { name: "Delete" }).click();
  await expect(row).toBeHidden();
});

/**
 * Roll targets slice 6 (#371): the item detail drawer's per-weapon
 * roll-target section. Uses a distinct perk pair from the test above so this
 * test's import cannot collide with — and report "already saved" instead of
 * "imported" against — a target the previous test already deleted.
 *
 * Same fake-Bungie fixture gap as above: `/api/rolltargets/matches` fails, so
 * the drawer section falls back to its own neutral "match status unknown"
 * state (scoped to this weapon's own saved targets) with a Retry banner,
 * rather than "Copies you own that match" or "Still chasing".
 */
test("Roll targets: item drawer section lists the weapon's target neutrally when matching fails", async ({
  page,
}) => {
  await page.goto("/rolls");
  await page
    .getByLabel("Paste a DIM-format wish list")
    .fill(`dimwishlist:item=${FIXTURES.collectionItemHash}&perks=11001,11003`);
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByText("1 imported")).toBeVisible();

  const drawer = await openCollectionDrawer(page);
  await expect(
    drawer.getByRole("heading", { name: /Your roll targets/ }),
  ).toBeVisible();
  const neutralRow = drawer
    .locator(".gt-rt-row")
    .filter({ hasText: "Match status unknown" });
  await expect(neutralRow).toBeVisible();
  await expect(neutralRow.getByText("Corkscrew Rifling")).toBeVisible();
  await expect(neutralRow.getByText("Firefly")).toBeVisible();
  await expect(drawer.getByText("Still chasing")).toHaveCount(0);
  await expect(drawer.getByRole("button", { name: "Retry" })).toBeVisible();
  await expect(
    drawer.getByRole("link", { name: "Manage roll targets" }),
  ).toBeVisible();

  // Clean up so later runs against the same webServer don't see a stale
  // target for this weapon.
  await drawer.getByRole("button", { name: "Close" }).click();
  await page.goto("/rolls");
  const row = page
    .locator(".gt-rt-row")
    .filter({ hasText: FIXTURES.collectionItemName })
    .first();
  await row.getByRole("button", { name: "Delete" }).click();
  await expect(row).toBeHidden();
});
