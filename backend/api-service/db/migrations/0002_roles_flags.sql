-- User roles, feature flags, and an admin audit trail (TODO item 13).
--
-- Role tiers (SMALLINT, same string-mapped pattern as wishlist priority, decision D3):
--   0 = standard, 1 = beta, 2 = alpha, 3 = admin
-- Self opt-in covers 0..2; admin (3) is granted only by an existing admin or the
-- ADMIN_MEMBERSHIP_IDS bootstrap at login.

ALTER TABLE users
    ADD COLUMN role SMALLINT NOT NULL DEFAULT 0 CHECK (role BETWEEN 0 AND 3);

-- Feature flags gate features per the design's store.jsx model:
--   enabled = false              -> hidden everywhere
--   enabled & tier < min_tier    -> locked (upsell)
--   enabled & tier >= min_tier   -> available
-- min_tier is 0..2 (standard/beta/alpha) — features never require admin to use.
CREATE TABLE feature_flags (
    key         TEXT        PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    category    TEXT        NOT NULL DEFAULT '',
    min_tier    SMALLINT    NOT NULL DEFAULT 0 CHECK (min_tier BETWEEN 0 AND 2),
    enabled     BOOLEAN     NOT NULL DEFAULT false,
    sort_order  SMALLINT    NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only audit of every admin-driven role change (and admin grant).
CREATE TABLE role_audit (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id   BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    target_user_id  BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    old_role        SMALLINT    NOT NULL,
    new_role        SMALLINT    NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX role_audit_target_idx ON role_audit (target_user_id, created_at DESC);

-- Seed the flag list from the design (frontend/design/src/admin-data.js).
-- Already-shipped features are seeded enabled & min_tier=0 so no existing user
-- loses anything they already have (TODO 13.1). Not-yet-built features are
-- seeded disabled at their target tier; an admin flips them on as they land.
INSERT INTO feature_flags (key, name, description, category, min_tier, enabled, sort_order) VALUES
    ('weekly-planner',     'Weekly Planner',                  '"This Week" hub — milestones, vendors, Xur and recommended actions.',                          'Core',            0, true,  0),
    ('wishlist-alerts',    'Wishlist availability alerts',    'Highlight wishlisted items the moment they are obtainable from a vendor or activity.',          'Personalization', 0, true,  1),
    ('global-search',      'Global item search',              'Search every collectible across the account from the top bar.',                                'Discovery',       0, true,  2),
    ('cosmetics',          'Cosmetics collections',           'Shaders, ghosts, sparrows and ships as visual galleries in Collections.',                       'Collections',     0, true,  3),
    ('catalysts-crafting', 'Catalyst & Crafting tracker',     'Exotic catalyst progress and weapon crafting patterns / Deepsight.',                            'Completion',      0, true,  4),
    ('triumphs-seals',     'Triumphs & Seals',                'Seal galleries with completion %, gilding and triumph breakdowns.',                             'Completion',      0, true,  5),
    ('god-roll',           'God-roll & owned-roll insights',  'Per-weapon owned rolls, god-roll matching and farm-source guidance. In development.',           'Power',           2, false, 6),
    ('notifications',      'Weekly digest notifications',     'Push "Xur has an exotic you are missing" and wishlist-available alerts. In development.',        'Engagement',      2, false, 7),
    ('char-loadouts',      'Character & loadout overview',    'Per-Guardian power, equipped loadout, subclass and reputation. In development.',                'Power',           2, false, 8),
    ('ui-tweaks',          'Design tweaks panel',             'Admin-only live design-token editor for design iteration. Not an end-user feature.',            'Internal',        2, false, 9);
