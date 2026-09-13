import { useContext, useEffect, useMemo, useSyncExternalStore } from "react";
import { IdentityContext } from "../contexts/IdentityMutation";
import { ApiError, apiFetch } from "../lib/api";
import { browserSessionClient } from "../lib/browserSessionBrowser";
import type { BrowserAuthSnapshot } from "../lib/browserSessionClient";
import type { APIPreferences, APIPreferencesRead } from "../types/api";

/**
 * The Preferences client (ADR 0021). One framework-neutral owner of the browser
 * preference projection and every transition that can replace it: synchronous
 * single-slot hydration keyed by Destiny membership, one read per resolved
 * membership, an optimistic single-flight write queue that coalesces changes
 * and rolls back with a typed error, cross-tab adoption, and revision ordering
 * across all three publishers.
 *
 * It deliberately does not use React Query: preferences are one small object
 * with no shared cache base and no invalidation fan-out, and they need
 * behaviours React Query does not provide. It lives in `src/data/` so its hook
 * sits behind the same import boundary as every other data-access module.
 *
 * It never derives an identity boundary itself. ADR 0017's composition
 * registers {@link PreferencesClient.reset} and calls it when the session
 * becomes anonymous or changes Destiny membership. Preferences are exempt from
 * the membership-refresh fan-out: a Bungie data refresh cannot change a
 * Guardian Tracker setting.
 */

export type CardStyle = "framed" | "compact";
export type Personalize = "off" | "normal";

export interface PreferenceValues {
  /** Item-card density on Collections: full framed cards or condensed rows. */
  cardStyle: CardStyle;
  /** Whether the "for you" personalization badges (Missing / Avail-now) show. */
  personalize: Personalize;
}

export const DEFAULT_PREFERENCES: PreferenceValues = {
  cardStyle: "framed",
  personalize: "normal",
};

/** How far the projection is from the server's stored state. */
export type PreferencesResolution =
  | { status: "anonymous" }
  | { status: "unresolved"; source: "cache" | "defaults" }
  | {
      status: "resolved";
      /** False when persistence was unavailable and the values are unstored. */
      persisted: boolean;
      onboarding: "required" | { completedAt: string };
    };

export type PreferenceErrorCode = "UNAVAILABLE" | "REJECTED" | "FAILED";

/** Why a preference change did not reach the server. */
export class PreferenceError extends Error {
  readonly cause: unknown;

  constructor(
    message: string,
    readonly code: PreferenceErrorCode,
    options?: { cause?: unknown },
  ) {
    super(message);
    this.name = "PreferenceError";
    this.cause = options?.cause;
  }
}

/** Orthogonal to resolution: a write is accepted while the read is in flight. */
export type PreferencesSave =
  | { status: "idle" }
  | { status: "saving" }
  | { status: "failed"; error: PreferenceError };

export interface PreferencesSnapshot {
  /** Always present, so a consumer renders with no loading branch. */
  readonly values: PreferenceValues;
  readonly resolution: PreferencesResolution;
  readonly save: PreferencesSave;
}

/**
 * The fail-closed onboarding gate. Only positive, stored evidence that the
 * member has not onboarded shows the tour; an anonymous, unresolved, failed or
 * degraded read never does.
 */
export function onboardingRequired(resolution: PreferencesResolution): boolean {
  return (
    resolution.status === "resolved" &&
    resolution.persisted &&
    resolution.onboarding === "required"
  );
}

/** A field-presence partial patch, matching the backend's atomic `Apply`. */
export interface PreferencesPatch {
  cardStyle?: CardStyle;
  personalize?: boolean;
  onboardingComplete?: true;
}

export interface PreferencesTransport {
  read(): Promise<APIPreferencesRead>;
  write(patch: PreferencesPatch): Promise<APIPreferences>;
}

export interface PreferencesPersistence {
  read(): string | null;
  write(value: string): void;
  remove(): void;
  /** Envelope changes made by other tabs. */
  subscribe(listener: (value: string | null) => void): () => void;
}

export interface PreferencesSession {
  getSnapshot(): BrowserAuthSnapshot;
}

export interface PreferencesClientDependencies {
  transport: PreferencesTransport;
  persistence: PreferencesPersistence;
  session: PreferencesSession;
  /** Revision clock. Revisions are monotonic within a tab regardless. */
  now?: () => number;
}

