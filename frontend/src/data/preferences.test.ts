import { describe, it, expect } from "vitest";
import { ApiError } from "../lib/api";
import type { BrowserAuthSnapshot } from "../lib/browserSessionClient";
import { sampleUser } from "../test/testServer";
import type {
  APIGuardianTrackerUser,
  APIPreferences,
  APIPreferencesRead,
} from "../types/api";
import {
  PreferenceError,
  createPreferencesClient,
  onboardingRequired,
  type PreferencesPatch,
} from "./preferences";

/**
 * Contract tests for the Preferences client (ADR 0021), over in-memory
 * persistence and a scripted transport. Every ordering question is settled by
 * deciding exactly when each read and write resolves.
 */

interface Deferred<T> {
  promise: Promise<T>;
  resolve(value: T): void;
  reject(error: unknown): void;
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

/** Let settled promises deliver, including a queued microtask flush. */
const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

const memberA: APIGuardianTrackerUser = {
  ...(sampleUser as APIGuardianTrackerUser),
};
const memberB: APIGuardianTrackerUser = {
  ...memberA,
  id: "other",
  membershipId: "other",
};
const keyOf = (user: APIGuardianTrackerUser) =>
  JSON.stringify([user.membershipType, user.membershipId]);
const signedIn = (user: APIGuardianTrackerUser): BrowserAuthSnapshot => ({
  status: "authenticated",
  user,
});

function envelope(
  user: APIGuardianTrackerUser,
  revision: number,
  values: { cardStyle: string; personalize: string },
) {
  return JSON.stringify({
    version: 1,
    membership: keyOf(user),
    revision,
    values,
  });
}

function harness({
  snapshot = signedIn(memberA),
  stored = null as string | null,
} = {}) {
  let current = snapshot;
  let value = stored;
  const storageListeners = new Set<(raw: string | null) => void>();
  const reads: Deferred<APIPreferencesRead>[] = [];
  const writes: { patch: PreferencesPatch; reply: Deferred<APIPreferences> }[] =
    [];
  const client = createPreferencesClient({
    session: { getSnapshot: () => current },
    persistence: {
      read: () => value,
      write: (raw) => {
        value = raw;
      },
      remove: () => {
        value = null;
      },
      subscribe(listener) {
        storageListeners.add(listener);
        return () => {
          storageListeners.delete(listener);
        };
      },
    },
    transport: {
      read() {
        const reply = deferred<APIPreferencesRead>();
        reads.push(reply);
        return reply.promise;
      },
      write(patch) {
        const reply = deferred<APIPreferences>();
        writes.push({ patch, reply });
        return reply.promise;
      },
    },
    now: () => 1_000,
  });
  return {
    client,
    reads,
    writes,
    stored: () => value,
    setSession(next: BrowserAuthSnapshot) {
      current = next;
    },
    emitStorage(raw: string | null) {
      for (const listener of storageListeners) listener(raw);
    },
  };
}

const serverValues = (
  overrides: Partial<APIPreferencesRead> = {},
): APIPreferencesRead => ({
  cardStyle: "framed",
  personalize: true,
  onboardedAt: null,
  persisted: true,
  ...overrides,
});

describe("hydration and identity", () => {
  it("hydrates the current membership's cached slot as unresolved", () => {
    const { client } = harness({
      stored: envelope(memberA, 5, {
        cardStyle: "compact",
        personalize: "off",
      }),
    });

    expect(client.getSnapshot().values).toEqual({
      cardStyle: "compact",
      personalize: "off",
    });
    expect(client.getSnapshot().resolution).toEqual({
      status: "unresolved",
      source: "cache",
    });
  });

  it("replaces another membership's slot, starting from defaults", () => {
    const h = harness({
      stored: envelope(memberB, 5, {
        cardStyle: "compact",
        personalize: "off",
      }),
    });

    expect(h.client.getSnapshot().values).toEqual({
      cardStyle: "framed",
      personalize: "normal",
    });
    expect(h.client.getSnapshot().resolution).toEqual({
      status: "unresolved",
      source: "defaults",
    });
    expect(h.stored()).toBeNull();
  });

  it("ignores a malformed envelope", () => {
    const h = harness({ stored: "{not json" });

    expect(h.client.getSnapshot().resolution).toEqual({
      status: "unresolved",
      source: "defaults",
    });
    expect(h.stored()).toBeNull();
  });

  it("holds no envelope, reads nothing and accepts no write when anonymous", async () => {
    const h = harness({
      snapshot: { status: "anonymous" },
      stored: envelope(memberA, 5, {
        cardStyle: "compact",
        personalize: "off",
      }),
    });

    h.client.ensureRead();
    h.client.setCardStyle("compact");
    await expect(h.client.completeOnboarding()).rejects.toBeInstanceOf(
      PreferenceError,
    );

    expect(h.client.getSnapshot().resolution).toEqual({ status: "anonymous" });
    expect(h.client.getSnapshot().values.cardStyle).toBe("framed");
    expect(h.stored()).toBeNull();
    expect(h.reads).toHaveLength(0);
    expect(h.writes).toHaveLength(0);
  });

  it("re-keys at an identity boundary and drops the departed slot", async () => {
    const h = harness();
    h.client.ensureRead();
    h.reads[0].resolve(serverValues({ cardStyle: "compact" }));
    await flush();
    expect(h.stored()).not.toBeNull();

    h.setSession(signedIn(memberB));
    h.client.reset();

    expect(h.client.getSnapshot().values.cardStyle).toBe("framed");
    expect(h.client.getSnapshot().resolution).toEqual({
      status: "unresolved",
      source: "defaults",
    });
    expect(h.stored()).toBeNull();

    h.setSession({ status: "anonymous" });
    h.client.reset();
    expect(h.client.getSnapshot().resolution).toEqual({ status: "anonymous" });
  });
});

describe("reads and provenance", () => {
  it("reads once per resolved membership", () => {
    const h = harness();

    h.client.ensureRead();
    h.client.ensureRead();
    expect(h.reads).toHaveLength(1);

    h.setSession(signedIn(memberB));
    h.client.reset();
    h.client.ensureRead();
    expect(h.reads).toHaveLength(2);
  });

  it("resolves a persisted read with its onboarding state and caches it", async () => {
    const h = harness();
    h.client.ensureRead();
    h.reads[0].resolve(serverValues({ personalize: false }));
    await flush();

    const { resolution, values } = h.client.getSnapshot();
    expect(resolution).toEqual({
      status: "resolved",
      persisted: true,
      onboarding: "required",
    });
    expect(values.personalize).toBe("off");
    expect(onboardingRequired(resolution)).toBe(true);
    expect(JSON.parse(h.stored()!)).toMatchObject({
      membership: keyOf(memberA),
      values: { cardStyle: "framed", personalize: "off" },
    });
  });

  it("carries a stored completion timestamp and closes the gate", async () => {
    const h = harness();
    h.client.ensureRead();
    h.reads[0].resolve(serverValues({ onboardedAt: "2026-07-12T15:30:00Z" }));
    await flush();

    const { resolution } = h.client.getSnapshot();
    expect(resolution).toEqual({
      status: "resolved",
      persisted: true,
      onboarding: { completedAt: "2026-07-12T15:30:00Z" },
    });
    expect(onboardingRequired(resolution)).toBe(false);
  });

  it("marks a degraded read unpersisted, keeps the cached values and closes the gate", async () => {
    const cached = envelope(memberA, 5, {
      cardStyle: "compact",
      personalize: "off",
    });
    const h = harness({ stored: cached });
    h.client.ensureRead();
    h.reads[0].resolve(serverValues({ persisted: false }));
    await flush();

    const { resolution, values } = h.client.getSnapshot();
    // Degraded defaults co-occur with "required" but must not show the tour.
    expect(resolution).toEqual({
      status: "resolved",
      persisted: false,
      onboarding: "required",
    });
    expect(onboardingRequired(resolution)).toBe(false);
    expect(values).toEqual({ cardStyle: "compact", personalize: "off" });
    expect(h.stored()).toBe(cached);
  });

  it("leaves a failed read unresolved, the gate closed, and does not retry", async () => {
    const h = harness();
    h.client.ensureRead();
    h.reads[0].reject(new ApiError("down", 500));
    await flush();
    h.client.ensureRead();

    expect(h.client.getSnapshot().resolution.status).toBe("unresolved");
    expect(onboardingRequired(h.client.getSnapshot().resolution)).toBe(false);
    expect(h.reads).toHaveLength(1);
  });
});

describe("writes", () => {
  it("applies optimistically and confirms from the server response", async () => {
    const h = harness();

    h.client.setCardStyle("compact");
    expect(h.client.getSnapshot().values.cardStyle).toBe("compact");
    expect(h.client.getSnapshot().save).toEqual({ status: "saving" });
    expect(h.writes.map((w) => w.patch)).toEqual([{ cardStyle: "compact" }]);

    h.writes[0].reply.resolve({
      cardStyle: "compact",
      personalize: true,
      onboardedAt: null,
    });
    await flush();

    expect(h.client.getSnapshot().save).toEqual({ status: "idle" });
    // A successful write is the recovery path that upgrades provenance.
    expect(h.client.getSnapshot().resolution).toEqual({
      status: "resolved",
      persisted: true,
      onboarding: "required",
    });
    expect(
      (JSON.parse(h.stored()!) as { values: { cardStyle: string } }).values
        .cardStyle,
    ).toBe("compact");
  });

  it("accepts a write before the read settles, sending only the changed field", async () => {
    const h = harness();
    h.client.ensureRead();
    h.client.setPersonalize("off");

    // Field presence: writing personalize before reading cannot clobber cardStyle.
    expect(h.writes[0].patch).toEqual({ personalize: false });

    h.reads[0].resolve(
      serverValues({ cardStyle: "compact", personalize: true }),
    );
    await flush();
    // The unsettled change still overlays the value the read confirmed.
    expect(h.client.getSnapshot().values).toEqual({
      cardStyle: "compact",
      personalize: "off",
    });
  });

  it("rolls back and surfaces a typed error on failure, without retrying", async () => {
    const h = harness();
    h.client.setCardStyle("compact");
    h.writes[0].reply.reject(
      new ApiError("database unavailable", 503, "DB_UNAVAILABLE"),
    );
    await flush();

    const { values, save } = h.client.getSnapshot();
    expect(values.cardStyle).toBe("framed");
    expect(save.status).toBe("failed");
    expect(save.status === "failed" && save.error.code).toBe("UNAVAILABLE");
    expect(save.status === "failed" && save.error).toBeInstanceOf(
      PreferenceError,
    );

    await flush();
    expect(h.writes).toHaveLength(1);
  });

  it("classifies a rejected patch separately from an unavailable store", async () => {
    const h = harness();
    h.client.setCardStyle("compact");
    h.writes[0].reply.reject(new ApiError("invalid card style", 400));
    await flush();

    const { save } = h.client.getSnapshot();
    expect(save.status === "failed" && save.error.code).toBe("REJECTED");
  });

  it("keeps one write in flight and coalesces later changes into one patch", async () => {
    const h = harness();
    h.client.setCardStyle("compact");
    h.client.setPersonalize("off");
    h.client.setCardStyle("framed");

    expect(h.writes).toHaveLength(1);
    expect(h.client.getSnapshot().values).toEqual({
      cardStyle: "framed",
      personalize: "off",
    });

    h.writes[0].reply.resolve({
      cardStyle: "compact",
      personalize: true,
      onboardedAt: null,
    });
    await flush();

    expect(h.writes).toHaveLength(2);
    expect(h.writes[1].patch).toEqual({
      personalize: false,
      cardStyle: "framed",
    });
    expect(h.client.getSnapshot().save).toEqual({ status: "saving" });
  });

  it("sends a change queued behind a failed write as a new choice", async () => {
    const h = harness();
    h.client.setCardStyle("compact");
    h.client.setPersonalize("off");
    h.writes[0].reply.reject(new ApiError("down", 503));
    await flush();

    expect(h.writes).toHaveLength(2);
    expect(h.writes[1].patch).toEqual({ personalize: false });
    expect(h.client.getSnapshot().values).toEqual({
      cardStyle: "framed",
      personalize: "off",
    });
  });

  it("never completes onboarding optimistically; the server stamps it", async () => {
    const h = harness();
    h.client.ensureRead();
    h.reads[0].resolve(serverValues());
    await flush();

    let done = false;
    const completion = h.client.completeOnboarding().then(() => {
      done = true;
    });
    expect(h.writes[0].patch).toEqual({ onboardingComplete: true });
    expect(onboardingRequired(h.client.getSnapshot().resolution)).toBe(true);

    h.writes[0].reply.resolve({
      cardStyle: "framed",
      personalize: true,
      onboardedAt: "2026-09-13T10:00:00Z",
    });
    await completion;

    expect(done).toBe(true);
    expect(h.client.getSnapshot().resolution).toEqual({
      status: "resolved",
      persisted: true,
      onboarding: { completedAt: "2026-09-13T10:00:00Z" },
    });
  });

  it("rejects onboarding completion with a typed error when the save fails", async () => {
    const h = harness();
    const completion = h.client.completeOnboarding();
    h.writes[0].reply.reject(new ApiError("down", 503));

    await expect(completion).rejects.toMatchObject({ code: "UNAVAILABLE" });
  });
});

describe("revision ordering and adoption", () => {
  it("does not let a read that settles after a write replace its values", async () => {
    const h = harness();
    h.client.ensureRead();
    h.client.setCardStyle("compact");
    h.writes[0].reply.resolve({
      cardStyle: "compact",
      personalize: true,
      onboardedAt: null,
    });
    await flush();

    h.reads[0].resolve(serverValues({ cardStyle: "framed" }));
    await flush();

    expect(h.client.getSnapshot().values.cardStyle).toBe("compact");
    expect(
      (JSON.parse(h.stored()!) as { values: { cardStyle: string } }).values
        .cardStyle,
    ).toBe("compact");
  });

  it("adopts a newer same-membership envelope and ignores older or foreign ones", () => {
    const h = harness({
      stored: envelope(memberA, 5, {
        cardStyle: "framed",
        personalize: "normal",
      }),
    });

    h.emitStorage(
      envelope(memberA, 10, { cardStyle: "compact", personalize: "off" }),
    );
    expect(h.client.getSnapshot().values).toEqual({
      cardStyle: "compact",
      personalize: "off",
    });

    h.emitStorage(
      envelope(memberA, 7, { cardStyle: "framed", personalize: "normal" }),
    );
    h.emitStorage(
      envelope(memberB, 99, { cardStyle: "framed", personalize: "normal" }),
    );
    h.emitStorage(null);
    expect(h.client.getSnapshot().values).toEqual({
      cardStyle: "compact",
      personalize: "off",
    });
  });

  it("discards work that settles after an identity boundary", async () => {
    const h = harness();
    h.client.ensureRead();
    h.client.setCardStyle("compact");
    const completion = h.client.completeOnboarding();
    // Caught before the boundary rejects it, so the rejection is handled.
    const outcome = completion.catch((error: unknown) => error);

    h.setSession(signedIn(memberB));
    h.client.reset();

    h.reads[0].resolve(serverValues({ cardStyle: "compact" }));
    h.writes[0].reply.resolve({
      cardStyle: "compact",
      personalize: true,
      onboardedAt: "2026-09-13T10:00:00Z",
    });
    await flush();

    expect(h.client.getSnapshot().values.cardStyle).toBe("framed");
    expect(h.client.getSnapshot().resolution.status).toBe("unresolved");
    expect(h.stored()).toBeNull();
    expect(await outcome).toBeInstanceOf(PreferenceError);
  });
});

describe("onboardingRequired", () => {
  it("opens only on stored evidence that onboarding has not happened", () => {
    expect(onboardingRequired({ status: "anonymous" })).toBe(false);
    expect(onboardingRequired({ status: "unresolved", source: "cache" })).toBe(
      false,
    );
    expect(
      onboardingRequired({ status: "unresolved", source: "defaults" }),
    ).toBe(false);
    expect(
      onboardingRequired({
        status: "resolved",
        persisted: false,
        onboarding: "required",
      }),
    ).toBe(false);
    expect(
      onboardingRequired({
        status: "resolved",
        persisted: true,
        onboarding: { completedAt: "2026-07-12T15:30:00Z" },
      }),
    ).toBe(false);
    expect(
      onboardingRequired({
        status: "resolved",
        persisted: true,
        onboarding: "required",
      }),
    ).toBe(true);
  });
});
