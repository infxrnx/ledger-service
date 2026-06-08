package usecase

import (
	"context"
	"strings"

	"github.com/portfolio/ledger-service/internal/domain"
)

type AccountUsecase struct {
	repo domain.AccountRepository
}

func NewAccountUsecase(repo domain.AccountRepository) *AccountUsecase {
	return &AccountUsecase{repo: repo}
}

func (u *AccountUsecase) CreateAccount(ctx context.Context, params domain.CreateAccountParams) (domain.Account, error) {
	params.UserID = strings.TrimSpace(params.UserID)
	params.Currency = strings.ToUpper(strings.TrimSpace(params.Currency))

	if params.UserID == "" || len(params.Currency) != 3 || params.Balance < 0 {
		return domain.Account{}, domain.ErrInvalidAccountData
	}

	return u.repo.Create(ctx, params)
}

func (u *AccountUsecase) GetAccount(ctx context.Context, id string) (domain.Account, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Account{}, domain.ErrInvalidAccountID
	}

	return u.repo.GetByID(ctx, id)
}

func (u *AccountUsecase) Transfer(ctx context.Context, params domain.TransferParams) ([]domain.LedgerEntry, error) {
	params.FromAccountID = strings.TrimSpace(params.FromAccountID)
	params.ToAccountID = strings.TrimSpace(params.ToAccountID)

	if params.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if params.FromAccountID == "" || params.ToAccountID == "" {
		return nil, domain.ErrInvalidAccountID
	}
	if params.FromAccountID == params.ToAccountID {
		return nil, domain.ErrSameAccountTransfer
	}

	return u.repo.Transfer(ctx, params)
}

func (u *AccountUsecase) ListTransfers(ctx context.Context, accountID string) ([]domain.LedgerEntry, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, domain.ErrInvalidAccountID
	}

	return u.repo.ListLedger(ctx, accountID)
}
