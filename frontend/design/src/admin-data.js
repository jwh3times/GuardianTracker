/* ============================================================
   GUARDIAN TRACKER — ROLES, USERS & FEATURE FLAGS
   Early-access tier model + admin console data.
   Tiers: standard < beta < alpha < admin
   ============================================================ */
(function () {
  const ROLES = ["standard", "beta", "alpha", "admin"];
  const ROLE_TIER = { standard: 0, beta: 1, alpha: 2, admin: 3 };
  const MIN_TIERS = ["standard", "beta", "alpha"]; // selectable flag gates

  window.GT.ROLES = ROLES;
  window.GT.ROLE_TIER = ROLE_TIER;
  window.GT.MIN_TIERS = MIN_TIERS;
  window.GT.tierOf = (r) => ROLE_TIER[r] ?? 0;
  window.GT.label.role = { standard: "Standard", beta: "Beta", alpha: "Alpha", admin: "Admin" };
  window.GT.label.tier = { standard: "Standard+", beta: "Beta+", alpha: "Alpha+" };
  window.GT.roleColor = {
    standard: "var(--c-common)", beta: "var(--c-rare)", alpha: "var(--c-exotic)", admin: "var(--c-admin)",
  };

  // current signed-in account (role lives in the store, defaults to admin for the demo)
  window.GT.currentUser = { id: "me", name: "Maya", handle: "Maya#4271", platform: "Steam" };

  // other accounts the admin manages
  window.GT.users = [
    { id: "u1", name: "Vega",     handle: "Vega#7781",     role: "alpha",    platform: "PlayStation", active: "2m ago" },
    { id: "u2", name: "Bryce",    handle: "Ironwolf#3390", role: "beta",     platform: "Xbox",        active: "11m ago" },
    { id: "u3", name: "Saoirse",  handle: "Nightweaver#01",role: "alpha",    platform: "Steam",       active: "1h ago" },
    { id: "u4", name: "Tomas",    handle: "Drifter4Life#9",role: "standard", platform: "Steam",       active: "3h ago" },
    { id: "u5", name: "Priya",    handle: "Lightbearer#22",role: "beta",     platform: "Epic",        active: "5h ago" },
    { id: "u6", name: "Kenji",    handle: "Sundance#1180", role: "standard", platform: "PlayStation", active: "yesterday" },
    { id: "u7", name: "Amara",    handle: "VoidWalker#700",role: "admin",    platform: "Steam",       active: "just now" },
    { id: "u8", name: "Devon",    handle: "Returned#4412", role: "standard", platform: "Xbox",        active: "2d ago" },
    { id: "u9", name: "Lin",      handle: "GlassRunner#56",role: "beta",     platform: "Steam",       active: "4d ago" },
    { id: "u10", name: "Marcus",  handle: "Shaxxfan#1999", role: "standard", platform: "Steam",       active: "1w ago" },
  ];

  // FEATURE FLAGS — each gates a real feature in the app
  window.GT.flags = [
    { id: "weekly-planner",   name: "Weekly Planner",            category: "Core",          minTier: "standard", enabled: true,
      desc: "“This Week” hub — milestones, vendors, Xûr and recommended actions." },
    { id: "wishlist-alerts",  name: "Wishlist availability alerts", category: "Personalization", minTier: "standard", enabled: true,
      desc: "Highlight wishlisted items the moment they’re obtainable from a vendor or activity." },
    { id: "global-search",    name: "Global item search",        category: "Discovery",     minTier: "beta", enabled: true,
      desc: "Search every collectible across the account from the top bar." },
    { id: "cosmetics",        name: "Cosmetics collections",     category: "Collections",   minTier: "beta", enabled: true,
      desc: "Shaders, ghosts, sparrows and ships as visual galleries in Collections." },
    { id: "catalysts-crafting", name: "Catalyst & Crafting tracker", category: "Completion", minTier: "beta", enabled: true,
      desc: "Exotic catalyst progress and weapon crafting patterns / Deepsight." },
    { id: "triumphs-seals",   name: "Triumphs & Seals",          category: "Completion",    minTier: "alpha", enabled: true,
      desc: "Seal galleries with completion %, gilding and triumph breakdowns." },
    { id: "god-roll",         name: "God-roll & owned-roll insights", category: "Power",     minTier: "alpha", enabled: true,
      desc: "Per-weapon owned rolls, god-roll matching and farm-source guidance." },
    { id: "notifications",    name: "Weekly digest notifications", category: "Engagement",  minTier: "alpha", enabled: false,
      desc: "Push “Xûr has an exotic you’re missing” and wishlist-available alerts. In development." },
    { id: "char-loadouts",    name: "Character & loadout overview", category: "Power",       minTier: "alpha", enabled: false,
      desc: "Per-Guardian power, equipped loadout, subclass and reputation. In development." },
  ];
})();
