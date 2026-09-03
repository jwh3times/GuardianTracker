import { describe, expect, it, vi } from "vitest";
import type { APIGuardianTrackerUser } from "../types/api";
import {
  BrowserSessionError,
  LifecycleCoordinationUnavailableError,
  createBrowserSessionClient,
  type AuthorizationCompletion,
  type AuthorizationStart,
  type BrowserSessionAuthTransport,
  type BrowserSessionLifecycleCoordinator,
  type BrowserSessionPersistence,
} from "./browserSessionClient";

const user = (
  membershipId = "membership-1",
  displayName = "Guardian",
): APIGuardianTrackerUser => ({
  id: membershipId,
  displayName,
  membershipId,
  membershipType: 3,
  platform: "Steam",
  role: "standard",
});

const replacementResponse = (
  token: string,
  replacementUser = user(),
): Response => Response.json({ token, user: replacementUser }, { status: 200 });

const response = (status: number, body: unknown = {}): Response =>
  status === 204
    ? new Response(null, { status })
    : Response.json(body, { status });

const envelope = (
  revision: number | bigint | string,
  projection:
    | { status: "anonymous" }
    | {
        status: "authenticated";
        accessToken: string;
        user: APIGuardianTrackerUser;
      },
  lineage = "lineage-1",
): string =>
  JSON.stringify({
    schemaVersion: 1,
    revision: revision.toString(),
    lineage,
    projection,
  });

const persistedLineage = (raw: string | null): string | undefined =>
  /"lineage":"([^"]+)"/.exec(raw ?? "")?.[1];

class MemoryPersistence implements BrowserSessionPersistence {
  raw: string | null = null;
  legacy = { token: null as string | null, user: null as string | null };
  obsoleteRefreshToken: string | null = null;
  writes: string[] = [];
  failNextWrite = false;
  echoWrites = false;
  failRead = false;
  failLegacyRead = false;
  failLegacyClear = false;
  failSubscribe = false;
  private readonly listeners = new Set<(value: string | null) => void>();

  read(): string | null {
    if (this.failRead) throw new Error("read blocked");
    return this.raw;
  }

  write(value: string): void {
    if (this.failNextWrite) {
      this.failNextWrite = false;
      throw new Error("storage full");
    }
    this.raw = value;
    this.writes.push(value);
    if (this.echoWrites) {
      for (const listener of this.listeners) listener(value);
    }
  }

  readLegacy(): { token: string | null; user: string | null } {
    if (this.failLegacyRead) throw new Error("legacy read blocked");
    return { ...this.legacy };
  }

  clearLegacy(): void {
    if (this.failLegacyClear) throw new Error("legacy clear blocked");
    this.legacy = { token: null, user: null };
    this.obsoleteRefreshToken = null;
  }

  subscribe(listener: (value: string | null) => void): () => void {
    if (this.failSubscribe) throw new Error("subscribe blocked");
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  externalWrite(value: string): void {
    this.raw = value;
    for (const listener of this.listeners) listener(value);
  }
}

class SerialCoordinator implements BrowserSessionLifecycleCoordinator {
  entries = 0;
  private tail: Promise<unknown> = Promise.resolve();

  runExclusive<T>(work: () => Promise<T>): Promise<T> {
    const result = this.tail.then(() => {
      this.entries += 1;
      return work();
    });
    this.tail = result.catch(() => undefined);
    return result;
  }
}

class ScriptedTransport implements BrowserSessionAuthTransport {
  authorizationStart: AuthorizationStart = {
    authUrl: "https://bungie.example/authorize",
    state: "state-1",
  };
  begin = vi.fn<BrowserSessionAuthTransport["beginAuthorization"]>(() =>
    Promise.resolve(this.authorizationStart),
  );
  complete = vi.fn<BrowserSessionAuthTransport["completeAuthorization"]>(() =>
    Promise.resolve(replacementResponse("callback-token")),
  );
  send = vi.fn<BrowserSessionAuthTransport["request"]>(() =>
    Promise.resolve(response(200)),
  );
  rotate = vi.fn<BrowserSessionAuthTransport["refresh"]>(() =>
    Promise.resolve(replacementResponse("refreshed-token")),
  );
  logout = vi.fn<BrowserSessionAuthTransport["end"]>(() =>
    Promise.resolve(response(204)),
  );

  beginAuthorization(): Promise<AuthorizationStart> {
    return this.begin();
  }

  completeAuthorization(input: AuthorizationCompletion): Promise<Response> {
    return this.complete(input);
  }

  request(
    input: RequestInfo | URL,
    init: RequestInit | undefined,
    accessToken: string | undefined,
  ): Promise<Response> {
    return this.send(input, init, accessToken);
  }

  refresh(): Promise<Response> {
    return this.rotate();
  }

