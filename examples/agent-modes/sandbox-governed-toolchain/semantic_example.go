package sandboxgovernedtoolchain

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
	patternName      = "sandbox-governed-toolchain"
	phase            = "P0"
	semanticAnchor   = "sandbox.allow_deny_egress_fallback"
	classification   = "sandbox.toolchain_governance"
	semanticToolName = "mode_sandbox_governed_toolchain_semantic_step"
)

type sandboxStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type sandboxPolicy struct {
	ID                 string
	Version            int
	AllowedTools       []string
	DeniedCapabilities []string
	AllowedEgressHosts []string
	FallbackMode       string
	DenyByDefault      bool
}

type toolRequest struct {
	ToolName        string
	Capability      string
	TargetDomain    string
	EgressHost      string
	Sensitivity     string
	RequiresNetwork bool
}

type sandboxState struct {
	Query              string
	PolicyID           string
	PolicyVersion      int
	RequestedTool      string
	RequestedEgress    string
	AllowDecision      string
	DenyReason         string
	EgressDecision     string
	FallbackPath       string
	GovernanceDecision string
	ReplayBinding      string
	AllowedTools       []string
	BlockedTools       []string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"runtime/security", "tool/local"}

var minimalSemanticSteps = []sandboxStep{
	{
		Marker:        "sandbox_allow_deny_classified",
		RuntimeDomain: "runtime/security",
		Intent:        "classify tool requests into allow or deny before execution",
		Outcome:       "primary request receives deterministic allow or deny decision",
	},
	{
		Marker:        "sandbox_egress_allowlist_checked",
		RuntimeDomain: "tool/local",
		Intent:        "check outbound egress host against sandbox allowlist",
		Outcome:       "egress decision becomes allow, deny_not_allowlisted, or not_required",
	},
	{
		Marker:        "sandbox_fallback_path_emitted",
		RuntimeDomain: "runtime/security",
		Intent:        "emit fallback path when primary or egress decision blocks direct execution",
		Outcome:       "fallback route is persisted for controlled continuation",
	},
}

var productionGovernanceSteps = []sandboxStep{
	{
		Marker:        "governance_sandbox_gate_enforced",
		RuntimeDomain: "runtime/security",
		Intent:        "enforce governance gate from allow/deny and fallback outcomes",
		Outcome:       "governance decision allow/allow_with_record/block is produced",
	},
	{
		Marker:        "governance_sandbox_replay_bound",
		RuntimeDomain: "tool/local",
		Intent:        "bind replay signature to policy version and sandbox decisions",
		Outcome:       "deterministic replay signature is emitted",
	},
}

var minimalPolicy = sandboxPolicy{
	ID:                 "sandbox-policy-v1",
	Version:            1,
	AllowedTools:       []string{"local_fs_scan", "local_grep", "local_structured_report"},
	DeniedCapabilities: []string{"shell_write_outside_workspace", "remote_exec"},
	AllowedEgressHosts: []string{"packages.internal.local", "metrics.internal.local"},
	FallbackMode:       "manual_review",
	DenyByDefault:      false,
}

var productionPolicy = sandboxPolicy{
	ID:                 "sandbox-policy-v2-strict",
	Version:            2,
	AllowedTools:       []string{"local_fs_scan", "local_grep", "local_structured_report", "net_http_fetch"},
	DeniedCapabilities: []string{"shell_write_outside_workspace", "remote_exec", "credential_exfiltration"},
	AllowedEgressHosts: []string{"packages.internal.local", "mirror.security.local"},
	FallbackMode:       "offline_cache",
	DenyByDefault:      true,
}

var minimalRequests = []toolRequest{
	{
		ToolName:        "local_fs_scan",
		Capability:      "read_workspace",
		TargetDomain:    "workspace",
		EgressHost:      "",
		Sensitivity:     "internal",
		RequiresNetwork: false,
	},
	{
		ToolName:        "net_http_fetch",
		Capability:      "remote_exec",
		TargetDomain:    "network",
		EgressHost:      "api.vendor.example",
		Sensitivity:     "restricted",
		RequiresNetwork: true,
	},
}

