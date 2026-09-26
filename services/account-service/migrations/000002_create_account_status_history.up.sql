BEGIN;

-- История статусов счёта, пишется в одной транзакции со сменой статуса.
CREATE TABLE account_status_history (
    id          BIGINT         GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id  UUID           NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    from_status account_status,
    to_status   account_status NOT NULL,
    -- NULL - системный переход или действие владельца счёта.
    actor_id    UUID,
    reason      TEXT           NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT account_status_history_progress_chk CHECK (from_status IS DISTINCT FROM to_status)
);

CREATE INDEX account_status_history_account_id_created_at_idx
    ON account_status_history (account_id, created_at DESC);

COMMIT;
