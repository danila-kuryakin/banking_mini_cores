BEGIN;

CREATE TYPE customer_status AS ENUM (
    'new', 'profile_filled', 'on_kyc', 'active', 'rejected', 'blocked'
);

CREATE TABLE customers (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID            NOT NULL,
    status            customer_status NOT NULL DEFAULT 'new',
    status_changed_at TIMESTAMPTZ     NOT NULL DEFAULT now(),

    first_name        TEXT,
    last_name         TEXT,
    patronymic        TEXT,
    birth_date        DATE,
    citizenship       TEXT,
    phone             TEXT,

    address_country     TEXT,
    address_city        TEXT,
    address_street      TEXT,
    address_building    TEXT,
    address_apartment   TEXT,

    created_at        TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT now()
);

-- Один клиент на одного пользователя.
CREATE UNIQUE INDEX customers_user_id_key ON customers (user_id);

-- ListCustomers: фильтр по статусу + стабильный порядок для пагинации.
CREATE INDEX customers_status_created_at_idx ON customers (status, created_at DESC, id DESC);

COMMIT;
