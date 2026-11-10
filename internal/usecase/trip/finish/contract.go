//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
)

type (
	tripRepository interface {
		GetByIDForUpdate(ctx context.Context, id uuid.UUID) (trip.Trip, error)
		Update(ctx context.Context, t trip.Trip) error
		AddStatusHistory(ctx context.Context, tripID uuid.UUID, from *trip.Status, to trip.Status, reason string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	tripMetrics interface {
		TripCompleted(ctx context.Context)
	}
)
