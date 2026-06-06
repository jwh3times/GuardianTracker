/* ============================================================
   GUARDIAN TRACKER — MOCK DATA (TypeScript port of design/src/data.js)
   Used to populate screens whose backends are not yet implemented
   (This Week, Catalysts & Crafting, Triumphs & Seals) and as a
   graceful fallback for Collections/Dashboard when live data is
   unavailable. Item icons are stylized placeholders (no assets).
   ============================================================ */
import type {
  Catalyst,
  Character,
  CraftPattern,
  Difficulty,
  GTItem,
  Priority,
  Rarity,
  Seal,
  Summary,
  TreeNode,
  Weekly,
  WishlistEntry,
} from "../types/design";

export const RARITIES: Rarity[] = [
  "exotic",
  "legendary",
  "rare",
  "uncommon",
  "common",
];
export const DIFFS: Difficulty[] = ["easy", "moderate", "challenging"];

export const RARITY_LABEL: Record<Rarity, string> = {
  exotic: "Exotic",
  legendary: "Legendary",
  rare: "Rare",
  uncommon: "Uncommon",
  common: "Common",
};
export const DIFF_LABEL: Record<Difficulty, string> = {
  easy: "Easy",
  moderate: "Moderate",
  challenging: "Challenging",
};
export const PRIORITY_LABEL: Record<Priority, string> = {
  urgent: "Urgent",
  high: "High",
  medium: "Medium",
  low: "Low",
};

export const TYPE_GLYPH: Record<string, string> = {
  "Auto Rifle": "AR",
  "Hand Cannon": "HC",
  "Pulse Rifle": "PR",
  "Scout Rifle": "SC",
  "Fusion Rifle": "FR",
  "Sniper Rifle": "SN",
  Shotgun: "SG",
  Sidearm: "SD",
  "Submachine Gun": "SMG",
  Bow: "BOW",
  "Rocket Launcher": "RL",
  "Grenade Launcher": "GL",
  Sword: "SWD",
  Glaive: "GLV",
  "Machine Gun": "MG",
  "Linear Fusion": "LFR",
  "Trace Rifle": "TR",
  Helmet: "HEL",
  Gauntlets: "ARM",
  "Chest Armor": "CHS",
  "Leg Armor": "LEG",
  "Class Item": "CLS",
  "Ghost Shell": "GHO",
  Sparrow: "SPR",
  Ship: "SHP",
  Shader: "SHD",
  Emblem: "EMB",
  Emote: "EMT",
};

export const characters: Character[] = [
  { id: "maya", name: "Maya", cls: "Warlock", race: "Awoken", power: 1810, emblem: 280 },
  { id: "titan", name: "Bryce", cls: "Titan", race: "Human", power: 1808, emblem: 30 },
  { id: "hunt", name: "Vega", cls: "Hunter", race: "Exo", power: 1805, emblem: 200 },
];

