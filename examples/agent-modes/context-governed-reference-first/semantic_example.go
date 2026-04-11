package contextgovernedreferencefirst

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	modecommon "github.com/FelixSeptem/baymax/examples/agent-modes/internal/modecommon"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

const (
	patternName      = "context-governed-reference-first"
	phase            = "P0"
	semanticAnchor   = "context.reference_first_isolate_edit_tiering"
	classification   = "context.reference_first_governance"
	semanticToolName = "mode_context_governed_reference_first_semantic_step"
)

type contextStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type referenceRecord struct {
	ID          string
	Source      string
	Text        string
	Freshness   int
	Mutable     bool
	Sensitivity string
}

type editCandidate struct {
	Target string
	Op     string
	Risk   string
	Reason string
}

type contextState struct {
	Query              string
	SelectedRefs       []string
	IsolatedRefs       []string
	IsolationPolicy    string
	EditDecision       string
	AllowedEdits       []string
	BlockedEdits       []string
	TierHot            []string
	TierWarm           []string
	TierCold           []string
	GovernanceDecision string
	ReplayBinding      string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"context/assembler", "context/guard", "context/journal"}

var minimalSemanticSteps = []contextStep{
	{
		Marker:        "context_reference_first_selected",
		RuntimeDomain: "context/assembler",
		Intent:        "select immutable reference evidence before any direct edit intent",
		Outcome:       "reference-first candidate set is produced for downstream isolation",
	},
	{
		Marker:        "context_isolate_handoff_applied",
		RuntimeDomain: "context/guard",
		Intent:        "isolate mutable or sensitive segments before context handoff",
		Outcome:       "handoff receives isolated references only",
	},
	{
		Marker:        "context_edit_gate_evaluated",
		RuntimeDomain: "context/journal",
		Intent:        "evaluate edit requests against isolated references and guard policy",
		Outcome:       "edit gate decision is emitted with allowed and blocked edits",
	},
}

var productionGovernanceSteps = []contextStep{
	{
		Marker:        "governance_context_tiering_enforced",
		RuntimeDomain: "context/assembler",
		Intent:        "enforce hot/warm/cold tiering for isolated reference bundle",
		Outcome:       "tiering decision is persisted as governance evidence",
	},
	{
		Marker:        "governance_context_replay_bound",
		RuntimeDomain: "context/guard",
		Intent:        "bind replay signature to tiering and edit gate governance decisions",
		Outcome:       "deterministic replay signature is emitted",
	},
}

var referenceCorpus = []referenceRecord{
	{
		ID:          "ref-runtime-config",
		Source:      "docs/runtime-config-diagnostics.md",
		Text:        "runtime config hot reload rollback fail fast and diagnostics",
		Freshness:   9,
		Mutable:     false,
		Sensitivity: "internal",
	},
	{
		ID:          "ref-context-policy",
		Source:      "docs/runtime-module-boundaries.md",
		Text:        "context governance isolate handoff and boundary policy",
		Freshness:   8,
		Mutable:     false,
		Sensitivity: "internal",
	},
	{
		ID:          "ref-operational-note",
		Source:      "ops/internal-notes.md",
		Text:        "operational rollback note and patch checklist",
		Freshness:   6,
		Mutable:     true,
		Sensitivity: "internal",
	},
	{
		ID:          "ref-security-redline",
		Source:      "security/redline.md",
		Text:        "security policy deny list and restricted escalation protocol",
		Freshness:   7,
		Mutable:     false,
		Sensitivity: "restricted",
	},
	{
		ID:          "ref-observability",
		Source:      "docs/mainline-contract-test-index.md",
		Text:        "observability replay contract baseline and gate coverage",
		Freshness:   7,
		Mutable:     false,
		Sensitivity: "internal",
	},
}

var minimalEdits = []editCandidate{
	{Target: "derived_summary", Op: "append_context", Risk: "low", Reason: "attach concise summary for responder"},
	{Target: "ref-operational-note", Op: "overwrite_source", Risk: "high", Reason: "attempt direct edit on mutable note"},
}

