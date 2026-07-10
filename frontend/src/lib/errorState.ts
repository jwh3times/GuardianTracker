import { ApiError } from "./api";

/** UI copy for a failed data fetch, branched on the backend's machine-readable code. */
export interface ErrorStateCopy {
  icon: string;
  title: string;
  body: string;
  privacyLink?: boolean;
}

/**
 * Maps an unknown query error to the failure-panel copy, branching on the
 * backend's machine-readable `code` (PRIVACY_RESTRICTION, MANIFEST_NOT_READY,
 * BUNGIE_ERROR, FEATURE_DISABLED, TIER_LOCKED). FEATURE_DISABLED and
 * TIER_LOCKED match on `code` only (never bare status) — the backend always
 * sets these codes when gating a flag, and other endpoints legitimately
 * return bare 404/403 (e.g. ACCOUNT_NOT_FOUND) that must not be mislabeled
 * as a disabled/locked feature. Shared by every page that hits a
 * Bungie-backed endpoint so the copy and branching stay consistent.
 */
export function errorState(error: unknown): ErrorStateCopy {
  if (error instanceof ApiError) {
    if (error.code === "PRIVACY_RESTRICTION") {
      return {
        icon: "lock",
        title: "Your Destiny profile is private",
        body: "Bungie privacy settings are blocking this data. Allow 'Show my Progression' on Bungie.net, then refresh.",
        privacyLink: true,
      };
    }
    if (error.code === "REFRESH_UNAVAILABLE") {
      return {
        icon: "refresh",
        title: "Reconnecting…",
        body: "Your session is refreshing. Give it a moment, then try again.",
      };
    }
    if (error.code === "MANIFEST_NOT_READY" || error.status === 503) {
      return {
        icon: "refresh",
        title: "Warming up…",
        body: "The Destiny item database is still downloading on the server. This takes under a minute — try again shortly.",
      };
    }
    if (error.code === "BUNGIE_ERROR" || error.status === 502) {
      return {
        icon: "info",
        title: "Bungie API unavailable",
        body: "Bungie's servers didn't respond. They may be down for maintenance — try again in a few minutes.",
      };
    }
    if (error.code === "FEATURE_DISABLED") {
      return {
        icon: "info",
        title: "This feature is not available",
        body: "This part of Guardian Tracker is turned off right now. Check back later.",
      };
    }
    if (error.code === "TIER_LOCKED") {
      return {
        icon: "lock",
        title: "Feature locked",
        body: "You don't have access to this feature yet. It may be limited to a higher access tier.",
      };
    }
  }
  return {
    icon: "refresh",
    title: "Couldn't load data",
    body: "We couldn't load this from Bungie. This can happen if your Destiny privacy is restricted or Bungie is unavailable — try refreshing.",
  };
}
