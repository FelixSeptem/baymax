package mailbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const MailboxContinuationVersion = "mailbox_continuation.v1"

var (
	ErrMailboxContinuationMalformed        = errors.New("mailbox continuation malformed")
	ErrMailboxContinuationCorrupt          = errors.New("mailbox continuation corrupt")
	ErrMailboxContinuationForeign          = errors.New("mailbox continuation foreign")
	ErrMailboxContinuationExpired          = errors.New("mailbox continuation expired")
	ErrMailboxContinuationUnavailable      = errors.New("mailbox continuation unavailable")
	ErrMailboxContinuationConflict         = errors.New("mailbox continuation conflict")
	ErrMailboxContinuationInProgress       = errors.New("mailbox continuation in progress")
	ErrMailboxContinuationRecoveryRequired = errors.New("mailbox continuation recovery required")
	ErrMailboxContinuationLeaseLost        = errors.New("mailbox continuation lease lost")
	ErrMailboxContinuationUnsupported      = errors.New("mailbox continuation unsupported")
)

type MailboxPauseReason string

const (
	MailboxPauseInputRequired    MailboxPauseReason = "input_required"
	MailboxPauseApprovalRequired MailboxPauseReason = "approval_required"
	MailboxPauseBudgetExhausted  MailboxPauseReason = "budget_exhausted"
)

type MailboxContinuationRef struct {
	SchemaVersion    string                `json:"schema_version"`
	CheckpointID     string                `json:"checkpoint_id"`
	Digest           string                `json:"digest"`
	OwnerNamespace   string                `json:"owner_namespace"`
	MessageID        string                `json:"message_id"`
	DeliveryID       string                `json:"delivery_id"`
	LeaseID          string                `json:"lease_id"`
	AttemptCausation string                `json:"attempt_causation"`
	RunID            string                `json:"run_id"`
	ParentRunID      string                `json:"parent_run_id"`
	CausationID      string                `json:"causation_id"`
	Deployment       types.DeploymentScope `json:"deployment,omitempty"`
}

type MailboxContinuation struct {
	Ref                     MailboxContinuationRef `json:"ref"`
	Binding                 HostPlanBinding        `json:"binding"`
	MessageID               string                 `json:"message_id"`
	DeliveryID              string                 `json:"delivery_id"`
	LeaseID                 string                 `json:"lease_id"`
	AttemptCausation        string                 `json:"attempt_causation"`
	DeliveryAttempt         int                    `json:"delivery_attempt"`
	LifecycleState          MessageState           `json:"lifecycle_state"`
	ParentRunID             string                 `json:"parent_run_id"`
	RunID                   string                 `json:"run_id"`
	HostCorrelation         string                 `json:"host_correlation"`
	AuthorityDigest         string                 `json:"authority_digest"`
	PlanDigest              string                 `json:"plan_digest"`
	WriteScope              string                 `json:"write_scope"`
	EffectNamespace         string                 `json:"effect_namespace"`
	Occurrence              string                 `json:"occurrence"`
	PauseReason             MailboxPauseReason     `json:"pause_reason"`
	PendingActionReference  string                 `json:"pending_action_reference,omitempty"`
	CompletedEffectRefs     []string               `json:"completed_effect_refs,omitempty"`
	ConsumedBudget          int64                  `json:"consumed_budget,omitempty"`
	OriginalDeadline        time.Time              `json:"original_deadline,omitempty"`
	PredecessorCheckpointID string                 `json:"predecessor_checkpoint_id,omitempty"`
	SuccessorCheckpointID   string                 `json:"successor_checkpoint_id,omitempty"`
	Generation              uint64                 `json:"generation"`
	CreatedAt               time.Time              `json:"created_at"`
	ExpiresAt               time.Time              `json:"expires_at"`
}

type MailboxResumeRequest struct {
	Ref            MailboxContinuationRef  `json:"ref"`
	Binding        HostPlanBinding         `json:"binding"`
	IdempotencyKey string                  `json:"idempotency_key"`
	Generation     uint64                  `json:"generation"`
	PauseReason    MailboxPauseReason      `json:"pause_reason"`
	Deployment     types.DeploymentRequest `json:"deployment,omitempty"`
}