var productionEdits = []editCandidate{
	{Target: "derived_summary", Op: "append_context", Risk: "low", Reason: "attach governed summary for execution agent"},
	{Target: "ref-runtime-config", Op: "annotate_reference", Risk: "medium", Reason: "add pointer without overwriting source"},
	{Target: "ref-security-redline", Op: "overwrite_source", Risk: "high", Reason: "forbidden direct overwrite on restricted source"},
}

func RunMinimal() {
	executeVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeVariant(modecommon.VariantProduction)
}

func executeVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&referenceFirstGovernanceTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(newContextWorkflowModel(variant), runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute context governed reference first workflow",
	}, nil)
	if err != nil {
		panic(err)
	}

	expected := expectedMarkersForVariant(variant)
	runtimePath := modecommon.ComposeRuntimePath(runtimeDomains)
	pathStatus := modecommon.RuntimePathStatus(result.ToolCalls, len(expected))
	governanceStatus := "baseline"
	if variant == modecommon.VariantProduction {
		governanceStatus = "enforced"
	}

	fmt.Println("agent-mode example")
	fmt.Printf("pattern=%s\n", patternName)
	fmt.Printf("variant=%s\n", variant)
	fmt.Printf("runtime.path=%s\n", strings.Join(runtimePath, ","))
	fmt.Printf("verification.mainline_runtime_path=%s\n", pathStatus)
	fmt.Printf("verification.semantic.phase=%s\n", phase)
	fmt.Printf("verification.semantic.anchor=%s\n", semanticAnchor)
	fmt.Printf("verification.semantic.classification=%s\n", classification)
	fmt.Printf("verification.semantic.runtime_path=%s\n", strings.Join(runtimePath, ","))
	fmt.Printf("verification.semantic.expected_markers=%s\n", strings.Join(expected, ","))
	fmt.Printf("verification.semantic.governance=%s\n", governanceStatus)
	fmt.Printf("verification.semantic.marker_count=%d\n", len(expected))
	for _, marker := range expected {
		fmt.Printf("verification.semantic.marker.%s=ok\n", modecommon.MarkerToken(marker))
	}
	fmt.Printf("result.tool_calls=%d\n", len(result.ToolCalls))
	fmt.Printf("result.final_answer=%s\n", result.FinalAnswer)
	fmt.Printf("result.signature=%d\n", modecommon.ComputeSignature(result.FinalAnswer, result.ToolCalls))
}

