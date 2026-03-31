package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	RuntimeObservabilityExportProfileNone     = "none"
	RuntimeObservabilityExportProfileOTLP     = "otlp"
	RuntimeObservabilityExportProfileLangfuse = "langfuse"
	RuntimeObservabilityExportProfileCustom   = "custom"
)

const (
	RuntimeObservabilityExportOnErrorFailFast         = "fail_fast"
	RuntimeObservabilityExportOnErrorDegradeAndRecord = "degrade_and_record"
)

const (
	RuntimeDiagnosticsBundleSectionTimeline        = "timeline"
	RuntimeDiagnosticsBundleSectionDiagnostics     = "diagnostics"
	RuntimeDiagnosticsBundleSectionEffectiveConfig = "effective_config"
	RuntimeDiagnosticsBundleSectionReplayHints     = "replay_hints"
	RuntimeDiagnosticsBundleSectionGateFingerprint = "gate_fingerprint"
)

const (
	RuntimeDiagnosticsBundleSchemaVersionV1 = "bundle.v1"
	RuntimeDiagnosticsReplayHintSchemaV1    = "observability.v1"
	RuntimeDiagnosticsGateFingerprintA55V1  = "gate.a55.v1"
)

const (
	RuntimeDiagnosticsBundleStatusDisabled = "disabled"
	RuntimeDiagnosticsBundleStatusSuccess  = "success"
	RuntimeDiagnosticsBundleStatusDegraded = "degraded"
	RuntimeDiagnosticsBundleStatusFailed   = "failed"
)

const (
	RuntimeDiagnosticsBundleReasonOutputUnavailable = ReadinessCodeDiagnosticsBundleOutputUnavailable
	RuntimeDiagnosticsBundleReasonPolicyInvalid     = ReadinessCodeDiagnosticsBundlePolicyInvalid
	RuntimeDiagnosticsBundleReasonMaxSizeExceeded   = "diagnostics.bundle.max_size_exceeded"
	RuntimeDiagnosticsBundleReasonSectionMissing    = "diagnostics.bundle.section_missing"
	RuntimeDiagnosticsBundleReasonUnknown           = "diagnostics.bundle.error"
)

type RuntimeObservabilityConfig struct {
	Export RuntimeObservabilityExportConfig `json:"export"`
}

type RuntimeObservabilityExportConfig struct {
	Enabled       bool   `json:"enabled"`
	Profile       string `json:"profile"`
	Endpoint      string `json:"endpoint"`
	QueueCapacity int    `json:"queue_capacity"`
	OnError       string `json:"on_error"`
}

type RuntimeDiagnosticsConfig struct {
	Bundle RuntimeDiagnosticsBundleConfig `json:"bundle"`
}

type RuntimeDiagnosticsBundleConfig struct {
	Enabled         bool     `json:"enabled"`
	OutputDir       string   `json:"output_dir"`
	MaxSizeMB       int      `json:"max_size_mb"`
	IncludeSections []string `json:"include_sections"`
}

func normalizeRuntimeObservabilityConfig(in RuntimeObservabilityConfig) RuntimeObservabilityConfig {
	base := DefaultConfig().Runtime.Observability
	out := in
	out.Export.Profile = normalizeRuntimeObservabilityExportProfile(out.Export.Profile)
	if out.Export.Profile == "" {
		out.Export.Profile = base.Export.Profile
	}
	out.Export.Endpoint = strings.TrimSpace(out.Export.Endpoint)
	if out.Export.QueueCapacity <= 0 {
		out.Export.QueueCapacity = base.Export.QueueCapacity
	}
	out.Export.OnError = normalizeRuntimeObservabilityExportOnError(out.Export.OnError)
	if out.Export.OnError == "" {
		out.Export.OnError = base.Export.OnError
	}
	return out
}

func normalizeRuntimeDiagnosticsBundleConfig(in RuntimeDiagnosticsBundleConfig) RuntimeDiagnosticsBundleConfig {
	base := DefaultConfig().Runtime.Diagnostics.Bundle
	out := in
	out.OutputDir = strings.TrimSpace(out.OutputDir)
	if out.OutputDir == "" {
		out.OutputDir = base.OutputDir
	}
	if out.MaxSizeMB <= 0 {
		out.MaxSizeMB = base.MaxSizeMB
	}
	out.IncludeSections = normalizeRuntimeDiagnosticsBundleSections(out.IncludeSections)
	if len(out.IncludeSections) == 0 {
		out.IncludeSections = append([]string(nil), base.IncludeSections...)
	}
	return out
}

func ValidateRuntimeObservabilityConfig(cfg RuntimeObservabilityConfig) error {
	return ValidateRuntimeObservabilityExportConfig(cfg.Export)
}

