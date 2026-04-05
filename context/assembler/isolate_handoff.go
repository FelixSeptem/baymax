package assembler

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	contextIsolateHandoffVersionV1 = "context_isolate_handoff.v1"

	isolateHandoffRejectInvalidSummary     = "invalid_summary"
	isolateHandoffRejectConfidenceBounds   = "confidence_out_of_range"
	isolateHandoffRejectConfidenceTooLow   = "confidence_below_min_confidence"
	isolateHandoffRejectTTLExpired         = "ttl_expired"
	isolateHandoffRejectEvidenceRefInvalid = "evidence_ref_invalid"
	isolateHandoffRejectEvidenceRefMissing = "evidence_ref_missing"
)

type isolateHandoffValidationError struct {
	reason string
	cause  error
}

func (e isolateHandoffValidationError) Error() string {
	if e.cause == nil {
		return e.reason
	}
	return fmt.Sprintf("%s: %v", e.reason, e.cause)
}

func (e isolateHandoffValidationError) Unwrap() error {
	return e.cause
}

func ingestIsolateHandoffChunks(
	chunks []string,
	source string,
	now time.Time,
	cfg runtimeconfig.RuntimeContextJITIsolateHandoffConfig,
	stagePolicy string,
) ([]string, types.IsolateHandoffIngestionPayload, error) {
	payload := types.IsolateHandoffIngestionPayload{
		DeferBody: true,
		Version:   contextIsolateHandoffVersionV1,
	}
	if len(chunks) == 0 {
		return nil, payload, nil
	}
	_, evidenceCatalog := discoverStage2References(chunks, source, len(chunks))
	out := make([]string, 0, len(chunks))
	for idx := range chunks {
		chunk := strings.TrimSpace(chunks[idx])
		candidate, ok := parseIsolateHandoffCandidate(chunk)
		if !ok {
			out = append(out, chunks[idx])
			continue
		}
		normalized, err := normalizeAndValidateIsolateHandoff(candidate, now, cfg, evidenceCatalog)
		if err != nil {
			if isBestEffortPolicy(stagePolicy) {
				payload.RejectedTotal++
				payload.RejectedReasons = append(payload.RejectedReasons, isolateHandoffRejectReason(err))
				continue
			}
			return nil, payload, fmt.Errorf("isolate_handoff[%d]: %w", idx, err)
		}
		payload.Handoffs = append(payload.Handoffs, sanitizedHandoffForIngestion(normalized))
		payload.AcceptedTotal++
		out = append(out, normalized.Summary)
	}
	if len(payload.RejectedReasons) > 0 {
		sort.Strings(payload.RejectedReasons)
	}
	return out, payload, nil
}

func parseIsolateHandoffCandidate(raw string) (types.IsolateHandoffPayload, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return types.IsolateHandoffPayload{}, false
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(trimmed), &probe); err != nil {
		return types.IsolateHandoffPayload{}, false
	}
	if !isIsolateHandoffShape(probe) {
		return types.IsolateHandoffPayload{}, false
	}
	canonical := types.IsolateHandoffPayload{}
	blob, err := json.Marshal(probe)
	if err != nil {
		return types.IsolateHandoffPayload{}, false
	}
	if err := json.Unmarshal(blob, &canonical); err != nil {
		return types.IsolateHandoffPayload{}, false
	}
	return canonical, true
}

func isIsolateHandoffShape(payload map[string]any) bool {
	_, hasSummary := payload["summary"]
	_, hasConfidence := payload["confidence"]
	_, hasTTL := payload["ttl"]
	return hasSummary && hasConfidence && hasTTL
}

func normalizeAndValidateIsolateHandoff(
	in types.IsolateHandoffPayload,
	now time.Time,
	cfg runtimeconfig.RuntimeContextJITIsolateHandoffConfig,
	evidenceCatalog map[string]string,
) (types.IsolateHandoffPayload, error) {
	out := in
	out.Summary = strings.TrimSpace(out.Summary)
	if out.Summary == "" {
		return types.IsolateHandoffPayload{}, isolateHandoffValidationError{
			reason: isolateHandoffRejectInvalidSummary,
			cause:  fmt.Errorf("summary is required"),
		}
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		return types.IsolateHandoffPayload{}, isolateHandoffValidationError{
			reason: isolateHandoffRejectConfidenceBounds,
			cause:  fmt.Errorf("confidence must be in [0,1], got %f", out.Confidence),
		}
	}
	if out.Confidence < cfg.MinConfidence {
		return types.IsolateHandoffPayload{}, isolateHandoffValidationError{
			reason: isolateHandoffRejectConfidenceTooLow,
			cause: fmt.Errorf(
				"confidence %f below runtime.context.jit.isolate_handoff.min_confidence %f",
				out.Confidence,
				cfg.MinConfidence,
			),
		}
	}
	if out.TTL <= 0 {
		out.TTL = now.Add(time.Duration(cfg.DefaultTTLMS) * time.Millisecond).UnixMilli()
	}
	if out.TTL <= now.UnixMilli() {
		return types.IsolateHandoffPayload{}, isolateHandoffValidationError{
			reason: isolateHandoffRejectTTLExpired,
			cause:  fmt.Errorf("ttl expired at unix_ms=%d", out.TTL),
		}
	}

	for idx := range out.Artifacts {
		out.Artifacts[idx].ID = strings.TrimSpace(out.Artifacts[idx].ID)
		out.Artifacts[idx].Type = strings.TrimSpace(out.Artifacts[idx].Type)
		out.Artifacts[idx].Locator = strings.TrimSpace(out.Artifacts[idx].Locator)
	}
	for i := range out.EvidenceRefs {
		ref := out.EvidenceRefs[i]
		if err := validateContextReference(ref); err != nil {
			return types.IsolateHandoffPayload{}, isolateHandoffValidationError{
				reason: isolateHandoffRejectEvidenceRefInvalid,
				cause:  fmt.Errorf("evidence_refs[%d]: %w", i, err),
			}
		}
		if _, ok := evidenceCatalog[ref.Locator]; !ok {
			return types.IsolateHandoffPayload{}, isolateHandoffValidationError{
				reason: isolateHandoffRejectEvidenceRefMissing,
				cause:  fmt.Errorf("evidence_refs[%d] locator not found: %s", i, ref.Locator),
			}
		}
	}
	return out, nil
}

func sanitizedHandoffForIngestion(in types.IsolateHandoffPayload) types.IsolateHandoffPayload {
	out := in
	if len(out.Artifacts) == 0 {
		return out
	}
	sanitized := make([]types.IsolateHandoffArtifact, 0, len(out.Artifacts))
	for _, artifact := range out.Artifacts {
		sanitized = append(sanitized, types.IsolateHandoffArtifact{
			ID:      strings.TrimSpace(artifact.ID),
			Type:    strings.TrimSpace(artifact.Type),
			Locator: strings.TrimSpace(artifact.Locator),
		})
	}
	out.Artifacts = sanitized
	return out
}

func isolateHandoffRejectReason(err error) string {
	var validationErr isolateHandoffValidationError
	if errors.As(err, &validationErr) {
		return validationErr.reason
	}
	return "invalid_payload"
}
