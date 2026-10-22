package planbridge

import (
	"context"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

const CapabilityManifestSchemaVersion = "host_runtime_capability_manifest.v1"

const (
	CapabilityManifestContractVersion    = "baymax.runtime.contract.v1"
	CapabilityManifestAPIVersion         = "host_runtime_capability_manifest.v1"
	LimitationDurableSourceNotConfigured = "durable_source_not_configured"
	LimitationCompositeGraphRestore      = "composite_graph_restore_unsupported"
	LimitationSourceEvidenceMissing      = "source_evidence_missing"
	LimitationPlanResumedTrigger         = "plan_resumed_trigger_unsupported"
)

type ExecutionForm string

const (
	ExecutionFormDirect       ExecutionForm = "direct"
	ExecutionFormSubagent     ExecutionForm = "subagent"
	ExecutionFormTeam         ExecutionForm = "team"
	ExecutionFormWorkflow     ExecutionForm = "workflow"
	ExecutionFormMailboxAsync ExecutionForm = "mailbox_async"
)

type CapabilityState string

const (
	CapabilitySupported   CapabilityState = "supported"
	CapabilityUnsupported CapabilityState = "unsupported"
	CapabilityUnavailable CapabilityState = "unavailable"
	CapabilityUnknown     CapabilityState = ""
)

// CapabilityManifest is a detached, source-owned declaration of what a host
// can request. It is not business authorization and never creates a source.
type CapabilityManifest struct {
	SchemaVersion   string                    `json:"schema_version"`
	ContractVersion string                    `json:"contract_version"`
	ModuleRevision  string                    `json:"module_revision"`
	APIVersion      string                    `json:"api_version"`
	Forms           []ExecutionFormCapability `json:"forms"`
}

type ExecutionFormCapability struct {
	Form                ExecutionForm                   `json:"form"`
	Admission           CapabilityState                 `json:"admission"`
	NativeEvents        CapabilityState                 `json:"native_events"`
	DurableHistory      CapabilityState                 `json:"durable_history"`
	AfterCursorReplay   CapabilityState                 `json:"after_cursor_replay"`
	CheckpointCreation  CapabilityState                 `json:"checkpoint_creation"`
	CrossProcessRestore CapabilityState                 `json:"cross_process_restore"`
	InputRequired       CapabilityState                 `json:"input_required"`
	ApprovalRequired    CapabilityState                 `json:"approval_required"`
	BudgetExhausted     CapabilityState                 `json:"budget_exhausted"`
	Cancellation        CapabilityState                 `json:"cancellation"`
	Timeout             CapabilityState                 `json:"timeout"`
	CausalRetry         CapabilityState                 `json:"causal_retry"`
	MailboxContinuation CapabilityState                 `json:"mailbox_continuation"`
	RetentionPolicy     string                          `json:"retention_policy"`
	EvidenceReference   string                          `json:"evidence_reference"`
	UnsupportedReasons  map[string]string               `json:"unsupported_reasons,omitempty"`
	SourceHistory       *types.HistoryCapabilities      `json:"source_history,omitempty"`
	SourceContinuation  *types.ContinuationCapabilities `json:"source_continuation,omitempty"`
	Deployment          types.DeploymentScope           `json:"deployment"`
}

// CapabilityManifestReader returns detached entries so callers cannot change
// a declaration after it has been validated.
type CapabilityManifestReader struct{ manifest CapabilityManifest }

// DefaultCapabilityManifest reports the conservative baseline. History,
// checkpoint and composite pause continuation stay explicit until a source
// owner is configured and its evidence is supplied.
func DefaultCapabilityManifest() CapabilityManifest {
	forms := []ExecutionForm{ExecutionFormDirect, ExecutionFormSubagent, ExecutionFormTeam, ExecutionFormWorkflow, ExecutionFormMailboxAsync}
	result := CapabilityManifest{
		SchemaVersion:   CapabilityManifestSchemaVersion,
		ContractVersion: CapabilityManifestContractVersion,
		ModuleRevision:  "baymax",
		APIVersion:      CapabilityManifestAPIVersion,
		Forms:           make([]ExecutionFormCapability, 0, len(forms)),
	}
	for _, form := range forms {
		entry := ExecutionFormCapability{
			Form: form, Admission: CapabilitySupported, NativeEvents: CapabilitySupported,
			DurableHistory: CapabilityUnsupported, AfterCursorReplay: CapabilityUnsupported,
			CheckpointCreation: CapabilityUnsupported, CrossProcessRestore: CapabilityUnsupported,
			InputRequired: CapabilityUnsupported, ApprovalRequired: CapabilityUnsupported, BudgetExhausted: CapabilityUnsupported,
			Cancellation: CapabilitySupported, Timeout: CapabilitySupported, CausalRetry: CapabilitySupported,
			MailboxContinuation: CapabilityUnsupported, RetentionPolicy: "source-owned", EvidenceReference: "fixture:host-runtime-capability-manifest.v1",
			UnsupportedReasons: map[string]string{
				"durable_history":       "source journal is not configured for this source",
				"after_cursor_replay":   "source journal is not configured for this source",
				"checkpoint_creation":   LimitationCompositeGraphRestore,
				"cross_process_restore": LimitationCompositeGraphRestore,
				"input_required":        LimitationCompositeGraphRestore,
				"approval_required":     LimitationCompositeGraphRestore,
				"budget_exhausted":      LimitationCompositeGraphRestore,
				"mailbox_continuation":  "mailbox form is not enabled by default",
			},
			Deployment: types.DefaultDeploymentScope(string(form), "source owner deployment evidence is not configured"),
		}
		if form == ExecutionFormDirect {
			for _, dimension := range continuationDimensions() {
				entry.UnsupportedReasons[dimension] = LimitationDurableSourceNotConfigured
			}
		}
		if form == ExecutionFormMailboxAsync {
			entry.MailboxContinuation = CapabilitySupported
			delete(entry.UnsupportedReasons, "mailbox_continuation")
		}
		result.Forms = append(result.Forms, entry)
	}
	return result
}

func continuationDimensions() []string {
	return []string{"checkpoint_creation", "cross_process_restore", "input_required", "approval_required", "budget_exhausted"}
}

// CapabilityManifestWithContinuation queries an explicitly configured source.
// Only a complete direct source declaration can enable the durable profile;
// composite restore and resumed host plans remain unsupported.
func CapabilityManifestWithContinuation(ctx context.Context, source types.ContinuationCapabilitySource) (CapabilityManifest, error) {
	result := DefaultCapabilityManifest()
	if source == nil {
		return result, fail(SourceUnavailable, "capability_manifest.source_continuation", nil)
	}
	caps, err := source.ContinuationCapabilities(ctx)
	if err != nil {
		return result, err
	}
	if err = caps.Validate(); err != nil {
		for i := range result.Forms {
			if result.Forms[i].Form == ExecutionFormDirect {
				for _, dimension := range continuationDimensions() {
					result.Forms[i].UnsupportedReasons[dimension] = LimitationSourceEvidenceMissing
				}
			}
		}
		return result, &Error{Code: UnsupportedCapability, Field: "capability_manifest.source_continuation", Limitation: LimitationSourceEvidenceMissing}
	}
	if caps.ContractVersion != result.ContractVersion || caps.APIVersion != result.APIVersion {
		return result, fail(UnsupportedVersion, "capability_manifest.source_continuation", nil)
	}
	for i := range result.Forms {
		entry := &result.Forms[i]
		if entry.Form != ExecutionFormDirect {
			continue
		}
		snapshot := caps.Clone()
		entry.SourceContinuation = &snapshot
		entry.Deployment = caps.Deployment
		entry.CheckpointCreation, entry.CrossProcessRestore = CapabilitySupported, CapabilitySupported
		entry.InputRequired, entry.ApprovalRequired, entry.BudgetExhausted = CapabilitySupported, CapabilitySupported, CapabilitySupported
		entry.RetentionPolicy = caps.RetentionPolicy
		entry.EvidenceReference = caps.EvidenceReferences["cross_process_restore"]
		for _, dimension := range continuationDimensions() {
			delete(entry.UnsupportedReasons, dimension)
		}
	}
	result.ModuleRevision = caps.ModuleRevision
	return result, validateCapabilityManifest(result)
}

// CapabilityManifestWithWorkflowContinuation promotes continuation dimensions
// only for an explicit workflow source owner. It is intentionally separate
// from the direct profile so package presence, history, or another composite
// source cannot accidentally advertise workflow restore.
func CapabilityManifestWithWorkflowContinuation(ctx context.Context, source types.ContinuationCapabilitySource) (CapabilityManifest, error) {
	return capabilityManifestWithContinuationForForm(ctx, source, ExecutionFormWorkflow, "workflow")
}

// CapabilityManifestWithChildContinuation promotes only the child source
// owner. It does not imply team, mailbox, or resumed-plan support.
func CapabilityManifestWithChildContinuation(ctx context.Context, source types.ContinuationCapabilitySource) (CapabilityManifest, error) {
	return capabilityManifestWithContinuationForForm(ctx, source, ExecutionFormSubagent, "child")
}

func capabilityManifestWithContinuationForForm(ctx context.Context, source types.ContinuationCapabilitySource, form ExecutionForm, namespace string) (CapabilityManifest, error) {
	result := DefaultCapabilityManifest()
	if source == nil {
		return result, fail(SourceUnavailable, "capability_manifest.source_continuation", nil)
	}
	caps, err := source.ContinuationCapabilities(ctx)
	if err != nil {
		return result, err
	}
	if err = caps.Validate(); err != nil {
		return result, &Error{Code: UnsupportedCapability, Field: "capability_manifest.source_continuation", Limitation: LimitationSourceEvidenceMissing}
	}
	if caps.ContractVersion != result.ContractVersion || caps.APIVersion != result.APIVersion || caps.SourceNamespace != namespace {
		return result, fail(UnsupportedVersion, "capability_manifest.source_continuation", nil)
	}
	for i := range result.Forms {
		entry := &result.Forms[i]
		if entry.Form != form {
			continue
		}
		snapshot := caps.Clone()
		entry.SourceContinuation = &snapshot
		entry.Deployment = caps.Deployment
		entry.CheckpointCreation, entry.CrossProcessRestore = CapabilitySupported, CapabilitySupported
		entry.InputRequired, entry.ApprovalRequired, entry.BudgetExhausted = CapabilitySupported, CapabilitySupported, CapabilitySupported
		entry.RetentionPolicy = caps.RetentionPolicy
		entry.EvidenceReference = caps.EvidenceReferences["cross_process_restore"]
		for _, dimension := range continuationDimensions() {
			delete(entry.UnsupportedReasons, dimension)
		}
	}
	result.ModuleRevision = caps.ModuleRevision
	return result, validateCapabilityManifest(result)
}

// CapabilityManifestWithHistory projects only a source owner's evidenced facts.
// It does not open a mailbox, create a history store, or imply checkpoint/restore.
func CapabilityManifestWithHistory(ctx context.Context, reader types.NativeEventHistoryReader, source types.SourceRef) (CapabilityManifest, error) {
	result := DefaultCapabilityManifest()
	if reader == nil {
		return result, types.ErrSourceUnavailable
	}
	caps, err := reader.Capabilities(ctx, source)
	if err != nil {
		return result, err
	}
	if caps.Source != source {
		return result, types.ErrHistoryUnsupported
	}
	form, ok := historyForm(source.Kind)
	if !ok {
		return result, types.ErrHistoryUnsupported
	}
	for i := range result.Forms {
		entry := &result.Forms[i]
		if entry.Form != form {
			continue
		}
		entry.RetentionPolicy = caps.RetentionWindow
		entry.EvidenceReference = "fixture:native-event-history.v1"
		if !caps.Supported {
			entry.DurableHistory = CapabilityUnsupported
			entry.AfterCursorReplay = CapabilityUnsupported
			entry.UnsupportedReasons["durable_history"] = caps.UnsupportedReason
			entry.UnsupportedReasons["after_cursor_replay"] = caps.UnsupportedReason
			continue
		}
		if caps.Deployment != (types.DeploymentScope{}) {
			entry.Deployment = caps.Deployment
		}
		capabilitySnapshot := caps
		if caps.RetentionFloor != nil {
			floor := *caps.RetentionFloor
			capabilitySnapshot.RetentionFloor = &floor
		}
		entry.SourceHistory = &capabilitySnapshot
		entry.AfterCursorReplay = CapabilitySupported
		delete(entry.UnsupportedReasons, "after_cursor_replay")
		if caps.Durable {
			entry.DurableHistory = CapabilitySupported
			delete(entry.UnsupportedReasons, "durable_history")
		} else {
			entry.DurableHistory = CapabilityUnsupported
			entry.UnsupportedReasons["durable_history"] = "source retention is process-local"
		}
	}
	return result, validateCapabilityManifest(result)
}

func historyForm(kind types.HistorySourceKind) (ExecutionForm, bool) {
	switch kind {
	case types.HistorySourceRunner:
		return ExecutionFormDirect, true
	case types.HistorySourceChild:
		return ExecutionFormSubagent, true
	case types.HistorySourceTeam:
		return ExecutionFormTeam, true
	case types.HistorySourceWorkflow:
		return ExecutionFormWorkflow, true
	case types.HistorySourceMailbox:
		return ExecutionFormMailboxAsync, true
	default:
		return "", false
	}
}

func NewCapabilityManifestReader(manifest CapabilityManifest) (*CapabilityManifestReader, error) {
	if err := validateCapabilityManifest(manifest); err != nil {
		return nil, err
	}
	return &CapabilityManifestReader{manifest: cloneManifest(manifest)}, nil
}

func (r *CapabilityManifestReader) Capability(form ExecutionForm) (ExecutionFormCapability, error) {
	if r == nil {
		return ExecutionFormCapability{}, fail(SourceUnavailable, "capability_manifest", nil)
	}
	for _, entry := range r.manifest.Forms {
		if entry.Form == form {
			return cloneCapability(entry), nil
		}
	}
	return ExecutionFormCapability{}, fail(UnsupportedCapability, "capability_manifest.form", nil)
}

func (r *CapabilityManifestReader) ValidateCompatibility(contractVersion, apiVersion string) error {
	if r == nil {
		return fail(SourceUnavailable, "capability_manifest", nil)
	}
	if strings.TrimSpace(contractVersion) == "" || strings.TrimSpace(apiVersion) == "" {
		return fail(UnsupportedVersion, "capability_manifest.version", nil)
	}
	if r.manifest.ContractVersion != contractVersion || r.manifest.APIVersion != apiVersion {
		return fail(UnsupportedVersion, "capability_manifest.version", nil)
	}
	return nil
}

func validateCapabilityManifest(manifest CapabilityManifest) error {
	if manifest.SchemaVersion != CapabilityManifestSchemaVersion {
		return fail(UnsupportedVersion, "capability_manifest.schema_version", nil)
	}
	for _, value := range []struct{ field, value string }{
		{"capability_manifest.contract_version", manifest.ContractVersion},
		{"capability_manifest.module_revision", manifest.ModuleRevision},
		{"capability_manifest.api_version", manifest.APIVersion},
	} {
		if strings.TrimSpace(value.value) == "" || len(value.value) > 1024 {
			return fail(InvalidPlan, value.field, nil)
		}
	}
	if len(manifest.Forms) == 0 || len(manifest.Forms) > 128 {
		return fail(InvalidPlan, "capability_manifest.forms", nil)
	}
	seen := map[ExecutionForm]struct{}{}
	for _, entry := range manifest.Forms {
		if strings.TrimSpace(string(entry.Form)) == "" {
			return fail(InvalidPlan, "capability_manifest.form", nil)
		}
		if _, exists := seen[entry.Form]; exists {
			return fail(InvalidPlan, "capability_manifest.form", nil)
		}
		seen[entry.Form] = struct{}{}
		if err := validateCapability(entry); err != nil {
			return err
		}
		if c := entry.SourceContinuation; c != nil && (c.ContractVersion != manifest.ContractVersion || c.APIVersion != manifest.APIVersion || c.ModuleRevision != manifest.ModuleRevision) {
			return fail(UnsupportedVersion, "capability_manifest.source_continuation", nil)
		}
	}
	return nil
}

func validateCapability(entry ExecutionFormCapability) error {
	states := map[string]CapabilityState{
		"admission": entry.Admission, "native_events": entry.NativeEvents,
		"durable_history": entry.DurableHistory, "after_cursor_replay": entry.AfterCursorReplay,
		"checkpoint_creation": entry.CheckpointCreation, "cross_process_restore": entry.CrossProcessRestore,
		"input_required": entry.InputRequired, "approval_required": entry.ApprovalRequired,
		"budget_exhausted": entry.BudgetExhausted, "cancellation": entry.Cancellation,
		"timeout": entry.Timeout, "causal_retry": entry.CausalRetry,
		"mailbox_continuation": entry.MailboxContinuation,
	}
	if strings.TrimSpace(entry.RetentionPolicy) == "" || len(entry.RetentionPolicy) > 1024 || strings.TrimSpace(entry.EvidenceReference) == "" || len(entry.EvidenceReference) > 1024 {
		return fail(InvalidPlan, "capability_manifest.entry", nil)
	}
	if entry.Deployment != (types.DeploymentScope{}) {
		if err := entry.Deployment.Validate(); err != nil {
			return fail(InvalidPlan, "capability_manifest.deployment", err)
		}
	}
	for dimension, state := range states {
		switch state {
		case CapabilitySupported:
		case CapabilityUnsupported, CapabilityUnavailable:
			if strings.TrimSpace(entry.UnsupportedReasons[dimension]) == "" || len(entry.UnsupportedReasons[dimension]) > 1024 {
				return fail(InvalidPlan, "capability_manifest.unsupported_reason", nil)
			}
		default:
			return fail(InvalidPlan, "capability_manifest."+dimension, nil)
		}
	}
	continuationDeclared := entry.CheckpointCreation == CapabilitySupported || entry.CrossProcessRestore == CapabilitySupported || entry.InputRequired == CapabilitySupported || entry.ApprovalRequired == CapabilitySupported || entry.BudgetExhausted == CapabilitySupported
	if continuationDeclared || entry.SourceContinuation != nil {
		if (entry.Form != ExecutionFormDirect && entry.Form != ExecutionFormWorkflow && entry.Form != ExecutionFormSubagent) || entry.SourceContinuation == nil || entry.SourceContinuation.Validate() != nil || !continuationDeclared {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		if entry.Form == ExecutionFormWorkflow && entry.SourceContinuation.SourceNamespace != "workflow" {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		if entry.Form == ExecutionFormSubagent && entry.SourceContinuation.SourceNamespace != "child" {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		for _, dimension := range continuationDimensions() {
			if states[dimension] != CapabilitySupported {
				return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
			}
		}
		if entry.RetentionPolicy != entry.SourceContinuation.RetentionPolicy || entry.EvidenceReference != entry.SourceContinuation.EvidenceReferences["cross_process_restore"] {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
	}
	historyDeclared := entry.DurableHistory == CapabilitySupported || entry.AfterCursorReplay == CapabilitySupported
	if historyDeclared && entry.SourceHistory == nil {
		return fail(InvalidPlan, "capability_manifest.source_history", nil)
	}
	if entry.SourceHistory != nil {
		history := entry.SourceHistory
		expectedForm, supportedSource := historyForm(history.Source.Kind)
		if !supportedSource || entry.Form != expectedForm || history.Source.Validate() != nil || !history.Supported || !history.SequenceDefined || history.RetentionFloor == nil || history.RetentionFloor.Validate() != nil {
			return fail(InvalidPlan, "capability_manifest.source_history", nil)
		}
		for _, value := range []string{history.EventIdentityNamespace, history.SequenceScope, history.RetentionWindow, history.CursorScope, history.CursorVersion, history.TerminalRecovery} {
			if strings.TrimSpace(value) == "" || len(value) > types.MaxHistoryReferenceBytes {
				return fail(InvalidPlan, "capability_manifest.source_history", nil)
			}
		}
		if entry.AfterCursorReplay != CapabilitySupported || (history.Durable != (entry.DurableHistory == CapabilitySupported)) {
			return fail(InvalidPlan, "capability_manifest.source_history", nil)
		}
	}
	return nil
}

func cloneManifest(manifest CapabilityManifest) CapabilityManifest {
	clone := manifest
	clone.Forms = make([]ExecutionFormCapability, len(manifest.Forms))
	for i := range manifest.Forms {
		clone.Forms[i] = cloneCapability(manifest.Forms[i])
	}
	return clone
}

func cloneCapability(entry ExecutionFormCapability) ExecutionFormCapability {
	clone := entry
	clone.UnsupportedReasons = make(map[string]string, len(entry.UnsupportedReasons))
	for key, value := range entry.UnsupportedReasons {
		clone.UnsupportedReasons[key] = value
	}
	if entry.SourceHistory != nil {
		history := *entry.SourceHistory
		if entry.SourceHistory.RetentionFloor != nil {
			floor := *entry.SourceHistory.RetentionFloor
			history.RetentionFloor = &floor
		}
		clone.SourceHistory = &history
	}
	if entry.SourceContinuation != nil {
		continuation := entry.SourceContinuation.Clone()
		clone.SourceContinuation = &continuation
	}
	clone.Deployment = entry.Deployment
	return clone
}
