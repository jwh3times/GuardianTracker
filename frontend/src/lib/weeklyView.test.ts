import { describe, it, expect } from "vitest";
import { toWeekly } from "./weeklyView";
import type {
  APIDifficulty,
  APIRecommendedAction,
  APIWeekly,
} from "../types/api";

function rawWeekly(overrides: Partial<APIWeekly> = {}): APIWeekly {
  return {
    resetLabel: "Tuesday 17:00 UTC",
    resetIn: { d: 2, h: 4, m: 0 },
    dailyResetIn: { h: 7, m: 30 },
    resetAt: "2026-06-16T17:00:00Z",
    fetchedAt: "2026-06-14T13:00:00Z",
    xur: null,
    milestones: [],
    recommended: [],
    dailyActions: [],
    ...overrides,
  };
}

/**
 * Takes a bare string, not APIDifficulty: the point of these cases is what the
 * adapter does with a tier the union does not cover.
 */
function action(diff: string): APIRecommendedAction {
  return {
    id: "eff-1",
    text: "Run Vault of Glass",
    detail: "Fills 3 missing items",
    badge: "activity",
    done: false,
    diff: diff as APIDifficulty,
    time: "45m",
  };
}

describe("toWeekly", () => {
  it("adapts a ranked title-case tier into the design vocabulary", () => {
    const w = toWeekly(rawWeekly({ recommended: [action("Challenging")] }));
    expect(w.recommended[0].diff).toBe("challenging");
  });

  it("adapts a legacy lowercase fallback tier unchanged", () => {
    const w = toWeekly(rawWeekly({ recommended: [action("moderate")] }));
    expect(w.recommended[0].diff).toBe("moderate");
  });

  it("collapses an unrecognised tier to unrated", () => {
    const w = toWeekly(rawWeekly({ recommended: [action("Brutal")] }));
    expect(w.recommended[0].diff).toBe("unrated");
  });

  it("preserves every other recommendation field verbatim", () => {
    const w = toWeekly(rawWeekly({ recommended: [action("Easy")] }));
    expect(w.recommended[0]).toEqual({
      id: "eff-1",
      text: "Run Vault of Glass",
      detail: "Fills 3 missing items",
      badge: "activity",
      done: false,
      diff: "easy",
      time: "45m",
    });
  });

  it("preserves ranked order across the projection", () => {
    const w = toWeekly(
      rawWeekly({
        recommended: [
          { ...action("Challenging"), id: "a" },
          { ...action("Easy"), id: "b" },
          { ...action("Moderate"), id: "c" },
        ],
      }),
    );
    expect(w.recommended.map((r) => r.id)).toEqual(["a", "b", "c"]);
  });

  it("turns a null recommendation list into an empty one", () => {
    // Go serialises a nil slice as null; the design type is non-nullable, so the
    // seam is where that becomes an empty list rather than every reader guarding.
    expect(toWeekly(rawWeekly({ recommended: null })).recommended).toEqual([]);
  });

  it("passes reset timing, Xûr, milestones and today's actions through", () => {
    const raw = rawWeekly({
      degraded: true,
      xur: {
        present: true,
        leavesIn: { d: 1, h: 2 },
        location: "The Tower",
        items: [
          {
            hash: "1",
            name: "Gjallarhorn",
            type: "Rocket Launcher",
            icon: "/icon.png",
            rarity: "exotic",
            missing: true,
            cost: "29 Strange Coins",
          },
        ],
      },
      milestones: [
        {
          id: "m-1",
          label: "Raid",
          name: "Vault of Glass",
          reward: "Pinnacle",
          missing: 3,
          note: "",
        },
      ],
      dailyActions: [
        {
          id: "t-1",
          category: "xur",
          icon: "bolt",
          text: "Visit Xûr",
          detail: "Tower",
          badge: "expiring",
          resetsIn: { d: 1 },
          done: false,
        },
      ],
    });
    const w = toWeekly(raw);
    expect(w.degraded).toBe(true);
    expect(w.xur).toEqual(raw.xur);
    expect(w.milestones).toEqual(raw.milestones);
    expect(w.dailyActions).toEqual(raw.dailyActions);
    expect(w.resetLabel).toBe("Tuesday 17:00 UTC");
    expect(w.resetAt).toBe("2026-06-16T17:00:00Z");
    expect(w.fetchedAt).toBe("2026-06-14T13:00:00Z");
    expect(w.resetIn).toEqual({ d: 2, h: 4, m: 0 });
    expect(w.dailyResetIn).toEqual({ h: 7, m: 30 });
  });

  it("does not mutate the raw payload", () => {
    const raw = rawWeekly({ recommended: [action("Challenging")] });
    toWeekly(raw);
    expect(raw.recommended?.[0].diff).toBe("Challenging");
  });
});
