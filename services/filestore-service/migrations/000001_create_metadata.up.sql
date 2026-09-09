BEGIN;

CREATE TYPE file_type AS ENUM (
    'passport', 'selfie', 'proof_of_address'
);

CREATE TYPE file_status AS ENUM (
    'uploaded', 'confirmed'
);

CREATE TABLE metadata (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL,
    type          file_type   NOT NULL,
    status        file_status NOT NULL DEFAULT 'uploaded',

    object_path   TEXT        NOT NULL,           -- путь в MinIO: {user_id}/{type}/{id}
    filename      TEXT        NOT NULL DEFAULT '', -- исходное имя файла у клиента, из InitUpload
    content_type  TEXT        NOT NULL DEFAULT '',-- реальный тип по magic bytes, а не заявленный клиентом
    size_bytes    BIGINT      NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    sha256        TEXT        NOT NULL DEFAULT '', -- контрольная сумма

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at  TIMESTAMPTZ                     -- NULL, пока status = 'uploaded'
);

CREATE UNIQUE INDEX metadata_object_path_key ON metadata (object_path);
CREATE INDEX metadata_user_id_created_at_idx ON metadata (user_id, created_at DESC, id DESC);
CREATE UNIQUE INDEX metadata_user_id_type_confirmed_key
    ON metadata (user_id, type) WHERE status = 'confirmed';

COMMIT;
