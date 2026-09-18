-- ADR 0023: since-last-visit digest snapshot. One row per user, replaced
-- atomically on each new visit; no history table and no row per collectible.
-- Keyed on user_id, mirroring user_preferences, rather than the
-- (membership_type, membership_id) pair the ADR itself sketches — see the
-- implementation note at the top of docs/adr/0023-since-last-visit-digest-snapshot.md.
--
-- current_visit_result holds the complete outcome computed at the current
-- visit's start (status, the acquired item hashes, and the previous visit's
-- last-activity time), so a second request within the same visit — including
-- one that arrives after a process restart — reconstructs an identical
-- digest from this row alone. It is small: normally a status string plus
-- zero to a few dozen item hashes. It is a separate column from `snapshot`
-- on purpose: `snapshot` is the reacquisition baseline gated by profile
-- privacy (ADR 0023), while current_visit_result records what was shown,
-- which is safe to persist even when a visit's read failed or was private.
CREATE TABLE digest_state (
    user_id               BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_activity_at      TIMESTAMPTZ NOT NULL,
    visit_started_at      TIMESTAMPTZ NOT NULL,
    snapshot              JSONB       NOT NULL,
    snapshot_taken_at     TIMESTAMPTZ NOT NULL,
    current_visit_result  JSONB       NOT NULL DEFAULT '{"status":"unavailable","acquired":[]}'
);
