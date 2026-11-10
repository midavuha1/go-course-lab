// Package container создаёт все зависимости сервиса и связывает их.
// Это единственное место, где вызываются конструкторы инфраструктуры и слоёв.
package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/midavuha1/go-course-lab/internal/config"
	"github.com/midavuha1/go-course-lab/internal/http/health"
	"github.com/midavuha1/go-course-lab/internal/http/router"
	createhttp "github.com/midavuha1/go-course-lab/internal/http/trip/create"
	finishhttp "github.com/midavuha1/go-course-lab/internal/http/trip/finish"
	gethttp "github.com/midavuha1/go-course-lab/internal/http/trip/get"
	"github.com/midavuha1/go-course-lab/internal/infra/postgres"
	"github.com/midavuha1/go-course-lab/internal/infra/postgres/transaction"
	"github.com/midavuha1/go-course-lab/internal/infra/telemetry"
	triprepo "github.com/midavuha1/go-course-lab/internal/repository/trip"
	createuc "github.com/midavuha1/go-course-lab/internal/usecase/trip/create"
	finishuc "github.com/midavuha1/go-course-lab/internal/usecase/trip/finish"
	getuc "github.com/midavuha1/go-course-lab/internal/usecase/trip/get"
)

type Container struct {
	Logger  *slog.Logger
	Server  *http.Server
	closers []func(context.Context) error
}

func New(ctx context.Context, cfg config.Config) (_ *Container, err error) {
	logger, err := telemetry.NewLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	c := &Container{Logger: logger}
	defer func() {
		if err != nil {
			_ = c.Close(ctx)
		}
	}()

	meterProvider := metricnoop.NewMeterProvider()

	pool, err := postgres.NewPool(ctx, cfg, nil)
	if err != nil {
		return nil, fmt.Errorf("init database pool: %w", err)
	}
	c.closers = append(c.closers, func(context.Context) error {
		pool.Close()
		return nil
	})

	txManager := transaction.New(pool)
	tripRepo := triprepo.New(txManager, cfg.DatabaseQueryTimeout)

	tripMetrics, err := telemetry.NewTripMetrics(meterProvider, tripRepo)
	if err != nil {
		return nil, err
	}

	createUC := createuc.New(tripRepo, txManager, tripMetrics, logger)
	getUC := getuc.New(tripRepo)
	finishUC := finishuc.New(tripRepo, txManager, tripMetrics, logger)

	handler := router.New(router.Handlers{
		Create: createhttp.New(createUC, logger),
		Get:    gethttp.New(getUC, logger),
		Finish: finishhttp.New(finishUC, logger),
		Health: health.New(pool, cfg.DatabaseQueryTimeout, logger),
	}, logger)

	c.Server = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return c, nil
}

func (c *Container) Close(ctx context.Context) error {
	errs := make([]error, 0, len(c.closers))
	for i := len(c.closers) - 1; i >= 0; i-- {
		if err := c.closers[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
