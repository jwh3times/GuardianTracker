import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  BROWSER_SESSION_LIFECYCLE_LOCK,
  BROWSER_SESSION_STORAGE_KEY,
  FetchBrowserSessionAuthTransport,
  LocalStorageBrowserSessionPersistence,
  WebLocksBrowserSessionCoordinator,
  browserSessionClient,
  createBrowserLocalStoragePersistence,
} from "./browserSessionBrowser";
import { LifecycleCoordinationUnavailableError } from "./browserSessionClient";

function bodyFromFetchCall(call: unknown): unknown {
  if (!Array.isArray(call)) return undefined;
  const init: unknown = call[1];
  return typeof init === "object" && init !== null && "body" in init
    ? init.body
    : undefined;
}

describe("LocalStorageBrowserSessionPersistence", () => {
  beforeEach(() => localStorage.clear());

  it("stores one envelope and clears every legacy JavaScript credential key", () => {
    localStorage.setItem("guardian_token", "access");
    localStorage.setItem("guardian_user", "user-json");
    localStorage.setItem("guardian_refresh_token", "must-be-removed");
    const persistence = new LocalStorageBrowserSessionPersistence();

    expect(persistence.readLegacy()).toEqual({
      token: "access",
      user: "user-json",
    });
    persistence.write("atomic-envelope");
    persistence.clearLegacy();

    expect(persistence.read()).toBe("atomic-envelope");
    expect(localStorage.getItem("guardian_token")).toBeNull();
    expect(localStorage.getItem("guardian_user")).toBeNull();
    expect(localStorage.getItem("guardian_refresh_token")).toBeNull();
  });

  it("forwards only storage events for the projection envelope", () => {
    const persistence = new LocalStorageBrowserSessionPersistence();
    const listener = vi.fn<(value: string | null) => void>();
    const unsubscribe = persistence.subscribe(listener);

    window.dispatchEvent(
      new StorageEvent("storage", {
        key: "unrelated",
        newValue: "ignore",
      }),
    );
    window.dispatchEvent(
      new StorageEvent("storage", {
        key: BROWSER_SESSION_STORAGE_KEY,
        newValue: "new-envelope",
      }),
    );
    unsubscribe();
    window.dispatchEvent(
      new StorageEvent("storage", {
        key: BROWSER_SESSION_STORAGE_KEY,
        newValue: "after-unsubscribe",
      }),
    );

    expect(listener).toHaveBeenCalledOnce();
    expect(listener).toHaveBeenCalledWith("new-envelope");
  });

  it("normalizes failure to acquire localStorage", () => {
    const inaccessibleWindow = {
      get localStorage(): Storage {
        throw new DOMException("blocked", "SecurityError");
      },
    } as unknown as Window;

    expect(() =>
      createBrowserLocalStoragePersistence(inaccessibleWindow),
    ).toThrowError(
      expect.objectContaining({ code: "PERSISTENCE_UNAVAILABLE" }),
    );
  });
});

describe("WebLocksBrowserSessionCoordinator", () => {
  it("uses the one origin-wide lifecycle lock exclusively", async () => {
    const request = vi.fn<
      (
        _name: string,
        _options: LockOptions,
        callback: (lock: Lock | null) => Promise<string> | string,
      ) => Promise<string>
    >((_name, _options, callback) => Promise.resolve(callback(null)));
    const coordinator = new WebLocksBrowserSessionCoordinator({
      request,
    } as unknown as LockManager);

    await expect(
      coordinator.runExclusive(() => Promise.resolve("result")),
    ).resolves.toBe("result");
    expect(request).toHaveBeenCalledWith(
      BROWSER_SESSION_LIFECYCLE_LOCK,
      { mode: "exclusive" },
      expect.any(Function),
    );
  });

  it("reports unsupported Web Locks explicitly", async () => {
    const coordinator = new WebLocksBrowserSessionCoordinator(undefined);
    await expect(
      coordinator.runExclusive(() => Promise.resolve(undefined)),
    ).rejects.toBeInstanceOf(LifecycleCoordinationUnavailableError);
  });
});

