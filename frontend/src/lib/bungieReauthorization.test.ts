import { beforeEach, describe, expect, it } from "vitest";
import {
  bungieReconnectReturnTo,
  clearBungieReconnect,
  hasBungieReconnectIntent,
  markBungieReconnect,
} from "./bungieReauthorization";

describe("Bungie reauthorization state", () => {
  beforeEach(() => sessionStorage.clear());

  it("stores only reconnect intent and a same-origin return path", () => {
    markBungieReconnect("/collections?node=10");

    expect(hasBungieReconnectIntent()).toBe(true);
    expect(bungieReconnectReturnTo()).toBe("/collections?node=10");
    expect(sessionStorage.length).toBe(2);
  });

  it("falls back to the dashboard for external or auth-flow return paths", () => {
    for (const unsafe of [
      "https://example.com",
      "//example.com",
      "/auth/callback?code=secret",
      "/reauthorize",
      "/login",
    ]) {
      clearBungieReconnect();
      markBungieReconnect(unsafe);
      expect(bungieReconnectReturnTo()).toBe("/dashboard");
    }
  });

  it("preserves the first return path when concurrent failures request reconnect", () => {
    markBungieReconnect("/collections");
    markBungieReconnect("/wishlist");

    expect(bungieReconnectReturnTo()).toBe("/collections");
  });
});
