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
