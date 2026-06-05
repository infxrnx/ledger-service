package domain

import (
	"context"
	"time"
)

type Account struct {
	ID        string
	UserID    string
	Currency  string
	Balance   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LedgerEntry struct {
	ID           string
	AccountID    string
	TransferID   string
	Direction    EntryDirection
	Amount       int64
	BalanceAfter int64
	CreatedAt    time.Time
}

type EntryDirection string

const (
	EntryDebit  EntryDirection = "debit"
	EntryCredit EntryDirection = "credit"
)

type CreateAccountParams struct {
	UserID   string
	Currency string
	Balance  int64
}

type TransferParams struct {
	FromAccountID string
	ToAccountID   string
	Amount        int64
}

type AccountRepository interface {
	Create(ctx context.Context, params CreateAccountParams) (Account, error)
	GetByID(ctx context.Context, id string) (Account, error)
	Transfer(ctx context.Context, params TransferParams) ([]LedgerEntry, error)
	ListLedger(ctx context.Context, accountID string) ([]LedgerEntry, error)
}
