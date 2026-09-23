-- Enable the god-roll flag now that roll targets are reachable over REST
-- (issue #367): RequireFlag 404s a disabled flag for everyone, including
-- admins, and 0002 seeded god-roll disabled. min_tier stays at alpha (2) —
-- unlike the flags 0002 already shipped enabled at min_tier=0, god-roll has
-- never been reachable by any current user, so there is no "no one loses
-- access" reason to seed it any wider than its design tier. The owner chose
-- this migration over a manual admin toggle so every deployment ships it
-- enabled from this release.
UPDATE feature_flags SET enabled = true WHERE key = 'god-roll';
