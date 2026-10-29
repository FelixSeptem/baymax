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
	LimitationMailboxContextSafePoint    = "mailbox_handler_context_safe_point_unsupported"
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
	Form                      ExecutionForm                   `json:"form"`
	Admission                 CapabilityState                 `json:"admission"`
	NativeEvents              CapabilityState                 `json:"native_events"`
	DurableHistory            CapabilityState                 `json:"durable_history"`
	AfterCursorReplay         CapabilityState                 `json:"after_cursor_replay"`
	SourceProjectionRecovery  CapabilityState                 `json:"source_projection_recovery"`
	CheckpointCreation        CapabilityState                 `json:"checkpoint_creation"`
	CrossProcessRestore       CapabilityState                 `json:"cross_process_restore"`
	InputRequired             CapabilityState                 `json:"input_required"`
	ApprovalRequired          CapabilityState                 `json:"approval_required"`
	BudgetExhausted           CapabilityState                 `json:"budget_exhausted"`
	Cancellation              CapabilityState                 `json:"cancellation"`
	Timeout                   CapabilityState                 `json:"timeout"`
	CausalRetry               CapabilityState                 `json:"causal_retry"`
	MailboxContinuation       CapabilityState                 `json:"mailbox_continuation"`
	HostContextAssembly       CapabilityState                 `json:"host_context_assembly"`
	HostContextHandoffRestore CapabilityState                 `json:"host_context_handoff_restore"`
	HostContext               *HostContextCapabilityEvidence  `json:"host_context,omitempty"`
	RetentionPolicy           string                          `json:"retention_policy"`
	EvidenceReference         string                          `json:"evidence_reference"`
	UnsupportedReasons        map[string]string               `json:"unsupported_reasons,omitempty"`
	SourceHistory             *types.HistoryCapabilities      `json:"source_history,omitempty"`
	SourceContinuation        *types.ContinuationCapabilities `json:"source_continuation,omitempty"`
	Deployment                types.DeploymentScope           `json:"deployment"`
}

// HostContextCapabilityEvidence describes one source owner's reference-only
// projection integration. Each supported cell requires its own evidence key.
type HostContextCapabilityEvidence struct {
	OwnerNamespace          string                `json:"owner_namespace"`
	SupportedNodeScopes     []string              `json:"supported_node_scopes"`
	ProjectionAPIVersion    string                `json:"projection_api_version"`
	ContinuationAPIVersion  string                `json:"continuation_api_version"`
	ModuleRevision          string                `json:"module_revision"`
	ResolverReauthorization bool                  `json:"resolver_reauthorization"`
	RetentionPolicy         string                `json:"retention_policy"`
	ExpiryPolicy            string                `json:"expiry_policy"`
	EvidenceReferences      map[string]string     `json:"evidence_references"`
	Deployment              types.DeploymentScope `json:"deployment"`
}

