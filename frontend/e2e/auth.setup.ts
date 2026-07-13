import { mkdir } from "node:fs/promises";
import path from "node:path";
import { test as setup } from "@playwright/test";
import { AUTH_STATE_PATH } from "./constants";
import {
  completeOnboarding,
  expectCookieBackedAuth,
  loginThroughOAuth,
  resetScenario,
  waitForSearchIndex,
} from "./helpers";

setup(
  "authenticate through the complete fake OAuth flow",
  async ({ page, request }) => {
    setup.setTimeout(150_000);
    await resetScenario(request);
    await loginThroughOAuth(page);
    await expectCookieBackedAuth(page);
    await completeOnboarding(page);
    await waitForSearchIndex(page);

    await mkdir(path.dirname(AUTH_STATE_PATH), { recursive: true });
    await page.context().storageState({ path: AUTH_STATE_PATH });
  },
);
