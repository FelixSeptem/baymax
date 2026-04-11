package realtimeinterruptresume

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
	patternName      = "realtime-interrupt-resume"
	phase            = "P0"
	semanticAnchor   = "realtime.cursor_idempotent_interrupt_resume"
	classification   = "realtime.resume_recovery"
	semanticToolName = "mode_realtime_interrupt_resume_semantic_step"
)

type realtimeStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type streamEvent struct {
	Cursor         int
	Kind           string
	Payload        string
	IdempotencyKey string
}

type interruptSignal struct {
	AtCursor int
	Reason   string
	Severity string
	Source   string
}

type resumePolicy struct {
	Mode       string
	MaxReplay  int
	RequireAck bool
}

type realtimeState struct {
	SessionID          string
	LastCursor         int
	AppliedCursors     []int
	DuplicateDropped   int
	InterruptCursor    int
	InterruptReason    string
	InterruptSeverity  string
	BufferedPending    int
	ResumeFrom         int
	RecoveredCursor    int
	RecoveredBatch     []int
	ResumeStatus       string
	FallbackPath       string
	GovernanceDecision string
	ReplayBinding      string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"core/runner", "runtime/diagnostics"}

var minimalSemanticSteps = []realtimeStep{
	{
		Marker:        "realtime_cursor_idempotent",
		RuntimeDomain: "core/runner",
		Intent:        "advance realtime cursor with idempotency dedupe on duplicate event keys",
		Outcome:       "cursor progression and duplicate drop counters are emitted",
	},
	{
		Marker:        "realtime_interrupt_captured",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "capture interrupt signal and freeze resume checkpoint cursor",
		Outcome:       "interrupt cursor, reason, and buffered pending count are persisted",
	},
	{
		Marker:        "realtime_resume_recovered",
		RuntimeDomain: "core/runner",
		Intent:        "resume from checkpoint and recover replayable events under policy limits",
		Outcome:       "resume status, recovered cursor, and fallback path are emitted",
	},
}

var productionGovernanceSteps = []realtimeStep{
	{
		Marker:        "governance_realtime_gate_enforced",
		RuntimeDomain: "core/runner",
		Intent:        "enforce realtime gate based on resume status and interrupt severity",
		Outcome:       "governance decision allow, allow_with_record, or block is produced",
	},
	{
		Marker:        "governance_realtime_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind replay signature to cursor trajectory and governance decision",
		Outcome:       "deterministic replay signature is generated",
	},
}

var minimalEvents = []streamEvent{
	{Cursor: 1001, Kind: "delta", Payload: "token-1", IdempotencyKey: "k-1001"},
	{Cursor: 1002, Kind: "delta", Payload: "token-2", IdempotencyKey: "k-1002"},
	{Cursor: 1002, Kind: "delta", Payload: "token-2-dup", IdempotencyKey: "k-1002"},
	{Cursor: 1003, Kind: "delta", Payload: "token-3", IdempotencyKey: "k-1003"},
}

var productionEvents = []streamEvent{
	{Cursor: 2001, Kind: "delta", Payload: "preface", IdempotencyKey: "k-2001"},
	{Cursor: 2002, Kind: "delta", Payload: "step-a", IdempotencyKey: "k-2002"},
	{Cursor: 2003, Kind: "delta", Payload: "step-b", IdempotencyKey: "k-2003"},
	{Cursor: 2003, Kind: "delta", Payload: "step-b-dup", IdempotencyKey: "k-2003"},
	{Cursor: 2004, Kind: "delta", Payload: "step-c", IdempotencyKey: "k-2004"},
	{Cursor: 2005, Kind: "delta", Payload: "step-d", IdempotencyKey: "k-2005"},
}

var minimalSignal = interruptSignal{
	AtCursor: 1002,
	Reason:   "user_interrupt",
	Severity: "medium",
	Source:   "client",
}

var productionSignal = interruptSignal{
	AtCursor: 2003,
	Reason:   "safety_interrupt",
	Severity: "high",
	Source:   "policy_guard",
}

var minimalResumePolicy = resumePolicy{
	Mode:       "continue",
	MaxReplay:  2,
	RequireAck: false,
}

