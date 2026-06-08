package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *Handler) nethttp.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusNoContent)
	})

	r.Post("/accounts", handler.CreateAccount)
	r.Get("/accounts/{id}", handler.GetAccount)
	r.Post("/transfers", handler.Transfer)
	r.Get("/transfers/{account_id}", handler.ListTransfers)

	return r
}
