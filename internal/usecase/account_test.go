package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/portfolio/ledger-service/internal/domain"
)

func TestAccountUsecaseTransferValidation(t *testing.T) {
	ctx := context.Background()
	repo := &fakeAccountRepo{}
	uc := NewAccountUsecase(repo)

	tests := []struct {
		name   string
		params domain.TransferParams
		err    error
	}{
		{
			name: "invalid amount",
			params: domain.TransferParams{
				FromAccountID: "from",
				ToAccountID:   "to",
				Amount:        0,
			},
			err: domain.ErrInvalidAmount,
		},
		{
			name: "empty account id",
			params: domain.TransferParams{
				FromAccountID: "",
				ToAccountID:   "to",
				Amount:        10,
			},
			err: domain.ErrInvalidAccountID,
		},
		{
			name: "same account",
			params: domain.TransferParams{
				FromAccountID: "acc",
				ToAccountID:   "acc",
				Amount:        10,
			},
			err: domain.ErrSameAccountTransfer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Transfer(ctx, tt.params)
			require.ErrorIs(t, err, tt.err)
		})
	}

	require.Equal(t, 0, repo.transferCalls)
}

func TestAccountUsecaseTransferRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := &fakeAccountRepo{transferErr: domain.ErrInsufficientFunds}
	uc := NewAccountUsecase(repo)

	_, err := uc.Transfer(ctx, domain.TransferParams{
		FromAccountID: "from",
		ToAccountID:   "to",
		Amount:        25,
	})

	require.ErrorIs(t, err, domain.ErrInsufficientFunds)
	require.Equal(t, 1, repo.transferCalls)
}

func TestAccountUsecaseCreateAccountNormalizesInput(t *testing.T) {
	ctx := context.Background()
	repo := &fakeAccountRepo{}
	uc := NewAccountUsecase(repo)

	account, err := uc.CreateAccount(ctx, domain.CreateAccountParams{
		UserID:   " user-1 ",
		Currency: "usd",
		Balance:  1500,
	})

	require.NoError(t, err)
	require.Equal(t, "user-1", repo.created.UserID)
	require.Equal(t, "USD", repo.created.Currency)
	require.Equal(t, int64(1500), account.Balance)
}

func TestAccountUsecaseCreateAccountRejectsBadData(t *testing.T) {
	ctx := context.Background()
	repo := &fakeAccountRepo{}
	uc := NewAccountUsecase(repo)

	_, err := uc.CreateAccount(ctx, domain.CreateAccountParams{
		UserID:   "user-1",
		Currency: "US",
		Balance:  100,
	})

	require.ErrorIs(t, err, domain.ErrInvalidAccountData)
	require.Equal(t, 0, repo.createCalls)
}

type fakeAccountRepo struct {
	created       domain.CreateAccountParams
	transferErr   error
	createCalls   int
	transferCalls int
}

func (r *fakeAccountRepo) Create(ctx context.Context, params domain.CreateAccountParams) (domain.Account, error) {
	r.createCalls++
	r.created = params
	now := time.Now().UTC()
	return domain.Account{
		ID:        "account-1",
		UserID:    params.UserID,
		Currency:  params.Currency,
		Balance:   params.Balance,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *fakeAccountRepo) GetByID(ctx context.Context, id string) (domain.Account, error) {
	return domain.Account{ID: id}, nil
}

func (r *fakeAccountRepo) Transfer(ctx context.Context, params domain.TransferParams) ([]domain.LedgerEntry, error) {
	r.transferCalls++
	if r.transferErr != nil {
		return nil, r.transferErr
	}
	return []domain.LedgerEntry{
		{AccountID: params.FromAccountID, Direction: domain.EntryDebit, Amount: params.Amount},
		{AccountID: params.ToAccountID, Direction: domain.EntryCredit, Amount: params.Amount},
	}, nil
}

func (r *fakeAccountRepo) ListLedger(ctx context.Context, accountID string) ([]domain.LedgerEntry, error) {
	return nil, nil
}
