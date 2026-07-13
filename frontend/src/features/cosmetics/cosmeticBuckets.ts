/**
 * Cosmetic itemType strings emitted by the backend's ItemTypeName
 * (backend/api-service/services/bungie/types.go).
 *
 * VERIFIED in Task 1 against backend source:
 *   - types.go: ItemTypeName cases 14→"Emblem", 21→"Ship", 22→"Sparrow",
 *     23→"Emote", 24→"Ghost" (NOT "Ghost Shell").
 *   - repository.go: cosmeticItemTypes confirms the same five integer types.
 *   - Shaders: classified as cosmetics by the backend (itemType "Shader")
 *     after Task 1b; TYPE_GLYPH already has Shader: "SHD".
 *   - Local manifest verification (2026-07-12): collectible ornaments are
 *     itemType 19 + itemSubType 21; collectible finishers are itemType 29.
 *     The backend normalizes these to "Ornament" and "Finisher".
 *
 * Task 1b made toDestinyItem emit the real ItemTypeName strings (previously
 * "Unknown" for cosmetics), so these values match the items-map itemType.
 *
 * Order here = gallery tab order. Every entry MUST have a TYPE_GLYPH entry in
 * src/lib/constants.ts for the icon fallback.
 */
export const COSMETIC_TYPES = [
  "Emblem",
  "Shader",
  "Ghost",
  "Ship",
  "Sparrow",
  "Emote",
  "Ornament",
  "Finisher",
] as const;

export type CosmeticType = (typeof COSMETIC_TYPES)[number];

/** O(1) membership test for classifying a GTItem by its `type` string. */
export const COSMETIC_TYPE_SET: ReadonlySet<string> = new Set(COSMETIC_TYPES);
