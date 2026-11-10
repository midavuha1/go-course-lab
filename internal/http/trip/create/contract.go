//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/model"
)

type usecase interface {
	Create(ctx context.Context, in model.CreateTripInput) (trip.Trip, error)
}