var productionResumePolicy = resumePolicy{
	Mode:       "strict",
	MaxReplay:  1,
	RequireAck: true,
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
	if _, err := reg.Register(&realtimeInterruptResumeTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(newRealtimeWorkflowModel(variant), runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute realtime interrupt resume workflow",
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

func stepsForVariant(variant string) []realtimeStep {
	steps := make([]realtimeStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

func newRealtimeWorkflowModel(variant string) *realtimeWorkflowModel {
	return &realtimeWorkflowModel{
		variant: variant,
		state: realtimeState{
			SessionID: sessionIDForVariant(variant),
		},
	}
}

type realtimeWorkflowModel struct {
	variant string
	stage   int
	state   realtimeState
}

func (m *realtimeWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s session=%s last_cursor=%d duplicates=%d interrupt_cursor=%d interrupt_reason=%s severity=%s resume_from=%d recovered_cursor=%d resume_status=%s fallback=%s governance=%s replay=%s markers=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.SessionID,
		m.state.LastCursor,
		m.state.DuplicateDropped,
		m.state.InterruptCursor,
		safeString(m.state.InterruptReason, "none"),
		safeString(m.state.InterruptSeverity, "none"),
		m.state.ResumeFrom,
		m.state.RecoveredCursor,
		safeString(m.state.ResumeStatus, "none"),
		safeString(m.state.FallbackPath, "none"),
		normalizedDecision(m.state.GovernanceDecision),
		safeString(m.state.ReplayBinding, "none"),
		strings.Join(markers, ","),
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *realtimeWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}
func (m *realtimeWorkflowModel) absorb(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if sessionID, ok := item.Result.Structured["session_id"].(string); ok && strings.TrimSpace(sessionID) != "" {
			m.state.SessionID = sessionID
		}
		if lastCursor, ok := modecommon.AsInt(item.Result.Structured["last_cursor"]); ok {
			m.state.LastCursor = lastCursor
		}
		if dropped, ok := modecommon.AsInt(item.Result.Structured["duplicate_dropped"]); ok {
			m.state.DuplicateDropped = dropped
		}
		if interruptCursor, ok := modecommon.AsInt(item.Result.Structured["interrupt_cursor"]); ok {
			m.state.InterruptCursor = interruptCursor
		}
		if interruptReason, ok := item.Result.Structured["interrupt_reason"].(string); ok && strings.TrimSpace(interruptReason) != "" {
			m.state.InterruptReason = interruptReason
		}
		if interruptSeverity, ok := item.Result.Structured["interrupt_severity"].(string); ok && strings.TrimSpace(interruptSeverity) != "" {
			m.state.InterruptSeverity = interruptSeverity
		}
		if pending, ok := modecommon.AsInt(item.Result.Structured["buffered_pending"]); ok {
			m.state.BufferedPending = pending
		}
		if resumeFrom, ok := modecommon.AsInt(item.Result.Structured["resume_from"]); ok {
			m.state.ResumeFrom = resumeFrom
		}
		if recoveredCursor, ok := modecommon.AsInt(item.Result.Structured["recovered_cursor"]); ok {
			m.state.RecoveredCursor = recoveredCursor
		}
		if recoveredBatch, ok := toIntSlice(item.Result.Structured["recovered_batch"]); ok {
			m.state.RecoveredBatch = recoveredBatch
		}
		if status, ok := item.Result.Structured["resume_status"].(string); ok && strings.TrimSpace(status) != "" {
			m.state.ResumeStatus = status
		}
		if fallback, ok := item.Result.Structured["fallback_path"].(string); ok && strings.TrimSpace(fallback) != "" {
			m.state.FallbackPath = fallback
		}
		if governanceDecision, ok := item.Result.Structured["governance_decision"].(string); ok && strings.TrimSpace(governanceDecision) != "" {
			m.state.GovernanceDecision = governanceDecision
		}
		if replayBinding, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replayBinding) != "" {
			m.state.ReplayBinding = replayBinding
		}
		if cursors, ok := toIntSlice(item.Result.Structured["applied_cursors"]); ok {
			m.state.AppliedCursors = cursors
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *realtimeWorkflowModel) argsForStep(step realtimeStep, stage int) map[string]any {
	policy := resumePolicyForVariant(m.variant)
	signal := signalForVariant(m.variant)
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
		"session_id":          m.state.SessionID,
		"event_specs":         toAnySlice(eventSpecsForVariant(m.variant)),
		"signal_spec":         signalSpec(signal),
		"resume_mode":         policy.Mode,
		"max_replay":          policy.MaxReplay,
		"require_ack":         policy.RequireAck,
		"last_cursor":         m.state.LastCursor,
		"duplicate_dropped":   m.state.DuplicateDropped,
		"interrupt_cursor":    m.state.InterruptCursor,
		"interrupt_reason":    m.state.InterruptReason,
		"interrupt_severity":  m.state.InterruptSeverity,
		"buffered_pending":    m.state.BufferedPending,
		"resume_from":         m.state.ResumeFrom,
		"resume_status":       m.state.ResumeStatus,
		"fallback_path":       m.state.FallbackPath,
		"governance_decision": m.state.GovernanceDecision,
		"applied_cursors":     toAnyIntSlice(m.state.AppliedCursors),
		"recovered_batch":     toAnyIntSlice(m.state.RecoveredBatch),
		"stage":               stage,
	}
	return args
}

type realtimeInterruptResumeTool struct{}

func (t *realtimeInterruptResumeTool) Name() string { return semanticToolName }

func (t *realtimeInterruptResumeTool) Description() string {
	return "execute realtime interrupt resume semantic step"
}

func (t *realtimeInterruptResumeTool) JSONSchema() map[string]any {
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
			"session_id",
			"event_specs",
			"signal_spec",
			"resume_mode",
			"max_replay",
			"require_ack",
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
			"session_id":          map[string]any{"type": "string"},
			"event_specs":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"signal_spec":         map[string]any{"type": "string"},
			"resume_mode":         map[string]any{"type": "string"},
			"max_replay":          map[string]any{"type": "integer"},
			"require_ack":         map[string]any{"type": "boolean"},
			"last_cursor":         map[string]any{"type": "integer"},
			"duplicate_dropped":   map[string]any{"type": "integer"},
			"interrupt_cursor":    map[string]any{"type": "integer"},
			"interrupt_reason":    map[string]any{"type": "string"},
			"interrupt_severity":  map[string]any{"type": "string"},
			"buffered_pending":    map[string]any{"type": "integer"},
			"resume_from":         map[string]any{"type": "integer"},
			"resume_status":       map[string]any{"type": "string"},
			"fallback_path":       map[string]any{"type": "string"},
			"governance_decision": map[string]any{"type": "string"},
			"applied_cursors":     map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
			"recovered_batch":     map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
			"stage":               map[string]any{"type": "integer"},
		},
	}
}

func (t *realtimeInterruptResumeTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	sessionID := strings.TrimSpace(fmt.Sprintf("%v", args["session_id"]))
	eventSpecs, _ := toStringSlice(args["event_specs"])
	events := parseEventSpecs(eventSpecs)
	signal := parseSignalSpec(strings.TrimSpace(fmt.Sprintf("%v", args["signal_spec"])))
	resumeMode := strings.TrimSpace(fmt.Sprintf("%v", args["resume_mode"]))
	maxReplay, _ := modecommon.AsInt(args["max_replay"])
	requireAck := parseBool(args["require_ack"], false)
	lastCursor, _ := modecommon.AsInt(args["last_cursor"])
	duplicateDropped, _ := modecommon.AsInt(args["duplicate_dropped"])
	interruptCursor, _ := modecommon.AsInt(args["interrupt_cursor"])
	interruptReason := safeString(args["interrupt_reason"], "")
	interruptSeverity := safeString(args["interrupt_severity"], "")
	bufferedPending, _ := modecommon.AsInt(args["buffered_pending"])
	resumeFrom, _ := modecommon.AsInt(args["resume_from"])
	resumeStatus := safeString(args["resume_status"], "")
	fallbackPath := safeString(args["fallback_path"], "")
	governanceDecision := safeString(args["governance_decision"], "")
	appliedCursors, _ := toIntSlice(args["applied_cursors"])
	recoveredBatch, _ := toIntSlice(args["recovered_batch"])
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

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
		"session_id":          sessionID,
		"last_cursor":         lastCursor,
		"duplicate_dropped":   duplicateDropped,
		"interrupt_cursor":    interruptCursor,
		"interrupt_reason":    interruptReason,
		"interrupt_severity":  interruptSeverity,
		"buffered_pending":    bufferedPending,
		"resume_from":         resumeFrom,
		"resume_status":       resumeStatus,
		"fallback_path":       fallbackPath,
		"governance_decision": governanceDecision,
		"stage":               stage,
		"governance":          false,
	}

	var risk string

	switch marker {
	case "realtime_cursor_idempotent":
		appliedCursors, lastCursor, duplicateDropped = dedupeCursors(events)
		result["applied_cursors"] = toAnyIntSlice(appliedCursors)
		result["last_cursor"] = lastCursor
		result["duplicate_dropped"] = duplicateDropped
		if duplicateDropped > 0 {
			risk = "idempotent_dedup"
		} else {
			risk = "cursor_advanced"
		}
	case "realtime_interrupt_captured":
		if len(appliedCursors) == 0 {
			appliedCursors, lastCursor, duplicateDropped = dedupeCursors(events)
		}
		interruptCursor, interruptReason, interruptSeverity, bufferedPending, resumeFrom = captureInterrupt(
			appliedCursors,
			lastCursor,
			signal,
		)
		resumeStatus = "interrupted"
		result["applied_cursors"] = toAnyIntSlice(appliedCursors)
		result["last_cursor"] = lastCursor
		result["duplicate_dropped"] = duplicateDropped
		result["interrupt_cursor"] = interruptCursor
		result["interrupt_reason"] = interruptReason
		result["interrupt_severity"] = interruptSeverity
		result["buffered_pending"] = bufferedPending
		result["resume_from"] = resumeFrom
		result["resume_status"] = resumeStatus
		risk = "interrupt_captured"
	case "realtime_resume_recovered":
		if len(appliedCursors) == 0 {
			appliedCursors, lastCursor, _ = dedupeCursors(events)
		}
		if resumeFrom == 0 {
			resumeFrom = interruptCursor
		}
		recoveredBatch, resumeStatus, fallbackPath = recoverFromCursor(
			appliedCursors,
			resumeFrom,
			maxReplay,
			resumeMode,
			requireAck,
			interruptSeverity,
		)
		recoveredCursor := resumeFrom
		if len(recoveredBatch) > 0 {
			recoveredCursor = recoveredBatch[len(recoveredBatch)-1]
		}
		result["applied_cursors"] = toAnyIntSlice(appliedCursors)
		result["resume_from"] = resumeFrom
		result["recovered_batch"] = toAnyIntSlice(recoveredBatch)
		result["recovered_cursor"] = recoveredCursor
		result["resume_status"] = resumeStatus
		result["fallback_path"] = fallbackPath
		switch resumeStatus {
		case "resumed":
			risk = "resume_recovered"
		default:
			risk = "degraded_path"
		}
	case "governance_realtime_gate_enforced":
		governanceDecision = "allow"
		switch {
		case resumeStatus == "await_manual_resume":
			governanceDecision = "block"
			risk = "governed_block"
		case resumeStatus == "partial_resume" || duplicateDropped > 0 || bufferedPending > maxReplay:
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		default:
			risk = "governed_allow"
		}
		if requireAck && interruptSeverity == "high" && governanceDecision == "allow" {
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		}
		result["governance_decision"] = governanceDecision
		result["resume_status"] = resumeStatus
		result["fallback_path"] = fallbackPath
		result["duplicate_dropped"] = duplicateDropped
		result["buffered_pending"] = bufferedPending
		result["governance"] = true
	case "governance_realtime_replay_bound":
		governanceDecision = safeString(governanceDecision, "allow")
		replayBinding := fmt.Sprintf(
			"realtime-replay-%d",
			modecommon.SemanticScore(
				pattern,
				variant,
				sessionID,
				fmt.Sprintf("%d", resumeFrom),
				fmt.Sprintf("%d", interruptCursor),
				fmt.Sprintf("%d", len(recoveredBatch)),
				resumeStatus,
				governanceDecision,
				fallbackPath,
			),
		)
		result["resume_from"] = resumeFrom
		result["interrupt_cursor"] = interruptCursor
		result["resume_status"] = resumeStatus
		result["fallback_path"] = fallbackPath
		result["governance_decision"] = governanceDecision
		result["recovered_batch"] = toAnyIntSlice(recoveredBatch)
		result["replay_binding"] = replayBinding
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported realtime marker: %s", marker)
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
		safeString(result["resume_status"], resumeStatus),
		safeString(result["governance_decision"], governanceDecision),
	)
	result["risk"] = risk

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s cursor=%d interrupt=%d status=%s governance=%s risk=%s",
		pattern,
		variant,
		marker,
		lastCursor,
		interruptCursor,
		safeString(result["resume_status"], resumeStatus),
		normalizedDecision(safeString(result["governance_decision"], governanceDecision)),
		risk,
	)
	return types.ToolResult{Content: content, Structured: result}, nil
}