// ---------- HERO ITEMS (hand-crafted, referenced across screens) ----------
const hero: Omit<GTItem, "id">[] = [
  {
    name: "Vex Mythoclast", type: "Fusion Rifle", slot: "Power", rarity: "exotic", diff: "challenging",
    source: "Vault of Glass", sourceDetail: "Atheon, Time's Conflux encounter", obtainable: true, collected: false,
    desc: "A weapon born of the Vault, its trigger pulses with the rhythm of the Vex collective consciousness.",
    perks: ["Temporal Unlimiter", "Overflow", "Rangefinder", "Adagio"],
  },
  {
    name: "Fatebringer", type: "Hand Cannon", slot: "Kinetic", rarity: "legendary", diff: "moderate",
    source: "Vault of Glass", sourceDetail: "Confluxes & Templar encounters", obtainable: true, collected: false,
    desc: "Time is a flat circle, and the Templar dies on every loop. So do you, occasionally.",
    perks: ["Explosive Payload", "Firefly", "Thresh", "Opening Shot"],
  },
  {
    name: "Eyasluna", type: "Hand Cannon", slot: "Kinetic", rarity: "legendary", diff: "moderate",
    source: "Trials of Osiris", sourceDetail: "Flawless / weekly rotation", obtainable: true, collected: false,
    desc: "A masterwork of Dead Orbit engineering, refit for the Crucible's cruelest arenas.",
    perks: ["Headstone", "Snapshot Sights", "Encore", "Killing Wind"],
  },
  {
    name: "Hawkmoon", type: "Hand Cannon", slot: "Kinetic", rarity: "exotic", diff: "moderate",
    source: "Xûr (this weekend)", sourceDetail: "Tower · weekend exotic offering", obtainable: true, collected: false,
    desc: "Seven bullets, and the eighth is luck. The Paracausal stacks remember every kill.",
    perks: ["Paracausal Charge", "Hawkmoon Catalyst"],
  },
  {
    name: "Found Verdict", type: "Pulse Rifle", slot: "Energy", rarity: "legendary", diff: "easy",
    source: "Gambit", sourceDetail: "Drifter reputation rank-up", obtainable: true, collected: false,
    desc: "The Vault always closes. The verdict, once found, is rarely overturned.",
    perks: ["Zen Moment", "Kill Clip", "Headseeker"],
  },
  {
    name: "Loaded Question", type: "Fusion Rifle", slot: "Energy", rarity: "legendary", diff: "moderate",
    source: "Nightfall: The Corrupted", sourceDetail: "Weekly Nightfall reward pool", obtainable: true, collected: false,
    desc: "The only stupid question is the one that doesn't overload the chamber.",
    perks: ["Reservoir Burst", "Auto-Loading Holster", "Backup Mag"],
  },
];

const NAMES = [
  "Long Shadow", "Distant Tumulus", "Falling Guillotine", "Cold Front", "The Palindrome", "Gnawing Hunger",
  "Sojourner's Tale", "Royal Chase", "Whispering Slab", "Pizzicato-22", "Forensic Nightmare", "Stars in Shadow",
  "Doom of Chelchis", "Hollow Denial", "Ammit AR2", "Tarnished Mettle", "Brigand's Law", "Tarrabah",
  "Without Remorse", "The Deicide", "Cantata-57", "Riptide", "Aisha's Care", "Unforgiven", "Beloved",
  "Adept Igneous", "Crux Termination", "Hung Jury", "Veiled Threat", "Quicksilver Storm", "Edge Transit",
  "Veiled Statement", "Imperial Decree", "Heritage", "Mindbender's Ambition", "Apex Predator", "Reckless Oracle",
  "Vision of Confluence", "Praedyth's Revenge", "Corrective Measure", "Hezen Vengeance", "Atheon's Epilogue",
];
const TYPES = ["Auto Rifle", "Hand Cannon", "Pulse Rifle", "Scout Rifle", "Sniper Rifle", "Shotgun", "Sidearm", "Submachine Gun", "Bow", "Sword", "Grenade Launcher", "Trace Rifle"];
const SLOTS = ["Kinetic", "Energy", "Power"];
const SOURCES: [string, string, Difficulty][] = [
  ["Vault of Glass", "raid encounters", "challenging"],
  ["Deep Stone Crypt", "raid encounters", "challenging"],
  ["Trials of Osiris", "weekend PvP", "challenging"],
  ["Grandmaster Nightfall", "high-tier strikes", "challenging"],
  ["Banshee-44", "gunsmith weekly rotation", "easy"],
  ["Gambit", "Drifter rank-up", "easy"],
  ["Crucible", "Shaxx reputation", "moderate"],
  ["Vanguard Ops", "Zavala reputation", "easy"],
  ["Nightfall: The Corrupted", "weekly reward pool", "moderate"],
  ["World drops", "general loot pool", "easy"],
  ["Dares of Eternity", "Xûr's treasure hoard", "moderate"],
  ["Seasonal activity", "this season's playlist", "moderate"],
];
const PERK_POOL = ["Outlaw", "Rampage", "Kill Clip", "Snapshot Sights", "Rangefinder", "Firefly", "Frenzy", "Subsistence", "Demolitionist", "Reconstruction", "Vorpal Weapon", "Opening Shot", "Zen Moment", "Headstone", "Incandescent", "Voltshot", "Eddy Current", "Adagio", "Threat Detector", "Auto-Loading Holster"];

