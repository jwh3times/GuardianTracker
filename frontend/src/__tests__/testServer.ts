import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

export const API = "http://localhost:8081";

export const sampleUser = {
  id: "4611686018467260757",
  displayName: "TestGuardian",
  membershipId: "4611686018467260757",
  membershipType: 3,
  platform: "steam",
};

export const sampleCollections = {
  weapons: {
    total: 10,
    collected: 8,
    missing: [
      {
        itemHash: "12345",
        name: "Fatebringer",
        description: "A legendary hand cannon.",
        icon: "/icons/fb.png",
        itemType: "Hand Cannon",
        tierType: 5,
        rarity: "Legendary",
        difficulty: "Challenging",
        sources: ["Vault of Glass raid"],
        isExotic: false,
      },
    ],
  },
  armor: { total: 5, collected: 5, missing: [] },
  exotics: { total: 3, collected: 2, missing: [] },
  cosmetics: { total: 4, collected: 1, missing: [] },
  fetchedAt: new Date().toISOString(),
};

export const sampleWishlist = [
  {
    id: "1",
    itemHash: 99999,
    name: "Gjallarhorn",
    itemType: "Rocket Launcher",
    rarity: "Exotic",
    icon: "/icons/gj.png",
    priority: "HIGH",
    notes: "the classic",
    sources: [],
    availableNow: true,
    availableFrom: "Xûr",
    dateAdded: new Date().toISOString(),
  },
];

export const sampleWeekly = {
  resetLabel: "Tuesday 17:00 UTC",
  resetIn: { d: 2, h: 4, m: 0 },
  dailyResetIn: { h: 7, m: 30 },
  resetAt: "2026-06-16T17:00:00Z",
  fetchedAt: new Date().toISOString(),
  xur: null,
  milestones: [],
  vendors: [],
  recommended: [],
  dailyActions: [],
};

export const sampleCatalysts = {
  items: [
    {
      id: "cat-sunshot",
      name: "Sunshot Catalyst",
      type: "Hand Cannon",
      icon: "/icons/sunshot.png",
      status: "in-progress",
      obj: { label: "Kills", cur: 200, max: 400 },
      source: "Crucible matches",
    },
    {
      id: "cat-riskrunner",
      name: "Riskrunner Catalyst",
      type: "Submachine Gun",
      icon: "/icons/riskrunner.png",
      status: "complete",
      obj: { label: "Kills", cur: 500, max: 500 },
      source: "Strikes",
    },
    {
      id: "cat-cerberus",
      name: "Cerberus+1 Catalyst",
      type: "Auto Rifle",
      icon: "",
      status: "missing",
      obj: null,
      source: "World drops",
    },
  ],
  fetchedAt: new Date().toISOString(),
};

export const sampleCrafting = {
  items: [
    {
      id: "craft-enigma",
      name: "The Enigma",
      type: "Glaive",
      patterns: { cur: 5, max: 5 },
      note: "1 to go",
      source: "The Witch Queen",
    },
    {
      id: "craft-osteo",
      name: "Osteo Striga",
      type: "Submachine Gun",
      patterns: { cur: 2, max: 5 },
      note: "3 to go",
      source: "Throne World",
    },
    {
      id: "craft-fresh",
      name: "Unworked Pattern",
      type: "Bow",
      patterns: { cur: 0, max: 5 },
      note: "5 to go",
      source: "Crafting",
    },
  ],
  fetchedAt: new Date().toISOString(),
};

export const sampleSeals = {
  items: [
    {
      id: "seal-conqueror",
      name: "Conqueror",
      pct: 80,
      gilded: 0,
      left: "2 triumphs left",
      triumphs: [
        { label: "Complete a GM", done: true, cur: 1, max: 1 },
        { label: "Flawless card", done: false, cur: 0, max: 1 },
      ],
    },
    {
      id: "seal-flawless",
      name: "Flawless",
      pct: 100,
      gilded: 2,
      left: "Complete",
      triumphs: [{ label: "Go Flawless", done: true, cur: 1, max: 1 }],
    },
  ],
  fetchedAt: new Date().toISOString(),
};

