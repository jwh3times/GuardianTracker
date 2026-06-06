/* ============================================================
   GUARDIAN TRACKER — STORE (role, users, feature flags)
   Context + useGT() hook. Persists to localStorage under gt:*.
   Gating model:
     flag.enabled === false        -> feature hidden everywhere
     enabled && tier < minTier     -> feature LOCKED (upsell)
     enabled && tier >= minTier     -> feature available
   ============================================================ */
const GTContext = React.createContext(null);

function _load(key, fallback) {
  try { const v = localStorage.getItem("gt:" + key); return v == null ? fallback : JSON.parse(v); }
  catch (e) { return fallback; }
}
function _save(key, val) { try { localStorage.setItem("gt:" + key, JSON.stringify(val)); } catch (e) {} }

function GTProvider({ children }) {
  const [role, _setRole]   = React.useState(() => _load("role", "admin"));
  const [prevTier, _setPT] = React.useState(() => _load("prevTier", "alpha"));
  const [users, _setUsers] = React.useState(() => _load("users", window.GT.users));
  const [flags, _setFlags] = React.useState(() => _load("flags", window.GT.flags));

  const setRole  = (v) => { _setRole(v); _save("role", v); };
  const setPrevTier = (v) => { _setPT(v); _save("prevTier", v); };
  const setUsers = (v) => { _setUsers(v); _save("users", v); };
  const setFlags = (v) => { _setFlags(v); _save("flags", v); };

  const tierOf = window.GT.tierOf;
  const tier = tierOf(role);
  const isAdmin = role === "admin";

  const flagState = (id) => {
    const flag = flags.find((f) => f.id === id);
    if (!flag) return { flag: null, enabled: false, accessible: false, locked: false };
    const enabled = !!flag.enabled;
    const accessible = enabled && tier >= tierOf(flag.minTier);
    return { flag, enabled, accessible, locked: enabled && !accessible };
  };
  const accessible = (id) => flagState(id).accessible;

  const setFlag = (id, patch) => setFlags(flags.map((f) => f.id === id ? { ...f, ...patch } : f));
  const setUserRole = (id, r) => {
    if (id === "me") { setRole(r); if (r !== "admin") setPrevTier(r); return; }
    setUsers(users.map((u) => u.id === id ? { ...u, role: r } : u));
  };

  // reach across the whole roster (current user + managed users)
  const roster = [{ id: "me", role }, ...users];
  const reach = (flag) => flag.enabled ? roster.filter((u) => tierOf(u.role) >= tierOf(flag.minTier)).length : 0;
  const totalUsers = roster.length;

  // demo admin simulation
  const simulateAdmin = (on) => {
    if (on) { if (role !== "admin") setPrevTier(role); setRole("admin"); }
    else { setRole(prevTier || "alpha"); }
  };
  const resetDemo = () => { setUsers(window.GT.users); setFlags(window.GT.flags); setRole("admin"); setPrevTier("alpha"); };

  const value = {
    role, setRole, prevTier, setPrevTier, tier, isAdmin,
    users, setUsers, setUserRole, flags, setFlags, setFlag,
    flagState, accessible, reach, totalUsers, roster,
    simulateAdmin, resetDemo,
  };
  return <GTContext.Provider value={value}>{children}</GTContext.Provider>;
}
function useGT() { return React.useContext(GTContext); }

Object.assign(window, { GTContext, GTProvider, useGT });
