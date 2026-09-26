BEGIN;

CREATE TYPE account_status AS ENUM ('active', 'blocked', 'closed');

-- Серийник номера счёта, уникальность номера держится на нём.
CREATE SEQUENCE account_number_seq MINVALUE 1 MAXVALUE 99999999999 NO CYCLE;

CREATE TABLE accounts (
    id                UUID           PRIMARY KEY,
    user_id           UUID           NOT NULL,
    number            CHAR(20)       NOT NULL UNIQUE,
    currency          CHAR(3)        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    status            account_status NOT NULL DEFAULT 'active',
    status_changed_at TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT now()
);

-- Список счетов клиента и подсчёт лимита по валюте.
CREATE INDEX accounts_user_id_id_idx ON accounts (user_id, id);

COMMIT;
