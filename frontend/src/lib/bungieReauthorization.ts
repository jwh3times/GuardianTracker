const RECONNECT_INTENT_KEY = "guardian_bungie_reconnect";
const RECONNECT_RETURN_TO_KEY = "guardian_bungie_reconnect_return_to";

const FALLBACK_RETURN_TO = "/dashboard";

function safeReturnPath(path: string | null | undefined): string {
  if (!path || !path.startsWith("/") || path.startsWith("//")) {
    return FALLBACK_RETURN_TO;
  }

  const pathname = path.split(/[?#]/, 1)[0];
  if (
    pathname === "/login" ||
    pathname === "/reauthorize" ||
    pathname === "/auth/callback"
  ) {
    return FALLBACK_RETURN_TO;
  }

  return path;
}

export function currentReturnPath(): string {
  return safeReturnPath(
    `${window.location.pathname}${window.location.search}${window.location.hash}`,
  );
}

export function markBungieReconnect(returnTo = FALLBACK_RETURN_TO): void {
  sessionStorage.setItem(RECONNECT_INTENT_KEY, "1");
  if (!sessionStorage.getItem(RECONNECT_RETURN_TO_KEY)) {
    sessionStorage.setItem(RECONNECT_RETURN_TO_KEY, safeReturnPath(returnTo));
  }
}

export function hasBungieReconnectIntent(): boolean {
  return sessionStorage.getItem(RECONNECT_INTENT_KEY) === "1";
}

export function bungieReconnectReturnTo(): string {
  return safeReturnPath(sessionStorage.getItem(RECONNECT_RETURN_TO_KEY));
}

export function clearBungieReconnect(): void {
  sessionStorage.removeItem(RECONNECT_INTENT_KEY);
  sessionStorage.removeItem(RECONNECT_RETURN_TO_KEY);
}
