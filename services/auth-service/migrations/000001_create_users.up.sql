BEGIN;

-- Значения совпадают с common.v1.Role без префикса ROLE_.
CREATE TYPE user_role AS ENUM ('client', 'officer', 'admin');

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role          user_role   NOT NULL DEFAULT 'client',
    customer_id   UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Email — это логин, уникальность без учёта регистра.
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- customer_id проставляет customer-service после создания профиля,
-- поэтому NULL допустим, но связь строго 1:1.
CREATE UNIQUE INDEX users_customer_id_key ON users (customer_id) WHERE customer_id IS NOT NULL;

COMMIT;
