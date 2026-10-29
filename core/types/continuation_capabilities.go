package types

import (
	"context"
	"fmt"
	"strings"
)

const UnsupportedReasonMailboxHandlerSafePoint = "mailbox_handler_safe_point_unsupported"

// ContinuationCapabilities is source evidence for the native continuation
// manifest. It is deliberately independent of runner implementation types:
// package presence or a checkpoint DTO is not evidence of resumability.
type ContinuationCapabilities struct {
	ProfileVersion      string `json:"profile_version"`
	SourceNamespace     string `json:"source_namespace"`
	ContractVersion     string `json:"contract_version"`
	ModuleRevision      string `json:"module_revision"`
	APIVersion          string `json:"api_version"`
	RetentionPolicy     string `json:"retention_policy"`
	ExpiryPolicy        string `json:"expiry_policy"`
	CheckpointCreation  bool   `json:"checkpoint_creation"`
	CrossProcessRestore bool   `json:"cross_process_restore"`
	// HandlerExecutionSafePoint is meaningful for mailbox owners. Lifecycle
	// continuation is not enough to restore a host handler boundary.
	HandlerExecutionSafePoint bool                             `json:"handler_execution_safe_point,omitempty"`
	PauseReasons              []string                         `json:"pause_reasons"`
	EvidenceReferences        map[string]string                `json:"evidence_references"`
	UnsupportedReasons        map[string]string                `json:"unsupported_reasons,omitempty"`
	PauseReasonEvidence       map[string]PauseReasonCapability `json:"pause_reason_evidence,omitempty"`
	Deployment                DeploymentScope                  `json:"deployment"`
}

