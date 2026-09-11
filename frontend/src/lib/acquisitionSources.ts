import { toDifficulty } from "./difficulty";
import type { APIAcquisitionSource } from "../types/api";
import type { AcquisitionSource } from "../types/design";

/** Adapt the canonical API source value without inventing an item-level tier. */
export function toAcquisitionSource(
  source: APIAcquisitionSource,
): AcquisitionSource {
  return {
    text: source.text,
    difficulty: toDifficulty(source.difficulty),
    raidDungeon: source.raidDungeon,
  };
}
