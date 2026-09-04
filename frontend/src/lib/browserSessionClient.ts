import type { APIGuardianTrackerUser } from "../types/api";

export type BrowserAuthSnapshot =
  | { status: "anonymous" }
  | { status: "authenticated"; user: APIGuardianTrackerUser };

export interface AuthorizationStart {
  authUrl: string;
  state: string;
}

export interface AuthorizationCompletion {
  code: string;
  state: string;
}

export interface BrowserSessionClient {
  getSnapshot(): BrowserAuthSnapshot;
  subscribe(listener: () => void): () => void;
  beginAuthorization(): Promise<AuthorizationStart>;
  completeAuthorization(input: AuthorizationCompletion): Promise<void>;
  request(input: RequestInfo | URL, init?: RequestInit): Promise<Response>;
  end(scope: "current" | "all"): Promise<void>;
}

export type BrowserSessionErrorCode =
  | "AUTHORIZATION_FAILED"
  | "AUTHORIZATION_OBSOLETE"
  | "AUTHORIZATION_UNAVAILABLE"
  | "CROSS_ORIGIN_REQUEST"
  | "INVALID_SESSION_RESPONSE"
  | "PERSISTENCE_UNAVAILABLE"
  | "REFRESH_UNAVAILABLE"
  | "SESSION_EXPIRED";

export class BrowserSessionError extends Error {
  public readonly cause: unknown;

  constructor(
    public readonly code: BrowserSessionErrorCode,
    message: string,
    public readonly status?: number,
    options?: { cause?: unknown },
  ) {
    super(message);
    this.name = "BrowserSessionError";
    this.cause = options?.cause;
  }
}

interface AuthenticatedReplacement {
  token: string;
  user: APIGuardianTrackerUser;
}

export interface BrowserSessionAuthTransport {
  beginAuthorization(): Promise<AuthorizationStart>;
  completeAuthorization(input: AuthorizationCompletion): Promise<Response>;
  request(
    input: RequestInfo | URL,
    init: RequestInit | undefined,
    accessToken: string | undefined,
  ): Promise<Response>;
  refresh(): Promise<Response>;
  end(scope: "current" | "all", accessToken: string): Promise<Response>;
}

export interface BrowserSessionPersistence {
  read(): string | null;
  write(value: string): void;
  readLegacy(): { token: string | null; user: string | null };
  clearLegacy(): void;
  subscribe(listener: (value: string | null) => void): () => void;
}

export interface BrowserSessionLifecycleCoordinator {
  runExclusive<T>(work: () => Promise<T>): Promise<T>;
}

export class LifecycleCoordinationUnavailableError extends Error {
  constructor() {
    super("Origin-wide browser session coordination is unavailable");
    this.name = "LifecycleCoordinationUnavailableError";
  }
}

export interface BrowserSessionClientDependencies {
  transport: BrowserSessionAuthTransport;
  persistence: BrowserSessionPersistence;
  coordinator: BrowserSessionLifecycleCoordinator;
}

const ENVELOPE_VERSION = 1;

type AnonymousProjection = { status: "anonymous" };
type AuthenticatedProjection = {
  status: "authenticated";
  accessToken: string;
  user: APIGuardianTrackerUser;
};
type Projection = AnonymousProjection | AuthenticatedProjection;

interface ProjectionEnvelope {
  schemaVersion: typeof ENVELOPE_VERSION;
  revision: bigint;
  lineage: string;
  projection: Projection;
}

function serializeEnvelope(envelope: ProjectionEnvelope): string {
  return JSON.stringify({
    ...envelope,
    revision: envelope.revision.toString(),
  });
}

const anonymousProjection = (): AnonymousProjection => ({
  status: "anonymous",
});

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: Record<string, unknown>, keys: string[]): boolean {
  const actual = Object.keys(value);
  return actual.length === keys.length && keys.every((key) => key in value);
}

