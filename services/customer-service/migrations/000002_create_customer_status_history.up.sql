BEGIN;

-- История переходов статуса клиента.
CREATE TABLE customer_status_history (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID            NOT NULL REFERENCES customers (user_id) ON DELETE CASCADE,

    from_status customer_status,
    to_status   customer_status NOT NULL,
    reason      TEXT            NOT NULL DEFAULT '',
    actor_id    UUID,

    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),

    -- Переход в самого себя переходом не является и в историю не пишется.
    CONSTRAINT customer_status_history_progress_chk CHECK (from_status IS DISTINCT FROM to_status)
);

CREATE INDEX customer_status_history_user_id_created_at_idx
    ON customer_status_history (user_id, created_at DESC, id DESC);

COMMIT;
