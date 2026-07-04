import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { API, server } from "../test/testServer";
import { apiFetch, ApiError } from "./api";

// MSW lifecycle (listen/resetHandlers/close) is global — see test/setup.ts.
// Only the "refresh handling" suite below uses MSW; the "apiFetch" suite
// above stubs global fetch directly and is unaffected.

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("apiFetch", () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    localStorage.clear();
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    // jsdom throws on real navigation; stub location for the redirect path.
    vi.stubGlobal("location", { href: "" });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends the bearer token and parses JSON", async () => {
    localStorage.setItem("guardian_token", "tok-1");
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { hello: "world" }));

    const out = await apiFetch<{ hello: string }>("/api/thing");

    expect(out.hello).toBe("world");
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe(`${API}/api/thing`);
    expect((init.headers as Record<string, string>).Authorization).toBe(
      "Bearer tok-1",
    );
  });

  it("throws ApiError carrying status and backend code", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(403, {
        error: "profile is private",
        code: "PRIVACY_RESTRICTION",
      }),
    );

    const err = await apiFetch("/api/thing").catch((e) => e);

    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(403);
    expect((err as ApiError).code).toBe("PRIVACY_RESTRICTION");
    expect((err as ApiError).message).toBe("profile is private");
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    localStorage.setItem("guardian_token", "stale");
    localStorage.setItem("guardian_refresh_token", "refresh-1");

    fetchMock.mockImplementation(async (url: string, init?: RequestInit) => {
      if (url.endsWith("/api/auth/refresh")) {
        return jsonResponse(200, {
          token: "fresh",
          refreshToken: "refresh-2",
          user: {
            id: "u",
            displayName: "G",
            membershipId: "u",
            membershipType: 3,
          },
        });
      }
      const auth = (init?.headers as Record<string, string>)?.Authorization;
      return auth === "Bearer fresh"
        ? jsonResponse(200, { ok: true })
        : jsonResponse(401, { error: "expired" });
    });

    const out = await apiFetch<{ ok: boolean }>("/api/thing");

    expect(out.ok).toBe(true);
    expect(localStorage.getItem("guardian_token")).toBe("fresh");
    expect(localStorage.getItem("guardian_refresh_token")).toBe("refresh-2");
  });

  it("shares one in-flight refresh across concurrent 401s", async () => {
    localStorage.setItem("guardian_token", "stale");
    localStorage.setItem("guardian_refresh_token", "refresh-1");

    let refreshCalls = 0;
    fetchMock.mockImplementation(async (url: string, init?: RequestInit) => {
      if (url.endsWith("/api/auth/refresh")) {
        refreshCalls++;
        await new Promise((r) => setTimeout(r, 10));
        return jsonResponse(200, {
          token: "fresh",
          refreshToken: "refresh-2",
          user: {
            id: "u",
            displayName: "G",
            membershipId: "u",
            membershipType: 3,
          },
        });
      }
      const auth = (init?.headers as Record<string, string>)?.Authorization;
      return auth === "Bearer fresh"
        ? jsonResponse(200, { ok: true })
        : jsonResponse(401, { error: "expired" });
    });

    await Promise.all([
      apiFetch("/api/a"),
      apiFetch("/api/b"),
      apiFetch("/api/c"),
    ]);

    expect(refreshCalls).toBe(1);
  });

  it("clears storage and redirects when the refresh fails", async () => {
    localStorage.setItem("guardian_token", "stale");
    localStorage.setItem("guardian_refresh_token", "dead");
    localStorage.setItem("guardian_user", "{}");

    fetchMock.mockImplementation(async (url: string) =>
      url.endsWith("/api/auth/refresh")
        ? jsonResponse(401, { error: "revoked" })
        : jsonResponse(401, { error: "expired" }),
    );

    await expect(apiFetch("/api/thing")).rejects.toThrow("Session expired");
    expect(localStorage.getItem("guardian_token")).toBeNull();
    expect(localStorage.getItem("guardian_user")).toBeNull();
    expect(window.location.href).toBe("/login");
  });

  it("returns undefined for empty bodies (204)", async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
    const out = await apiFetch<void>("/api/thing", { method: "DELETE" });
    expect(out).toBeUndefined();
  });
});

describe("apiFetch refresh handling", () => {
  beforeEach(() => {
    localStorage.setItem("guardian_token", "expired");
    localStorage.setItem("guardian_refresh_token", "refresh-abc");
    localStorage.setItem(
      "guardian_user",
      JSON.stringify({ membershipId: "1" }),
    );
  });
  afterEach(() => localStorage.clear());

  it("keeps the session and throws a retryable error when refresh is rate-limited (429)", async () => {
    server.use(
      http.get(`${API}/api/thing`, () =>
        HttpResponse.json({ error: "nope" }, { status: 401 }),
      ),
      http.post(`${API}/api/auth/refresh`, () =>
        HttpResponse.json({ error: "slow down" }, { status: 429 }),
      ),
    );
    await expect(apiFetch("/api/thing")).rejects.toBeInstanceOf(ApiError);
    // Session NOT destroyed — the refresh token survives a transient failure.
    expect(localStorage.getItem("guardian_refresh_token")).toBe("refresh-abc");
  });

  it("keeps the session when refresh hits a 5xx", async () => {
    server.use(
      http.get(`${API}/api/thing`, () =>
        HttpResponse.json({ error: "nope" }, { status: 401 }),
      ),
      http.post(`${API}/api/auth/refresh`, () =>
        HttpResponse.json({ error: "boom" }, { status: 503 }),
      ),
    );
    await expect(apiFetch("/api/thing")).rejects.toBeInstanceOf(ApiError);
    expect(localStorage.getItem("guardian_token")).toBe("expired");
  });

  it("clears the session on a definitive 401 from the refresh endpoint", async () => {
    server.use(
      http.get(`${API}/api/thing`, () =>
        HttpResponse.json({ error: "nope" }, { status: 401 }),
      ),
      http.post(`${API}/api/auth/refresh`, () =>
        HttpResponse.json({ error: "revoked" }, { status: 401 }),
      ),
    );
    await expect(apiFetch("/api/thing")).rejects.toBeInstanceOf(ApiError);
    // Definitive auth failure — session cleared.
    expect(localStorage.getItem("guardian_token")).toBeNull();
    expect(localStorage.getItem("guardian_refresh_token")).toBeNull();
  });

  it("retries the original request after a successful refresh", async () => {
    let served = false;
    server.use(
      http.get(`${API}/api/thing`, () => {
        if (!served) {
          served = true;
          return HttpResponse.json({ error: "nope" }, { status: 401 });
        }
        return HttpResponse.json({ ok: true });
      }),
      http.post(`${API}/api/auth/refresh`, () =>
        HttpResponse.json({
          token: "fresh",
          refreshToken: "refresh-def",
          user: { membershipId: "1" },
        }),
      ),
    );
    await expect(apiFetch("/api/thing")).resolves.toEqual({ ok: true });
    expect(localStorage.getItem("guardian_token")).toBe("fresh");
  });
});