function userFrom(value: unknown): APIGuardianTrackerUser | null {
  if (!isRecord(value)) return null;

  const role = value.role;
  const validRole =
    role === undefined ||
    role === "standard" ||
    role === "beta" ||
    role === "alpha" ||
    role === "admin";

  const hasCredentialExtra = Object.keys(value).some((key) => {
    const normalized = key.toLowerCase();
    return (
      ![
        "id",
        "displayname",
        "membershipid",
        "membershiptype",
        "platform",
        "role",
      ].includes(normalized) &&
      (normalized.includes("token") ||
        normalized.includes("credential") ||
        normalized.includes("secret"))
    );
  });
  if (
    typeof value.id !== "string" ||
    value.id.length === 0 ||
    typeof value.displayName !== "string" ||
    value.displayName.length === 0 ||
    typeof value.membershipId !== "string" ||
    value.membershipId.length === 0 ||
    value.id !== value.membershipId ||
    typeof value.membershipType !== "number" ||
    !Number.isSafeInteger(value.membershipType) ||
    (value.platform !== undefined && typeof value.platform !== "string") ||
    !validRole ||
    hasCredentialExtra
  ) {
    return null;
  }

  return {
    id: value.id,
    displayName: value.displayName,
    membershipId: value.membershipId,
    membershipType: value.membershipType,
    ...(value.platform === undefined ? {} : { platform: value.platform }),
    ...(role === undefined ? {} : { role }),
  };
}

function replacementFrom(value: unknown): AuthenticatedReplacement | null {
  if (
    !isRecord(value) ||
    typeof value.token !== "string" ||
    value.token.length === 0
  ) {
    return null;
  }
  const hasCredentialExtra = Object.keys(value).some((key) => {
    const normalized = key.toLowerCase();
    return (
      normalized !== "token" &&
      (normalized.includes("token") ||
        normalized.includes("credential") ||
        normalized.includes("secret"))
    );
  });
  if (hasCredentialExtra) return null;
  const user = userFrom(value.user);
  return user ? { token: value.token, user } : null;
}

function parseEnvelope(value: string | null): ProjectionEnvelope | null {
  if (value === null) return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    return null;
  }

  if (
    !isRecord(parsed) ||
    !hasOnlyKeys(parsed, [
      "schemaVersion",
      "revision",
      "lineage",
      "projection",
    ]) ||
    parsed.schemaVersion !== ENVELOPE_VERSION ||
    typeof parsed.revision !== "string" ||
    !/^[1-9]\d*$/.test(parsed.revision) ||
    typeof parsed.lineage !== "string" ||
    parsed.lineage.length === 0 ||
    !isRecord(parsed.projection)
  ) {
    return null;
  }

  if (
    parsed.projection.status === "anonymous" &&
    hasOnlyKeys(parsed.projection, ["status"])
  ) {
    return {
      schemaVersion: ENVELOPE_VERSION,
      revision: BigInt(parsed.revision),
      lineage: parsed.lineage,
      projection: anonymousProjection(),
    };
  }

  if (
    parsed.projection.status !== "authenticated" ||
    !hasOnlyKeys(parsed.projection, ["status", "accessToken", "user"]) ||
    typeof parsed.projection.accessToken !== "string" ||
    parsed.projection.accessToken.length === 0 ||
    userFrom(parsed.projection.user) === null
  ) {
    return null;
  }

  const user = userFrom(parsed.projection.user);
  if (!user) return null;
  return {
    schemaVersion: ENVELOPE_VERSION,
    revision: BigInt(parsed.revision),
    lineage: parsed.lineage,
    projection: {
      status: "authenticated",
      accessToken: parsed.projection.accessToken,
      user,
    },
  };
}

function publicSnapshot(projection: Projection): BrowserAuthSnapshot {
  return projection.status === "anonymous"
    ? Object.freeze({ status: "anonymous" })
    : Object.freeze({
        status: "authenticated",
        user: Object.freeze({ ...projection.user }),
      });
}

function sameMembership(
  left: APIGuardianTrackerUser,
  right: APIGuardianTrackerUser,
): boolean {
  return (
    left.membershipId === right.membershipId &&
    left.membershipType === right.membershipType
  );
}

async function responseJson(response: Response): Promise<unknown> {
  try {
    return (await response.clone().json()) as unknown;
  } catch {
    return null;
  }
}

async function isBungieReauthorization(response: Response): Promise<boolean> {
  if (response.status !== 401) return false;
  const body = await responseJson(response);
  return isRecord(body) && body.code === "BUNGIE_REAUTH_REQUIRED";
}

