package create

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/model"
)

type Usecase struct {
	repo    tripRepository
	tx      txManager
	metrics tripMetrics
	logger  *slog.Logger
}

func New(repo tripRepository, tx txManager, metrics tripMetrics, logger *slog.Logger) *Usecase {
	return &Usecase{repo: repo, tx: tx, metrics: metrics, logger: logger}
}

func (u *Usecase) Create(ctx context.Context, in model.CreateTripInput) (trip.Trip, error) {
	t, err := trip.New(uuid.New(), in.UserID, in.DriverID, in.Start, in.End, in.Price, time.Now().UTC())
	if err != nil {
		return trip.Trip{}, fmt.Errorf("new trip: %w", err)
	}

	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.repo.Create(ctx, t); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}
		if err := u.repo.AddStatusHistory(ctx, t.ID, nil, trip.StatusActive, trip.ReasonCreated); err != nil {
			return fmt.Errorf("save status history: %w", err)
		}
		return nil
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	u.metrics.TripCreated(ctx)
	u.logger.InfoContext(ctx, "trip created",
		slog.String("trip_id", t.ID.String()),
		slog.String("driver_id", t.DriverID.String()),
	)
	return t, nil
}
