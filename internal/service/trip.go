package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/internal/model"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type TripRepository interface {
	Create(ctx context.Context, t model.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, now time.Time) (model.Trip, error)
	AddStatusHistory(ctx context.Context, tripID uuid.UUID, from *string, to, reason string) error
}

type TripService struct {
	tx   TxManager
	repo TripRepository
}

func NewTripService(tx TxManager, repo TripRepository) *TripService {
	return &TripService{tx: tx, repo: repo}
}

func (s *TripService) CreateTrip(ctx context.Context, t model.Trip) (model.Trip, error) {

	t.ID = uuid.New()
	t.Status = "active"
	t.StartedAt = time.Now().UTC()

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, t); err != nil {
			return err
		}

		if err := s.repo.AddStatusHistory(ctx, t.ID, nil, "active", "trip created"); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return model.Trip{}, err
	}

	return t, nil
}

func (s *TripService) GetTrip(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TripService) FinishTrip(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	var finishedTrip model.Trip
	now := time.Now().UTC()
	fromStatus := "active"

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		finishedTrip, err = s.repo.Finish(ctx, id, now)
		if err != nil {
			return err
		}

		if err := s.repo.AddStatusHistory(ctx, id, &fromStatus, "completed", "trip finished"); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return model.Trip{}, err
	}

	return finishedTrip, nil
}
