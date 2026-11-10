package model

import (
	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
)

type CreateTripInput struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	Start    trip.Point
	End      trip.Point
	Price    int64
}
