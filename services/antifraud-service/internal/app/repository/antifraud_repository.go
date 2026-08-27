package repository

type Antifraud interface {
	CheckTransfer() error
}
