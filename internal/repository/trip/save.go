package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	domain "github.com/midavuha1/go-course-lab/internal/domain/trip"
)

func (r *Repository) Create(ctx context.Context, t domain.Trip) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	query, args, err := r.psql.Insert(tableTrips).
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude",
			"end_latitude", "end_longitude", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID, t.Start.Latitude, t.Start.Longitude,
			t.End.Latitude, t.End.Longitude, t.Price, string(t.Status), t.StartedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip: %w", err)
	}

	if _, err := r.db.Executor(ctx).Exec(ctx, query, args...); err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok &&
			pgErr.Code == pgCodeUniqueViolation && pgErr.ConstraintName == constraintDriverActive {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, t domain.Trip) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	query, args, err := r.psql.Update(tableTrips).
		Set("status", string(t.Status)).
		Set("finished_at", t.FinishedAt).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": t.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip: %w", err)
	}

	tag, err := r.db.Executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTripNotFound
	}
	return nil
}

func (r *Repository) AddStatusHistory(
	ctx context.Context, tripID uuid.UUID, from *domain.Status, to domain.Status, reason string,
) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	query, args, err := r.psql.Insert(tableStatusHistory).
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, statusPtr(from), string(to), reason).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert status history: %w", err)
	}

	if _, err := r.db.Executor(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}
	return nil
}
