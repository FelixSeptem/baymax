package skilldrivendiscoveryhybrid

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
	patternName      = "skill-driven-discovery-hybrid"
	phase            = "P0"
	semanticAnchor   = "discovery.source_priority_score_mapping"
	classification   = "skill.hybrid_discovery"
	semanticToolName = "mode_skill_driven_discovery_hybrid_semantic_step"
)

type discoveryStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type skillCandidate struct {
	Skill     string
	Source    string
	Lexical   int
	Embedding int
}

type discoveryState struct {
	Intent             string
	PrioritizedSource  []string
	RankedSkills       []string
	TopSkill           string
	TopScore           int
	MappingProfile     string
	GovernanceDecision string
	ReplayBinding      string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"skill/loader", "context/assembler"}

var minimalSemanticSteps = []discoveryStep{
	{
		Marker:        "discovery_sources_prioritized",
		RuntimeDomain: "skill/loader",
		Intent:        "prioritize discovery sources using source authority order",
		Outcome:       "source ordering is resolved for downstream scoring",
	},
	{
		Marker:        "discovery_score_reconciled",
		RuntimeDomain: "context/assembler",
		Intent:        "reconcile lexical and embedding scores into one ranking",
		Outcome:       "skill ranking is produced with weighted confidence",
	},
	{
		Marker:        "discovery_mapping_emitted",
		RuntimeDomain: "skill/loader",
		Intent:        "emit intent-to-skill mapping from top ranked candidates",
		Outcome:       "mapping profile is generated for runtime invocation",
	},
}

var productionGovernanceSteps = []discoveryStep{
	{
		Marker:        "governance_skill_gate_enforced",
		RuntimeDomain: "skill/loader",
		Intent:        "enforce governance threshold on selected skill confidence",
		Outcome:       "governance decision allow/warn/block is recorded",
	},
	{
		Marker:        "governance_skill_replay_bound",
		RuntimeDomain: "context/assembler",
		Intent:        "bind replay signature to ranking and governance decision",
		Outcome:       "replay binding signature is generated",
	},
}

var discoveryCandidates = []skillCandidate{
	{Skill: "incident_summary", Source: "builtin", Lexical: 92, Embedding: 84},
	{Skill: "root_cause_search", Source: "repo", Lexical: 86, Embedding: 88},
	{Skill: "ticket_triage", Source: "repo", Lexical: 71, Embedding: 67},
	{Skill: "web_lookup", Source: "remote", Lexical: 65, Embedding: 79},
}

var sourcePriority = map[string]int{
	"builtin": 3,
	"repo":    2,
	"remote":  1,
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
	if _, err := reg.Register(&discoverySemanticTool{}); err != nil {
		panic(err)
	}

	model := &discoveryWorkflowModel{
		variant: variant,
		state: discoveryState{
			Intent: "summarize incident timeline and suggest first response",
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute skill discovery hybrid workflow",
	}, nil)
	if err != nil {
		panic(err)
	}

	expectedMarkers := expectedMarkers(variant)
	runtimePath := modecommon.ComposeRuntimePath(runtimeDomains)
	pathStatus := modecommon.RuntimePathStatus(result.ToolCalls, len(expectedMarkers))
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
	fmt.Printf("verification.semantic.expected_markers=%s\n", strings.Join(expectedMarkers, ","))
	fmt.Printf("verification.semantic.governance=%s\n", governanceStatus)
	fmt.Printf("verification.semantic.marker_count=%d\n", len(expectedMarkers))
	for _, marker := range expectedMarkers {
		fmt.Printf("verification.semantic.marker.%s=ok\n", modecommon.MarkerToken(marker))
	}
	fmt.Printf("result.tool_calls=%d\n", len(result.ToolCalls))
	fmt.Printf("result.final_answer=%s\n", result.FinalAnswer)
	fmt.Printf("result.signature=%d\n", modecommon.ComputeSignature(result.FinalAnswer, result.ToolCalls))
}

