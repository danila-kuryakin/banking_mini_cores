BEGIN;

CREATE TABLE idempotency_keys (
    endpoint     TEXT        NOT NULL,   -- "customer.v1.CustomerService/UpdateProfile"
    key          TEXT        NOT NULL,   -- idempotency_key из запроса
    request_hash BYTEA       NOT NULL,   -- тот же ключ с другим телом — конфликт, а не повтор
    response     JSONB,                  -- сохранённый ответ для повторного вызова
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (endpoint, key)
);

CREATE INDEX idempotency_keys_created_at_idx ON idempotency_keys (created_at); -- TTL-уборка

COMMIT;