describe("FetchBrowserSessionAuthTransport", () => {
  const fetchMock = vi.fn<typeof fetch>();
  const baseUrl = "https://api.example";

  beforeEach(() => fetchMock.mockReset());
  afterEach(() => vi.restoreAllMocks());

  it("uses the existing authorization endpoints and form-encoded callback wire", async () => {
    fetchMock
      .mockResolvedValueOnce(
        Response.json({
          authUrl: "https://bungie.example/authorize",
          state: "state-1",
        }),
      )
      .mockResolvedValueOnce(Response.json({ token: "token", user: {} }));
    const transport = new FetchBrowserSessionAuthTransport(baseUrl, fetchMock);

    await expect(transport.beginAuthorization()).resolves.toEqual({
      authUrl: "https://bungie.example/authorize",
      state: "state-1",
    });
    await transport.completeAuthorization({ code: "code-1", state: "state-1" });

    expect(fetchMock).toHaveBeenNthCalledWith(1, `${baseUrl}/api/auth/bungie`);
    const callbackBody = bodyFromFetchCall(fetchMock.mock.calls[1]);
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      `${baseUrl}/api/auth/bungie/callback`,
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: callbackBody,
      }),
    );
    expect(callbackBody).toBeInstanceOf(URLSearchParams);
    if (!(callbackBody instanceof URLSearchParams)) {
      throw new Error("callback body was not form encoded");
    }
    expect(callbackBody.toString()).toBe("code=code-1&state=state-1");
  });

  it("attaches the private bearer token while preserving caller headers", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));
    const transport = new FetchBrowserSessionAuthTransport(baseUrl, fetchMock);

    await transport.request(
      "/api/protected",
      {
        method: "PATCH",
        headers: {
          Authorization: "Bearer caller-must-not-win",
          "X-Test": "kept",
        },
      },
      "private-token",
    );

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe(`${baseUrl}/api/protected`);
    expect(init).toMatchObject({ method: "PATCH", credentials: "include" });
    const headers = new Headers(init?.headers);
    expect(headers.get("Authorization")).toBe("Bearer private-token");
    expect(headers.get("X-Test")).toBe("kept");
  });

  it("removes caller-supplied authorization from anonymous requests", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));
    const transport = new FetchBrowserSessionAuthTransport(baseUrl, fetchMock);

    await transport.request(
      "/api/public",
      { headers: { Authorization: "Bearer caller-token" } },
      undefined,
    );

    const headers = new Headers(fetchMock.mock.calls[0]?.[1]?.headers);
    expect(headers.has("Authorization")).toBe(false);
  });

  it.each([
    ["absolute string", "https://evil.example/api"],
    ["URL", new URL("https://evil.example/api")],
    ["Request", new Request("https://evil.example/api")],
  ])("rejects a cross-origin %s before calling fetch", async (_name, input) => {
    const transport = new FetchBrowserSessionAuthTransport(baseUrl, fetchMock);

    await expect(
      transport.request(
        input,
        { headers: { Authorization: "caller" } },
        "secret",
      ),
    ).rejects.toMatchObject({ code: "CROSS_ORIGIN_REQUEST" });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("refreshes and ends current/all sessions with the existing wire contract", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));
    const transport = new FetchBrowserSessionAuthTransport(baseUrl, fetchMock);

    await transport.refresh();
    await transport.end("current", "token-1");
    await transport.end("all", "token-2");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      `${baseUrl}/api/auth/refresh`,
      `${baseUrl}/api/auth/logout`,
      `${baseUrl}/api/auth/logout/all`,
    ]);
    for (const [, init] of fetchMock.mock.calls) {
      expect(init?.credentials).toBe("include");
      expect(init?.method).toBe("POST");
    }
    expect(fetchMock.mock.calls[0]?.[1]?.body).toBe("{}");
    expect(fetchMock.mock.calls[1]?.[1]?.body).toBeUndefined();
    expect(fetchMock.mock.calls[2]?.[1]?.body).toBeUndefined();
  });
});

describe("production browser session singleton", () => {
  it("is defined without being activated by the existing auth modules", () => {
    expect(browserSessionClient).toHaveProperty("getSnapshot");
  });
});
