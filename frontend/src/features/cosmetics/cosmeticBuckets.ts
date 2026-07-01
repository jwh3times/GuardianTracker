/**
 * Cosmetic itemType strings emitted by the backend's ItemTypeName
 * (backend/api-service/services/bungie/types.go).
 *
 * VERIFIED in Task 1 against backend source:
 *   - types.go: ItemTypeName cases 14→"Emblem", 21→"Ship", 22→"Sparrow",
 *     23→"Emote", 24→"Ghost" (NOT "Ghost Shell").
 *   - repository.go: cosmeticItemTypes confirms the same five integer types.
 *   - Shaders are NOT cosmetics in this system (absent from cosmeticItemTypes;
 *     ItemTypeName has no shader case).
 *   - Ornaments are NOT a distinct flat itemType; leave them out of v1.
 *
 * KNOWN CONCERN (Task 1): toDestinyItem in service.go currently emits
 * "Unknown" for cosmetics instead of using ItemTypeName. The items map will
 * have itemType:"Unknown" until that function is extended. Later tasks that
 * classify by itemType must account for this (or a backend fix must land first).
 *
 * Order here = gallery tab order. Every entry MUST have a TYPE_GLYPH entry in
 * src/lib/constants.ts for the icon fallback.
 */
export const COSMETIC_TYPES = [
  "Emblem",
  "Ship",
  "Sparrow",
  "Ghost",
  "Emote",
] as const;

export type CosmeticType = (typeof COSMETIC_TYPES)[number];

/** O(1) membership test for classifying a GTItem by its `type` string. */
export const COSMETIC_TYPE_SET: ReadonlySet<string> = new Set(COSMETIC_TYPES);
