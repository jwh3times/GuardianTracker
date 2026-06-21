-- Rename the single-device logout event type into the "logout." family so it shares
-- the prefix with "logout.all" (matching the login./refresh./role. convention). This
-- lets the admin "Logouts" filter (prefix match on "logout.") catch both events.
UPDATE audit_log SET event_type = 'logout.session' WHERE event_type = 'logout';
