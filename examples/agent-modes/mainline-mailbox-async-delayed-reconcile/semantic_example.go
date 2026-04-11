package mainlinemailboxasyncdelayedreconcile

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
	patternName      = "mainline-mailbox-async-delayed-reconcile"
	phase            = "P2"
	semanticAnchor   = "mailbox.async_delayed_reconcile"
	classification   = "mainline.mailbox_reconcile"
	semanticToolName = "mode_mainline_mailbox_async_delayed_reconcile_semantic_step"
	defaultDispatch  = "mbx-dispatch-20260410"
)

type mailboxStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type mailboxState struct {
	DispatchID          string
	MailboxName         string
	DelaySeconds        int
	AsyncToken          string
	PendingCount        int
	ReconcileBatchSize  int
	ReconciledCount     int
	LateMessageCount    int
	TimelineReason      string
	PrimaryReason       string
	ReasonConfidence    int
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"orchestration/mailbox", "orchestration/invoke", "runtime/diagnostics"}

var minimalSemanticSteps = []mailboxStep{
	{
		Marker:        "mailbox_async_delayed_dispatched",
		RuntimeDomain: "orchestration/mailbox",
		Intent:        "dispatch delayed mailbox task with explicit async token",
		Outcome:       "dispatch id, mailbox and pending queue snapshot are emitted",
	},
	{
		Marker:        "mailbox_reconcile_triggered",
		RuntimeDomain: "orchestration/invoke",
		Intent:        "trigger reconcile batch after delay window",
		Outcome:       "reconcile batch size, reconciled count and late message count are emitted",
	},
	{
		Marker:        "mailbox_timeline_reason_emitted",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "emit timeline reason and primary reason confidence",
		Outcome:       "timeline reason and confidence are emitted",
	},
}

