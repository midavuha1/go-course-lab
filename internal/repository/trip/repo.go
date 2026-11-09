package trip

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const (
	tableTrips             = "trips"
	tableStatusHistory     = "trip_status_history"
	constraintDriverActive = "trips_driver_active_unique_idx"
	pgCodeUniqueViolation  = "23505"
)

var columns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude", "end_latitude", "end_longitude",
	"price", "status", "started_at", "finished_at",
}

type Repository struct {
	db           executorProvider
	queryTimeout time.Duration
	psql         sq.StatementBuilderType
}

func New(db executorProvider, queryTimeout time.Duration) *Repository {
	return &Repository{
		db:           db,
		queryTimeout: queryTimeout,
		psql:         sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *Repository) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, r.queryTimeout)
}
