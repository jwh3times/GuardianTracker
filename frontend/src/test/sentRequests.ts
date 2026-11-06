import { onTestFinished, vi } from "vitest";

type FetchArgs = Parameters<typeof fetch>;

function sentMethod([input, init]: FetchArgs): string {
  return (
    init?.method ?? (input instanceof Request ? input.method : "GET")
  ).toUpperCase();
}

function sentUrl([input]: FetchArgs): string {
  if (input instanceof Request) return input.url;
  return new URL(input).href;
}

/**
 * Observes what the app handed to `fetch`, for the facts a handler can no
 * longer see. In Node, MSW 3 intercepts at the socket and gives handlers a
 * Request rebuilt from the wire: it carries its own AbortSignal, and
 * client-only options such as `credentials` never reach it.
 *
 * Call it before rendering. Inside a handler, pass the handler's `request` to
 * look up the matching `fetch` call; calls to the same method and URL are
 * matched to handler invocations in the order they were made.
 */
export function observeSentRequests() {
  const spy = vi.spyOn(globalThis, "fetch");
  onTestFinished(() => spy.mockRestore());

  const matched = new WeakMap<Request, FetchArgs>();
  let claimed = 0;
  const claimedCalls = new Set<number>();

  const callFor = (request: Request): FetchArgs => {
    const known = matched.get(request);
    if (known) return known;
    const index = spy.mock.calls.findIndex(
      (call, i) =>
        !claimedCalls.has(i) &&
        sentMethod(call) === request.method.toUpperCase() &&
        sentUrl(call) === request.url,
    );
    if (index === -1) {
      throw new Error(
        `No unmatched fetch call for ${request.method} ${request.url} (${claimed} already matched)`,
      );
    }
    claimedCalls.add(index);
    claimed++;
    const call = spy.mock.calls[index];
    matched.set(request, call);
    return call;
  };

  return {
    /** The AbortSignal the app passed to `fetch` for this handled request. */
    signal(request: Request): AbortSignal {
      const [input, init] = callFor(request);
      const signal =
        init?.signal ?? (input instanceof Request ? input.signal : undefined);
      if (!signal) {
        throw new Error(
          `The app passed no signal for ${request.method} ${request.url}`,
        );
      }
      return signal;
    },
    /** The `credentials` mode the app passed to `fetch` for this request. */
    credentials(request: Request): RequestCredentials | undefined {
      const [input, init] = callFor(request);
      return (
        init?.credentials ??
        (input instanceof Request ? input.credentials : undefined)
      );
    },
  };
}

/** Resolves once `signal` aborts, including when it already has. */
export function aborted(signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal.aborted) resolve();
    else signal.addEventListener("abort", () => resolve(), { once: true });
  });
}
