package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"validation", &trip.ValidationError{Field: "price", Reason: "is required"}, http.StatusBadRequest, "invalid_request"},
		{"driver busy, wrapped by usecase", fmt.Errorf("create trip: %w", trip.ErrDriverBusy), http.StatusConflict, "driver_busy"},
		{"not found", trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"completed", trip.ErrTripCompleted, http.StatusConflict, "trip_completed"},
		{"unknown error", errors.New("pgx: password authentication failed for user tripgo"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := domainerror.Problem(tt.err)

			if status != tt.wantStatus || body.Code != tt.wantCode || int(body.Status) != tt.wantStatus {
				t.Fatalf("got %d %q, want %d %q", status, body.Code, tt.wantStatus, tt.wantCode)
			}
			if status == http.StatusInternalServerError && body.Detail != nil && strings.Contains(*body.Detail, "password") {
				t.Fatalf("internal details leaked: %q", *body.Detail)
			}
		})
	}
}
