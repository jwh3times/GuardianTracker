-- Record server-authoritative completion of the first-run onboarding flow.
-- NULL means the user has not completed onboarding. Completion is intentionally
-- irreversible through the preferences API so an ordinary update cannot reset it.
ALTER TABLE user_preferences
    ADD COLUMN onboarded_at TIMESTAMPTZ NULL;
