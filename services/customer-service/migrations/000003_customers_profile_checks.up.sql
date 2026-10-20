BEGIN;

-- Значения синхронизируются с internal/domain/constants.go.
ALTER TABLE customers
    ADD CONSTRAINT customers_first_name_chk CHECK (char_length(first_name) <= 100 AND first_name = btrim(first_name)),
    ADD CONSTRAINT customers_last_name_chk CHECK (char_length(last_name) <= 100 AND last_name = btrim(last_name)),
    ADD CONSTRAINT customers_citizenship_chk CHECK (char_length(citizenship) <= 64 AND citizenship = btrim(citizenship)),
    ADD CONSTRAINT customers_phone_chk CHECK (phone = '' OR phone ~ '^\+?[0-9]{7,15}$');

COMMIT;
