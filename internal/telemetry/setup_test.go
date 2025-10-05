package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type stubSpanExporter struct{ shutdowns int }

func (stubSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

func (s *stubSpanExporter) Shutdown(context.Context) error {
	s.shutdowns++
	return nil
}

func TestInitProviderNone(t *testing.T) {
	t.Setenv("CHAINCTL_OTEL_EXPORTER", "none")
	shutdown, err := InitProvider(context.Background())
	if err != nil {
		t.Fatalf("init provider: %v", err)
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestInitProviderStdout(t *testing.T) {
	t.Setenv("CHAINCTL_OTEL_EXPORTER", "stdout")
	shutdown, err := InitProvider(context.Background())
	if err != nil {
		t.Fatalf("init provider: %v", err)
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown function")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestInitProviderUnknownFallsBack(t *testing.T) {
	t.Setenv("CHAINCTL_OTEL_EXPORTER", "invalid")
	shutdown, err := InitProvider(context.Background())
	if err != nil {
		t.Fatalf("init provider: %v", err)
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown function even for invalid exporter")
	}
}

func TestInstallProviderWithNilMeter(t *testing.T) {
	exporter := &stubSpanExporter{}
	shutdown, err := installProvider(context.Background(), exporter, nil)
	if err != nil {
		t.Fatalf("install provider: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if exporter.shutdowns == 0 {
		t.Fatalf("expected exporter shutdown to be invoked")
	}
}

func TestHashInstanceIDUsesEnv(t *testing.T) {
	t.Setenv("CHAINCTL_CLUSTER_ID", "my-cluster")
	hashed := hashInstanceID()
	want := sha256.Sum256([]byte("my-cluster"))
	if hashed != hex.EncodeToString(want[:]) {
		t.Fatalf("expected hash %s, got %s", hex.EncodeToString(want[:]), hashed)
	}
}

func TestHashInstanceIDFallsBackToHostname(t *testing.T) {
	t.Setenv("CHAINCTL_CLUSTER_ID", "")
	hashed := hashInstanceID()
	if len(hashed) != 64 {
		t.Fatalf("expected sha256 hex length, got %d", len(hashed))
	}
}

func TestInitProviderOTLPGRPC(t *testing.T) {
	original := otlpGRPCFactory
	t.Cleanup(func() { otlpGRPCFactory = original })

	exporter := &stubSpanExporter{}
	otlpGRPCFactory = func(context.Context) (sdktrace.SpanExporter, error) {
		return exporter, nil
	}

	t.Setenv("CHAINCTL_OTEL_EXPORTER", "otlp-grpc")
	shutdown, err := InitProvider(context.Background())
	if err != nil {
		t.Fatalf("init provider: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if exporter.shutdowns == 0 {
		t.Fatalf("expected exporter shutdown to be invoked")
	}
}

func TestInitProviderOTLPHTTP(t *testing.T) {
	original := otlpHTTPFactory
	t.Cleanup(func() { otlpHTTPFactory = original })

	exporter := &stubSpanExporter{}
	otlpHTTPFactory = func(context.Context) (sdktrace.SpanExporter, error) {
		return exporter, nil
	}

	t.Setenv("CHAINCTL_OTEL_EXPORTER", "otlp-http")
	shutdown, err := InitProvider(context.Background())
	if err != nil {
		t.Fatalf("init provider: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if exporter.shutdowns == 0 {
		t.Fatalf("expected exporter shutdown to be invoked")
	}
}
