BEGIN;

CREATE TYPE balance_kind AS ENUM ('customer', 'cash');
CREATE TYPE transaction_type AS ENUM ('transfer', 'deposit');
CREATE TYPE transaction_status AS ENUM ('completed', 'declined');
CREATE TYPE entry_direction AS ENUM ('debit', 'credit');

CREATE TABLE balances (
    account_id UUID         PRIMARY KEY,
    kind       balance_kind NOT NULL DEFAULT 'customer',
    currency   CHAR(3)      NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    amount     BIGINT       NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT balances_non_negative_chk CHECK (kind = 'cash' OR amount >= 0)
);

CREATE UNIQUE INDEX balances_cash_currency_key ON balances (currency) WHERE kind = 'cash';

CREATE TABLE transactions (
    id              UUID               PRIMARY KEY,
    type            transaction_type   NOT NULL,
    status          transaction_status NOT NULL,
    from_account_id UUID               NOT NULL REFERENCES balances (account_id),
    to_account_id   UUID               NOT NULL REFERENCES balances (account_id),
    amount          BIGINT             NOT NULL CHECK (amount > 0),
    currency        CHAR(3)            NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    initiator_id    UUID,
    decline_reason  TEXT               NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT transactions_distinct_accounts_chk CHECK (from_account_id <> to_account_id),
    CONSTRAINT transactions_decline_reason_chk CHECK ((status = 'declined') = (decline_reason <> ''))
);

CREATE INDEX transactions_from_account_id_id_idx ON transactions (from_account_id, id DESC);
CREATE INDEX transactions_to_account_id_id_idx ON transactions (to_account_id, id DESC);

CREATE TABLE entries (
    id             UUID            PRIMARY KEY,
    transaction_id UUID            NOT NULL REFERENCES transactions (id),
    account_id     UUID            NOT NULL REFERENCES balances (account_id),
    direction      entry_direction NOT NULL,
    amount         BIGINT          NOT NULL CHECK (amount > 0),
    currency       CHAR(3)         NOT NULL,
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX entries_transaction_id_idx ON entries (transaction_id);
CREATE INDEX entries_account_id_idx ON entries (account_id);

-- Инвариант двойной записи проверяется на коммите, когда в транзакции уже есть все проводки.
CREATE FUNCTION check_transaction_balanced() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    tx_id            UUID;
    tx_status        transaction_status;
    tx_currency      CHAR(3);
    entry_count      INT;
    debit_sum        NUMERIC;
    credit_sum       NUMERIC;
    foreign_currency INT;
BEGIN
    IF TG_TABLE_NAME = 'transactions' THEN
        tx_id := NEW.id;
    ELSE
        tx_id := NEW.transaction_id;
    END IF;

    SELECT status, currency INTO tx_status, tx_currency FROM transactions WHERE id = tx_id;

    SELECT count(*),
           coalesce(sum(amount) FILTER (WHERE direction = 'debit'), 0),
           coalesce(sum(amount) FILTER (WHERE direction = 'credit'), 0),
           count(*) FILTER (WHERE currency <> tx_currency)
    INTO entry_count, debit_sum, credit_sum, foreign_currency
    FROM entries
    WHERE transaction_id = tx_id;

    IF tx_status = 'completed' AND entry_count < 2 THEN
        RAISE EXCEPTION 'transaction % must have at least two entries', tx_id;
    END IF;

    IF tx_status = 'declined' AND entry_count > 0 THEN
        RAISE EXCEPTION 'declined transaction % must not have entries', tx_id;
    END IF;

    IF debit_sum <> credit_sum THEN
        RAISE EXCEPTION 'transaction % is unbalanced: debit %, credit %', tx_id, debit_sum, credit_sum;
    END IF;

    IF foreign_currency > 0 THEN
        RAISE EXCEPTION 'transaction % has entries in another currency', tx_id;
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER transactions_balanced_trg
    AFTER INSERT ON transactions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_transaction_balanced();

CREATE CONSTRAINT TRIGGER entries_balanced_trg
    AFTER INSERT ON entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_transaction_balanced();

-- Журнал проводок только дописывается: исправление - это новая транзакция, а не правка старой.
CREATE FUNCTION forbid_ledger_rewrite() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% is append-only', TG_TABLE_NAME;
END;
$$;

CREATE TRIGGER transactions_append_only_trg
    BEFORE UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE FUNCTION forbid_ledger_rewrite();

CREATE TRIGGER entries_append_only_trg
    BEFORE UPDATE OR DELETE ON entries
    FOR EACH ROW EXECUTE FUNCTION forbid_ledger_rewrite();

COMMIT;