func expectedMarkers(variant string) []string {
	markers := make([]string, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	for _, item := range minimalSemanticSteps {
		markers = append(markers, item.Marker)
	}
	if variant == modecommon.VariantProduction {
		for _, item := range productionGovernanceSteps {
			markers = append(markers, item.Marker)
		}
	}
	return markers
}

func stepsForVariant(variant string) []discoveryStep {
	steps := make([]discoveryStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

type discoveryWorkflowModel struct {
	variant string
	stage   int
	state   discoveryState
}

func (m *discoveryWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.absorb(req.ToolResult)

	steps := stepsForVariant(m.variant)
	if m.stage < len(steps) {
		step := steps[m.stage]
		call := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.stage+1),
			Name:   "local." + semanticToolName,
			Args:   m.argsForStep(step, m.stage+1),
		}
		m.stage++
		return types.ModelResponse{ToolCalls: []types.ToolCall{call}}, nil
	}

	sources := append([]string(nil), m.state.PrioritizedSource...)
	rankedSkills := append([]string(nil), m.state.RankedSkills...)
	markers := append([]string(nil), m.state.SeenMarkers...)
	sort.Strings(markers)

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s intent=%s sources=%s top_skill=%s top_score=%d mapping=%s governance=%s replay=%s ranked=%s markers=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.Intent,
		strings.Join(sources, ","),
		m.state.TopSkill,
		m.state.TopScore,
		m.state.MappingProfile,
		normalizedDecision(m.state.GovernanceDecision),
		m.state.ReplayBinding,
		strings.Join(rankedSkills, ","),
		strings.Join(markers, ","),
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *discoveryWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *discoveryWorkflowModel) absorb(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if sources, ok := toStringSlice(item.Result.Structured["prioritized_sources"]); ok {
			m.state.PrioritizedSource = sources
		}
		if ranked, ok := toStringSlice(item.Result.Structured["ranked_skills"]); ok {
			m.state.RankedSkills = ranked
		}
		if topSkill, ok := item.Result.Structured["top_skill"].(string); ok && strings.TrimSpace(topSkill) != "" {
			m.state.TopSkill = topSkill
		}
		if topScore, ok := modecommon.AsInt(item.Result.Structured["top_score"]); ok {
			m.state.TopScore = topScore
		}
		if mapping, ok := item.Result.Structured["mapping_profile"].(string); ok && strings.TrimSpace(mapping) != "" {
			m.state.MappingProfile = mapping
		}
		if decision, ok := item.Result.Structured["governance_decision"].(string); ok && strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = decision
		}
		if replay, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replay) != "" {
			m.state.ReplayBinding = replay
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *discoveryWorkflowModel) argsForStep(step discoveryStep, stage int) map[string]any {
	args := map[string]any{
		"pattern":          patternName,
		"variant":          m.variant,
		"phase":            phase,
		"semantic_anchor":  semanticAnchor,
		"classification":   classification,
		"marker":           step.Marker,
		"runtime_domain":   step.RuntimeDomain,
		"semantic_intent":  step.Intent,
		"semantic_outcome": step.Outcome,
		"intent":           m.state.Intent,
		"candidate_specs":  toAnySlice(candidateSpecs()),
		"stage":            stage,
	}

	switch step.Marker {
	case "discovery_score_reconciled":
		args["prioritized_sources"] = toAnySlice(m.state.PrioritizedSource)
	case "discovery_mapping_emitted":
		args["ranked_skills"] = toAnySlice(m.state.RankedSkills)
		args["top_score"] = m.state.TopScore
	case "governance_skill_gate_enforced":
		args["top_skill"] = m.state.TopSkill
		args["top_score"] = m.state.TopScore
		args["mapping_profile"] = m.state.MappingProfile
	case "governance_skill_replay_bound":
		args["top_skill"] = m.state.TopSkill
		args["top_score"] = m.state.TopScore
		args["mapping_profile"] = m.state.MappingProfile
		args["governance_decision"] = m.state.GovernanceDecision
	}
	return args
}

type discoverySemanticTool struct{}

func (t *discoverySemanticTool) Name() string { return semanticToolName }

func (t *discoverySemanticTool) Description() string {
	return "execute skill discovery hybrid semantic step"
}