  end(scope: "current" | "all", accessToken: string): Promise<Response> {
    return this.logout(scope, accessToken);
  }
}

function setup(persistence = new MemoryPersistence()) {
  const transport = new ScriptedTransport();
  const coordinator = new SerialCoordinator();
  const client = createBrowserSessionClient({
    transport,
    persistence,
    coordinator,
  });
  return { client, coordinator, persistence, transport };
}

function authenticatedPersistence(
  accessToken = "token-1",
  sessionUser = user(),
): MemoryPersistence {
  const persistence = new MemoryPersistence();
  persistence.raw = envelope(4, {
    status: "authenticated",
    accessToken,
    user: sessionUser,
  });
  return persistence;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { promise, reject, resolve };
}

function deferredJsonResponse(value: unknown, status = 200) {
  let release!: () => void;
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      release = () => {
        controller.enqueue(new TextEncoder().encode(JSON.stringify(value)));
        controller.close();
      };
    },
  });
  return {
    response: new Response(body, {
      status,
      headers: { "Content-Type": "application/json" },
    }),
    release,
  };
}

describe("BrowserSessionClient hydration", () => {
  it("synchronously migrates one complete legacy pair and exposes no token", () => {
    const persistence = new MemoryPersistence();
    persistence.legacy = {
      token: "legacy-token",
      user: JSON.stringify(user()),
    };
    persistence.obsoleteRefreshToken = "must-not-survive";

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
    expect(client.getSnapshot()).not.toHaveProperty("accessToken");
    expect(persistence.legacy).toEqual({ token: null, user: null });
    expect(persistence.obsoleteRefreshToken).toBeNull();
    expect(persistence.writes).toHaveLength(1);
  });

  it.each([
    ["partial pair", "token", null],
    ["corrupt user", "token", "not-json"],
    ["incomplete user", "token", JSON.stringify({ membershipId: "1" })],
  ])(
    "repairs %s to an anonymous atomic envelope",
    (_name, token, storedUser) => {
      const persistence = new MemoryPersistence();
      persistence.legacy = { token, user: storedUser };

      const { client } = setup(persistence);

      expect(client.getSnapshot()).toEqual({ status: "anonymous" });
      expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
        schemaVersion: 1,
        revision: "1",
        projection: { status: "anonymous" },
      });
      expect(persistence.legacy).toEqual({ token: null, user: null });
    },
  );

  it("repairs a corrupt envelope instead of resurrecting legacy credentials", () => {
    const persistence = new MemoryPersistence();
    persistence.raw = "{bad";
    persistence.legacy = {
      token: "stale-token",
      user: JSON.stringify(user()),
    };

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(persistence.legacy).toEqual({ token: null, user: null });
  });

  it.each([
    ["empty id", { ...user(), id: "" }],
    ["empty membership", { ...user(), id: "", membershipId: "" }],
    ["empty display name", { ...user(), displayName: "" }],
    ["mismatched legacy id", { ...user(), id: "different" }],
    ["fractional membership type", { ...user(), membershipType: 3.5 }],
    [
      "unsafe membership type",
      { ...user(), membershipType: Number.MAX_SAFE_INTEGER + 1 },
    ],
    ["credential-bearing extra", { ...user(), refreshToken: "forbidden" }],
  ])("rejects a legacy user with %s", (_name, legacyUser) => {
    const persistence = new MemoryPersistence();
    persistence.legacy = {
      token: "legacy-token",
      user: JSON.stringify(legacyUser),
    };

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("allowlists persisted user fields instead of retaining unknown data", () => {
    const persistence = new MemoryPersistence();
    persistence.raw = JSON.stringify({
      schemaVersion: 1,
      revision: "3",
      lineage: "lineage-1",
      projection: {
        status: "authenticated",
        accessToken: "token",
        user: { ...user(), harmlessFutureField: "discarded" },
      },
    });

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
    expect(client.getSnapshot()).not.toHaveProperty("harmlessFutureField");
    expect(persistence.raw).toContain("harmlessFutureField");
    expect(persistence.writes).toHaveLength(0);
  });

  it("repairs a persisted user containing credential material", () => {
    const persistence = new MemoryPersistence();
    persistence.raw = JSON.stringify({
      schemaVersion: 1,
      revision: "3",
      lineage: "lineage-1",
      projection: {
        status: "authenticated",
        accessToken: "access-token",
        user: { ...user(), credentialSecret: "forbidden" },
      },
    });

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(persistence.raw).not.toContain("credentialSecret");
  });

  it("rejects a tombstone carrying any credential material", () => {
    const persistence = new MemoryPersistence();
    persistence.raw = JSON.stringify({
      schemaVersion: 1,
      revision: "7",
      lineage: "lineage-1",
      projection: { status: "anonymous", accessToken: "must-not-survive" },
    });

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      schemaVersion: 1,
      revision: "1",
      projection: { status: "anonymous" },
    });
  });

  it("increments an arbitrarily large decimal revision without saturation", async () => {
    const persistence = new MemoryPersistence();
    const hugeRevision = "999999999999999999999999999999999999999999";
    persistence.raw = envelope(hugeRevision, {
      status: "authenticated",
      accessToken: "token",
      user: user(),
    });

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
    await client.end("current");
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      revision: "1000000000000000000000000000000000000000000",
      projection: { status: "anonymous" },
    });
  });

  it("repairs a non-canonical decimal revision", () => {
    const persistence = new MemoryPersistence();
    persistence.raw = envelope("01", { status: "anonymous" });
    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      revision: "1",
    });
  });

  it("repairs a legacy numeric envelope revision", () => {
    const persistence = new MemoryPersistence();
    persistence.raw = JSON.stringify({
      schemaVersion: 1,
      revision: 4,
      lineage: "lineage-1",
      projection: {
        status: "authenticated",
        accessToken: "must-not-survive",
        user: user(),
      },
    });

    const { client } = setup(persistence);

    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      revision: "1",
      projection: { status: "anonymous" },
    });
  });

  it.each([
    ["envelope read", "failRead"],
    ["legacy read", "failLegacyRead"],
    ["legacy cleanup", "failLegacyClear"],
    ["storage subscription", "failSubscribe"],
  ] as const)("normalizes a blocked %s", (_name, failure) => {
    const persistence = new MemoryPersistence();
    persistence[failure] = true;

    expect(() => setup(persistence)).toThrowError(
      expect.objectContaining({ code: "PERSISTENCE_UNAVAILABLE" }),
    );
  });
});

