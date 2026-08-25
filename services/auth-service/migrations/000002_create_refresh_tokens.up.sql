BEGIN;

CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash BYTEA       NOT NULL,   -- SHA-256 от refresh-токена; сам токен не храним
    expires_at TIMESTAMPTZ NOT NULL,   -- TokenPair.refresh_expires_at
    revoked_at TIMESTAMPTZ,            -- Logout и ротация при Refresh
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX refresh_tokens_token_hash_key ON refresh_tokens (token_hash);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);       -- выход со всех устройств
CREATE INDEX refresh_tokens_expires_at_idx ON refresh_tokens (expires_at); -- чистка протухших

COMMIT;