function replayableInput(input: RequestInfo | URL): () => RequestInfo | URL {
  if (typeof Request !== "undefined" && input instanceof Request) {
    const template = input.clone();
    return () => template.clone();
  }
  return () => input;
}

function createLineage(): string {
  return globalThis.crypto.randomUUID();
}

class BrowserSessionClientImplementation implements BrowserSessionClient {
  private envelope: ProjectionEnvelope;
  private snapshot: BrowserAuthSnapshot;
  private generation = 0;
  private pendingWrite: string | null = null;
  private pendingEnds = 0;
  private authorizationCompletionsInFlight = 0;
  private readonly listeners = new Set<() => void>();
  private readonly completions = new Map<string, Promise<void>>();

  constructor(private readonly dependencies: BrowserSessionClientDependencies) {
    this.envelope = this.hydrate();
    this.snapshot = publicSnapshot(this.envelope.projection);
    try {
      dependencies.persistence.subscribe((value) => this.adopt(value));
    } catch (error) {
      throw this.persistenceError(error);
    }
  }

  getSnapshot(): BrowserAuthSnapshot {
    return this.snapshot;
  }

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  async beginAuthorization(): Promise<AuthorizationStart> {
    const generation = this.generation;
    let start: AuthorizationStart;
    try {
      start = await this.dependencies.transport.beginAuthorization();
    } catch (error) {
      this.adoptPersisted();
      if (this.generation !== generation) {
        throw new BrowserSessionError(
          "AUTHORIZATION_OBSOLETE",
          "The browser session changed while authorization was starting",
        );
      }
      if (error instanceof BrowserSessionError) throw error;
      throw new BrowserSessionError(
        "AUTHORIZATION_FAILED",
        "Could not start Bungie authorization",
        undefined,
        { cause: error },
      );
    }
    this.adoptPersisted();
    if (this.generation !== generation) {
      throw new BrowserSessionError(
        "AUTHORIZATION_OBSOLETE",
        "The browser session changed while authorization was starting",
      );
    }
    return start;
  }

  completeAuthorization(input: AuthorizationCompletion): Promise<void> {
    const key = JSON.stringify([input.code, input.state]);
    const existing = this.completions.get(key);
    if (existing) return existing;

    const waitsForPendingEnd = this.pendingEnds > 0;
    const generation = this.generation;
    const operation = this.runCoordinated("authorization", async () => {
      this.adoptPersisted();
      const committedGeneration = waitsForPendingEnd
        ? this.generation
        : generation;
      if (
        waitsForPendingEnd &&
        this.envelope.projection.status !== "anonymous"
      ) {
        throw new BrowserSessionError(
          "AUTHORIZATION_OBSOLETE",
          "Authorization could not follow an uncommitted session end",
        );
      }
      this.assertGeneration(committedGeneration);

      this.authorizationCompletionsInFlight += 1;

      try {
        let response: Response;
        try {
          response =
            await this.dependencies.transport.completeAuthorization(input);
        } catch (error) {
          this.adoptPersisted();
          this.assertGeneration(committedGeneration);
          if (error instanceof BrowserSessionError) throw error;
          throw new BrowserSessionError(
            "AUTHORIZATION_FAILED",
            "Guardian Tracker authorization failed",
            undefined,
            { cause: error },
          );
        }
        this.adoptPersisted();
        this.assertGeneration(committedGeneration);
        if (!response.ok) {
          const body = await responseJson(response);
          this.adoptPersisted();
          this.assertGeneration(committedGeneration);
          const message =
            isRecord(body) &&
            typeof body.error === "string" &&
            body.error.length > 0
              ? body.error
              : `Authentication failed (${response.status})`;
          throw new BrowserSessionError(
            "AUTHORIZATION_FAILED",
            message,
            response.status,
          );
        }

        const replacement = replacementFrom(await responseJson(response));
        this.adoptPersisted();
        this.assertGeneration(committedGeneration);
        if (!replacement) {
          throw new BrowserSessionError(
            "INVALID_SESSION_RESPONSE",
            "Authorization returned an invalid browser session",
            response.status,
          );
        }

        this.publish(
          {
            status: "authenticated",
            accessToken: replacement.token,
            user: replacement.user,
          },
          undefined,
          createLineage(),
        );
      } finally {
        this.authorizationCompletionsInFlight -= 1;
      }
    }).finally(() => {
      if (this.completions.get(key) === operation) {
        this.completions.delete(key);
      }
    });

    this.completions.set(key, operation);
    return operation;
  }

