package http

import (
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"

	"github.com/go-chi/chi/v5"

	"github.com/portfolio/ledger-service/internal/domain"
)

type accountService interface {
	CreateAccount(ctx context.Context, params domain.CreateAccountParams) (domain.Account, error)
	GetAccount(ctx context.Context, id string) (domain.Account, error)
	Transfer(ctx context.Context, params domain.TransferParams) ([]domain.LedgerEntry, error)
	ListTransfers(ctx context.Context, accountID string) ([]domain.LedgerEntry, error)
}

type Handler struct {
	accounts accountService
}

func NewHandler(accounts accountService) *Handler {
	return &Handler{accounts: accounts}
}

func (h *Handler) CreateAccount(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid json body")
		return
	}

	account, err := h.accounts.CreateAccount(r.Context(), domain.CreateAccountParams{
		UserID:   req.UserID,
		Currency: req.Currency,
		Balance:  req.Balance,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, nethttp.StatusCreated, accountResponseFromDomain(account))
}

func (h *Handler) GetAccount(w nethttp.ResponseWriter, r *nethttp.Request) {
	account, err := h.accounts.GetAccount(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, nethttp.StatusOK, accountResponseFromDomain(account))
}

func (h *Handler) Transfer(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid json body")
		return
	}

	entries, err := h.accounts.Transfer(r.Context(), domain.TransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	resp := make([]ledgerEntryResponse, 0, len(entries))
	for _, entry := range entries {
		resp = append(resp, ledgerEntryResponseFromDomain(entry))
	}

	writeJSON(w, nethttp.StatusCreated, transferResponse{Entries: resp})
}

func (h *Handler) ListTransfers(w nethttp.ResponseWriter, r *nethttp.Request) {
	entries, err := h.accounts.ListTransfers(r.Context(), chi.URLParam(r, "account_id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	resp := make([]ledgerEntryResponse, 0, len(entries))
	for _, entry := range entries {
		resp = append(resp, ledgerEntryResponseFromDomain(entry))
	}

	writeJSON(w, nethttp.StatusOK, transferResponse{Entries: resp})
}

func writeDomainError(w nethttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		writeError(w, nethttp.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrSameAccountTransfer),
		errors.Is(err, domain.ErrCurrencyMismatch),
		errors.Is(err, domain.ErrInvalidAccountData),
		errors.Is(err, domain.ErrInvalidAccountID):
		writeError(w, nethttp.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInsufficientFunds):
		writeError(w, nethttp.StatusConflict, err.Error())
	default:
		writeError(w, nethttp.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w nethttp.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w nethttp.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

type createAccountRequest struct {
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
	Balance  int64  `json:"balance"`
}

type transferRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
}

type accountResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ledgerEntryResponse struct {
	ID           string `json:"id"`
	AccountID    string `json:"account_id"`
	TransferID   string `json:"transfer_id"`
	Direction    string `json:"direction"`
	Amount       int64  `json:"amount"`
	BalanceAfter int64  `json:"balance_after"`
	CreatedAt    string `json:"created_at"`
}

type transferResponse struct {
	Entries []ledgerEntryResponse `json:"entries"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func accountResponseFromDomain(account domain.Account) accountResponse {
	return accountResponse{
		ID:        account.ID,
		UserID:    account.UserID,
		Currency:  account.Currency,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: account.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ledgerEntryResponseFromDomain(entry domain.LedgerEntry) ledgerEntryResponse {
	return ledgerEntryResponse{
		ID:           entry.ID,
		AccountID:    entry.AccountID,
		TransferID:   entry.TransferID,
		Direction:    string(entry.Direction),
		Amount:       entry.Amount,
		BalanceAfter: entry.BalanceAfter,
		CreatedAt:    entry.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
