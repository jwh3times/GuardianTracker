import { afterEach, describe, expect, it, vi } from "vitest";
import {
  completionHandoffTarget,
  handOffOAuthCompletion,
} from "./oauthCompletionHandoff";

const tunnelCallback = {
  origin: "https://tunnel.example",
  pathname: "/auth/callback",
  search: "?code=onetime&state=v2.signed",
  hash: "",
};

afterEach(() => {
  vi.restoreAllMocks();
});

describe("completionHandoffTarget", () => {
  it("forwards a callback from another origin, keeping its path and query", () => {
    expect(
      completionHandoffTarget("http://localhost:5273", tunnelCallback),
    ).toBe("http://localhost:5273/auth/callback?code=onetime&state=v2.signed");
  });

  it("keeps Bungie's error parameters and any fragment", () => {
    expect(
      completionHandoffTarget("http://localhost:5273", {
        ...tunnelCallback,
        search: "?error=access_denied&error_description=Denied",
        hash: "#x",
      }),
    ).toBe(
      "http://localhost:5273/auth/callback?error=access_denied&error_description=Denied#x",
    );
  });

  it("accepts a trailing slash on the configured origin", () => {
    expect(
      completionHandoffTarget("http://localhost:5273/", tunnelCallback),
    ).toBe("http://localhost:5273/auth/callback?code=onetime&state=v2.signed");
  });

  it.each(["http://127.0.0.1:5273", "http://[::1]:5273"])(
    "accepts the loopback origin %s",
    (configured) => {
      expect(completionHandoffTarget(configured, tunnelCallback)).toBe(
        `${configured}/auth/callback?code=onetime&state=v2.signed`,
      );
    },
  );

  it.each([
    { name: "unset", configured: undefined },
    { name: "empty", configured: "" },
    { name: "whitespace", configured: "   " },
  ])("does nothing, silently, when the setting is $name", ({ configured }) => {
    // Every Docker build without the setting receives an empty string, so an
    // unset value must not warn on each sign-in.
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(completionHandoffTarget(configured, tunnelCallback)).toBeNull();
    expect(warn).not.toHaveBeenCalled();
  });

  it("does nothing when the callback is already on the completion origin", () => {
    expect(
      completionHandoffTarget("http://localhost:5273", {
        ...tunnelCallback,
        origin: "http://localhost:5273",
      }),
    ).toBeNull();
  });

  it.each([
    { name: "not a URL", configured: "localhost:5273" },
    { name: "not http(s)", configured: "javascript:alert(1)" },
    // ws: keeps a real origin, so only the protocol rule can reject it.
    { name: "a non-http(s) origin", configured: "ws://localhost:5273" },
    { name: "carrying a path", configured: "http://localhost:5273/evil" },
    { name: "carrying a query", configured: "http://localhost:5273?x=1" },
    { name: "carrying credentials", configured: "http://user@localhost:5273" },
    { name: "not a loopback host", configured: "https://app.example" },
    // Fails closed: the parsed origin is lowercased, so it no longer matches.
    { name: "uppercase", configured: "HTTP://LOCALHOST:5273" },
    // Fails closed: the parsed origin drops the default port.
    { name: "an explicit default port", configured: "http://localhost:80" },
  ])("ignores a setting that is $name, with a warning", ({ configured }) => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(completionHandoffTarget(configured, tunnelCallback)).toBeNull();
    expect(warn).toHaveBeenCalledTimes(1);
  });
});

describe("handOffOAuthCompletion", () => {
  it("replaces the location with the handoff target and reports that it navigated", () => {
    const replace = vi.fn<(url: string) => void>();
    expect(
      handOffOAuthCompletion("http://localhost:5273", {
        ...tunnelCallback,
        replace,
      }),
    ).toBe(true);
    expect(replace).toHaveBeenCalledTimes(1);
    expect(replace).toHaveBeenCalledWith(
      "http://localhost:5273/auth/callback?code=onetime&state=v2.signed",
    );
  });

  it("leaves the page alone when there is nowhere to hand off to", () => {
    const replace = vi.fn<(url: string) => void>();
    expect(
      handOffOAuthCompletion(undefined, { ...tunnelCallback, replace }),
    ).toBe(false);
    expect(replace).not.toHaveBeenCalled();
  });
});
