import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { clearBungieReconnect } from "./bungieReauthorization";
import type {
  BrowserAuthSnapshot,
  BrowserSessionClient,
} from "./browserSessionClient";

// An injected/default client must not be reattached to a different projection
// after the whole application was unmounted and therefore stopped observing.
const clientOwners = new WeakMap<QueryClient, BrowserAuthSnapshot>();

function membership(snapshot: BrowserAuthSnapshot): string {
  return snapshot.status === "anonymous"
    ? "anonymous"
    : JSON.stringify([
        snapshot.user.membershipType,
        snapshot.user.membershipId,
      ]);
}

export interface ApplicationIdentity {
  revision: number;
  client: QueryClient;
  isCurrent(): boolean;
}

/**
 * Application composition owns caches and provider lifetimes, never session
 * transitions. `boundaryResets` are the identity-bound owners that live outside
 * the QueryClient (ADR 0021's preferences client): each runs once when this
 * composition starts, and again at every identity boundary alongside the
 * QueryClient clear. Same-membership refresh runs none of them.
 */
export function createApplicationIdentity(
  session: BrowserSessionClient,
  initialClient: QueryClient,
  boundaryResets: ReadonlyArray<() => void> = [],
) {
  const initialSnapshot = session.getSnapshot();
  let identity = membership(initialSnapshot);
  let observedSnapshot = initialSnapshot;
  const previousOwner = clientOwners.get(initialClient);
  const staleInitialClient =
    previousOwner !== undefined && previousOwner !== initialSnapshot;
  let revision = staleInitialClient ? 1 : 0;
  let scope = makeScope(staleInitialClient ? freshClient() : initialClient);
  let pendingRetirement = staleInitialClient ? initialClient : undefined;
  const listeners = new Set<() => void>();
  let unsubscribe: (() => void) | undefined;

  function runBoundaryResets() {
    for (const reset of boundaryResets) {
      try {
        reset();
      } catch {
        // A module's reset must not retain the departing provider tree.
      }
    }
  }

  // Each registered owner projects the identity this composition starts in.
  runBoundaryResets();

  function freshClient() {
    return new QueryClient({
      defaultOptions: initialClient.getDefaultOptions(),
      queryCache: new QueryCache(initialClient.getQueryCache().config),
      mutationCache: new MutationCache(initialClient.getMutationCache().config),
    });
  }

  function makeScope(client: QueryClient): ApplicationIdentity {
    clientOwners.set(client, session.getSnapshot());
    const capturedRevision = revision;
    const capturedIdentity = identity;
    return {
      revision,
      client,
      isCurrent: () =>
        capturedRevision === revision &&
        capturedIdentity === membership(session.getSnapshot()) &&
        (listeners.size > 0 || observedSnapshot === session.getSnapshot()),
    };
  }

  function synchronize() {
    observedSnapshot = session.getSnapshot();
    const next = membership(observedSnapshot);
    if (next === identity) {
      clientOwners.set(scope.client, session.getSnapshot());
      return;
    }
    identity = next;
    revision += 1;
    // Cancellation stops query delivery; a distinct client also isolates callbacks
    // already executing (notably an optimistic rollback after an awaited mutation).
    const departing = scope.client;
    scope = makeScope(freshClient());
    void departing.cancelQueries({}, { silent: true });
    departing.clear();
    try {
      clearBungieReconnect();
    } catch {
      /* Storage failure cannot retain the departing provider tree. */
    }
    try {
      // Weekly action completion is an identity fact; its legacy keys contain
      // only a reset timestamp. Browser appearance/filter preferences stay put.
      for (let index = localStorage.length - 1; index >= 0; index -= 1) {
        const key = localStorage.key(index);
        if (key?.startsWith("gt_done:")) localStorage.removeItem(key);
      }
    } catch {
      // Storage cleanup must not retain the departing provider tree.
    }
    runBoundaryResets();
    for (const listener of listeners) listener();
  }

  return {
    getSnapshot: () => scope,
    subscribe(listener: () => void): () => void {
      listeners.add(listener);
      if (pendingRetirement) {
        void pendingRetirement.cancelQueries({}, { silent: true });
        pendingRetirement.clear();
        pendingRetirement = undefined;
      }
      unsubscribe ??= session.subscribe(synchronize);
      // Catch a transition between the initial render and subscription.
      synchronize();
      return () => {
        listeners.delete(listener);
        if (listeners.size === 0) {
          unsubscribe?.();
          unsubscribe = undefined;
        }
      };
    },
  };
}
