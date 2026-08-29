package trace

import (
	"maps"
	"slices"
	"strconv"
	"strings"
)

const OTelSemconvVersionV1 = "otel_semconv.v1"

const (
	TraceDomainRun    = "run"
	TraceDomainModel  = "model"
	TraceDomainTool   = "tool"
	TraceDomainMCP    = "mcp"
	TraceDomainMemory = "memory"
	TraceDomainHITL   = "hitl"
)

const (
	CanonicalSpanRun    = "agent.run"
	CanonicalSpanModel  = "agent.model"
	CanonicalSpanTool   = "agent.tool"
	CanonicalSpanMCP    = "agent.mcp"
	CanonicalSpanMemory = "agent.memory"
	CanonicalSpanHITL   = "agent.hitl"
)

const (
	AttrTraceSchemaVersion              = "trace.schema_version"
	AttrDomain                          = "trace.domain"
	AttrRunID                           = "run.id"
	AttrMode                            = "run.mode"
	AttrStepID                          = "step.id"
	AttrProtocolSource                  = "agent.protocol.source"
	AttrCausationID                     = "agent.protocol.causation_id"
	AttrArtifactID                      = "agent.protocol.artifact_id"
	AttrCheckpointID                    = "agent.protocol.checkpoint_id"
	AttrProtocolProfileVersion          = "agent.protocol.profile_version"
	AttrProtocolCapabilityDecision      = "agent.protocol.capability_decision"
	AttrProtocolCapabilityReason        = "agent.protocol.capability_reason"
	AttrProtocolAdmissionPolicy         = "agent.protocol.admission_policy"
	AttrProtocolAdmissionDecision       = "agent.protocol.admission_decision"
	AttrProtocolAdmissionReason         = "agent.protocol.admission_reason"
	AttrToolName                        = "tool.name"
	AttrMCPTransport                    = "mcp.transport"
	AttrMemoryScope                     = "memory_scope_selected"
	AttrBudgetDecision                  = "budget_decision"
	AttrPolicyDecisionPath              = "policy_decision_path"
	AttrStreamSubscriptionID            = "agent.stream.subscription_id"
	AttrStreamBindingPhase              = "agent.stream.binding_phase"
	AttrStreamBindingDecision           = "agent.stream.binding_decision"
	AttrStreamBindingReason             = "agent.stream.binding_reason"
	AttrStreamBindingCursorMode         = "agent.stream.cursor_mode"
	AttrStreamBindingSequenceBoundary   = "agent.stream.sequence_boundary"
	AttrProtocolCheckpointRelation      = "agent.protocol.checkpoint_relation"
	AttrProtocolCheckpointRestoreSource = "agent.protocol.checkpoint_restore_source"
	AttrProtocolWorkspaceProvenance     = "agent.protocol.workspace_provenance"
	AttrProtocolWorkspaceDriftReason    = "agent.protocol.workspace_drift_reason"
)

// ProtocolAttributes returns additive protocol correlation attributes while
// preserving the existing semantic-convention topology and attribute names.
func ProtocolAttributes(runID, stepID, source, causationID, artifactID, checkpointID string) map[string]string {
	attrs := map[string]string{
		AttrRunID:          strings.TrimSpace(runID),
		AttrStepID:         strings.TrimSpace(stepID),
		AttrProtocolSource: strings.TrimSpace(source),
		AttrCausationID:    strings.TrimSpace(causationID),
		AttrArtifactID:     strings.TrimSpace(artifactID),
		AttrCheckpointID:   strings.TrimSpace(checkpointID),
	}
	for key, value := range attrs {
		if value == "" {
			delete(attrs, key)
		}
	}
	return attrs
}