func expectedMarkersForVariant(variant string) []string {
	markers := make([]string, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	for _, step := range minimalSemanticSteps {
		markers = append(markers, step.Marker)
	}
	if variant == modecommon.VariantProduction {
		for _, step := range productionGovernanceSteps {
			markers = append(markers, step.Marker)
		}
	}
	return markers
}

func stepsForVariant(variant string) []contextStep {
	steps := make([]contextStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

func newContextWorkflowModel(variant string) *contextWorkflowModel {
	return &contextWorkflowModel{
		variant: variant,
		state: contextState{
			Query: queryForVariant(variant),
		},
	}
}

type contextWorkflowModel struct {
	variant string
	stage   int
	state   contextState
}

func (m *contextWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.absorb(req.ToolResult)

	plan := stepsForVariant(m.variant)
	if m.stage < len(plan) {
		step := plan[m.stage]
		call := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.stage+1),
			Name:   "local." + semanticToolName,
			Args:   m.argsForStep(step, m.stage+1),
		}
		m.stage++
		return types.ModelResponse{ToolCalls: []types.ToolCall{call}}, nil
	}

	markers := append([]string(nil), m.state.SeenMarkers...)
	sort.Strings(markers)
	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s query=%s selected=%s isolated=%s edit_decision=%s allowed=%s blocked=%s tier_hot=%s tier_warm=%s tier_cold=%s governance=%s replay=%s markers=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.Query,
		strings.Join(m.state.SelectedRefs, ","),
		strings.Join(m.state.IsolatedRefs, ","),
		safeString(m.state.EditDecision, "none"),
		strings.Join(m.state.AllowedEdits, ","),
		strings.Join(m.state.BlockedEdits, ","),
		strings.Join(m.state.TierHot, ","),
		strings.Join(m.state.TierWarm, ","),
		strings.Join(m.state.TierCold, ","),
		normalizedDecision(m.state.GovernanceDecision),
		safeString(m.state.ReplayBinding, "none"),
		strings.Join(markers, ","),
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *contextWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}
func (m *contextWorkflowModel) absorb(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if query, ok := item.Result.Structured["query"].(string); ok && strings.TrimSpace(query) != "" {
			m.state.Query = query
		}
		if refs, ok := toStringSlice(item.Result.Structured["selected_refs"]); ok {
			m.state.SelectedRefs = refs
		}
		if refs, ok := toStringSlice(item.Result.Structured["isolated_refs"]); ok {
			m.state.IsolatedRefs = refs
		}
		if policy, ok := item.Result.Structured["isolation_policy"].(string); ok && strings.TrimSpace(policy) != "" {
			m.state.IsolationPolicy = policy
		}
		if decision, ok := item.Result.Structured["edit_decision"].(string); ok && strings.TrimSpace(decision) != "" {
			m.state.EditDecision = decision
		}
		if edits, ok := toStringSlice(item.Result.Structured["allowed_edits"]); ok {
			m.state.AllowedEdits = edits
		}
		if edits, ok := toStringSlice(item.Result.Structured["blocked_edits"]); ok {
			m.state.BlockedEdits = edits
		}
		if refs, ok := toStringSlice(item.Result.Structured["tier_hot"]); ok {
			m.state.TierHot = refs
		}
		if refs, ok := toStringSlice(item.Result.Structured["tier_warm"]); ok {
			m.state.TierWarm = refs
		}
		if refs, ok := toStringSlice(item.Result.Structured["tier_cold"]); ok {
			m.state.TierCold = refs
		}
		if governanceDecision, ok := item.Result.Structured["governance_decision"].(string); ok && strings.TrimSpace(governanceDecision) != "" {
			m.state.GovernanceDecision = governanceDecision
		}
		if replayBinding, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replayBinding) != "" {
			m.state.ReplayBinding = replayBinding
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *contextWorkflowModel) argsForStep(step contextStep, stage int) map[string]any {
	args := map[string]any{
		"pattern":             patternName,
		"variant":             m.variant,
		"phase":               phase,
		"semantic_anchor":     semanticAnchor,
		"classification":      classification,
		"marker":              step.Marker,
		"runtime_domain":      step.RuntimeDomain,
		"semantic_intent":     step.Intent,
		"semantic_outcome":    step.Outcome,
		"query":               m.state.Query,
		"reference_specs":     toAnySlice(referenceSpecs()),
		"edit_specs":          toAnySlice(editSpecsForVariant(m.variant)),
		"selected_refs":       toAnySlice(m.state.SelectedRefs),
		"isolated_refs":       toAnySlice(m.state.IsolatedRefs),
		"isolation_policy":    m.state.IsolationPolicy,
		"edit_decision":       m.state.EditDecision,
		"allowed_edits":       toAnySlice(m.state.AllowedEdits),
		"blocked_edits":       toAnySlice(m.state.BlockedEdits),
		"tier_hot":            toAnySlice(m.state.TierHot),
		"tier_warm":           toAnySlice(m.state.TierWarm),
		"tier_cold":           toAnySlice(m.state.TierCold),
		"governance_decision": m.state.GovernanceDecision,
		"stage":               stage,
	}
	return args
}

type referenceFirstGovernanceTool struct{}

func (t *referenceFirstGovernanceTool) Name() string { return semanticToolName }

func (t *referenceFirstGovernanceTool) Description() string {
	return "execute context reference-first governance semantic step"
}

func (t *referenceFirstGovernanceTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []any{
			"pattern",
			"variant",
			"phase",
			"semantic_anchor",
			"classification",
			"marker",
			"runtime_domain",
			"semantic_intent",
			"semantic_outcome",
			"query",
			"reference_specs",
			"edit_specs",
			"stage",
		},
		"properties": map[string]any{
			"pattern":             map[string]any{"type": "string"},
			"variant":             map[string]any{"type": "string"},
			"phase":               map[string]any{"type": "string"},
			"semantic_anchor":     map[string]any{"type": "string"},
			"classification":      map[string]any{"type": "string"},
			"marker":              map[string]any{"type": "string"},
			"runtime_domain":      map[string]any{"type": "string"},
			"semantic_intent":     map[string]any{"type": "string"},
			"semantic_outcome":    map[string]any{"type": "string"},
			"query":               map[string]any{"type": "string"},
			"reference_specs":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"edit_specs":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"selected_refs":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"isolated_refs":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"isolation_policy":    map[string]any{"type": "string"},
			"edit_decision":       map[string]any{"type": "string"},
			"allowed_edits":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"blocked_edits":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"tier_hot":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"tier_warm":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"tier_cold":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"governance_decision": map[string]any{"type": "string"},
			"stage":               map[string]any{"type": "integer"},
		},
	}
}

