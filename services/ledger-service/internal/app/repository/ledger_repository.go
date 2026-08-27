package repository

type Ledger interface {
	GetTransaction() error
}
