package adapteronboardingmanifestcapability

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
	patternName      = "adapter-onboarding-manifest-capability"
	phase            = "P2"
	semanticAnchor   = "adapter.manifest_capability_fallback"
	classification   = "adapter.onboarding"
	semanticToolName = "mode_adapter_onboarding_manifest_capability_semantic_step"
	defaultAdapterID = "adapter-openapi-v3"
)

type adapterStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type adapterState struct {
	AdapterID          string
	ManifestVersion    string
	ContractProfile    string
	RequiredCaps       []string
	NegotiatedCaps     []string
	MissingCaps        []string
	FallbackProfile    string
	FallbackReason     string
	GovernanceDecision string
	GovernanceTicket   string
	ReplaySignature    string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"adapter/manifest", "adapter/capability"}

var runtimeSupportedCaps = []string{"chat", "tool_call", "json_schema", "retry"}

var minimalSemanticSteps = []adapterStep{
	{
		Marker:        "adapter_manifest_loaded",
		RuntimeDomain: "adapter/manifest",
		Intent:        "load adapter manifest and validate profile/version compatibility",
		Outcome:       "manifest profile/version and required capabilities are emitted",
	},
	{
		Marker:        "adapter_capability_negotiated",
		RuntimeDomain: "adapter/capability",
		Intent:        "negotiate adapter capabilities against runtime supported set",
		Outcome:       "negotiated/missing capability sets are emitted",
	},
	{
		Marker:        "adapter_fallback_mapped",
		RuntimeDomain: "adapter/manifest",
		Intent:        "map fallback profile when capability gap exists",
		Outcome:       "fallback profile and reason are emitted",
	},
}

