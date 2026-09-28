package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/model"
	"github.com/midavuha1/go-course-lab/internal/service"
)

// Pinger: всё, что хендлеру нужно знать про базу.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	trips       *service.TripService
	db          Pinger
	pingTimeout time.Duration
}

func New(trips *service.TripService, db Pinger, pingTimeout time.Duration) *Handler {
	return &Handler{trips: trips, db: db, pingTimeout: pingTimeout}
}

// Компилятор проверит, что реализованы все пять методов из контракта.
var _ api.ServerInterface = (*Handler)(nil)

// HandleBindError вызывается сгенерированным кодом, когда tripId не UUID и т.п.
func (h *Handler) HandleBindError(w http.ResponseWriter, r *http.Request, err error) {
	writeInvalidRequest(w, r, err.Error())
}

func toAPI(t model.Trip) api.Trip {
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

// Health: жив ли процесс. В базу НЕ ходит.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

// Ready: готов ли сервис принимать трафик, то есть доступна ли база.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.pingTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		slog.Warn("readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	t, err := h.trips.GetTrip(r.Context(), tripId)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPI(t))
}

// validate возвращает текст ошибки или пустую строку, если тело корректно.
func validate(b api.TripData) string {
	if b.UserId == uuid.Nil {
		return "user_id is required"
	}
	if b.DriverId == uuid.Nil {
		return "driver_id is required"
	}
	if b.StartPoint.Latitude < -90 || b.StartPoint.Latitude > 90 {
		return "start_point latitude must be between -90 and 90"
	}
	if b.StartPoint.Longitude < -180 || b.StartPoint.Longitude > 180 {
		return "start_point longitude must be between -180 and 180"
	}
	if b.EndPoint.Latitude < -90 || b.EndPoint.Latitude > 90 {
		return "end_point latitude must be between -90 and 90"
	}
	if b.EndPoint.Longitude < -180 || b.EndPoint.Longitude > 180 {
		return "end_point longitude must be between -180 and 180"
	}
	if b.Price < 0 {
		return "price must be non-negative"
	}
	return ""
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var body api.TripData
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeInvalidRequest(w, r, "invalid JSON body: "+err.Error())
		return
	}
	if msg := validate(body); msg != "" {
		writeInvalidRequest(w, r, msg)
		return
	}

	trip := model.Trip{
		UserID:   body.UserId,
		DriverID: body.DriverId,
		Start:    model.Point{Latitude: body.StartPoint.Latitude, Longitude: body.StartPoint.Longitude},
		End:      model.Point{Latitude: body.EndPoint.Latitude, Longitude: body.EndPoint.Longitude},
		Price:    body.Price,
	}

	created, err := h.trips.CreateTrip(r.Context(), trip)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+created.ID.String())
	writeJSON(w, http.StatusCreated, toAPI(created))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.trips.FinishTrip(r.Context(), tripId)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPI(trip))
}
