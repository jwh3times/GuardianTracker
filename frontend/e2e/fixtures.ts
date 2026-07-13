import { test as base } from "@playwright/test";
import {
  resetScenario,
  setWeeklyScenario,
  type WeeklyScenario,
} from "./helpers";

type ScenarioFixture = {
  weekly: (scenario: WeeklyScenario) => Promise<void>;
  reset: () => Promise<void>;
};

export const test = base.extend<{ scenario: ScenarioFixture }>({
  // `use.reducedMotion` in playwright.config.ts resolves correctly on the
  // project (testInfo.project.use.reducedMotion === "reduce") but does not reach
  // the browser context — every page in the context, including a blank data URL,
  // still reports prefers-reduced-motion: no-preference. Applying it per page is
  // what actually activates the app's reduced-motion CSS, which both the
  // accessibility assertion and the visual snapshots depend on.
  page: async ({ page }, provide) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await provide(page);
  },

  scenario: async ({ request }, provide) => {
    await resetScenario(request);
    await provide({
      weekly: (scenario) => setWeeklyScenario(request, scenario),
      reset: () => resetScenario(request),
    });
    await resetScenario(request);
  },
});

export { expect } from "@playwright/test";
