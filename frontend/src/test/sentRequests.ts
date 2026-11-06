import { onTestFinished, vi } from "vitest";

type FetchArgs = Parameters<typeof fetch>;
type SentOption = "method" | "signal" | "credentials";

/**
 * One option of a `fetch` call, resolved the way fetch resolves it: `init`
 * wins whenever it sets the option, even to `null` (which clears an input
 * Request's signal); otherwise an input Request supplies its own.
 */
function sentOption<K extends SentOption>(
  [input, init]: FetchArgs,
  key: K,
): RequestInit[K] | Request[K] | undefined {
  if (init?.[key] !== undefined) return init[key];
  return input instanceof Request ? input[key] : undefined;
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
  const claimedCalls = new Set<number>();

  const callFor = (request: Request): FetchArgs => {
    const known = matched.get(request);
    if (known) return known;
    const index = spy.mock.calls.findIndex(
      (call, i) =>
        !claimedCalls.has(i) &&
        (sentOption(call, "method") ?? "GET").toUpperCase() ===
          request.method.toUpperCase() &&
        sentUrl(call) === request.url,
    );
    if (index === -1) {
      throw new Error(
        `No unmatched fetch call for ${request.method} ${request.url} (${claimedCalls.size} already matched)`,
      );
    }
    claimedCalls.add(index);
    const call = spy.mock.calls[index];
    matched.set(request, call);
    return call;
  };

  return {
    /** The AbortSignal the app passed to `fetch` for this handled request. */
    signal(request: Request): AbortSignal {
      const signal = sentOption(callFor(request), "signal");
      if (!signal) {
        throw new Error(
          `The app passed no signal for ${request.method} ${request.url}`,
        );
      }
      return signal;
    },
    /** The `credentials` mode the app passed to `fetch` for this request. */
    credentials(request: Request): RequestCredentials | undefined {
      return sentOption(callFor(request), "credentials");
    },
  };
}

/** Resolves once `signal` aborts, including when it already has. */
export function whenAborted(signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal.aborted) resolve();
    else signal.addEventListener("abort", () => resolve(), { once: true });
  });
}
