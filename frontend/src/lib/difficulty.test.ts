import { describe, it, expect } from "vitest";
import { toDifficulty } from "./difficulty";

describe("toDifficulty", () => {
  it("maps every canonical title-case tier to its design value", () => {
    expect(toDifficulty("Easy")).toBe("easy");
    expect(toDifficulty("Moderate")).toBe("moderate");
    expect(toDifficulty("Challenging")).toBe("challenging");
    expect(toDifficulty("Unrated")).toBe("unrated");
  });

  it("still accepts the legacy lowercase spellings during migration", () => {
    expect(toDifficulty("easy")).toBe("easy");
    expect(toDifficulty("moderate")).toBe("moderate");
    expect(toDifficulty("challenging")).toBe("challenging");
    expect(toDifficulty("unrated")).toBe("unrated");
  });

  it("collapses an unrecognised tier to the explicit unrated state", () => {
    // Not a guessed tier: "unrated" is the same honest answer the backend gives
    // when no source keyword matched.
    expect(toDifficulty("Impossible")).toBe("unrated");
    expect(toDifficulty("CHALLENGING")).toBe("unrated");
    expect(toDifficulty("")).toBe("unrated");
  });

  it("treats a missing tier as unrated rather than throwing", () => {
    expect(toDifficulty(undefined)).toBe("unrated");
    expect(toDifficulty(null)).toBe("unrated");
  });

  it("does not resolve inherited Object properties as tiers", () => {
    expect(toDifficulty("toString")).toBe("unrated");
    expect(toDifficulty("constructor")).toBe("unrated");
  });
});