  async request(
    input: RequestInfo | URL,
    init?: RequestInit,
  ): Promise<Response> {
    const captured = this.envelope;
    const generation = this.generation;
    const nextInput = replayableInput(input);
    const accessToken =
      this.pendingEnds === 0 && captured.projection.status === "authenticated"
        ? captured.projection.accessToken
        : undefined;

    const response = await this.dependencies.transport.request(
      nextInput(),
      init,
      accessToken,
    );
    if (
      response.status !== 401 ||
      captured.projection.status === "anonymous" ||
      (await isBungieReauthorization(response))
    ) {
      return response;
    }
    const capturedUser = captured.projection.user;

    return this.runCoordinated("refresh", async () => {
      this.adoptPersisted();

      if (this.generation !== generation) {
        return this.retryWithAdoptedSameMembership(
          capturedUser,
          captured.revision,
          captured.lineage,
          nextInput,
          init,
          response,
        );
      }

      let refreshResponse: Response;
      try {
        refreshResponse = await this.dependencies.transport.refresh();
      } catch (error) {
        this.adoptPersisted();
        if (this.generation !== generation) {
          return this.retryWithAdoptedSameMembership(
            capturedUser,
            captured.revision,
            captured.lineage,
            nextInput,
            init,
            response,
          );
        }
        throw new BrowserSessionError(
          "REFRESH_UNAVAILABLE",
          "Session refresh temporarily unavailable",
          undefined,
          { cause: error },
        );
      }

      this.adoptPersisted();
      if (this.generation !== generation) {
        return this.retryWithAdoptedSameMembership(
          capturedUser,
          captured.revision,
          captured.lineage,
          nextInput,
          init,
          response,
        );
      }

      if (refreshResponse.status === 401) {
        this.endGeneration(generation);
        throw new BrowserSessionError(
          "SESSION_EXPIRED",
          "Session expired",
          401,
        );
      }
      if (!refreshResponse.ok) return refreshResponse;

      const replacement = replacementFrom(await responseJson(refreshResponse));
      this.adoptPersisted();
      if (this.generation !== generation) {
        return this.retryWithAdoptedSameMembership(
          capturedUser,
          captured.revision,
          captured.lineage,
          nextInput,
          init,
          response,
        );
      }
      if (!replacement) {
        throw new BrowserSessionError(
          "INVALID_SESSION_RESPONSE",
          "Refresh returned an invalid browser session",
          refreshResponse.status,
        );
      }
      const refreshedGeneration = this.publish(
        {
          status: "authenticated",
          accessToken: replacement.token,
          user: replacement.user,
        },
        undefined,
        captured.lineage,
      );
      const retried = await this.dependencies.transport.request(
        nextInput(),
        init,
        replacement.token,
      );
      if (retried.status === 401 && !(await isBungieReauthorization(retried))) {
        this.endGeneration(refreshedGeneration);
        throw new BrowserSessionError(
          "SESSION_EXPIRED",
          "Session expired",
          401,
        );
      }
      return retried;
    });
  }

