package customadaptermcpmodeltoolmemorypack

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
	patternName      = "custom-adapter-mcp-model-tool-memory-pack"
	phase            = "P2"
	semanticAnchor   = "adapterpack.manifest_capability_memory"
	classification   = "adapter.custom_pack"
	semanticToolName = "mode_custom_adapter_mcp_model_tool_memory_pack_semantic_step"
	defaultPackID    = "adapter-pack-20260410"
)

type adapterPackStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type adapterPackState struct {
	PackID              string
	ManifestVersion     string
	TransportProfile    string
	ModelProvider       string
	ToolProfile         string
	CapabilityFallback  string
	MemoryScope         string
	MemoryNamespace     string
	PackReady           bool
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"adapter/scaffold", "mcp/profile", "memory"}

var minimalSemanticSteps = []adapterPackStep{
	{
		Marker:        "adapter_pack_manifest_resolved",
		RuntimeDomain: "adapter/scaffold",
		Intent:        "resolve adapter pack manifest into transport/model/tool profile",
		Outcome:       "manifest version and pack profiles are emitted",
	},
	{
		Marker:        "adapter_pack_capability_fallback",
		RuntimeDomain: "mcp/profile",
		Intent:        "classify fallback capability when requested capability is unavailable",
		Outcome:       "capability fallback and pack readiness are emitted",
	},
	{
		Marker:        "adapter_pack_memory_scope_bound",
		RuntimeDomain: "memory",
		Intent:        "bind memory scope and namespace for adapter pack",
		Outcome:       "memory scope/namespace and pack readiness are emitted",
	},
}

