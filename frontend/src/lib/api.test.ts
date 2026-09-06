import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiFetch, ApiError, API_URL } from "./api";
import { browserSessionClient } from "./browserSessionBrowser";
import { sampleUser } from "../test/testServer";
import { BrowserSessionError } from "./browserSessionClient";

describe("apiFetch response adapter", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });
  afterEach(() => vi.restoreAllMocks());
  it("delegates transport and parses JSON", async () => {
    const request = vi
      .spyOn(browserSessionClient, "request")
      .mockResolvedValue(Response.json({ hello: "world" }));
    await expect(
      apiFetch("/api/thing", { method: "POST", body: "{}" }),
    ).resolves.toEqual({ hello: "world" });
    expect(request).toHaveBeenCalledWith(
      `${API_URL}/api/thing`,
      expect.objectContaining({ method: "POST", body: "{}" }),
    );
    expect(
      new Headers(request.mock.calls[0][1]?.headers).get("Content-Type"),
    ).toBe("application/json");
  });
  it("preserves caller headers and returns undefined for an empty response", async () => {
    const request = vi
      .spyOn(browserSessionClient, "request")
      .mockResolvedValue(new Response(null, { status: 204 }));
    await expect(
      apiFetch("/api/thing", {
        headers: new Headers({ "Content-Type": "text/plain" }),
      }),
    ).resolves.toBeUndefined();
    expect(
      new Headers(request.mock.calls[0][1]?.headers).get("Content-Type"),
    ).toBe("text/plain");
  });
  it("retains backend error classification and retry timing", async () => {
    vi.spyOn(browserSessionClient, "request").mockResolvedValue(
      Response.json(
        { error: "Private", code: "PRIVACY_RESTRICTION", retryAfter: 3 },
        { status: 403 },
      ),
    );
    await expect(apiFetch("/api/thing")).rejects.toMatchObject({
      name: "ApiError",
      message: "Private",
      status: 403,
      code: "PRIVACY_RESTRICTION",
      retryAfter: 3,
    });
  });
  it("maps typed session failures without owning logout or refresh", async () => {
    vi.spyOn(browserSessionClient, "request").mockRejectedValue(
      new BrowserSessionError("SESSION_EXPIRED", "Session expired", 401),
    );
    await expect(apiFetch("/api/thing")).rejects.toMatchObject({
      status: 401,
      code: "SESSION_EXPIRED",
    });
  });
  it("reports reconnect intent for UI routing", async () => {
    vi.spyOn(browserSessionClient, "request").mockResolvedValue(
      Response.json(
        { error: "Reconnect Bungie", code: "BUNGIE_REAUTH_REQUIRED" },
        { status: 401 },
      ),
    );
    await expect(apiFetch("/api/thing")).rejects.toBeInstanceOf(ApiError);
    expect(sessionStorage.getItem("guardian_bungie_reconnect")).toBe("1");
  });
  it("does not mark reconnect for a projection adopted while the error body parses", async () => {
    let finishParsing!: (body: unknown) => void;
    const pendingBody = new Promise<unknown>((resolve) => {
      finishParsing = resolve;
    });
    const response = new Response(null, { status: 401 });
    const parsed = new Response(null, { status: 401 });
    vi.spyOn(parsed, "json").mockReturnValue(pendingBody);
    vi.spyOn(response, "clone").mockReturnValue(parsed);
    vi.spyOn(browserSessionClient, "request").mockResolvedValue(response);
    const original = { status: "authenticated" as const, user: sampleUser };
    const snapshot = vi
      .spyOn(browserSessionClient, "getSnapshot")
      .mockReturnValue(original);
    const request = apiFetch("/api/thing");
    await vi.waitFor(() => expect(parsed.json).toHaveBeenCalled());
    snapshot.mockReturnValue({ status: "anonymous" });
    finishParsing({
      error: "Reconnect Bungie",
      code: "BUNGIE_REAUTH_REQUIRED",
    });
    await expect(request).rejects.toMatchObject({ code: "SESSION_CHANGED" });
    expect(sessionStorage.getItem("guardian_bungie_reconnect")).toBeNull();
  });
});