// HostContextCapabilityEvidenceSource lets an existing source owner publish
// the complete, detached evidence required to promote its context cells. The
// source does not create an owner, execute a graph, or resolve host context.
// Callers must explicitly bind this source to the owner they operate.
type HostContextCapabilityEvidenceSource interface {
	HostContextCapabilityEvidence(context.Context) (ExecutionForm, HostContextCapabilityEvidence, error)
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
			SourceProjectionRecovery: CapabilityUnsupported,
			CheckpointCreation:       CapabilityUnsupported, CrossProcessRestore: CapabilityUnsupported,
			InputRequired: CapabilityUnsupported, ApprovalRequired: CapabilityUnsupported, BudgetExhausted: CapabilityUnsupported,
			HostContextAssembly: CapabilityUnsupported, HostContextHandoffRestore: CapabilityUnsupported,
			Cancellation: CapabilitySupported, Timeout: CapabilitySupported, CausalRetry: CapabilitySupported,
			MailboxContinuation: CapabilityUnsupported, RetentionPolicy: "source-owned", EvidenceReference: "fixture:host-runtime-capability-manifest.v1",
			UnsupportedReasons: map[string]string{
				"durable_history":              "source journal is not configured for this source",
				"after_cursor_replay":          "source journal is not configured for this source",
				"source_projection_recovery":   "source commit/projection ledger is not configured for this source",
				"checkpoint_creation":          LimitationCompositeGraphRestore,
				"cross_process_restore":        LimitationCompositeGraphRestore,
				"input_required":               LimitationCompositeGraphRestore,
				"approval_required":            LimitationCompositeGraphRestore,
				"budget_exhausted":             LimitationCompositeGraphRestore,
				"mailbox_continuation":         "mailbox form is not enabled by default",
				"host_context_assembly":        "source owner context binding evidence is not configured",
				"host_context_handoff_restore": "source owner context restore evidence is not configured",
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
			entry.UnsupportedReasons["host_context_assembly"] = LimitationMailboxContextSafePoint
			entry.UnsupportedReasons["host_context_handoff_restore"] = LimitationMailboxContextSafePoint
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
		promotePauseReasonCells(entry, caps, false)
		entry.RetentionPolicy = caps.RetentionPolicy
		entry.EvidenceReference = caps.EvidenceReferences["cross_process_restore"]
		for _, dimension := range continuationDimensions()[:2] {
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

// CapabilityManifestWithTeamContinuation promotes only the explicitly
// configured team owner. It does not imply child, workflow, mailbox, or
// resumed-plan support.
func CapabilityManifestWithTeamContinuation(ctx context.Context, source types.ContinuationCapabilitySource) (CapabilityManifest, error) {
	return capabilityManifestWithContinuationForForm(ctx, source, ExecutionFormTeam, "team")
}

// CapabilityManifestWithMailboxContinuation promotes only the existing
// mailbox owner. It never creates a queue or execution loop.
func CapabilityManifestWithMailboxContinuation(ctx context.Context, source types.ContinuationCapabilitySource) (CapabilityManifest, error) {
	return capabilityManifestWithContinuationForForm(ctx, source, ExecutionFormMailboxAsync, "mailbox")
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
		promotePauseReasonCells(entry, caps, form == ExecutionFormMailboxAsync)
		if form == ExecutionFormMailboxAsync {
			entry.MailboxContinuation = CapabilitySupported
			delete(entry.UnsupportedReasons, "mailbox_continuation")
		}
		entry.RetentionPolicy = caps.RetentionPolicy
		entry.EvidenceReference = caps.EvidenceReferences["cross_process_restore"]
		for _, dimension := range continuationDimensions()[:2] {
			delete(entry.UnsupportedReasons, dimension)
		}
	}
	result.ModuleRevision = caps.ModuleRevision
	return result, validateCapabilityManifest(result)
}

func promotePauseReasonCells(entry *ExecutionFormCapability, caps types.ContinuationCapabilities, mailboxLifecycleOnly bool) {
	for _, dimension := range continuationDimensions()[2:] {
		evidence, declared := caps.PauseReason(dimension)
		state := CapabilityUnsupported
		reason := LimitationSourceEvidenceMissing
		switch {
		case declared && evidence.Supported && (!mailboxLifecycleOnly || mailboxHandlerSafePoint(caps)):
			state = CapabilitySupported
			reason = ""
		case declared && strings.TrimSpace(evidence.UnsupportedReason) != "":
			reason = evidence.UnsupportedReason
		case mailboxLifecycleOnly && !mailboxHandlerSafePoint(caps):
			reason = LimitationMailboxContextSafePoint
		}
		switch dimension {
		case "input_required":
			entry.InputRequired = state
		case "approval_required":
			entry.ApprovalRequired = state
		case "budget_exhausted":
			entry.BudgetExhausted = state
		}
		if state == CapabilitySupported {
			delete(entry.UnsupportedReasons, dimension)
		} else {
			entry.UnsupportedReasons[dimension] = reason
		}
	}
}

func mailboxHandlerSafePoint(caps types.ContinuationCapabilities) bool {
	return caps.SourceNamespace != "mailbox" || caps.HandlerExecutionSafePoint
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
			entry.SourceProjectionRecovery = CapabilityUnsupported
			entry.UnsupportedReasons["source_projection_recovery"] = caps.UnsupportedReason
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
		if caps.ProjectionRecovery {
			entry.SourceProjectionRecovery = CapabilitySupported
			delete(entry.UnsupportedReasons, "source_projection_recovery")
		} else {
			entry.SourceProjectionRecovery = CapabilityUnsupported
			reason := caps.ProjectionRecoveryReason
			if strings.TrimSpace(reason) == "" {
				reason = "source commit/projection ledger is not configured for this source"
			}
			entry.UnsupportedReasons["source_projection_recovery"] = reason
		}
	}
	return result, validateCapabilityManifest(result)
}

// CapabilityManifestWithHostContextEvidence promotes both context cells only
// for an explicit owner source with complete, form-local evidence. It never
// infers context support from a continuation, history reader, package, or DTO.
// Mailbox lifecycle evidence remains insufficient until the mailbox owner
// exposes a message/delivery/lease-bound handler safe point.
func CapabilityManifestWithHostContextEvidence(ctx context.Context, source HostContextCapabilityEvidenceSource) (CapabilityManifest, error) {
	result := DefaultCapabilityManifest()
	if source == nil {
		return result, fail(SourceUnavailable, "capability_manifest.host_context", nil)
	}
	form, evidence, err := source.HostContextCapabilityEvidence(ctx)
	if err != nil {
		return result, err
	}
	if form == ExecutionFormMailboxAsync {
		return result, &Error{Code: UnsupportedCapability, Field: "capability_manifest.host_context", Limitation: LimitationMailboxContextSafePoint}
	}
	found := false
	for i := range result.Forms {
		entry := &result.Forms[i]
		if entry.Form != form {
			continue
		}
		found = true
		evidenceSnapshot := evidence
		evidenceSnapshot.SupportedNodeScopes = append([]string(nil), evidence.SupportedNodeScopes...)
		evidenceSnapshot.EvidenceReferences = make(map[string]string, len(evidence.EvidenceReferences))
		for key, value := range evidence.EvidenceReferences {
			evidenceSnapshot.EvidenceReferences[key] = value
		}
		entry.HostContextAssembly = CapabilitySupported
		entry.HostContextHandoffRestore = CapabilitySupported
		entry.HostContext = &evidenceSnapshot
		entry.RetentionPolicy = evidence.RetentionPolicy
		entry.EvidenceReference = evidence.EvidenceReferences["host_context_handoff_restore"]
		entry.Deployment = evidence.Deployment
		delete(entry.UnsupportedReasons, "host_context_assembly")
		delete(entry.UnsupportedReasons, "host_context_handoff_restore")
	}
	if !found {
		return result, fail(UnsupportedCapability, "capability_manifest.host_context.form", nil)
	}
	if strings.TrimSpace(evidence.ModuleRevision) != result.ModuleRevision {
		return result, fail(UnsupportedVersion, "capability_manifest.host_context.module_revision", nil)
	}
	if err := validateCapabilityManifest(result); err != nil {
		return DefaultCapabilityManifest(), err
	}
	return result, nil
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
		if err := validateCapability(entry, manifest.ModuleRevision); err != nil {
			return err
		}
		if c := entry.SourceContinuation; c != nil && (c.ContractVersion != manifest.ContractVersion || c.APIVersion != manifest.APIVersion || c.ModuleRevision != manifest.ModuleRevision) {
			return fail(UnsupportedVersion, "capability_manifest.source_continuation", nil)
		}
	}
	return nil
}

