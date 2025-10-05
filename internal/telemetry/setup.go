package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	stdoutTracerFactory = func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	stdoutMeterFactory = func() (sdkmetric.Exporter, error) {
		return stdoutmetric.New()
	}
	otlpGRPCFactory = func(ctx context.Context) (sdktrace.SpanExporter, error) {
		return otlptrace.New(ctx, otlptracegrpc.NewClient())
	}
	otlpHTTPFactory = func(ctx context.Context) (sdktrace.SpanExporter, error) {
		return otlptrace.New(ctx, otlptracehttp.NewClient())
	}
)

const ShutdownTimeout = 5 * time.Second

// InitProvider configures global OpenTelemetry providers based on environment variables.
func InitProvider(ctx context.Context) (func(context.Context) error, error) {
	exporter := os.Getenv("CHAINCTL_OTEL_EXPORTER")
	switch exporter {
	case "", "none":
		return func(context.Context) error { return nil }, nil
	case "stdout":
		tracer, err := stdoutTracerFactory()
		if err != nil {
			return nil, err
		}
		meter, err := stdoutMeterFactory()
		if err != nil {
			return nil, err
		}
		return installProvider(ctx, tracer, meter)
	case "otlp-grpc":
		tracer, err := otlpGRPCFactory(ctx)
		if err != nil {
			return nil, err
		}
		return installProvider(ctx, tracer, nil)
	case "otlp-http":
		tracer, err := otlpHTTPFactory(ctx)
		if err != nil {
			return nil, err
		}
		return installProvider(ctx, tracer, nil)
	default:
		return func(context.Context) error { return nil }, nil
	}
}

func installProvider(ctx context.Context, tracer sdktrace.SpanExporter, meter sdkmetric.Exporter) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("chainctl"),
			semconv.ServiceInstanceIDKey.String(hashInstanceID()),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tracer),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	var mp *sdkmetric.MeterProvider
	if meter != nil {
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(meter)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
	}

	return func(ctx context.Context) error {
		if mp != nil {
			if err := mp.Shutdown(ctx); err != nil {
				return err
			}
		}
		return tp.Shutdown(ctx)
	}, nil
}

func hashInstanceID() string {
	input := os.Getenv("CHAINCTL_CLUSTER_ID")
	if input == "" {
		if host, err := os.Hostname(); err == nil {
			input = host
		}
	}
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}