var productionRequests = []toolRequest{
	{
		ToolName:        "net_http_fetch",
		Capability:      "read_remote_metadata",
		TargetDomain:    "network",
		EgressHost:      "api.vendor.example",
		Sensitivity:     "internal",
		RequiresNetwork: true,
	},
	{
		ToolName:        "local_structured_report",
		Capability:      "read_workspace",
		TargetDomain:    "workspace",
		EgressHost:      "",
		Sensitivity:     "internal",
		RequiresNetwork: false,
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
	if _, err := reg.Register(&sandboxGovernanceTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(newSandboxWorkflowModel(variant), runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute sandbox governed toolchain workflow",
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

func stepsForVariant(variant string) []sandboxStep {
	steps := make([]sandboxStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

func newSandboxWorkflowModel(variant string) *sandboxWorkflowModel {
	policy := policyForVariant(variant)
	return &sandboxWorkflowModel{
		variant: variant,
		state: sandboxState{
			Query:         queryForVariant(variant),
			PolicyID:      policy.ID,
			PolicyVersion: policy.Version,
		},
	}
}

type sandboxWorkflowModel struct {
	variant string
	stage   int
	state   sandboxState
}

func (m *sandboxWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s policy=%s@%d query=%s requested_tool=%s egress=%s allow_decision=%s deny_reason=%s egress_decision=%s fallback=%s governance=%s replay=%s allowed=%s blocked=%s markers=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.PolicyID,
		m.state.PolicyVersion,
		m.state.Query,
		safeString(m.state.RequestedTool, "none"),
		safeString(m.state.RequestedEgress, "none"),
		safeString(m.state.AllowDecision, "none"),
		safeString(m.state.DenyReason, "none"),
		safeString(m.state.EgressDecision, "none"),
		safeString(m.state.FallbackPath, "none"),
		normalizedDecision(m.state.GovernanceDecision),
		safeString(m.state.ReplayBinding, "none"),
		strings.Join(m.state.AllowedTools, ","),
		strings.Join(m.state.BlockedTools, ","),
		strings.Join(markers, ","),
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *sandboxWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}
func (m *sandboxWorkflowModel) absorb(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if query, ok := item.Result.Structured["query"].(string); ok && strings.TrimSpace(query) != "" {
			m.state.Query = query
		}
		if policyID, ok := item.Result.Structured["policy_id"].(string); ok && strings.TrimSpace(policyID) != "" {
			m.state.PolicyID = policyID
		}
		if policyVersion, ok := modecommon.AsInt(item.Result.Structured["policy_version"]); ok {
			m.state.PolicyVersion = policyVersion
		}
		if requestedTool, ok := item.Result.Structured["requested_tool"].(string); ok && strings.TrimSpace(requestedTool) != "" {
			m.state.RequestedTool = requestedTool
		}
		if requestedEgress, ok := item.Result.Structured["requested_egress"].(string); ok && strings.TrimSpace(requestedEgress) != "" {
			m.state.RequestedEgress = requestedEgress
		}
		if allowDecision, ok := item.Result.Structured["allow_decision"].(string); ok && strings.TrimSpace(allowDecision) != "" {
			m.state.AllowDecision = allowDecision
		}
		if denyReason, ok := item.Result.Structured["deny_reason"].(string); ok && strings.TrimSpace(denyReason) != "" {
			m.state.DenyReason = denyReason
		}
		if egressDecision, ok := item.Result.Structured["egress_decision"].(string); ok && strings.TrimSpace(egressDecision) != "" {
			m.state.EgressDecision = egressDecision
		}
		if fallbackPath, ok := item.Result.Structured["fallback_path"].(string); ok && strings.TrimSpace(fallbackPath) != "" {
			m.state.FallbackPath = fallbackPath
		}
		if governanceDecision, ok := item.Result.Structured["governance_decision"].(string); ok && strings.TrimSpace(governanceDecision) != "" {
			m.state.GovernanceDecision = governanceDecision
		}
		if replayBinding, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replayBinding) != "" {
			m.state.ReplayBinding = replayBinding
		}
		if tools, ok := toStringSlice(item.Result.Structured["allowed_tools"]); ok {
			m.state.AllowedTools = tools
		}
		if tools, ok := toStringSlice(item.Result.Structured["blocked_tools"]); ok {
			m.state.BlockedTools = tools
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *sandboxWorkflowModel) argsForStep(step sandboxStep, stage int) map[string]any {
	policy := policyForVariant(m.variant)
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
		"policy_id":           policy.ID,
		"policy_version":      policy.Version,
		"allowed_tools":       toAnySlice(policy.AllowedTools),
		"denied_capabilities": toAnySlice(policy.DeniedCapabilities),
		"allowed_egress":      toAnySlice(policy.AllowedEgressHosts),
		"fallback_mode":       policy.FallbackMode,
		"deny_by_default":     policy.DenyByDefault,
		"request_specs":       toAnySlice(requestSpecsForVariant(m.variant)),
		"requested_tool":      m.state.RequestedTool,
		"requested_egress":    m.state.RequestedEgress,
		"allow_decision":      m.state.AllowDecision,
		"deny_reason":         m.state.DenyReason,
		"egress_decision":     m.state.EgressDecision,
		"fallback_path":       m.state.FallbackPath,
		"governance_decision": m.state.GovernanceDecision,
		"allowed_state_tools": toAnySlice(m.state.AllowedTools),
		"blocked_state_tools": toAnySlice(m.state.BlockedTools),
		"stage":               stage,
	}
	return args
}

type sandboxGovernanceTool struct{}

func (t *sandboxGovernanceTool) Name() string { return semanticToolName }

func (t *sandboxGovernanceTool) Description() string {
	return "execute sandbox governance semantic step"
}

func (t *sandboxGovernanceTool) JSONSchema() map[string]any {
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
			"policy_id",
			"policy_version",
			"allowed_tools",
			"denied_capabilities",
			"allowed_egress",
			"fallback_mode",
			"deny_by_default",
			"request_specs",
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
			"policy_id":           map[string]any{"type": "string"},
			"policy_version":      map[string]any{"type": "integer"},
			"allowed_tools":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"denied_capabilities": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"allowed_egress":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"fallback_mode":       map[string]any{"type": "string"},
			"deny_by_default":     map[string]any{"type": "boolean"},
			"request_specs":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"requested_tool":      map[string]any{"type": "string"},
			"requested_egress":    map[string]any{"type": "string"},
			"allow_decision":      map[string]any{"type": "string"},
			"deny_reason":         map[string]any{"type": "string"},
			"egress_decision":     map[string]any{"type": "string"},
			"fallback_path":       map[string]any{"type": "string"},
			"governance_decision": map[string]any{"type": "string"},
			"allowed_state_tools": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"blocked_state_tools": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"stage":               map[string]any{"type": "integer"},
		},
	}
}

func (t *sandboxGovernanceTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	policyID := strings.TrimSpace(fmt.Sprintf("%v", args["policy_id"]))
	policyVersion, _ := modecommon.AsInt(args["policy_version"])
	allowedTools, _ := toStringSlice(args["allowed_tools"])
	deniedCapabilities, _ := toStringSlice(args["denied_capabilities"])
	allowedEgress, _ := toStringSlice(args["allowed_egress"])
	fallbackMode := strings.TrimSpace(fmt.Sprintf("%v", args["fallback_mode"]))
	denyByDefault := parseBool(args["deny_by_default"], false)
	requestSpecs, _ := toStringSlice(args["request_specs"])
	requests := parseRequestSpecs(requestSpecs)
	requestIndex := mapRequestByTool(requests)
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	requestedTool := safeString(args["requested_tool"], "")
	requestedEgress := safeString(args["requested_egress"], "")
	allowDecision := safeString(args["allow_decision"], "")
	denyReason := safeString(args["deny_reason"], "")
	egressDecision := safeString(args["egress_decision"], "")
	fallbackPath := safeString(args["fallback_path"], "")
	governanceDecision := safeString(args["governance_decision"], "")
	allowedStateTools, _ := toStringSlice(args["allowed_state_tools"])
	blockedStateTools, _ := toStringSlice(args["blocked_state_tools"])

	result := map[string]any{
		"pattern":             pattern,
		"variant":             variant,
		"phase":               phaseValue,
		"semantic_anchor":     anchor,
		"classification":      classValue,
		"marker":              marker,
		"runtime_domain":      runtimeDomain,
		"semantic_intent":     intent,
		"semantic_outcome":    outcome,
		"query":               query,
		"policy_id":           policyID,
		"policy_version":      policyVersion,
		"requested_tool":      requestedTool,
		"requested_egress":    requestedEgress,
		"allow_decision":      allowDecision,
		"deny_reason":         denyReason,
		"egress_decision":     egressDecision,
		"fallback_path":       fallbackPath,
		"governance_decision": governanceDecision,
		"stage":               stage,
		"governance":          false,
	}

	var risk string

	switch marker {
	case "sandbox_allow_deny_classified":
		primary := selectPrimaryRequest(requests)
		allowedStateTools, blockedStateTools = classifyRequests(requests, allowedTools, deniedCapabilities, denyByDefault)
		if primary.ToolName == "" {
			primary = toolRequest{ToolName: "none", RequiresNetwork: false}
		}
		allowDecision, denyReason = evaluatePrimaryAccess(primary, allowedTools, deniedCapabilities, denyByDefault)
		requestedTool = primary.ToolName
		requestedEgress = primary.EgressHost
		result["requested_tool"] = requestedTool
		result["requested_egress"] = safeString(requestedEgress, "none")
		result["allow_decision"] = allowDecision
		result["deny_reason"] = safeString(denyReason, "none")
		result["allowed_tools"] = toAnySlice(uniqueSorted(allowedStateTools))
		result["blocked_tools"] = toAnySlice(uniqueSorted(blockedStateTools))
		if strings.HasPrefix(allowDecision, "deny") {
			risk = "deny_path"
		} else {
			risk = "allow_path"
		}
	case "sandbox_egress_allowlist_checked":
		if requestedTool == "" {
			primary := selectPrimaryRequest(requests)
			requestedTool = primary.ToolName
			requestedEgress = primary.EgressHost
			allowDecision, denyReason = evaluatePrimaryAccess(primary, allowedTools, deniedCapabilities, denyByDefault)
		}
		selected := requestIndex[requestedTool]
		switch {
		case strings.HasPrefix(allowDecision, "deny"):
			egressDecision = "skipped_due_to_deny"
		case !selected.RequiresNetwork:
			egressDecision = "not_required"
		case isAllowedEgress(requestedEgress, allowedEgress):
			egressDecision = "allow"
		default:
			egressDecision = "deny_not_allowlisted"
		}
		result["requested_tool"] = requestedTool
		result["requested_egress"] = safeString(requestedEgress, "none")
		result["allow_decision"] = allowDecision
		result["deny_reason"] = safeString(denyReason, "none")
		result["egress_decision"] = egressDecision
		result["allowed_tools"] = toAnySlice(uniqueSorted(allowedStateTools))
		result["blocked_tools"] = toAnySlice(uniqueSorted(blockedStateTools))
		if egressDecision == "deny_not_allowlisted" {
			risk = "egress_blocked"
		} else {
			risk = "egress_checked"
		}
	case "sandbox_fallback_path_emitted":
		switch {
		case strings.HasPrefix(allowDecision, "deny"):
			fallbackPath = "manual_review"
			mode := strings.TrimSpace(strings.ToLower(fallbackMode))
			if mode != "" && mode != "manual_review" {
				fallbackPath = fallbackMode + "_manual_review"
			}
		case egressDecision == "deny_not_allowlisted":
			fallbackPath = "offline_cached_mirror"
			if fallbackMode == "offline_cache" {
				fallbackPath = "offline_cache_replay"
			}
		default:
			fallbackPath = "direct_execute"
		}
		result["requested_tool"] = requestedTool
		result["requested_egress"] = safeString(requestedEgress, "none")
		result["allow_decision"] = allowDecision
		result["deny_reason"] = safeString(denyReason, "none")
		result["egress_decision"] = egressDecision
		result["fallback_path"] = fallbackPath
		result["allowed_tools"] = toAnySlice(uniqueSorted(allowedStateTools))
		result["blocked_tools"] = toAnySlice(uniqueSorted(blockedStateTools))
		if fallbackPath == "direct_execute" {
			risk = "fallback_not_needed"
		} else {
			risk = "fallback_selected"
		}
	case "governance_sandbox_gate_enforced":
		governanceDecision = "allow"
		switch {
		case strings.HasPrefix(allowDecision, "deny"):
			governanceDecision = "block"
			risk = "governed_block"
		case egressDecision == "deny_not_allowlisted":
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		default:
			risk = "governed_allow"
		}
		if denyByDefault && egressDecision == "allow" {
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		}
		result["requested_tool"] = requestedTool
		result["requested_egress"] = safeString(requestedEgress, "none")
		result["allow_decision"] = allowDecision
		result["deny_reason"] = safeString(denyReason, "none")
		result["egress_decision"] = egressDecision
		result["fallback_path"] = fallbackPath
		result["governance_decision"] = governanceDecision
		result["allowed_tools"] = toAnySlice(uniqueSorted(allowedStateTools))
		result["blocked_tools"] = toAnySlice(uniqueSorted(blockedStateTools))
		result["governance"] = true
	case "governance_sandbox_replay_bound":
		governanceDecision = safeString(governanceDecision, "allow")
		replayBinding := fmt.Sprintf(
			"sandbox-replay-%d",
			modecommon.SemanticScore(
				pattern,
				variant,
				policyID,
				fmt.Sprintf("%d", policyVersion),
				safeString(requestedTool, "none"),
				safeString(requestedEgress, "none"),
				safeString(allowDecision, "none"),
				safeString(egressDecision, "none"),
				safeString(fallbackPath, "none"),
				governanceDecision,
			),
		)
		result["requested_tool"] = safeString(requestedTool, "none")
		result["requested_egress"] = safeString(requestedEgress, "none")
		result["allow_decision"] = safeString(allowDecision, "none")
		result["deny_reason"] = safeString(denyReason, "none")
		result["egress_decision"] = safeString(egressDecision, "none")
		result["fallback_path"] = safeString(fallbackPath, "none")
		result["governance_decision"] = governanceDecision
		result["replay_binding"] = replayBinding
		result["allowed_tools"] = toAnySlice(uniqueSorted(allowedStateTools))
		result["blocked_tools"] = toAnySlice(uniqueSorted(blockedStateTools))
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported sandbox marker: %s", marker)
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
		safeString(result["allow_decision"], allowDecision),
		safeString(result["egress_decision"], egressDecision),
		safeString(result["governance_decision"], governanceDecision),
	)
	result["risk"] = risk

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s tool=%s allow=%s egress=%s governance=%s risk=%s",
		pattern,
		variant,
		marker,
		safeString(result["requested_tool"], requestedTool),
		safeString(result["allow_decision"], allowDecision),
		safeString(result["egress_decision"], egressDecision),
		normalizedDecision(safeString(result["governance_decision"], governanceDecision)),
		risk,
	)
	return types.ToolResult{Content: content, Structured: result}, nil
}

func policyForVariant(variant string) sandboxPolicy {
	if variant == modecommon.VariantProduction {
		return productionPolicy
	}
	return minimalPolicy
}

func queryForVariant(variant string) string {
	if variant == modecommon.VariantProduction {
		return "run network metadata fetch with sandbox governance"
	}
	return "scan workspace safely and avoid remote execution"
}

func requestSpecsForVariant(variant string) []string {
	requests := minimalRequests
	if variant == modecommon.VariantProduction {
		requests = productionRequests
	}
	specs := make([]string, 0, len(requests))
	for _, req := range requests {
		specs = append(specs, fmt.Sprintf(
			"%s|%s|%s|%s|%s|%t",
			req.ToolName,
			req.Capability,
			req.TargetDomain,
			req.EgressHost,
			req.Sensitivity,
			req.RequiresNetwork,
		))
	}
	return specs
}

func parseRequestSpecs(specs []string) []toolRequest {
	out := make([]toolRequest, 0, len(specs))
	for _, spec := range specs {
		parts := strings.Split(spec, "|")
		if len(parts) != 6 {
			continue
		}
		out = append(out, toolRequest{
			ToolName:        strings.TrimSpace(parts[0]),
			Capability:      strings.TrimSpace(parts[1]),
			TargetDomain:    strings.TrimSpace(parts[2]),
			EgressHost:      strings.TrimSpace(parts[3]),
			Sensitivity:     strings.TrimSpace(parts[4]),
			RequiresNetwork: parseBool(parts[5], false),
		})
	}
	if len(out) == 0 {
		out = append(out, minimalRequests...)
	}
	return out
}

func mapRequestByTool(requests []toolRequest) map[string]toolRequest {
	index := make(map[string]toolRequest, len(requests))
	for _, req := range requests {
		index[req.ToolName] = req
	}
	return index
}

func selectPrimaryRequest(requests []toolRequest) toolRequest {
	if len(requests) == 0 {
		return toolRequest{}
	}
	for _, req := range requests {
		if req.RequiresNetwork {
			return req
		}
	}
	return requests[0]
}

func classifyRequests(requests []toolRequest, allowedTools []string, deniedCapabilities []string, denyByDefault bool) ([]string, []string) {
	allowedSet := make(map[string]struct{}, len(allowedTools))
	for _, tool := range allowedTools {
		allowedSet[strings.TrimSpace(tool)] = struct{}{}
	}
	deniedSet := make(map[string]struct{}, len(deniedCapabilities))
	for _, cap := range deniedCapabilities {
		deniedSet[strings.TrimSpace(cap)] = struct{}{}
	}
	allowed := make([]string, 0, len(requests))
	blocked := make([]string, 0, len(requests))
	for _, req := range requests {
		_, toolAllowed := allowedSet[req.ToolName]
		_, capDenied := deniedSet[req.Capability]
		if capDenied {
			blocked = append(blocked, req.ToolName+":capability_denied")
			continue
		}
		if toolAllowed {
			allowed = append(allowed, req.ToolName)
			continue
		}
		if denyByDefault {
			blocked = append(blocked, req.ToolName+":deny_by_default")
		} else {
			allowed = append(allowed, req.ToolName+":shadow_allow")
		}
	}
	return uniqueSorted(allowed), uniqueSorted(blocked)
}

func evaluatePrimaryAccess(primary toolRequest, allowedTools []string, deniedCapabilities []string, denyByDefault bool) (string, string) {
	if primary.ToolName == "" {
		return "deny_no_request", "no_request"
	}
	for _, cap := range deniedCapabilities {
		if strings.TrimSpace(cap) == primary.Capability {
			return "deny_capability", "capability_denied"
		}
	}
	for _, tool := range allowedTools {
		if strings.TrimSpace(tool) == primary.ToolName {
			return "allow_primary", ""
		}
	}
	if denyByDefault {
		return "deny_by_default", "tool_not_allowlisted"
	}
	return "allow_shadow", "tool_not_allowlisted_but_permitted"
}

func isAllowedEgress(host string, allowlist []string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return true
	}
	for _, item := range allowlist {
		if strings.TrimSpace(strings.ToLower(item)) == host {
			return true
		}
	}
	return false
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