// PauseReasonCapability is detached source evidence for one native pause
// reason. Supported entries must point at source evidence; unsupported entries
// must explain why the owner cannot restore that reason.
type PauseReasonCapability struct {
	Supported         bool   `json:"supported"`
	EvidenceReference string `json:"evidence_reference,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
}

// ContinuationCapabilitySource is implemented by the configured source owner.
// A host must query this owner; it must not infer capabilities from concrete
// store types or imported packages.
type ContinuationCapabilitySource interface {
	ContinuationCapabilities(context.Context) (ContinuationCapabilities, error)
}

// PauseReason returns normalized evidence for a canonical pause reason. New
// sources use PauseReasonEvidence; legacy complete sources are normalized from
// PauseReasons and EvidenceReferences for compatibility.
func (c ContinuationCapabilities) PauseReason(reason string) (PauseReasonCapability, bool) {
	if evidence, ok := c.PauseReasonEvidence[reason]; ok {
		return evidence, true
	}
	if _, ok := map[string]struct{}{"input_required": {}, "approval_required": {}, "budget_exhausted": {}}[reason]; !ok {
		return PauseReasonCapability{}, false
	}
	for _, declared := range c.PauseReasons {
		if declared == reason && strings.TrimSpace(c.EvidenceReferences[reason]) != "" {
			return PauseReasonCapability{Supported: true, EvidenceReference: c.EvidenceReferences[reason]}, true
		}
	}
	if reasonText := strings.TrimSpace(c.UnsupportedReasons[reason]); reasonText != "" {
		return PauseReasonCapability{UnsupportedReason: reasonText}, true
	}
	return PauseReasonCapability{}, false
}

func (c ContinuationCapabilities) Validate() error {
	for name, value := range map[string]string{
		"profile_version": c.ProfileVersion, "source_namespace": c.SourceNamespace,
		"contract_version": c.ContractVersion, "module_revision": c.ModuleRevision,
		"api_version": c.APIVersion, "retention_policy": c.RetentionPolicy, "expiry_policy": c.ExpiryPolicy,
	} {
		if strings.TrimSpace(value) == "" || len(value) > 256 {
			return fmt.Errorf("continuation capability %s is empty or too long", name)
		}
	}
	if c.ProfileVersion != "native_continuation.v1" {
		return fmt.Errorf("unsupported continuation capability profile")
	}
	if !c.CheckpointCreation || !c.CrossProcessRestore {
		return fmt.Errorf("continuation source is not durable")
	}
	want := map[string]struct{}{"input_required": {}, "approval_required": {}, "budget_exhausted": {}}
	if len(c.PauseReasonEvidence) > 0 {
		if len(c.PauseReasons) != 0 {
			return fmt.Errorf("continuation source mixes legacy and per-reason pause evidence")
		}
		if len(c.PauseReasonEvidence) != len(want) {
			return fmt.Errorf("continuation source must declare every pause reason")
		}
		for reason := range want {
			if _, exists := c.EvidenceReferences[reason]; exists {
				return fmt.Errorf("continuation source mixes legacy and per-reason evidence for %s", reason)
			}
		}
		for reason, entry := range c.PauseReasonEvidence {
			if _, ok := want[reason]; !ok {
				return fmt.Errorf("unsupported continuation pause reason")
			}
			if entry.Supported {
				if strings.TrimSpace(entry.EvidenceReference) == "" || len(entry.EvidenceReference) > 256 || strings.HasPrefix(entry.EvidenceReference, "package:") || strings.TrimSpace(entry.UnsupportedReason) != "" {
					return fmt.Errorf("invalid supported evidence for %s", reason)
				}
			} else if strings.TrimSpace(entry.UnsupportedReason) == "" || len(entry.UnsupportedReason) > 256 || strings.TrimSpace(entry.EvidenceReference) != "" {
				return fmt.Errorf("invalid unsupported evidence for %s", reason)
			}
			delete(want, reason)
		}
		if len(want) != 0 {
			return fmt.Errorf("continuation source must declare every pause reason")
		}
		for _, dimension := range []string{"checkpoint_creation", "cross_process_restore"} {
			value := strings.TrimSpace(c.EvidenceReferences[dimension])
			if value == "" || len(value) > 256 || strings.HasPrefix(value, "package:") {
				return fmt.Errorf("continuation evidence for %s is invalid", dimension)
			}
		}
	} else {
		if len(c.PauseReasons) != len(want) {
			return fmt.Errorf("continuation source must evidence all pause reasons")
		}
		for _, reason := range c.PauseReasons {
			if _, ok := want[reason]; !ok {
				return fmt.Errorf("unsupported continuation pause reason")
			}
			delete(want, reason)
		}
		if len(want) != 0 || len(c.EvidenceReferences) != 5 {
			return fmt.Errorf("continuation source evidence is incomplete")
		}
	}
	for _, dimension := range []string{"checkpoint_creation", "cross_process_restore"} {
		value := strings.TrimSpace(c.EvidenceReferences[dimension])
		if value == "" || len(value) > 256 || strings.HasPrefix(value, "package:") {
			return fmt.Errorf("continuation evidence for %s is invalid", dimension)
		}
	}
	if len(c.PauseReasonEvidence) == 0 {
		for _, dimension := range []string{"input_required", "approval_required", "budget_exhausted"} {
			value := strings.TrimSpace(c.EvidenceReferences[dimension])
			if value == "" || len(value) > 256 || strings.HasPrefix(value, "package:") {
				return fmt.Errorf("continuation evidence for %s is invalid", dimension)
			}
		}
	}
	if len(c.UnsupportedReasons) > 5 {
		return fmt.Errorf("too many continuation limitations")
	}
	for dimension, reason := range c.UnsupportedReasons {
		if _, known := c.EvidenceReferences[dimension]; !known || strings.TrimSpace(reason) == "" || len(reason) > 256 {
			return fmt.Errorf("invalid continuation limitation")
		}
	}
	if c.Deployment != (DeploymentScope{}) {
		if err := c.Deployment.Validate(); err != nil {
			return fmt.Errorf("continuation deployment scope: %w", err)
		}
	}
	return nil
}

func (c ContinuationCapabilities) Clone() ContinuationCapabilities {
	evidence, reasons, pauseEvidence := c.EvidenceReferences, c.UnsupportedReasons, c.PauseReasonEvidence
	c.PauseReasons = append([]string(nil), c.PauseReasons...)
	c.EvidenceReferences = make(map[string]string, len(evidence))
	for key, value := range evidence {
		c.EvidenceReferences[key] = value
	}
	c.UnsupportedReasons = make(map[string]string, len(reasons))
	for key, value := range reasons {
		c.UnsupportedReasons[key] = value
	}
	if pauseEvidence != nil {
		c.PauseReasonEvidence = make(map[string]PauseReasonCapability, len(pauseEvidence))
		for key, value := range pauseEvidence {
			c.PauseReasonEvidence[key] = value
		}
	}
	return c
}
