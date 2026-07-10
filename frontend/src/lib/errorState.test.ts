import { describe, it, expect } from "vitest";
import { errorState, type ErrorStateCopy } from "./errorState";
import { ApiError } from "./api";

describe("errorState", () => {
  it("maps REFRESH_UNAVAILABLE (503) to 'Reconnecting…' copy (not manifest warming)", () => {
    const error = new ApiError(
      "Session refresh temporarily unavailable",
      503,
      "REFRESH_UNAVAILABLE",
    );
    const result = errorState(error);

    expect(result).toEqual({
      icon: "refresh",
      title: "Reconnecting…",
      body: "Your session is refreshing. Give it a moment, then try again.",
    } as ErrorStateCopy);

    // Explicitly verify it's NOT the manifest warming copy.
    expect(result.title).not.toBe("Warming up…");
    expect(result.body).not.toContain("item database");
  });

  it("maps plain 503 (MANIFEST_NOT_READY) to 'Warming up…' copy", () => {
    const error = new ApiError("Manifest not ready", 503, "MANIFEST_NOT_READY");
    const result = errorState(error);

    expect(result.title).toBe("Warming up…");
    expect(result.body).toContain("item database");
  });

  it("maps 503 with no explicit code to 'Warming up…' copy", () => {
    const error = new ApiError("Service unavailable", 503);
    const result = errorState(error);

    expect(result.title).toBe("Warming up…");
    expect(result.body).toContain("item database");
  });

  it("maps PRIVACY_RESTRICTION to privacy copy with lock icon", () => {
    const error = new ApiError(
      "Profile is private",
      403,
      "PRIVACY_RESTRICTION",
    );
    const result = errorState(error);

    expect(result.icon).toBe("lock");
    expect(result.title).toBe("Your Destiny profile is private");
    expect(result.privacyLink).toBe(true);
  });

  it("maps BUNGIE_ERROR to Bungie unavailable copy", () => {
    const error = new ApiError("Bungie error", 502, "BUNGIE_ERROR");
    const result = errorState(error);

    expect(result.title).toBe("Bungie API unavailable");
  });

  it("maps plain 502 to Bungie unavailable copy", () => {
    const error = new ApiError("Bad gateway", 502);
    const result = errorState(error);

    expect(result.title).toBe("Bungie API unavailable");
  });

  it("maps unknown error to generic fallback", () => {
    const result = errorState(new Error("Generic error"));

    expect(result.title).toBe("Couldn't load data");
  });
});

describe("errorState feature-flag codes", () => {
  it("maps FEATURE_DISABLED to an unavailable-feature panel", () => {
    const copy = errorState(new ApiError("gone", 404, "FEATURE_DISABLED"));
    expect(copy.title).toMatch(/not available|unavailable/i);
  });

  it("maps TIER_LOCKED to an upsell panel", () => {
    const copy = errorState(new ApiError("locked", 403, "TIER_LOCKED"));
    expect(copy.title).toMatch(/locked|access/i);
  });
});
