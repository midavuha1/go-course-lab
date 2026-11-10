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

	"github.com/midavuha1/go-course-lab/internal/config"
	"github.com/midavuha1/go-course-lab/internal/container"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c, err := container.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("build container: %w", err)
	}
	logger := c.Logger

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting HTTP server", slog.String("addr", cfg.HTTPAddr))
		if err := c.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		runErr = fmt.Errorf("listen and serve: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	logger.Info("graceful shutdown initiated", slog.Duration("timeout", cfg.ShutdownTimeout))
	if err := c.Server.Shutdown(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("server shutdown: %w", err))
	}

	if err := c.Close(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("close dependencies: %w", err))
	}
	if runErr == nil {
		logger.Info("server stopped gracefully")
	}
	return runErr
}
