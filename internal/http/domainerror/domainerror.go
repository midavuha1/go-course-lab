package domainerror

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/domain/trip"
)

const problemBaseURL = "https://tripgo.example/problems/"

func Problem(err error) (int, api.Problem) {
	if verr, ok := errors.AsType[*trip.ValidationError](err); ok {
		return build(http.StatusBadRequest, "invalid-request", "Invalid request", "invalid_request", verr.Error())
	}

	switch {
	case errors.Is(err, trip.ErrDriverBusy):
		return build(http.StatusConflict, "driver-busy", "Driver busy", "driver_busy",
			"Driver already has an active trip")
	case errors.Is(err, trip.ErrTripNotFound):
		return build(http.StatusNotFound, "trip-not-found", "Trip not found", "trip_not_found",
			"Trip with this id does not exist")
	case errors.Is(err, trip.ErrTripCompleted):
		return build(http.StatusConflict, "trip-completed", "Trip completed", "trip_completed",
			"Trip is already completed")
	default:
		return build(http.StatusInternalServerError, "internal-error", "Internal error", "internal_error",
			"Internal server error")
	}
}

func Write(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status, body := Problem(err)
	if status >= http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "request failed", slog.String("error", err.Error()))
	}
	respond(w, r, status, body)
}

func Respond(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	respond(w, r, status, body)
}

func respond(w http.ResponseWriter, r *http.Request, status int, body api.Problem) {
	instance := r.URL.Path
	body.Instance = &instance

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func build(status int32, slug, title, code, detail string) (int, api.Problem) {
	return int(status), api.Problem{
		Type:   problemBaseURL + slug,
		Title:  title,
		Status: status,
		Code:   code,
		Detail: &detail,
	}
}
