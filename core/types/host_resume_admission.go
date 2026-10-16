package types

import (
	"context"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	HostResumeAdmissionVersion  = "host_resume_admission.v1"
	HostResumeMaxReferenceBytes = 256
	HostResumeMaxDigestBytes    = 128
	HostResumeMaxEnvelopeBytes  = 32 * 1024
)

// HostResumeReason is deliberately finite. Callback errors and business bodies
// must never appear in an admission error, event, or transport response.
type HostResumeReason string

const (
	HostResumeInvalidBinding      HostResumeReason = "host_resume.invalid_binding"
	HostResumeBindingMismatch     HostResumeReason = "host_resume.binding_mismatch"
	HostResumeAuthorityDenied     HostResumeReason = "host_resume.authority_denied"
	HostResumeRevisionDrift       HostResumeReason = "host_resume.revision_drift"
	HostResumePlanDrift           HostResumeReason = "host_resume.plan_drift"
	HostResumeScopeDrift          HostResumeReason = "host_resume.scope_drift"
	HostResumeConfirmationInvalid HostResumeReason = "host_resume.confirmation_invalid"
	HostResumeEffectConflict      HostResumeReason = "host_resume.effect_conflict"
	HostResumeRevalidationFailed  HostResumeReason = "host_resume.revalidation_failed"
)

func (r HostResumeReason) Error() string { return string(r) }

func (r HostResumeReason) Valid() bool {
	switch r {
	case HostResumeInvalidBinding, HostResumeBindingMismatch, HostResumeAuthorityDenied,
		HostResumeRevisionDrift, HostResumePlanDrift, HostResumeScopeDrift,
		HostResumeConfirmationInvalid, HostResumeEffectConflict, HostResumeRevalidationFailed:
		return true
	}
	return false
}

// HostResumeRequest is a detached source snapshot, never an authority claim.
type HostResumeRequest struct {
	Version    string                `json:"version"`
	Checkpoint RunCheckpoint         `json:"checkpoint"`
	Decision   DynamicActionDecision `json:"decision"`
}

func (r HostResumeRequest) Detached() HostResumeRequest {
	r.Checkpoint.ToolResultRefs = append([]string(nil), r.Checkpoint.ToolResultRefs...)
	if r.Checkpoint.PendingAction != nil {
		a := *r.Checkpoint.PendingAction
		r.Checkpoint.PendingAction = &a
	}
	return r
}

func (r HostResumeRequest) Validate() error {
	if r.Version != HostResumeAdmissionVersion || !hostResumeEnvelope(r) || !hostResumeCheckpointValid(r.Checkpoint) || r.Decision.Validate() != nil {
		return HostResumeInvalidBinding
	}
	d := r.Decision.Normalized()
	for _, s := range []string{d.RunID, d.SessionID, d.CheckpointID, d.CheckpointVersion, d.Token, d.IdempotencyKey} {
		if !hostResumeString(s, HostResumeMaxReferenceBytes) {
			return HostResumeInvalidBinding
		}
	}
	if !hostResumeString(d.CheckpointDigest, HostResumeMaxDigestBytes) || (d.Reference != "" && !hostResumeString(d.Reference, HostResumeMaxReferenceBytes)) {
		return HostResumeInvalidBinding
	}
	if !hostResumeDecisionMatches(r.Checkpoint, d) {
		return HostResumeBindingMismatch
	}
	return nil
}

type HostResumePlan struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Digest  string `json:"digest"`
}

// HostResumeBinding is loaded from trusted paused application records by the
// verifier. A client-supplied digest does not establish this binding.
type HostResumeBinding struct {
	Version               string         `json:"version"`
	Checkpoint            RunCheckpoint  `json:"checkpoint"`
	Plan                  HostResumePlan `json:"plan"`
	BusinessRevision      string         `json:"business_revision"`
	AuthorityDigest       string         `json:"authority_digest"`
	ToolCapabilityDigest  string         `json:"tool_capability_digest"`
	WriteScopeDigest      string         `json:"write_scope_digest"`
	ConfirmationReference string         `json:"confirmation_reference"`
	EffectReference       string         `json:"effect_reference"`
}

// HostResumeFacts contains current host facts and actual authorization checks.
// It describes eligibility at admission; the business executor still needs CAS.
type HostResumeFacts struct {
	Plan                   HostResumePlan `json:"plan"`
	BusinessRevision       string         `json:"business_revision"`
	AuthorityDigest        string         `json:"authority_digest"`
	ToolCapabilityDigest   string         `json:"tool_capability_digest"`
	WriteScopeDigest       string         `json:"write_scope_digest"`
	ConfirmationReference  string         `json:"confirmation_reference"`
	EffectReference        string         `json:"effect_reference"`
	DecisionAuthorized     bool           `json:"decision_authorized"`
	ExecutionAuthorized    bool           `json:"execution_authorized"`
	CancellationAuthorized bool           `json:"cancellation_authorized"`
	ConfirmationValid      bool           `json:"confirmation_valid"`
	EffectEligible         bool           `json:"effect_eligible"`
}

type HostResumeResult struct {
	Version    string                   `json:"version"`
	Status     RunResumeAdmissionStatus `json:"status"`
	Binding    HostResumeBinding        `json:"binding"`
	Current    HostResumeFacts          `json:"current"`
	ReasonCode HostResumeReason         `json:"reason_code,omitempty"`
}

// HostResumeVerifier runs synchronously outside source locks. Implementations
// must load trusted records, remain read-only, and cooperate with ctx. They
// must not mutate the source or commit business effects from this callback.
type HostResumeVerifier interface {
	RevalidateHostResume(context.Context, HostResumeRequest) (HostResumeResult, error)
}

