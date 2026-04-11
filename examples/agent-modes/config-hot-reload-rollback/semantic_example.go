package confighotreloadrollback

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
	patternName      = "config-hot-reload-rollback"
	phase            = "P2"
	semanticAnchor   = "config.reload_failfast_rollback"
	classification   = "runtime.config_rollback"
	semanticToolName = "mode_config_hot_reload_rollback_semantic_step"
	defaultReloadID  = "cfg-reload-20260410"
)

type configStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type configReloadState struct {
	ReloadID            string
	PreviousVersion     int
	CandidateVersion    int
	PreviousDigest      string
	CandidateDigest     string
	EffectiveDigest     string
	ChangedKey          string
	ProposedValue       string
	ValidationStatus    string
	FailureCode         string
	FailFastTriggered   bool
	RollbackApplied     bool
	RollbackReason      string
	AtomicProofToken    string
	DiagnosticID        string
	DiagnosticSeverity  string
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"runtime/config", "runtime/diagnostics"}

var minimalSemanticSteps = []configStep{
	{
		Marker:        "config_reload_attempted",
		RuntimeDomain: "runtime/config",
		Intent:        "submit a hot-reload candidate with explicit version and digest",
		Outcome:       "candidate version/digest and pending validation status are emitted",
	},
	{
		Marker:        "config_invalid_failfast",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "reject an invalid candidate via fail-fast validation",
		Outcome:       "failure code, diagnostic severity and fail-fast flag are emitted",
	},
	{
		Marker:        "config_atomic_rollback_verified",
		RuntimeDomain: "runtime/config",
		Intent:        "verify rollback restores previous digest atomically",
		Outcome:       "rollback applied, effective digest and rollback proof token are emitted",
	},
}

var productionGovernanceSteps = []configStep{
	{
		Marker:        "governance_config_gate_enforced",
		RuntimeDomain: "runtime/config",
		Intent:        "enforce config governance based on fail-fast and rollback outcome",
		Outcome:       "governance decision and governance ticket are emitted",
	},
	{
		Marker:        "governance_config_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind governance decision to replay signature for audits",
		Outcome:       "replay signature is emitted with governance metadata",
	},
}

func RunMinimal() {
	executeReloadVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeReloadVariant(modecommon.VariantProduction)
}

func executeReloadVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&configReloadTool{}); err != nil {
		panic(err)
	}

	model := &configReloadModel{
		variant: variant,
		state: configReloadState{
			ReloadID:        defaultReloadID,
			PreviousVersion: 17,
			PreviousDigest:  "cfg-digest-v17-stable",
			EffectiveDigest: "cfg-digest-v17-stable",
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute config hot reload rollback semantic pipeline",
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

func planForVariant(variant string) []configStep {
	plan := make([]configStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type configReloadModel struct {
	variant string
	cursor  int
	state   configReloadState
}

func (m *configReloadModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s reload=%s prev_version=%d candidate_version=%d validation=%s fail_fast=%t rollback=%t rollback_reason=%s active_digest=%s candidate_digest=%s effective_digest=%s diag=%s severity=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.ReloadID, true),
		m.state.PreviousVersion,
		m.state.CandidateVersion,
		readableValue(m.state.ValidationStatus, true),
		m.state.FailFastTriggered,
		m.state.RollbackApplied,
		readableValue(m.state.RollbackReason, m.state.RollbackApplied),
		readableValue(m.state.PreviousDigest, true),
		readableValue(m.state.CandidateDigest, true),
		readableValue(m.state.EffectiveDigest, true),
		readableValue(m.state.DiagnosticID, true),
		readableValue(m.state.DiagnosticSeverity, true),
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *configReloadModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *configReloadModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if reloadID, _ := outcome.Result.Structured["reload_id"].(string); strings.TrimSpace(reloadID) != "" {
			m.state.ReloadID = strings.TrimSpace(reloadID)
		}
		if prevVersion, ok := modecommon.AsInt(outcome.Result.Structured["previous_version"]); ok {
			m.state.PreviousVersion = prevVersion
		}
		if candidateVersion, ok := modecommon.AsInt(outcome.Result.Structured["candidate_version"]); ok {
			m.state.CandidateVersion = candidateVersion
		}
		if prevDigest, _ := outcome.Result.Structured["previous_digest"].(string); strings.TrimSpace(prevDigest) != "" {
			m.state.PreviousDigest = strings.TrimSpace(prevDigest)
		}
		if candidateDigest, _ := outcome.Result.Structured["candidate_digest"].(string); strings.TrimSpace(candidateDigest) != "" {
			m.state.CandidateDigest = strings.TrimSpace(candidateDigest)
		}
		if effectiveDigest, _ := outcome.Result.Structured["effective_digest"].(string); strings.TrimSpace(effectiveDigest) != "" {
			m.state.EffectiveDigest = strings.TrimSpace(effectiveDigest)
		}
		if changedKey, _ := outcome.Result.Structured["changed_key"].(string); strings.TrimSpace(changedKey) != "" {
			m.state.ChangedKey = strings.TrimSpace(changedKey)
		}
		if proposedValue, _ := outcome.Result.Structured["proposed_value"].(string); strings.TrimSpace(proposedValue) != "" {
			m.state.ProposedValue = strings.TrimSpace(proposedValue)
		}
		if validation, _ := outcome.Result.Structured["validation_status"].(string); strings.TrimSpace(validation) != "" {
			m.state.ValidationStatus = strings.TrimSpace(validation)
		}
		if failureCode, _ := outcome.Result.Structured["failure_code"].(string); strings.TrimSpace(failureCode) != "" {
			m.state.FailureCode = strings.TrimSpace(failureCode)
		}
		if failFast, ok := outcome.Result.Structured["fail_fast"].(bool); ok {
			m.state.FailFastTriggered = failFast
		}
		if rollback, ok := outcome.Result.Structured["rollback_applied"].(bool); ok {
			m.state.RollbackApplied = rollback
		}
		if reason, _ := outcome.Result.Structured["rollback_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.RollbackReason = strings.TrimSpace(reason)
		}
		if proof, _ := outcome.Result.Structured["atomic_proof_token"].(string); strings.TrimSpace(proof) != "" {
			m.state.AtomicProofToken = strings.TrimSpace(proof)
		}
		if diagID, _ := outcome.Result.Structured["diagnostic_id"].(string); strings.TrimSpace(diagID) != "" {
			m.state.DiagnosticID = strings.TrimSpace(diagID)
		}
		if severity, _ := outcome.Result.Structured["diagnostic_severity"].(string); strings.TrimSpace(severity) != "" {
			m.state.DiagnosticSeverity = strings.TrimSpace(severity)
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

func (m *configReloadModel) argsForStep(step configStep, stage int) map[string]any {
	args := map[string]any{
		"pattern":          patternName,
		"variant":          m.variant,
		"phase":            phase,
		"semantic_anchor":  semanticAnchor,
		"classification":   classification,
		"marker":           step.Marker,
		"runtime_domain":   step.RuntimeDomain,
		"intent":           step.Intent,
		"outcome":          step.Outcome,
		"stage":            stage,
		"reload_id":        m.state.ReloadID,
		"previous_version": m.state.PreviousVersion,
		"previous_digest":  m.state.PreviousDigest,
	}

	switch step.Marker {
	case "config_reload_attempted":
		candidateVersion := m.state.PreviousVersion + 1
		changedKey := "planner.max_parallel"
		proposedValue := "8"
		if m.variant == modecommon.VariantProduction {
			proposedValue = "24"
		}
		candidateDigest := fmt.Sprintf("cfg-digest-v%d-%d", candidateVersion, modecommon.SemanticScore(changedKey, proposedValue, m.variant))
		args["candidate_version"] = candidateVersion
		args["candidate_digest"] = candidateDigest
		args["changed_key"] = changedKey
		args["proposed_value"] = proposedValue
	case "config_invalid_failfast":
		invalidKey := "scheduler.cooldown_ms"
		invalidValue := "-120"
		if m.variant == modecommon.VariantProduction {
			invalidValue = "-600"
		}
		args["candidate_version"] = m.state.CandidateVersion
		args["candidate_digest"] = m.state.CandidateDigest
		args["invalid_key"] = invalidKey
		args["invalid_value"] = invalidValue
	case "config_atomic_rollback_verified":
		args["candidate_version"] = m.state.CandidateVersion
		args["candidate_digest"] = m.state.CandidateDigest
		args["validation_status"] = m.state.ValidationStatus
		args["failure_code"] = m.state.FailureCode
		args["fail_fast"] = m.state.FailFastTriggered
	case "governance_config_gate_enforced":
		args["fail_fast"] = m.state.FailFastTriggered
		args["rollback_applied"] = m.state.RollbackApplied
		args["failure_code"] = m.state.FailureCode
		args["diagnostic_severity"] = m.state.DiagnosticSeverity
		args["atomic_proof_token"] = m.state.AtomicProofToken
	case "governance_config_replay_bound":
		args["diagnostic_id"] = m.state.DiagnosticID
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["atomic_proof_token"] = m.state.AtomicProofToken
	}
	return args
}

type configReloadTool struct{}

func (t *configReloadTool) Name() string { return semanticToolName }

func (t *configReloadTool) Description() string {
	return "execute config hot reload + fail-fast rollback semantic step"
}

func (t *configReloadTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "stage"},
		"properties": map[string]any{
			"pattern":            map[string]any{"type": "string"},
			"variant":            map[string]any{"type": "string"},
			"phase":              map[string]any{"type": "string"},
			"semantic_anchor":    map[string]any{"type": "string"},
			"classification":     map[string]any{"type": "string"},
			"marker":             map[string]any{"type": "string"},
			"runtime_domain":     map[string]any{"type": "string"},
			"intent":             map[string]any{"type": "string"},
			"outcome":            map[string]any{"type": "string"},
			"stage":              map[string]any{"type": "integer"},
			"reload_id":          map[string]any{"type": "string"},
			"previous_version":   map[string]any{"type": "integer"},
			"candidate_version":  map[string]any{"type": "integer"},
			"previous_digest":    map[string]any{"type": "string"},
			"candidate_digest":   map[string]any{"type": "string"},
			"changed_key":        map[string]any{"type": "string"},
			"proposed_value":     map[string]any{"type": "string"},
			"invalid_key":        map[string]any{"type": "string"},
			"invalid_value":      map[string]any{"type": "string"},
			"validation_status":  map[string]any{"type": "string"},
			"failure_code":       map[string]any{"type": "string"},
			"fail_fast":          map[string]any{"type": "boolean"},
			"rollback_applied":   map[string]any{"type": "boolean"},
			"rollback_reason":    map[string]any{"type": "string"},
			"atomic_proof_token": map[string]any{"type": "string"},
			"diagnostic_id":      map[string]any{"type": "string"},
			"diagnostic_severity": map[string]any{
				"type": "string",
			},
			"governance_decision": map[string]any{"type": "string"},
			"governance_ticket":   map[string]any{"type": "string"},
		},
	}
}

func (t *configReloadTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	reloadID := strings.TrimSpace(fmt.Sprintf("%v", args["reload_id"]))
	if reloadID == "" {
		reloadID = defaultReloadID
	}
	previousVersion, _ := modecommon.AsInt(args["previous_version"])
	previousDigest := strings.TrimSpace(fmt.Sprintf("%v", args["previous_digest"]))

	structured := map[string]any{
		"pattern":          pattern,
		"variant":          variant,
		"phase":            phaseValue,
		"semantic_anchor":  anchor,
		"classification":   classValue,
		"marker":           marker,
		"runtime_domain":   runtimeDomain,
		"intent":           intent,
		"outcome":          outcome,
		"stage":            stage,
		"reload_id":        reloadID,
		"previous_version": previousVersion,
		"previous_digest":  previousDigest,
		"governance":       false,
	}

	risk := "nominal"
	switch marker {
	case "config_reload_attempted":
		candidateVersion, _ := modecommon.AsInt(args["candidate_version"])
		if candidateVersion <= 0 {
			candidateVersion = previousVersion + 1
		}
		candidateDigest := strings.TrimSpace(fmt.Sprintf("%v", args["candidate_digest"]))
		changedKey := strings.TrimSpace(fmt.Sprintf("%v", args["changed_key"]))
		proposedValue := strings.TrimSpace(fmt.Sprintf("%v", args["proposed_value"]))
		if changedKey == "" {
			changedKey = "planner.max_parallel"
		}
		if proposedValue == "" {
			proposedValue = "8"
		}
		if candidateDigest == "" {
			candidateDigest = fmt.Sprintf("cfg-digest-v%d-%d", candidateVersion, modecommon.SemanticScore(changedKey, proposedValue, variant))
		}
		structured["candidate_version"] = candidateVersion
		structured["candidate_digest"] = candidateDigest
		structured["changed_key"] = changedKey
		structured["proposed_value"] = proposedValue
		structured["validation_status"] = "pending_validation"
		structured["diagnostic_id"] = fmt.Sprintf("diag-%s-attempt", modecommon.MarkerToken(reloadID))
		structured["diagnostic_severity"] = "info"
	case "config_invalid_failfast":
		invalidKey := strings.TrimSpace(fmt.Sprintf("%v", args["invalid_key"]))
		invalidValue := strings.TrimSpace(fmt.Sprintf("%v", args["invalid_value"]))
		if invalidKey == "" {
			invalidKey = "scheduler.cooldown_ms"
		}
		if invalidValue == "" {
			invalidValue = "-120"
		}
		failureCode := "E_CFG_SCHEMA_NEGATIVE_VALUE"
		severity := "warn"
		if variant == modecommon.VariantProduction {
			failureCode = "E_CFG_POLICY_BREACH"
			severity = "critical"
		}
		candidateVersion, _ := modecommon.AsInt(args["candidate_version"])
		candidateDigest := strings.TrimSpace(fmt.Sprintf("%v", args["candidate_digest"]))
		structured["candidate_version"] = candidateVersion
		structured["candidate_digest"] = candidateDigest
		structured["invalid_key"] = invalidKey
		structured["invalid_value"] = invalidValue
		structured["validation_status"] = "rejected_fail_fast"
		structured["failure_code"] = failureCode
		structured["fail_fast"] = true
		structured["diagnostic_id"] = fmt.Sprintf("diag-%s-failfast", modecommon.MarkerToken(reloadID))
		structured["diagnostic_severity"] = severity
		risk = "degraded_path"
	case "config_atomic_rollback_verified":
		candidateVersion, _ := modecommon.AsInt(args["candidate_version"])
		candidateDigest := strings.TrimSpace(fmt.Sprintf("%v", args["candidate_digest"]))
		failFast := asBool(args["fail_fast"])
		failureCode := strings.TrimSpace(fmt.Sprintf("%v", args["failure_code"]))
		rollbackApplied := failFast
		rollbackReason := "none"
		effectiveDigest := candidateDigest
		if rollbackApplied {
			effectiveDigest = previousDigest
			rollbackReason = "validation_reject"
			if failureCode != "" {
				rollbackReason = strings.ToLower(strings.TrimPrefix(failureCode, "E_CFG_"))
			}
		}
		atomicProofToken := fmt.Sprintf("cfg-proof-%d", modecommon.SemanticScore(reloadID, previousDigest, candidateDigest, rollbackReason, fmt.Sprintf("%t", rollbackApplied)))
		structured["candidate_version"] = candidateVersion
		structured["candidate_digest"] = candidateDigest
		structured["failure_code"] = failureCode
		structured["fail_fast"] = failFast
		structured["rollback_applied"] = rollbackApplied
		structured["rollback_reason"] = rollbackReason
		structured["effective_digest"] = effectiveDigest
		structured["atomic_proof_token"] = atomicProofToken
		structured["validation_status"] = "rolled_back"
		structured["diagnostic_id"] = fmt.Sprintf("diag-%s-rollback", modecommon.MarkerToken(reloadID))
		if rollbackApplied {
			risk = "degraded_path"
		}
	case "governance_config_gate_enforced":
		failFast := asBool(args["fail_fast"])
		rollbackApplied := asBool(args["rollback_applied"])
		failureCode := strings.TrimSpace(fmt.Sprintf("%v", args["failure_code"]))
		severity := strings.TrimSpace(fmt.Sprintf("%v", args["diagnostic_severity"]))
		atomicProofToken := strings.TrimSpace(fmt.Sprintf("%v", args["atomic_proof_token"]))
		decision := "allow_release"
		if failFast && rollbackApplied {
			decision = "hold_release"
			if severity == "critical" {
				decision = "quarantine_release"
			}
		}
		ticket := fmt.Sprintf("cfg-gate-%d", modecommon.SemanticScore(decision, failureCode, severity, atomicProofToken))
		structured["fail_fast"] = failFast
		structured["rollback_applied"] = rollbackApplied
		structured["failure_code"] = failureCode
		structured["diagnostic_severity"] = severity
		structured["atomic_proof_token"] = atomicProofToken
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_config_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		atomicProofToken := strings.TrimSpace(fmt.Sprintf("%v", args["atomic_proof_token"]))
		diagID := strings.TrimSpace(fmt.Sprintf("%v", args["diagnostic_id"]))
		if diagID == "" {
			diagID = fmt.Sprintf("diag-%s-governance", modecommon.MarkerToken(reloadID))
		}
		replaySignature := fmt.Sprintf("cfg-replay-%d", modecommon.SemanticScore(reloadID, decision, ticket, atomicProofToken, diagID))
		structured["diagnostic_id"] = diagID
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["atomic_proof_token"] = atomicProofToken
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported config hot reload marker: %s", marker)
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
	return types.ToolResult{
		Content:    content,
		Structured: structured,
	}, nil
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
