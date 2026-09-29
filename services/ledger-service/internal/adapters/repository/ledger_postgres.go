package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

const BALANCE_KIND_CASH = "cash"

type LedgerRepo struct {
	db *pgxpool.Pool
}

func NewLedgerRepo(db *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{db: db}
}

func (r *LedgerRepo) EnsureCashAccounts(ctx context.Context, currencies []models.Currency) (map[models.Currency]uuid.UUID, error) {
	for _, currency := range currencies {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("ensure cash accounts: id: %w", err)
		}

		if _, err := r.db.Exec(ctx, ENSURE_CASH_ACCOUNT_QUERY, id, currency); err != nil {
			return nil, fmt.Errorf("ensure cash account %s: %w", currency, err)
		}
	}

	rows, err := r.db.Query(ctx, LIST_CASH_ACCOUNTS_QUERY)
	if err != nil {
		return nil, fmt.Errorf("list cash accounts: %w", err)
	}
	defer rows.Close()

	cash := make(map[models.Currency]uuid.UUID, len(currencies))

	for rows.Next() {
		var (
			currency string
			id       uuid.UUID
		)

		if err := rows.Scan(&currency, &id); err != nil {
			return nil, fmt.Errorf("scan cash accounts: %w", err)
		}

		cash[models.Currency(currency)] = id
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cash accounts: %w", err)
	}

	return cash, nil
}

// Post: проверка средств, проводки, остатки и outbox - в одной транзакции БД под блокировкой обоих остатков.
func (r *LedgerRepo) Post(ctx context.Context, p models.Posting) (*models.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("post: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	balances, err := lockBalances(ctx, tx, p)
	if err != nil {
		return nil, err
	}

	from := balances[p.From]

	if from.kind != BALANCE_KIND_CASH && from.amount < p.Amount {
		declined, err := insertTransaction(ctx, tx, p, models.STATUS_DECLINED, domain.DECLINE_INSUFFICIENT_FUNDS)
		if err != nil {
			return nil, err
		}

		if err := commitWithEvent(ctx, tx, declined); err != nil {
			return nil, err
		}

		return declined, domain.ErrInsufficientFunds
	}

	completed, err := insertTransaction(ctx, tx, p, models.STATUS_COMPLETED, "")
	if err != nil {
		return nil, err
	}

	entries := []models.Entry{
		{AccountID: p.From, Direction: models.DIRECTION_DEBIT},
		{AccountID: p.To, Direction: models.DIRECTION_CREDIT},
	}

	for i := range entries {
		entry, err := insertEntry(ctx, tx, completed, entries[i])
		if err != nil {
			return nil, err
		}

		entries[i] = entry
	}

	if _, err := tx.Exec(ctx, UPDATE_BALANCE_QUERY, p.From, -p.Amount); err != nil {
		return nil, fmt.Errorf("post: debit balance: %w", err)
	}

	if _, err := tx.Exec(ctx, UPDATE_BALANCE_QUERY, p.To, p.Amount); err != nil {
		return nil, fmt.Errorf("post: credit balance: %w", err)
	}

	completed.Entries = entries

	if err := commitWithEvent(ctx, tx, completed); err != nil {
		return nil, err
	}

	return completed, nil
}

// Decline отмена перевода
func (r *LedgerRepo) Decline(ctx context.Context, p models.Posting, reason string) (*models.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("decline: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureBalances(ctx, tx, p); err != nil {
		return nil, err
	}

	declined, err := insertTransaction(ctx, tx, p, models.STATUS_DECLINED, reason)
	if err != nil {
		return nil, err
	}

	if err := commitWithEvent(ctx, tx, declined); err != nil {
		return nil, err
	}

	return declined, nil
}

func (r *LedgerRepo) GetTransaction(ctx context.Context, transactionId uuid.UUID) (*models.Transaction, error) {
	transaction, err := scanTransaction(r.db.QueryRow(ctx, GET_TRANSACTION_QUERY, transactionId))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTransactionNotFound
		}

		return nil, fmt.Errorf("get transaction: %w", err)
	}

	entries, err := listEntries(ctx, r.db, transactionId)
	if err != nil {
		return nil, err
	}

	transaction.Entries = entries

	return transaction, nil
}

func (r *LedgerRepo) ListTransactions(ctx context.Context, accountID uuid.UUID, page models.Page) ([]*models.Transaction, error) {
	rows, err := r.db.Query(ctx, LIST_TRANSACTIONS_QUERY, accountID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]*models.Transaction, 0, page.Limit)

	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transactions: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}

	return transactions, nil
}

func (r *LedgerRepo) GetBalance(ctx context.Context, accountID uuid.UUID) (*models.Balance, error) {
	var (
		balance  models.Balance
		currency string
	)

	err := r.db.QueryRow(ctx, GET_BALANCE_QUERY, accountID).Scan(&balance.AccountID, &currency, &balance.Amount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBalanceNotFound
		}

		return nil, fmt.Errorf("get balance: %w", err)
	}

	balance.Currency = models.Currency(currency)

	return &balance, nil
}

type lockedBalance struct {
	kind     string
	currency models.Currency
	amount   int64
}

