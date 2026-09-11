import type { APIDifficulty } from "../types/api";
import type { Difficulty } from "../types/design";

/**
 * Wire difficulty value → design vocabulary.
 *
 * The canonical title-case keys are the tiers `services/sources` emits for both
 * acquisition sources and weekly recommendations. The lowercase keys are the
 * legacy pre-ADR-0016 weekly fallback spellings, tolerated during migration so
 * a stale or replayed response degrades to the right badge instead of to
 * `unrated`.
 *
 * Declared over both unions on purpose: adding a tier to either one fails to
 * compile here rather than silently falling through to `unrated` at runtime.
 */
const WIRE_TIERS: Record<APIDifficulty | Difficulty, Difficulty> = {
  Easy: "easy",
  Moderate: "moderate",
  Challenging: "challenging",
  Unrated: "unrated",
  easy: "easy",
  moderate: "moderate",
  challenging: "challenging",
  unrated: "unrated",
};

/**
 * Looked up through a Map rather than by indexing {@link WIRE_TIERS} directly:
 * the input is arbitrary server-supplied text, and an object index would answer
 * for inherited keys like `toString` or `constructor` with something that is
 * not a tier at all.
 */
const DIFFICULTY_BY_WIRE = new Map<string, Difficulty>(
  Object.entries(WIRE_TIERS),
);

/**
 * The one adapter from a wire difficulty tier to the design vocabulary, shared
 * by acquisition sources and weekly recommendations.
 *
 * An unrecognised or missing value becomes the explicit `unrated` state — the
 * same honest "no keyword matched" answer the backend gives — rather than a
 * guessed tier or a raw string the badge tables cannot look up.
 */
export function toDifficulty(raw: string | undefined | null): Difficulty {
  if (!raw) return "unrated";
  return DIFFICULTY_BY_WIRE.get(raw) ?? "unrated";
}