export interface PreferencesClient {
  getSnapshot: () => PreferencesSnapshot;
  subscribe: (listener: () => void) => () => void;
  /** Read once for the current membership; a no-op if already issued. */
  ensureRead: () => void;
  setCardStyle: (cardStyle: CardStyle) => void;
  setPersonalize: (personalize: Personalize) => void;
  /** Resolves once the server has stamped completion; never optimistic. */
  completeOnboarding: () => Promise<void>;
  /** The identity boundary, called by ADR 0017's composition. */
  reset: () => void;
}

const ENVELOPE_VERSION = 1;

/** The cache of the last server-confirmed values. Never evidence of storage. */
interface Envelope {
  version: typeof ENVELOPE_VERSION;
  membership: string;
  revision: number;
  values: PreferenceValues;
}

function parseEnvelope(raw: string | null): Envelope | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (typeof parsed !== "object" || parsed === null) return null;
    const envelope = parsed as Record<string, unknown>;
    const values = envelope.values as Record<string, unknown> | null;
    if (
      envelope.version !== ENVELOPE_VERSION ||
      typeof envelope.membership !== "string" ||
      typeof envelope.revision !== "number" ||
      !Number.isFinite(envelope.revision) ||
      typeof values !== "object" ||
      values === null ||
      (values.cardStyle !== "framed" && values.cardStyle !== "compact") ||
      (values.personalize !== "normal" && values.personalize !== "off")
    ) {
      return null;
    }
    return {
      version: ENVELOPE_VERSION,
      membership: envelope.membership,
      revision: envelope.revision,
      values: {
        cardStyle: values.cardStyle,
        personalize: values.personalize,
      },
    };
  } catch {
    return null;
  }
}

function membershipKey(snapshot: BrowserAuthSnapshot): string | null {
  return snapshot.status === "anonymous"
    ? null
    : JSON.stringify([
        snapshot.user.membershipType,
        snapshot.user.membershipId,
      ]);
}

function toValues(wire: APIPreferences): PreferenceValues {
  return {
    cardStyle: wire.cardStyle === "compact" ? "compact" : "framed",
    personalize: wire.personalize ? "normal" : "off",
  };
}

function toResolution(
  wire: APIPreferences,
  persisted: boolean,
): PreferencesResolution {
  return {
    status: "resolved",
    persisted,
    onboarding: wire.onboardedAt
      ? { completedAt: wire.onboardedAt }
      : "required",
  };
}

function toPreferenceError(error: unknown): PreferenceError {
  if (error instanceof PreferenceError) return error;
  if (error instanceof ApiError) {
    const code: PreferenceErrorCode =
      error.status === 503
        ? "UNAVAILABLE"
        : error.status >= 400 && error.status < 500
          ? "REJECTED"
          : "FAILED";
    return new PreferenceError(error.message, code, { cause: error });
  }
  return new PreferenceError(
    error instanceof Error ? error.message : "Preference update failed",
    "FAILED",
    { cause: error },
  );
}

const IDLE: PreferencesSave = { status: "idle" };
const SAVING: PreferencesSave = { status: "saving" };

interface Waiter {
  resolve(): void;
  reject(error: PreferenceError): void;
}

function isEmpty(patch: PreferencesPatch): boolean {
  return Object.keys(patch).length === 0;
}

function sessionChanged(): PreferenceError {
  return new PreferenceError("The browser session changed", "FAILED");
}