let _id = 0;
const nid = () => "i" + ++_id;
function pick<T>(a: T[], seed: number): T {
  return a[Math.abs(Math.floor(Math.sin(seed) * 1e4)) % a.length];
}

function makeItem(name: string, i: number, forceRarity?: Rarity): GTItem {
  const t = pick(TYPES, i * 1.7 + 3);
  const src = pick(SOURCES, i * 2.3 + 1);
  const rarity: Rarity =
    forceRarity || (i % 9 === 0 ? "exotic" : i % 3 === 0 ? "legendary" : i % 5 === 0 ? "rare" : "legendary");
  const collected = (i * 7) % 10 < 4; // ~40% collected
  const obtainable = (i * 3) % 10 < 7;
  const perks = [pick(PERK_POOL, i + 1), pick(PERK_POOL, i + 9), pick(PERK_POOL, i + 17), pick(PERK_POOL, i + 23)];
  return {
    id: nid(), name, type: t, slot: pick(SLOTS, i + 2), rarity,
    diff: src[2], source: src[0], sourceDetail: src[1],
    obtainable, collected,
    desc: `A ${rarity} ${t.toLowerCase()} sought by Guardians chasing the perfect roll. Sourced from ${src[0].toLowerCase()}.`,
    perks: [...new Set(perks)],
  };
}

export const items: GTItem[] = [];
hero.forEach((h) => items.push({ id: nid(), ...h }));
NAMES.forEach((n, i) => items.push(makeItem(n, i)));
["Iron Companionship", "Star-Spangled", "Notorious Gambler", "Solstice Glow", "Twilight Gleam", "New Pacific"].forEach(
  (n, i) =>
    items.push({
      ...makeItem(n, i + 40, "legendary"),
      type: pick(["Shader", "Ghost Shell", "Sparrow", "Ship", "Emblem"], i),
      slot: "Cosmetic",
    })
);

export const tree: TreeNode[] = [
  {
    id: "weapons", label: "Weapons", pct: 81, count: [612, 756], children: [
      { id: "w-raids", label: "Raids", pct: 60, count: [27, 45] },
      { id: "w-dungeons", label: "Dungeons", pct: 71, count: [22, 31] },
      { id: "w-trials", label: "Trials", pct: 55, count: [11, 20] },
      { id: "w-nightfall", label: "Nightfall", pct: 88, count: [14, 16] },
      { id: "w-world", label: "World", pct: 92, count: [120, 130] },
      { id: "w-vendor", label: "Vendor", pct: 78, count: [44, 56] },
    ],
  },
  {
    id: "armor", label: "Armor", pct: 72, count: [410, 569], children: [
      { id: "a-warlock", label: "Warlock", pct: 75, count: [140, 187] },
      { id: "a-titan", label: "Titan", pct: 70, count: [135, 192] },
      { id: "a-hunter", label: "Hunter", pct: 71, count: [135, 190] },
    ],
  },
  {
    id: "exotics", label: "Exotics", pct: 68, count: [98, 144], children: [
      { id: "e-weapons", label: "Exotic Weapons", pct: 72, count: [62, 86] },
      { id: "e-armor", label: "Exotic Armor", pct: 62, count: [36, 58] },
    ],
  },
  {
    id: "cosmetics", label: "Cosmetics", pct: 55, count: [84, 153], children: [
      { id: "c-shaders", label: "Shaders", pct: 61, count: [40, 66] },
      { id: "c-ghosts", label: "Ghost Shells", pct: 48, count: [12, 25] },
      { id: "c-sparrows", label: "Sparrows", pct: 52, count: [16, 31] },
      { id: "c-ships", label: "Ships", pct: 51, count: [16, 31] },
    ],
  },
];