  async end(scope: "current" | "all"): Promise<void> {
    this.adoptPersisted();
    const captured = this.envelope.projection;
    const cleanupInFlightCallback = this.authorizationCompletionsInFlight > 0;

    // Retire request/callback authority immediately. The public projection is
    // changed only after its durable tombstone has been written under the lock.
    this.generation += 1;
    this.pendingEnds += 1;

    let preservedIdentity: ProjectionEnvelope | null = null;
    const coordinatedEnd = async () => {
      const persisted = this.readPersistedEnvelope();
      if (!persisted) throw this.persistenceError();

      if (persisted.revision < this.envelope.revision) {
        throw this.persistenceError();
      }
      if (
        captured.status === "authenticated" &&
        persisted.projection.status === "authenticated" &&
        !sameMembership(captured.user, persisted.projection.user)
      ) {
        preservedIdentity = persisted;
        return;
      }

      const accessToken =
        persisted.projection.status === "authenticated"
          ? persisted.projection.accessToken
          : null;
      this.publish(anonymousProjection(), persisted.revision, createLineage());

      if (cleanupInFlightCallback) {
        try {
          const refreshResponse = await this.dependencies.transport.refresh();
          if (!refreshResponse.ok) return;
          const replacement = replacementFrom(
            await responseJson(refreshResponse),
          );
          if (!replacement) return;
          await this.dependencies.transport.end(scope, replacement.token);
        } catch {
          // The tombstone is durable; callback-created session cleanup is best effort.
        }
        return;
      }

      if (accessToken === null) return;
      try {
        const response = await this.dependencies.transport.end(
          scope,
          accessToken,
        );
        if (response.status !== 401) return;

        const refreshResponse = await this.dependencies.transport.refresh();
        if (!refreshResponse.ok) return;
        const replacement = replacementFrom(
          await responseJson(refreshResponse),
        );
        if (!replacement) return;
        await this.dependencies.transport.end(scope, replacement.token);
      } catch {
        // Logout is locally final. Remote cleanup is deliberately best effort.
      }
    };

    try {
      await this.dependencies.coordinator.runExclusive(coordinatedEnd);
    } catch (error) {
      if (!(error instanceof LifecycleCoordinationUnavailableError))
        throw error;

      // No shared-cookie mutation is safe without origin-wide exclusion. The
      // local tombstone is still durable and logout remains best effort.
      const persisted = this.readPersistedEnvelope();
      if (!persisted) throw this.persistenceError();
      this.publish(anonymousProjection(), persisted.revision, createLineage());
    } finally {
      this.pendingEnds -= 1;
    }

    if (this.pendingEnds === 0 && preservedIdentity !== null) {
      this.adoptEnvelope(preservedIdentity);
    } else if (this.pendingEnds === 0) {
      this.adoptPersisted();
    }
  }

  private hydrate(): ProjectionEnvelope {
    let rawEnvelope: string | null;
    try {
      rawEnvelope = this.dependencies.persistence.read();
    } catch (error) {
      throw this.persistenceError(error);
    }
    const stored = parseEnvelope(rawEnvelope);
    if (stored) {
      this.clearLegacy();
      return stored;
    }

    let projection: Projection = anonymousProjection();
    if (rawEnvelope === null) {
      let legacy: { token: string | null; user: string | null };
      try {
        legacy = this.dependencies.persistence.readLegacy();
      } catch (error) {
        throw this.persistenceError(error);
      }
      if (legacy.token !== null && legacy.user !== null) {
        let user: unknown;
        try {
          user = JSON.parse(legacy.user);
        } catch {
          user = null;
        }
        const migratedUser = userFrom(user);
        if (legacy.token.length > 0 && migratedUser) {
          projection = {
            status: "authenticated",
            accessToken: legacy.token,
            user: migratedUser,
          };
        }
      }
    }

    const repaired: ProjectionEnvelope = {
      schemaVersion: ENVELOPE_VERSION,
      revision: 1n,
      lineage: createLineage(),
      projection,
    };
    try {
      this.dependencies.persistence.write(serializeEnvelope(repaired));
      this.clearLegacy();
    } catch (error) {
      if (error instanceof BrowserSessionError) throw error;
      throw new BrowserSessionError(
        "PERSISTENCE_UNAVAILABLE",
        "Browser session persistence is unavailable",
        undefined,
        { cause: error },
      );
    }
    return repaired;
  }

  private adopt(value: string | null): void {
    // Browser storage events do not echo into their originating document. The
    // guard preserves that contract for deterministic adapters that broadcast.
    if (value === this.pendingWrite) return;
    const incoming = parseEnvelope(value);
    if (!incoming || incoming.revision <= this.envelope.revision) return;
    if (this.pendingEnds > 0) return;
    this.adoptEnvelope(incoming);
  }

  private adoptPersisted(): void {
    const raw = this.readPersisted();
    if (raw !== null && parseEnvelope(raw) === null) {
      throw this.persistenceError();
    }
    this.adopt(raw);
  }