func (t *referenceFirstGovernanceTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx

	pattern := strings.TrimSpace(fmt.Sprintf("%v", args["pattern"]))
	variant := strings.TrimSpace(fmt.Sprintf("%v", args["variant"]))
	phaseValue := strings.TrimSpace(fmt.Sprintf("%v", args["phase"]))
	anchor := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_anchor"]))
	classValue := strings.TrimSpace(fmt.Sprintf("%v", args["classification"]))
	marker := strings.TrimSpace(fmt.Sprintf("%v", args["marker"]))
	runtimeDomain := strings.TrimSpace(fmt.Sprintf("%v", args["runtime_domain"]))
	intent := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_intent"]))
	outcome := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_outcome"]))
	query := strings.TrimSpace(fmt.Sprintf("%v", args["query"]))
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	refSpecs, _ := toStringSlice(args["reference_specs"])
	references := parseReferenceSpecs(refSpecs)
	refIndex := mapReferenceByID(references)

	editSpecs, _ := toStringSlice(args["edit_specs"])
	edits := parseEditSpecs(editSpecs)

	selectedRefs, _ := toStringSlice(args["selected_refs"])
	isolatedRefs, _ := toStringSlice(args["isolated_refs"])
	allowedEdits, _ := toStringSlice(args["allowed_edits"])
	blockedEdits, _ := toStringSlice(args["blocked_edits"])
	tierHot, _ := toStringSlice(args["tier_hot"])
	tierWarm, _ := toStringSlice(args["tier_warm"])
	tierCold, _ := toStringSlice(args["tier_cold"])

	result := map[string]any{
		"pattern":          pattern,
		"variant":          variant,
		"phase":            phaseValue,
		"semantic_anchor":  anchor,
		"classification":   classValue,
		"marker":           marker,
		"runtime_domain":   runtimeDomain,
		"semantic_intent":  intent,
		"semantic_outcome": outcome,
		"query":            query,
		"stage":            stage,
		"governance":       false,
	}

	var risk string
	editDecision := safeString(args["edit_decision"], "none")
	governanceDecision := safeString(args["governance_decision"], "")

	switch marker {
	case "context_reference_first_selected":
		selectedRefs, droppedRefs := selectReferenceFirst(query, references, 3)
		result["selected_refs"] = toAnySlice(selectedRefs)
		result["dropped_refs"] = toAnySlice(droppedRefs)
		result["isolation_policy"] = "reference_first_prefilter"
		if len(droppedRefs) > 0 {
			risk = "reference_filtered"
		} else {
			risk = "reference_selected"
		}
	case "context_isolate_handoff_applied":
		if len(selectedRefs) == 0 {
			selectedRefs, _ = selectReferenceFirst(query, references, 3)
		}
		isolatedRefs, droppedRefs := isolateForHandoff(selectedRefs, refIndex)
		policy := "isolated_handoff"
		if len(isolatedRefs) == 0 && len(selectedRefs) > 0 {
			isolatedRefs = append(isolatedRefs, selectedRefs[0])
			policy = "fallback_single_reference"
		}
		result["selected_refs"] = toAnySlice(selectedRefs)
		result["isolated_refs"] = toAnySlice(isolatedRefs)
		result["dropped_refs"] = toAnySlice(droppedRefs)
		result["isolation_policy"] = policy
		if len(droppedRefs) > 0 {
			risk = "isolation_trimmed"
		} else {
			risk = "isolation_clean"
		}
	case "context_edit_gate_evaluated":
		if len(isolatedRefs) == 0 {
			isolatedRefs = append([]string(nil), selectedRefs...)
		}
		allowedEdits, blockedEdits, editDecision := evaluateEditGate(edits, isolatedRefs)
		result["isolated_refs"] = toAnySlice(isolatedRefs)
		result["allowed_edits"] = toAnySlice(allowedEdits)
		result["blocked_edits"] = toAnySlice(blockedEdits)
		result["edit_decision"] = editDecision
		result["isolation_policy"] = safeString(args["isolation_policy"], "isolated_handoff")
		switch {
		case editDecision == "block":
			risk = "degraded_path"
		case len(blockedEdits) > 0:
			risk = "edit_guarded"
		default:
			risk = "edit_allowed"
		}
	case "governance_context_tiering_enforced":
		if len(isolatedRefs) == 0 {
			isolatedRefs = append([]string(nil), selectedRefs...)
		}
		tierHot, tierWarm, tierCold = assignTiers(isolatedRefs, refIndex)
		governanceDecision = "allow"
		switch {
		case len(tierHot) == 0:
			governanceDecision = "block_missing_hot_tier"
			risk = "governed_block"
		case len(blockedEdits) > 0:
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		default:
			risk = "governed_allow"
		}
		result["isolated_refs"] = toAnySlice(isolatedRefs)
		result["allowed_edits"] = toAnySlice(allowedEdits)
		result["blocked_edits"] = toAnySlice(blockedEdits)
		result["tier_hot"] = toAnySlice(tierHot)
		result["tier_warm"] = toAnySlice(tierWarm)
		result["tier_cold"] = toAnySlice(tierCold)
		result["governance_decision"] = governanceDecision
		result["edit_decision"] = safeString(args["edit_decision"], "none")
		result["governance"] = true
	case "governance_context_replay_bound":
		governanceDecision = safeString(governanceDecision, "allow")
		replayBinding := fmt.Sprintf(
			"context-replay-%d",
			modecommon.SemanticScore(
				pattern,
				variant,
				query,
				strings.Join(uniqueSorted(selectedRefs), ","),
				strings.Join(uniqueSorted(isolatedRefs), ","),
				safeString(args["edit_decision"], "none"),
				governanceDecision,
				strings.Join(uniqueSorted(tierHot), ","),
				strings.Join(uniqueSorted(tierWarm), ","),
				strings.Join(uniqueSorted(tierCold), ","),
			),
		)
		result["selected_refs"] = toAnySlice(uniqueSorted(selectedRefs))
		result["isolated_refs"] = toAnySlice(uniqueSorted(isolatedRefs))
		result["allowed_edits"] = toAnySlice(uniqueSorted(allowedEdits))
		result["blocked_edits"] = toAnySlice(uniqueSorted(blockedEdits))
		result["tier_hot"] = toAnySlice(uniqueSorted(tierHot))
		result["tier_warm"] = toAnySlice(uniqueSorted(tierWarm))
		result["tier_cold"] = toAnySlice(uniqueSorted(tierCold))
		result["edit_decision"] = safeString(args["edit_decision"], "none")
		result["governance_decision"] = governanceDecision
		result["replay_binding"] = replayBinding
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported context reference-first marker: %s", marker)
	}

	result["score"] = modecommon.SemanticScore(
		pattern,
		variant,
		phaseValue,
		anchor,
		classValue,
		marker,
		runtimeDomain,
		risk,
		safeString(result["edit_decision"], editDecision),
		safeString(result["governance_decision"], governanceDecision),
	)
	result["risk"] = risk

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s isolation=%s edit=%s governance=%s risk=%s",
		pattern,
		variant,
		marker,
		safeString(result["isolation_policy"], safeString(args["isolation_policy"], "none")),
		safeString(result["edit_decision"], editDecision),
		normalizedDecision(safeString(result["governance_decision"], governanceDecision)),
		risk,
	)

	return types.ToolResult{Content: content, Structured: result}, nil
}
func queryForVariant(variant string) string {
	if variant == modecommon.VariantProduction {
		return "reference first isolate restricted policy before edit"
	}
	return "reference first context isolate edit rollback"
}

