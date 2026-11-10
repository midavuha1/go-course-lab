package create

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/http/domainerror"
	"github.com/midavuha1/go-course-lab/internal/http/respond"
)

const maxBodyBytes = 1 << 20

type Handler struct {
	uc     usecase
	logger *slog.Logger
}

func New(uc usecase, logger *slog.Logger) *Handler {
	return &Handler{uc: uc, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		domainerror.Write(w, r, h.logger, &trip.ValidationError{Field: fieldBody, Reason: "cannot be read"})
		return
	}

	body, err := decode(raw)
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	created, err := h.uc.Create(r.Context(), toInput(body))
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+created.ID.String())
	respond.JSON(w, http.StatusCreated, toAPI(created))
}