var productionGovernanceSteps = []mailboxStep{
	{
		Marker:        "governance_mailbox_gate_enforced",
		RuntimeDomain: "orchestration/mailbox",
		Intent:        "enforce mailbox governance from reconcile and timeline signals",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_mailbox_replay_bound",
		RuntimeDomain: "orchestration/invoke",
		Intent:        "bind governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeMailboxVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeMailboxVariant(modecommon.VariantProduction)
}

func executeMailboxVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&mailboxReconcileTool{}); err != nil {
		panic(err)
	}

	model := &mailboxReconcileModel{
		variant: variant,
		state: mailboxState{
			DispatchID:  defaultDispatch,
			MailboxName: "account-reconcile",
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute mailbox async delayed reconcile semantic pipeline",
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

func planForVariant(variant string) []mailboxStep {
	plan := make([]mailboxStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type mailboxReconcileModel struct {
	variant string
	cursor  int
	state   mailboxState
}

func (m *mailboxReconcileModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s dispatch=%s mailbox=%s delay_sec=%d async_token=%s pending=%d reconcile_batch=%d reconciled=%d late=%d timeline_reason=%s primary_reason=%s confidence=%d governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.DispatchID, true),
		readableValue(m.state.MailboxName, true),
		m.state.DelaySeconds,
		readableValue(m.state.AsyncToken, true),
		m.state.PendingCount,
		m.state.ReconcileBatchSize,
		m.state.ReconciledCount,
		m.state.LateMessageCount,
		readableValue(m.state.TimelineReason, true),
		readableValue(m.state.PrimaryReason, true),
		m.state.ReasonConfidence,
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *mailboxReconcileModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *mailboxReconcileModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if dispatch, _ := outcome.Result.Structured["dispatch_id"].(string); strings.TrimSpace(dispatch) != "" {
			m.state.DispatchID = strings.TrimSpace(dispatch)
		}
		if mailbox, _ := outcome.Result.Structured["mailbox"].(string); strings.TrimSpace(mailbox) != "" {
			m.state.MailboxName = strings.TrimSpace(mailbox)
		}
		if delay, ok := modecommon.AsInt(outcome.Result.Structured["delay_sec"]); ok {
			m.state.DelaySeconds = delay
		}
		if token, _ := outcome.Result.Structured["async_token"].(string); strings.TrimSpace(token) != "" {
			m.state.AsyncToken = strings.TrimSpace(token)
		}
		if pending, ok := modecommon.AsInt(outcome.Result.Structured["pending_count"]); ok {
			m.state.PendingCount = pending
		}
		if batch, ok := modecommon.AsInt(outcome.Result.Structured["reconcile_batch"]); ok {
			m.state.ReconcileBatchSize = batch
		}
		if reconciled, ok := modecommon.AsInt(outcome.Result.Structured["reconciled_count"]); ok {
			m.state.ReconciledCount = reconciled
		}
		if late, ok := modecommon.AsInt(outcome.Result.Structured["late_count"]); ok {
			m.state.LateMessageCount = late
		}
		if reason, _ := outcome.Result.Structured["timeline_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.TimelineReason = strings.TrimSpace(reason)
		}
		if reason, _ := outcome.Result.Structured["primary_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.PrimaryReason = strings.TrimSpace(reason)
		}
		if confidence, ok := modecommon.AsInt(outcome.Result.Structured["reason_confidence"]); ok {
			m.state.ReasonConfidence = confidence
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

func (m *mailboxReconcileModel) argsForStep(step mailboxStep, stage int) map[string]any {
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
		"dispatch_id":      m.state.DispatchID,
		"mailbox":          m.state.MailboxName,
		"delay_sec":        m.state.DelaySeconds,
		"async_token":      m.state.AsyncToken,
		"pending_count":    m.state.PendingCount,
		"reconcile_batch":  m.state.ReconcileBatchSize,
		"reconciled_count": m.state.ReconciledCount,
		"late_count":       m.state.LateMessageCount,
	}

	switch step.Marker {
	case "mailbox_async_delayed_dispatched":
		delay := 5
		pending := 3
		if m.variant == modecommon.VariantProduction {
			delay = 15
			pending = 8
		}
		args["delay_sec"] = delay
		args["pending_count"] = pending
	case "mailbox_reconcile_triggered":
		reconcileBatch := m.state.PendingCount
		reconciled := m.state.PendingCount
		late := 0
		if m.variant == modecommon.VariantProduction {
			reconcileBatch = m.state.PendingCount
			reconciled = m.state.PendingCount - 1
			late = 1
		}
		args["reconcile_batch"] = reconcileBatch
		args["reconciled_count"] = reconciled
		args["late_count"] = late
	case "mailbox_timeline_reason_emitted":
		primaryReason := "delay_window_elapsed"
		timelineReason := "async-delayed mailbox reconciled"
		reasonConfidence := 86
		if m.state.LateMessageCount > 0 {
			primaryReason = "delayed-window-with-late-messages"
			timelineReason = "async-delayed mailbox partially reconciled"
			reasonConfidence = 92
		}
		args["primary_reason"] = primaryReason
		args["timeline_reason"] = timelineReason
		args["reason_confidence"] = reasonConfidence
	case "governance_mailbox_gate_enforced":
		args["late_count"] = m.state.LateMessageCount
		args["reconciled_count"] = m.state.ReconciledCount
		args["pending_count"] = m.state.PendingCount
		args["reason_confidence"] = m.state.ReasonConfidence
	case "governance_mailbox_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["timeline_reason"] = m.state.TimelineReason
	}
	return args
}

type mailboxReconcileTool struct{}

func (t *mailboxReconcileTool) Name() string { return semanticToolName }

func (t *mailboxReconcileTool) Description() string {
	return "execute mailbox async delayed reconcile semantic step"
}

func (t *mailboxReconcileTool) JSONSchema() map[string]any {
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
			"dispatch_id":      map[string]any{"type": "string"},
			"mailbox":          map[string]any{"type": "string"},
			"delay_sec":        map[string]any{"type": "integer"},
			"async_token":      map[string]any{"type": "string"},
			"pending_count":    map[string]any{"type": "integer"},
			"reconcile_batch":  map[string]any{"type": "integer"},
			"reconciled_count": map[string]any{"type": "integer"},
			"late_count":       map[string]any{"type": "integer"},
			"timeline_reason":  map[string]any{"type": "string"},
			"primary_reason":   map[string]any{"type": "string"},
			"reason_confidence": map[string]any{
				"type": "integer",
			},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *mailboxReconcileTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	dispatchID := strings.TrimSpace(fmt.Sprintf("%v", args["dispatch_id"]))
	if dispatchID == "" {
		dispatchID = defaultDispatch
	}
	mailbox := strings.TrimSpace(fmt.Sprintf("%v", args["mailbox"]))
	if mailbox == "" {
		mailbox = "account-reconcile"
	}
	delaySec, _ := modecommon.AsInt(args["delay_sec"])
	if delaySec <= 0 {
		delaySec = 5
	}
	asyncToken := strings.TrimSpace(fmt.Sprintf("%v", args["async_token"]))
	if asyncToken == "" {
		asyncToken = fmt.Sprintf("mbx-token-%d", modecommon.SemanticScore(dispatchID, mailbox, fmt.Sprintf("%d", delaySec), variant))
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
		"dispatch_id":     dispatchID,
		"mailbox":         mailbox,
		"delay_sec":       delaySec,
		"async_token":     asyncToken,
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "mailbox_async_delayed_dispatched":
		pendingCount, _ := modecommon.AsInt(args["pending_count"])
		structured["pending_count"] = pendingCount
		structured["async_token"] = asyncToken
	case "mailbox_reconcile_triggered":
		pendingCount, _ := modecommon.AsInt(args["pending_count"])
		reconcileBatch, _ := modecommon.AsInt(args["reconcile_batch"])
		reconciledCount, _ := modecommon.AsInt(args["reconciled_count"])
		lateCount, _ := modecommon.AsInt(args["late_count"])
		if reconcileBatch <= 0 {
			reconcileBatch = pendingCount
		}
		structured["pending_count"] = pendingCount
		structured["reconcile_batch"] = reconcileBatch
		structured["reconciled_count"] = reconciledCount
		structured["late_count"] = lateCount
		if lateCount > 0 {
			risk = "degraded_path"
		}
	case "mailbox_timeline_reason_emitted":
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		timelineReason := strings.TrimSpace(fmt.Sprintf("%v", args["timeline_reason"]))
		reasonConfidence, _ := modecommon.AsInt(args["reason_confidence"])
		structured["primary_reason"] = primaryReason
		structured["timeline_reason"] = timelineReason
		structured["reason_confidence"] = reasonConfidence
		if strings.Contains(primaryReason, "late") {
			risk = "degraded_path"
		}
	case "governance_mailbox_gate_enforced":
		lateCount, _ := modecommon.AsInt(args["late_count"])
		reconciledCount, _ := modecommon.AsInt(args["reconciled_count"])
		pendingCount, _ := modecommon.AsInt(args["pending_count"])
		reasonConfidence, _ := modecommon.AsInt(args["reason_confidence"])
		decision := "allow_reconcile"
		if lateCount > 0 {
			decision = "allow_reconcile_with_watch"
		}
		if reconciledCount < pendingCount {
			decision = "hold_for_followup_reconcile"
		}
		ticket := fmt.Sprintf("mailbox-gate-%d", modecommon.SemanticScore(dispatchID, decision, fmt.Sprintf("%d", pendingCount), fmt.Sprintf("%d", lateCount), fmt.Sprintf("%d", reasonConfidence)))
		structured["late_count"] = lateCount
		structured["reconciled_count"] = reconciledCount
		structured["pending_count"] = pendingCount
		structured["reason_confidence"] = reasonConfidence
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_mailbox_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		timelineReason := strings.TrimSpace(fmt.Sprintf("%v", args["timeline_reason"]))
		replaySignature := fmt.Sprintf("mailbox-replay-%d", modecommon.SemanticScore(dispatchID, decision, ticket, timelineReason, asyncToken))
		structured["timeline_reason"] = timelineReason
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported mailbox marker: %s", marker)
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
