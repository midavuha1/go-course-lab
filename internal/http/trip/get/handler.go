package get

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/midavuha1/go-course-lab/internal/http/domainerror"
	"github.com/midavuha1/go-course-lab/internal/http/respond"
)

type Handler struct {
	uc     usecase
	logger *slog.Logger
}

func New(uc usecase, logger *slog.Logger) *Handler {
	return &Handler{uc: uc, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	t, err := h.uc.Get(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}
	respond.JSON(w, http.StatusOK, toAPI(t))
}
