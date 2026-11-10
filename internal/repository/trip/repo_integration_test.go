//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/midavuha1/go-course-lab/internal/config"
	domain "github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/infra/postgres"
	"github.com/midavuha1/go-course-lab/internal/infra/postgres/transaction"
	triprepo "github.com/midavuha1/go-course-lab/internal/repository/trip"
)

type env struct {
	pool *pgxpool.Pool
	tx   *transaction.Manager
	repo *triprepo.Repository
}

func setup(t *testing.T) env {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is empty: run via make test-integration")
	}
	cfg := config.Config{
		DatabaseURL:             url,
		DatabaseMaxConns:        4,
		DatabaseMaxConnLifetime: time.Minute,
		DatabaseConnectTimeout:  5 * time.Second,
	}
	pool, err := postgres.NewPool(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	tx := transaction.New(pool)
	return env{pool: pool, tx: tx, repo: triprepo.New(tx, 3*time.Second)}
}

func newTrip(t *testing.T, e env) domain.Trip {
	t.Helper()
	tr, err := domain.New(uuid.New(), uuid.New(), uuid.New(),
		domain.Point{Latitude: 59.9, Longitude: 30.3}, domain.Point{Latitude: 59.93, Longitude: 30.36},
		1450, time.Now().UTC())
	if err != nil {
		t.Fatalf("domain.New: %v", err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", tr.ID)
	})
	return tr
}

func TestRepository_CreateAndGet(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	tr := newTrip(t, e)

	if err := e.repo.Create(ctx, tr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := e.repo.GetByID(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.DriverID != tr.DriverID || got.Status != domain.StatusActive || got.Price != tr.Price {
		t.Fatalf("got %+v, want %+v", got, tr)
	}
}

func TestRepository_DriverBusy(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	first := newTrip(t, e)
	second := newTrip(t, e)
	second.DriverID = first.DriverID

	if err := e.repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := e.repo.Create(ctx, second); !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("second Create err = %v, want ErrDriverBusy", err)
	}
}

func TestRepository_GetNotFound(t *testing.T) {
	e := setup(t)
	if _, err := e.repo.GetByID(context.Background(), uuid.New()); !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("err = %v, want ErrTripNotFound", err)
	}
}

func TestTransaction_RollbackBothInserts(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	tr := newTrip(t, e)

	err := e.tx.Do(ctx, func(ctx context.Context) error {
		if err := e.repo.Create(ctx, tr); err != nil {
			return err
		}
		return e.repo.AddStatusHistory(ctx, tr.ID, nil, domain.Status("bogus"), "must fail")
	})
	if err == nil {
		t.Fatal("want error from the second insert")
	}

	if _, err := e.repo.GetByID(ctx, tr.ID); !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("trip must be rolled back, GetByID err = %v", err)
	}
	var n int
	if err := e.pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id = $1", tr.ID).Scan(&n); err != nil {
		t.Fatalf("count history: %v", err)
	}
	if n != 0 {
		t.Fatalf("history rows = %d, want 0", n)
	}
}
