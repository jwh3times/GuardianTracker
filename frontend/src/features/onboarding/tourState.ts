export const TOUR_ROUTES = [
  "/dashboard",
  "/this-week",
  "/collections",
] as const;

export type TourState =
  | { phase: "welcome"; step: 0 }
  | { phase: "tour"; step: 0 | 1 | 2 }
  | { phase: "complete"; step: 2 };

export type TourAction =
  { type: "start" } | { type: "next" } | { type: "skip" };

export const initialTourState: TourState = { phase: "welcome", step: 0 };

export function tourReducer(state: TourState, action: TourAction): TourState {
  if (action.type === "skip") return { phase: "complete", step: 2 };
  if (action.type === "start" && state.phase === "welcome") {
    return { phase: "tour", step: 0 };
  }
  if (action.type === "next" && state.phase === "tour") {
    if (state.step === 2) return { phase: "complete", step: 2 };
    return { phase: "tour", step: (state.step + 1) as 1 | 2 };
  }
  return state;
}