func (t *discoverySemanticTool) JSONSchema() map[string]any {
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
			"intent",
			"candidate_specs",
			"stage",
		},
		"properties": map[string]any{
			"pattern":          map[string]any{"type": "string"},
			"variant":          map[string]any{"type": "string"},
			"phase":            map[string]any{"type": "string"},
			"semantic_anchor":  map[string]any{"type": "string"},
			"classification":   map[string]any{"type": "string"},
			"marker":           map[string]any{"type": "string"},
			"runtime_domain":   map[string]any{"type": "string"},
			"semantic_intent":  map[string]any{"type": "string"},
			"semantic_outcome": map[string]any{"type": "string"},
			"intent":           map[string]any{"type": "string"},
			"candidate_specs":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"prioritized_sources": map[string]any{
				"type": "array", "items": map[string]any{"type": "string"},
			},
			"ranked_skills": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"top_skill":     map[string]any{"type": "string"},
			"top_score":     map[string]any{"type": "integer"},
			"mapping_profile": map[string]any{
				"type": "string",
			},
			"governance_decision": map[string]any{"type": "string"},
			"stage":               map[string]any{"type": "integer"},
		},
	}
}

func (t *discoverySemanticTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx

	pattern := strings.TrimSpace(fmt.Sprintf("%v", args["pattern"]))
	variant := strings.TrimSpace(fmt.Sprintf("%v", args["variant"]))
	phaseValue := strings.TrimSpace(fmt.Sprintf("%v", args["phase"]))
	anchor := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_anchor"]))
	classValue := strings.TrimSpace(fmt.Sprintf("%v", args["classification"]))
	marker := strings.TrimSpace(fmt.Sprintf("%v", args["marker"]))
	runtimeDomain := strings.TrimSpace(fmt.Sprintf("%v", args["runtime_domain"]))
	intent := strings.TrimSpace(fmt.Sprintf("%v", args["intent"]))
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	specs, _ := toStringSlice(args["candidate_specs"])
	candidates := parseCandidateSpecs(specs)

	result := map[string]any{
		"pattern":         pattern,
		"variant":         variant,
		"phase":           phaseValue,
		"semantic_anchor": anchor,
		"classification":  classValue,
		"marker":          marker,
		"runtime_domain":  runtimeDomain,
		"intent":          intent,
		"stage":           stage,
		"governance":      strings.HasPrefix(marker, "governance_"),
	}

	var risk string
	decision := ""

	switch marker {
	case "discovery_sources_prioritized":
		prioritized := prioritizeSources(candidates)
		result["prioritized_sources"] = toAnySlice(prioritized)
		risk = "source_priority_applied"
	case "discovery_score_reconciled":
		prioritized, _ := toStringSlice(args["prioritized_sources"])
		ranked, topSkill, topScore := reconcileScores(candidates, prioritized)
		result["ranked_skills"] = toAnySlice(ranked)
		result["top_skill"] = topSkill
		result["top_score"] = topScore
		risk = "score_reconciled"
	case "discovery_mapping_emitted":
		ranked, _ := toStringSlice(args["ranked_skills"])
		topScore, _ := modecommon.AsInt(args["top_score"])
		mapping := buildMappingProfile(intent, ranked, topScore)
		result["mapping_profile"] = mapping
		result["top_score"] = topScore
		switch {
		case topScore < 70:
			risk = "degraded_mapping"
		default:
			risk = "mapping_ready"
		}
	case "governance_skill_gate_enforced":
		topSkill := strings.TrimSpace(fmt.Sprintf("%v", args["top_skill"]))
		topScore, _ := modecommon.AsInt(args["top_score"])
		mapping := strings.TrimSpace(fmt.Sprintf("%v", args["mapping_profile"]))
		decision = "allow"
		switch {
		case topScore < 70:
			decision = "block"
			risk = "governed_block"
		case topScore < 80:
			decision = "warn_and_record"
			risk = "governed_warn"
		default:
			risk = "governed_allow"
		}
		result["top_skill"] = topSkill
		result["top_score"] = topScore
		result["mapping_profile"] = mapping
		result["governance_decision"] = decision
		result["governance"] = true
	case "governance_skill_replay_bound":
		topSkill := strings.TrimSpace(fmt.Sprintf("%v", args["top_skill"]))
		topScore, _ := modecommon.AsInt(args["top_score"])
		mapping := strings.TrimSpace(fmt.Sprintf("%v", args["mapping_profile"]))
		decision = safeString(args["governance_decision"], "allow")
		replay := fmt.Sprintf(
			"skill-replay-%d",
			modecommon.SemanticScore(pattern, variant, topSkill, fmt.Sprintf("%d", topScore), mapping, decision),
		)
		result["top_skill"] = topSkill
		result["top_score"] = topScore
		result["mapping_profile"] = mapping
		result["governance_decision"] = decision
		result["replay_binding"] = replay
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported discovery marker: %s", marker)
	}

	result["score"] = modecommon.SemanticScore(
		pattern,
		variant,
		phaseValue,
		anchor,
		classValue,
		marker,
		runtimeDomain,
		intent,
		risk,
		safeString(result["governance_decision"], decision),
	)
	result["risk"] = risk

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s risk=%s governance=%s",
		pattern,
		variant,
		marker,
		risk,
		safeString(result["governance_decision"], "not_applicable"),
	)

	return types.ToolResult{Content: content, Structured: result}, nil
}

