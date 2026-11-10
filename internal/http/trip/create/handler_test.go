package create_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/http/trip/create"
	"github.com/midavuha1/go-course-lab/internal/model"
)

const (
	validBody = `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",` +
		`"start_point":{"latitude":59.9,"longitude":30.3},"end_point":{"latitude":59.93,"longitude":30.36},"price":1450}`
	codeInvalidRequest = "invalid_request"
)

func serve(h *create.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	return rec
}

func TestHandle_Created(t *testing.T) {
	uc := NewMockusecase(gomock.NewController(t))
	uc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, in model.CreateTripInput) (trip.Trip, error) {
			return trip.Trip{
				ID: uuid.New(), UserID: in.UserID, DriverID: in.DriverID, Start: in.Start, End: in.End,
				Price: in.Price, Status: trip.StatusActive, StartedAt: time.Now(),
			}, nil
		},
	)

	rec := serve(create.New(uc, slog.New(slog.DiscardHandler)), validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	if !strings.HasPrefix(rec.Header().Get("Location"), "/api/v1/trips/") {
		t.Fatalf("Location = %q", rec.Header().Get("Location"))
	}
}

func TestHandle_Errors(t *testing.T) {
	noPrice := strings.Replace(validBody, `,"price":1450`, "", 1)
	nullStart := strings.Replace(validBody, `"start_point":{"latitude":59.9,"longitude":30.3}`, `"start_point":null`, 1)

	tests := []struct {
		name       string
		body       string
		ucErr      error // nil — usecase не должен вызываться
		wantStatus int
		wantCode   string
	}{
		{"missing price", noPrice, nil, http.StatusBadRequest, codeInvalidRequest},
		{"null start_point", nullStart, nil, http.StatusBadRequest, codeInvalidRequest},
		{"not a JSON", "{", nil, http.StatusBadRequest, codeInvalidRequest},
		{"driver busy", validBody, fmt.Errorf("create trip: %w", trip.ErrDriverBusy), http.StatusConflict, "driver_busy"},
		{"repository failure", validBody, errors.New("pgx: connection refused 10.0.0.5:5432"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMockusecase(gomock.NewController(t))
			if tt.ucErr != nil {
				uc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(trip.Trip{}, tt.ucErr)
			}

			rec := serve(create.New(uc, slog.New(slog.DiscardHandler)), tt.body)

			var p api.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("body is not a problem: %v", err)
			}
			if rec.Code != tt.wantStatus || p.Code != tt.wantCode {
				t.Fatalf("got %d %q, want %d %q", rec.Code, p.Code, tt.wantStatus, tt.wantCode)
			}
			if strings.Contains(rec.Body.String(), "pgx") {
				t.Fatal("internal error details leaked to client")
			}
		})
	}
}
