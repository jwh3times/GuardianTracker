import { describe, expect, it } from "vitest";
import { initialTourState, tourReducer } from "./tourState";

describe("tourReducer", () => {
  it("starts at the Dashboard step after the welcome screen", () => {
    expect(tourReducer(initialTourState, { type: "start" })).toEqual({
      phase: "tour",
      step: 0,
    });
  });

  it("advances through all three spotlight steps", () => {
    const dashboard = tourReducer(initialTourState, { type: "start" });
    const weekly = tourReducer(dashboard, { type: "next" });
    const collections = tourReducer(weekly, { type: "next" });

    expect(weekly).toEqual({ phase: "tour", step: 1 });
    expect(collections).toEqual({ phase: "tour", step: 2 });
    expect(tourReducer(collections, { type: "next" })).toEqual({
      phase: "complete",
      step: 2,
    });
  });

  it("can be skipped from any phase", () => {
    expect(tourReducer(initialTourState, { type: "skip" }).phase).toBe(
      "complete",
    );
    expect(
      tourReducer({ phase: "tour", step: 1 }, { type: "skip" }).phase,
    ).toBe("complete");
  });

  it("ignores actions that do not apply to the current phase", () => {
    expect(tourReducer(initialTourState, { type: "next" })).toBe(
      initialTourState,
    );
  });
});
