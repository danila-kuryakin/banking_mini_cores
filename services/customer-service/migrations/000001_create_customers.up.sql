BEGIN;

-- Значения совпадают с customer.v1.CustomerStatus без префикса CUSTOMER_STATUS_.
CREATE TYPE customer_status AS ENUM (
    'new', 'profile_filled', 'on_kyc', 'active', 'rejected', 'blocked'
);

CREATE TABLE customers (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Ссылка на users.id из auth-service. Внешнего ключа нет намеренно:
    -- это чужая БД и чужой сервис, связь держится на уровне приложения.
    user_id           UUID            NOT NULL,
    status            customer_status NOT NULL DEFAULT 'new',
    status_changed_at TIMESTAMPTZ     NOT NULL DEFAULT now(),

    -- Profile: до первого UpdateProfile всё NULL, статус при этом 'new'.
    first_name        TEXT,
    last_name         TEXT,
    birth_date        DATE,           -- google.type.Date
    citizenship       TEXT,
    phone             TEXT,

    -- Address внутри Profile, разложен плоско.
    address_country     TEXT,
    address_city        TEXT,
    address_street      TEXT,
    address_building    TEXT,
    address_postal_code TEXT,

    created_at        TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT now()
);

-- Один клиент на одного пользователя.
CREATE UNIQUE INDEX customers_user_id_key ON customers (user_id);

-- ListCustomers: фильтр по статусу + стабильный порядок для пагинации.
CREATE INDEX customers_status_created_at_idx ON customers (status, created_at DESC, id DESC);

COMMIT;
