import { describe, it, expect } from "vitest";
import {
  applyDefaultFilter,
  applySearch,
  groupChasingTargets,
  groupMatchesByTarget,
  isTargetPerk,
  type WeaponLookup,
} from "./rollTargetsView";
import type { RollTarget, RollTargetMatch } from "../../types/design";

function rt(overrides: Partial<RollTarget> = {}): RollTarget {
  return {
    id: "1",
    itemHash: "100",
    anyWeapon: false,
    wanted: true,
    perks: ["Arrowhead Brake", "Explosive Payload"],
    notes: "",
    dateAdded: "2026-09-01T00:00:00Z",
    ...overrides,
  };
}

function lookup(overrides: Partial<WeaponLookup> = {}): WeaponLookup {
  return {
    name: (hash) => (hash === "100" ? "Fatebringer" : ""),
    icon: () => undefined,
    type: () => "Hand Cannon",
    acquiredOrWishlisted: () => false,
    ...overrides,
  };
}

describe("groupChasingTargets", () => {
  it("groups weapon-bound targets by hash and any-weapon targets into one bucket", () => {
    const targets = [
      rt({ id: "1", itemHash: "100" }),
      rt({ id: "2", itemHash: "100", perks: ["Corkscrew Rifling"] }),
      rt({ id: "3", itemHash: null, anyWeapon: true, perks: ["Rampage"] }),
    ];
    const groups = groupChasingTargets(targets, lookup());
    expect(groups).toHaveLength(2);
    const fatebringer = groups.find((g) => g.key === "100")!;
    expect(fatebringer.name).toBe("Fatebringer");
    expect(fatebringer.targets).toHaveLength(2);
    const any = groups.find((g) => g.anyWeapon)!;
    expect(any.name).toBe("Any weapon");
    expect(any.targets).toHaveLength(1);
  });

  it("sorts any-weapon last, and named groups alphabetically", () => {
    const targets = [
      rt({ id: "1", itemHash: null, anyWeapon: true }),
      rt({ id: "2", itemHash: "200" }),
      rt({ id: "3", itemHash: "100" }),
    ];
    const groups = groupChasingTargets(
      targets,
      lookup({ name: (h) => (h === "100" ? "Alpha" : "Zulu") }),
    );
    expect(groups.map((g) => g.name)).toEqual(["Alpha", "Zulu", "Any weapon"]);
  });

  it("falls back to a hash-derived name when the lookup has none", () => {
    const groups = groupChasingTargets(
      [rt({ itemHash: "999" })],
      lookup({ name: () => "" }),
    );
    expect(groups[0].name).toBe("Item 999");
  });
});

describe("applyDefaultFilter", () => {
  const groups = groupChasingTargets(
    [
      rt({ id: "1", itemHash: "100" }),
      rt({ id: "2", itemHash: "200" }),
      rt({ id: "3", itemHash: null, anyWeapon: true }),
    ],
    lookup(),
  );

  it("keeps only acquired/wishlisted weapons and any-weapon targets by default", () => {
    const { groups: filtered, hiddenCount } = applyDefaultFilter(groups, {
      showAll: false,
      lookup: lookup({ acquiredOrWishlisted: (h) => h === "100" }),
    });
    expect(filtered.map((g) => g.key).sort()).toEqual(["100", "any"]);
    expect(hiddenCount).toBe(1);
  });

  it("shows everything and reports zero hidden when showAll is set", () => {
    const { groups: filtered, hiddenCount } = applyDefaultFilter(groups, {
      showAll: true,
      lookup: lookup(),
    });
    expect(filtered).toHaveLength(3);
    expect(hiddenCount).toBe(0);
  });
});

describe("applySearch", () => {
  const groups = groupChasingTargets(
    [
      rt({ id: "1", itemHash: "100", perks: ["Arrowhead Brake"] }),
      rt({ id: "2", itemHash: "200", perks: ["Rampage"] }),
    ],
    lookup({ name: (h) => (h === "100" ? "Fatebringer" : "Midnight Coup") }),
  );

  it("matches by weapon name, case-insensitively", () => {
    const out = applySearch(groups, "fate");
    expect(out.map((g) => g.name)).toEqual(["Fatebringer"]);
  });

  it("matches by a target's perk name when the weapon name does not match", () => {
    const out = applySearch(groups, "rampage");
    expect(out.map((g) => g.name)).toEqual(["Midnight Coup"]);
  });

  it("returns every group unchanged for an empty term", () => {
    expect(applySearch(groups, "  ")).toEqual(groups);
  });

  it("drops a group entirely when nothing in it matches", () => {
    expect(applySearch(groups, "nonsense")).toEqual([]);
  });
});

describe("groupMatchesByTarget", () => {
  function match(overrides: Partial<RollTargetMatch> = {}): RollTargetMatch {
    return {
      targetId: "1",
      itemHash: "100",
      instanceId: "inst-1",
      perks: ["Arrowhead Brake", "Explosive Payload"],
      targetPerks: ["Arrowhead Brake"],
      ...overrides,
    };
  }

  it("groups copies under their target and labels an any-weapon target's heading", () => {
    const target = rt({
      id: "1",
      itemHash: null,
      anyWeapon: true,
      perks: ["Arrowhead Brake"],
    });
    const targetsById = new Map([["1", target]]);
    const matches = [match({ instanceId: "a" }), match({ instanceId: "b" })];
    const groups = groupMatchesByTarget(matches, targetsById, lookup(), true);
    expect(groups).toHaveLength(1);
    expect(groups[0].heading).toBe("Any weapon with Arrowhead Brake");
    expect(groups[0].copies).toHaveLength(2);
  });

  it("labels a weapon-bound target's heading with the resolved weapon name", () => {
    const target = rt({ id: "1", itemHash: "100" });
    const targetsById = new Map([["1", target]]);
    const groups = groupMatchesByTarget([match()], targetsById, lookup(), true);
    expect(groups[0].heading).toBe("Fatebringer");
  });

  it("falls back to a reasonable target when the id is missing from the list cache", () => {
    const groups = groupMatchesByTarget(
      [match({ targetId: "9" })],
      new Map(),
      lookup(),
      false,
    );
    expect(groups[0].target.id).toBe("9");
    expect(groups[0].target.wanted).toBe(false);
    expect(groups[0].heading).toBe("Fatebringer");
  });
});

describe("isTargetPerk", () => {
  it("matches case-insensitively", () => {
    expect(isTargetPerk("arrowhead brake", ["Arrowhead Brake"])).toBe(true);
    expect(isTargetPerk("Firefly", ["Arrowhead Brake"])).toBe(false);
  });
});