describe("BrowserSessionClient establishment and adoption", () => {
  it("starts authorization through the client interface", async () => {
    const { client } = setup();

    await expect(client.beginAuthorization()).resolves.toEqual({
      authUrl: "https://bungie.example/authorize",
      state: "state-1",
    });
  });

  it("normalizes authorization-start network failures", async () => {
    const { client, transport } = setup();
    transport.begin.mockRejectedValue(new TypeError("offline"));

    await expect(client.beginAuthorization()).rejects.toMatchObject({
      code: "AUTHORIZATION_FAILED",
    });
  });

  it("prefers an obsolete-start failure when logout retires a rejected start", async () => {
    const { client, transport } = setup();
    const gate = deferred<AuthorizationStart>();
    transport.begin.mockReturnValue(gate.promise);
    const starting = client.beginAuthorization();
    await vi.waitFor(() => expect(transport.begin).toHaveBeenCalledOnce());

    await client.end("current");
    gate.reject(new TypeError("offline"));

    await expect(starting).rejects.toMatchObject({
      code: "AUTHORIZATION_OBSOLETE",
    });
  });

  it("persists an authorization replacement before publishing it once", async () => {
    const persistence = new MemoryPersistence();
    const { client, coordinator } = setup(persistence);
    persistence.echoWrites = true;
    const observations: Array<{ snapshot: unknown; persisted: string | null }> =
      [];
    client.subscribe(() =>
      observations.push({
        snapshot: client.getSnapshot(),
        persisted: persistence.raw,
      }),
    );

    await client.completeAuthorization({ code: "code-1", state: "state-1" });

    expect(observations).toHaveLength(1);
    expect(observations[0]?.snapshot).toEqual({
      status: "authenticated",
      user: user(),
    });
    expect(JSON.parse(observations[0]?.persisted ?? "null")).toMatchObject({
      projection: { status: "authenticated", accessToken: "callback-token" },
    });
    expect(coordinator.entries).toBe(1);
  });

  it("shares a repeated callback while it is in flight", async () => {
    const { client, transport } = setup();
    const gate = deferred<Response>();
    transport.complete.mockReturnValue(gate.promise);

    const first = client.completeAuthorization({
      code: "same-code",
      state: "same-state",
    });
    const second = client.completeAuthorization({
      code: "same-code",
      state: "same-state",
    });
    gate.resolve(replacementResponse("new-token"));

    await Promise.all([first, second]);
    expect(transport.complete).toHaveBeenCalledTimes(1);
  });

  it("rejects authorization before exchange when Web Locks are unavailable", async () => {
    const persistence = new MemoryPersistence();
    const transport = new ScriptedTransport();
    const client = createBrowserSessionClient({
      transport,
      persistence,
      coordinator: {
        runExclusive: () =>
          Promise.reject(new LifecycleCoordinationUnavailableError()),
      },
    });

    await expect(
      client.completeAuthorization({ code: "code", state: "state" }),
    ).rejects.toMatchObject({ code: "AUTHORIZATION_UNAVAILABLE" });
    expect(transport.complete).not.toHaveBeenCalled();
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("does not publish when persistence fails", async () => {
    const { client, persistence } = setup();
    const listener = vi.fn<() => void>();
    client.subscribe(listener);
    persistence.failNextWrite = true;

    await expect(
      client.completeAuthorization({ code: "code", state: "state" }),
    ).rejects.toMatchObject({ code: "PERSISTENCE_UNAVAILABLE" });
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(listener).not.toHaveBeenCalled();
  });

  it("surfaces invalid callback replacements as typed failures", async () => {
    const { client, transport } = setup();
    transport.complete.mockResolvedValue(
      Response.json({ token: "token-without-a-user" }),
    );

    await expect(
      client.completeAuthorization({ code: "code", state: "state" }),
    ).rejects.toMatchObject({ code: "INVALID_SESSION_RESPONSE" });
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("rejects a callback replacement carrying credential extras", async () => {
    const { client, transport } = setup();
    transport.complete.mockResolvedValue(
      Response.json({
        token: "access-token",
        refreshToken: "must-not-cross-into-javascript-state",
        user: user(),
      }),
    );

    await expect(
      client.completeAuthorization({ code: "code", state: "state" }),
    ).rejects.toMatchObject({ code: "INVALID_SESSION_RESPONSE" });
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("normalizes callback network failures", async () => {
    const { client, transport } = setup();
    transport.complete.mockRejectedValue(new TypeError("offline"));

    await expect(
      client.completeAuthorization({ code: "code", state: "state" }),
    ).rejects.toMatchObject({ code: "AUTHORIZATION_FAILED" });
  });

  it.each([
    [403, { error: "Origin is not allowed" }, "Origin is not allowed"],
    [500, {}, "Authentication failed (500)"],
  ])(
    "preserves callback error copy for status %s",
    async (status, body, expectedMessage) => {
      const { client, transport } = setup();
      transport.complete.mockResolvedValue(response(status, body));

      await expect(
        client.completeAuthorization({ code: "code", state: "state" }),
      ).rejects.toMatchObject({
        code: "AUTHORIZATION_FAILED",
        message: expectedMessage,
        status,
      });
    },
  );

  it("rechecks generation after parsing a callback error body", async () => {
    const { client, transport } = setup();
    const delayed = deferredJsonResponse({ error: "stale error" }, 403);
    transport.complete.mockResolvedValue(delayed.response);
    const completion = client.completeAuthorization({
      code: "code",
      state: "state",
    });
    await vi.waitFor(() => expect(transport.complete).toHaveBeenCalledOnce());

    const ending = client.end("current");
    delayed.release();

    await expect(completion).rejects.toMatchObject({
      code: "AUTHORIZATION_OBSOLETE",
    });
    await ending;
  });

  it("cannot resurrect after logout while a callback body is parsing", async () => {
    const persistence = authenticatedPersistence();
    const { client, transport } = setup(persistence);
    const delayed = deferredJsonResponse({
      token: "must-not-publish",
      user: user(),
    });
    transport.complete.mockResolvedValue(delayed.response);

    const completion = client.completeAuthorization({
      code: "code",
      state: "state",
    });
    await vi.waitFor(() => expect(transport.complete).toHaveBeenCalledOnce());
    const ending = client.end("current");
    delayed.release();

    await expect(completion).rejects.toMatchObject({
      code: "AUTHORIZATION_OBSOLETE",
    });
    await ending;
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("anonymous end fences a callback delayed in transport", async () => {
    const { client, transport } = setup();
    const gate = deferred<Response>();
    transport.complete.mockReturnValue(gate.promise);
    const completion = client.completeAuthorization({
      code: "code",
      state: "state",
    });
    await vi.waitFor(() => expect(transport.complete).toHaveBeenCalledOnce());

    const ending = client.end("current");
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    gate.resolve(replacementResponse("must-not-publish"));

    await expect(completion).rejects.toMatchObject({
      code: "AUTHORIZATION_OBSOLETE",
    });
    await ending;
    expect(transport.rotate).toHaveBeenCalledOnce();
    expect(transport.logout).toHaveBeenCalledWith("current", "refreshed-token");
  });

  it("anonymous end fences a callback delayed while parsing its body", async () => {
    const { client, transport } = setup();
    const delayed = deferredJsonResponse({
      token: "must-not-publish",
      user: user(),
    });
    transport.complete.mockResolvedValue(delayed.response);
    const completion = client.completeAuthorization({
      code: "code",
      state: "state",
    });
    await vi.waitFor(() => expect(transport.complete).toHaveBeenCalledOnce());

    const ending = client.end("current");
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    delayed.release();

    await expect(completion).rejects.toMatchObject({
      code: "AUTHORIZATION_OBSOLETE",
    });
    await ending;
    expect(transport.rotate).toHaveBeenCalledOnce();
    expect(transport.logout).toHaveBeenCalledWith("current", "refreshed-token");
  });

  it("waits for a pending end before completing from its committed tombstone", async () => {
    const persistence = authenticatedPersistence("old-token");
    const transport = new ScriptedTransport();
    const coordinator = new SerialCoordinator();
    const blocker = deferred<void>();
    const occupied = coordinator.runExclusive(() => blocker.promise);
    const client = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });

    const ending = client.end("current");
    const completion = client.completeAuthorization({
      code: "new-code",
      state: "new-state",
    });
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
    expect(transport.complete).not.toHaveBeenCalled();

    blocker.resolve();
    await occupied;
    await ending;
    await completion;

    expect(transport.logout).toHaveBeenCalledWith("current", "old-token");
    expect(transport.complete).toHaveBeenCalledWith({
      code: "new-code",
      state: "new-state",
    });
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      revision: "6",
      projection: {
        status: "authenticated",
        accessToken: "callback-token",
      },
    });
  });

  it("cannot replace a newer identity while a callback body is parsing", async () => {
    const persistence = new MemoryPersistence();
    const { client, transport } = setup(persistence);
    const delayed = deferredJsonResponse({
      token: "stale-callback",
      user: user("membership-1"),
    });
    transport.complete.mockResolvedValue(delayed.response);
    const completion = client.completeAuthorization({
      code: "code",
      state: "state",
    });
    await vi.waitFor(() => expect(transport.complete).toHaveBeenCalledOnce());

    persistence.externalWrite(
      envelope(2, {
        status: "authenticated",
        accessToken: "new-identity",
        user: user("membership-2"),
      }),
    );
    delayed.release();

    await expect(completion).rejects.toMatchObject({
      code: "AUTHORIZATION_OBSOLETE",
    });
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user("membership-2"),
    });
  });

  it("orders a two-tab callback before a queued logout without revision collision", async () => {
    const persistence = authenticatedPersistence("original-token");
    persistence.echoWrites = true;
    const transport = new ScriptedTransport();
    const coordinator = new SerialCoordinator();
    const callbackTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    const logoutTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    const delayed = deferredJsonResponse({
      token: "callback-token",
      user: user(),
    });
    transport.complete.mockResolvedValue(delayed.response);

    const completion = callbackTab.completeAuthorization({
      code: "code",
      state: "state",
    });
    await vi.waitFor(() => expect(transport.complete).toHaveBeenCalledOnce());
    const ending = logoutTab.end("current");
    delayed.release();

    await completion;
    await ending;
    expect(callbackTab.getSnapshot()).toEqual({ status: "anonymous" });
    expect(logoutTab.getSnapshot()).toEqual({ status: "anonymous" });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      revision: "6",
      projection: { status: "anonymous" },
    });
    expect(transport.logout).toHaveBeenCalledWith("current", "callback-token");
  });

  it("cleans up an initially anonymous callback completed by another tab", async () => {
    const persistence = new MemoryPersistence();
    persistence.echoWrites = true;
    const transport = new ScriptedTransport();
    const coordinator = new SerialCoordinator();
    const callbackTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    const logoutTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    const gate = deferred<Response>();
    transport.complete.mockReturnValue(gate.promise);

    const completion = callbackTab.completeAuthorization({
      code: "code",
      state: "state",
    });
    await vi.waitFor(() => expect(transport.complete).toHaveBeenCalledOnce());
    const ending = logoutTab.end("current");
    gate.resolve(replacementResponse("callback-token"));

    await completion;
    await ending;
    expect(callbackTab.getSnapshot()).toEqual({ status: "anonymous" });
    expect(logoutTab.getSnapshot()).toEqual({ status: "anonymous" });
    expect(transport.logout).toHaveBeenCalledWith("current", "callback-token");
    expect(transport.rotate).not.toHaveBeenCalled();
  });

  it("rejects a completion retired by local logout", async () => {
    const persistence = authenticatedPersistence();
    const { client, transport } = setup(persistence);
    const gate = deferred<Response>();
    transport.complete.mockReturnValue(gate.promise);

    const completion = client.completeAuthorization({
      code: "code",
      state: "state",
    });
    await Promise.resolve();
    const ending = client.end("current");
    gate.resolve(replacementResponse("must-not-publish"));

    await expect(completion).rejects.toMatchObject({
      code: "AUTHORIZATION_OBSOLETE",
    });
    await ending;
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(transport.rotate).toHaveBeenCalledOnce();
    expect(transport.logout).toHaveBeenCalledWith("current", "refreshed-token");
  });

  it("adopts only valid newer storage transitions and notifies once", () => {
    const persistence = authenticatedPersistence();
    const { client } = setup(persistence);
    const listener = vi.fn<() => void>();
    client.subscribe(listener);

    persistence.externalWrite(
      envelope(5, {
        status: "authenticated",
        accessToken: "rotated",
        user: user("membership-1", "Updated Guardian"),
      }),
    );
    persistence.externalWrite(envelope(5, { status: "anonymous" }));
    persistence.externalWrite(envelope(3, { status: "anonymous" }));
    persistence.externalWrite("not-json");

    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user("membership-1", "Updated Guardian"),
    });
    expect(listener).toHaveBeenCalledTimes(1);
  });

  it("returns a stable snapshot reference until a transition is accepted", () => {
    const persistence = authenticatedPersistence();
    const { client } = setup(persistence);
    const initial = client.getSnapshot();

    expect(client.getSnapshot()).toBe(initial);
    persistence.externalWrite(
      envelope(5, {
        status: "authenticated",
        accessToken: "new-token",
        user: user("membership-2"),
      }),
    );

    expect(client.getSnapshot()).not.toBe(initial);
    expect(client.getSnapshot()).toBe(client.getSnapshot());
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user("membership-2"),
    });
  });
});

