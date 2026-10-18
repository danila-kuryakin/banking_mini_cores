BEGIN;

CREATE TABLE login_attempts (
    email             TEXT        PRIMARY KEY,   -- нормализованный email, без FK: считаем и несуществующие
    failed_count      INTEGER     NOT NULL CHECK (failed_count > 0),
    window_started_at TIMESTAMPTZ NOT NULL,
    locked_until      TIMESTAMPTZ,
    updated_at        TIMESTAMPTZ NOT NULL
);

COMMIT;
