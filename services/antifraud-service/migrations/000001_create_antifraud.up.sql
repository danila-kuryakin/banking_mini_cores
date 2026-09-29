BEGIN;

CREATE TYPE af_verdict AS ENUM ('allow', 'deny');

CREATE TABLE af_rules (
    code        TEXT        PRIMARY KEY,
    description TEXT        NOT NULL,
    enabled     BOOLEAN     NOT NULL DEFAULT true,
    params      JSONB       NOT NULL DEFAULT '{}',
    updated_by  UUID,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Пороги в минорных единицах: 100 000 RUB за раз и 300 000 RUB в день, 10 000 и 30 000 USD/EUR.
INSERT INTO af_rules (code, description, params) VALUES
    ('single_amount_limit', 'one transfer must not exceed the currency limit',
     '{"limits": {"RUB": 10000000, "USD": 1000000, "EUR": 1000000}}'),
    ('velocity', 'no more than max_count allowed transfers from an account within window_seconds',
     '{"max_count": 5, "window_seconds": 60}'),
    ('daily_amount_limit', 'completed transfers from an account per UTC day must not exceed the currency limit',
     '{"limits": {"RUB": 30000000, "USD": 3000000, "EUR": 3000000}}'),
    ('same_account', 'transfer to the same account is denied', '{}');

CREATE TABLE af_checks (
    id              UUID        PRIMARY KEY,
    initiator_id    UUID        NOT NULL,
    from_account_id UUID        NOT NULL,
    to_account_id   UUID        NOT NULL,
    amount          BIGINT      NOT NULL CHECK (amount > 0),
    currency        CHAR(3)     NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    verdict         af_verdict  NOT NULL,
    triggered_rules TEXT[]      NOT NULL DEFAULT '{}',
    reasons         TEXT[]      NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT af_checks_verdict_rules_chk CHECK ((verdict = 'deny') = (cardinality(triggered_rules) > 0))
);

-- Velocity считает разрешённые проверки счёта за окно.
CREATE INDEX af_checks_velocity_idx ON af_checks (from_account_id, created_at) WHERE verdict = 'allow';

-- Сумма проведённых переводов счёта за день UTC, копится по событиям ledger.
CREATE TABLE af_daily_totals (
    account_id UUID        NOT NULL,
    day        DATE        NOT NULL,
    currency   CHAR(3)     NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    amount     BIGINT      NOT NULL CHECK (amount >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (account_id, day, currency)
);

-- Дедупликация событий Kafka: доставка at-least-once.
CREATE TABLE processed_events (
    event_id     UUID        PRIMARY KEY,
    event_type   TEXT        NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
