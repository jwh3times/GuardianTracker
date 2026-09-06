import "@testing-library/jest-dom";
import { afterAll, afterEach, beforeAll, beforeEach, vi } from "vitest";
import { server } from "./testServer";

// Node 22+ defines an experimental global `localStorage` that is undefined
// unless --localstorage-file is passed, shadowing jsdom's implementation.
// Provide a simple in-memory Storage so app code (auth tokens, prefs,
// character pick, weekly checkmarks) works in tests.
class MemoryStorage implements Storage {
  private store = new Map<string, string>();

  get length(): number {
    return this.store.size;
  }
  clear(): void {
    this.store.clear();
  }
  getItem(key: string): string | null {
    return this.store.has(key) ? this.store.get(key)! : null;
  }
  key(index: number): string | null {
    return [...this.store.keys()][index] ?? null;
  }
  removeItem(key: string): void {
    this.store.delete(key);
  }
  setItem(key: string, value: string): void {
    this.store.set(key, String(value));
  }
}

if (
  typeof globalThis.localStorage === "undefined" ||
  globalThis.localStorage == null
) {
  Object.defineProperty(globalThis, "localStorage", {
    value: new MemoryStorage(),
    writable: true,
    configurable: true,
  });
}

// A single Storage instance is shared across every test in a worker; clear it
// after each test so tokens/prefs written by one test don't bleed into the next.
afterEach(() => {
  globalThis.localStorage?.clear();
});

// One MSW lifecycle for every suite — individual files keep using
// server.use(...) for per-test overrides (resetHandlers clears them).
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// Each test gets a real client with deterministic origin-wide coordination.
// Production remains one eagerly hydrated singleton; test fixtures may seed storage
// before their first render/request without leaking a prior test's projection.
vi.mock("../lib/browserSessionBrowser", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../lib/browserSessionBrowser")>();
  const { createBrowserSessionClient } =
    await import("../lib/browserSessionClient");
  let client: ReturnType<typeof createBrowserSessionClient> | undefined;
  let tail: Promise<unknown> = Promise.resolve();
  beforeEach(() => {
    client = undefined;
    tail = Promise.resolve();
  });
  const getClient = () =>
    (client ??= createBrowserSessionClient({
      persistence: new actual.LocalStorageBrowserSessionPersistence(
        localStorage,
        window,
      ),
      transport: new actual.FetchBrowserSessionAuthTransport(
        actual.BROWSER_SESSION_API_URL,
        (...args) => fetch(...args),
      ),
      coordinator: {
        runExclusive: (work) => {
          const next = tail.then(work);
          tail = next.catch(() => {});
          return next;
        },
      },
    }));
  return {
    ...actual,
    browserSessionClient: {
      getSnapshot: () => getClient().getSnapshot(),
      subscribe: (listener: () => void) => getClient().subscribe(listener),
      beginAuthorization: () => getClient().beginAuthorization(),
      completeAuthorization: (
        input: import("../lib/browserSessionClient").AuthorizationCompletion,
      ) => getClient().completeAuthorization(input),
      request: (input: RequestInfo | URL, init?: RequestInit) =>
        getClient().request(input, init),
      end: (scope: "current" | "all") => getClient().end(scope),
    },
  };
});
