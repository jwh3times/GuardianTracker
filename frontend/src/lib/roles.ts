// Role tier model — port of frontend/design/src/admin-data.js constants.
// Tiers are ordered: standard < beta < alpha < admin. The authoritative role
// always comes from the server (GET /api/flags); these are labels + ordering.

export type Role = "standard" | "beta" | "alpha" | "admin";
/** Tiers a user can self-select / a flag can gate to (admin is never a gate). */
export type Tier = "standard" | "beta" | "alpha";

export const ROLES: Role[] = ["standard", "beta", "alpha", "admin"];
export const MIN_TIERS: Tier[] = ["standard", "beta", "alpha"];

export const ROLE_TIER: Record<Role, number> = {
  standard: 0,
  beta: 1,
  alpha: 2,
  admin: 3,
};

/** Numeric tier for a role; unknown strings resolve to standard (0). */
export const tierOf = (role: string | undefined): number =>
  ROLE_TIER[(role ?? "standard") as Role] ?? 0;

export const ROLE_LABEL: Record<Role, string> = {
  standard: "Standard",
  beta: "Beta",
  alpha: "Alpha",
  admin: "Admin",
};

/** "Standard+" style labels for flag-gate / opt-in segments. */
export const TIER_LABEL: Record<Tier, string> = {
  standard: "Standard+",
  beta: "Beta+",
  alpha: "Alpha+",
};

export const ROLE_COLOR: Record<Role, string> = {
  standard: "var(--c-common)",
  beta: "var(--c-rare)",
  alpha: "var(--c-exotic)",
  admin: "var(--c-admin)",
};

/** Color for a role string, falling back to the standard color. */
export const roleColor = (role: string | undefined): string =>
  ROLE_COLOR[(role ?? "standard") as Role] ?? ROLE_COLOR.standard;