func ensureBalances(ctx context.Context, tx pgx.Tx, p models.Posting) error {
	ids := []uuid.UUID{p.From, p.To}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })

	for _, id := range ids {
		if _, err := tx.Exec(ctx, ENSURE_BALANCE_QUERY, id, p.Currency); err != nil {
			return fmt.Errorf("ensure balance %s: %w", id, err)
		}
	}

	return nil
}

// lockBalances заводит недостающие строки остатков и блокирует обе, сверяя валюту.
func lockBalances(ctx context.Context, tx pgx.Tx, p models.Posting) (map[uuid.UUID]lockedBalance, error) {
	if err := ensureBalances(ctx, tx, p); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, LOCK_BALANCES_QUERY, []uuid.UUID{p.From, p.To})
	if err != nil {
		return nil, fmt.Errorf("lock balances: %w", err)
	}
	defer rows.Close()

	balances := make(map[uuid.UUID]lockedBalance, 2)

	for rows.Next() {
		var (
			id       uuid.UUID
			balance  lockedBalance
			currency string
		)

		if err := rows.Scan(&id, &balance.kind, &currency, &balance.amount); err != nil {
			return nil, fmt.Errorf("scan locked balance: %w", err)
		}

		balance.currency = models.Currency(currency)
		balances[id] = balance
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locked balances: %w", err)
	}

	for _, id := range []uuid.UUID{p.From, p.To} {
		balance, ok := balances[id]
		if !ok {
			return nil, fmt.Errorf("lock balances: balance %s is missing", id)
		}

		if balance.currency != p.Currency {
			return nil, fmt.Errorf("%w: account %s is in %s", domain.ErrCurrencyMismatch, id, balance.currency)
		}
	}

	return balances, nil
}

func insertTransaction(
	ctx context.Context,
	tx pgx.Tx,
	p models.Posting,
	status models.TransactionStatus,
	declineReason string,
) (*models.Transaction, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("transaction id: %w", err)
	}

	transaction, err := scanTransaction(tx.QueryRow(ctx, INSERT_TRANSACTION_QUERY,
		id,
		p.Type,
		status,
		p.From,
		p.To,
		p.Amount,
		p.Currency,
		p.InitiatorID,
		declineReason,
	))
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	transaction.FromUserID = p.FromUserID
	transaction.ToUserID = p.ToUserID

	return transaction, nil
}

func insertEntry(ctx context.Context, tx pgx.Tx, transaction *models.Transaction, entry models.Entry) (models.Entry, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return models.Entry{}, fmt.Errorf("entry id: %w", err)
	}

	entry.ID = id
	entry.TransactionID = transaction.ID
	entry.Amount = transaction.Amount
	entry.Currency = transaction.Currency
	entry.CreatedAt = transaction.CreatedAt

	_, err = tx.Exec(ctx, INSERT_ENTRY_QUERY,
		entry.ID,
		entry.TransactionID,
		entry.AccountID,
		entry.Direction,
		entry.Amount,
		entry.Currency,
		entry.CreatedAt,
	)
	if err != nil {
		return models.Entry{}, fmt.Errorf("insert entry: %w", err)
	}

	return entry, nil
}

// commitWithEvent кладёт событие в outbox и коммитит: событие появится ровно тогда, когда появятся деньги.
func commitWithEvent(ctx context.Context, tx pgx.Tx, transaction *models.Transaction) error {
	msg, err := models.NewTransactionEvent(transaction)
	if err != nil {
		return fmt.Errorf("build event: %w", err)
	}

	_, err = tx.Exec(ctx, INSERT_OUTBOX_QUERY, msg.ID, msg.Key, msg.EventType, []byte(msg.Payload), msg.OccurredAt)
	if err != nil {
		return fmt.Errorf("write outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listEntries(ctx context.Context, q querier, transactionID uuid.UUID) ([]models.Entry, error) {
	rows, err := q.Query(ctx, LIST_ENTRIES_QUERY, transactionID)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	entries := make([]models.Entry, 0, 2)

	for rows.Next() {
		var (
			entry     models.Entry
			direction string
			currency  string
		)

		err := rows.Scan(&entry.ID, &entry.TransactionID, &entry.AccountID, &direction, &entry.Amount, &currency, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan entries: %w", err)
		}

		entry.Direction = models.Direction(direction)
		entry.Currency = models.Currency(currency)
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return entries, nil
}

type row interface {
	Scan(dest ...any) error
}

func scanTransaction(r row) (*models.Transaction, error) {
	var (
		transaction models.Transaction
		txType      string
		status      string
		currency    string
		createdAt   time.Time
	)

	err := r.Scan(
		&transaction.ID,
		&txType,
		&status,
		&transaction.FromAccountID,
		&transaction.ToAccountID,
		&transaction.Amount,
		&currency,
		&transaction.InitiatorID,
		&transaction.DeclineReason,
		&createdAt,
	)
	if err != nil {
		return nil, err
	}

	transaction.Type = models.TransactionType(txType)
	transaction.Status = models.TransactionStatus(status)
	transaction.Currency = models.Currency(currency)
	transaction.CreatedAt = createdAt.UTC()

	return &transaction, nil
}
