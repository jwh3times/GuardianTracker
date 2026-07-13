import { test, expect } from "./fixtures";
import { FIXTURES } from "./constants";
import {
  blockThirdPartyAssets,
  dynamicTextMasks,
  openCollectionDrawer,
} from "./helpers";

test.beforeEach(async ({ page, scenario }) => {
  await blockThirdPartyAssets(page);
  await scenario.weekly("normal");
});

async function settle(page: Parameters<typeof blockThirdPartyAssets>[0]) {
  await page.evaluate(async () => {
    await document.fonts.ready;
  });
}

async function capture(
  page: Parameters<typeof blockThirdPartyAssets>[0],
  name: string,
) {
  await settle(page);
  await expect(page).toHaveScreenshot(name, {
    fullPage: true,
    mask: dynamicTextMasks(page),
  });
}

test("Dashboard", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page.getByRole("heading", { name: /Welcome,/ })).toBeVisible();
  await capture(page, "dashboard.png");
});

test("Collections", async ({ page }) => {
  await page.goto("/collections");
  await expect(
    page.getByRole("heading", { name: "Collections" }),
  ).toBeVisible();
  await capture(page, "collections.png");
});

test("item drawer", async ({ page }) => {
  await openCollectionDrawer(page);
  await capture(page, "item-drawer.png");
});

test("Wishlist", async ({ page }) => {
  const drawer = await openCollectionDrawer(page);
  const add = drawer.getByRole("button", { name: "Add to Wishlist" });
  if (await add.isVisible()) await add.click();
  await page.goto("/wishlist");
  const item = page.locator(".gt-wl-item", {
    hasText: FIXTURES.wishlistItemName,
  });
  try {
    await expect(item).toBeVisible();
    await capture(page, "wishlist.png");
  } finally {
    if (await item.isVisible()) {
      await item.getByRole("button", { name: "Remove" }).click();
      await expect(item).toBeHidden();
    }
  }
});

test("Xur", async ({ page }) => {
  await page.goto("/this-week");
  await expect(page.getByText("Xûr — weekend exotics")).toBeVisible();
  await capture(page, "xur.png");
});

test("Catalysts", async ({ page }) => {
  await page.goto("/catalysts");
  await expect(page.getByText(FIXTURES.catalystName)).toBeVisible();
  await capture(page, "catalysts.png");
});

test("Crafting", async ({ page }) => {
  await page.goto("/catalysts");
  const crafting = page.getByRole("button", { name: "Crafting Patterns" });
  await crafting.click();
  await expect(crafting).toHaveAttribute("data-on", "true");
  await capture(page, "crafting.png");
});

test("Triumphs", async ({ page }) => {
  await page.goto("/triumphs");
  await expect(page.locator(".gt-seal").first()).toBeVisible();
  await capture(page, "triumphs.png");
});
