BEGIN;

-- Дедупликация событий Kafka: доставка at-least-once.
CREATE TABLE processed_events (
    event_id     UUID        PRIMARY KEY,
    event_type   TEXT        NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
    id         UUID        PRIMARY KEY,
    user_id    UUID        NOT NULL,
    event_id   UUID        NOT NULL REFERENCES processed_events (event_id),
    event_type TEXT        NOT NULL,
    text       TEXT        NOT NULL CHECK (text <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT notifications_event_user_key UNIQUE (event_id, user_id)
);

CREATE INDEX notifications_user_id_created_at_idx ON notifications (user_id, created_at DESC, id DESC);

COMMIT;
