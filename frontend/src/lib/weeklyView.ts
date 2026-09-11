import { toDifficulty } from "./difficulty";
import type { APIRecommendedAction, APIWeekly } from "../types/api";
import type { RecommendedAction, Weekly } from "../types/design";

/**
 * The sole adapter for the weekly payload (`APIWeekly` → design `Weekly`).
 *
 * Reset timing, milestones, Xûr inventory and today's actions already arrive in
 * the design vocabulary and pass through unchanged. The recommendation does
 * not: ranked actions serialize the canonical title-case tier (`Challenging`)
 * while the pre-ADR-0016 fallback path serialized lowercase, and the design
 * badge tables are keyed lowercase only. Feature modules used to assert the
 * whole payload into `Weekly`, so a ranked tier reached `Badge` as a key no
 * colour or label table held — the badge lost both without a type error. Doing
 * the projection here is what fixes it, and is why no feature module may cast
 * weekly JSON to a design type again.
 */
export function toWeekly(raw: APIWeekly): Weekly {
  return {
    ...raw,
    recommended: (raw.recommended ?? []).map(toRecommendedAction),
  };
}

function toRecommendedAction(action: APIRecommendedAction): RecommendedAction {
  return { ...action, diff: toDifficulty(action.diff) };
}
