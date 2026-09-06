BEGIN;

CREATE TYPE customer_status AS ENUM (
    'new', 'profile_filled', 'on_kyc', 'active', 'rejected', 'blocked'
);

CREATE TABLE customers (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID            NOT NULL,
    status            customer_status NOT NULL DEFAULT 'new',
    status_changed_at TIMESTAMPTZ     NOT NULL DEFAULT now(),

    first_name        TEXT            NOT NULL DEFAULT '',
    last_name         TEXT            NOT NULL DEFAULT '',
    citizenship       TEXT            NOT NULL DEFAULT '',
    phone             TEXT            NOT NULL DEFAULT '',
    birth_date        DATE,

    created_at        TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX customers_user_id_key ON customers (user_id);

CREATE INDEX customers_created_at_id_idx ON customers (created_at DESC, id DESC);

COMMIT;
