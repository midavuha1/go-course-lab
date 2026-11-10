package finish_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/usecase/trip/finish"
)

func ptr[T any](v T) *T { return &v }

func runInTx(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
}

func TestUsecase_Finish(t *testing.T) {
	id := uuid.New()
	errDB := errors.New("connection reset by peer")

	tests := []struct {
		name    string
		setup   func(repo *MocktripRepository, tx *MocktxManager, metrics *MocktripMetrics)
		wantErr error
	}{
		{
			name: "active trip finishes",
			setup: func(repo *MocktripRepository, tx *MocktxManager, metrics *MocktripMetrics) {
				runInTx(tx)
				repo.EXPECT().GetByIDForUpdate(gomock.Any(), id).Return(trip.Trip{ID: id, Status: trip.StatusActive}, nil)
				repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().AddStatusHistory(gomock.Any(), id, ptr(trip.StatusActive), trip.StatusCompleted, trip.ReasonFinished).Return(nil)
				metrics.EXPECT().TripCompleted(gomock.Any())
			},
		},
		{
			name: "completed trip: no update, no history, no metric",
			setup: func(repo *MocktripRepository, tx *MocktxManager, _ *MocktripMetrics) {
				runInTx(tx)
				repo.EXPECT().GetByIDForUpdate(gomock.Any(), id).Return(trip.Trip{ID: id, Status: trip.StatusCompleted}, nil)
			},
			wantErr: trip.ErrTripCompleted,
		},
		{
			name: "trip not found",
			setup: func(repo *MocktripRepository, tx *MocktxManager, _ *MocktripMetrics) {
				runInTx(tx)
				repo.EXPECT().GetByIDForUpdate(gomock.Any(), id).Return(trip.Trip{}, trip.ErrTripNotFound)
			},
			wantErr: trip.ErrTripNotFound,
		},
		{
			name: "update failure is returned wrapped, history not written",
			setup: func(repo *MocktripRepository, tx *MocktxManager, _ *MocktripMetrics) {
				runInTx(tx)
				repo.EXPECT().GetByIDForUpdate(gomock.Any(), id).Return(trip.Trip{ID: id, Status: trip.StatusActive}, nil)
				repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fmt.Errorf("update trip: %w", errDB))
			},
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			metrics := NewMocktripMetrics(ctrl)
			tt.setup(repo, tx, metrics)

			uc := finish.New(repo, tx, metrics, slog.New(slog.DiscardHandler))

			got, err := uc.Finish(context.Background(), id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Status != trip.StatusCompleted || got.FinishedAt == nil {
				t.Fatalf("unexpected trip: %+v", got)
			}
		})
	}
}
