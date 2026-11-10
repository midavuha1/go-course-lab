package middleware

import (
	"net/http"
	"strconv"

	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tp trace.TracerProvider, propagator propagation.TextMapPropagator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		named := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route := RoutePattern(r)
			span := trace.SpanFromContext(r.Context())
			span.SetName(r.Method + " " + route)
			span.SetAttributes(attribute.String("http.route", route))
			if status := statusOf(ww); status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
				span.SetAttributes(attribute.String("error.type", strconv.Itoa(status)))
			}
		})
		return otelhttp.NewHandler(named, "http.request",
			otelhttp.WithTracerProvider(tp),
			otelhttp.WithPropagators(propagator),
			otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
		)
	}
}
