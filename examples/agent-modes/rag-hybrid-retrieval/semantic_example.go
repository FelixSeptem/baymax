package raghybridretrieval

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
	patternName      = "rag-hybrid-retrieval"
	phase            = "P0"
	semanticAnchor   = "retrieval.candidate_rerank_fallback"
	classification   = "rag.hybrid_retrieval"
	semanticToolName = "mode_rag_hybrid_retrieval_semantic_step"
	defaultQuery     = "runtime rollback config hot reload failfast"
)

type retrievalStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type retrievalDoc struct {
	ID        string
	Text      string
	Freshness int
}

type retrievalState struct {
	Query           string
	CandidateIDs    []string
	RankedIDs       []string
	SeenMarkers     []string
	TopScore        int
	FallbackRoute   string
	BudgetTrimmed   bool
	ReplaySignature string
	TotalScore      int
}

var runtimeDomains = []string{"memory", "context/assembler"}

var retrievalCorpus = []retrievalDoc{
	{ID: "doc-reload", Text: "runtime config reload failfast rollback atomic safety", Freshness: 9},
	{ID: "doc-budget", Text: "budget admission policy trace runtime diagnostics", Freshness: 6},
	{ID: "doc-memory", Text: "memory scope retrieval fallback summary replay fixture", Freshness: 8},
	{ID: "doc-security", Text: "security sandbox deny policy event delivery", Freshness: 5},
}

var minimalSemanticSteps = []retrievalStep{
	{
		Marker:        "retrieval_candidates_built",
		RuntimeDomain: "memory",
		Intent:        "build retrieval candidates from in-memory corpus for rollback query",
		Outcome:       "candidate set with lexical scores is produced",
	},
	{
		Marker:        "retrieval_rerank_applied",
		RuntimeDomain: "context/assembler",
		Intent:        "rerank candidates using freshness and lexical confidence",
		Outcome:       "top ranked candidate list is stabilized",
	},
	{
		Marker:        "retrieval_fallback_classified",
		RuntimeDomain: "memory",
		Intent:        "classify whether fallback summary is needed when confidence is low",
		Outcome:       "fallback route classification is emitted",
	},
}