export const weekly: Weekly = {
  resetLabel: "Tue 17:00 UTC",
  resetIn: { d: 2, h: 14, m: 6 },
  xur: {
    present: true, leavesIn: { d: 1, h: 6, m: 12 }, location: "The Tower, Hangar",
    items: [
      { name: "Hawkmoon", type: "Hand Cannon", rarity: "exotic", missing: true, cost: "29 Strange Coin" },
      { name: "Gemini Jester", type: "Leg Armor", rarity: "exotic", missing: false, cost: "23 Strange Coin" },
      { name: "Ophidian Aspect", type: "Gauntlets", rarity: "exotic", missing: false, cost: "23 Strange Coin" },
    ],
  },
  milestones: [
    { id: "raid", label: "Featured Raid", name: "Vault of Glass", reward: "Pinnacle gear", missing: 2, note: "2 weapons you're missing" },
    { id: "nf", label: "Nightfall", name: "The Corrupted", reward: "Loaded Question", missing: 1, note: "Loaded Question — missing" },
    { id: "weak", label: "Weekly Challenges", name: "Vanguard / Crucible / Gambit", reward: "Powerful gear", missing: 0, note: "3 challenges" },
  ],
  vendors: [
    { id: "banshee", name: "Banshee-44", role: "Gunsmith", missing: 1, items: ["Ammit AR2", "Hung Jury", "Cantata-57"] },
    { id: "ada", name: "Ada-1", role: "Shaders & mods", missing: 0, items: ["Iron Companionship", "Solstice Glow"] },
  ],
  recommended: [
    { id: "r1", text: "Run Vault of Glass", detail: "2 missing weapons", badge: "completes-set", done: false, diff: "challenging", time: "~45 min" },
    { id: "r2", text: "Buy Hawkmoon from Xûr", detail: "Missing exotic · leaves in 1d 6h", badge: "missing", done: false, diff: "easy", time: "2 min" },
    { id: "r3", text: "Nightfall on Legend", detail: "Loaded Question reward", badge: "missing", done: false, diff: "moderate", time: "~20 min" },
    { id: "r4", text: "Banshee-44 weekly", detail: "Ammit AR2 — missing", badge: "missing", done: false, diff: "easy", time: "2 min" },
  ],
};

export const wishlist: WishlistEntry[] = [
  { id: "wl1", name: "Fatebringer", type: "Hand Cannon", rarity: "legendary", priority: "urgent", avail: { now: true, where: "Vault of Glass" }, notes: "Want Explosive Payload + Firefly roll", added: "3d ago" },
  { id: "wl2", name: "Vex Mythoclast", type: "Fusion Rifle", rarity: "exotic", priority: "high", avail: { now: true, where: "Vault of Glass" }, notes: "Last raid exotic I need for the seal.", added: "1w ago" },
  { id: "wl3", name: "Hawkmoon", type: "Hand Cannon", rarity: "exotic", priority: "high", avail: { now: true, where: "Xûr · weekend" }, notes: "", added: "5d ago" },
  { id: "wl4", name: "Eyasluna", type: "Hand Cannon", rarity: "legendary", priority: "medium", avail: { now: true, where: "Trials of Osiris" }, notes: "Headstone + Snapshot for PvP.", added: "2w ago" },
  { id: "wl5", name: "Apex Predator", type: "Rocket Launcher", rarity: "legendary", priority: "medium", avail: { now: false, where: "Last Wish raid" }, notes: "Bait and Switch craftable.", added: "2w ago" },
  { id: "wl6", name: "Doom of Chelchis", type: "Scout Rifle", rarity: "legendary", priority: "medium", avail: { now: false, where: "King's Fall raid" }, notes: "", added: "3w ago" },
  { id: "wl7", name: "The Palindrome", type: "Hand Cannon", rarity: "legendary", priority: "low", avail: { now: false, where: "Competitive Crucible" }, notes: "", added: "1mo ago" },
];

export const catalysts: Catalyst[] = [
  { id: "cat1", name: "Sunshot Catalyst", status: "in-progress", obj: { label: "Kills", cur: 320, max: 500 }, source: "Random world drop / activities" },
  { id: "cat2", name: "Whisper of the Worm", status: "missing", obj: null, source: "The Whisper exotic mission" },
  { id: "cat3", name: "Sweet Business", status: "complete", obj: { label: "Kills", cur: 700, max: 700 }, source: "Strikes & Crucible" },
  { id: "cat4", name: "Riskrunner", status: "in-progress", obj: { label: "Arc kills", cur: 410, max: 500 }, source: "Override / Strikes" },
  { id: "cat5", name: "Trinity Ghoul", status: "missing", obj: null, source: "Lake of Shadows GM" },
  { id: "cat6", name: "Ace of Spades", status: "complete", obj: { label: "Kills", cur: 500, max: 500 }, source: "Crucible & Gambit" },
  { id: "cat7", name: "Outbreak Perfected", status: "in-progress", obj: { label: "Zero Hour clears", cur: 2, max: 3 }, source: "Zero Hour mission" },
  { id: "cat8", name: "Thunderlord", status: "missing", obj: null, source: "Heroic public events" },
];

