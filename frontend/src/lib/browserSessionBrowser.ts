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

export const BROWSER_SESSION_STORAGE_KEY = "guardian_browser_session";
export const BROWSER_SESSION_LIFECYCLE_LOCK =
  "guardian-tracker-browser-session-lifecycle";
export const BROWSER_SESSION_API_URL =
  import.meta.env.VITE_API_URL || "http://localhost:8081";

const LEGACY_TOKEN_KEY = "guardian_token";
const LEGACY_USER_KEY = "guardian_user";
const OBSOLETE_REFRESH_TOKEN_KEY = "guardian_refresh_token";

export class LocalStorageBrowserSessionPersistence implements BrowserSessionPersistence {
  constructor(
    private readonly storage: Storage = localStorage,
    private readonly events: Window = window,
  ) {}

  read(): string | null {
    return this.storage.getItem(BROWSER_SESSION_STORAGE_KEY);
  }

  write(value: string): void {
    this.storage.setItem(BROWSER_SESSION_STORAGE_KEY, value);
  }

  readLegacy(): { token: string | null; user: string | null } {
    return {
      token: this.storage.getItem(LEGACY_TOKEN_KEY),
      user: this.storage.getItem(LEGACY_USER_KEY),
    };
  }

  clearLegacy(): void {
    this.storage.removeItem(LEGACY_TOKEN_KEY);
    this.storage.removeItem(LEGACY_USER_KEY);
    this.storage.removeItem(OBSOLETE_REFRESH_TOKEN_KEY);
  }

  subscribe(listener: (value: string | null) => void): () => void {
    const onStorage = (event: StorageEvent) => {
      if (
        event.key === BROWSER_SESSION_STORAGE_KEY &&
        (event.storageArea === null || event.storageArea === this.storage)
      ) {
        listener(event.newValue);
      }
    };
    this.events.addEventListener("storage", onStorage);
    return () => this.events.removeEventListener("storage", onStorage);
  }
}

export class WebLocksBrowserSessionCoordinator implements BrowserSessionLifecycleCoordinator {
  constructor(private readonly locks: LockManager | undefined) {}

  runExclusive<T>(work: () => Promise<T>): Promise<T> {
    if (!this.locks) {
      return Promise.reject(new LifecycleCoordinationUnavailableError());
    }
    return this.locks.request(
      BROWSER_SESSION_LIFECYCLE_LOCK,
      { mode: "exclusive" },
      work,
    );
  }
}

function withBearer(
  input: RequestInfo | URL,
  init: RequestInit | undefined,
  accessToken: string | undefined,
): Headers {
  const headers = new Headers(
    typeof Request !== "undefined" && input instanceof Request
      ? input.headers
      : undefined,
  );
  new Headers(init?.headers).forEach((value, key) => headers.set(key, value));
  headers.delete("Authorization");
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  return headers;
}

function apiInput(
  input: RequestInfo | URL,
  baseUrl: string,
): RequestInfo | URL {
  const requestUrl =
    typeof input === "string"
      ? new URL(input, baseUrl)
      : input instanceof URL
        ? input
        : new URL(input.url);
  if (requestUrl.origin !== new URL(baseUrl).origin) {
    throw new BrowserSessionError(
      "CROSS_ORIGIN_REQUEST",
      "Authenticated browser requests must stay on the API origin",
    );
  }
  return typeof input === "string" ? requestUrl.toString() : input;
}

export class FetchBrowserSessionAuthTransport implements BrowserSessionAuthTransport {
  constructor(
    private readonly baseUrl: string,
    private readonly fetchImplementation: typeof fetch = fetch,
  ) {}

  async beginAuthorization(): Promise<AuthorizationStart> {
    const response = await this.fetchImplementation(
      `${this.baseUrl}/api/auth/bungie`,
    );
    if (!response.ok) {
      throw new BrowserSessionError(
        "AUTHORIZATION_FAILED",
        "Could not start Bungie authorization",
        response.status,
      );
    }
    const value = (await response.json()) as unknown;
    if (
      typeof value !== "object" ||
      value === null ||
      !("authUrl" in value) ||
      typeof value.authUrl !== "string" ||
      value.authUrl.length === 0 ||
      !("state" in value) ||
      typeof value.state !== "string" ||
      value.state.length === 0
    ) {
      throw new BrowserSessionError(
        "AUTHORIZATION_FAILED",
        "Authorization setup returned an invalid response",
        response.status,
      );
    }
    return { authUrl: value.authUrl, state: value.state };
  }

  completeAuthorization(input: AuthorizationCompletion): Promise<Response> {
    const body = new URLSearchParams({ code: input.code, state: input.state });
    return this.fetchImplementation(
      `${this.baseUrl}/api/auth/bungie/callback`,
      {
        method: "POST",
        credentials: "include",
        body,
      },
    );
  }

  async request(
    input: RequestInfo | URL,
    init: RequestInit | undefined,
    accessToken: string | undefined,
  ): Promise<Response> {
    const target = apiInput(input, this.baseUrl);
    return this.fetchImplementation(target, {
      ...init,
      credentials: "include",
      headers: withBearer(input, init, accessToken),
    });
  }

  refresh(): Promise<Response> {
    return this.fetchImplementation(`${this.baseUrl}/api/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    });
  }

  end(scope: "current" | "all", accessToken: string): Promise<Response> {
    const suffix = scope === "all" ? "/all" : "";
    return this.fetchImplementation(
      `${this.baseUrl}/api/auth/logout${suffix}`,
      {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${accessToken}`,
        },
      },
    );
  }
}

export function createBrowserLocalStoragePersistence(
  browserWindow: Window = window,
): LocalStorageBrowserSessionPersistence {
  try {
    return new LocalStorageBrowserSessionPersistence(
      browserWindow.localStorage,
      browserWindow,
    );
  } catch (error) {
    throw new BrowserSessionError(
      "PERSISTENCE_UNAVAILABLE",
      "Browser session persistence is unavailable",
      undefined,
      { cause: error },
    );
  }
}

const browserPersistence = createBrowserLocalStoragePersistence();
const browserCoordinator = new WebLocksBrowserSessionCoordinator(
  window.navigator.locks,
);
const browserTransport = new FetchBrowserSessionAuthTransport(
  BROWSER_SESSION_API_URL,
  (...args) => fetch(...args),
);

/** Shared by React projection and the REST response adapter. */
export const browserSessionClient = createBrowserSessionClient({
  transport: browserTransport,
  persistence: browserPersistence,
  coordinator: browserCoordinator,
});
