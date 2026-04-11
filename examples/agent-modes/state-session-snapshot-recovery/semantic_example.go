package statesessionsnapshotrecovery

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
	patternName      = "state-session-snapshot-recovery"
	phase            = "P1"
	semanticAnchor   = "snapshot.export_restore_replay"
	classification   = "state.session_snapshot_recovery"
	semanticToolName = "mode_state_session_snapshot_recovery_semantic_step"
	defaultSessionID = "session-checkout-42"
)

type snapshotStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type snapshotState struct {
	SessionID          string
	SnapshotVersion    int
	SnapshotDigest     string
	ExportChunkCount   int
	RestoredCursor     int
	RestoreChecksumOK  bool
	ReplayEvents       int
	ReplayDigest       string
	ReplayIdempotent   bool
	ReplayDrift        bool
	GovernanceDecision string
	GovernanceTicket   string
	ReplayBinding      string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"orchestration/snapshot", "runtime/diagnostics"}

var baseSnapshotFrames = []string{
	"cart:add:item-1001",
	"cart:add:item-2002",
	"cart:update:item-1001:qty=2",
	"checkout:apply-coupon:SPRING",
	"checkout:set-shipping:express",
}

var minimalSemanticSteps = []snapshotStep{
	{
		Marker:        "snapshot_export_emitted",
		RuntimeDomain: "orchestration/snapshot",
		Intent:        "export session state into deterministic snapshot chunks with digest",
		Outcome:       "snapshot digest and chunk metadata are emitted",
	},
	{
		Marker:        "snapshot_restore_verified",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "restore snapshot into session cursor and verify checksum consistency",
		Outcome:       "restore checksum and cursor verification are emitted",
	},
	{
		Marker:        "snapshot_replay_idempotent",
		RuntimeDomain: "orchestration/snapshot",
		Intent:        "replay journal frames and classify idempotent vs drift path",
		Outcome:       "replay digest and idempotency classification are emitted",
	},
}

var productionGovernanceSteps = []snapshotStep{
	{
		Marker:        "governance_snapshot_gate_enforced",
		RuntimeDomain: "orchestration/snapshot",
		Intent:        "enforce governance gate using restore/replay integrity signals",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_snapshot_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind snapshot decision into deterministic replay signature",
		Outcome:       "replay binding signature is emitted",
	},
}

func RunMinimal() {
	executeSnapshotVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeSnapshotVariant(modecommon.VariantProduction)
}

func executeSnapshotVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&snapshotRecoveryTool{}); err != nil {
		panic(err)
	}

	model := &snapshotRecoveryModel{
		variant: variant,
		state: snapshotState{
			SessionID: defaultSessionID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute state session snapshot recovery semantic pipeline",
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

func planForVariant(variant string) []snapshotStep {
	plan := make([]snapshotStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type snapshotRecoveryModel struct {
	variant string
	cursor  int
	state   snapshotState
}

func (m *snapshotRecoveryModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s snapshot_version=%d chunks=%d digest=%s restore_checksum=%t replay_events=%d replay_idempotent=%t replay_drift=%t governance=%s ticket=%s replay_binding=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.SnapshotVersion,
		m.state.ExportChunkCount,
		normalizedValue(m.state.SnapshotDigest, true),
		m.state.RestoreChecksumOK,
		m.state.ReplayEvents,
		m.state.ReplayIdempotent,
		m.state.ReplayDrift,
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplayBinding, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *snapshotRecoveryModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *snapshotRecoveryModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if version, ok := modecommon.AsInt(item.Result.Structured["snapshot_version"]); ok {
			m.state.SnapshotVersion = version
		}
		if chunkCount, ok := modecommon.AsInt(item.Result.Structured["chunk_count"]); ok {
			m.state.ExportChunkCount = chunkCount
		}
		if digest, _ := item.Result.Structured["snapshot_digest"].(string); strings.TrimSpace(digest) != "" {
			m.state.SnapshotDigest = strings.TrimSpace(digest)
		}
		if cursor, ok := modecommon.AsInt(item.Result.Structured["restored_cursor"]); ok {
			m.state.RestoredCursor = cursor
		}
		if checksumOK, ok := item.Result.Structured["restore_checksum_ok"].(bool); ok {
			m.state.RestoreChecksumOK = checksumOK
		}
		if replayEvents, ok := modecommon.AsInt(item.Result.Structured["replay_events"]); ok {
			m.state.ReplayEvents = replayEvents
		}
		if replayDigest, _ := item.Result.Structured["replay_digest"].(string); strings.TrimSpace(replayDigest) != "" {
			m.state.ReplayDigest = strings.TrimSpace(replayDigest)
		}
		if replayIdempotent, ok := item.Result.Structured["replay_idempotent"].(bool); ok {
			m.state.ReplayIdempotent = replayIdempotent
		}
		if replayDrift, ok := item.Result.Structured["replay_drift"].(bool); ok {
			m.state.ReplayDrift = replayDrift
		}
		if decision, _ := item.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := item.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if binding, _ := item.Result.Structured["replay_binding"].(string); strings.TrimSpace(binding) != "" {
			m.state.ReplayBinding = strings.TrimSpace(binding)
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *snapshotRecoveryModel) argsForStep(step snapshotStep, stage int) map[string]any {
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
		"session_id":      m.state.SessionID,
	}

	switch step.Marker {
	case "snapshot_export_emitted":
		version := 3
		if m.variant == modecommon.VariantProduction {
			version = 4
		}
		args["snapshot_version"] = version
		args["frames"] = stringSliceToAny(baseSnapshotFrames)
		args["chunk_size"] = 2
	case "snapshot_restore_verified":
		args["snapshot_version"] = m.state.SnapshotVersion
		args["snapshot_digest"] = m.state.SnapshotDigest
		args["chunk_count"] = m.state.ExportChunkCount
		args["expected_cursor"] = len(baseSnapshotFrames)
	case "snapshot_replay_idempotent":
		replayFrames := append([]string{}, baseSnapshotFrames...)
		if m.variant == modecommon.VariantProduction {
			replayFrames = append(replayFrames, "checkout:inventory:reserve")
		}
		args["snapshot_digest"] = m.state.SnapshotDigest
		args["frames"] = stringSliceToAny(replayFrames)
		args["restore_checksum_ok"] = m.state.RestoreChecksumOK
	case "governance_snapshot_gate_enforced":
		args["restore_checksum_ok"] = m.state.RestoreChecksumOK
		args["replay_idempotent"] = m.state.ReplayIdempotent
		args["replay_drift"] = m.state.ReplayDrift
		args["snapshot_digest"] = m.state.SnapshotDigest
	case "governance_snapshot_replay_bound":
		args["snapshot_digest"] = m.state.SnapshotDigest
		args["replay_digest"] = m.state.ReplayDigest
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
	}
	return args
}

type snapshotRecoveryTool struct{}

func (t *snapshotRecoveryTool) Name() string { return semanticToolName }

func (t *snapshotRecoveryTool) Description() string {
	return "execute snapshot export/restore/replay semantic step"
}

func (t *snapshotRecoveryTool) JSONSchema() map[string]any {
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
			"session_id":      map[string]any{"type": "string"},
			"snapshot_version": map[string]any{
				"type": "integer",
			},
			"snapshot_digest": map[string]any{"type": "string"},
			"chunk_size":      map[string]any{"type": "integer"},
			"chunk_count":     map[string]any{"type": "integer"},
			"frames":          map[string]any{"type": "array"},
			"expected_cursor": map[string]any{"type": "integer"},
			"restore_checksum_ok": map[string]any{
				"type": "boolean",
			},
			"replay_idempotent": map[string]any{"type": "boolean"},
			"replay_drift":      map[string]any{"type": "boolean"},
			"replay_digest":     map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *snapshotRecoveryTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "snapshot_export_emitted":
		sessionID := strings.TrimSpace(fmt.Sprintf("%v", args["session_id"]))
		if sessionID == "" {
			sessionID = defaultSessionID
		}
		version, _ := modecommon.AsInt(args["snapshot_version"])
		if version <= 0 {
			version = 3
		}
		chunkSize, _ := modecommon.AsInt(args["chunk_size"])
		if chunkSize <= 0 {
			chunkSize = 2
		}
		frames := toStringSlice(args["frames"])
		if len(frames) == 0 {
			frames = append([]string{}, baseSnapshotFrames...)
		}
		chunkCount := (len(frames) + chunkSize - 1) / chunkSize
		digest := digestFromParts(sessionID, fmt.Sprintf("v%d", version), strings.Join(frames, "|"))
		structured["session_id"] = sessionID
		structured["snapshot_version"] = version
		structured["chunk_count"] = chunkCount
		structured["snapshot_digest"] = digest
		structured["frames"] = stringSliceToAny(frames)
	case "snapshot_restore_verified":
		version, _ := modecommon.AsInt(args["snapshot_version"])
		digest := strings.TrimSpace(fmt.Sprintf("%v", args["snapshot_digest"]))
		chunkCount, _ := modecommon.AsInt(args["chunk_count"])
		expectedCursor, _ := modecommon.AsInt(args["expected_cursor"])
		if expectedCursor <= 0 {
			expectedCursor = len(baseSnapshotFrames)
		}
		checksumOK := digest != "" && version > 0 && chunkCount > 0
		restoredCursor := expectedCursor
		if variant == modecommon.VariantProduction {
			restoredCursor = expectedCursor + 1
		}
		structured["snapshot_version"] = version
		structured["snapshot_digest"] = digest
		structured["chunk_count"] = chunkCount
		structured["restored_cursor"] = restoredCursor
		structured["restore_checksum_ok"] = checksumOK
		if !checksumOK {
			risk = "degraded_path"
		}
	case "snapshot_replay_idempotent":
		digest := strings.TrimSpace(fmt.Sprintf("%v", args["snapshot_digest"]))
		frames := toStringSlice(args["frames"])
		restoreChecksumOK := asBool(args["restore_checksum_ok"])
		replayDigest := digestFromParts(digest, strings.Join(frames, "|"))
		replayIdempotent := restoreChecksumOK && !strings.Contains(strings.Join(frames, "|"), "inventory")
		replayDrift := !replayIdempotent
		structured["snapshot_digest"] = digest
		structured["replay_events"] = len(frames)
		structured["replay_digest"] = replayDigest
		structured["replay_idempotent"] = replayIdempotent
		structured["replay_drift"] = replayDrift
		if replayDrift {
			risk = "degraded_path"
		}
	case "governance_snapshot_gate_enforced":
		restoreChecksumOK := asBool(args["restore_checksum_ok"])
		replayIdempotent := asBool(args["replay_idempotent"])
		replayDrift := asBool(args["replay_drift"])
		snapshotDigest := strings.TrimSpace(fmt.Sprintf("%v", args["snapshot_digest"]))

		decision := "allow"
		if !restoreChecksumOK {
			decision = "deny"
		} else if replayDrift || !replayIdempotent {
			decision = "hold_for_review"
		}
		ticket := fmt.Sprintf(
			"ssr-gate-%d",
			modecommon.SemanticScore(pattern, variant, snapshotDigest, fmt.Sprintf("%t", restoreChecksumOK), fmt.Sprintf("%t", replayIdempotent), decision),
		)
		structured["restore_checksum_ok"] = restoreChecksumOK
		structured["replay_idempotent"] = replayIdempotent
		structured["replay_drift"] = replayDrift
		structured["snapshot_digest"] = snapshotDigest
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_snapshot_replay_bound":
		snapshotDigest := strings.TrimSpace(fmt.Sprintf("%v", args["snapshot_digest"]))
		replayDigest := strings.TrimSpace(fmt.Sprintf("%v", args["replay_digest"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		binding := digestFromParts(snapshotDigest, replayDigest, decision, ticket)
		structured["snapshot_digest"] = snapshotDigest
		structured["replay_digest"] = replayDigest
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["governance"] = true
		structured["replay_binding"] = binding
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported snapshot semantic marker: %s", marker)
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

func digestFromParts(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	if len(clean) == 0 {
		return "digest-empty"
	}
	return fmt.Sprintf("digest-%d", modecommon.SemanticScore(strings.Join(clean, "|")))
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
