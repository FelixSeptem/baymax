package evalcontract

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const ContinuityComparisonVersionV1 = "eval_continuity_comparison.v1"

const (
	ContinuityMaxFacts          = 128
	ContinuityMaxSerializedSize = 64 * 1024
)

const (
	ContinuityPhaseBaseline  = "baseline"
	ContinuityPhaseCandidate = "candidate"
)

const (
	ContinuityKindIdentityAgent    = "identity.agent"
	ContinuityKindIdentityRole     = "identity.role"
	ContinuityKindIdentityTeam     = "identity.team"
	ContinuityKindObjective        = "objective"
	ContinuityKindTask             = "task"
	ContinuityKindAttempt          = "attempt"
	ContinuityKindLease            = "lease"
	ContinuityKindWorkspaceBinding = "workspace_binding"
	ContinuityKindPendingRequest   = "pending_request"
	ContinuityKindCheckpoint       = "checkpoint"
	ContinuityKindArtifact         = "artifact"
)

const (
	ReasonContinuitySchemaDrift              = "continuity_schema_drift"
	ReasonContinuityIdentityDrift            = "continuity_identity_drift"
	ReasonContinuityObjectiveDrift           = "continuity_objective_drift"
	ReasonContinuityTaskAssociationDrift     = "continuity_task_association_drift"
	ReasonContinuityAttemptLeaseDrift        = "continuity_attempt_lease_drift"
	ReasonContinuityWorkspaceBindingDrift    = "continuity_workspace_binding_drift"
	ReasonContinuityPendingRequestDrift      = "continuity_pending_request_drift"
	ReasonContinuityCheckpointDrift          = "continuity_checkpoint_drift"
	ReasonContinuityArtifactReferenceDrift   = "continuity_artifact_reference_drift"
	ReasonContinuityOwnerDrift               = "continuity_owner_drift"
	ReasonContinuityReferenceMissing         = "continuity_reference_missing"
	ReasonContinuityDuplicateReference       = "continuity_duplicate_reference"
	ReasonContinuityPrivacyViolation         = "continuity_privacy_violation"
	ReasonContinuityRunStreamParityDrift     = "continuity_run_stream_parity_drift"
	ReasonContinuityRecoveryIdempotencyDrift = "continuity_recovery_idempotency_drift"
)

type ContinuityFact struct {
	Kind     string `json:"kind"`
	Owner    string `json:"owner"`
	ID       string `json:"id"`
	Digest   string `json:"digest,omitempty"`
	Version  string `json:"version,omitempty"`
	Required bool   `json:"required,omitempty"`
	Body     string `json:"body,omitempty"`
}

type ContinuityProjection struct {
	Version     string           `json:"version"`
	RunID       string           `json:"run_id"`
	SessionID   string           `json:"session_id,omitempty"`
	Phase       string           `json:"phase"`
	Mode        string           `json:"mode,omitempty"`
	OperationID string           `json:"operation_id,omitempty"`
	StateDigest string           `json:"state_digest,omitempty"`
	Facts       []ContinuityFact `json:"facts"`
}

type ContinuityDrift struct {
	Class          string `json:"class"`
	Kind           string `json:"kind"`
	Key            string `json:"key"`
	BaselineOwner  string `json:"baseline_owner,omitempty"`
	CandidateOwner string `json:"candidate_owner,omitempty"`
}

type ContinuityComparison struct {
	Version   string               `json:"version"`
	ID        string               `json:"id"`
	Baseline  ContinuityProjection `json:"baseline"`
	Candidate ContinuityProjection `json:"candidate"`
	Passed    bool                 `json:"passed"`
	Drifts    []ContinuityDrift    `json:"drifts,omitempty"`
}

var continuityKinds = map[string]struct{}{
	ContinuityKindIdentityAgent: {}, ContinuityKindIdentityRole: {}, ContinuityKindIdentityTeam: {},
	ContinuityKindObjective: {}, ContinuityKindTask: {}, ContinuityKindAttempt: {}, ContinuityKindLease: {},
	ContinuityKindWorkspaceBinding: {}, ContinuityKindPendingRequest: {}, ContinuityKindCheckpoint: {}, ContinuityKindArtifact: {},
}

