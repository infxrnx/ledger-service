package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/portfolio/ledger-service/internal/domain"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Create(ctx context.Context, params domain.CreateAccountParams) (domain.Account, error) {
	const query = `
		INSERT INTO accounts (user_id, currency, balance)
		VALUES ($1, upper($2), $3)
		RETURNING id::text, user_id, currency, balance, created_at, updated_at`

	var account domain.Account
	err := r.pool.QueryRow(ctx, query, params.UserID, params.Currency, params.Balance).Scan(
		&account.ID,
		&account.UserID,
		&account.Currency,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return domain.Account{}, fmt.Errorf("create account: %w", err)
	}

	return account, nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (domain.Account, error) {
	const query = `
		SELECT id::text, user_id, currency, balance, created_at, updated_at
		FROM accounts
		WHERE id = $1::uuid`

	account, err := scanAccount(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, fmt.Errorf("get account: %w", err)
	}

	return account, nil
}

func (r *AccountRepository) Transfer(ctx context.Context, params domain.TransferParams) ([]domain.LedgerEntry, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transfer tx: %w", err)
	}
	defer tx.Rollback(ctx)

	accounts, err := lockTransferAccounts(ctx, tx, params.FromAccountID, params.ToAccountID)
	if err != nil {
		return nil, err
	}

	from, ok := accounts[params.FromAccountID]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	to, ok := accounts[params.ToAccountID]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}

	if from.Currency != to.Currency {
		return nil, domain.ErrCurrencyMismatch
	}
	if from.Balance < params.Amount {
		return nil, domain.ErrInsufficientFunds
	}

	from.Balance -= params.Amount
	to.Balance += params.Amount

	if err := updateBalance(ctx, tx, from); err != nil {
		return nil, err
	}
	if err := updateBalance(ctx, tx, to); err != nil {
		return nil, err
	}

	transferID, err := newTransferID(ctx, tx)
	if err != nil {
		return nil, err
	}

	debit, err := insertLedgerEntry(ctx, tx, from.ID, transferID, domain.EntryDebit, params.Amount, from.Balance)
	if err != nil {
		return nil, err
	}
	credit, err := insertLedgerEntry(ctx, tx, to.ID, transferID, domain.EntryCredit, params.Amount, to.Balance)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transfer tx: %w", err)
	}

	return []domain.LedgerEntry{debit, credit}, nil
}

func (r *AccountRepository) ListLedger(ctx context.Context, accountID string) ([]domain.LedgerEntry, error) {
	const query = `
		SELECT id::text, account_id::text, transfer_id::text, direction, amount, balance_after, created_at
		FROM ledger_entries
		WHERE account_id = $1::uuid
		ORDER BY created_at DESC, id DESC`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("list ledger entries: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.LedgerEntry, 0)
	for rows.Next() {
		entry, err := scanLedgerEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read ledger entries: %w", err)
	}

	return entries, nil
}

func lockTransferAccounts(ctx context.Context, tx pgx.Tx, fromID, toID string) (map[string]domain.Account, error) {
	const query = `
		SELECT id::text, user_id, currency, balance, created_at, updated_at
		FROM accounts
		WHERE id IN ($1::uuid, $2::uuid)
		ORDER BY id
		FOR UPDATE`

	rows, err := tx.Query(ctx, query, fromID, toID)
	if err != nil {
		return nil, fmt.Errorf("lock transfer accounts: %w", err)
	}
	defer rows.Close()

	accounts := make(map[string]domain.Account, 2)
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts[account.ID] = account
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read locked accounts: %w", err)
	}

	return accounts, nil
}

func updateBalance(ctx context.Context, tx pgx.Tx, account domain.Account) error {
	const query = `
		UPDATE accounts
		SET balance = $2, updated_at = now()
		WHERE id = $1::uuid`

	tag, err := tx.Exec(ctx, query, account.ID, account.Balance)
	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrInvalidTransferState
	}
	return nil
}

func newTransferID(ctx context.Context, tx pgx.Tx) (string, error) {
	var id string
	if err := tx.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&id); err != nil {
		return "", fmt.Errorf("create transfer id: %w", err)
	}
	return id, nil
}

func insertLedgerEntry(ctx context.Context, tx pgx.Tx, accountID, transferID string, direction domain.EntryDirection, amount, balanceAfter int64) (domain.LedgerEntry, error) {
	const query = `
		INSERT INTO ledger_entries (account_id, transfer_id, direction, amount, balance_after)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)
		RETURNING id::text, account_id::text, transfer_id::text, direction, amount, balance_after, created_at`

	entry, err := scanLedgerEntry(tx.QueryRow(ctx, query, accountID, transferID, direction, amount, balanceAfter))
	if err != nil {
		return domain.LedgerEntry{}, fmt.Errorf("insert ledger entry: %w", err)
	}
	return entry, nil
}

func scanAccount(row pgx.Row) (domain.Account, error) {
	var account domain.Account
	err := row.Scan(
		&account.ID,
		&account.UserID,
		&account.Currency,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	return account, err
}

func scanLedgerEntry(row pgx.Row) (domain.LedgerEntry, error) {
	var entry domain.LedgerEntry
	err := row.Scan(
		&entry.ID,
		&entry.AccountID,
		&entry.TransferID,
		&entry.Direction,
		&entry.Amount,
		&entry.BalanceAfter,
		&entry.CreatedAt,
	)
	return entry, err
}
