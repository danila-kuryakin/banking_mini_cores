BEGIN;

CREATE TABLE sanctions_list (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT        NOT NULL,
    last_name  TEXT        NOT NULL,
    birth_date DATE        NOT NULL,
    -- Идентификатор источника откуда запись.
    source     TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Сверка идёт без учёта регистра, поэтому и уникальность такая же: иначе
-- "Petrov" и "petrov" стали бы двумя разными людьми.
CREATE UNIQUE INDEX sanctions_list_person_key
    ON sanctions_list (lower(last_name), lower(first_name), birth_date);

CREATE INDEX sanctions_list_lookup_idx
    ON sanctions_list (lower(last_name), birth_date);

COMMIT;
