package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeObservabilityConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.Observability.Export.Enabled {
		t.Fatal("runtime.observability.export.enabled = true, want false")
	}
	if cfg.Runtime.Observability.Export.Profile != RuntimeObservabilityExportProfileNone {
		t.Fatalf(
			"runtime.observability.export.profile = %q, want %q",
			cfg.Runtime.Observability.Export.Profile,
			RuntimeObservabilityExportProfileNone,
		)
	}
	if cfg.Runtime.Observability.Export.QueueCapacity <= 0 {
		t.Fatalf("runtime.observability.export.queue_capacity = %d, want >0", cfg.Runtime.Observability.Export.QueueCapacity)
	}
	if cfg.Runtime.Observability.Export.OnError != RuntimeObservabilityExportOnErrorDegradeAndRecord {
		t.Fatalf(
			"runtime.observability.export.on_error = %q, want %q",
			cfg.Runtime.Observability.Export.OnError,
			RuntimeObservabilityExportOnErrorDegradeAndRecord,
		)
	}
	if cfg.Runtime.Observability.Tracing.OTel.Enabled {
		t.Fatal("runtime.observability.tracing.otel.enabled = true, want false")
	}
	if cfg.Runtime.Observability.Tracing.OTel.Protocol != RuntimeObservabilityTracingOTelProtocolGRPC {
		t.Fatalf(
			"runtime.observability.tracing.otel.protocol = %q, want %q",
			cfg.Runtime.Observability.Tracing.OTel.Protocol,
			RuntimeObservabilityTracingOTelProtocolGRPC,
		)
	}
	if cfg.Runtime.Observability.Tracing.OTel.SampleRatio <= 0 || cfg.Runtime.Observability.Tracing.OTel.SampleRatio > 1 {
		t.Fatalf("runtime.observability.tracing.otel.sample_ratio = %v, want in (0,1]", cfg.Runtime.Observability.Tracing.OTel.SampleRatio)
	}
	if cfg.Runtime.Observability.Tracing.OTel.ExportTimeout <= 0 {
		t.Fatalf("runtime.observability.tracing.otel.export_timeout = %v, want >0", cfg.Runtime.Observability.Tracing.OTel.ExportTimeout)
	}
	if cfg.Runtime.Observability.Tracing.OTel.SchemaVersion != RuntimeObservabilityTraceSchemaVersionOTelSemconvV1 {
		t.Fatalf(
			"runtime.observability.tracing.otel.schema_version = %q, want %q",
			cfg.Runtime.Observability.Tracing.OTel.SchemaVersion,
			RuntimeObservabilityTraceSchemaVersionOTelSemconvV1,
		)
	}
	if cfg.Runtime.Observability.Tracing.OTel.OnError != RuntimeObservabilityExportOnErrorDegradeAndRecord {
		t.Fatalf(
			"runtime.observability.tracing.otel.on_error = %q, want %q",
			cfg.Runtime.Observability.Tracing.OTel.OnError,
			RuntimeObservabilityExportOnErrorDegradeAndRecord,
		)
	}
	if len(cfg.Runtime.Observability.Tracing.OTel.ResourceAttributes) == 0 {
		t.Fatal("runtime.observability.tracing.otel.resource_attributes should not be empty")
	}
	if cfg.Runtime.Diagnostics.Bundle.Enabled {
		t.Fatal("runtime.diagnostics.bundle.enabled = true, want false")
	}
	if strings.TrimSpace(cfg.Runtime.Diagnostics.Bundle.OutputDir) == "" {
		t.Fatal("runtime.diagnostics.bundle.output_dir should not be empty by default")
	}
	if cfg.Runtime.Diagnostics.Bundle.MaxSizeMB <= 0 {
		t.Fatalf("runtime.diagnostics.bundle.max_size_mb = %d, want >0", cfg.Runtime.Diagnostics.Bundle.MaxSizeMB)
	}
	if len(cfg.Runtime.Diagnostics.Bundle.IncludeSections) == 0 {
		t.Fatal("runtime.diagnostics.bundle.include_sections should not be empty")
	}
}

