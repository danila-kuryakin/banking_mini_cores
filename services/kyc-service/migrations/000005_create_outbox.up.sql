BEGIN;

-- События заявок пишутся здесь в одной транзакции со сменой статуса, в Kafka их переносит воркер.
CREATE TABLE outbox (
    id           UUID        PRIMARY KEY,
    message_key  TEXT        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;

COMMIT;