// Resolved feature flags for the signed-in (admin) test user. Shipped features
// are accessible; an alpha-gated, enabled feature is locked for lower tiers.
export const sampleFlags = {
  role: "admin" as const,
  flags: [
    {
      key: "catalysts-crafting",
      name: "Catalyst & Crafting tracker",
      desc: "Exotic catalyst progress and weapon crafting patterns.",
      category: "Completion",
      minTier: "standard",
      enabled: true,
      accessible: true,
      locked: false,
    },
    {
      key: "triumphs-seals",
      name: "Triumphs & Seals",
      desc: "Seal galleries with completion %, gilding and triumph breakdowns.",
      category: "Completion",
      minTier: "standard",
      enabled: true,
      accessible: true,
      locked: false,
    },
  ],
};

export const sampleAdminUsers = [
  {
    id: "1",
    displayName: "TestGuardian",
    membershipId: "4611686018467260757",
    platform: "steam",
    role: "admin" as const,
    lastActive: new Date().toISOString(),
  },
  {
    id: "2",
    displayName: "Vega",
    membershipId: "200",
    platform: "psn",
    role: "beta" as const,
    lastActive: new Date().toISOString(),
  },
];

export const sampleAdminFlags = [
  {
    key: "catalysts-crafting",
    name: "Catalyst & Crafting tracker",
    description: "Exotic catalyst progress and weapon crafting patterns.",
    category: "Completion",
    minTier: "standard",
    enabled: true,
    updatedAt: new Date().toISOString(),
  },
  {
    key: "god-roll",
    name: "God-roll insights",
    description: "Per-weapon owned rolls and god-roll matching.",
    category: "Power",
    minTier: "alpha",
    enabled: false,
    updatedAt: new Date().toISOString(),
  },
];

export const defaultHandlers = [
  http.get(`${API}/api/auth/profile`, () => HttpResponse.json({ user: sampleUser })),
  http.get(`${API}/api/flags`, () => HttpResponse.json(sampleFlags)),
  http.put(`${API}/api/account/role`, () => HttpResponse.json({ role: "beta" })),
  http.get(`${API}/api/admin/users`, () => HttpResponse.json(sampleAdminUsers)),
  http.put(`${API}/api/admin/users/:id/role`, () =>
    HttpResponse.json({ id: "2", role: "alpha", previousRole: "beta" })
  ),
  http.get(`${API}/api/admin/flags`, () => HttpResponse.json(sampleAdminFlags)),
  http.put(`${API}/api/admin/flags/:key`, ({ params }) =>
    HttpResponse.json({ ...sampleAdminFlags[0], key: params.key as string, enabled: true })
  ),
  http.get(`${API}/api/preferences`, () =>
    HttpResponse.json({ cardStyle: "framed", personalize: true })
  ),
  http.put(`${API}/api/preferences`, () =>
    HttpResponse.json({ cardStyle: "framed", personalize: true })
  ),
  http.get(`${API}/api/characters/:type/:id`, () => HttpResponse.json([])),
  http.get(`${API}/api/collections/:type/:id`, () => HttpResponse.json(sampleCollections)),
  http.post(`${API}/api/collections/:type/:id/refresh`, () =>
    HttpResponse.json({ success: true, message: "ok" })
  ),
  http.get(`${API}/api/wishlist`, () => HttpResponse.json(sampleWishlist)),
  http.get(`${API}/api/weekly/recommendations`, () => HttpResponse.json(sampleWeekly)),
  http.get(`${API}/api/catalysts/:type/:id`, () => HttpResponse.json(sampleCatalysts)),
  http.get(`${API}/api/crafting/:type/:id`, () => HttpResponse.json(sampleCrafting)),
  http.get(`${API}/api/seals/:type/:id`, () => HttpResponse.json(sampleSeals)),
  http.get(`${API}/api/items/search`, () => HttpResponse.json([])),
];

export const server = setupServer(...defaultHandlers);
