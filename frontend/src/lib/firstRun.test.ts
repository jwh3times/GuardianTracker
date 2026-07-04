import { describe, expect, it } from "vitest";
import { isFirstRun, markFirstRunDone } from "./firstRun";

describe("firstRun", () => {
  it("is true before marking and false after", () => {
    expect(isFirstRun("123")).toBe(true);
    markFirstRunDone("123");
    expect(isFirstRun("123")).toBe(false);
  });

  it("is tracked per membership id", () => {
    markFirstRunDone("123");
    expect(isFirstRun("456")).toBe(true);
  });

  it("works without a membership id", () => {
    expect(isFirstRun()).toBe(true);
    markFirstRunDone();
    expect(isFirstRun()).toBe(false);
  });
});
