package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
)

func TestTrip_Finish(t *testing.T) {
	at := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		status     trip.Status
		wantErr    error
		wantStatus trip.Status
	}{
		{name: "active trip finishes", status: trip.StatusActive, wantErr: nil, wantStatus: trip.StatusCompleted},
		{name: "completed trip cannot finish again", status: trip.StatusCompleted, wantErr: trip.ErrTripCompleted, wantStatus: trip.StatusCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := trip.Trip{Status: tt.status}

			err := tr.Finish(at)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tr.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", tr.Status, tt.wantStatus)
			}
			if tt.wantErr == nil && (tr.FinishedAt == nil || !tr.FinishedAt.Equal(at)) {
				t.Fatalf("FinishedAt = %v, want %v", tr.FinishedAt, at)
			}
			if tt.wantErr != nil && tr.FinishedAt != nil {
				t.Fatal("FinishedAt must stay nil when finish is rejected")
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		userID, driverID uuid.UUID
		start, end       trip.Point
		price            int64
	}
	ok := args{
		userID:   uuid.New(),
		driverID: uuid.New(),
		start:    trip.Point{Latitude: 59.9, Longitude: 30.3},
		end:      trip.Point{Latitude: 59.93, Longitude: 30.36},
		price:    1450,
	}

	tests := []struct {
		name      string
		mutate    func(a *args) // портит одно поле валидных данных
		wantField string        // "" — ошибки быть не должно
	}{
		{name: "valid", mutate: func(*args) {}, wantField: ""},
		{name: "no user", mutate: func(a *args) { a.userID = uuid.Nil }, wantField: "user_id"},
		{name: "no driver", mutate: func(a *args) { a.driverID = uuid.Nil }, wantField: "driver_id"},
		{name: "latitude out of range", mutate: func(a *args) { a.start.Latitude = 91 }, wantField: "start_point.latitude"},
		{name: "longitude out of range", mutate: func(a *args) { a.end.Longitude = -181 }, wantField: "end_point.longitude"},
		{name: "negative price", mutate: func(a *args) { a.price = -1 }, wantField: "price"},
	}

	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := ok
			tt.mutate(&a)

			got, err := trip.New(uuid.New(), a.userID, a.driverID, a.start, a.end, a.price, now)

			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Status != trip.StatusActive || !got.StartedAt.Equal(now) {
					t.Fatalf("unexpected trip: %+v", got)
				}
				return
			}
			verr, isValidation := errors.AsType[*trip.ValidationError](err)
			if !isValidation || verr.Field != tt.wantField {
				t.Fatalf("err = %v, want ValidationError on %q", err, tt.wantField)
			}
		})
	}
}