func NormalizeContinuityProjection(in ContinuityProjection) (ContinuityProjection, string, error) {
	in.Version = strings.TrimSpace(strings.ToLower(in.Version))
	in.RunID = strings.TrimSpace(in.RunID)
	in.SessionID = strings.TrimSpace(in.SessionID)
	in.Phase = strings.TrimSpace(strings.ToLower(in.Phase))
	if in.Version != ContinuityComparisonVersionV1 || in.RunID == "" || (in.Phase != ContinuityPhaseBaseline && in.Phase != ContinuityPhaseCandidate) || len(in.Facts) == 0 || len(in.Facts) > ContinuityMaxFacts {
		return ContinuityProjection{}, "", fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	in.Mode = strings.TrimSpace(strings.ToLower(in.Mode))
	in.OperationID = strings.TrimSpace(in.OperationID)
	in.StateDigest = strings.TrimSpace(in.StateDigest)
	if in.Mode != "" && in.Mode != "run" && in.Mode != "stream" {
		return ContinuityProjection{}, "", fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	seen := make(map[string]struct{}, len(in.Facts))
	for i := range in.Facts {
		f := &in.Facts[i]
		f.Kind, f.Owner, f.ID = strings.TrimSpace(strings.ToLower(f.Kind)), strings.TrimSpace(f.Owner), strings.TrimSpace(f.ID)
		f.Digest, f.Version = strings.TrimSpace(f.Digest), strings.TrimSpace(f.Version)
		if f.Body != "" {
			return ContinuityProjection{}, "", fmt.Errorf("%s", ReasonContinuityPrivacyViolation)
		}
		if _, ok := continuityKinds[f.Kind]; !ok || f.Owner == "" || f.ID == "" {
			return ContinuityProjection{}, "", fmt.Errorf("%s", ReasonContinuitySchemaDrift)
		}
		key := f.Kind + "\x00" + f.ID
		if _, ok := seen[key]; ok {
			return ContinuityProjection{}, "", fmt.Errorf("%s", ReasonContinuityDuplicateReference)
		}
		seen[key] = struct{}{}
	}
	serialized, err := json.Marshal(in)
	if err != nil {
		return ContinuityProjection{}, "", err
	}
	if len(serialized) > ContinuityMaxSerializedSize {
		return ContinuityProjection{}, "", fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	sort.Slice(in.Facts, func(i, j int) bool {
		if in.Facts[i].Kind == in.Facts[j].Kind {
			return in.Facts[i].ID < in.Facts[j].ID
		}
		return in.Facts[i].Kind < in.Facts[j].Kind
	})
	digest, err := digestValue(in)
	return in, digest, err
}

func CompareContinuity(baseline, candidate ContinuityProjection) (ContinuityComparison, error) {
	baseline.Phase = ContinuityPhaseBaseline
	candidate.Phase = ContinuityPhaseCandidate
	base, _, err := NormalizeContinuityProjection(baseline)
	if err != nil {
		return ContinuityComparison{}, err
	}
	got, _, err := NormalizeContinuityProjection(candidate)
	if err != nil {
		return ContinuityComparison{}, err
	}
	if base.RunID != got.RunID || (base.SessionID != "" && got.SessionID != "" && base.SessionID != got.SessionID) {
		return ContinuityComparison{}, fmt.Errorf("%s", ReasonContinuityIdentityDrift)
	}
	if base.Mode != "" && got.Mode != "" && base.Mode != got.Mode {
		return ContinuityComparison{}, fmt.Errorf("%s", ReasonContinuityRunStreamParityDrift)
	}
	if base.OperationID != "" && got.OperationID != "" && base.OperationID == got.OperationID && base.StateDigest != got.StateDigest {
		return ContinuityComparison{}, fmt.Errorf("%s", ReasonContinuityRecoveryIdempotencyDrift)
	}
	byKey := func(facts []ContinuityFact) map[string]ContinuityFact {
		m := make(map[string]ContinuityFact, len(facts))
		for _, f := range facts {
			m[f.Kind+"\x00"+f.ID] = f
		}
		return m
	}
	bm, cm := byKey(base.Facts), byKey(got.Facts)
	result := ContinuityComparison{Version: ContinuityComparisonVersionV1, Baseline: base, Candidate: got}
	// When a semantic axis has one reference on each side but its stable ID
	// changes, report an axis drift rather than two opaque missing references.
	byKind := func(facts []ContinuityFact) map[string][]ContinuityFact {
		m := map[string][]ContinuityFact{}
		for _, f := range facts {
			m[f.Kind] = append(m[f.Kind], f)
		}
		return m
	}
	bk, ck := byKind(base.Facts), byKind(got.Facts)
	for kind, left := range bk {
		if len(left) == 1 && len(ck[kind]) == 1 && left[0].ID != ck[kind][0].ID {
			result.Drifts = append(result.Drifts, ContinuityDrift{Class: continuityDriftClass(kind), Kind: kind, Key: left[0].ID})
			delete(bm, kind+"\x00"+left[0].ID)
			delete(cm, kind+"\x00"+ck[kind][0].ID)
		}
	}
	keys := make([]string, 0, len(bm)+len(cm))
	all := map[string]struct{}{}
	for k := range bm {
		all[k] = struct{}{}
	}
	for k := range cm {
		all[k] = struct{}{}
	}
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b, bok := bm[key]
		c, cok := cm[key]
		if !bok || !cok {
			result.Drifts = append(result.Drifts, ContinuityDrift{Class: ReasonContinuityReferenceMissing, Kind: factKind(key), Key: factID(key)})
			continue
		}
		if b.Owner != c.Owner {
			result.Drifts = append(result.Drifts, ContinuityDrift{Class: ReasonContinuityOwnerDrift, Kind: b.Kind, Key: b.ID, BaselineOwner: b.Owner, CandidateOwner: c.Owner})
			continue
		}
		if b.Digest == c.Digest && b.Version == c.Version && b.Required == c.Required {
			continue
		}
		result.Drifts = append(result.Drifts, ContinuityDrift{Class: continuityDriftClass(b.Kind), Kind: b.Kind, Key: b.ID})
	}
	sort.Slice(result.Drifts, func(i, j int) bool {
		if result.Drifts[i].Class == result.Drifts[j].Class {
			if result.Drifts[i].Kind == result.Drifts[j].Kind {
				return result.Drifts[i].Key < result.Drifts[j].Key
			}
			return result.Drifts[i].Kind < result.Drifts[j].Kind
		}
		return result.Drifts[i].Class < result.Drifts[j].Class
	})
	result.Passed = len(result.Drifts) == 0
	result.ID, err = digestValue(struct {
		Version, RunID, SessionID string
		Drifts                    []ContinuityDrift
	}{ContinuityComparisonVersionV1, base.RunID, base.SessionID, result.Drifts})
	return result, err
}

func factKind(key string) string {
	if i := strings.IndexByte(key, 0); i >= 0 {
		return key[:i]
	}
	return key
}
func factID(key string) string {
	if i := strings.IndexByte(key, 0); i >= 0 {
		return key[i+1:]
	}
	return key
}
func continuityDriftClass(kind string) string {
	switch kind {
	case ContinuityKindIdentityAgent, ContinuityKindIdentityRole, ContinuityKindIdentityTeam:
		return ReasonContinuityIdentityDrift
	case ContinuityKindObjective:
		return ReasonContinuityObjectiveDrift
	case ContinuityKindTask:
		return ReasonContinuityTaskAssociationDrift
	case ContinuityKindAttempt, ContinuityKindLease:
		return ReasonContinuityAttemptLeaseDrift
	case ContinuityKindWorkspaceBinding:
		return ReasonContinuityWorkspaceBindingDrift
	case ContinuityKindPendingRequest:
		return ReasonContinuityPendingRequestDrift
	case ContinuityKindCheckpoint:
		return ReasonContinuityCheckpointDrift
	case ContinuityKindArtifact:
		return ReasonContinuityArtifactReferenceDrift
	default:
		return ReasonContinuitySchemaDrift
	}
}