type MailboxResumeClaim struct {
	Request    MailboxResumeRequest `json:"request"`
	AttemptID  string               `json:"attempt_id"`
	Generation uint64               `json:"generation"`
	State      string               `json:"state"`
	Duplicate  bool                 `json:"duplicate,omitempty"`
	Result     *HostPlanOutcome     `json:"result,omitempty"`
}

type MailboxContinuationRecord struct {
	Continuation MailboxContinuation `json:"continuation"`
	Claim        *MailboxResumeClaim `json:"claim,omitempty"`
	Identity     string              `json:"identity,omitempty"`
	Recovered    bool                `json:"recovered,omitempty"`
}

func validMailboxPauseReason(r MailboxPauseReason) bool {
	return r == MailboxPauseInputRequired || r == MailboxPauseApprovalRequired || r == MailboxPauseBudgetExhausted
}
func (r MailboxContinuationRef) Validate() error {
	if r.SchemaVersion != MailboxContinuationVersion || r.CheckpointID == "" || r.Digest == "" || r.OwnerNamespace == "" || r.MessageID == "" || r.DeliveryID == "" || r.LeaseID == "" || r.AttemptCausation == "" || r.RunID == "" {
		return ErrMailboxContinuationMalformed
	}
	return nil
}
func (c MailboxContinuation) Detached() MailboxContinuation {
	out := c
	out.CompletedEffectRefs = append([]string(nil), c.CompletedEffectRefs...)
	return out
}
func SealMailboxContinuation(c MailboxContinuation) (MailboxContinuation, error) {
	c = c.Detached()
	c.Ref.SchemaVersion = MailboxContinuationVersion
	c.Ref.MessageID = c.MessageID
	c.Ref.DeliveryID = c.DeliveryID
	c.Ref.LeaseID = c.LeaseID
	c.Ref.AttemptCausation = c.AttemptCausation
	c.Ref.RunID = c.RunID
	c.Ref.ParentRunID = c.ParentRunID
	c.Ref.CausationID = c.Binding.CausationID
	c.Ref.CheckpointID = "pending"
	c.Ref.Digest = "pending"
	if err := c.validateShape(); err != nil {
		return MailboxContinuation{}, err
	}
	c.Ref.CheckpointID = ""
	c.Ref.Digest = ""
	b, _ := json.Marshal(c)
	h := sha256.Sum256(b)
	c.Ref.CheckpointID = "mailbox-" + hex.EncodeToString(h[:16])
	c.Ref.Digest = "sha256:" + hex.EncodeToString(h[:])
	if err := c.Validate(); err != nil {
		return MailboxContinuation{}, err
	}
	return c, nil
}
func (c MailboxContinuation) Validate() error {
	if err := c.validateShape(); err != nil {
		return err
	}
	x := c.Detached()
	x.Ref.CheckpointID = ""
	x.Ref.Digest = ""
	b, _ := json.Marshal(x)
	h := sha256.Sum256(b)
	if c.Ref.CheckpointID != "mailbox-"+hex.EncodeToString(h[:16]) || c.Ref.Digest != "sha256:"+hex.EncodeToString(h[:]) {
		return ErrMailboxContinuationCorrupt
	}
	return nil
}
func (c MailboxContinuation) validateShape() error {
	if err := c.Ref.Validate(); err != nil {
		return err
	}
	if err := normalizeHostPlanBindingCheck(c.Binding); err != nil {
		return err
	}
	if c.MessageID == "" || c.DeliveryID == "" || c.LeaseID == "" || c.AttemptCausation == "" || c.RunID == "" || c.ParentRunID == "" || c.HostCorrelation == "" || c.AuthorityDigest == "" || c.PlanDigest == "" || c.WriteScope == "" || c.EffectNamespace == "" || c.Occurrence == "" || !validMailboxPauseReason(c.PauseReason) || c.Generation == 0 || c.CreatedAt.IsZero() || !c.ExpiresAt.After(c.CreatedAt) || c.ExpiresAt.Sub(c.CreatedAt) > 24*time.Hour {
		return ErrMailboxContinuationMalformed
	}
	if c.Ref.MessageID != c.MessageID || c.Ref.DeliveryID != c.DeliveryID || c.Ref.LeaseID != c.LeaseID || c.Ref.AttemptCausation != c.AttemptCausation || c.Ref.RunID != c.RunID || c.Ref.ParentRunID != c.ParentRunID {
		return ErrMailboxContinuationForeign
	}
	b, err := json.Marshal(c)
	if err != nil || len(b) > 64*1024 {
		return ErrMailboxContinuationMalformed
	}
	return nil
}
func normalizeHostPlanBindingCheck(b HostPlanBinding) error {
	_, err := normalizeHostPlanBinding(b)
	return err
}
func (r MailboxResumeRequest) Validate() error {
	if err := r.Ref.Validate(); err != nil {
		return err
	}
	if err := normalizeHostPlanBindingCheck(r.Binding); err != nil {
		return err
	}
	if r.IdempotencyKey == "" || r.Generation == 0 || !validMailboxPauseReason(r.PauseReason) {
		return ErrMailboxContinuationMalformed
	}
	return nil
}
func (r MailboxResumeRequest) identity() string {
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func cloneMailboxContinuationRecords(in map[string]MailboxContinuationRecord) map[string]MailboxContinuationRecord {
	if len(in) == 0 {
		return map[string]MailboxContinuationRecord{}
	}
	out := make(map[string]MailboxContinuationRecord, len(in))
	for k, v := range in {
		v.Continuation = v.Continuation.Detached()
		if v.Claim != nil {
			x := *v.Claim
			if v.Claim.Result != nil {
				y := *v.Claim.Result
				x.Result = &y
			}
			v.Claim = &x
		}
		out[k] = v
	}
	return out
}

type continuationStore interface {
	continuationSnapshot(context.Context, func(Snapshot) error) error
	continuationMutate(context.Context, func(*mailboxState) error) error
}

func (m *Mailbox) continuationSnapshot(ctx context.Context, fn func(Snapshot) error) error {
	owner, ok := m.store.(continuationStore)
	if !ok {
		return ErrMailboxContinuationUnsupported
	}
	return owner.continuationSnapshot(ctx, fn)
}

func (m *Mailbox) continuationMutate(ctx context.Context, fn func(*mailboxState) error) error {
	owner, ok := m.store.(continuationStore)
	if !ok {
		return ErrMailboxContinuationUnsupported
	}
	return owner.continuationMutate(ctx, fn)
}

func (m *Mailbox) ContinuationCapabilities(context.Context) (types.ContinuationCapabilities, error) {
	if m == nil || m.store == nil {
		return types.ContinuationCapabilities{}, ErrMailboxContinuationUnavailable
	}
	_, durable := m.store.(*FileStore)
	scope := types.DefaultDeploymentScope("mailbox", "mailbox continuation store is process-local")
	if durable {
		scope = types.SameHostSingleWriterScope("mailbox", "fixture:mailbox-continuation.v1")
	}
	caps := types.ContinuationCapabilities{
		ProfileVersion: types.NativeContinuationVersion, SourceNamespace: "mailbox", ContractVersion: "baymax.runtime.contract.v1", ModuleRevision: "baymax", APIVersion: "host_runtime_capability_manifest.v1", RetentionPolicy: "source-owned-mailbox-checkpoint", ExpiryPolicy: "checkpoint-expiry", CheckpointCreation: true, CrossProcessRestore: durable,
		EvidenceReferences: map[string]string{"checkpoint_creation": "fixture:mailbox-continuation.v1", "cross_process_restore": "fixture:mailbox-continuation.v1"},
		PauseReasonEvidence: map[string]types.PauseReasonCapability{
			"input_required":    {UnsupportedReason: types.UnsupportedReasonMailboxHandlerSafePoint},
			"approval_required": {UnsupportedReason: types.UnsupportedReasonMailboxHandlerSafePoint},
			"budget_exhausted":  {UnsupportedReason: types.UnsupportedReasonMailboxHandlerSafePoint},
		},
		Deployment: scope,
	}
	if !durable {
		caps.UnsupportedReasons = map[string]string{"cross_process_restore": "mailbox continuation store is process-local"}
	}
	return caps, nil
}

func (m *Mailbox) SaveHostPlanContinuation(ctx context.Context, continuation MailboxContinuation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil || m.store == nil {
		return ErrMailboxContinuationUnavailable
	}
	if err := continuation.Validate(); err != nil {
		return err
	}
	m.continuationMu.Lock()
	defer m.continuationMu.Unlock()
	err := m.continuationMutate(ctx, func(state *mailboxState) error {
		if state.Continuations == nil {
			state.Continuations = map[string]MailboxContinuationRecord{}
		}
		if old, ok := state.Continuations[continuation.Ref.CheckpointID]; ok {
			if old.Continuation.Ref.Digest == continuation.Ref.Digest {
				return nil
			}
			return ErrMailboxContinuationConflict
		}
		current, ok := snapshotRecord(state.snapshot(), continuation.MessageID)
		if !ok {
			return ErrMailboxContinuationUnavailable
		}
		if current.DeliveryID != continuation.DeliveryID || current.LeaseID != continuation.LeaseID || current.AttemptCausation != continuation.AttemptCausation || current.State != StateInFlight {
			return ErrMailboxContinuationLeaseLost
		}
		state.Continuations[continuation.Ref.CheckpointID] = MailboxContinuationRecord{Continuation: continuation.Detached()}
		return nil
	})
	if errors.Is(err, ErrMailboxContinuationUnsupported) {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func (m *Mailbox) LoadHostPlanContinuation(ctx context.Context, ref MailboxContinuationRef) (MailboxContinuation, error) {
	if err := ctx.Err(); err != nil {
		return MailboxContinuation{}, err
	}
	if m == nil || m.store == nil {
		return MailboxContinuation{}, ErrMailboxContinuationUnavailable
	}
	if err := ref.Validate(); err != nil {
		return MailboxContinuation{}, err
	}
	m.continuationMu.Lock()
	defer m.continuationMu.Unlock()
	var loaded MailboxContinuation
	err := m.continuationSnapshot(ctx, func(snapshot Snapshot) error {
		record, ok := snapshot.Continuations[ref.CheckpointID]
		if !ok {
			return ErrMailboxContinuationUnavailable
		}
		if record.Continuation.Ref.Digest != ref.Digest {
			return ErrMailboxContinuationForeign
		}
		if !m.nowTime().Before(record.Continuation.ExpiresAt) {
			return ErrMailboxContinuationExpired
		}
		if err := record.Continuation.Validate(); err != nil {
			return err
		}
		loaded = record.Continuation.Detached()
		return nil
	})
	if err != nil {
		return MailboxContinuation{}, err
	}
	return loaded, nil
}

func (m *Mailbox) ClaimHostPlanResume(ctx context.Context, req MailboxResumeRequest) (MailboxResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return MailboxResumeClaim{}, err
	}
	if m == nil || m.store == nil {
		return MailboxResumeClaim{}, ErrMailboxContinuationUnavailable
	}
	if err := req.Validate(); err != nil {
		return MailboxResumeClaim{}, err
	}
	m.continuationMu.Lock()
	defer m.continuationMu.Unlock()
	var claim MailboxResumeClaim
	err := m.continuationMutate(ctx, func(state *mailboxState) error {
		record, ok := state.Continuations[req.Ref.CheckpointID]
		if !ok {
			return ErrMailboxContinuationUnavailable
		}
		continuation := record.Continuation
		if continuation.Ref.Digest != req.Ref.Digest || continuation.Binding != req.Binding || continuation.Generation != req.Generation || continuation.PauseReason != req.PauseReason {
			return ErrMailboxContinuationConflict
		}
		if !m.nowTime().Before(continuation.ExpiresAt) {
			return ErrMailboxContinuationExpired
		}
		if !continuation.OriginalDeadline.IsZero() && !m.nowTime().Before(continuation.OriginalDeadline) {
			return ErrMailboxContinuationRecoveryRequired
		}
		current, ok := snapshotRecord(state.snapshot(), continuation.MessageID)
		if !ok {
			return ErrMailboxContinuationUnavailable
		}
		if current.DeliveryID != continuation.DeliveryID || current.LeaseID != continuation.LeaseID || current.AttemptCausation != continuation.AttemptCausation || current.State != StateInFlight {
			return ErrMailboxContinuationLeaseLost
		}
		identity := req.identity()
		if record.Claim != nil {
			if record.Identity != identity {
				return ErrMailboxContinuationConflict
			}
			claim = *record.Claim
			claim.Duplicate = true
			if claim.Result != nil {
				return nil
			}
			if record.Recovered {
				return ErrMailboxContinuationRecoveryRequired
			}
			return ErrMailboxContinuationInProgress
		}
		claim = MailboxResumeClaim{Request: req, AttemptID: req.Ref.CheckpointID + ":attempt", Generation: continuation.Generation + 1, State: "claimed"}
		record.Claim = &claim
		record.Identity = identity
		state.Continuations[req.Ref.CheckpointID] = record
		return nil
	})
	return claim, err
}

func (m *Mailbox) MarkHostPlanResumeRunning(ctx context.Context, claim MailboxResumeClaim) (MailboxResumeClaim, error) {
	return m.mutateHostPlanClaim(ctx, claim, func(record *MailboxContinuationRecord) error {
		if record.Recovered {
			return ErrMailboxContinuationRecoveryRequired
		}
		if record.Claim.State == "running" {
			return nil
		}
		if record.Claim.State != "claimed" {
			return ErrMailboxContinuationConflict
		}
		record.Claim.State = "running"
		return nil
	})
}
func (m *Mailbox) CommitHostPlanResumeResult(ctx context.Context, claim MailboxResumeClaim, result HostPlanOutcome) (MailboxResumeClaim, error) {
	return m.mutateHostPlanClaimResult(ctx, claim, result)
}

// ResumeHostPlanContinuation claims the existing delivery and returns its
// current source projection. It does not publish, consume, acknowledge,
// requeue, retry, or execute a host effect; the existing mailbox owner remains
// responsible for the subsequent lifecycle transition.
func (m *Mailbox) ResumeHostPlanContinuation(ctx context.Context, req MailboxResumeRequest) (HostPlanOutcome, error) {
	claim, err := m.ClaimHostPlanResume(ctx, req)
	if err != nil {
		if claim.Result != nil {
			return *claim.Result, nil
		}
		return HostPlanOutcome{}, err
	}
	if claim.Duplicate && claim.Result != nil {
		return *claim.Result, nil
	}
	if _, err := m.MarkHostPlanResumeRunning(ctx, claim); err != nil {
		return HostPlanOutcome{}, err
	}
	page, err := m.Query(ctx, QueryRequest{MessageID: req.Ref.MessageID})
	if err != nil || len(page.Items) != 1 {
		if err != nil {
			return HostPlanOutcome{}, err
		}
		return HostPlanOutcome{}, ErrMailboxContinuationUnavailable
	}
	outcome, err := HostPlanOutcomeFromRecord(page.Items[0])
	if err != nil {
		return HostPlanOutcome{}, err
	}
	if _, err := m.CommitHostPlanResumeResult(ctx, claim, outcome); err != nil {
		return HostPlanOutcome{}, err
	}
	return outcome, nil
}
func (m *Mailbox) mutateHostPlanClaim(ctx context.Context, claim MailboxResumeClaim, mutate func(*MailboxContinuationRecord) error) (MailboxResumeClaim, error) {
	m.continuationMu.Lock()
	defer m.continuationMu.Unlock()
	var updated MailboxResumeClaim
	err := m.continuationMutate(ctx, func(state *mailboxState) error {
		record, ok := state.Continuations[claim.Request.Ref.CheckpointID]
		if !ok || record.Claim == nil || record.Identity != claim.Request.identity() || record.Claim.AttemptID != claim.AttemptID {
			return ErrMailboxContinuationConflict
		}
		if err := mutate(&record); err != nil {
			return err
		}
		state.Continuations[claim.Request.Ref.CheckpointID] = record
		updated = *record.Claim
		return nil
	})
	return updated, err
}
func (m *Mailbox) mutateHostPlanClaimResult(ctx context.Context, claim MailboxResumeClaim, result HostPlanOutcome) (MailboxResumeClaim, error) {
	return m.mutateHostPlanClaim(ctx, claim, func(record *MailboxContinuationRecord) error {
		if record.Claim.Result != nil {
			if *record.Claim.Result == result {
				return nil
			}
			return ErrMailboxContinuationConflict
		}
		if record.Recovered {
			return ErrMailboxContinuationRecoveryRequired
		}
		if record.Claim.State != "claimed" && record.Claim.State != "running" {
			return ErrMailboxContinuationConflict
		}
		saved := result
		record.Claim.Result = &saved
		record.Claim.State = "completed"
		return nil
	})
}
func snapshotRecord(snapshot Snapshot, id string) (Record, bool) {
	for _, record := range snapshot.Records {
		if record.Envelope.MessageID == id {
			return record, true
		}
	}
	return Record{}, false
}