// EventStreamBindingAttributes returns additive, bounded stream correlation.
// Cursor bodies, event payloads, and arbitrary subscriber metadata are
// intentionally excluded to keep OTel cardinality bounded.
func EventStreamBindingAttributes(subscriptionID, phase, decision, reason, cursorMode string, sequenceBoundary int64) map[string]string {
	attrs := map[string]string{
		AttrStreamSubscriptionID:    strings.TrimSpace(subscriptionID),
		AttrStreamBindingPhase:      strings.TrimSpace(phase),
		AttrStreamBindingDecision:   strings.TrimSpace(decision),
		AttrStreamBindingReason:     strings.TrimSpace(reason),
		AttrStreamBindingCursorMode: strings.TrimSpace(cursorMode),
	}
	if sequenceBoundary > 0 {
		attrs[AttrStreamBindingSequenceBoundary] = strconv.FormatInt(sequenceBoundary, 10)
	}
	for key, value := range attrs {
		if value == "" {
			delete(attrs, key)
		}
	}
	return attrs
}

// ProtocolDecisionAttributes adds bounded descriptor/admission correlation
// without changing the existing trace topology or authorization semantics.
func ProtocolDecisionAttributes(profileVersion, capabilityDecision, capabilityReason, admissionPolicy, admissionDecision, admissionReason string) map[string]string {
	attrs := map[string]string{
		AttrProtocolProfileVersion:     strings.TrimSpace(profileVersion),
		AttrProtocolCapabilityDecision: strings.TrimSpace(capabilityDecision),
		AttrProtocolCapabilityReason:   strings.TrimSpace(capabilityReason),
		AttrProtocolAdmissionPolicy:    strings.TrimSpace(admissionPolicy),
		AttrProtocolAdmissionDecision:  strings.TrimSpace(admissionDecision),
		AttrProtocolAdmissionReason:    strings.TrimSpace(admissionReason),
	}
	for key, value := range attrs {
		if value == "" {
			delete(attrs, key)
		}
	}
	return attrs
}

// CheckpointProvenanceAttributes returns bounded provenance state/reason
// attributes; checkpoint/workspace identifiers and integrity values are not
// emitted to avoid high-cardinality telemetry.
func CheckpointProvenanceAttributes(relation, restoreSource, provenanceState, driftReason string) map[string]string {
	attrs := map[string]string{
		AttrProtocolCheckpointRelation:      strings.TrimSpace(relation),
		AttrProtocolCheckpointRestoreSource: strings.TrimSpace(restoreSource),
		AttrProtocolWorkspaceProvenance:     strings.TrimSpace(provenanceState),
		AttrProtocolWorkspaceDriftReason:    strings.TrimSpace(driftReason),
	}
	for key, value := range attrs {
		if value == "" {
			delete(attrs, key)
		}
	}
	return attrs
}

type SpanTopologySpec struct {
	Domain            string
	SpanName          string
	ParentDomain      string
	CanonicalAttrKeys []string
}

type SemanticSpan struct {
	Domain     string
	Parent     string
	Attributes map[string]string
}

var semconvTopologyV1 = map[string]SpanTopologySpec{
	TraceDomainRun: {
		Domain:       TraceDomainRun,
		SpanName:     CanonicalSpanRun,
		ParentDomain: "",
		CanonicalAttrKeys: []string{
			AttrTraceSchemaVersion,
			AttrDomain,
			AttrRunID,
			AttrMode,
			AttrBudgetDecision,
			AttrPolicyDecisionPath,
		},
	},
	TraceDomainModel: {
		Domain:       TraceDomainModel,
		SpanName:     CanonicalSpanModel,
		ParentDomain: TraceDomainRun,
		CanonicalAttrKeys: []string{
			AttrTraceSchemaVersion,
			AttrDomain,
			AttrRunID,
			AttrStepID,
			AttrMode,
			AttrBudgetDecision,
		},
	},
	TraceDomainTool: {
		Domain:       TraceDomainTool,
		SpanName:     CanonicalSpanTool,
		ParentDomain: TraceDomainRun,
		CanonicalAttrKeys: []string{
			AttrTraceSchemaVersion,
			AttrDomain,
			AttrRunID,
			AttrStepID,
			AttrToolName,
			AttrBudgetDecision,
		},
	},
	TraceDomainMCP: {
		Domain:       TraceDomainMCP,
		SpanName:     CanonicalSpanMCP,
		ParentDomain: TraceDomainTool,
		CanonicalAttrKeys: []string{
			AttrTraceSchemaVersion,
			AttrDomain,
			AttrRunID,
			AttrStepID,
			AttrMCPTransport,
		},
	},
	TraceDomainMemory: {
		Domain:       TraceDomainMemory,
		SpanName:     CanonicalSpanMemory,
		ParentDomain: TraceDomainRun,
		CanonicalAttrKeys: []string{
			AttrTraceSchemaVersion,
			AttrDomain,
			AttrRunID,
			AttrMemoryScope,
			AttrBudgetDecision,
		},
	},
	TraceDomainHITL: {
		Domain:       TraceDomainHITL,
		SpanName:     CanonicalSpanHITL,
		ParentDomain: TraceDomainRun,
		CanonicalAttrKeys: []string{
			AttrTraceSchemaVersion,
			AttrDomain,
			AttrRunID,
			AttrStepID,
		},
	},
}

