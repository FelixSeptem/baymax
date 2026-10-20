package types

import (
	"context"
	"fmt"
	"strings"
)

// ContinuationCapabilities is source evidence for the native continuation
// manifest. It is deliberately independent of runner implementation types:
// package presence or a checkpoint DTO is not evidence of resumability.
type ContinuationCapabilities struct {
	ProfileVersion      string            `json:"profile_version"`
	SourceNamespace     string            `json:"source_namespace"`
	ContractVersion     string            `json:"contract_version"`
	ModuleRevision      string            `json:"module_revision"`
	APIVersion          string            `json:"api_version"`
	RetentionPolicy     string            `json:"retention_policy"`
	ExpiryPolicy        string            `json:"expiry_policy"`
	CheckpointCreation  bool              `json:"checkpoint_creation"`
	CrossProcessRestore bool              `json:"cross_process_restore"`
	PauseReasons        []string          `json:"pause_reasons"`
	EvidenceReferences  map[string]string `json:"evidence_references"`
	UnsupportedReasons  map[string]string `json:"unsupported_reasons,omitempty"`
}

// ContinuationCapabilitySource is implemented by the configured source owner.
// A host must query this owner; it must not infer capabilities from concrete
// store types or imported packages.
type ContinuationCapabilitySource interface {
	ContinuationCapabilities(context.Context) (ContinuationCapabilities, error)
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
	for _, dimension := range []string{"checkpoint_creation", "cross_process_restore", "input_required", "approval_required", "budget_exhausted"} {
		value := strings.TrimSpace(c.EvidenceReferences[dimension])
		if value == "" || len(value) > 256 || strings.HasPrefix(value, "package:") {
			return fmt.Errorf("continuation evidence for %s is invalid", dimension)
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
	return nil
}

func (c ContinuationCapabilities) Clone() ContinuationCapabilities {
	evidence, reasons := c.EvidenceReferences, c.UnsupportedReasons
	c.PauseReasons = append([]string(nil), c.PauseReasons...)
	c.EvidenceReferences = make(map[string]string, len(evidence))
	for key, value := range evidence {
		c.EvidenceReferences[key] = value
	}
	c.UnsupportedReasons = make(map[string]string, len(reasons))
	for key, value := range reasons {
		c.UnsupportedReasons[key] = value
	}
	return c
}
