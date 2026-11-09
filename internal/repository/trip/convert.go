package trip

import (
	"github.com/jackc/pgx/v5"

	domain "github.com/midavuha1/go-course-lab/internal/domain/trip"
)

func scanTrip(row pgx.Row) (domain.Trip, error) {
	var (
		t      domain.Trip
		status string
	)
	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.Start.Latitude, &t.Start.Longitude, &t.End.Latitude, &t.End.Longitude,
		&t.Price, &status, &t.StartedAt, &t.FinishedAt,
	)
	if err != nil {
		return domain.Trip{}, err
	}
	t.Status = domain.Status(status)
	return t, nil
}

func statusPtr(s *domain.Status) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
