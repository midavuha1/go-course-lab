package middleware

import (
	"fmt"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HTTPMetrics struct {
	duration metric.Float64Histogram
	active   metric.Int64UpDownCounter
}

func NewHTTPMetrics(mp metric.MeterProvider) (*HTTPMetrics, error) {
	meter := mp.Meter("trip-service/http")

	duration, err := meter.Float64Histogram("http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Длительность обработки HTTP-запроса"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create duration histogram: %w", err)
	}

	active, err := meter.Int64UpDownCounter("http.server.active_requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Запросы в обработке прямо сейчас"),
	)
	if err != nil {
		return nil, fmt.Errorf("create active requests counter: %w", err)
	}

	return &HTTPMetrics{duration: duration, active: active}, nil
}

func (m *HTTPMetrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			base := metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", RoutePattern(r)),
			)

			m.active.Add(ctx, 1, base)
			defer m.active.Add(ctx, -1, base)

			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			m.duration.Record(ctx, time.Since(start).Seconds(),
				metric.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", RoutePattern(r)),
					attribute.Int("http.response.status_code", statusOf(ww)),
				),
			)
		})
	}
}
