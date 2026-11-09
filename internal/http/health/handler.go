package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/http/respond"
)

type Handler struct {
	db      pinger
	timeout time.Duration
	logger  *slog.Logger
}

func New(db pinger, timeout time.Duration, logger *slog.Logger) *Handler {
	return &Handler{db: db, timeout: timeout, logger: logger}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.logger.WarnContext(ctx, "readiness check failed", slog.String("error", err.Error()))
		respond.JSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	respond.JSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
