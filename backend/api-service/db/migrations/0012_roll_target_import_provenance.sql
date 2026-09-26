-- Existing targets stay ungrouped. Provenance does not participate in roll identity.
ALTER TABLE roll_targets
    ADD COLUMN import_id UUID,
    ADD COLUMN import_title TEXT,
    ADD CONSTRAINT roll_targets_import_title_length CHECK (char_length(import_title) <= 500),
    ADD CONSTRAINT roll_targets_import_title_requires_id CHECK (import_id IS NOT NULL OR import_title IS NULL);

CREATE INDEX idx_roll_targets_user_import ON roll_targets (user_id, import_id)
    WHERE import_id IS NOT NULL;
