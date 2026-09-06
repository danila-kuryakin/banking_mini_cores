BEGIN;

-- Значения совпадают с common.v1.Role без префикса ROLE_.
CREATE TYPE user_role AS ENUM ('client', 'officer', 'admin');

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role          user_role   NOT NULL DEFAULT 'client',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Email — это логин, уникальность без учёта регистра.
CREATE UNIQUE INDEX users_email_key ON users (lower(email));


COMMIT;
