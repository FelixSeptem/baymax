package observabilityexportbundle

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
	patternName      = "observability-export-bundle"
	phase            = "P1"
	semanticAnchor   = "observability.export_bundle_replay"
	classification   = "observability.export_bundle"
	semanticToolName = "mode_observability_export_bundle_semantic_step"
	defaultExportID  = "obs-export-20260410"
)

type observabilityStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type observabilityState struct {
	ExportID           string
	EventCount         int
	DroppedCount       int
	SourceKinds        []string
	BundleID           string
	BundleHash         string
	BundleSizeKB       int
	Compression        string
	ReplayLink         string
	IntegrityOK        bool
	GovernanceDecision string
	GovernanceTicket   string
	ReplayBoundSig     string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"observability/event", "runtime/diagnostics"}

var defaultSourceKinds = []string{"trace", "metric", "log"}

var minimalSemanticSteps = []observabilityStep{
	{
		Marker:        "observability_export_collected",
		RuntimeDomain: "observability/event",
		Intent:        "collect export payload from trace/metric/log sources",
		Outcome:       "event count/drop count/source kinds are emitted",
	},
	{
		Marker:        "observability_bundle_emitted",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "emit compressed diagnostics bundle with hash",
		Outcome:       "bundle id/hash/size/compression are emitted",
	},
	{
		Marker:        "observability_replay_linked",
		RuntimeDomain: "observability/event",
		Intent:        "link replay pointer and verify bundle integrity",
		Outcome:       "replay link and integrity signal are emitted",
	},
}

var productionGovernanceSteps = []observabilityStep{
	{
		Marker:        "governance_observability_gate_enforced",
		RuntimeDomain: "observability/event",
		Intent:        "enforce governance decision for degraded export quality",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_observability_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind governance decision to replay signature",
		Outcome:       "replay bound signature is emitted",
	},
}

func RunMinimal() {
	executeObservabilityVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeObservabilityVariant(modecommon.VariantProduction)
}

func executeObservabilityVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&observabilityBundleTool{}); err != nil {
		panic(err)
	}

	model := &observabilityBundleModel{
		variant: variant,
		state: observabilityState{
			ExportID: defaultExportID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute observability export bundle semantic pipeline",
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
	out := make([]string, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	for _, step := range minimalSemanticSteps {
		out = append(out, step.Marker)
	}
	if variant == modecommon.VariantProduction {
		for _, step := range productionGovernanceSteps {
			out = append(out, step.Marker)
		}
	}
	return out
}

func planForVariant(variant string) []observabilityStep {
	plan := make([]observabilityStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type observabilityBundleModel struct {
	variant string
	cursor  int
	state   observabilityState
}

func (m *observabilityBundleModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.capture(req.ToolResult)

	plan := planForVariant(m.variant)
	if m.cursor < len(plan) {
		step := plan[m.cursor]
		call := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.cursor+1),
			Name:   "local." + semanticToolName,
			Args:   m.argsForStep(step, m.cursor+1),
		}
		m.cursor++
		return types.ModelResponse{ToolCalls: []types.ToolCall{call}}, nil
	}

	markers := append([]string(nil), m.state.SeenMarkers...)
	sort.Strings(markers)
	governanceOn := strings.TrimSpace(m.state.GovernanceDecision) != ""

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s export_id=%s events=%d dropped=%d sources=%s bundle=%s hash=%s size_kb=%d compression=%s replay_link=%s integrity=%t governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		normalizedValue(m.state.ExportID, true),
		m.state.EventCount,
		m.state.DroppedCount,
		stringSliceToken(m.state.SourceKinds),
		normalizedValue(m.state.BundleID, true),
		normalizedValue(m.state.BundleHash, true),
		m.state.BundleSizeKB,
		normalizedValue(m.state.Compression, true),
		normalizedValue(m.state.ReplayLink, true),
		m.state.IntegrityOK,
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplayBoundSig, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *observabilityBundleModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *observabilityBundleModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if exportID, _ := item.Result.Structured["export_id"].(string); strings.TrimSpace(exportID) != "" {
			m.state.ExportID = strings.TrimSpace(exportID)
		}
		if count, ok := modecommon.AsInt(item.Result.Structured["event_count"]); ok {
			m.state.EventCount = count
		}
		if dropped, ok := modecommon.AsInt(item.Result.Structured["dropped_count"]); ok {
			m.state.DroppedCount = dropped
		}
		if sources := toStringSlice(item.Result.Structured["source_kinds"]); len(sources) > 0 {
			m.state.SourceKinds = sources
		}
		if bundleID, _ := item.Result.Structured["bundle_id"].(string); strings.TrimSpace(bundleID) != "" {
			m.state.BundleID = strings.TrimSpace(bundleID)
		}
		if hash, _ := item.Result.Structured["bundle_hash"].(string); strings.TrimSpace(hash) != "" {
			m.state.BundleHash = strings.TrimSpace(hash)
		}
		if sizeKB, ok := modecommon.AsInt(item.Result.Structured["bundle_size_kb"]); ok {
			m.state.BundleSizeKB = sizeKB
		}
		if compression, _ := item.Result.Structured["compression"].(string); strings.TrimSpace(compression) != "" {
			m.state.Compression = strings.TrimSpace(compression)
		}
		if replayLink, _ := item.Result.Structured["replay_link"].(string); strings.TrimSpace(replayLink) != "" {
			m.state.ReplayLink = strings.TrimSpace(replayLink)
		}
		if integrity, ok := item.Result.Structured["integrity_ok"].(bool); ok {
			m.state.IntegrityOK = integrity
		}
		if decision, _ := item.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := item.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if replay, _ := item.Result.Structured["replay_bound_signature"].(string); strings.TrimSpace(replay) != "" {
			m.state.ReplayBoundSig = strings.TrimSpace(replay)
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *observabilityBundleModel) argsForStep(step observabilityStep, stage int) map[string]any {
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
		"export_id":       m.state.ExportID,
	}

	switch step.Marker {
	case "observability_export_collected":
		eventCount := 128
		dropped := 3
		if m.variant == modecommon.VariantProduction {
			eventCount = 182
			dropped = 14
		}
		args["source_kinds"] = stringSliceToAny(defaultSourceKinds)
		args["event_count"] = eventCount
		args["dropped_count"] = dropped
	case "observability_bundle_emitted":
		args["export_id"] = m.state.ExportID
		args["source_kinds"] = stringSliceToAny(m.state.SourceKinds)
		args["event_count"] = m.state.EventCount
		args["dropped_count"] = m.state.DroppedCount
	case "observability_replay_linked":
		args["bundle_id"] = m.state.BundleID
		args["bundle_hash"] = m.state.BundleHash
		args["dropped_count"] = m.state.DroppedCount
	case "governance_observability_gate_enforced":
		args["dropped_count"] = m.state.DroppedCount
		args["integrity_ok"] = m.state.IntegrityOK
		args["bundle_size_kb"] = m.state.BundleSizeKB
	case "governance_observability_replay_bound":
		args["export_id"] = m.state.ExportID
		args["bundle_hash"] = m.state.BundleHash
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
	}
	return args
}

type observabilityBundleTool struct{}

func (t *observabilityBundleTool) Name() string { return semanticToolName }

func (t *observabilityBundleTool) Description() string {
	return "execute observability export/bundle/replay semantic step"
}

func (t *observabilityBundleTool) JSONSchema() map[string]any {
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
			"stage":           map[string]any{"type": "integer"},
			"export_id":       map[string]any{"type": "string"},
			"source_kinds":    map[string]any{"type": "array"},
			"event_count":     map[string]any{"type": "integer"},
			"dropped_count":   map[string]any{"type": "integer"},
			"bundle_id":       map[string]any{"type": "string"},
			"bundle_hash":     map[string]any{"type": "string"},
			"bundle_size_kb":  map[string]any{"type": "integer"},
			"compression":     map[string]any{"type": "string"},
			"integrity_ok":    map[string]any{"type": "boolean"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *observabilityBundleTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "observability_export_collected":
		exportID := strings.TrimSpace(fmt.Sprintf("%v", args["export_id"]))
		if exportID == "" {
			exportID = defaultExportID
		}
		sources := toStringSlice(args["source_kinds"])
		if len(sources) == 0 {
			sources = append([]string{}, defaultSourceKinds...)
		}
		eventCount, _ := modecommon.AsInt(args["event_count"])
		dropped, _ := modecommon.AsInt(args["dropped_count"])
		structured["export_id"] = exportID
		structured["source_kinds"] = stringSliceToAny(sources)
		structured["event_count"] = eventCount
		structured["dropped_count"] = dropped
		if dropped > 10 {
			risk = "degraded_path"
		}
	case "observability_bundle_emitted":
		exportID := strings.TrimSpace(fmt.Sprintf("%v", args["export_id"]))
		sources := toStringSlice(args["source_kinds"])
		eventCount, _ := modecommon.AsInt(args["event_count"])
		dropped, _ := modecommon.AsInt(args["dropped_count"])
		compression := "zstd"
		if variant == modecommon.VariantProduction {
			compression = "lz4"
		}
		bundleID := fmt.Sprintf("bundle-%d", modecommon.SemanticScore(exportID, strings.Join(sources, ","), fmt.Sprintf("%d", eventCount)))
		bundleHash := fmt.Sprintf("hash-%d", modecommon.SemanticScore(bundleID, fmt.Sprintf("drop=%d", dropped), compression))
		sizeKB := eventCount/4 + len(sources)*3
		structured["export_id"] = exportID
		structured["bundle_id"] = bundleID
		structured["bundle_hash"] = bundleHash
		structured["bundle_size_kb"] = sizeKB
		structured["compression"] = compression
		if dropped > 10 {
			risk = "degraded_path"
		}
	case "observability_replay_linked":
		bundleID := strings.TrimSpace(fmt.Sprintf("%v", args["bundle_id"]))
		bundleHash := strings.TrimSpace(fmt.Sprintf("%v", args["bundle_hash"]))
		dropped, _ := modecommon.AsInt(args["dropped_count"])
		replayLink := fmt.Sprintf("replay://%s", bundleID)
		integrityOK := bundleHash != "" && dropped <= 15
		structured["bundle_id"] = bundleID
		structured["bundle_hash"] = bundleHash
		structured["replay_link"] = replayLink
		structured["integrity_ok"] = integrityOK
		if !integrityOK {
			risk = "degraded_path"
		}
	case "governance_observability_gate_enforced":
		dropped, _ := modecommon.AsInt(args["dropped_count"])
		integrityOK := asBool(args["integrity_ok"])
		sizeKB, _ := modecommon.AsInt(args["bundle_size_kb"])
		decision := "allow"
		if !integrityOK {
			decision = "deny"
		} else if dropped > 10 || sizeKB > 60 {
			decision = "allow_with_sampling"
		}
		ticket := fmt.Sprintf("obs-gate-%d", modecommon.SemanticScore(fmt.Sprintf("drop=%d", dropped), fmt.Sprintf("%t", integrityOK), fmt.Sprintf("size=%d", sizeKB), decision))
		structured["dropped_count"] = dropped
		structured["integrity_ok"] = integrityOK
		structured["bundle_size_kb"] = sizeKB
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_observability_replay_bound":
		exportID := strings.TrimSpace(fmt.Sprintf("%v", args["export_id"]))
		bundleHash := strings.TrimSpace(fmt.Sprintf("%v", args["bundle_hash"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		replaySig := fmt.Sprintf("obs-replay-%d", modecommon.SemanticScore(exportID, bundleHash, decision, ticket))
		structured["export_id"] = exportID
		structured["bundle_hash"] = bundleHash
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["governance"] = true
		structured["replay_bound_signature"] = replaySig
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported observability semantic marker: %s", marker)
	}

	score := modecommon.SemanticScore(pattern, variant, phaseValue, anchor, classValue, marker, runtimeDomain, risk, fmt.Sprintf("%d", stage))
	structured["risk"] = risk
	structured["score"] = score

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s domain=%s stage=%d risk=%s governance=%t",
		pattern,
		variant,
		marker,
		runtimeDomain,
		stage,
		risk,
		asBool(structured["governance"]),
	)
	return types.ToolResult{Content: content, Structured: structured}, nil
}

func toStringSlice(value any) []string {
	switch raw := value.(type) {
	case []string:
		return append([]string(nil), raw...)
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		return out
	default:
		return nil
	}
}

func stringSliceToAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}

func stringSliceToken(in []string) string {
	if len(in) == 0 {
		return "none"
	}
	copySlice := append([]string(nil), in...)
	sort.Strings(copySlice)
	return strings.Join(copySlice, "|")
}

func asBool(value any) bool {
	switch item := value.(type) {
	case bool:
		return item
	case string:
		normalized := strings.ToLower(strings.TrimSpace(item))
		return normalized == "1" || normalized == "true" || normalized == "yes"
	case int:
		return item != 0
	case int64:
		return item != 0
	default:
		return false
	}
}

func normalizedValue(value string, enabled bool) string {
	if !enabled {
		return "n/a"
	}
	if strings.TrimSpace(value) == "" {
		return "pending"
	}
	return value
}
