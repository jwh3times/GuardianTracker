import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { chromium, expect, test } from "@playwright/test";

// Guardian Tracker is installable from its web manifest, with no service
// worker. Only full Chromium answers Page.getInstallabilityErrors: the default
// headless shell returns an empty list even for a missing or broken manifest,
// and a non-persistent context always adds "in-incognito".
test("the app is installable from its web manifest", async ({ baseURL }) => {
  const profile = mkdtempSync(path.join(tmpdir(), "gt-installability-"));
  const context = await chromium.launchPersistentContext(profile, {
    channel: "chromium",
  });
  try {
    const page = await context.newPage();
    await page.goto(new URL("/login", baseURL).toString());
    await expect(
      page.getByRole("button", { name: "Sign in with Bungie" }),
    ).toBeVisible();

    const cdp = await context.newCDPSession(page);
    const manifest = await cdp.send("Page.getAppManifest");
    expect(manifest.url).toMatch(/\/site\.webmanifest$/);
    expect(manifest.errors).toEqual([]);

    const { installabilityErrors } = await cdp.send(
      "Page.getInstallabilityErrors",
    );
    expect(installabilityErrors).toEqual([]);

    const themeColor = await page
      .locator('meta[name="theme-color"]')
      .getAttribute("content");
    const declared = JSON.parse(manifest.data ?? "{}") as {
      theme_color?: string;
    };
    expect(themeColor).toBe(declared.theme_color);
  } finally {
    await context.close();
    rmSync(profile, { recursive: true, force: true });
  }
});
