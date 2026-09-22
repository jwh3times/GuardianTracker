-- Roll targets gain a stance and an any-weapon scope, so a DIM-format file can
-- be imported without discarding two of its three legal roll forms.
--
-- `wanted = false` is DIM's undesirable ("trash") roll: a combination the player
-- wants to be told about so they can dismantle it. A NULL item_hash is DIM's
-- any-item wildcard, which names perks without naming a weapon.
--
-- The one-target-per-weapon rule from 0009 is replaced. It could not survive
-- either addition: a player wants both a wanted and an unwanted roll on the same
-- weapon, and a wish list normally lists several acceptable rolls per gun. The
-- new key is the roll itself, which still stops a re-imported file from
-- duplicating everything, and NULLS NOT DISTINCT makes that hold for wildcards
-- too — without it PostgreSQL treats every NULL item_hash as unique and a
-- re-import would stack wildcard rows without limit.
--
-- perks is stored sorted, which is what makes it usable as a key. Order carries
-- no meaning: a target's perks are AND-ed, so ["Outlaw","Firefly"] and
-- ["Firefly","Outlaw"] are the same wanted roll and must not both be storable.
ALTER TABLE roll_targets
    ADD COLUMN wanted BOOLEAN NOT NULL DEFAULT true,
    ALTER COLUMN item_hash DROP NOT NULL;

ALTER TABLE roll_targets
    DROP CONSTRAINT roll_targets_user_id_item_hash_key;

ALTER TABLE roll_targets
    ADD CONSTRAINT roll_targets_roll_key
    UNIQUE NULLS NOT DISTINCT (user_id, item_hash, wanted, perks);