describe("BrowserSessionClient authenticated requests", () => {
  it("never probes refresh for an anonymous request", async () => {
    const { client, transport } = setup();
    transport.send.mockResolvedValue(response(401));

    const result = await client.request("/api/protected");

    expect(result.status).toBe(401);
    expect(transport.send).toHaveBeenCalledWith(
      "/api/protected",
      undefined,
      undefined,
    );
    expect(transport.rotate).not.toHaveBeenCalled();
  });

  it("refreshes under coordination, publishes, and retries once", async () => {
    const { client, coordinator, transport } = setup(
      authenticatedPersistence("stale"),
    );
    transport.send.mockImplementation((_input, _init, token) =>
      Promise.resolve(
        token === "refreshed-token" ? response(200) : response(401),
      ),
    );
    const listener = vi.fn<() => void>();
    client.subscribe(listener);

    const result = await client.request("/api/protected");

    expect(result.status).toBe(200);
    expect(transport.rotate).toHaveBeenCalledTimes(1);
    expect(transport.send).toHaveBeenNthCalledWith(
      2,
      "/api/protected",
      undefined,
      "refreshed-token",
    );
    expect(coordinator.entries).toBe(1);
    expect(listener).toHaveBeenCalledTimes(1);
  });

  it("serializes concurrent 401s and reuses the newer token", async () => {
    const { client, transport } = setup(authenticatedPersistence("stale"));
    transport.send.mockImplementation((_input, _init, token) =>
      Promise.resolve(
        token === "refreshed-token" ? response(200) : response(401),
      ),
    );

    const results = await Promise.all([
      client.request("/api/one"),
      client.request("/api/two"),
    ]);

    expect(results.map((item) => item.status)).toEqual([200, 200]);
    expect(transport.rotate).toHaveBeenCalledTimes(1);
  });

  it("does not reuse a token from a same-membership login on a new lineage", async () => {
    const persistence = authenticatedPersistence("old-token");
    const { client, transport } = setup(persistence);
    const delayed401 = deferred<Response>();
    transport.send.mockReturnValueOnce(delayed401.promise);

    const staleRequest = client.request("/api/protected");
    await vi.waitFor(() => expect(transport.send).toHaveBeenCalledOnce());
    await client.end("current");
    await client.completeAuthorization({
      code: "new-code",
      state: "new-state",
    });
    delayed401.resolve(response(401));

    await expect(staleRequest).resolves.toMatchObject({ status: 401 });
    expect(transport.rotate).not.toHaveBeenCalled();
    expect(transport.send).toHaveBeenCalledTimes(1);
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
  });

  it("preserves lineage on refresh and changes it on establishment and end", async () => {
    const persistence = authenticatedPersistence("stale");
    const { client, transport } = setup(persistence);
    const initialLineage = persistedLineage(persistence.raw);
    transport.send.mockImplementation((_input, _init, token) =>
      Promise.resolve(
        token === "refreshed-token" ? response(200) : response(401),
      ),
    );

    await client.request("/api/protected");
    const refreshedLineage = persistedLineage(persistence.raw);
    expect(refreshedLineage).toBe(initialLineage);

    await client.completeAuthorization({ code: "code", state: "state" });
    const establishedLineage = persistedLineage(persistence.raw);
    expect(establishedLineage).not.toBe(refreshedLineage);

    await client.end("current");
    const endedLineage = persistedLineage(persistence.raw);
    expect(endedLineage).not.toBe(establishedLineage);
  });

  it("coordinates two tab clients so only one rotates the cookie", async () => {
    const persistence = authenticatedPersistence("stale");
    persistence.echoWrites = true;
    const transport = new ScriptedTransport();
    const coordinator = new SerialCoordinator();
    const firstTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    const secondTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    transport.send.mockImplementation((_input, _init, token) =>
      Promise.resolve(
        token === "refreshed-token" ? response(200) : response(401),
      ),
    );

    const results = await Promise.all([
      firstTab.request("/api/first-tab"),
      secondTab.request("/api/second-tab"),
    ]);

    expect(results.map((result) => result.status)).toEqual([200, 200]);
    expect(transport.rotate).toHaveBeenCalledTimes(1);
    expect(firstTab.getSnapshot()).toEqual(secondTab.getSnapshot());
  });

  it("re-reads a newer same-membership envelope inside the lock", async () => {
    const persistence = authenticatedPersistence("stale");
    const transport = new ScriptedTransport();
    transport.send.mockImplementation((_input, _init, token) =>
      Promise.resolve(
        token === "other-tab-token" ? response(200) : response(401),
      ),
    );
    const coordinator: BrowserSessionLifecycleCoordinator = {
      runExclusive: (work) => {
        persistence.raw = envelope(5, {
          status: "authenticated",
          accessToken: "other-tab-token",
          user: user(),
        });
        return work();
      },
    };
    const client = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });

    expect((await client.request("/api/protected")).status).toBe(200);
    expect(transport.rotate).not.toHaveBeenCalled();
    expect(transport.send).toHaveBeenLastCalledWith(
      "/api/protected",
      undefined,
      "other-tab-token",
    );
  });

  it("preserves the projection when Web Locks are unavailable", async () => {
    const persistence = authenticatedPersistence("still-valid");
    const transport = new ScriptedTransport();
    transport.send.mockResolvedValue(response(401));
    const client = createBrowserSessionClient({
      transport,
      persistence,
      coordinator: {
        runExclusive: () => {
          throw new LifecycleCoordinationUnavailableError();
        },
      },
    });

    await expect(client.request("/api/protected")).rejects.toMatchObject({
      code: "REFRESH_UNAVAILABLE",
    });
    expect(client.getSnapshot().status).toBe("authenticated");
    expect(transport.rotate).not.toHaveBeenCalled();
  });

  it.each([403, 429, 500])(
    "surfaces refresh %s without clearing the projection",
    async (status) => {
      const { client, transport } = setup(authenticatedPersistence("stale"));
      transport.send.mockResolvedValue(response(401));
      transport.rotate.mockResolvedValue(response(status, { error: "failed" }));

      expect((await client.request("/api/protected")).status).toBe(status);
      expect(client.getSnapshot().status).toBe("authenticated");
    },
  );

  it("clears only its captured generation on definitive refresh 401", async () => {
    const { client, transport } = setup(authenticatedPersistence("stale"));
    transport.send.mockResolvedValue(response(401));
    transport.rotate.mockResolvedValue(response(401));

    await expect(client.request("/api/protected")).rejects.toMatchObject({
      code: "SESSION_EXPIRED",
    });
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("keeps the projection on a refresh network failure", async () => {
    const { client, transport } = setup(authenticatedPersistence("stale"));
    transport.send.mockResolvedValue(response(401));
    transport.rotate.mockRejectedValue(new TypeError("offline"));

    await expect(client.request("/api/protected")).rejects.toMatchObject({
      code: "REFRESH_UNAVAILABLE",
    });
    expect(client.getSnapshot().status).toBe("authenticated");
  });

  it("cannot resurrect after logout while a refresh body is parsing", async () => {
    const { client, transport } = setup(authenticatedPersistence("stale"));
    const delayed = deferredJsonResponse({
      token: "must-not-publish",
      user: user(),
    });
    transport.send.mockResolvedValue(response(401));
    transport.rotate.mockResolvedValue(delayed.response);

    const request = client.request("/api/protected");
    await vi.waitFor(() => expect(transport.rotate).toHaveBeenCalledOnce());
    const ending = client.end("current");
    delayed.release();

    await expect(request).resolves.toMatchObject({ status: 401 });
    await ending;
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(transport.send).toHaveBeenCalledTimes(1);
  });

  it("cannot replace a newer identity while a refresh body is parsing", async () => {
    const persistence = authenticatedPersistence("stale");
    const { client, transport } = setup(persistence);
    const delayed = deferredJsonResponse({
      token: "stale-refresh",
      user: user("membership-1"),
    });
    transport.send.mockResolvedValue(response(401));
    transport.rotate.mockResolvedValue(delayed.response);
    const request = client.request("/api/protected");
    await vi.waitFor(() => expect(transport.rotate).toHaveBeenCalledOnce());

    persistence.externalWrite(
      envelope(5, {
        status: "authenticated",
        accessToken: "new-identity",
        user: user("membership-2"),
      }),
    );
    delayed.release();

    await expect(request).resolves.toMatchObject({ status: 401 });
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user("membership-2"),
    });
  });

  it("orders a two-tab refresh before a queued logout without revision collision", async () => {
    const persistence = authenticatedPersistence("stale");
    persistence.echoWrites = true;
    const transport = new ScriptedTransport();
    const coordinator = new SerialCoordinator();
    const requestTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    const logoutTab = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });
    const delayed = deferredJsonResponse({
      token: "refreshed-token",
      user: user(),
    });
    transport.send.mockImplementation((_input, _init, token) =>
      Promise.resolve(
        token === "refreshed-token" ? response(200) : response(401),
      ),
    );
    transport.rotate.mockResolvedValue(delayed.response);

    const request = requestTab.request("/api/protected");
    await vi.waitFor(() => expect(transport.rotate).toHaveBeenCalledOnce());
    const ending = logoutTab.end("current");
    delayed.release();

    await expect(request).resolves.toMatchObject({ status: 200 });
    await ending;
    expect(requestTab.getSnapshot()).toEqual({ status: "anonymous" });
    expect(logoutTab.getSnapshot()).toEqual({ status: "anonymous" });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      revision: "6",
      projection: { status: "anonymous" },
    });
    expect(transport.logout).toHaveBeenCalledWith("current", "refreshed-token");
  });

  it("does not refresh a Bungie reauthorization response", async () => {
    const { client, transport } = setup(authenticatedPersistence());
    transport.send.mockResolvedValue(
      response(401, { code: "BUNGIE_REAUTH_REQUIRED" }),
    );

    expect((await client.request("/api/protected")).status).toBe(401);
    expect(transport.rotate).not.toHaveBeenCalled();
    expect(client.getSnapshot().status).toBe("authenticated");
  });

  it("ends the refreshed generation after a second protected 401", async () => {
    const { client, transport } = setup(authenticatedPersistence("stale"));
    transport.send.mockResolvedValue(response(401));

    await expect(client.request("/api/protected")).rejects.toMatchObject({
      code: "SESSION_EXPIRED",
    });
    expect(transport.send).toHaveBeenCalledTimes(2);
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });
});