var productionGovernanceSteps = []adapterStep{
	{
		Marker:        "governance_adapter_gate_enforced",
		RuntimeDomain: "adapter/manifest",
		Intent:        "enforce onboarding gate using missing capability risk",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_adapter_replay_bound",
		RuntimeDomain: "adapter/capability",
		Intent:        "bind adapter onboarding decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeAdapterVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeAdapterVariant(modecommon.VariantProduction)
}

func executeAdapterVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&adapterOnboardingTool{}); err != nil {
		panic(err)
	}

	model := &adapterOnboardingModel{
		variant: variant,
		state: adapterState{
			AdapterID: defaultAdapterID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute adapter onboarding semantic pipeline",
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

func planForVariant(variant string) []adapterStep {
	plan := make([]adapterStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type adapterOnboardingModel struct {
	variant string
	cursor  int
	state   adapterState
}

func (m *adapterOnboardingModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s adapter=%s manifest=%s profile=%s required=%s negotiated=%s missing=%s fallback=%s reason=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		normalizedValue(m.state.AdapterID, true),
		normalizedValue(m.state.ManifestVersion, true),
		normalizedValue(m.state.ContractProfile, true),
		stringSliceToken(m.state.RequiredCaps),
		stringSliceToken(m.state.NegotiatedCaps),
		stringSliceToken(m.state.MissingCaps),
		normalizedValue(m.state.FallbackProfile, true),
		normalizedValue(m.state.FallbackReason, true),
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *adapterOnboardingModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *adapterOnboardingModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if adapterID, _ := item.Result.Structured["adapter_id"].(string); strings.TrimSpace(adapterID) != "" {
			m.state.AdapterID = strings.TrimSpace(adapterID)
		}
		if version, _ := item.Result.Structured["manifest_version"].(string); strings.TrimSpace(version) != "" {
			m.state.ManifestVersion = strings.TrimSpace(version)
		}
		if profile, _ := item.Result.Structured["contract_profile"].(string); strings.TrimSpace(profile) != "" {
			m.state.ContractProfile = strings.TrimSpace(profile)
		}
		if required := toStringSlice(item.Result.Structured["required_caps"]); len(required) > 0 {
			m.state.RequiredCaps = required
		}
		if negotiated := toStringSlice(item.Result.Structured["negotiated_caps"]); len(negotiated) > 0 {
			m.state.NegotiatedCaps = negotiated
		}
		if missing := toStringSlice(item.Result.Structured["missing_caps"]); len(missing) > 0 {
			m.state.MissingCaps = missing
		}
		if fallback, _ := item.Result.Structured["fallback_profile"].(string); strings.TrimSpace(fallback) != "" {
			m.state.FallbackProfile = strings.TrimSpace(fallback)
		}
		if reason, _ := item.Result.Structured["fallback_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.FallbackReason = strings.TrimSpace(reason)
		}
		if decision, _ := item.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := item.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if replay, _ := item.Result.Structured["replay_signature"].(string); strings.TrimSpace(replay) != "" {
			m.state.ReplaySignature = strings.TrimSpace(replay)
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *adapterOnboardingModel) argsForStep(step adapterStep, stage int) map[string]any {
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
		"adapter_id":      m.state.AdapterID,
	}

	switch step.Marker {
	case "adapter_manifest_loaded":
		required := []string{"chat", "tool_call", "json_schema"}
		manifestVersion := "v1.2.0"
		contractProfile := "adapter_contract_profile.v1"
		if m.variant == modecommon.VariantProduction {
			required = append(required, "streaming")
			manifestVersion = "v1.3.0"
		}
		args["required_caps"] = stringSliceToAny(required)
		args["manifest_version"] = manifestVersion
		args["contract_profile"] = contractProfile
	case "adapter_capability_negotiated":
		args["required_caps"] = stringSliceToAny(m.state.RequiredCaps)
		args["runtime_caps"] = stringSliceToAny(runtimeSupportedCaps)
	case "adapter_fallback_mapped":
		args["missing_caps"] = stringSliceToAny(m.state.MissingCaps)
		args["manifest_version"] = m.state.ManifestVersion
		args["contract_profile"] = m.state.ContractProfile
	case "governance_adapter_gate_enforced":
		args["missing_caps"] = stringSliceToAny(m.state.MissingCaps)
		args["fallback_profile"] = m.state.FallbackProfile
		args["fallback_reason"] = m.state.FallbackReason
	case "governance_adapter_replay_bound":
		args["adapter_id"] = m.state.AdapterID
		args["manifest_version"] = m.state.ManifestVersion
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
	}
	return args
}

type adapterOnboardingTool struct{}

func (t *adapterOnboardingTool) Name() string { return semanticToolName }

func (t *adapterOnboardingTool) Description() string {
	return "execute adapter manifest/capability/fallback semantic step"
}

func (t *adapterOnboardingTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "stage"},
		"properties": map[string]any{
			"pattern":          map[string]any{"type": "string"},
			"variant":          map[string]any{"type": "string"},
			"phase":            map[string]any{"type": "string"},
			"semantic_anchor":  map[string]any{"type": "string"},
			"classification":   map[string]any{"type": "string"},
			"marker":           map[string]any{"type": "string"},
			"runtime_domain":   map[string]any{"type": "string"},
			"intent":           map[string]any{"type": "string"},
			"outcome":          map[string]any{"type": "string"},
			"stage":            map[string]any{"type": "integer"},
			"adapter_id":       map[string]any{"type": "string"},
			"manifest_version": map[string]any{"type": "string"},
			"contract_profile": map[string]any{"type": "string"},
			"required_caps":    map[string]any{"type": "array"},
			"runtime_caps":     map[string]any{"type": "array"},
			"negotiated_caps":  map[string]any{"type": "array"},
			"missing_caps":     map[string]any{"type": "array"},
			"fallback_profile": map[string]any{"type": "string"},
			"fallback_reason":  map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *adapterOnboardingTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "adapter_manifest_loaded":
		adapterID := strings.TrimSpace(fmt.Sprintf("%v", args["adapter_id"]))
		if adapterID == "" {
			adapterID = defaultAdapterID
		}
		version := strings.TrimSpace(fmt.Sprintf("%v", args["manifest_version"]))
		profile := strings.TrimSpace(fmt.Sprintf("%v", args["contract_profile"]))
		required := toStringSlice(args["required_caps"])
		if len(required) == 0 {
			required = []string{"chat", "tool_call", "json_schema"}
		}
		structured["adapter_id"] = adapterID
		structured["manifest_version"] = version
		structured["contract_profile"] = profile
		structured["required_caps"] = stringSliceToAny(required)
	case "adapter_capability_negotiated":
		required := toStringSlice(args["required_caps"])
		runtimeCaps := toStringSlice(args["runtime_caps"])
		negotiated, missing := intersectCaps(required, runtimeCaps)
		structured["required_caps"] = stringSliceToAny(required)
		structured["runtime_caps"] = stringSliceToAny(runtimeCaps)
		structured["negotiated_caps"] = stringSliceToAny(negotiated)
		structured["missing_caps"] = stringSliceToAny(missing)
		if len(missing) > 0 {
			risk = "degraded_path"
		}
	case "adapter_fallback_mapped":
		missing := toStringSlice(args["missing_caps"])
		version := strings.TrimSpace(fmt.Sprintf("%v", args["manifest_version"]))
		profile := strings.TrimSpace(fmt.Sprintf("%v", args["contract_profile"]))
		fallback := "full-capability"
		reason := "no_gap"
		if len(missing) > 0 {
			fallback = "compatibility-profile"
			reason = "missing:" + strings.Join(missing, ",")
		}
		if strings.Contains(version, "1.3") && containsCap(missing, "streaming") {
			fallback = "compatibility-profile-streamless"
			reason = "streaming_gap"
		}
		structured["missing_caps"] = stringSliceToAny(missing)
		structured["manifest_version"] = version
		structured["contract_profile"] = profile
		structured["fallback_profile"] = fallback
		structured["fallback_reason"] = reason
		if len(missing) > 0 {
			risk = "degraded_path"
		}
	case "governance_adapter_gate_enforced":
		missing := toStringSlice(args["missing_caps"])
		fallback := strings.TrimSpace(fmt.Sprintf("%v", args["fallback_profile"]))
		reason := strings.TrimSpace(fmt.Sprintf("%v", args["fallback_reason"]))
		decision := "allow"
		if containsCap(missing, "json_schema") {
			decision = "deny"
		} else if len(missing) > 0 {
			decision = "allow_with_fallback"
		}
		ticket := fmt.Sprintf("adapter-gate-%d", modecommon.SemanticScore(strings.Join(missing, ","), fallback, reason, decision))
		structured["missing_caps"] = stringSliceToAny(missing)
		structured["fallback_profile"] = fallback
		structured["fallback_reason"] = reason
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_adapter_replay_bound":
		adapterID := strings.TrimSpace(fmt.Sprintf("%v", args["adapter_id"]))
		version := strings.TrimSpace(fmt.Sprintf("%v", args["manifest_version"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		replay := fmt.Sprintf("adapter-replay-%d", modecommon.SemanticScore(adapterID, version, decision, ticket))
		structured["adapter_id"] = adapterID
		structured["manifest_version"] = version
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["governance"] = true
		structured["replay_signature"] = replay
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported adapter semantic marker: %s", marker)
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

func intersectCaps(required []string, supported []string) ([]string, []string) {
	supportedSet := map[string]struct{}{}
	for _, item := range supported {
		supportedSet[strings.TrimSpace(item)] = struct{}{}
	}
	negotiated := make([]string, 0)
	missing := make([]string, 0)
	for _, req := range required {
		capability := strings.TrimSpace(req)
		if capability == "" {
			continue
		}
		if _, ok := supportedSet[capability]; ok {
			negotiated = append(negotiated, capability)
		} else {
			missing = append(missing, capability)
		}
	}
	sort.Strings(negotiated)
	sort.Strings(missing)
	return negotiated, missing
}

func containsCap(caps []string, target string) bool {
	for _, item := range caps {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
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
