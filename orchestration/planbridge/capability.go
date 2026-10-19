package planbridge

import "strings"

const CapabilityManifestSchemaVersion = "host_runtime_capability_manifest.v1"

const (
	CapabilityManifestContractVersion = "baymax.runtime.contract.v1"
	CapabilityManifestAPIVersion      = "host_runtime_capability_manifest.v1"
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
	Form                ExecutionForm     `json:"form"`
	Admission           CapabilityState   `json:"admission"`
	NativeEvents        CapabilityState   `json:"native_events"`
	DurableHistory      CapabilityState   `json:"durable_history"`
	AfterCursorReplay   CapabilityState   `json:"after_cursor_replay"`
	CheckpointCreation  CapabilityState   `json:"checkpoint_creation"`
	CrossProcessRestore CapabilityState   `json:"cross_process_restore"`
	InputRequired       CapabilityState   `json:"input_required"`
	ApprovalRequired    CapabilityState   `json:"approval_required"`
	BudgetExhausted     CapabilityState   `json:"budget_exhausted"`
	Cancellation        CapabilityState   `json:"cancellation"`
	Timeout             CapabilityState   `json:"timeout"`
	CausalRetry         CapabilityState   `json:"causal_retry"`
	MailboxContinuation CapabilityState   `json:"mailbox_continuation"`
	RetentionPolicy     string            `json:"retention_policy"`
	EvidenceReference   string            `json:"evidence_reference"`
	UnsupportedReasons  map[string]string `json:"unsupported_reasons,omitempty"`
}

// CapabilityManifestReader returns detached entries so callers cannot change
// a declaration after it has been validated.
type CapabilityManifestReader struct{ manifest CapabilityManifest }

// DefaultCapabilityManifest reports the conservative Spec 1 baseline. History,
// checkpoint and composite pause continuation stay explicit until later Specs.
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
				"durable_history":       "source journal is Spec 2",
				"after_cursor_replay":   "source journal is Spec 2",
				"checkpoint_creation":   "durable checkpoint is Spec 3",
				"cross_process_restore": "durable checkpoint is Spec 3",
				"input_required":        "composite continuation is Spec 3",
				"approval_required":     "composite continuation is Spec 3",
				"budget_exhausted":      "composite continuation is Spec 3",
				"mailbox_continuation":  "mailbox form is not enabled by default",
			},
		}
		if form == ExecutionFormMailboxAsync {
			entry.MailboxContinuation = CapabilitySupported
			delete(entry.UnsupportedReasons, "mailbox_continuation")
		}
		result.Forms = append(result.Forms, entry)
	}
	return result
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
	return clone
}
