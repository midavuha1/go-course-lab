package trip

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	domain "github.com/midavuha1/go-course-lab/internal/domain/trip"
)

func (r *Repository) CountActive(ctx context.Context) (int64, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	query, args, err := r.psql.Select("count(*)").
		From(tableTrips).
		Where(sq.Eq{"status": string(domain.StatusActive)}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count active trips: %w", err)
	}

	var n int64
	if err := r.db.Executor(ctx).QueryRow(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count active trips: %w", err)
	}
	return n, nil
}