describe("BrowserSessionClient logout", () => {
  it("publishes the durable tombstone before best-effort cleanup settles", async () => {
    const { client, transport } = setup(authenticatedPersistence("old-token"));
    const gate = deferred<Response>();
    transport.logout.mockReturnValue(gate.promise);
    const listener = vi.fn<() => void>();
    client.subscribe(listener);

    const ending = client.end("current");

    await vi.waitFor(() => expect(transport.logout).toHaveBeenCalledOnce());
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(listener).toHaveBeenCalledOnce();
    gate.resolve(response(204));
    await ending;
    expect(transport.logout).toHaveBeenCalledWith("current", "old-token");
  });

  it("uses a stale-token refresh only for cleanup and never republishes it", async () => {
    const { client, transport } = setup(authenticatedPersistence("old-token"));
    transport.logout
      .mockResolvedValueOnce(response(401))
      .mockResolvedValueOnce(response(204));
    const listener = vi.fn<() => void>();
    client.subscribe(listener);

    await client.end("all");

    expect(transport.rotate).toHaveBeenCalledTimes(1);
    expect(transport.logout).toHaveBeenNthCalledWith(
      2,
      "all",
      "refreshed-token",
    );
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(listener).toHaveBeenCalledTimes(1);
  });

  it("stays anonymous when remote cleanup fails", async () => {
    const { client, transport } = setup(authenticatedPersistence());
    transport.logout.mockRejectedValue(new TypeError("offline"));

    await expect(client.end("current")).resolves.toBeUndefined();
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("persists anonymous and skips remote cleanup when coordination is unavailable", async () => {
    const persistence = authenticatedPersistence("token");
    const transport = new ScriptedTransport();
    const client = createBrowserSessionClient({
      transport,
      persistence,
      coordinator: {
        runExclusive: () =>
          Promise.reject(new LifecycleCoordinationUnavailableError()),
      },
    });

    const ending = client.end("current");
    await expect(ending).resolves.toBeUndefined();
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      projection: { status: "anonymous" },
    });
    expect(transport.logout).not.toHaveBeenCalled();
  });

  it("isolates subscriber exceptions and still performs remote cleanup", async () => {
    const { client, transport } = setup(authenticatedPersistence());
    const survivingListener = vi.fn<() => void>();
    client.subscribe(() => {
      throw new Error("consumer failed");
    });
    client.subscribe(survivingListener);

    await client.end("current");

    expect(survivingListener).toHaveBeenCalledOnce();
    expect(transport.logout).toHaveBeenCalledOnce();
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
  });

  it("does not resurface an interim authenticated storage event while end waits", async () => {
    const persistence = authenticatedPersistence("old-token");
    const transport = new ScriptedTransport();
    const coordinator = new SerialCoordinator();
    const blocker = deferred<void>();
    const occupied = coordinator.runExclusive(() => blocker.promise);
    const client = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });

    const ending = client.end("current");
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
    persistence.externalWrite(
      envelope(5, {
        status: "authenticated",
        accessToken: "interim-token",
        user: user(),
      }),
    );
    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });

    await client.request("/api/while-ending");
    expect(transport.send).toHaveBeenCalledWith(
      "/api/while-ending",
      undefined,
      undefined,
    );

    blocker.resolve();
    await occupied;
    await ending;
    expect(client.getSnapshot()).toEqual({ status: "anonymous" });
    expect(transport.logout).toHaveBeenCalledWith("current", "interim-token");
  });

  it("keeps the committed projection and notification state when tombstone persistence fails", async () => {
    const persistence = authenticatedPersistence("old-token");
    const { client, transport } = setup(persistence);
    const listener = vi.fn<() => void>();
    client.subscribe(listener);
    persistence.failNextWrite = true;

    await expect(client.end("current")).rejects.toMatchObject({
      code: "PERSISTENCE_UNAVAILABLE",
    });

    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user(),
    });
    expect(JSON.parse(persistence.raw ?? "null")).toMatchObject({
      revision: "4",
      projection: { status: "authenticated", accessToken: "old-token" },
    });
    expect(listener).not.toHaveBeenCalled();
    expect(transport.logout).not.toHaveBeenCalled();
  });

  it("does not clear or end a newer different membership found inside the lock", async () => {
    const persistence = authenticatedPersistence(
      "old-token",
      user("membership-1"),
    );
    const transport = new ScriptedTransport();
    const coordinator: BrowserSessionLifecycleCoordinator = {
      runExclusive: (work) => {
        persistence.raw = envelope(5, {
          status: "authenticated",
          accessToken: "new-token",
          user: user("membership-2"),
        });
        return work();
      },
    };
    const client = createBrowserSessionClient({
      transport,
      persistence,
      coordinator,
    });

    await client.end("all");

    expect(client.getSnapshot()).toEqual({
      status: "authenticated",
      user: user("membership-2"),
    });
    expect(transport.logout).not.toHaveBeenCalled();
  });
});

describe("BrowserSessionError", () => {
  it("is a typed client failure", () => {
    const error = new BrowserSessionError("REFRESH_UNAVAILABLE", "unavailable");
    expect(error).toBeInstanceOf(Error);
    expect(error.code).toBe("REFRESH_UNAVAILABLE");
  });
});