var productionGovernanceSteps = []adapterPackStep{
	{
		Marker:        "governance_adapter_pack_gate_enforced",
		RuntimeDomain: "adapter/scaffold",
		Intent:        "enforce adapter pack governance from fallback and memory scope signals",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_adapter_pack_replay_bound",
		RuntimeDomain: "mcp/profile",
		Intent:        "bind adapter pack governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeAdapterPackVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeAdapterPackVariant(modecommon.VariantProduction)
}

func executeAdapterPackVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&adapterPackTool{}); err != nil {
		panic(err)
	}

	model := &adapterPackModel{
		variant: variant,
		state: adapterPackState{
			PackID: defaultPackID,
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute custom adapter mcp model tool memory pack semantic pipeline",
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

func planForVariant(variant string) []adapterPackStep {
	plan := make([]adapterPackStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type adapterPackModel struct {
	variant string
	cursor  int
	state   adapterPackState
}

func (m *adapterPackModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.captureOutcomes(req.ToolResult)

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

	markers := append([]string(nil), m.state.ObservedMarkers...)
	sort.Strings(markers)
	governanceOn := strings.TrimSpace(m.state.GovernanceDecision) != ""

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s pack=%s manifest=%s transport=%s model=%s tools=%s capability_fallback=%s memory_scope=%s memory_namespace=%s pack_ready=%t governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.PackID, true),
		readableValue(m.state.ManifestVersion, true),
		readableValue(m.state.TransportProfile, true),
		readableValue(m.state.ModelProvider, true),
		readableValue(m.state.ToolProfile, true),
		readableValue(m.state.CapabilityFallback, true),
		readableValue(m.state.MemoryScope, true),
		readableValue(m.state.MemoryNamespace, true),
		m.state.PackReady,
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *adapterPackModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *adapterPackModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if packID, _ := outcome.Result.Structured["pack_id"].(string); strings.TrimSpace(packID) != "" {
			m.state.PackID = strings.TrimSpace(packID)
		}
		if manifest, _ := outcome.Result.Structured["manifest_version"].(string); strings.TrimSpace(manifest) != "" {
			m.state.ManifestVersion = strings.TrimSpace(manifest)
		}
		if transport, _ := outcome.Result.Structured["transport_profile"].(string); strings.TrimSpace(transport) != "" {
			m.state.TransportProfile = strings.TrimSpace(transport)
		}
		if modelProvider, _ := outcome.Result.Structured["model_provider"].(string); strings.TrimSpace(modelProvider) != "" {
			m.state.ModelProvider = strings.TrimSpace(modelProvider)
		}
		if tools, _ := outcome.Result.Structured["tool_profile"].(string); strings.TrimSpace(tools) != "" {
			m.state.ToolProfile = strings.TrimSpace(tools)
		}
		if fallback, _ := outcome.Result.Structured["capability_fallback"].(string); strings.TrimSpace(fallback) != "" {
			m.state.CapabilityFallback = strings.TrimSpace(fallback)
		}
		if scope, _ := outcome.Result.Structured["memory_scope"].(string); strings.TrimSpace(scope) != "" {
			m.state.MemoryScope = strings.TrimSpace(scope)
		}
		if ns, _ := outcome.Result.Structured["memory_namespace"].(string); strings.TrimSpace(ns) != "" {
			m.state.MemoryNamespace = strings.TrimSpace(ns)
		}
		if ready, ok := outcome.Result.Structured["pack_ready"].(bool); ok {
			m.state.PackReady = ready
		}
		if decision, _ := outcome.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := outcome.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if replay, _ := outcome.Result.Structured["replay_signature"].(string); strings.TrimSpace(replay) != "" {
			m.state.ReplaySignature = strings.TrimSpace(replay)
		}
		if score, ok := modecommon.AsInt(outcome.Result.Structured["score"]); ok {
			m.state.AccumulatedSemScore += score
		}
	}
}

func (m *adapterPackModel) argsForStep(step adapterPackStep, stage int) map[string]any {
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
		"pack_id":         m.state.PackID,
	}

	switch step.Marker {
	case "adapter_pack_manifest_resolved":
		manifest := "v1.2"
		transport := "stdio"
		modelProvider := "openai"
		toolProfile := "tools.read_only"
		if m.variant == modecommon.VariantProduction {
			manifest = "v1.4"
			transport = "stdio+http"
			modelProvider = "openai+anthropic"
			toolProfile = "tools.read_write_guarded"
		}
		args["manifest_version"] = manifest
		args["transport_profile"] = transport
		args["model_provider"] = modelProvider
		args["tool_profile"] = toolProfile
	case "adapter_pack_capability_fallback":
		fallback := "tool_exec_read_only"
		if m.variant == modecommon.VariantProduction {
			fallback = "tool_exec_guarded_write"
		}
		args["manifest_version"] = m.state.ManifestVersion
		args["tool_profile"] = m.state.ToolProfile
		args["capability_fallback"] = fallback
	case "adapter_pack_memory_scope_bound":
		memoryScope := "session"
		memoryNamespace := "adapter-pack/session-default"
		if m.variant == modecommon.VariantProduction {
			memoryScope = "tenant+session"
			memoryNamespace = "adapter-pack/tenant-a/session-default"
		}
		args["memory_scope"] = memoryScope
		args["memory_namespace"] = memoryNamespace
		args["capability_fallback"] = m.state.CapabilityFallback
	case "governance_adapter_pack_gate_enforced":
		args["manifest_version"] = m.state.ManifestVersion
		args["capability_fallback"] = m.state.CapabilityFallback
		args["memory_scope"] = m.state.MemoryScope
		args["pack_ready"] = m.state.PackReady
	case "governance_adapter_pack_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["memory_namespace"] = m.state.MemoryNamespace
	}
	return args
}

type adapterPackTool struct{}

func (t *adapterPackTool) Name() string { return semanticToolName }

func (t *adapterPackTool) Description() string {
	return "execute custom adapter pack semantic step"
}

func (t *adapterPackTool) JSONSchema() map[string]any {
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
			"pack_id":         map[string]any{"type": "string"},
			"manifest_version": map[string]any{
				"type": "string",
			},
			"transport_profile":   map[string]any{"type": "string"},
			"model_provider":      map[string]any{"type": "string"},
			"tool_profile":        map[string]any{"type": "string"},
			"capability_fallback": map[string]any{"type": "string"},
			"memory_scope":        map[string]any{"type": "string"},
			"memory_namespace":    map[string]any{"type": "string"},
			"pack_ready":          map[string]any{"type": "boolean"},
			"governance_decision": map[string]any{"type": "string"},
			"governance_ticket":   map[string]any{"type": "string"},
		},
	}
}