func validateCapability(entry ExecutionFormCapability, manifestModuleRevision string) error {
	states := map[string]CapabilityState{
		"admission": entry.Admission, "native_events": entry.NativeEvents,
		"durable_history": entry.DurableHistory, "after_cursor_replay": entry.AfterCursorReplay,
		"checkpoint_creation": entry.CheckpointCreation, "cross_process_restore": entry.CrossProcessRestore,
		"input_required": entry.InputRequired, "approval_required": entry.ApprovalRequired,
		"budget_exhausted": entry.BudgetExhausted, "cancellation": entry.Cancellation,
		"timeout": entry.Timeout, "causal_retry": entry.CausalRetry,
		"mailbox_continuation":         entry.MailboxContinuation,
		"host_context_assembly":        entry.HostContextAssembly,
		"host_context_handoff_restore": entry.HostContextHandoffRestore,
	}
	if entry.SourceProjectionRecovery != CapabilityUnknown || entry.SourceHistory != nil {
		states["source_projection_recovery"] = entry.SourceProjectionRecovery
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
		if (entry.Form != ExecutionFormDirect && entry.Form != ExecutionFormWorkflow && entry.Form != ExecutionFormSubagent && entry.Form != ExecutionFormTeam && entry.Form != ExecutionFormMailboxAsync) || entry.SourceContinuation == nil || entry.SourceContinuation.Validate() != nil || !continuationDeclared {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		if entry.Form == ExecutionFormWorkflow && entry.SourceContinuation.SourceNamespace != "workflow" {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		if entry.Form == ExecutionFormSubagent && entry.SourceContinuation.SourceNamespace != "child" {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		if entry.Form == ExecutionFormTeam && entry.SourceContinuation.SourceNamespace != "team" {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		if entry.Form == ExecutionFormMailboxAsync && (entry.SourceContinuation.SourceNamespace != "mailbox" || entry.MailboxContinuation != CapabilitySupported) {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		deployment := entry.SourceContinuation.Deployment
		if deployment != (types.DeploymentScope{}) && deployment.Validate() != nil {
			return fail(InvalidPlan, "capability_manifest.source_continuation.deployment", nil)
		}
		if states["checkpoint_creation"] != CapabilitySupported || states["cross_process_restore"] != CapabilitySupported {
			return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
		}
		for _, dimension := range continuationDimensions()[2:] {
			if states[dimension] == CapabilitySupported {
				evidence, ok := entry.SourceContinuation.PauseReason(dimension)
				if !ok || !evidence.Supported {
					return fail(InvalidPlan, "capability_manifest.source_continuation", nil)
				}
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
		if entry.AfterCursorReplay != CapabilitySupported || (history.Durable != (entry.DurableHistory == CapabilitySupported)) || entry.SourceProjectionRecovery == CapabilityUnknown {
			return fail(InvalidPlan, "capability_manifest.source_history", nil)
		}
	}
	contextDeclared := entry.HostContextAssembly == CapabilitySupported || entry.HostContextHandoffRestore == CapabilitySupported
	if contextDeclared && entry.HostContext == nil {
		return fail(InvalidPlan, "capability_manifest.host_context", nil)
	}
	if entry.HostContext != nil {
		evidence := entry.HostContext
		ownerNamespace, continuationVersion, allowedScopes := hostContextOwnerContract(entry.Form)
		if strings.TrimSpace(evidence.OwnerNamespace) == "" || len(evidence.OwnerNamespace) > 256 || len(evidence.SupportedNodeScopes) == 0 ||
			evidence.ProjectionAPIVersion != "host_context_projection.v1" || strings.TrimSpace(evidence.ContinuationAPIVersion) == "" ||
			strings.TrimSpace(evidence.ModuleRevision) == "" || !evidence.ResolverReauthorization ||
			strings.TrimSpace(evidence.RetentionPolicy) == "" || strings.TrimSpace(evidence.ExpiryPolicy) == "" || evidence.Deployment.Validate() != nil || !evidence.Deployment.Supported || evidence.Deployment.OwnerNamespace != evidence.OwnerNamespace {
			return fail(InvalidPlan, "capability_manifest.host_context", nil)
		}
		if ownerNamespace == "" || evidence.OwnerNamespace != ownerNamespace || evidence.ContinuationAPIVersion != continuationVersion {
			return fail(UnsupportedVersion, "capability_manifest.host_context.owner", nil)
		}
		if evidence.Deployment != entry.Deployment {
			return fail(InvalidPlan, "capability_manifest.host_context.deployment", nil)
		}
		seenScopes := map[string]struct{}{}
		for _, scope := range evidence.SupportedNodeScopes {
			if strings.TrimSpace(scope) == "" || len(scope) > 256 {
				return fail(InvalidPlan, "capability_manifest.host_context.node_scope", nil)
			}
			if _, allowed := allowedScopes[scope]; !allowed {
				return fail(InvalidPlan, "capability_manifest.host_context.node_scope", nil)
			}
			if _, exists := seenScopes[scope]; exists {
				return fail(InvalidPlan, "capability_manifest.host_context.node_scope", nil)
			}
			seenScopes[scope] = struct{}{}
		}
		for dimension, state := range map[string]CapabilityState{
			"host_context_assembly":        entry.HostContextAssembly,
			"host_context_handoff_restore": entry.HostContextHandoffRestore,
		} {
			if state == CapabilitySupported && strings.TrimSpace(evidence.EvidenceReferences[dimension]) == "" {
				return fail(InvalidPlan, "capability_manifest.host_context.evidence", nil)
			}
		}
		if contextDeclared {
			if evidence.ModuleRevision != manifestModuleRevision {
				return fail(UnsupportedVersion, "capability_manifest.host_context.module_revision", nil)
			}
		}
	}
	return nil
}

func hostContextOwnerContract(form ExecutionForm) (string, string, map[string]struct{}) {
	var namespace, apiVersion string
	var scopes []string
	switch form {
	case ExecutionFormDirect:
		namespace, apiVersion, scopes = "runner", "direct_context.v1", []string{"run"}
	case ExecutionFormSubagent:
		namespace, apiVersion, scopes = "child", "child_continuation.v1", []string{"task", "attempt"}
	case ExecutionFormTeam:
		namespace, apiVersion, scopes = "team", "team_continuation.v1", []string{"member"}
	case ExecutionFormWorkflow:
		namespace, apiVersion, scopes = "workflow", "workflow_continuation.v1", []string{"step"}
	case ExecutionFormMailboxAsync:
		namespace, apiVersion, scopes = "mailbox", "mailbox_continuation.v1", []string{"message", "delivery", "lease"}
	default:
		return "", "", nil
	}
	allowed := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		allowed[scope] = struct{}{}
	}
	return namespace, apiVersion, allowed
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
	if entry.HostContext != nil {
		evidence := *entry.HostContext
		evidence.SupportedNodeScopes = append([]string(nil), entry.HostContext.SupportedNodeScopes...)
		evidence.EvidenceReferences = make(map[string]string, len(entry.HostContext.EvidenceReferences))
		for key, value := range entry.HostContext.EvidenceReferences {
			evidence.EvidenceReferences[key] = value
		}
		clone.HostContext = &evidence
	}
	clone.Deployment = entry.Deployment
	return clone
}