var productionGovernanceSteps = []retrievalStep{
	{
		Marker:        "governance_retrieval_budget_gate",
		RuntimeDomain: "memory",
		Intent:        "enforce retrieval candidate budget before final response",
		Outcome:       "candidate budget gating decision is persisted",
	},
	{
		Marker:        "governance_retrieval_replay_bound",
		RuntimeDomain: "context/assembler",
		Intent:        "emit replay signature for retrieval decision determinism",
		Outcome:       "replay bound signature is generated",
	},
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
	if _, err := reg.Register(&ragRetrievalSemanticTool{}); err != nil {
		panic(err)
	}

	model := &retrievalWorkflowModel{
		variant: variant,
		state: retrievalState{
			Query: defaultQuery,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute rag hybrid retrieval semantic workflow",
	}, nil)
	if err != nil {
		panic(err)
	}

	expectedMarkers := expectedMarkersForVariant(variant)
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

func expectedMarkersForVariant(variant string) []string {
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

func planForVariant(variant string) []retrievalStep {
	plan := make([]retrievalStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type retrievalWorkflowModel struct {
	variant   string
	nextStage int
	state     retrievalState
}

func (m *retrievalWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.absorb(req.ToolResult)

	plan := planForVariant(m.variant)
	if m.nextStage < len(plan) {
		step := plan[m.nextStage]
		toolCall := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.nextStage+1),
			Name:   "local." + semanticToolName,
			Args:   m.argsForStep(step, m.nextStage+1),
		}
		m.nextStage++
		return types.ModelResponse{ToolCalls: []types.ToolCall{toolCall}}, nil
	}

	sortedMarkers := append([]string(nil), m.state.SeenMarkers...)
	sort.Strings(sortedMarkers)
	sortedRanked := append([]string(nil), m.state.RankedIDs...)
	sort.Strings(sortedRanked)
	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s markers=%s ranked=%s top_score=%d fallback=%s budget_trimmed=%t replay_signature=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		strings.Join(sortedMarkers, ","),
		strings.Join(sortedRanked, ","),
		m.state.TopScore,
		m.state.FallbackRoute,
		m.state.BudgetTrimmed,
		m.state.ReplaySignature,
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *retrievalWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *retrievalWorkflowModel) absorb(results []types.ToolCallOutcome) {
	for _, item := range results {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}

		if ids, ok := toStringSlice(item.Result.Structured["candidate_ids"]); ok {
			m.state.CandidateIDs = ids
		}
		if ids, ok := toStringSlice(item.Result.Structured["ranked_ids"]); ok {
			m.state.RankedIDs = ids
		}
		if topScore, ok := modecommon.AsInt(item.Result.Structured["top_score"]); ok {
			m.state.TopScore = topScore
		}
		if fallbackRoute, ok := item.Result.Structured["fallback_route"].(string); ok && strings.TrimSpace(fallbackRoute) != "" {
			m.state.FallbackRoute = fallbackRoute
		}
		if budgetTrimmed, ok := item.Result.Structured["budget_trimmed"].(bool); ok {
			m.state.BudgetTrimmed = budgetTrimmed
		}
		if replaySignature, ok := item.Result.Structured["replay_signature"].(string); ok && strings.TrimSpace(replaySignature) != "" {
			m.state.ReplaySignature = replaySignature
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *retrievalWorkflowModel) argsForStep(step retrievalStep, stage int) map[string]any {
	args := map[string]any{
		"pattern":         patternName,
		"variant":         m.variant,
		"phase":           phase,
		"semantic_anchor": semanticAnchor,
		"classification":  classification,
		"marker":          step.Marker,
		"runtime_domain":  step.RuntimeDomain,
		"intent":          step.Intent,
		"outcome":         step.Outcome,
		"stage":           stage,
		"query":           m.state.Query,
	}

	switch step.Marker {
	case "retrieval_candidates_built":
		args["candidate_pool"] = toAnySlice(corpusTexts())
	case "retrieval_rerank_applied":
		args["candidate_ids"] = toAnySlice(m.state.CandidateIDs)
		args["freshness"] = corpusFreshness()
	case "retrieval_fallback_classified":
		args["top_score"] = m.state.TopScore
		args["candidate_count"] = len(m.state.RankedIDs)
	case "governance_retrieval_budget_gate":
		args["ranked_ids"] = toAnySlice(m.state.RankedIDs)
		args["budget_limit"] = 2
	case "governance_retrieval_replay_bound":
		args["ranked_ids"] = toAnySlice(m.state.RankedIDs)
		args["fallback_route"] = m.state.FallbackRoute
		args["existing_signature"] = m.state.ReplaySignature
	}
	return args
}

type ragRetrievalSemanticTool struct{}

func (t *ragRetrievalSemanticTool) Name() string { return semanticToolName }

func (t *ragRetrievalSemanticTool) Description() string {
	return "execute rag hybrid retrieval semantic step"
}

func (t *ragRetrievalSemanticTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "stage"},
		"properties": map[string]any{
			"pattern":         map[string]any{"type": "string"},
			"variant":         map[string]any{"type": "string"},
			"phase":           map[string]any{"type": "string"},
			"semantic_anchor": map[string]any{"type": "string"},
			"classification":  map[string]any{"type": "string"},
			"marker":          map[string]any{"type": "string"},
			"runtime_domain":  map[string]any{"type": "string"},
			"intent":          map[string]any{"type": "string"},
			"outcome":         map[string]any{"type": "string"},
			"query":           map[string]any{"type": "string"},
			"candidate_pool":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"candidate_ids":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"ranked_ids":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"freshness":       map[string]any{"type": "object"},
			"budget_limit":    map[string]any{"type": "integer"},
			"top_score":       map[string]any{"type": "integer"},
			"candidate_count": map[string]any{"type": "integer"},
			"fallback_route":  map[string]any{"type": "string"},
			"existing_signature": map[string]any{
				"type": "string",
			},
			"stage": map[string]any{"type": "integer"},
		},
	}
}

func (t *ragRetrievalSemanticTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx

	pattern := strings.TrimSpace(fmt.Sprintf("%v", args["pattern"]))
	variant := strings.TrimSpace(fmt.Sprintf("%v", args["variant"]))
	phaseValue := strings.TrimSpace(fmt.Sprintf("%v", args["phase"]))
	anchor := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_anchor"]))
	classValue := strings.TrimSpace(fmt.Sprintf("%v", args["classification"]))
	marker := strings.TrimSpace(fmt.Sprintf("%v", args["marker"]))
	runtimeDomain := strings.TrimSpace(fmt.Sprintf("%v", args["runtime_domain"]))
	intent := strings.TrimSpace(fmt.Sprintf("%v", args["intent"]))
	outcome := strings.TrimSpace(fmt.Sprintf("%v", args["outcome"]))
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	structured := map[string]any{
		"pattern":         pattern,
		"variant":         variant,
		"phase":           phaseValue,
		"semantic_anchor": anchor,
		"classification":  classValue,
		"marker":          marker,
		"runtime_domain":  runtimeDomain,
		"intent":          intent,
		"outcome":         outcome,
		"stage":           stage,
		"governance":      strings.HasPrefix(marker, "governance_"),
	}

	query := strings.TrimSpace(fmt.Sprintf("%v", args["query"]))
	if query == "" {
		query = defaultQuery
	}

	risk := "nominal"
	switch marker {
	case "retrieval_candidates_built":
		pool, _ := toStringSlice(args["candidate_pool"])
		candidateIDs, topScore := buildCandidates(query, pool)
		structured["candidate_ids"] = candidateIDs
		structured["top_score"] = topScore
		risk = "candidate_selection"
	case "retrieval_rerank_applied":
		candidateIDs, _ := toStringSlice(args["candidate_ids"])
		rankedIDs, topScore := rerankCandidates(candidateIDs)
		structured["ranked_ids"] = rankedIDs
		structured["top_score"] = topScore
		risk = "rerank_adjustment"
	case "retrieval_fallback_classified":
		topScore, _ := modecommon.AsInt(args["top_score"])
		fallbackRoute := "direct_answer"
		if topScore < 2 {
			fallbackRoute = "memory_summary"
			risk = "degraded_path"
		}
		structured["fallback_route"] = fallbackRoute
		structured["top_score"] = topScore
	case "governance_retrieval_budget_gate":
		rankedIDs, _ := toStringSlice(args["ranked_ids"])
		budgetLimit, ok := modecommon.AsInt(args["budget_limit"])
		if !ok || budgetLimit <= 0 {
			budgetLimit = 2
		}
		budgetTrimmed := len(rankedIDs) > budgetLimit
		if budgetTrimmed {
			rankedIDs = append([]string(nil), rankedIDs[:budgetLimit]...)
		}
		structured["ranked_ids"] = rankedIDs
		structured["budget_limit"] = budgetLimit
		structured["budget_trimmed"] = budgetTrimmed
		structured["governance"] = true
		risk = "governed"
	case "governance_retrieval_replay_bound":
		rankedIDs, _ := toStringSlice(args["ranked_ids"])
		fallbackRoute := strings.TrimSpace(fmt.Sprintf("%v", args["fallback_route"]))
		replaySignature := fmt.Sprintf(
			"retrieval-replay-%d",
			modecommon.SemanticScore(pattern, variant, strings.Join(rankedIDs, ","), fallbackRoute),
		)
		structured["ranked_ids"] = rankedIDs
		structured["fallback_route"] = fallbackRoute
		structured["replay_signature"] = replaySignature
		structured["governance"] = true
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported rag retrieval marker: %s", marker)
	}

	structured["risk"] = risk
	structured["score"] = modecommon.SemanticScore(pattern, variant, phaseValue, anchor, classValue, marker, runtimeDomain, risk, query)

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s domain=%s stage=%d risk=%s governance=%t",
		pattern,
		variant,
		marker,
		runtimeDomain,
		stage,
		risk,
		structured["governance"],
	)

	return types.ToolResult{Content: content, Structured: structured}, nil
}

