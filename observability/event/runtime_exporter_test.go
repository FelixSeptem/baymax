package event

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestDefaultRuntimeExporterResolverProfilesAndCanonicalError(t *testing.T) {
	resolver := newDefaultRuntimeExporterResolver(nil)
	tests := []struct {
		name    string
		cfg     runtimeconfig.RuntimeObservabilityExportConfig
		wantErr string
	}{
		{
			name: "none",
			cfg: runtimeconfig.RuntimeObservabilityExportConfig{
				Enabled: true,
				Profile: "NONE",
			},
		},
		{
			name: "otlp",
			cfg: runtimeconfig.RuntimeObservabilityExportConfig{
				Enabled:  true,
				Profile:  "OTLP",
				Endpoint: "https://otlp.example/v1/traces",
			},
		},
		{
			name: "langfuse",
			cfg: runtimeconfig.RuntimeObservabilityExportConfig{
				Enabled:  true,
				Profile:  "LANGFUSE",
				Endpoint: "https://langfuse.example",
			},
		},
		{
			name: "custom",
			cfg: runtimeconfig.RuntimeObservabilityExportConfig{
				Enabled:  true,
				Profile:  "CUSTOM",
				Endpoint: "custom://sink",
			},
		},
		{
			name: "invalid",
			cfg: runtimeconfig.RuntimeObservabilityExportConfig{
				Enabled: true,
				Profile: "jaeger",
			},
			wantErr: runtimeconfig.ReadinessCodeObservabilityExportProfileInvalid,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolver.Resolve(tc.cfg)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("Resolve returned error: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q but got nil", tc.wantErr)
				}
				canonical := canonicalizeRuntimeExportError(err, tc.cfg.Profile)
				if canonical.Code != tc.wantErr {
					t.Fatalf("canonical code=%q, want %q (err=%v)", canonical.Code, tc.wantErr, err)
				}
			}
		})
	}
}

func TestRuntimeRecorderExporterConsumesRedactedPayloadAfterRecordRun(t *testing.T) {
	mgr := newRuntimeRecorderTestManager(t, `
runtime:
  observability:
    export:
      enabled: true
      profile: custom
      endpoint: custom://sink
      queue_capacity: 32
      on_error: degrade_and_record
security:
  redaction:
    enabled: true
    strategy: keyword
    keywords: [secret]
`)
	exp := &fakeRuntimeExporter{}
	rec := NewRuntimeRecorder(mgr, WithRuntimeExporterFactory(runtimeconfig.RuntimeObservabilityExportProfileCustom, func(_ runtimeconfig.RuntimeObservabilityExportConfig) (RuntimeExporter, error) {
		return exp, nil
	}))
	t.Cleanup(rec.Close)

	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		Time:    time.Now(),
		RunID:   "run-a55-redact",
		Payload: map[string]any{
			"status":        "success",
			"latency_ms":    int64(10),
			"tool_calls":    1,
			"client_secret": "raw-secret",
		},
	})
	runs := mgr.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("run diagnostics len=%d, want 1", len(runs))
	}
	waitForRuntimeCondition(t, 2*time.Second, func() bool {
		return exp.ExportCallTotal() > 0
	})
	exported := exp.FirstEvent()
	if exported.Type != "run.finished" {
		t.Fatalf("exported type=%q, want run.finished", exported.Type)
	}
	if got := payloadString(exported.Payload, "client_secret"); got != "***" {
		t.Fatalf("exported payload should be redacted, got %q", got)
	}
}

func TestRuntimeRecorderExporterQueueOverflowDegradeAndRecord(t *testing.T) {
	mgr := newRuntimeRecorderTestManager(t, `
runtime:
  observability:
    export:
      enabled: true
      profile: custom
      endpoint: custom://sink
      queue_capacity: 1
      on_error: degrade_and_record
`)
	block := make(chan struct{})
	exp := &fakeRuntimeExporter{
		exportFn: func(_ []types.Event) error {
			<-block
			return nil
		},
	}
	rec := NewRuntimeRecorder(mgr, WithRuntimeExporterFactory(runtimeconfig.RuntimeObservabilityExportProfileCustom, func(_ runtimeconfig.RuntimeObservabilityExportConfig) (RuntimeExporter, error) {
		return exp, nil
	}))
	defer close(block)
	t.Cleanup(rec.Close)

	for i := 0; i < 8; i++ {
		rec.OnEvent(context.Background(), types.Event{
			Version:   types.EventSchemaVersionV1,
			Type:      "run.finished",
			Time:      time.Now(),
			RunID:     "run-a55-overflow",
			Iteration: i,
			Payload: map[string]any{
				"status":     "success",
				"latency_ms": int64(1),
			},
		})
	}
	waitForRuntimeCondition(t, 2*time.Second, func() bool {
		s := rec.ExportSnapshot()
		return s.DropTotal > 0 && s.Status == RuntimeExportStatusDegraded
	})
	snapshot := rec.ExportSnapshot()
	if snapshot.LastReasonCode != RuntimeExportReasonQueueOverflow {
		t.Fatalf("last_reason_code=%q, want %q", snapshot.LastReasonCode, RuntimeExportReasonQueueOverflow)
	}
}

