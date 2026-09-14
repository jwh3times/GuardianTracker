/**
 * Development-only OAuth callback handoff (issue #317).
 *
 * Bungie redirects only to HTTPS, so local development points its redirect URI
 * at an HTTPS tunnel. Sign-in, though, starts on the local frontend, and the API
 * binds completion to a SameSite=Lax transaction cookie it set there. The
 * callback page loads on the tunnel origin, where completing would be a
 * cross-site request that the browser sends without that cookie — so the API
 * rejects the state. The reconnect intent is lost the same way: it lives in the
 * local origin's sessionStorage.
 *
 * When VITE_OAUTH_COMPLETION_ORIGIN names the origin sign-in started on, a
 * callback that lands anywhere else forwards itself there before doing anything,
 * and completion runs same-site. The destination is build configuration, never
 * part of the URL. Unset — every production build — changes nothing.
 */

export const OAUTH_COMPLETION_ORIGIN: string | undefined = import.meta.env
  .VITE_OAUTH_COMPLETION_ORIGIN;

type CallbackLocation = Pick<
  Location,
  "origin" | "pathname" | "search" | "hash"
>;

/**
 * The URL to forward the callback to, or null when it should complete here.
 *
 * Only a bare http(s) origin is accepted. Anything else is ignored with a
 * warning rather than followed: a path, query, or credentials in the setting
 * would let configuration steer the authorization code somewhere unexpected.
 */
export function completionHandoffTarget(
  configured: string | undefined,
  current: CallbackLocation,
): string | null {
  const value = configured?.trim();
  if (!value) return null;

  let target: URL;
  try {
    target = new URL(value);
  } catch {
    console.warn(
      `Ignoring VITE_OAUTH_COMPLETION_ORIGIN: "${value}" is not a URL.`,
    );
    return null;
  }
  if (target.protocol !== "http:" && target.protocol !== "https:") {
    console.warn(
      `Ignoring VITE_OAUTH_COMPLETION_ORIGIN: "${value}" is not an http(s) origin.`,
    );
    return null;
  }
  if (value.replace(/\/$/, "") !== target.origin) {
    console.warn(
      `Ignoring VITE_OAUTH_COMPLETION_ORIGIN: "${value}" must be a bare origin such as ${target.origin}.`,
    );
    return null;
  }
  if (target.origin === current.origin) return null;

  return `${target.origin}${current.pathname}${current.search}${current.hash}`;
}

/**
 * Forwards the callback to the configured completion origin. Returns true when
 * it navigated, in which case the caller must not attempt completion.
 */
export function handOffOAuthCompletion(
  configured: string | undefined = OAUTH_COMPLETION_ORIGIN,
  location: CallbackLocation & Pick<Location, "replace"> = window.location,
): boolean {
  const target = completionHandoffTarget(configured, location);
  if (!target) return false;
  // replace, not assign: the tunnel copy of the callback must not be a Back
  // target, or Back would resubmit a spent single-use code.
  location.replace(target);
  return true;
}