export function createPreferencesClient({
  transport,
  persistence,
  session,
  now = Date.now,
}: PreferencesClientDependencies): PreferencesClient {
  const listeners = new Set<() => void>();
  // Bumped at every identity boundary; settling work from an earlier epoch is
  // discarded rather than published into the next membership's projection.
  let epoch = 0;
  let membership: string | null = null;
  let confirmed: PreferenceValues = DEFAULT_PREFERENCES;
  let confirmedRevision = 0;
  let lastRevision = 0;
  let resolution: PreferencesResolution = { status: "anonymous" };
  let save: PreferencesSave = IDLE;
  let readEpoch = -1;
  let pending: PreferencesPatch = {};
  let pendingWaiters: Waiter[] = [];
  let inflight: PreferencesPatch | null = null;
  let snapshot: PreferencesSnapshot = {
    values: confirmed,
    resolution,
    save,
  };

  function nextRevision(): number {
    lastRevision = Math.max(lastRevision + 1, now());
    return lastRevision;
  }

  /** The confirmed values with every unsettled change laid over them. */
  function displayed(): PreferenceValues {
    let values = confirmed;
    for (const patch of [inflight, pending]) {
      if (!patch) continue;
      if (patch.cardStyle !== undefined) {
        values = { ...values, cardStyle: patch.cardStyle };
      }
      if (patch.personalize !== undefined) {
        values = {
          ...values,
          personalize: patch.personalize ? "normal" : "off",
        };
      }
    }
    return values;
  }

  function publish() {
    snapshot = { values: displayed(), resolution, save };
    for (const listener of listeners) listener();
  }

  function writeEnvelope() {
    if (membership === null) return;
    const envelope: Envelope = {
      version: ENVELOPE_VERSION,
      membership,
      revision: confirmedRevision,
      values: confirmed,
    };
    try {
      persistence.write(JSON.stringify(envelope));
    } catch {
      // The in-memory projection stays authoritative for this tab.
    }
  }

  function removeEnvelope() {
    try {
      persistence.remove();
    } catch {
      // A storage failure must not block the identity boundary.
    }
  }

  function rejectAll(waiters: Waiter[], error: PreferenceError) {
    for (const waiter of waiters) waiter.reject(error);
  }

  /** Project the current session: one slot, or none when anonymous. */
  function hydrate() {
    membership = membershipKey(session.getSnapshot());
    pending = {};
    inflight = null;
    readEpoch = -1;
    save = IDLE;
    if (membership === null) {
      removeEnvelope();
      confirmed = DEFAULT_PREFERENCES;
      confirmedRevision = 0;
      resolution = { status: "anonymous" };
      return;
    }
    let raw: string | null;
    try {
      raw = persistence.read();
    } catch {
      raw = null;
    }
    const envelope = parseEnvelope(raw);
    if (envelope && envelope.membership === membership) {
      confirmed = envelope.values;
      confirmedRevision = envelope.revision;
      lastRevision = Math.max(lastRevision, envelope.revision);
      resolution = { status: "unresolved", source: "cache" };
      return;
    }
    // A slot for another membership (or an unreadable one) is replaced.
    if (raw !== null) removeEnvelope();
    confirmed = DEFAULT_PREFERENCES;
    confirmedRevision = 0;
    resolution = { status: "unresolved", source: "defaults" };
  }

  function adopt(raw: string | null) {
    if (membership === null) return;
    const envelope = parseEnvelope(raw);
    if (
      !envelope ||
      envelope.membership !== membership ||
      envelope.revision <= confirmedRevision
    ) {
      return;
    }
    confirmed = envelope.values;
    confirmedRevision = envelope.revision;
    lastRevision = Math.max(lastRevision, envelope.revision);
    publish();
  }

  function flush() {
    if (inflight || membership === null || isEmpty(pending)) return;
    const patch = pending;
    const waiters = pendingWaiters;
    pending = {};
    pendingWaiters = [];
    inflight = patch;
    save = SAVING;
    const writingEpoch = epoch;
    publish();
    transport.write(patch).then(
      (wire) => {
        if (writingEpoch !== epoch) {
          rejectAll(waiters, sessionChanged());
          return;
        }
        inflight = null;
        // A patch response is the server's post-patch state: the newest
        // revision this tab knows, and proof that persistence is available.
        confirmed = toValues(wire);
        confirmedRevision = nextRevision();
        resolution = toResolution(wire, true);
        writeEnvelope();
        for (const waiter of waiters) waiter.resolve();
        if (!isEmpty(pending)) {
          flush();
          return;
        }
        save = IDLE;
        publish();
      },
      (error: unknown) => {
        if (writingEpoch !== epoch) {
          rejectAll(waiters, sessionChanged());
          return;
        }
        // Roll back: dropping the in-flight patch returns the rendered state
        // to the last one the server confirmed. There is no automatic retry.
        inflight = null;
        const failure = toPreferenceError(error);
        save = { status: "failed", error: failure };
        rejectAll(waiters, failure);
        publish();
        // A change queued meanwhile is a new choice, not a retry of this one.
        if (!isEmpty(pending)) queueMicrotask(flush);
      },
    );
  }

  function enqueue(patch: PreferencesPatch, waiter?: Waiter) {
    pending = { ...pending, ...patch };
    if (waiter) pendingWaiters.push(waiter);
    if (inflight) {
      publish();
      return;
    }
    flush();
  }

  hydrate();
  snapshot = { values: displayed(), resolution, save };
  persistence.subscribe(adopt);

  return {
    getSnapshot: () => snapshot,

    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },

    ensureRead() {
      if (membership === null || readEpoch === epoch) return;
      readEpoch = epoch;
      const readingEpoch = epoch;
      // Taken at dispatch: a read that settles after a later write or adopted
      // envelope is older than both and must not replace their values.
      const token = nextRevision();
      transport.read().then(
        (wire) => {
          if (readingEpoch !== epoch) return;
          const persisted = wire.persisted === true;
          if (resolution.status !== "resolved") {
            resolution = toResolution(wire, persisted);
          }
          // Degraded defaults are not the member's values, so they neither
          // replace what is displayed nor enter the cache.
          if (persisted && token > confirmedRevision) {
            confirmed = toValues(wire);
            confirmedRevision = token;
            writeEnvelope();
          }
          publish();
        },
        () => {
          // A failed read stays unresolved, so the onboarding gate fails
          // closed. It is not retried: one read per resolved membership.
        },
      );
    },

    setCardStyle(cardStyle) {
      if (membership !== null) enqueue({ cardStyle });
    },

    setPersonalize(personalize) {
      if (membership !== null)
        enqueue({ personalize: personalize === "normal" });
    },

    completeOnboarding() {
      if (membership === null) {
        return Promise.reject(
          new PreferenceError("No signed-in member to onboard", "REJECTED"),
        );
      }
      return new Promise<void>((resolve, reject) => {
        enqueue({ onboardingComplete: true }, { resolve, reject });
      });
    },

    reset() {
      epoch += 1;
      const departedWaiters = pendingWaiters;
      hydrate();
      pendingWaiters = [];
      rejectAll(departedWaiters, sessionChanged());
      publish();
    },
  };
}