func TestRuntimeRecorderExporterFailFastStopsOnError(t *testing.T) {
	mgr := newRuntimeRecorderTestManager(t, `
runtime:
  observability:
    export:
      enabled: true
      profile: custom
      endpoint: custom://sink
      queue_capacity: 8
      on_error: fail_fast
`)
	exp := &fakeRuntimeExporter{
		exportFn: func(_ []types.Event) error {
			return errors.New("dial tcp 127.0.0.1:9: connectex: connection refused")
		},
	}
	rec := NewRuntimeRecorder(mgr, WithRuntimeExporterFactory(runtimeconfig.RuntimeObservabilityExportProfileCustom, func(_ runtimeconfig.RuntimeObservabilityExportConfig) (RuntimeExporter, error) {
		return exp, nil
	}))
	t.Cleanup(rec.Close)

	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		Time:    time.Now(),
		RunID:   "run-a55-fail-fast",
		Payload: map[string]any{"status": "success"},
	})
	waitForRuntimeCondition(t, 2*time.Second, func() bool {
		s := rec.ExportSnapshot()
		return s.Status == RuntimeExportStatusFailed && s.ErrorTotal > 0
	})
	before := exp.ExportCallTotal()
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		Time:    time.Now(),
		RunID:   "run-a55-fail-fast-2",
		Payload: map[string]any{"status": "success"},
	})
	time.Sleep(80 * time.Millisecond)
	after := exp.ExportCallTotal()
	if after != before {
		t.Fatalf("fail_fast should stop exporter after error, before=%d after=%d", before, after)
	}
	snapshot := rec.ExportSnapshot()
	if snapshot.LastReasonCode != runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable {
		t.Fatalf("last_reason_code=%q, want %q", snapshot.LastReasonCode, runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable)
	}
}

func TestRuntimeRecorderExporterLangfuseAuthInvalidCanonicalReason(t *testing.T) {
	mgr := newRuntimeRecorderTestManager(t, `
runtime:
  observability:
    export:
      enabled: true
      profile: langfuse
      endpoint: https://langfuse.example/auth_invalid
      queue_capacity: 8
      on_error: degrade_and_record
`)
	rec := NewRuntimeRecorder(mgr)
	t.Cleanup(rec.Close)

	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		Time:    time.Now(),
		RunID:   "run-a55-auth",
		Payload: map[string]any{"status": "success"},
	})
	waitForRuntimeCondition(t, 2*time.Second, func() bool {
		s := rec.ExportSnapshot()
		return s.Status == RuntimeExportStatusDegraded && s.ErrorTotal > 0
	})
	snapshot := rec.ExportSnapshot()
	if snapshot.LastReasonCode != runtimeconfig.ReadinessCodeObservabilityExportAuthInvalid {
		t.Fatalf("last_reason_code=%q, want %q", snapshot.LastReasonCode, runtimeconfig.ReadinessCodeObservabilityExportAuthInvalid)
	}
}

func TestRuntimeRecorderExporterConcurrentQueueDelivery(t *testing.T) {
	mgr := newRuntimeRecorderTestManager(t, `
runtime:
  observability:
    export:
      enabled: true
      profile: custom
      endpoint: custom://sink
      queue_capacity: 256
      on_error: degrade_and_record
`)
	exp := &fakeRuntimeExporter{}
	rec := NewRuntimeRecorder(mgr, WithRuntimeExporterFactory(runtimeconfig.RuntimeObservabilityExportProfileCustom, func(_ runtimeconfig.RuntimeObservabilityExportConfig) (RuntimeExporter, error) {
		return exp, nil
	}))
	t.Cleanup(rec.Close)

	const workers = 8
	const perWorker = 25
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				rec.OnEvent(context.Background(), types.Event{
					Version:   types.EventSchemaVersionV1,
					Type:      "run.finished",
					Time:      time.Now(),
					RunID:     "run-a55-concurrent",
					Iteration: i*perWorker + j,
					Payload:   map[string]any{"status": "success"},
				})
			}
		}()
	}
	wg.Wait()
	waitForRuntimeCondition(t, 2*time.Second, func() bool {
		return exp.ExportCallTotal() > 0
	})
	snapshot := rec.ExportSnapshot()
	if snapshot.ErrorTotal != 0 {
		t.Fatalf("error_total=%d, want 0", snapshot.ErrorTotal)
	}
}

type fakeRuntimeExporter struct {
	mu       sync.Mutex
	events   []types.Event
	exportFn func(events []types.Event) error
}

func (e *fakeRuntimeExporter) ExportEvents(_ context.Context, events []types.Event) error {
	e.mu.Lock()
	e.events = append(e.events, events...)
	fn := e.exportFn
	e.mu.Unlock()
	if fn != nil {
		return fn(events)
	}
	return nil
}

func (e *fakeRuntimeExporter) Flush(_ context.Context) error {
	return nil
}

func (e *fakeRuntimeExporter) Shutdown(_ context.Context) error {
	return nil
}

func (e *fakeRuntimeExporter) ExportCallTotal() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.events)
}

func (e *fakeRuntimeExporter) FirstEvent() types.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.events) == 0 {
		return types.Event{}
	}
	return e.events[0]
}

func newRuntimeRecorderTestManager(t *testing.T, content string) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A55_EXPORT_TEST",
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func waitForRuntimeCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not satisfied before timeout")
}
