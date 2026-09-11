BEGIN;

CREATE TYPE application_status AS ENUM (
    'submitted', 'auto_checks', 'auto_rejected', 'in_review', 'docs_requested', 'approved', 'rejected'
);

CREATE TYPE document_type AS ENUM ('passport', 'selfie', 'proof_of_address');

CREATE TABLE applications (
    id              UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID               NOT NULL,
    status          application_status NOT NULL DEFAULT 'submitted',

    -- Офицер, взявший заявку в работу. NULL - заявку никто не взял.
    reviewer_id     UUID,
    -- Причина отказа или комментарий к запросу документов.
    reason          TEXT               NOT NULL DEFAULT '',
    -- Типы, которые попросили догрузить. Непусто только в docs_requested.
    requested_types document_type[]    NOT NULL DEFAULT '{}',

    -- Снимок анкеты на момент подачи.
    first_name      TEXT               NOT NULL DEFAULT '',
    last_name       TEXT               NOT NULL DEFAULT '',
    birth_date      DATE,

    submitted_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),
    decided_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ        NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ        NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX applications_active_user_id_key
    ON applications (user_id)
    WHERE status IN ('submitted', 'auto_checks', 'in_review', 'docs_requested');

-- Очередь офицера: выборка по статусу, свежие сверху.
CREATE INDEX applications_status_created_at_idx
    ON applications (status, created_at DESC, id DESC);

CREATE INDEX applications_user_id_created_at_idx
    ON applications (user_id, created_at DESC, id DESC);

COMMIT;
