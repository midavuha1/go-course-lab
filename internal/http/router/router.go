package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/domain/trip"
	"github.com/midavuha1/go-course-lab/internal/http/domainerror"
	"github.com/midavuha1/go-course-lab/internal/http/health"
	"github.com/midavuha1/go-course-lab/internal/http/trip/create"
	"github.com/midavuha1/go-course-lab/internal/http/trip/finish"
	"github.com/midavuha1/go-course-lab/internal/http/trip/get"
)

type Handlers struct {
	Create *create.Handler
	Get    *get.Handler
	Finish *finish.Handler
	Health *health.Handler
}

type server struct {
	h Handlers
}

var _ api.ServerInterface = (*server)(nil)

func (s *server) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	s.h.Create.Handle(w, r)
}

func (s *server) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.h.Get.Handle(w, r, tripID)
}

func (s *server) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.h.Finish.Handle(w, r, tripID)
}

func (s *server) Health(w http.ResponseWriter, r *http.Request) { s.h.Health.Health(w, r) }

func (s *server) Ready(w http.ResponseWriter, r *http.Request) { s.h.Health.Ready(w, r) }

func New(h Handlers, logger *slog.Logger, middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middlewares...)

	api.HandlerWithOptions(&server{h: h}, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
			domainerror.Write(w, req, logger, &trip.ValidationError{Field: "path", Reason: err.Error()})
		},
	})
	return r
}