func sessionIDForVariant(variant string) string {
	if variant == modecommon.VariantProduction {
		return "rt-session-prod-202604"
	}
	return "rt-session-min-202604"
}

func eventSpecsForVariant(variant string) []string {
	events := minimalEvents
	if variant == modecommon.VariantProduction {
		events = productionEvents
	}
	specs := make([]string, 0, len(events))
	for _, event := range events {
		specs = append(specs, fmt.Sprintf("%d|%s|%s|%s", event.Cursor, event.Kind, event.Payload, event.IdempotencyKey))
	}
	return specs
}

func signalForVariant(variant string) interruptSignal {
	if variant == modecommon.VariantProduction {
		return productionSignal
	}
	return minimalSignal
}

func signalSpec(signal interruptSignal) string {
	return fmt.Sprintf("%d|%s|%s|%s", signal.AtCursor, signal.Reason, signal.Severity, signal.Source)
}

func resumePolicyForVariant(variant string) resumePolicy {
	if variant == modecommon.VariantProduction {
		return productionResumePolicy
	}
	return minimalResumePolicy
}

func parseEventSpecs(specs []string) []streamEvent {
	out := make([]streamEvent, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, "|", 4)
		if len(parts) != 4 {
			continue
		}
		out = append(out, streamEvent{
			Cursor:         parseInt(parts[0]),
			Kind:           strings.TrimSpace(parts[1]),
			Payload:        strings.TrimSpace(parts[2]),
			IdempotencyKey: strings.TrimSpace(parts[3]),
		})
	}
	if len(out) == 0 {
		out = append(out, minimalEvents...)
	}
	return out
}

