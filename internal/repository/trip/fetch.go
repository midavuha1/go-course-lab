package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/midavuha1/go-course-lab/internal/domain/trip"
)

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return r.get(ctx, id, false)
}

func (r *Repository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return r.get(ctx, id, true)
}

func (r *Repository) get(ctx context.Context, id uuid.UUID, forUpdate bool) (domain.Trip, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	builder := r.psql.Select(columns...).From(tableTrips).Where(sq.Eq{"id": id})
	if forUpdate {
		builder = builder.Suffix("FOR UPDATE")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build select trip: %w", err)
	}

	t, err := scanTrip(r.db.Executor(ctx).QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Trip{}, domain.ErrTripNotFound
		}
		return domain.Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
}
