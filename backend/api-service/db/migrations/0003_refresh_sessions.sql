-- Per-device refresh-token sessions (closes SECURITY.md known-limitation #3 with
-- full multi-device support).
--
-- Each login creates one session (one device). Its access and refresh tokens carry
-- the session id (the JWT `sid` claim); the refresh token also carries a `jti` that
-- rotates on every refresh. On refresh the service compare-and-swaps current_jti for
-- the session: a presented jti that no longer matches means the token was already
-- rotated or is being replayed (theft) — the whole session is revoked. Sessions are
-- independent, so refreshing (or logging out) on one device never disturbs another.
--
-- token_version (users.token_version) remains the account-wide "sign out everywhere"
-- kill switch: bumping it invalidates every device's access token at once.

CREATE TABLE refresh_sessions (
    id            TEXT        PRIMARY KEY,                       -- session id; the JWT `sid` claim
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    membership_id TEXT        NOT NULL,                          -- denormalized for lookup by JWT claim
    current_jti   TEXT        NOT NULL,                          -- jti of the currently valid refresh token
    user_agent    TEXT        NOT NULL DEFAULT '',               -- best-effort device label
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at    TIMESTAMPTZ NOT NULL                           -- refresh-token expiry; rejected/pruned past this
);

-- Look up all of a user's sessions (sign-out-everywhere; future "active sessions" UI).
CREATE INDEX refresh_sessions_membership_idx ON refresh_sessions (membership_id);

-- Support the periodic pruner that deletes expired sessions.
CREATE INDEX refresh_sessions_expires_idx ON refresh_sessions (expires_at);
