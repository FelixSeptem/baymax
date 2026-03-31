package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type DiagnosticsBundleGenerateRequest struct {
	RunID              string
	GeneratedAt        time.Time
	GateFingerprint    string
	RunFinishedPayload map[string]any
}

type DiagnosticsBundleGenerateResult struct {
	BundleDir    string
	ManifestPath string
	Manifest     DiagnosticsBundleManifest
	Status       string
	ReasonCode   string
	Total        int
}

type DiagnosticsBundleManifest struct {
	SchemaVersion string                            `json:"schema_version"`
	GeneratedAt   time.Time                         `json:"generated_at"`
	Metadata      DiagnosticsBundleManifestMetadata `json:"metadata"`
	Sections      []DiagnosticsBundleManifestItem   `json:"sections"`
}

type DiagnosticsBundleManifestMetadata struct {
	RunID             string `json:"run_id,omitempty"`
	RuntimeEnvPrefix  string `json:"runtime_env_prefix,omitempty"`
	RuntimeConfigPath string `json:"runtime_config_path,omitempty"`
	ReplayHintSchema  string `json:"replay_hint_schema"`
	GateFingerprint   string `json:"gate_fingerprint"`
	RedactionStatus   string `json:"redaction_status"`
}

type DiagnosticsBundleManifestItem struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type DiagnosticsBundleError struct {
	Code    string
	Message string
	Err     error
}

func (e *DiagnosticsBundleError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Err != nil {
		msg = strings.TrimSpace(e.Err.Error())
	}
	code := strings.TrimSpace(e.Code)
	switch {
	case code != "" && msg != "":
		return code + ": " + msg
	case code != "":
		return code
	default:
		return msg
	}
}

