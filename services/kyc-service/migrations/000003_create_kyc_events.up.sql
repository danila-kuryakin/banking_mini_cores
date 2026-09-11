BEGIN;

-- Аудит заявки: кто, когда и какой переход сделал. Пишется в одной транзакции со сменой статуса заявки.
CREATE TABLE kyc_events (
    id             UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID               NOT NULL REFERENCES applications (id) ON DELETE CASCADE,

    from_status    application_status,
    to_status      application_status NOT NULL,

    actor_id       UUID,
    comment        TEXT               NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now(),

    -- Переход статуса в самого себя переходом не является и в историю не пишется.
    CONSTRAINT kyc_events_progress_chk CHECK (from_status IS DISTINCT FROM to_status)
);

CREATE INDEX kyc_events_application_id_created_at_idx
    ON kyc_events (application_id, created_at DESC, id DESC);

COMMIT;
