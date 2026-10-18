package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/FelixSeptem/baymax/context/hostprojection"
)

const HostContextProjectionReplayVersion = "host_context_projection_replay.v1"

type HostContextProjectionFixture struct {
	Version string                               `json:"version"`
	Cases   []HostContextProjectionReplayFixture `json:"cases"`
}

type HostContextProjectionReplayFixture struct {
	CaseID                    string                    `json:"case_id"`
	Descriptor                hostprojection.Descriptor `json:"descriptor"`
	CanonicalizeFixtureDigest bool                      `json:"canonicalize_fixture_digest,omitempty"`
	Authorization             string                    `json:"authorization,omitempty"`
	SourceAvailability        string                    `json:"source_availability,omitempty"`
	Restore                   string                    `json:"restore,omitempty"`
	SideEffectBeforeRejection bool                      `json:"side_effect_before_rejection,omitempty"`
	ExpectedReason            string                    `json:"expected_reason,omitempty"`
	Run                       string                    `json:"run"`
	Stream                    string                    `json:"stream"`
}

type HostContextProjectionReplayCase struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason,omitempty"`
	Digest string `json:"digest,omitempty"`
}

type HostContextProjectionReplayResult struct {
	Version string                            `json:"version"`
	Cases   []HostContextProjectionReplayCase `json:"cases"`
}

func (r HostContextProjectionReplayResult) Equal(other HostContextProjectionReplayResult) bool {
	return reflectHostContextReplayResult(r) == reflectHostContextReplayResult(other)
}

func reflectHostContextReplayResult(result HostContextProjectionReplayResult) string {
	raw, _ := json.Marshal(result)
	return string(raw)
}

// ReplayHostContextProjectionFixtureJSON is deterministic and read-only. It
// does not instantiate a resolver or runtime owner, so replay cannot read a
// source body, call a provider/tool/network, or alter RuntimeRecorder state.
func ReplayHostContextProjectionFixtureJSON(raw []byte) (HostContextProjectionReplayResult, error) {
	if len(raw) == 0 || len(raw) > 2<<20 || bytes.Contains(bytes.ToLower(raw), []byte(`"body"`)) {
		return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.replay_schema_drift")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var fixture HostContextProjectionFixture
	if err := decoder.Decode(&fixture); err != nil || decoder.Decode(new(any)) != io.EOF {
		return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.replay_schema_drift")
	}
	if fixture.Version != HostContextProjectionReplayVersion || len(fixture.Cases) == 0 || len(fixture.Cases) > 256 {
		return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.replay_schema_drift")
	}
	result := HostContextProjectionReplayResult{Version: fixture.Version, Cases: make([]HostContextProjectionReplayCase, 0, len(fixture.Cases))}
	seen := make(map[string]struct{}, len(fixture.Cases))
	for _, item := range fixture.Cases {
		id := strings.TrimSpace(item.CaseID)
		if id == "" || item.Run != item.Stream || (item.Run != "admitted" && item.Run != "rejected") {
			return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.run_stream_mismatch")
		}
		if _, exists := seen[id]; exists {
			return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.replay_schema_drift")
		}
		seen[id] = struct{}{}
		descriptor := item.Descriptor
		if item.CanonicalizeFixtureDigest {
			descriptor.Digest = ""
			digest, digestErr := hostprojection.CanonicalDigest(descriptor)
			if digestErr != nil {
				return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.replay_schema_drift")
			}
			descriptor.Digest = digest
		}
		if err := validateReplayReferenceOnly(item, descriptor); err != nil {
			return HostContextProjectionReplayResult{}, err
		}
		admitted, err := hostprojection.Admit(descriptor)
		caseResult := HostContextProjectionReplayCase{CaseID: id}
		if err != nil {
			classified, ok := err.(*hostprojection.Error)
			if !ok {
				return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.replay_schema_drift")
			}
			caseResult.Reason = string(classified.Code)
			if item.Run != "rejected" {
				return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.run_stream_mismatch")
			}
		} else {
			caseResult.Digest = admitted.Descriptor().Digest
			caseResult.Reason, err = declaredReplayReason(item)
			if err != nil {
				return HostContextProjectionReplayResult{}, err
			}
		}
		if item.ExpectedReason != "" && item.ExpectedReason != caseResult.Reason {
			return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.replay_schema_drift")
		}
		if (caseResult.Reason == "" && item.Run != "admitted") || (caseResult.Reason != "" && item.Run != "rejected") {
			return HostContextProjectionReplayResult{}, fmt.Errorf("host_context.run_stream_mismatch")
		}
		result.Cases = append(result.Cases, caseResult)
	}
	return result, nil
}

func validateReplayReferenceOnly(item HostContextProjectionReplayFixture, descriptor hostprojection.Descriptor) error {
	for _, reference := range descriptor.SourceReferences {
		if !strings.HasPrefix(reference.Reference, "opaque://") || strings.Contains(strings.ToLower(reference.Reference), "body") || strings.Contains(strings.ToLower(reference.Digest), "body") {
			return fmt.Errorf("host_context.privacy_leakage")
		}
	}
	for _, value := range []string{item.Authorization, item.SourceAvailability, item.Restore, item.ExpectedReason} {
		if strings.Contains(strings.ToLower(value), "body") {
			return fmt.Errorf("host_context.privacy_leakage")
		}
	}
	return nil
}

func declaredReplayReason(item HostContextProjectionReplayFixture) (string, error) {
	if item.SideEffectBeforeRejection {
		return "host_context.side_effect_before_rejection", nil
	}
	switch item.Authorization {
	case "", "current":
	case "revoked", "denied":
		return string(hostprojection.CodeResolverDenied), nil
	default:
		return "", fmt.Errorf("host_context.replay_schema_drift")
	}
	switch item.SourceAvailability {
	case "", "available":
	case "missing":
		return string(hostprojection.CodeSourceNotFound), nil
	default:
		return "", fmt.Errorf("host_context.replay_schema_drift")
	}
	switch item.Restore {
	case "", "not_requested", "restored", "rejected":
		return "", nil
	default:
		return "", fmt.Errorf("host_context.replay_schema_drift")
	}
}
