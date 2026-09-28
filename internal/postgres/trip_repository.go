package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/midavuha1/go-course-lab/internal/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var tripColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude", "end_latitude", "end_longitude",
	"price", "status", "started_at", "finished_at",
}

func scanTrip(row pgx.Row) (model.Trip, error) {
	var t model.Trip
	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.Start.Latitude, &t.Start.Longitude, &t.End.Latitude, &t.End.Longitude,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
	)
	return t, err
}

type TripRepository struct {
	tx           *TxManager
	queryTimeout time.Duration
}

func NewTripRepository(tx *TxManager, queryTimeout time.Duration) *TripRepository {
	return &TripRepository{tx: tx, queryTimeout: queryTimeout}
}

func (r *TripRepository) AddStatusHistory(ctx context.Context, tripID uuid.UUID, from *string, to, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	sql, args, err := psql.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, from, to, reason).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert status history: %w", err)
	}

	if _, err := r.tx.Executor(ctx).Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}
	return nil
}

func (r *TripRepository) Create(ctx context.Context, t model.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	sql, args, err := psql.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID, t.Start.Latitude, t.Start.Longitude, t.End.Latitude, t.End.Longitude, t.Price, t.Status, t.StartedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip: %w", err)
	}

	_, err = r.tx.Executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_driver_active_unique_idx" {
			return model.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	sql, args, err := psql.Select(tripColumns...).
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build select trip: %w", err)
	}

	row := r.tx.Executor(ctx).QueryRow(ctx, sql, args...)
	trip, err := scanTrip(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Trip{}, model.ErrTripNotFound
		}
		return model.Trip{}, fmt.Errorf("scan trip: %w", err)
	}
	return trip, nil
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID, now time.Time) (model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	sql, args, err := psql.Update("trips").
		Set("status", "completed").
		Set("finished_at", now).
		Set("updated_at", now).
		Where(sq.Eq{"id": id, "status": "active"}).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build update trip: %w", err)
	}

	row := r.tx.Executor(ctx).QueryRow(ctx, sql, args...)
	trip, err := scanTrip(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, getErr := r.GetByID(ctx, id)
			if errors.Is(getErr, model.ErrTripNotFound) {
				return model.Trip{}, model.ErrTripNotFound
			}
			if getErr == nil {
				return model.Trip{}, model.ErrTripCompleted
			}
			return model.Trip{}, fmt.Errorf("check trip status after no rows: %w", getErr)
		}
		return model.Trip{}, fmt.Errorf("scan finished trip: %w", err)
	}
	return trip, nil
}