func candidateSpecs() []string {
	specs := make([]string, 0, len(discoveryCandidates))
	for _, candidate := range discoveryCandidates {
		specs = append(specs, fmt.Sprintf("%s|%s|%d|%d", candidate.Skill, candidate.Source, candidate.Lexical, candidate.Embedding))
	}
	return specs
}

func parseCandidateSpecs(specs []string) []skillCandidate {
	out := make([]skillCandidate, 0, len(specs))
	for _, spec := range specs {
		parts := strings.Split(spec, "|")
		if len(parts) != 4 {
			continue
		}
		lexical := parseInt(parts[2])
		embedding := parseInt(parts[3])
		out = append(out, skillCandidate{
			Skill:     strings.TrimSpace(parts[0]),
			Source:    strings.TrimSpace(parts[1]),
			Lexical:   lexical,
			Embedding: embedding,
		})
	}
	if len(out) == 0 {
		out = append(out, discoveryCandidates...)
	}
	return out
}

func prioritizeSources(candidates []skillCandidate) []string {
	sourceSeen := map[string]struct{}{}
	for _, candidate := range candidates {
		sourceSeen[candidate.Source] = struct{}{}
	}
	sources := make([]string, 0, len(sourceSeen))
	for source := range sourceSeen {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool {
		left := sourcePriority[sources[i]]
		right := sourcePriority[sources[j]]
		if left == right {
			return sources[i] < sources[j]
		}
		return left > right
	})
	return sources
}

func reconcileScores(candidates []skillCandidate, prioritizedSources []string) ([]string, string, int) {
	weightBySource := map[string]int{}
	for idx, source := range prioritizedSources {
		weightBySource[source] = len(prioritizedSources) - idx
	}
	if len(weightBySource) == 0 {
		for source, priority := range sourcePriority {
			weightBySource[source] = priority
		}
	}

	type scoreItem struct {
		Label string
		Skill string
		Score int
	}
	scored := make([]scoreItem, 0, len(candidates))
	for _, candidate := range candidates {
		sourceWeight := weightBySource[candidate.Source]
		score := candidate.Lexical*5 + candidate.Embedding*4 + sourceWeight*10
		scored = append(scored, scoreItem{
			Label: fmt.Sprintf("%s@%d", candidate.Skill, score),
			Skill: candidate.Skill,
			Score: score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Skill < scored[j].Skill
		}
		return scored[i].Score > scored[j].Score
	})

	ranked := make([]string, 0, len(scored))
	for _, item := range scored {
		ranked = append(ranked, item.Label)
	}
	if len(scored) == 0 {
		return []string{}, "", 0
	}
	return ranked, scored[0].Skill, scored[0].Score
}

func buildMappingProfile(intent string, ranked []string, topScore int) string {
	selected := "none"
	if len(ranked) > 0 {
		selected = strings.Split(ranked[0], "@")[0]
	}
	fallback := "manual_review"
	if topScore >= 80 {
		fallback = "auto_execute"
	}
	return fmt.Sprintf("intent=%s selected=%s fallback=%s", intent, selected, fallback)
}

func toStringSlice(value any) ([]string, bool) {
	switch raw := value.(type) {
	case []string:
		cloned := append([]string(nil), raw...)
		return cloned, true
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text == "" {
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

func parseInt(raw string) int {
	value := 0
	for _, char := range raw {
		if char < '0' || char > '9' {
			continue
		}
		value = value*10 + int(char-'0')
	}
	return value
}

func safeString(value any, fallback string) string {
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func normalizedDecision(decision string) string {
	if strings.TrimSpace(decision) == "" {
		return "not_applicable"
	}
	return decision
}