func TestRuntimeObservabilityConfigEnvOverridePrecedence(t *testing.T) {
	bundleDir := filepath.ToSlash(filepath.Join(t.TempDir(), "env-bundles"))
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_EXPORT_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_EXPORT_PROFILE", RuntimeObservabilityExportProfileLangfuse)
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_EXPORT_ENDPOINT", "https://langfuse.env.test")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_EXPORT_QUEUE_CAPACITY", "2048")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_EXPORT_ON_ERROR", RuntimeObservabilityExportOnErrorFailFast)
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_PROTOCOL", RuntimeObservabilityTracingOTelProtocolHTTPProtobuf)
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_ENDPOINT", "http://collector.env.test:4318/v1/traces")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_SAMPLE_RATIO", "0.25")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_EXPORT_TIMEOUT", "7s")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_RESOURCE_ATTRIBUTES", "service.name=baymax-test,service.version=v1")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_SCHEMA_VERSION", RuntimeObservabilityTraceSchemaVersionOTelSemconvV1)
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_ON_ERROR", RuntimeObservabilityExportOnErrorFailFast)
	t.Setenv("BAYMAX_RUNTIME_DIAGNOSTICS_BUNDLE_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_DIAGNOSTICS_BUNDLE_OUTPUT_DIR", bundleDir)
	t.Setenv("BAYMAX_RUNTIME_DIAGNOSTICS_BUNDLE_MAX_SIZE_MB", "32")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  observability:
    export:
      enabled: false
      profile: none
      endpoint: ""
      queue_capacity: 64
      on_error: degrade_and_record
    tracing:
      otel:
        enabled: false
        protocol: grpc
        endpoint: ""
        sample_ratio: 1
        export_timeout: 3s
        resource_attributes:
          service.name: from-file
        schema_version: otel_semconv.v1
        on_error: degrade_and_record
  diagnostics:
    bundle:
      enabled: false
      output_dir: /tmp/file-bundles
      max_size_mb: 8
      include_sections: [timeline, diagnostics]
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Runtime.Observability.Export.Enabled {
		t.Fatal("runtime.observability.export.enabled = false, want true from env")
	}
	if cfg.Runtime.Observability.Export.Profile != RuntimeObservabilityExportProfileLangfuse {
		t.Fatalf(
			"runtime.observability.export.profile = %q, want %q from env",
			cfg.Runtime.Observability.Export.Profile,
			RuntimeObservabilityExportProfileLangfuse,
		)
	}
	if cfg.Runtime.Observability.Export.Endpoint != "https://langfuse.env.test" {
		t.Fatalf(
			"runtime.observability.export.endpoint = %q, want env override",
			cfg.Runtime.Observability.Export.Endpoint,
		)
	}
	if cfg.Runtime.Observability.Export.QueueCapacity != 2048 {
		t.Fatalf(
			"runtime.observability.export.queue_capacity = %d, want 2048 from env",
			cfg.Runtime.Observability.Export.QueueCapacity,
		)
	}
	if cfg.Runtime.Observability.Export.OnError != RuntimeObservabilityExportOnErrorFailFast {
		t.Fatalf(
			"runtime.observability.export.on_error = %q, want %q from env",
			cfg.Runtime.Observability.Export.OnError,
			RuntimeObservabilityExportOnErrorFailFast,
		)
	}
	if !cfg.Runtime.Observability.Tracing.OTel.Enabled {
		t.Fatal("runtime.observability.tracing.otel.enabled = false, want true from env")
	}
	if cfg.Runtime.Observability.Tracing.OTel.Protocol != RuntimeObservabilityTracingOTelProtocolHTTPProtobuf {
		t.Fatalf(
			"runtime.observability.tracing.otel.protocol = %q, want %q from env",
			cfg.Runtime.Observability.Tracing.OTel.Protocol,
			RuntimeObservabilityTracingOTelProtocolHTTPProtobuf,
		)
	}
	if cfg.Runtime.Observability.Tracing.OTel.Endpoint != "http://collector.env.test:4318/v1/traces" {
		t.Fatalf("runtime.observability.tracing.otel.endpoint = %q, want env override", cfg.Runtime.Observability.Tracing.OTel.Endpoint)
	}
	if cfg.Runtime.Observability.Tracing.OTel.SampleRatio != 0.25 {
		t.Fatalf("runtime.observability.tracing.otel.sample_ratio = %v, want 0.25", cfg.Runtime.Observability.Tracing.OTel.SampleRatio)
	}
	if cfg.Runtime.Observability.Tracing.OTel.ExportTimeout != 7*time.Second {
		t.Fatalf("runtime.observability.tracing.otel.export_timeout = %v, want 7s", cfg.Runtime.Observability.Tracing.OTel.ExportTimeout)
	}
	if cfg.Runtime.Observability.Tracing.OTel.OnError != RuntimeObservabilityExportOnErrorFailFast {
		t.Fatalf(
			"runtime.observability.tracing.otel.on_error = %q, want %q from env",
			cfg.Runtime.Observability.Tracing.OTel.OnError,
			RuntimeObservabilityExportOnErrorFailFast,
		)
	}
	if cfg.Runtime.Observability.Tracing.OTel.ResourceAttributes["service.name"] != "baymax-test" ||
		cfg.Runtime.Observability.Tracing.OTel.ResourceAttributes["service.version"] != "v1" {
		t.Fatalf("runtime.observability.tracing.otel.resource_attributes mismatch: %#v", cfg.Runtime.Observability.Tracing.OTel.ResourceAttributes)
	}
	if !cfg.Runtime.Diagnostics.Bundle.Enabled {
		t.Fatal("runtime.diagnostics.bundle.enabled = false, want true from env")
	}
	if cfg.Runtime.Diagnostics.Bundle.OutputDir != bundleDir {
		t.Fatalf(
			"runtime.diagnostics.bundle.output_dir = %q, want %q from env",
			cfg.Runtime.Diagnostics.Bundle.OutputDir,
			bundleDir,
		)
	}
	if cfg.Runtime.Diagnostics.Bundle.MaxSizeMB != 32 {
		t.Fatalf(
			"runtime.diagnostics.bundle.max_size_mb = %d, want 32 from env",
			cfg.Runtime.Diagnostics.Bundle.MaxSizeMB,
		)
	}
}

func TestRuntimeObservabilityConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Observability.Export.Profile = "jaeger"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.export.profile") {
		t.Fatalf("expected runtime.observability.export.profile validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Export.Profile = RuntimeObservabilityExportProfileNone
	cfg.Runtime.Observability.Export.Endpoint = "https://should-be-empty.example"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.export.endpoint must be empty") {
		t.Fatalf("expected runtime.observability.export.endpoint empty validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Export.Enabled = true
	cfg.Runtime.Observability.Export.Profile = RuntimeObservabilityExportProfileOTLP
	cfg.Runtime.Observability.Export.Endpoint = ""
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.export.endpoint is required") {
		t.Fatalf("expected runtime.observability.export.endpoint required validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Export.OnError = "panic"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.export.on_error") {
		t.Fatalf("expected runtime.observability.export.on_error validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Export.QueueCapacity = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.export.queue_capacity") {
		t.Fatalf("expected runtime.observability.export.queue_capacity validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Diagnostics.Bundle.OutputDir = "."
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.diagnostics.bundle.output_dir") {
		t.Fatalf("expected runtime.diagnostics.bundle.output_dir validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Diagnostics.Bundle.IncludeSections = []string{"unknown"}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.diagnostics.bundle.include_sections") {
		t.Fatalf("expected runtime.diagnostics.bundle.include_sections validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Diagnostics.Bundle.MaxSizeMB = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.diagnostics.bundle.max_size_mb") {
		t.Fatalf("expected runtime.diagnostics.bundle.max_size_mb validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Tracing.OTel.Protocol = "thrift"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.tracing.otel.protocol") {
		t.Fatalf("expected runtime.observability.tracing.otel.protocol validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Tracing.OTel.Enabled = true
	cfg.Runtime.Observability.Tracing.OTel.Endpoint = ""
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.tracing.otel.endpoint is required") {
		t.Fatalf("expected runtime.observability.tracing.otel.endpoint required validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Tracing.OTel.SampleRatio = 1.5
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.tracing.otel.sample_ratio") {
		t.Fatalf("expected runtime.observability.tracing.otel.sample_ratio validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Tracing.OTel.ExportTimeout = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.tracing.otel.export_timeout") {
		t.Fatalf("expected runtime.observability.tracing.otel.export_timeout validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Tracing.OTel.SchemaVersion = "otel_semconv.v2"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.tracing.otel.schema_version") {
		t.Fatalf("expected runtime.observability.tracing.otel.schema_version validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Observability.Tracing.OTel.ResourceAttributes = map[string]string{"service.name": ""}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.observability.tracing.otel.resource_attributes") {
		t.Fatalf("expected runtime.observability.tracing.otel.resource_attributes validation error, got %v", err)
	}
}

func TestRuntimeObservabilityConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_EXPORT_ENABLED", "definitely-not-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.observability.export.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.observability.export.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_EXPORT_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_DIAGNOSTICS_BUNDLE_ENABLED", "definitely-not-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.diagnostics.bundle.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.diagnostics.bundle.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_DIAGNOSTICS_BUNDLE_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_OBSERVABILITY_TRACING_OTEL_ENABLED", "definitely-not-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.observability.tracing.otel.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.observability.tracing.otel.enabled, got %v", err)
	}
}

func TestRuntimeObservabilityTracingEndpointFallbackToExportOTLPEndpoint(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  observability:
    export:
      enabled: true
      profile: otlp
      endpoint: http://collector.file.test:4318/v1/traces
      queue_capacity: 128
      on_error: degrade_and_record
    tracing:
      otel:
        enabled: true
        protocol: grpc
        endpoint: ""
        sample_ratio: 0.5
        export_timeout: 4s
        resource_attributes:
          service.name: fallback-test
        schema_version: otel_semconv.v1
        on_error: degrade_and_record
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Runtime.Observability.Tracing.OTel.Endpoint != "http://collector.file.test:4318/v1/traces" {
		t.Fatalf(
			"runtime.observability.tracing.otel.endpoint = %q, want fallback to runtime.observability.export.endpoint",
			cfg.Runtime.Observability.Tracing.OTel.Endpoint,
		)
	}
}
