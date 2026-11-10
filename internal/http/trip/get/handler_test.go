package get_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/http/trip/get"
)

func TestHandle(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name       string
		ucTrip     trip.Trip
		ucErr      error
		wantStatus int
		wantCode   string
	}{
		{"found", trip.Trip{ID: id, Status: trip.StatusActive, StartedAt: time.Now()}, nil, http.StatusOK, ""},
		{"not found", trip.Trip{}, fmt.Errorf("get trip: %w", trip.ErrTripNotFound), http.StatusNotFound, "trip_not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMockusecase(gomock.NewController(t))
			uc.EXPECT().Get(gomock.Any(), id).Return(tt.ucTrip, tt.ucErr)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
			get.New(uc, slog.New(slog.DiscardHandler)).Handle(rec, req, id)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantCode != "" {
				var p api.Problem
				if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.Code != tt.wantCode {
					t.Fatalf("problem = %+v (%v), want code %q", p, err, tt.wantCode)
				}
				return
			}
			var got api.Trip
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Id != id {
				t.Fatalf("trip = %+v (%v), want id %s", got, err, id)
			}
		})
	}
}
