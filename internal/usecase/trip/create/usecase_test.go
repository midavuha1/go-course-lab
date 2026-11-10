package create_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/model"
	"github.com/midavuha1/go-course-lab/internal/usecase/trip/create"
)

func validInput() model.CreateTripInput {
	return model.CreateTripInput{
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		Start:    trip.Point{Latitude: 59.9, Longitude: 30.3},
		End:      trip.Point{Latitude: 59.93, Longitude: 30.36},
		Price:    1450,
	}
}

func runInTx(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
}

func TestUsecase_Create(t *testing.T) {
	errDB := errors.New("connection reset by peer")

	tests := []struct {
		name    string
		setup   func(repo *MocktripRepository, tx *MocktxManager, metrics *MocktripMetrics)
		wantErr error
	}{
		{
			name: "success: trip saved, history written, metric counted",
			setup: func(repo *MocktripRepository, tx *MocktxManager, metrics *MocktripMetrics) {
				runInTx(tx)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().AddStatusHistory(gomock.Any(), gomock.Any(), nil, trip.StatusActive, trip.ReasonCreated).Return(nil)
				metrics.EXPECT().TripCreated(gomock.Any())
			},
		},
		{
			name: "driver busy: no history, no metric",
			setup: func(repo *MocktripRepository, tx *MocktxManager, _ *MocktripMetrics) {
				runInTx(tx)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(trip.ErrDriverBusy)
			},
			wantErr: trip.ErrDriverBusy,
		},
		{
			name: "repository failure is returned wrapped",
			setup: func(repo *MocktripRepository, tx *MocktxManager, _ *MocktripMetrics) {
				runInTx(tx)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fmt.Errorf("insert trip: %w", errDB))
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

			uc := create.New(repo, tx, metrics, slog.New(slog.DiscardHandler))
			in := validInput()

			got, err := uc.Create(context.Background(), in)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.ID == uuid.Nil || got.Status != trip.StatusActive || got.DriverID != in.DriverID {
				t.Fatalf("unexpected trip: %+v", got)
			}
		})
	}
}

func TestUsecase_Create_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := create.New(NewMocktripRepository(ctrl), NewMocktxManager(ctrl), NewMocktripMetrics(ctrl),
		slog.New(slog.DiscardHandler))

	in := validInput()
	in.Price = -1

	_, err := uc.Create(context.Background(), in)

	if verr, ok := errors.AsType[*trip.ValidationError](err); !ok || verr.Field != "price" {
		t.Fatalf("want ValidationError on price, got %v", err)
	}
}
