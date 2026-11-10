package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/exaring/otelpgx"
	"go.opentelemetry.io/otel"

	"github.com/midavuha1/go-course-lab/internal/config"
	"github.com/midavuha1/go-course-lab/internal/http/health"
	"github.com/midavuha1/go-course-lab/internal/http/middleware"
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

	providers, err := telemetry.Setup(ctx)
	if err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	c.closers = append(c.closers, providers.Shutdown)

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Warn("telemetry export failed", slog.String("error", err.Error()))
	}))

	dbTracer := otelpgx.NewTracer(
		otelpgx.WithTracerProvider(providers.Tracer),
		otelpgx.WithMeterProvider(providers.Meter),
		otelpgx.WithMetricOperationName(otelpgx.SQLOperationName),
	)

	pool, err := postgres.NewPool(ctx, cfg, dbTracer)
	if err != nil {
		return nil, fmt.Errorf("init database pool: %w", err)
	}
	c.closers = append(c.closers, func(context.Context) error {
		pool.Close()
		return nil
	})

	txManager := transaction.New(pool)
	tripRepo := triprepo.New(txManager, cfg.DatabaseQueryTimeout)

	tripMetrics, err := telemetry.NewTripMetrics(providers.Meter, tripRepo)
	if err != nil {
		return nil, err
	}

	httpMetrics, err := middleware.NewHTTPMetrics(providers.Meter)
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
	}, logger,
		middleware.Recover(logger),
		middleware.Tracing(providers.Tracer, providers.Propagator),
		middleware.Logging(logger),
		httpMetrics.Middleware(),
	)

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
