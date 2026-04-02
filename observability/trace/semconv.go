package trace

import (
	"maps"
	"slices"
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
	AttrTraceSchemaVersion = "trace.schema_version"
	AttrDomain             = "trace.domain"
	AttrRunID              = "run.id"
	AttrMode               = "run.mode"
	AttrStepID             = "step.id"
	AttrToolName           = "tool.name"
	AttrMCPTransport       = "mcp.transport"
	AttrMemoryScope        = "memory_scope_selected"
	AttrBudgetDecision     = "budget_decision"
	AttrPolicyDecisionPath = "policy_decision_path"
)

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
