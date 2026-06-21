-- Unified append-only audit trail (closes security-limitations #4).
-- Supersedes the role-change-only role_audit table: its rows are copied in and it
-- is dropped. details is JSONB (deliberately overriding decision D4's no-JSONB rule,
-- which was preferences-specific; audit payloads are heterogeneous per event type).
CREATE TABLE audit_log (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type          TEXT        NOT NULL,
    outcome             TEXT        NOT NULL DEFAULT 'success' CHECK (outcome IN ('success','failure')),
    actor_user_id       BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    actor_membership_id TEXT        NOT NULL DEFAULT '',
    target_user_id      BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    session_id          TEXT,
    ip                  INET,
    user_agent          TEXT        NOT NULL DEFAULT '',
    details             JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_created_idx ON audit_log (created_at DESC, id DESC);
CREATE INDEX audit_log_event_idx   ON audit_log (event_type, created_at DESC);
CREATE INDEX audit_log_actor_idx   ON audit_log (actor_user_id, created_at DESC);
CREATE INDEX audit_log_target_idx  ON audit_log (target_user_id, created_at DESC);

-- Preserve existing role-change history, then retire the specialized table.
INSERT INTO audit_log (event_type, outcome, actor_user_id, target_user_id, details, created_at)
SELECT 'role.change.admin', 'success', actor_user_id, target_user_id,
       jsonb_build_object('oldRole', old_role, 'newRole', new_role), created_at
FROM role_audit;

DROP TABLE role_audit;
