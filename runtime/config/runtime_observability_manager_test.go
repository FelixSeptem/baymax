package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagerRuntimeObservabilityInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	bundleDir := filepath.ToSlash(filepath.Join(t.TempDir(), "bundles"))
	writeRuntimeObservabilityReloadConfig(t, file, runtimeObservabilityReloadInput{
		Profile:   RuntimeObservabilityExportProfileNone,
		OnError:   RuntimeObservabilityExportOnErrorDegradeAndRecord,
		OutputDir: bundleDir,
	})

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_OBSERVABILITY_TEST", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig()

	writeRuntimeObservabilityReloadConfig(t, file, runtimeObservabilityReloadInput{
		Profile:   "jaeger",
		OnError:   RuntimeObservabilityExportOnErrorDegradeAndRecord,
		OutputDir: bundleDir,
	})
	time.Sleep(250 * time.Millisecond)
	afterProfile := mgr.EffectiveConfig()
	if afterProfile.Runtime.Observability.Export.Profile != before.Runtime.Observability.Export.Profile {
		t.Fatalf(
			"invalid runtime.observability.export.profile should rollback, got %q want %q",
			afterProfile.Runtime.Observability.Export.Profile,
			before.Runtime.Observability.Export.Profile,
		)
	}
	assertLatestReloadFailed(t, mgr, "runtime.observability.export.profile")

	writeRuntimeObservabilityReloadConfig(t, file, runtimeObservabilityReloadInput{
		Profile:   RuntimeObservabilityExportProfileNone,
		OnError:   RuntimeObservabilityExportOnErrorDegradeAndRecord,
		OutputDir: ".",
	})
	time.Sleep(250 * time.Millisecond)
	afterOutput := mgr.EffectiveConfig()
	if afterOutput.Runtime.Diagnostics.Bundle.OutputDir != before.Runtime.Diagnostics.Bundle.OutputDir {
		t.Fatalf(
			"invalid runtime.diagnostics.bundle.output_dir should rollback, got %q want %q",
			afterOutput.Runtime.Diagnostics.Bundle.OutputDir,
			before.Runtime.Diagnostics.Bundle.OutputDir,
		)
	}
	assertLatestReloadFailed(t, mgr, "runtime.diagnostics.bundle.output_dir")

	writeRuntimeObservabilityReloadConfig(t, file, runtimeObservabilityReloadInput{
		Profile:   RuntimeObservabilityExportProfileNone,
		OnError:   "explode",
		OutputDir: bundleDir,
	})
	time.Sleep(250 * time.Millisecond)
	afterPolicy := mgr.EffectiveConfig()
	if afterPolicy.Runtime.Observability.Export.OnError != before.Runtime.Observability.Export.OnError {
		t.Fatalf(
			"invalid runtime.observability.export.on_error should rollback, got %q want %q",
			afterPolicy.Runtime.Observability.Export.OnError,
			before.Runtime.Observability.Export.OnError,
		)
	}
	assertLatestReloadFailed(t, mgr, "runtime.observability.export.on_error")
}

func TestManagerRuntimeObservabilityTracingInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  observability:
    export:
      enabled: true
      profile: otlp
      endpoint: http://127.0.0.1:4318/v1/traces
      queue_capacity: 128
      on_error: degrade_and_record
    tracing:
      otel:
        enabled: true
        protocol: grpc
        endpoint: http://127.0.0.1:4318/v1/traces
        sample_ratio: 1.0
        export_timeout: 5s
        resource_attributes:
          service.name: baymax-runtime
        schema_version: otel_semconv.v1
        on_error: degrade_and_record
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A61_TEST", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig()
	if before.Runtime.Observability.Tracing.OTel.Protocol != RuntimeObservabilityTracingOTelProtocolGRPC {
		t.Fatalf(
			"before runtime.observability.tracing.otel.protocol = %q, want %q",
			before.Runtime.Observability.Tracing.OTel.Protocol,
			RuntimeObservabilityTracingOTelProtocolGRPC,
		)
	}

	writeConfig(t, file, `
runtime:
  observability:
    export:
      enabled: true
      profile: otlp
      endpoint: http://127.0.0.1:4318/v1/traces
      queue_capacity: 128
      on_error: degrade_and_record
    tracing:
      otel:
        enabled: true
        protocol: thrift
        endpoint: http://127.0.0.1:4318/v1/traces
        sample_ratio: 1.0
        export_timeout: 5s
        resource_attributes:
          service.name: baymax-runtime
        schema_version: otel_semconv.v1
        on_error: degrade_and_record
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig()
	if after.Runtime.Observability.Tracing.OTel.Protocol != before.Runtime.Observability.Tracing.OTel.Protocol {
		t.Fatalf(
			"invalid runtime.observability.tracing.otel.protocol should rollback, got %q want %q",
			after.Runtime.Observability.Tracing.OTel.Protocol,
			before.Runtime.Observability.Tracing.OTel.Protocol,
		)
	}
	assertLatestReloadFailed(t, mgr, "runtime.observability.tracing.otel.protocol")
}

type runtimeObservabilityReloadInput struct {
	Profile   string
	OnError   string
	OutputDir string
}

func writeRuntimeObservabilityReloadConfig(t *testing.T, file string, in runtimeObservabilityReloadInput) {
	t.Helper()
	writeConfig(t, file, fmt.Sprintf(`
runtime:
  observability:
    export:
      enabled: false
      profile: %q
      endpoint: ""
      queue_capacity: 128
      on_error: %q
  diagnostics:
    bundle:
      enabled: true
      output_dir: %q
      max_size_mb: 64
      include_sections: [timeline, diagnostics, effective_config]
reload:
  enabled: true
  debounce: 20ms
`, strings.TrimSpace(in.Profile), strings.TrimSpace(in.OnError), strings.TrimSpace(in.OutputDir)))
}

func assertLatestReloadFailed(t *testing.T, mgr *Manager, contains string) {
	t.Helper()
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
	if key := strings.TrimSpace(contains); key != "" && !strings.Contains(reloads[0].Error, key) {
		t.Fatalf("reload error %q does not contain %q", reloads[0].Error, key)
	}
}
