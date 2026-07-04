/**
 * First-run detection for onboarding (ROADMAP §2). A per-membership localStorage
 * marker: absent = this browser has never finished a first Dashboard visit for
 * that account. Deliberately client-side — no server "first login" signal exists
 * and the cost of being wrong is one extra (dismissible) panel.
 */
const key = (membershipId?: string) =>
  `guardian_first_run_done${membershipId ? `:${membershipId}` : ""}`;

export function isFirstRun(membershipId?: string): boolean {
  return localStorage.getItem(key(membershipId)) === null;
}

export function markFirstRunDone(membershipId?: string): void {
  localStorage.setItem(key(membershipId), new Date().toISOString());
}