func referenceSpecs() []string {
	specs := make([]string, 0, len(referenceCorpus))
	for _, ref := range referenceCorpus {
		specs = append(
			specs,
			fmt.Sprintf(
				"%s|%s|%d|%t|%s|%s",
				ref.ID,
				ref.Source,
				ref.Freshness,
				ref.Mutable,
				ref.Sensitivity,
				ref.Text,
			),
		)
	}
	return specs
}

func editSpecsForVariant(variant string) []string {
	edits := minimalEdits
	if variant == modecommon.VariantProduction {
		edits = productionEdits
	}
	specs := make([]string, 0, len(edits))
	for _, edit := range edits {
		specs = append(specs, fmt.Sprintf("%s|%s|%s|%s", edit.Target, edit.Op, edit.Risk, edit.Reason))
	}
	return specs
}

func parseReferenceSpecs(specs []string) []referenceRecord {
	refs := make([]referenceRecord, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, "|", 6)
		if len(parts) != 6 {
			continue
		}
		refs = append(refs, referenceRecord{
			ID:          strings.TrimSpace(parts[0]),
			Source:      strings.TrimSpace(parts[1]),
			Freshness:   parseInt(parts[2]),
			Mutable:     parseBool(parts[3], false),
			Sensitivity: strings.TrimSpace(parts[4]),
			Text:        strings.TrimSpace(parts[5]),
		})
	}
	if len(refs) == 0 {
		refs = append(refs, referenceCorpus...)
	}
	return refs
}