export const PREFERENCES_STORAGE_KEY = "guardian_preferences";

/** The pre-ADR-0021 global key, removed rather than migrated. */
const LEGACY_PREFERENCES_KEY = "guardian_prefs";

class LocalStoragePreferencesPersistence implements PreferencesPersistence {
  constructor(
    private readonly storage: Storage,
    private readonly events: Window,
  ) {}

  read(): string | null {
    return this.storage.getItem(PREFERENCES_STORAGE_KEY);
  }

  write(value: string): void {
    this.storage.setItem(PREFERENCES_STORAGE_KEY, value);
  }

  remove(): void {
    this.storage.removeItem(PREFERENCES_STORAGE_KEY);
  }

  subscribe(listener: (value: string | null) => void): () => void {
    const onStorage = (event: StorageEvent) => {
      if (
        event.key === PREFERENCES_STORAGE_KEY &&
        (event.storageArea === null || event.storageArea === this.storage)
      ) {
        listener(event.newValue);
      }
    };
    this.events.addEventListener("storage", onStorage);
    return () => this.events.removeEventListener("storage", onStorage);
  }
}

try {
  localStorage.removeItem(LEGACY_PREFERENCES_KEY);
} catch {
  // Unavailable storage leaves nothing to clean up.
}

/** The production singleton, shared by every React consumer. */
export const preferencesClient: PreferencesClient = createPreferencesClient({
  transport: {
    read: () => apiFetch<APIPreferencesRead>("/api/preferences"),
    write: (patch) =>
      apiFetch<APIPreferences>("/api/preferences", {
        method: "PUT",
        body: JSON.stringify(patch),
      }),
  },
  persistence: new LocalStoragePreferencesPersistence(localStorage, window),
  session: browserSessionClient,
});

/**
 * The member's preferences, their provenance and save state, and the actions
 * that change them.
 *
 * Actions are fenced by the identity scope they were rendered in, so a setter
 * captured before a membership change cannot write for the next membership.
 */
export function usePreferences() {
  const scope = useContext(IdentityContext);
  if (!scope) {
    throw new Error("usePreferences must be used within AppProviders");
  }
  const snapshot = useSyncExternalStore(
    preferencesClient.subscribe,
    preferencesClient.getSnapshot,
    preferencesClient.getSnapshot,
  );

  // A reset publishes a new resolution object, which re-issues the read for
  // the next membership; for the same membership this is a no-op.
  useEffect(() => {
    preferencesClient.ensureRead();
  }, [snapshot.resolution]);

  const actions = useMemo(
    () => ({
      setCardStyle: (cardStyle: CardStyle) => {
        if (scope.isCurrent()) preferencesClient.setCardStyle(cardStyle);
      },
      setPersonalize: (personalize: Personalize) => {
        if (scope.isCurrent()) preferencesClient.setPersonalize(personalize);
      },
      completeOnboarding: (): Promise<void> =>
        scope.isCurrent()
          ? preferencesClient.completeOnboarding()
          : Promise.reject(sessionChanged()),
    }),
    [scope],
  );

  return { ...snapshot, ...actions };
}
