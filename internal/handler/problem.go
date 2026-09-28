package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, slug, title, code, detail string) {
	instance := r.URL.Path
	p := api.Problem{
		Type:     "https://tripgo.example/problems/" + slug,
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: &instance,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

func writeInvalidRequest(w http.ResponseWriter, r *http.Request, detail string) {
	writeProblem(w, r, http.StatusBadRequest, "invalid-request", "Invalid request", "invalid_request", detail)
}

// writeError переводит доменную ошибку в HTTP-ответ. Единственное место такого перевода.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrDriverBusy):
		writeProblem(w, r, http.StatusConflict, "driver-busy", "Driver busy", "driver_busy", "Driver already has an active trip")
	case errors.Is(err, model.ErrTripNotFound):
		writeProblem(w, r, http.StatusNotFound, "trip-not-found", "Trip not found", "trip_not_found", "Trip with this id does not exist")
	case errors.Is(err, model.ErrTripCompleted):
		writeProblem(w, r, http.StatusConflict, "trip-completed", "Trip completed", "trip_completed", "Trip is already completed")
	default:
		// Внутренние детали (текст ошибки БД, SQL) клиенту не отдаём, только в лог.
		slog.Error("internal error", "error", err, "path", r.URL.Path)
		writeProblem(w, r, http.StatusInternalServerError, "internal-error", "Internal error", "internal_error", "Internal server error")
	}
}