func ValidateRuntimeObservabilityExportConfig(cfg RuntimeObservabilityExportConfig) error {
	normalized := cfg
	normalized.Profile = normalizeRuntimeObservabilityExportProfile(normalized.Profile)
	if normalized.Profile == "" {
		normalized.Profile = normalizeRuntimeObservabilityExportProfile(DefaultConfig().Runtime.Observability.Export.Profile)
	}
	normalized.Endpoint = strings.TrimSpace(normalized.Endpoint)
	normalized.OnError = normalizeRuntimeObservabilityExportOnError(normalized.OnError)
	if normalized.OnError == "" {
		normalized.OnError = normalizeRuntimeObservabilityExportOnError(DefaultConfig().Runtime.Observability.Export.OnError)
	}
	if !isSupportedRuntimeObservabilityExportProfile(normalized.Profile) {
		return fmt.Errorf(
			"runtime.observability.export.profile must be one of [%s,%s,%s,%s], got %q",
			RuntimeObservabilityExportProfileNone,
			RuntimeObservabilityExportProfileOTLP,
			RuntimeObservabilityExportProfileLangfuse,
			RuntimeObservabilityExportProfileCustom,
			cfg.Profile,
		)
	}
	if cfg.QueueCapacity <= 0 {
		return fmt.Errorf("runtime.observability.export.queue_capacity must be > 0")
	}
	switch normalized.OnError {
	case RuntimeObservabilityExportOnErrorFailFast, RuntimeObservabilityExportOnErrorDegradeAndRecord:
	default:
		return fmt.Errorf(
			"runtime.observability.export.on_error must be one of [%s,%s], got %q",
			RuntimeObservabilityExportOnErrorFailFast,
			RuntimeObservabilityExportOnErrorDegradeAndRecord,
			cfg.OnError,
		)
	}
	if strings.ContainsRune(normalized.Endpoint, '\x00') {
		return fmt.Errorf("runtime.observability.export.endpoint contains invalid null character")
	}
	if normalized.Profile == RuntimeObservabilityExportProfileNone && normalized.Endpoint != "" {
		return fmt.Errorf(
			"runtime.observability.export.endpoint must be empty when runtime.observability.export.profile=%s",
			RuntimeObservabilityExportProfileNone,
		)
	}
	if normalized.Enabled &&
		normalized.Profile != RuntimeObservabilityExportProfileNone &&
		normalized.Endpoint == "" {
		return fmt.Errorf("runtime.observability.export.endpoint is required when runtime.observability.export.enabled=true")
	}
	return nil
}

func ValidateRuntimeDiagnosticsConfig(cfg RuntimeDiagnosticsConfig) error {
	return ValidateRuntimeDiagnosticsBundleConfig(cfg.Bundle)
}

func ValidateRuntimeDiagnosticsBundleConfig(cfg RuntimeDiagnosticsBundleConfig) error {
	normalized := cfg
	normalized.OutputDir = strings.TrimSpace(normalized.OutputDir)
	normalized.IncludeSections = normalizeRuntimeDiagnosticsBundleSections(normalized.IncludeSections)
	if err := validateRuntimeDiagnosticsBundleOutputDir(normalized.OutputDir); err != nil {
		return err
	}
	if cfg.MaxSizeMB <= 0 {
		return fmt.Errorf("runtime.diagnostics.bundle.max_size_mb must be > 0")
	}
	if len(normalized.IncludeSections) == 0 {
		return fmt.Errorf("runtime.diagnostics.bundle.include_sections must not be empty")
	}
	seen := map[string]struct{}{}
	for _, raw := range normalized.IncludeSections {
		section := strings.ToLower(strings.TrimSpace(raw))
		if section == "" {
			continue
		}
		if _, ok := seen[section]; ok {
			continue
		}
		seen[section] = struct{}{}
		if !isSupportedRuntimeDiagnosticsBundleSection(section) {
			return fmt.Errorf("runtime.diagnostics.bundle.include_sections contains unsupported section %q", raw)
		}
	}
	if len(seen) == 0 {
		return fmt.Errorf("runtime.diagnostics.bundle.include_sections must contain at least one section")
	}
	return nil
}

func validateRuntimeDiagnosticsBundleOutputDir(raw string) error {
	path := strings.TrimSpace(raw)
	if path == "" {
		return fmt.Errorf("runtime.diagnostics.bundle.output_dir is required")
	}
	if strings.ContainsRune(path, '\x00') {
		return fmt.Errorf("runtime.diagnostics.bundle.output_dir contains invalid null character")
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.TrimSpace(clean) == "" {
		return fmt.Errorf("runtime.diagnostics.bundle.output_dir is malformed")
	}
	return nil
}

func normalizeRuntimeObservabilityExportProfile(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeRuntimeObservabilityExportOnError(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeRuntimeDiagnosticsBundleSections(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		section := strings.ToLower(strings.TrimSpace(item))
		if section == "" {
			continue
		}
		if _, ok := seen[section]; ok {
			continue
		}
		seen[section] = struct{}{}
		out = append(out, section)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isSupportedRuntimeObservabilityExportProfile(raw string) bool {
	switch normalizeRuntimeObservabilityExportProfile(raw) {
	case RuntimeObservabilityExportProfileNone,
		RuntimeObservabilityExportProfileOTLP,
		RuntimeObservabilityExportProfileLangfuse,
		RuntimeObservabilityExportProfileCustom:
		return true
	default:
		return false
	}
}

func isSupportedRuntimeDiagnosticsBundleSection(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case RuntimeDiagnosticsBundleSectionTimeline,
		RuntimeDiagnosticsBundleSectionDiagnostics,
		RuntimeDiagnosticsBundleSectionEffectiveConfig,
		RuntimeDiagnosticsBundleSectionReplayHints,
		RuntimeDiagnosticsBundleSectionGateFingerprint:
		return true
	default:
		return false
	}
}