func (e *DiagnosticsBundleError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func DiagnosticsBundleErrorCode(err error) string {
	var typed *DiagnosticsBundleError
	if errors.As(err, &typed) && typed != nil {
		return strings.TrimSpace(typed.Code)
	}
	return ""
}

func (m *Manager) GenerateDiagnosticsBundle(ctx context.Context, req DiagnosticsBundleGenerateRequest) (DiagnosticsBundleGenerateResult, error) {
	if m == nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonUnknown,
				Total:      1,
				Manifest: DiagnosticsBundleManifest{
					SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
				},
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonUnknown,
				Message: "runtime config manager is nil",
			}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonUnknown,
				Total:      1,
				Manifest: DiagnosticsBundleManifest{
					SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
				},
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonUnknown,
				Message: "diagnostics bundle generation canceled",
				Err:     err,
			}
	}

	cfg := normalizeRuntimeDiagnosticsBundleConfig(m.EffectiveConfig().Runtime.Diagnostics.Bundle)
	if !cfg.Enabled {
		return DiagnosticsBundleGenerateResult{
			Status: RuntimeDiagnosticsBundleStatusDisabled,
			Total:  0,
			Manifest: DiagnosticsBundleManifest{
				SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
			},
		}, nil
	}
	if err := validateRuntimeDiagnosticsBundleOutputDir(cfg.OutputDir); err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonPolicyInvalid,
				Total:      1,
				Manifest: DiagnosticsBundleManifest{
					SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
				},
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonPolicyInvalid,
				Message: "invalid runtime diagnostics bundle output dir",
				Err:     err,
			}
	}

	sections := normalizeRuntimeDiagnosticsBundleSections(cfg.IncludeSections)
	if err := ensureRuntimeDiagnosticsBundleRequiredSections(sections); err != nil {
		return DiagnosticsBundleGenerateResult{
			Status:     RuntimeDiagnosticsBundleStatusFailed,
			ReasonCode: RuntimeDiagnosticsBundleReasonSectionMissing,
			Total:      1,
			Manifest: DiagnosticsBundleManifest{
				SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
			},
		}, err
	}

	now := req.GeneratedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	runID := strings.TrimSpace(req.RunID)
	fingerprint := strings.TrimSpace(req.GateFingerprint)
	if fingerprint == "" {
		fingerprint = RuntimeDiagnosticsGateFingerprintA55V1
	}

	sectionFiles, err := m.buildDiagnosticsBundleSections(sections, runID, now, fingerprint, req.RunFinishedPayload)
	if err != nil {
		return DiagnosticsBundleGenerateResult{
			Status:     RuntimeDiagnosticsBundleStatusFailed,
			ReasonCode: DiagnosticsBundleErrorCode(err),
			Total:      1,
			Manifest: DiagnosticsBundleManifest{
				SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
			},
		}, err
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o700); err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Total:      1,
				Manifest: DiagnosticsBundleManifest{
					SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
				},
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Message: "prepare diagnostics bundle output dir",
				Err:     err,
			}
	}

	manifest := DiagnosticsBundleManifest{
		SchemaVersion: RuntimeDiagnosticsBundleSchemaVersionV1,
		GeneratedAt:   now,
		Metadata: DiagnosticsBundleManifestMetadata{
			RunID:             runID,
			RuntimeEnvPrefix:  strings.TrimSpace(m.envPrefix),
			RuntimeConfigPath: strings.TrimSpace(m.filePath),
			ReplayHintSchema:  RuntimeDiagnosticsReplayHintSchemaV1,
			GateFingerprint:   fingerprint,
			RedactionStatus:   "redacted",
		},
		Sections: make([]DiagnosticsBundleManifestItem, 0, len(sectionFiles)),
	}
	totalBytes := int64(0)
	for _, section := range sectionFiles {
		item := DiagnosticsBundleManifestItem{
			Name:      section.Name,
			File:      section.FileName,
			SHA256:    hashBundleBytes(section.Content),
			SizeBytes: int64(len(section.Content)),
		}
		manifest.Sections = append(manifest.Sections, item)
		totalBytes += item.SizeBytes
	}
	sort.Slice(manifest.Sections, func(i, j int) bool {
		return manifest.Sections[i].Name < manifest.Sections[j].Name
	})

	manifestBytes, err := marshalBundleJSON(manifest)
	if err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonUnknown,
				Total:      1,
				Manifest:   manifest,
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonUnknown,
				Message: "marshal diagnostics bundle manifest",
				Err:     err,
			}
	}
	totalBytes += int64(len(manifestBytes))
	maxBytes := int64(cfg.MaxSizeMB) * 1024 * 1024
	if maxBytes > 0 && totalBytes > maxBytes {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonMaxSizeExceeded,
				Total:      1,
				Manifest:   manifest,
			}, &DiagnosticsBundleError{
				Code: RuntimeDiagnosticsBundleReasonMaxSizeExceeded,
				Message: fmt.Sprintf(
					"diagnostics bundle size exceeds max_size_mb (size=%d bytes, max=%d bytes)",
					totalBytes,
					maxBytes,
				),
			}
	}

	baseName := sanitizeDiagnosticsBundlePathSegment(runID)
	if baseName == "" {
		baseName = "run"
	}
	baseName = fmt.Sprintf("%s-%s", baseName, now.Format("20060102T150405.000000000Z"))
	baseName = strings.ReplaceAll(baseName, ".", "")

	finalDir := filepath.Join(cfg.OutputDir, baseName)
	tmpDir := finalDir + ".tmp"
	if err := os.RemoveAll(tmpDir); err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Total:      1,
				Manifest:   manifest,
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Message: "clean diagnostics bundle temp dir",
				Err:     err,
			}
	}
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Total:      1,
				Manifest:   manifest,
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Message: "create diagnostics bundle temp dir",
				Err:     err,
			}
	}

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	for _, section := range sectionFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, section.FileName), section.Content, 0o600); err != nil {
			return DiagnosticsBundleGenerateResult{
					Status:     RuntimeDiagnosticsBundleStatusFailed,
					ReasonCode: RuntimeDiagnosticsBundleReasonOutputUnavailable,
					Total:      1,
					Manifest:   manifest,
				}, &DiagnosticsBundleError{
					Code:    RuntimeDiagnosticsBundleReasonOutputUnavailable,
					Message: "write diagnostics bundle section",
					Err:     err,
				}
		}
	}
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Total:      1,
				Manifest:   manifest,
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Message: "write diagnostics bundle manifest",
				Err:     err,
			}
	}

	finalDir = ensureDiagnosticsBundleUniqueDir(finalDir)
	if err := os.Rename(tmpDir, finalDir); err != nil {
		return DiagnosticsBundleGenerateResult{
				Status:     RuntimeDiagnosticsBundleStatusFailed,
				ReasonCode: RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Total:      1,
				Manifest:   manifest,
			}, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonOutputUnavailable,
				Message: "publish diagnostics bundle",
				Err:     err,
			}
	}

	return DiagnosticsBundleGenerateResult{
		BundleDir:    finalDir,
		ManifestPath: filepath.Join(finalDir, "manifest.json"),
		Manifest:     manifest,
		Status:       RuntimeDiagnosticsBundleStatusSuccess,
		Total:        1,
	}, nil
}

type diagnosticsBundleSectionFile struct {
	Name     string
	FileName string
	Content  []byte
}

func (m *Manager) buildDiagnosticsBundleSections(
	sections []string,
	runID string,
	generatedAt time.Time,
	fingerprint string,
	runFinishedPayload map[string]any,
) ([]diagnosticsBundleSectionFile, error) {
	out := make([]diagnosticsBundleSectionFile, 0, len(sections))
	for _, section := range sections {
		payload, err := m.buildDiagnosticsBundleSectionPayload(section, runID, generatedAt, fingerprint, runFinishedPayload)
		if err != nil {
			return nil, err
		}
		content, err := marshalAndRedactBundlePayload(m, payload)
		if err != nil {
			return nil, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonUnknown,
				Message: fmt.Sprintf("marshal section %q", section),
				Err:     err,
			}
		}
		if len(content) == 0 {
			return nil, &DiagnosticsBundleError{
				Code:    RuntimeDiagnosticsBundleReasonSectionMissing,
				Message: fmt.Sprintf("section %q is empty", section),
			}
		}
		out = append(out, diagnosticsBundleSectionFile{
			Name:     section,
			FileName: section + ".json",
			Content:  content,
		})
	}
	return out, nil
}