func parseEditSpecs(specs []string) []editCandidate {
	edits := make([]editCandidate, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, "|", 4)
		if len(parts) != 4 {
			continue
		}
		edits = append(edits, editCandidate{
			Target: strings.TrimSpace(parts[0]),
			Op:     strings.TrimSpace(parts[1]),
			Risk:   strings.TrimSpace(parts[2]),
			Reason: strings.TrimSpace(parts[3]),
		})
	}
	return edits
}

func mapReferenceByID(refs []referenceRecord) map[string]referenceRecord {
	index := make(map[string]referenceRecord, len(refs))
	for _, ref := range refs {
		index[ref.ID] = ref
	}
	return index
}

func selectReferenceFirst(query string, refs []referenceRecord, budget int) ([]string, []string) {
	if budget <= 0 {
		budget = 3
	}
	terms := tokenize(query)
	sourceWeight := map[string]int{
		"docs/runtime-config-diagnostics.md":   9,
		"docs/runtime-module-boundaries.md":    8,
		"docs/mainline-contract-test-index.md": 7,
		"security/redline.md":                  6,
		"ops/internal-notes.md":                4,
	}
	type scoredRef struct {
		ID    string
		Score int
	}
	scored := make([]scoredRef, 0, len(refs))
	for _, ref := range refs {
		score := ref.Freshness * 10
		score += sourceWeight[ref.Source]
		if !ref.Mutable {
			score += 20
		}
		if ref.Sensitivity == "restricted" {
			score -= 15
		}
		text := strings.ToLower(ref.Text)
		for _, term := range terms {
			if term == "" {
				continue
			}
			if strings.Contains(text, term) {
				score += 5
			}
		}
		scored = append(scored, scoredRef{ID: ref.ID, Score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].ID < scored[j].ID
		}
		return scored[i].Score > scored[j].Score
	})

	selected := make([]string, 0, budget)
	dropped := make([]string, 0)
	for idx, item := range scored {
		if idx < budget {
			selected = append(selected, item.ID)
			continue
		}
		dropped = append(dropped, item.ID)
	}
	if len(selected) == 0 && len(scored) > 0 {
		selected = append(selected, scored[0].ID)
	}
	return uniqueSorted(selected), uniqueSorted(dropped)
}