export const crafting: CraftPattern[] = [
  { id: "cr1", name: "The Enigma", type: "Glaive", patterns: { cur: 3, max: 5 }, note: "2 red-borders to go", source: "Seasonal activity / Banshee" },
  { id: "cr2", name: "Calus Mini-Tool", type: "Submachine Gun", patterns: { cur: 5, max: 5 }, note: "Craftable now", source: "Crafted — complete" },
  { id: "cr3", name: "Edge Transit", type: "Grenade Launcher", patterns: { cur: 1, max: 5 }, note: "4 red-borders to go", source: "Onslaught / world drop" },
  { id: "cr4", name: "Brya's Love", type: "Pulse Rifle", patterns: { cur: 4, max: 5 }, note: "1 red-border to go", source: "Vault of Glass red-borders" },
  { id: "cr5", name: "Hollow Denial", type: "Bow", patterns: { cur: 0, max: 5 }, note: "Not yet started", source: "Seasonal vendor" },
  { id: "cr6", name: "Riptide", type: "Fusion Rifle", patterns: { cur: 5, max: 5 }, note: "Craftable now", source: "Crafted — complete" },
];

export const seals: Seal[] = [
  { id: "s1", name: "Conqueror", pct: 92, gilded: 0, left: "3 triumphs left", triumphs: [
    { label: "Complete 5 Grandmaster Nightfalls", done: true, cur: 5, max: 5 },
    { label: "Solo a Legend Nightfall", done: false, cur: 2, max: 5 },
    { label: "Flawless a Master Nightfall", done: false, cur: 0, max: 1 },
  ] },
  { id: "s2", name: "Flawless", pct: 78, gilded: 0, left: "title in reach", triumphs: [
    { label: "Win a Trials match", done: true, cur: 1, max: 1 },
    { label: "Go Flawless", done: false, cur: 5, max: 7 },
    { label: "Flawless on 3 maps", done: false, cur: 1, max: 3 },
  ] },
  { id: "s3", name: "Rivensbane", pct: 100, gilded: 2, left: "Gilded ×2", triumphs: [
    { label: "Clear Last Wish", done: true, cur: 1, max: 1 },
    { label: "All challenges", done: true, cur: 6, max: 6 },
  ] },
  { id: "s4", name: "Chronicler", pct: 40, gilded: 0, left: "lore hunt", triumphs: [
    { label: "Collect lore books", done: false, cur: 12, max: 30 },
    { label: "Hidden records", done: false, cur: 4, max: 18 },
  ] },
  { id: "s5", name: "Dredgen", pct: 64, gilded: 0, left: "Gambit grind", triumphs: [
    { label: "Defeat envoys", done: true, cur: 25, max: 25 },
    { label: "Win matches", done: false, cur: 18, max: 40 },
  ] },
  { id: "s6", name: "Splintered", pct: 86, gilded: 0, left: "2 secrets left", triumphs: [
    { label: "Seasonal story", done: true, cur: 8, max: 8 },
    { label: "Hidden caches", done: false, cur: 6, max: 8 },
  ] },
];

export const summary: Summary = {
  overall: 74,
  categories: [
    { id: "weapons", label: "Weapons", pct: 81, count: [612, 756] },
    { id: "armor", label: "Armor", pct: 72, count: [410, 569] },
    { id: "exotics", label: "Exotics", pct: 68, count: [98, 144] },
    { id: "cosmetics", label: "Cosmetics", pct: 55, count: [84, 153] },
  ],
  totals: { collected: 1204, total: 1628, missing: 424 },
  updatedAgo: "4m",
};