type HostResumeVerifierFunc func(context.Context, HostResumeRequest) (HostResumeResult, error)

func (f HostResumeVerifierFunc) RevalidateHostResume(ctx context.Context, r HostResumeRequest) (HostResumeResult, error) {
	if f == nil {
		return HostResumeResult{}, HostResumeRevalidationFailed
	}
	return f(ctx, r)
}

// ValidateHostResumeResult checks the bounded response and derives drift from
// the trusted paused binding and current facts. Accepted alone is insufficient.
func ValidateHostResumeResult(r HostResumeRequest, x HostResumeResult) error {
	if err := r.Validate(); err != nil {
		return err
	}
	b, c := x.Binding, x.Current
	if x.Version != HostResumeAdmissionVersion || b.Version != HostResumeAdmissionVersion || !hostResumeEnvelope(x) || !hostResumeCheckpointValid(b.Checkpoint) ||
		!hostResumeFieldsValid(b.Plan, b.BusinessRevision, b.AuthorityDigest, b.ToolCapabilityDigest, b.WriteScopeDigest, b.ConfirmationReference, b.EffectReference) ||
		!hostResumeFieldsValid(c.Plan, c.BusinessRevision, c.AuthorityDigest, c.ToolCapabilityDigest, c.WriteScopeDigest, c.ConfirmationReference, c.EffectReference) {
		return HostResumeInvalidBinding
	}
	if x.Status != RunResumeAdmissionAccepted && x.Status != RunResumeAdmissionRejected {
		return HostResumeInvalidBinding
	}
	if (x.Status == RunResumeAdmissionRejected && !x.ReasonCode.Valid()) || (x.Status == RunResumeAdmissionAccepted && x.ReasonCode != "") {
		return HostResumeInvalidBinding
	}
	source := r.Checkpoint
	if !hostResumeDecisionMatches(b.Checkpoint, r.Decision.Normalized()) || b.Checkpoint.Mode != source.Mode || b.Checkpoint.Iteration != source.Iteration || *b.Checkpoint.PendingAction != *source.PendingAction {
		return HostResumeBindingMismatch
	}
	if x.Status == RunResumeAdmissionRejected {
		return x.ReasonCode
	}
	if !c.DecisionAuthorized {
		return HostResumeAuthorityDenied
	}
	if b.ConfirmationReference != r.Decision.Normalized().Reference || c.ConfirmationReference != b.ConfirmationReference || !c.ConfirmationValid {
		return HostResumeConfirmationInvalid
	}
	if c.EffectReference != b.EffectReference || !c.EffectEligible {
		return HostResumeEffectConflict
	}
	if r.Decision.Decision != DynamicActionDecisionConfirm {
		if !c.CancellationAuthorized {
			return HostResumeAuthorityDenied
		}
		return nil
	}
	if !c.ExecutionAuthorized || c.AuthorityDigest != b.AuthorityDigest {
		return HostResumeAuthorityDenied
	}
	if c.BusinessRevision != b.BusinessRevision {
		return HostResumeRevisionDrift
	}
	if c.Plan != b.Plan {
		return HostResumePlanDrift
	}
	if c.ToolCapabilityDigest != b.ToolCapabilityDigest || c.WriteScopeDigest != b.WriteScopeDigest {
		return HostResumeScopeDrift
	}
	return nil
}

func hostResumeFieldsValid(p HostResumePlan, revision, authority, tools, scope, confirmation, effect string) bool {
	if p.Version <= 0 || !hostResumeString(p.ID, HostResumeMaxReferenceBytes) {
		return false
	}
	for _, s := range []string{revision, confirmation, effect} {
		if !hostResumeString(s, HostResumeMaxReferenceBytes) {
			return false
		}
	}
	for _, s := range []string{p.Digest, authority, tools, scope} {
		if !hostResumeString(s, HostResumeMaxDigestBytes) {
			return false
		}
	}
	return true
}

func hostResumeString(s string, limit int) bool {
	if s == "" || s != strings.TrimSpace(s) || len(s) > limit || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func hostResumeEnvelope(x any) bool {
	b, err := json.Marshal(x)
	return err == nil && len(b) <= HostResumeMaxEnvelopeBytes
}

func hostResumeCheckpointValid(c RunCheckpoint) bool {
	if c.Version != DynamicActionResumeProtocolVersion || c.Validate() != nil || (c.Mode != "run" && c.Mode != "stream") {
		return false
	}
	for _, s := range []string{c.RunID, c.SessionID, c.CheckpointID, c.Version} {
		if !hostResumeString(s, HostResumeMaxReferenceBytes) {
			return false
		}
	}
	if !hostResumeString(c.Digest, HostResumeMaxDigestBytes) {
		return false
	}
	a := c.PendingAction
	for _, s := range []string{a.Token, a.Kind, a.RunID, a.SessionID, a.CallID, a.Source} {
		if !hostResumeString(s, HostResumeMaxReferenceBytes) {
			return false
		}
	}
	if !a.Resumable || !hostResumeString(a.Digest, HostResumeMaxDigestBytes) {
		return false
	}
	for _, s := range c.ToolResultRefs {
		if !hostResumeString(s, HostResumeMaxReferenceBytes) {
			return false
		}
	}
	return true
}

func hostResumeDecisionMatches(c RunCheckpoint, d DynamicActionDecision) bool {
	return c.RunID == d.RunID && c.SessionID == d.SessionID && c.CheckpointID == d.CheckpointID && c.Version == d.CheckpointVersion && c.Digest == d.CheckpointDigest && c.PendingAction != nil && c.PendingAction.Token == d.Token
}