func isolateForHandoff(selected []string, refIndex map[string]referenceRecord) ([]string, []string) {
	isolated := make([]string, 0, len(selected))
	dropped := make([]string, 0)
	for _, id := range selected {
		ref, ok := refIndex[id]
		if !ok {
			dropped = append(dropped, id+":missing_reference")
			continue
		}
		if ref.Mutable {
			dropped = append(dropped, id+":mutable")
			continue
		}
		if ref.Sensitivity == "restricted" {
			dropped = append(dropped, id+":restricted")
			continue
		}
		isolated = append(isolated, id)
	}
	return uniqueSorted(isolated), uniqueSorted(dropped)
}

func evaluateEditGate(edits []editCandidate, isolated []string) ([]string, []string, string) {
	isolatedSet := map[string]struct{}{}
	for _, id := range isolated {
		isolatedSet[id] = struct{}{}
	}
	allowed := make([]string, 0)
	blocked := make([]string, 0)
	for _, edit := range edits {
		label := fmt.Sprintf("%s:%s", edit.Target, edit.Op)
		risk := strings.ToLower(strings.TrimSpace(edit.Risk))
		if risk == "high" {
			blocked = append(blocked, label+":risk_high")
			continue
		}
		if edit.Op == "overwrite_source" {
			blocked = append(blocked, label+":overwrite_forbidden")
			continue
		}
		if edit.Target != "derived_summary" {
			if _, ok := isolatedSet[edit.Target]; !ok {
				blocked = append(blocked, label+":out_of_isolation_scope")
				continue
			}
		}
		allowed = append(allowed, label)
	}
	allowed = uniqueSorted(allowed)
	blocked = uniqueSorted(blocked)
	decision := "allow"
	if len(allowed) == 0 && len(blocked) > 0 {
		decision = "block"
	} else if len(blocked) > 0 {
		decision = "allow_with_guard"
	}
	return allowed, blocked, decision
}

func assignTiers(isolated []string, refIndex map[string]referenceRecord) ([]string, []string, []string) {
	hot := make([]string, 0)
	warm := make([]string, 0)
	cold := make([]string, 0)
	for _, id := range isolated {
		ref, ok := refIndex[id]
		if !ok {
			continue
		}
		if ref.Sensitivity == "restricted" || ref.Freshness <= 5 {
			cold = append(cold, id)
			continue
		}
		if ref.Freshness >= 8 {
			hot = append(hot, id)
			continue
		}
		warm = append(warm, id)
	}
	hot = uniqueSorted(hot)
	warm = uniqueSorted(warm)
	cold = uniqueSorted(cold)
	if len(hot) == 0 && len(warm) > 0 {
		hot = append(hot, warm[0])
		warm = warm[1:]
	}
	return hot, warm, cold
}

func toStringSlice(value any) ([]string, bool) {
	switch raw := value.(type) {
	case []string:
		return append([]string(nil), raw...), true
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text == "" || text == "<nil>" {
				continue
			}
			out = append(out, text)
		}
		return out, true
	default:
		return nil, false
	}
}

func toAnySlice(in []string) []any {
	if len(in) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}

func uniqueSorted(in []string) []string {
	set := map[string]struct{}{}
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func parseBool(value any, fallback bool) bool {
	switch raw := value.(type) {
	case bool:
		return raw
	default:
		text := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", value)))
		if text == "" || text == "<nil>" {
			return fallback
		}
		switch text {
		case "1", "true", "yes", "y":
			return true
		case "0", "false", "no", "n":
			return false
		default:
			return fallback
		}
	}
}

func parseInt(raw string) int {
	sign := 1
	value := 0
	for idx, char := range raw {
		if idx == 0 && char == '-' {
			sign = -1
			continue
		}
		if char < '0' || char > '9' {
			continue
		}
		value = value*10 + int(char-'0')
	}
	return sign * value
}

func safeString(value any, fallback string) string {
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func normalizedDecision(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not_applicable"
	}
	return value
}

func tokenize(query string) []string {
	normalized := strings.ToLower(query)
	replacer := strings.NewReplacer(
		",", " ",
		".", " ",
		";", " ",
		":", " ",
		"|", " ",
		"\n", " ",
		"\t", " ",
	)
	normalized = replacer.Replace(normalized)
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return []string{"reference", "context"}
	}
	return parts
}
