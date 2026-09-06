import { QueryClient } from "@tanstack/react-query";
import {
  browserSessionClient,
  BROWSER_SESSION_API_URL,
} from "./browserSessionBrowser";
import { BrowserSessionError } from "./browserSessionClient";
import {
  currentReturnPath,
  markBungieReconnect,
} from "./bungieReauthorization";

/** Base URL of the Go API service. Exported so pre-auth flows (e.g. the OAuth
 *  callback) that can't use apiFetch still derive the host from one place. */
export const API_URL = BROWSER_SESSION_API_URL;

if (import.meta.env.DEV) {
  console.log("API URL:", API_URL);
}

/**
 * ApiError carries the HTTP status and the backend's machine-readable `code`
 * (e.g. PRIVACY_RESTRICTION, MANIFEST_NOT_READY, BUNGIE_ERROR) so callers can
 * branch their error UI instead of showing one generic failure state.
 */
export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
    public retryAfter?: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

interface APIErrorBody {
  error?: string;
  code?: string;
  retryAfter?: number;
}

async function responseErrorBody(res: Response): Promise<APIErrorBody> {
  return (await res.json().catch(() => ({}))) as APIErrorBody;
}

async function redirectForBungieReconnect(res: Response): Promise<void> {
  if (res.status !== 401) return;

  const snapshot = browserSessionClient.getSnapshot();
  const errorBody = await responseErrorBody(res.clone());
  if (errorBody.code !== "BUNGIE_REAUTH_REQUIRED") return;
  // Parsing yields; a response must not route a projection adopted meanwhile.
  if (browserSessionClient.getSnapshot() !== snapshot) {
    throw new ApiError("The browser session changed", 401, "SESSION_CHANGED");
  }

  // Bungie's public-client authorization expired; the Guardian Tracker app
  // session is still valid. Preserve it and route through the dedicated
  // reconnect flow instead of rotating or clearing the app's refresh session.
  markBungieReconnect(currentReturnPath());
  throw new ApiError(
    errorBody.error || "Bungie authorization expired",
    res.status,
    errorBody.code,
    errorBody.retryAfter,
  );
}

export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const headers = new Headers(init?.headers);
  if (!headers.has("Content-Type"))
    headers.set("Content-Type", "application/json");
  let res: Response;
  try {
    res = await browserSessionClient.request(`${API_URL}${path}`, {
      ...init,
      headers,
    });
  } catch (error) {
    if (error instanceof BrowserSessionError) {
      throw new ApiError(error.message, error.status ?? 503, error.code);
    }
    throw error;
  }
  await redirectForBungieReconnect(res);

  if (!res.ok) {
    const errorBody = await responseErrorBody(res);
    throw new ApiError(
      errorBody.error || `API error ${res.status}`,
      res.status,
      errorBody.code,
      errorBody.retryAfter,
    );
  }

  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
    },
  },
});
