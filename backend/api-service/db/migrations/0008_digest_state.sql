-- ADR 0023: since-last-visit digest snapshot. One row per user, replaced
-- atomically on each new visit; no history table and no row per collectible.
-- Keyed on user_id, mirroring user_preferences, rather than the
-- (membership_type, membership_id) pair the ADR itself sketches — see the
-- implementation note at the top of docs/adr/0023-since-last-visit-digest-snapshot.md.
CREATE TABLE digest_state (
    user_id           BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_activity_at  TIMESTAMPTZ NOT NULL,
    visit_started_at  TIMESTAMPTZ NOT NULL,
    snapshot          JSONB       NOT NULL,
    snapshot_taken_at TIMESTAMPTZ NOT NULL
);
