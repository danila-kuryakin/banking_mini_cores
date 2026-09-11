BEGIN;

CREATE TYPE check_type AS ENUM ('sanctions', 'age');
CREATE TYPE check_result AS ENUM ('passed', 'failed');

-- Результат каждой авто-проверки по заявке.
CREATE TABLE kyc_checks (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID         NOT NULL REFERENCES applications (id) ON DELETE CASCADE,
    type           check_type   NOT NULL,
    result         check_result NOT NULL,
    details        TEXT         NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Одна проверка каждого типа на заявку: воркер может взять заявку повторно
-- после падения, и результат должен перезаписываться, а не удваиваться.
CREATE UNIQUE INDEX kyc_checks_application_id_type_key
    ON kyc_checks (application_id, type);

COMMIT;