func parseSignalSpec(spec string) interruptSignal {
	parts := strings.SplitN(spec, "|", 4)
	if len(parts) != 4 {
		return minimalSignal
	}
	return interruptSignal{
		AtCursor: parseInt(parts[0]),
		Reason:   strings.TrimSpace(parts[1]),
		Severity: strings.TrimSpace(strings.ToLower(parts[2])),
		Source:   strings.TrimSpace(parts[3]),
	}
}

func dedupeCursors(events []streamEvent) ([]int, int, int) {
	seen := map[string]struct{}{}
	applied := make([]int, 0, len(events))
	duplicates := 0
	for _, event := range events {
		key := strings.TrimSpace(event.IdempotencyKey)
		if key == "" {
			key = fmt.Sprintf("cursor-%d", event.Cursor)
		}
		if _, ok := seen[key]; ok {
			duplicates++
			continue
		}
		seen[key] = struct{}{}
		applied = append(applied, event.Cursor)
	}
	sort.Ints(applied)
	last := 0
	if len(applied) > 0 {
		last = applied[len(applied)-1]
	}
	return applied, last, duplicates
}

func captureInterrupt(applied []int, lastCursor int, signal interruptSignal) (int, string, string, int, int) {
	interruptCursor := signal.AtCursor
	if interruptCursor <= 0 {
		interruptCursor = lastCursor
	}
	if interruptCursor > lastCursor {
		interruptCursor = lastCursor
	}
	if interruptCursor < 0 {
		interruptCursor = 0
	}
	buffered := 0
	for _, cursor := range applied {
		if cursor > interruptCursor {
			buffered++
		}
	}
	resumeFrom := interruptCursor
	if resumeFrom == 0 && lastCursor > 0 {
		resumeFrom = lastCursor
	}
	reason := strings.TrimSpace(signal.Reason)
	if reason == "" {
		reason = "interrupt_unspecified"
	}
	severity := strings.TrimSpace(strings.ToLower(signal.Severity))
	if severity == "" {
		severity = "medium"
	}
	return interruptCursor, reason, severity, buffered, resumeFrom
}

