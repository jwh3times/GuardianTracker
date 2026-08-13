import type { APIAcquisitionSource } from "../types/api";
import type { AcquisitionSource, Difficulty } from "../types/design";

const DIFFICULTY_MAP: Record<string, Difficulty> = {
  Easy: "easy",
  Moderate: "moderate",
  Challenging: "challenging",
  Unrated: "unrated",
};

/** Adapt the canonical API source value without inventing an item-level tier. */
export function toAcquisitionSource(
  source: APIAcquisitionSource,
): AcquisitionSource {
  return {
    text: source.text,
    difficulty: DIFFICULTY_MAP[source.difficulty] ?? "unrated",
    raidDungeon: source.raidDungeon,
  };
}
