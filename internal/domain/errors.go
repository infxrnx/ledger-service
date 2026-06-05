package domain

import "errors"

var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrInvalidAmount        = errors.New("amount must be greater than zero")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrSameAccountTransfer  = errors.New("source and destination accounts are the same")
	ErrCurrencyMismatch     = errors.New("accounts have different currencies")
	ErrInvalidAccountData   = errors.New("invalid account data")
	ErrInvalidAccountID     = errors.New("invalid account id")
	ErrInvalidTransferState = errors.New("invalid transfer state")
)
