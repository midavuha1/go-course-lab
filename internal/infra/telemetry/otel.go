package telemetry

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Providers struct {
	Tracer     *sdktrace.TracerProvider
	Meter      *sdkmetric.MeterProvider
	Propagator propagation.TextMapPropagator
}

func Setup(ctx context.Context) (*Providers, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", ServiceName),
			attribute.String("service.version", buildVersion()),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	spanExporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("create span exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(spanExporter), sdktrace.WithResource(res))

	reader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("create metric reader: %w", err), tp.Shutdown(ctx))
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithResource(res))

	return &Providers{
		Tracer:     tp,
		Meter:      mp,
		Propagator: propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	}, nil
}

func (p *Providers) Shutdown(ctx context.Context) error {
	return errors.Join(p.Tracer.Shutdown(ctx), p.Meter.Shutdown(ctx))
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return s.Value[:7]
		}
	}
	return info.Main.Version
}
