package create

import (
	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/model"
)

func toInput(b api.TripData) model.CreateTripInput {
	return model.CreateTripInput{
		UserID:   b.UserId,
		DriverID: b.DriverId,
		Start:    trip.Point{Latitude: b.StartPoint.Latitude, Longitude: b.StartPoint.Longitude},
		End:      trip.Point{Latitude: b.EndPoint.Latitude, Longitude: b.EndPoint.Longitude},
		Price:    b.Price,
	}
}

func toAPI(t trip.Trip) api.Trip {
	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.Start.Latitude, Longitude: t.Start.Longitude},
		EndPoint:   api.Coordinates{Latitude: t.End.Latitude, Longitude: t.End.Longitude},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