func corpusTexts() []string {
	out := make([]string, 0, len(retrievalCorpus))
	for _, doc := range retrievalCorpus {
		out = append(out, fmt.Sprintf("%s|%s", doc.ID, doc.Text))
	}
	return out
}

func corpusFreshness() map[string]any {
	out := map[string]any{}
	for _, doc := range retrievalCorpus {
		out[doc.ID] = doc.Freshness
	}
	return out
}

func buildCandidates(query string, pool []string) ([]string, int) {
	terms := tokenize(query)
	candidateScores := map[string]int{}
	for _, item := range pool {
		parts := strings.SplitN(item, "|", 2)
		if len(parts) != 2 {
			continue
		}
		docID := parts[0]
		text := strings.ToLower(parts[1])
		score := 0
		for _, term := range terms {
			if term == "" {
				continue
			}
			if strings.Contains(text, term) {
				score++
			}
		}
		if score > 0 {
			candidateScores[docID] = score
		}
	}
	if len(candidateScores) == 0 {
		candidateScores["doc-memory"] = 1
	}
	ids := make([]string, 0, len(candidateScores))
	topScore := 0
	for id, score := range candidateScores {
		ids = append(ids, id)
		if score > topScore {
			topScore = score
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		if candidateScores[ids[i]] == candidateScores[ids[j]] {
			return ids[i] < ids[j]
		}
		return candidateScores[ids[i]] > candidateScores[ids[j]]
	})
	return ids, topScore
}

func rerankCandidates(candidateIDs []string) ([]string, int) {
	if len(candidateIDs) == 0 {
		return []string{"doc-memory"}, 1
	}
	freshness := map[string]int{}
	for _, doc := range retrievalCorpus {
		freshness[doc.ID] = doc.Freshness
	}
	sorted := append([]string(nil), candidateIDs...)
	sort.Slice(sorted, func(i, j int) bool {
		left := freshness[sorted[i]]
		right := freshness[sorted[j]]
		if left == right {
			return sorted[i] < sorted[j]
		}
		return left > right
	})
	topScore := freshness[sorted[0]]
	if topScore <= 0 {
		topScore = 1
	}
	return sorted, topScore
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

func tokenize(query string) []string {
	normalized := strings.ToLower(query)
	replacer := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "|", " ", "\n", " ", "\t", " ")
	normalized = replacer.Replace(normalized)
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return []string{"runtime", "rollback"}
	}
	return parts
}