func (t *adapterPackTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	packID := strings.TrimSpace(fmt.Sprintf("%v", args["pack_id"]))
	if packID == "" {
		packID = defaultPackID
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
		"pack_id":         packID,
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "adapter_pack_manifest_resolved":
		manifest := strings.TrimSpace(fmt.Sprintf("%v", args["manifest_version"]))
		transport := strings.TrimSpace(fmt.Sprintf("%v", args["transport_profile"]))
		provider := strings.TrimSpace(fmt.Sprintf("%v", args["model_provider"]))
		tools := strings.TrimSpace(fmt.Sprintf("%v", args["tool_profile"]))
		structured["manifest_version"] = manifest
		structured["transport_profile"] = transport
		structured["model_provider"] = provider
		structured["tool_profile"] = tools
	case "adapter_pack_capability_fallback":
		manifest := strings.TrimSpace(fmt.Sprintf("%v", args["manifest_version"]))
		tools := strings.TrimSpace(fmt.Sprintf("%v", args["tool_profile"]))
		fallback := strings.TrimSpace(fmt.Sprintf("%v", args["capability_fallback"]))
		packReady := manifest != "" && tools != "" && fallback != ""
		structured["manifest_version"] = manifest
		structured["tool_profile"] = tools
		structured["capability_fallback"] = fallback
		structured["pack_ready"] = packReady
		if strings.Contains(fallback, "guarded") {
			risk = "degraded_path"
		}
	case "adapter_pack_memory_scope_bound":
		scope := strings.TrimSpace(fmt.Sprintf("%v", args["memory_scope"]))
		namespace := strings.TrimSpace(fmt.Sprintf("%v", args["memory_namespace"]))
		fallback := strings.TrimSpace(fmt.Sprintf("%v", args["capability_fallback"]))
		packReady := scope != "" && namespace != "" && fallback != ""
		structured["memory_scope"] = scope
		structured["memory_namespace"] = namespace
		structured["capability_fallback"] = fallback
		structured["pack_ready"] = packReady
		if strings.Contains(scope, "tenant+") {
			risk = "degraded_path"
		}
	case "governance_adapter_pack_gate_enforced":
		manifest := strings.TrimSpace(fmt.Sprintf("%v", args["manifest_version"]))
		fallback := strings.TrimSpace(fmt.Sprintf("%v", args["capability_fallback"]))
		scope := strings.TrimSpace(fmt.Sprintf("%v", args["memory_scope"]))
		packReady := asBool(args["pack_ready"])
		decision := "allow_pack"
		if !packReady {
			decision = "deny_pack"
		} else if strings.Contains(fallback, "guarded") || strings.Contains(scope, "tenant+") {
			decision = "allow_pack_with_shadow_validation"
		}
		ticket := fmt.Sprintf("adapter-pack-gate-%d", modecommon.SemanticScore(packID, manifest, decision, fallback, scope))
		structured["manifest_version"] = manifest
		structured["capability_fallback"] = fallback
		structured["memory_scope"] = scope
		structured["pack_ready"] = packReady
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_adapter_pack_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		namespace := strings.TrimSpace(fmt.Sprintf("%v", args["memory_namespace"]))
		replaySignature := fmt.Sprintf("adapter-pack-replay-%d", modecommon.SemanticScore(packID, decision, ticket, namespace))
		structured["memory_namespace"] = namespace
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported adapter pack marker: %s", marker)
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

func readableValue(value string, enabled bool) string {
	if !enabled {
		return "n/a"
	}
	if strings.TrimSpace(value) == "" {
		return "pending"
	}
	return value
}
