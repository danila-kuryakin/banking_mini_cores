package repository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/migrations"
)

const (
	TEST_CURRENCY      = models.Currency("RUB")
	TEST_START_BALANCE = 1_000
	TEST_TRANSFERS     = 100
	TEST_AMOUNT_A_TO_B = 70
	TEST_AMOUNT_B_TO_A = 30
	TEST_DB_PORT_ENV   = "LEDGER_TEST_DB_PORT"
	TEST_DB_NAME       = "banking_ledger"
)

// testPool поднимает схему в отдельной базе; без LEDGER_TEST_DB_PORT тест пропускается, чтобы не задеть рабочую базу.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	rawPort := os.Getenv(TEST_DB_PORT_ENV)
	if rawPort == "" {
		t.Skipf("%s is not set: the integration test needs a separate Postgres", TEST_DB_PORT_ENV)
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatalf("%s: %v", TEST_DB_PORT_ENV, err)
	}

	cfg := config.DataBaseConfig{
		Host:     "localhost",
		Port:     port,
		User:     TEST_DB_NAME,
		Password: "123",
		DBName:   TEST_DB_NAME,
		SSLMode:  "disable",
		MaxConns: 20,
	}

	if err := migrator.Up(cfg, migrations.FS, migrations.Dir); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := conn.NewConnectionDB(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func TestConcurrentTransfers(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	repo := NewLedgerRepo(pool)

	cash, err := repo.EnsureCashAccounts(ctx, []models.Currency{TEST_CURRENCY})
	if err != nil {
		t.Fatal(err)
	}

	a, b := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	for _, id := range []uuid.UUID{a, b} {
		_, err := repo.Post(ctx, models.Posting{
			Type: models.TYPE_DEPOSIT, From: cash[TEST_CURRENCY], To: id, Amount: TEST_START_BALANCE, Currency: TEST_CURRENCY,
		})
		if err != nil {
			t.Fatalf("deposit: %v", err)
		}
	}

	type result struct {
		tx  *models.Transaction
		err error
	}

	results := make([]result, TEST_TRANSFERS)

	var wg sync.WaitGroup

	for i := range TEST_TRANSFERS {
		wg.Go(func() {
			p := models.Posting{Type: models.TYPE_TRANSFER, From: a, To: b, Amount: TEST_AMOUNT_A_TO_B, Currency: TEST_CURRENCY}
			if i%2 == 1 {
				p = models.Posting{Type: models.TYPE_TRANSFER, From: b, To: a, Amount: TEST_AMOUNT_B_TO_A, Currency: TEST_CURRENCY}
			}

			tx, err := repo.Post(ctx, p)
			results[i] = result{tx: tx, err: err}
		})
	}

	wg.Wait()

	var expectA, expectB int64 = TEST_START_BALANCE, TEST_START_BALANCE

	completed, declined := 0, 0
	ids := make(map[uuid.UUID]bool, TEST_TRANSFERS)

	for i, r := range results {
		switch {
		case r.err == nil:
			completed++

			if r.tx.FromAccountID == a {
				expectA -= r.tx.Amount
				expectB += r.tx.Amount
			} else {
				expectB -= r.tx.Amount
				expectA += r.tx.Amount
			}
		case errors.Is(r.err, domain.ErrInsufficientFunds):
			declined++

			if r.tx == nil || r.tx.Status != models.STATUS_DECLINED {
				t.Fatalf("transfer %d: declined transaction is not returned", i)
			}
		default:
			t.Fatalf("transfer %d: %v", i, r.err)
		}

		if ids[r.tx.ID] {
			t.Fatalf("transaction %s is duplicated", r.tx.ID)
		}

		ids[r.tx.ID] = true
	}

	if completed+declined != TEST_TRANSFERS {
		t.Fatalf("lost transfers: completed %d, declined %d", completed, declined)
	}

	if declined == 0 {
		t.Fatal("scenario must hit insufficient funds at least once")
	}

	balanceA, balanceB := mustBalance(t, repo, a), mustBalance(t, repo, b)

	if balanceA != expectA || balanceB != expectB {
		t.Fatalf("balances: got a=%d b=%d, want a=%d b=%d", balanceA, balanceB, expectA, expectB)
	}

	if balanceA < 0 || balanceB < 0 {
		t.Fatalf("negative balance: a=%d b=%d", balanceA, balanceB)
	}

	if balanceA+balanceB != 2*TEST_START_BALANCE {
		t.Fatalf("money is not conserved: %d", balanceA+balanceB)
	}

	for _, id := range []uuid.UUID{a, b} {
		if fromEntries := entriesBalance(t, pool, id); fromEntries != mustBalance(t, repo, id) {
			t.Fatalf("account %s: materialized balance differs from entries: %d", id, fromEntries)
		}
	}

	var stored, events int

	err = pool.QueryRow(ctx, `SELECT count(*) FROM transactions WHERE type = 'transfer' AND from_account_id = ANY($1::uuid[])`, []uuid.UUID{a, b}).Scan(&stored)
	if err != nil {
		t.Fatal(err)
	}

	if stored != TEST_TRANSFERS {
		t.Fatalf("stored %d transfers, want %d", stored, TEST_TRANSFERS)
	}

	err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE payload->>'transaction_id' = ANY($1::text[])`, idStrings(ids)).Scan(&events)
	if err != nil {
		t.Fatal(err)
	}

	if events != TEST_TRANSFERS {
		t.Fatalf("outbox has %d events, want %d", events, TEST_TRANSFERS)
	}
}

func TestUnbalancedTransactionIsRejected(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	repo := NewLedgerRepo(pool)

	cash, err := repo.EnsureCashAccounts(ctx, []models.Currency{TEST_CURRENCY})
	if err != nil {
		t.Fatal(err)
	}

	account := uuid.Must(uuid.NewV7())

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	p := models.Posting{Type: models.TYPE_DEPOSIT, From: cash[TEST_CURRENCY], To: account, Amount: 100, Currency: TEST_CURRENCY}

	if err := ensureBalances(ctx, tx, p); err != nil {
		t.Fatal(err)
	}

	transaction, err := insertTransaction(ctx, tx, p, models.STATUS_COMPLETED, "")
	if err != nil {
		t.Fatal(err)
	}

	// Только кредит без дебета: деньги из ниоткуда.
	if _, err := insertEntry(ctx, tx, transaction, models.Entry{AccountID: account, Direction: models.DIRECTION_CREDIT}); err != nil {
		t.Fatal(err)
	}

	if err := tx.Commit(ctx); err == nil {
		t.Fatal("commit of an unbalanced transaction must fail")
	}
}

func mustBalance(t *testing.T, repo *LedgerRepo, id uuid.UUID) int64 {
	t.Helper()

	balance, err := repo.GetBalance(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}

	return balance.Amount
}

func entriesBalance(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) int64 {
	t.Helper()

	var sum int64

	err := pool.QueryRow(context.Background(), `
		SELECT coalesce(sum(CASE direction WHEN 'credit' THEN amount ELSE -amount END), 0)
		FROM entries WHERE account_id = $1`, id).Scan(&sum)
	if err != nil {
		t.Fatal(err)
	}

	return sum
}

func idStrings(ids map[uuid.UUID]bool) []string {
	out := make([]string, 0, len(ids))

	for id := range ids {
		out = append(out, id.String())
	}

	return out
}
