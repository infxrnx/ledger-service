package main

import (
	"context"
	"errors"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/portfolio/ledger-service/internal/delivery/http"

	"github.com/portfolio/ledger-service/internal/config"
	repopg "github.com/portfolio/ledger-service/internal/repository/postgres"
	"github.com/portfolio/ledger-service/internal/usecase"
	"github.com/portfolio/ledger-service/pkg/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	accountRepo := repopg.NewAccountRepository(pool)
	accountUsecase := usecase.NewAccountUsecase(accountRepo)
	handler := httpdelivery.NewHandler(accountUsecase)

	server := &nethttp.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      httpdelivery.NewRouter(handler),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server started", "addr", cfg.HTTP.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("http server stopped")
}
