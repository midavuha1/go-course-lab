package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

const unknownRoute = "unknown"

func RoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return unknownRoute
	}
	if p := rctx.RoutePattern(); p != "" {
		return p
	}
	if rctx.Routes != nil {
		tctx := chi.NewRouteContext()
		if rctx.Routes.Match(tctx, r.Method, r.URL.Path) {
			return tctx.RoutePattern()
		}
	}
	return unknownRoute
}

func statusOf(ww chimw.WrapResponseWriter) int {
	if s := ww.Status(); s != 0 {
		return s
	}
	return http.StatusOK
}
