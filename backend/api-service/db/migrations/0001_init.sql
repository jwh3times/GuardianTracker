CREATE TABLE users (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    membership_id   TEXT        NOT NULL UNIQUE,
    membership_type SMALLINT    NOT NULL,
    display_name    TEXT        NOT NULL DEFAULT '',
    token_version   INTEGER     NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wishlist_items (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_hash  BIGINT      NOT NULL CHECK (item_hash >= 0 AND item_hash < 4294967296),
    priority   SMALLINT    NOT NULL DEFAULT 1 CHECK (priority BETWEEN 0 AND 3),
    notes      TEXT        NOT NULL DEFAULT '' CHECK (char_length(notes) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, item_hash)
);

CREATE TABLE user_preferences (
    user_id     BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    card_style  TEXT        NOT NULL DEFAULT 'framed' CHECK (card_style IN ('framed','compact')),
    personalize BOOLEAN     NOT NULL DEFAULT true,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE bungie_tokens (
    user_id            BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    access_token_enc   BYTEA       NOT NULL,
    refresh_token_enc  BYTEA       NOT NULL,
    access_expires_at  TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ NOT NULL,
    key_version        SMALLINT    NOT NULL DEFAULT 1,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