func (m *Manager) buildDiagnosticsBundleSectionPayload(
	section string,
	runID string,
	generatedAt time.Time,
	fingerprint string,
	runFinishedPayload map[string]any,
) (any, error) {
	section = strings.ToLower(strings.TrimSpace(section))
	switch section {
	case RuntimeDiagnosticsBundleSectionTimeline:
		runs := m.RecentRuns(200)
		windowStart := time.Time{}
		windowEnd := time.Time{}
		if len(runs) > 0 {
			windowStart = runs[0].Time.UTC()
			windowEnd = runs[len(runs)-1].Time.UTC()
		}
		return map[string]any{
			"run_id":       strings.TrimSpace(runID),
			"window_start": windowStart,
			"window_end":   windowEnd,
			"recent_runs":  runs,
		}, nil
	case RuntimeDiagnosticsBundleSectionDiagnostics:
		return map[string]any{
			"run_id":               strings.TrimSpace(runID),
			"recent_runs":          m.RecentRuns(200),
			"recent_calls":         m.RecentCalls(200),
			"recent_reloads":       m.RecentReloads(100),
			"recent_skills":        m.RecentSkills(200),
			"run_finished_payload": cloneBundlePayload(runFinishedPayload),
		}, nil
	case RuntimeDiagnosticsBundleSectionEffectiveConfig:
		return map[string]any{
			"effective_config": m.EffectiveConfigSanitized(),
		}, nil
	case RuntimeDiagnosticsBundleSectionReplayHints:
		return map[string]any{
			"fixture_schema_version": RuntimeDiagnosticsReplayHintSchemaV1,
			"run_id":                 strings.TrimSpace(runID),
			"normalization_hints": map[string]any{
				"status_case":          "lower",
				"reason_code_prefix":   "observability.export.|diagnostics.bundle.",
				"bundle_schema_prefix": "bundle.",
			},
		}, nil
	case RuntimeDiagnosticsBundleSectionGateFingerprint:
		return map[string]any{
			"fingerprint":  strings.TrimSpace(fingerprint),
			"generated_at": generatedAt.UTC(),
			"gates": []string{
				"scripts/check-observability-export-and-bundle-contract.sh",
				"scripts/check-observability-export-and-bundle-contract.ps1",
			},
		}, nil
	default:
		return nil, &DiagnosticsBundleError{
			Code:    RuntimeDiagnosticsBundleReasonPolicyInvalid,
			Message: fmt.Sprintf("unsupported diagnostics bundle section %q", section),
		}
	}
}

func ensureRuntimeDiagnosticsBundleRequiredSections(sections []string) error {
	required := []string{
		RuntimeDiagnosticsBundleSectionTimeline,
		RuntimeDiagnosticsBundleSectionDiagnostics,
		RuntimeDiagnosticsBundleSectionEffectiveConfig,
		RuntimeDiagnosticsBundleSectionReplayHints,
		RuntimeDiagnosticsBundleSectionGateFingerprint,
	}
	seen := map[string]struct{}{}
	for _, section := range sections {
		seen[strings.ToLower(strings.TrimSpace(section))] = struct{}{}
	}
	missing := make([]string, 0, len(required))
	for _, section := range required {
		if _, ok := seen[section]; ok {
			continue
		}
		missing = append(missing, section)
	}
	if len(missing) == 0 {
		return nil
	}
	return &DiagnosticsBundleError{
		Code: RuntimeDiagnosticsBundleReasonSectionMissing,
		Message: fmt.Sprintf(
			"runtime.diagnostics.bundle.include_sections misses required sections: %s",
			strings.Join(missing, ","),
		),
	}
}

func marshalAndRedactBundlePayload(m *Manager, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	redacted := strings.TrimSpace(m.RedactJSONText(string(raw)))
	if redacted == "" {
		redacted = "{}"
	}
	var normalized any
	if err := json.Unmarshal([]byte(redacted), &normalized); err != nil {
		return nil, err
	}
	return marshalBundleJSON(normalized)
}

func marshalBundleJSON(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func cloneBundlePayload(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func hashBundleBytes(in []byte) string {
	sum := sha256.Sum256(in)
	return hex.EncodeToString(sum[:])
}

func sanitizeDiagnosticsBundlePathSegment(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z':
			b.WriteRune(ch)
		case ch >= 'A' && ch <= 'Z':
			b.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		case ch == '-', ch == '_':
			b.WriteRune(ch)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func ensureDiagnosticsBundleUniqueDir(path string) string {
	base := strings.TrimSpace(path)
	if base == "" {
		return path
	}
	if _, err := os.Stat(base); errors.Is(err, os.ErrNotExist) {
		return base
	}
	for idx := 1; idx <= 9999; idx++ {
		candidate := fmt.Sprintf("%s-%d", base, idx)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UTC().UnixNano())
}
