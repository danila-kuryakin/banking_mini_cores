BEGIN;

ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS customers_phone_chk,
    DROP CONSTRAINT IF EXISTS customers_citizenship_chk,
    DROP CONSTRAINT IF EXISTS customers_last_name_chk,
    DROP CONSTRAINT IF EXISTS customers_first_name_chk;

COMMIT;
