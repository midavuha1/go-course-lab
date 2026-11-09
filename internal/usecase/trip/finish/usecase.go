package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
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

func (u *Usecase) Finish(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	var finished trip.Trip
	now := time.Now().UTC()

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		t, err := u.repo.GetByIDForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("load trip: %w", err)
		}

		from := t.Status
		if err := t.Finish(now); err != nil {
			return err
		}

		if err := u.repo.Update(ctx, t); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}
		if err := u.repo.AddStatusHistory(ctx, t.ID, &from, t.Status, trip.ReasonFinished); err != nil {
			return fmt.Errorf("save status history: %w", err)
		}

		finished = t
		return nil
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	u.metrics.TripCompleted(ctx)
	u.logger.InfoContext(ctx, "trip finished", slog.String("trip_id", finished.ID.String()))
	return finished, nil
}