  private publish(
    projection: Projection,
    baseRevision = this.envelope.revision,
    lineage = this.envelope.lineage,
    notify = true,
  ): number {
    const next: ProjectionEnvelope = {
      schemaVersion: ENVELOPE_VERSION,
      revision: baseRevision + 1n,
      lineage,
      projection,
    };
    this.writeEnvelope(next);
    this.envelope = next;
    this.snapshot = publicSnapshot(next.projection);
    this.generation += 1;
    if (notify) this.notify();
    return this.generation;
  }

  private writeEnvelope(envelopeToWrite: ProjectionEnvelope): void {
    const serialized = serializeEnvelope(envelopeToWrite);
    this.pendingWrite = serialized;
    try {
      this.dependencies.persistence.write(serialized);
    } catch (error) {
      throw new BrowserSessionError(
        "PERSISTENCE_UNAVAILABLE",
        "Browser session persistence is unavailable",
        undefined,
        { cause: error },
      );
    } finally {
      this.pendingWrite = null;
    }
  }

  private readPersisted(): string | null {
    try {
      return this.dependencies.persistence.read();
    } catch (error) {
      throw this.persistenceError(error);
    }
  }

  private readPersistedEnvelope(): ProjectionEnvelope | null {
    const raw = this.readPersisted();
    if (raw === null) return null;
    const persisted = parseEnvelope(raw);
    if (!persisted) throw this.persistenceError();
    return persisted;
  }

  private clearLegacy(): void {
    try {
      this.dependencies.persistence.clearLegacy();
    } catch (error) {
      throw this.persistenceError(error);
    }
  }

  private persistenceError(cause?: unknown): BrowserSessionError {
    return new BrowserSessionError(
      "PERSISTENCE_UNAVAILABLE",
      "Browser session persistence is unavailable",
      undefined,
      { cause },
    );
  }

  private endGeneration(generation: number): void {
    if (this.generation === generation) {
      this.publish(anonymousProjection(), undefined, createLineage());
    }
  }

  private async retryWithAdoptedSameMembership(
    originalUser: APIGuardianTrackerUser,
    originalRevision: bigint,
    originalLineage: string,
    nextInput: () => RequestInfo | URL,
    init: RequestInit | undefined,
    originalResponse: Response,
  ): Promise<Response> {
    const projection = this.envelope.projection;
    if (
      this.envelope.revision <= originalRevision ||
      this.envelope.lineage !== originalLineage ||
      projection.status !== "authenticated" ||
      !sameMembership(originalUser, projection.user)
    ) {
      return originalResponse;
    }

    const generation = this.generation;
    const response = await this.dependencies.transport.request(
      nextInput(),
      init,
      projection.accessToken,
    );
    if (response.status === 401 && !(await isBungieReauthorization(response))) {
      this.endGeneration(generation);
      throw new BrowserSessionError("SESSION_EXPIRED", "Session expired", 401);
    }
    return response;
  }

  private assertGeneration(generation: number): void {
    if (this.generation !== generation) {
      throw new BrowserSessionError(
        "AUTHORIZATION_OBSOLETE",
        "The browser session changed before authorization completed",
      );
    }
  }

  private adoptEnvelope(incoming: ProjectionEnvelope): void {
    if (incoming.revision <= this.envelope.revision) return;
    this.envelope = incoming;
    this.snapshot = publicSnapshot(incoming.projection);
    this.generation += 1;
    this.notify();
  }

  private async runCoordinated<T>(
    operation: "authorization" | "refresh",
    work: () => Promise<T>,
  ): Promise<T> {
    try {
      return await this.dependencies.coordinator.runExclusive(work);
    } catch (error) {
      if (error instanceof LifecycleCoordinationUnavailableError) {
        const authorization = operation === "authorization";
        throw new BrowserSessionError(
          authorization ? "AUTHORIZATION_UNAVAILABLE" : "REFRESH_UNAVAILABLE",
          authorization
            ? "Authorization requires origin-wide browser session coordination"
            : "Origin-wide browser session coordination is unavailable",
          undefined,
          { cause: error },
        );
      }
      throw error;
    }
  }

  private notify(): void {
    for (const listener of this.listeners) {
      try {
        listener();
      } catch {
        // One consumer must not interrupt publication or another consumer.
      }
    }
  }
}

export function createBrowserSessionClient(
  dependencies: BrowserSessionClientDependencies,
): BrowserSessionClient {
  return new BrowserSessionClientImplementation(dependencies);
}
