package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/midavuha1/go-course-lab/internal/config"
	"github.com/midavuha1/go-course-lab/internal/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Инициализация пула соединений.
	// Если база недоступна, сервис упадет здесь, не дойдя до поднятия HTTP-сервера.
	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init database pool: %w", err)
	}

	// 2. Гарантированное закрытие пула при выходе из run().
	// Благодаря стеку defer, это выполнится ПОСЛЕ srv.Shutdown(),
	// что соответствует правилу: сначала останавливаем прием запросов,
	// дожидаясь текущих, и только потом рвем соединения с БД.
	defer pool.Close()

	r := chi.NewRouter()
	// TODO позже: сюда подключим сгенерированные ручки, передав им pool

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting HTTP server", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	slog.Info("graceful shutdown initiated", "timeout", cfg.ShutdownTimeout)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}