func CanonicalSemconvTopologyV1() map[string]SpanTopologySpec {
	out := make(map[string]SpanTopologySpec, len(semconvTopologyV1))
	for domain, spec := range semconvTopologyV1 {
		cloned := spec
		cloned.CanonicalAttrKeys = append([]string(nil), spec.CanonicalAttrKeys...)
		out[domain] = cloned
	}
	return out
}

func CanonicalSemconvSpec(domain string) (SpanTopologySpec, bool) {
	spec, ok := semconvTopologyV1[strings.ToLower(strings.TrimSpace(domain))]
	if !ok {
		return SpanTopologySpec{}, false
	}
	out := spec
	out.CanonicalAttrKeys = append([]string(nil), spec.CanonicalAttrKeys...)
	return out, true
}

func CanonicalAttributeMap(domain string, attrs map[string]string) map[string]string {
	spec, ok := CanonicalSemconvSpec(domain)
	if !ok {
		return map[string]string{}
	}
	in := map[string]string{}
	if attrs != nil {
		maps.Copy(in, attrs)
	}
	out := make(map[string]string, len(spec.CanonicalAttrKeys))
	out[AttrTraceSchemaVersion] = OTelSemconvVersionV1
	out[AttrDomain] = spec.Domain
	for _, key := range spec.CanonicalAttrKeys {
		if key == AttrTraceSchemaVersion || key == AttrDomain {
			continue
		}
		value := strings.TrimSpace(in[key])
		if value == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func NormalizeSemanticSpans(in []SemanticSpan) []SemanticSpan {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]SemanticSpan{}
	for _, raw := range in {
		spec, ok := CanonicalSemconvSpec(raw.Domain)
		if !ok {
			continue
		}
		parent := strings.ToLower(strings.TrimSpace(raw.Parent))
		if parent == "" {
			parent = spec.ParentDomain
		}
		normalized := SemanticSpan{
			Domain:     spec.Domain,
			Parent:     parent,
			Attributes: CanonicalAttributeMap(spec.Domain, raw.Attributes),
		}
		fingerprint := semanticSpanFingerprint(normalized)
		seen[fingerprint] = normalized
	}
	out := make([]SemanticSpan, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	slices.SortFunc(out, func(a, b SemanticSpan) int {
		return strings.Compare(semanticSpanFingerprint(a), semanticSpanFingerprint(b))
	})
	return out
}

func SemanticallyEquivalentSpans(left, right []SemanticSpan) bool {
	lhs := NormalizeSemanticSpans(left)
	rhs := NormalizeSemanticSpans(right)
	if len(lhs) != len(rhs) {
		return false
	}
	for i := range lhs {
		if semanticSpanFingerprint(lhs[i]) != semanticSpanFingerprint(rhs[i]) {
			return false
		}
	}
	return true
}

func semanticSpanFingerprint(span SemanticSpan) string {
	keys := make([]string, 0, len(span.Attributes))
	for key := range span.Attributes {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys)+2)
	parts = append(parts, strings.TrimSpace(span.Domain))
	parts = append(parts, strings.TrimSpace(span.Parent))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.TrimSpace(span.Attributes[key]))
	}
	return strings.Join(parts, "|")
}
