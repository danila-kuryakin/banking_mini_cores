BEGIN;

DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS balances;
DROP FUNCTION IF EXISTS forbid_ledger_rewrite();
DROP FUNCTION IF EXISTS check_transaction_balanced();
DROP TYPE IF EXISTS entry_direction;
DROP TYPE IF EXISTS transaction_status;
DROP TYPE IF EXISTS transaction_type;
DROP TYPE IF EXISTS balance_kind;

COMMIT;
