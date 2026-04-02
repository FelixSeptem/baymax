package trace

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExporter struct {
	lastRequest ExportRequest
	calls       int
	err         error
}

func (f *fakeExporter) Export(_ context.Context, req ExportRequest) error {
	f.calls++
	f.lastRequest = req
	return f.err
}

func TestExportRuntimeDisabledReturnsDisabledWithoutExporterCall(t *testing.T) {
	exporter := &fakeExporter{}
	runtime := NewExportRuntime(exporter)

	got := runtime.Export(context.Background(), ExportConfig{
		Enabled:       false,
		SchemaVersion: OTelSemconvVersionV1,
	}, ExportPayload{
		SchemaVersion: OTelSemconvVersionV1,
	})

	if got.Status != TraceExportStatusDisabled || got.ReasonCode != TraceExportReasonNone {
		t.Fatalf("disabled export result mismatch: %#v", got)
	}
	if exporter.calls != 0 {
		t.Fatalf("disabled export should not call exporter, calls=%d", exporter.calls)
	}
}

func TestExportRuntimeSupportsLocalAndRemoteCollectorSmoke(t *testing.T) {
	exporter := &fakeExporter{}
	runtime := NewExportRuntime(exporter)
	cfg := ExportConfig{
		Enabled:       true,
		Protocol:      TraceExportProtocolHTTPProtobuf,
		SchemaVersion: OTelSemconvVersionV1,
		OnError:       TraceExportOnErrorDegradeAndRecord,
		ExportTimeout: 2 * time.Second,
		ResourceAttributes: map[string]string{
			"service.name": "baymax-runtime",
		},
	}
	spans := []SemanticSpan{
		{
			Domain: TraceDomainRun,
			Attributes: map[string]string{
				AttrRunID: "run-local",
			},
		},
	}

	localResult := runtime.Export(context.Background(), withEndpoint(cfg, "http://127.0.0.1:4318/v1/traces"), ExportPayload{
		SchemaVersion: OTelSemconvVersionV1,
		Spans:         spans,
	})
	if localResult.Status != TraceExportStatusSuccess || localResult.ReasonCode != TraceExportReasonNone {
		t.Fatalf("local collector smoke failed: %#v", localResult)
	}
	if exporter.lastRequest.Endpoint != "http://127.0.0.1:4318/v1/traces" {
		t.Fatalf("local endpoint mismatch: %#v", exporter.lastRequest)
	}

	remoteResult := runtime.Export(context.Background(), withEndpoint(cfg, "https://otel.example.com/v1/traces"), ExportPayload{
		SchemaVersion: OTelSemconvVersionV1,
		Spans:         spans,
	})
	if remoteResult.Status != TraceExportStatusSuccess || remoteResult.ReasonCode != TraceExportReasonNone {
		t.Fatalf("remote collector smoke failed: %#v", remoteResult)
	}
	if exporter.lastRequest.Endpoint != "https://otel.example.com/v1/traces" {
		t.Fatalf("remote endpoint mismatch: %#v", exporter.lastRequest)
	}
}

func TestExportRuntimeErrorClassificationAndOnErrorPolicy(t *testing.T) {
	exporter := &fakeExporter{}
	runtime := NewExportRuntime(exporter)
	cfg := ExportConfig{
		Enabled:       true,
		Endpoint:      "http://127.0.0.1:4318/v1/traces",
		Protocol:      TraceExportProtocolGRPC,
		SchemaVersion: OTelSemconvVersionV1,
		OnError:       TraceExportOnErrorDegradeAndRecord,
		ExportTimeout: 2 * time.Second,
	}

	exporter.err = errors.New("dial tcp 127.0.0.1:4318: connect: connection refused")
	got := runtime.Export(context.Background(), cfg, ExportPayload{SchemaVersion: OTelSemconvVersionV1})
	if got.Status != TraceExportStatusDegraded || got.ReasonCode != TraceExportReasonCollectorUnreachable {
		t.Fatalf("collector-unreachable classification mismatch: %#v", got)
	}

	exporter.err = context.DeadlineExceeded
	got = runtime.Export(context.Background(), withOnError(cfg, TraceExportOnErrorFailFast), ExportPayload{SchemaVersion: OTelSemconvVersionV1})
	if got.Status != TraceExportStatusFailed || got.ReasonCode != TraceExportReasonTimeout {
		t.Fatalf("timeout fail_fast classification mismatch: %#v", got)
	}

	exporter.err = errors.New("401 unauthorized")
	got = runtime.Export(context.Background(), cfg, ExportPayload{SchemaVersion: OTelSemconvVersionV1})
	if got.Status != TraceExportStatusDegraded || got.ReasonCode != TraceExportReasonAuthFailed {
		t.Fatalf("auth classification mismatch: %#v", got)
	}

	exporter.err = errors.New("unknown boom")
	first := runtime.Export(context.Background(), cfg, ExportPayload{SchemaVersion: OTelSemconvVersionV1})
	second := runtime.Export(context.Background(), cfg, ExportPayload{SchemaVersion: OTelSemconvVersionV1})
	if first.ReasonCode != TraceExportReasonUnknown || second.ReasonCode != TraceExportReasonUnknown {
		t.Fatalf("unknown classification mismatch: first=%#v second=%#v", first, second)
	}
}

func TestExportRuntimePreservesSchemaVersionWhenPayloadMissing(t *testing.T) {
	exporter := &fakeExporter{}
	runtime := NewExportRuntime(exporter)
	cfg := ExportConfig{
		Enabled:       true,
		Endpoint:      "http://127.0.0.1:4318/v1/traces",
		Protocol:      TraceExportProtocolGRPC,
		SchemaVersion: OTelSemconvVersionV1,
		OnError:       TraceExportOnErrorDegradeAndRecord,
		ExportTimeout: time.Second,
	}
	got := runtime.Export(context.Background(), cfg, ExportPayload{})
	if got.SchemaVersion != OTelSemconvVersionV1 {
		t.Fatalf("schema version fallback mismatch: %#v", got)
	}
	if exporter.lastRequest.SchemaVersion != OTelSemconvVersionV1 {
		t.Fatalf("export request schema version mismatch: %#v", exporter.lastRequest)
	}
}

func withEndpoint(cfg ExportConfig, endpoint string) ExportConfig {
	cfg.Endpoint = endpoint
	return cfg
}

func withOnError(cfg ExportConfig, policy string) ExportConfig {
	cfg.OnError = policy
	return cfg
}
