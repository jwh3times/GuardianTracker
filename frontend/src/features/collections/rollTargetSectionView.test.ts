import { describe, it, expect } from "vitest";
import {
  groupMatchesByCopy,
  matchesForItem,
  weaponBoundTargetsFor,
} from "./rollTargetSectionView";
import type { RollTarget, RollTargetMatch } from "../../types/design";

function rt(overrides: Partial<RollTarget> = {}): RollTarget {
  return {
    id: "1",
    itemHash: "500",
    anyWeapon: false,
    wanted: true,
    perks: ["Arrowhead Brake", "Explosive Payload"],
    notes: "",
    dateAdded: "2026-09-01T00:00:00Z",
    ...overrides,
  };
}

function match(overrides: Partial<RollTargetMatch> = {}): RollTargetMatch {
  return {
    targetId: "1",
    itemHash: "500",
    instanceId: "inst-1",
    perks: ["Arrowhead Brake", "Explosive Payload"],
    targetPerks: ["Arrowhead Brake"],
    ...overrides,
  };
}

describe("weaponBoundTargetsFor", () => {
  it("keeps only this weapon's targets", () => {
    const targets = [
      rt({ id: "1", itemHash: "500" }),
      rt({ id: "2", itemHash: "600" }),
    ];
    expect(weaponBoundTargetsFor(targets, "500").map((t) => t.id)).toEqual([
      "1",
    ]);
  });

  it("excludes an any-weapon target (null itemHash) even for the right weapon", () => {
    const targets = [
      rt({ id: "1", itemHash: "500" }),
      rt({ id: "2", itemHash: null, anyWeapon: true }),
    ];
    expect(weaponBoundTargetsFor(targets, "500").map((t) => t.id)).toEqual([
      "1",
    ]);
  });

  it("returns nothing for a weapon with no bound targets", () => {
    expect(weaponBoundTargetsFor([rt({ itemHash: "600" })], "500")).toEqual([]);
  });
});

describe("matchesForItem", () => {
  it("keeps only matches whose owned copy is this item, including an any-weapon target's match", () => {
    const matches = [
      match({ targetId: "1", itemHash: "500" }),
      match({ targetId: "2", itemHash: "600" }),
      // An any-weapon target's match still carries the owned copy's own
      // itemHash — this item — so it is included by the same equality check.
      match({ targetId: "aw", itemHash: "500", instanceId: "inst-2" }),
    ];
    expect(matchesForItem(matches, "500").map((m) => m.targetId)).toEqual([
      "1",
      "aw",
    ]);
  });
});

describe("groupMatchesByCopy", () => {
  it("labels distinct owned copies Copy 1, Copy 2 in order of first appearance", () => {
    const matches = [
      match({ instanceId: "inst-a", targetId: "1" }),
      match({ instanceId: "inst-b", targetId: "1" }),
    ];
    const groups = groupMatchesByCopy(matches, new Map([["1", rt()]]));
    expect(groups.map((g) => g.label)).toEqual(["Copy 1", "Copy 2"]);
  });

  it("collects every target one copy satisfies under a single group, badge per target", () => {
    const targetsById = new Map([
      ["1", rt({ id: "1", perks: ["Arrowhead Brake"] })],
      [
        "2",
        rt({
          id: "2",
          itemHash: null,
          anyWeapon: true,
          perks: ["Explosive Payload"],
        }),
      ],
    ]);
    const matches = [
      match({
        instanceId: "inst-a",
        targetId: "1",
        targetPerks: ["Arrowhead Brake"],
      }),
      match({
        instanceId: "inst-a",
        targetId: "2",
        targetPerks: ["Explosive Payload"],
      }),
    ];
    const groups = groupMatchesByCopy(matches, targetsById);
    expect(groups).toHaveLength(1);
    expect(groups[0].satisfiedTargets).toEqual([
      { targetId: "1", label: "Arrowhead Brake" },
      { targetId: "2", label: "Any weapon: Explosive Payload" },
    ]);
    // Union of both targets' own perks, for chip highlighting.
    expect(groups[0].targetPerks.sort()).toEqual(
      ["Arrowhead Brake", "Explosive Payload"].sort(),
    );
  });

  it("falls back to the match's own targetPerks when the target id is missing from the list cache", () => {
    const groups = groupMatchesByCopy(
      [match({ targetId: "missing", targetPerks: ["Rampage"] })],
      new Map(),
    );
    expect(groups[0].satisfiedTargets).toEqual([
      { targetId: "missing", label: "Rampage" },
    ]);
  });
});
