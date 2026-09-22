-- Roll targets: the player's own saved perk combination wanted on a specific
-- weapon (CONTEXT.md). Distinct from wishlist_items, which saves a wanted item
-- hash with no perk dimension; the two are separate concepts and one table
-- cannot carry both without making every row half-empty.
--
-- Keyed on user_id like wishlist_items, user_preferences and digest_state,
-- rather than the (membership_type, membership_id) pair ADR 0023 sketched.
--
-- perks is a text array of perk display names, not plug hashes. A perk's base
-- and enhanced variants are two different hashes sharing one name, and the
-- Manifest links them by no field of its own (see manifest.PerkPlug, v1.9.0),
-- so a stored hash would match only the variant it was captured from. The name
-- is the identity that survives enhancement, and it is resolved back to hashes
-- per socket pool at match time.
--
-- UNIQUE (user_id, item_hash) keeps one target per weapon per user: a second
-- wanted roll for the same weapon is an edit, not a new row.
CREATE TABLE roll_targets (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_hash  BIGINT      NOT NULL CHECK (item_hash >= 0 AND item_hash < 4294967296),
    perks      TEXT[]      NOT NULL CHECK (cardinality(perks) BETWEEN 1 AND 10),
    notes      TEXT        NOT NULL DEFAULT '' CHECK (char_length(notes) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, item_hash)
);

CREATE INDEX roll_targets_user_idx ON roll_targets (user_id, created_at DESC);