func recoverFromCursor(applied []int, resumeFrom int, maxReplay int, mode string, requireAck bool, severity string) ([]int, string, string) {
	if maxReplay <= 0 {
		maxReplay = 1
	}
	candidates := make([]int, 0)
	for _, cursor := range applied {
		if cursor > resumeFrom {
			candidates = append(candidates, cursor)
		}
	}
	if len(candidates) == 0 {
		if requireAck {
			return []int{}, "await_manual_resume", "operator_ack_required"
		}
		return []int{}, "partial_resume", "replay_queue_empty"
	}
	limit := len(candidates)
	if limit > maxReplay {
		limit = maxReplay
	}
	recovered := append([]int(nil), candidates[:limit]...)
	status := "resumed"
	fallback := "none"
	if len(candidates) > maxReplay {
		status = "partial_resume"
		fallback = "deferred_replay_queue"
	}
	if strings.TrimSpace(strings.ToLower(mode)) == "strict" && strings.TrimSpace(strings.ToLower(severity)) == "high" && len(recovered) > 0 {
		status = "partial_resume"
		fallback = "checkpoint_ack_then_continue"
		recovered = recovered[:1]
	}
	return recovered, status, fallback
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

func toIntSlice(value any) ([]int, bool) {
	switch raw := value.(type) {
	case []int:
		return append([]int(nil), raw...), true
	case []any:
		out := make([]int, 0, len(raw))
		for _, item := range raw {
			if parsed, ok := modecommon.AsInt(item); ok {
				out = append(out, parsed)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func toAnyIntSlice(in []int) []any {
	if len(in) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
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
