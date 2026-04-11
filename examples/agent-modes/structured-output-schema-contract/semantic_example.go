package structuredoutputschemacontract

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
	patternName      = "structured-output-schema-contract"
	phase            = "P0"
	semanticAnchor   = "schema.validate_compat_drift"
	classification   = "structured_output.schema_contract"
	semanticToolName = "mode_structured_output_schema_contract_semantic_step"
)

type schemaStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type contractSchema struct {
	ID                string
	Version           int
	MinCompatible     int
	MaxCompatible     int
	RequiredFields    []string
	OptionalFields    []string
	AdditiveOnly      bool
	AllowUnknownField bool
}

type schemaPayload struct {
	Version int
	Fields  map[string]any
}

type schemaState struct {
	ContractID         string
	ContractVersion    int
	PayloadVersion     int
	Compatibility      string
	DriftClass         string
	UnknownFields      []string
	MissingFields      []string
	GovernanceDecision string
	ReplayBinding      string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"core/types", "runtime/diagnostics"}

var minimalSemanticSteps = []schemaStep{
	{
		Marker:        "schema_contract_loaded",
		RuntimeDomain: "core/types",
		Intent:        "load schema contract profile and required field set",
		Outcome:       "contract metadata and required fields are available",
	},
	{
		Marker:        "schema_compat_window_checked",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "evaluate payload version against compatibility window",
		Outcome:       "compatibility status is classified as in-window or out-of-window",
	},
	{
		Marker:        "schema_drift_signal_emitted",
		RuntimeDomain: "core/types",
		Intent:        "compute schema drift class from missing/unknown fields",
		Outcome:       "drift class and field-level diagnostics are emitted",
	},
}

var productionGovernanceSteps = []schemaStep{
	{
		Marker:        "governance_schema_gate_enforced",
		RuntimeDomain: "core/types",
		Intent:        "enforce governance admission for schema drift severity",
		Outcome:       "governance decision is persisted for blocking/warning policy",
	},
	{
		Marker:        "governance_schema_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind replay signature to contract version and drift decision",
		Outcome:       "replay binding signature is generated",
	},
}

var activeSchema = contractSchema{
	ID:                "order-response.v3",
	Version:           3,
	MinCompatible:     2,
	MaxCompatible:     3,
	RequiredFields:    []string{"order_id", "status", "total_amount"},
	OptionalFields:    []string{"currency", "discount_amount", "trace_id"},
	AdditiveOnly:      true,
	AllowUnknownField: true,
}

var minimalPayload = schemaPayload{
	Version: 3,
	Fields: map[string]any{
		"order_id":     "SO-2026-0001",
		"status":       "paid",
		"total_amount": 198,
		"currency":     "USD",
	},
}

var productionPayload = schemaPayload{
	Version: 3,
	Fields: map[string]any{
		"order_id":           "SO-2026-0009",
		"status":             "paid",
		"total_amount":       298,
		"currency":           "USD",
		"discount_amount":    20,
		"trace_id":           "trace-structured-2026",
		"governance_context": "risk_reviewed",
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
	if _, err := reg.Register(&schemaSemanticTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(newSchemaWorkflowModel(variant), runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute structured output schema contract workflow",
	}, nil)
	if err != nil {
		panic(err)
	}

	expected := expectedMarkers(variant)
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

func planSteps(variant string) []schemaStep {
	steps := make([]schemaStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

type schemaWorkflowModel struct {
	variant string
	stage   int
	state   schemaState
}

func newSchemaWorkflowModel(variant string) *schemaWorkflowModel {
	payload := payloadForVariant(variant)
	return &schemaWorkflowModel{
		variant: variant,
		state: schemaState{
			PayloadVersion: payload.Version,
		},
	}
}

func (m *schemaWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.consumeOutcomes(req.ToolResult)

	steps := planSteps(m.variant)
	if m.stage < len(steps) {
		step := steps[m.stage]
		call := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.stage+1),
			Name:   "local." + semanticToolName,
			Args:   m.buildArgs(step, m.stage+1),
		}
		m.stage++
		return types.ModelResponse{ToolCalls: []types.ToolCall{call}}, nil
	}

	markers := append([]string(nil), m.state.SeenMarkers...)
	sort.Strings(markers)
	unknown := append([]string(nil), m.state.UnknownFields...)
	sort.Strings(unknown)
	missing := append([]string(nil), m.state.MissingFields...)
	sort.Strings(missing)

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s contract=%s payload_version=%d compatibility=%s drift=%s governance=%s unknown=%s missing=%s replay=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.ContractID,
		m.state.PayloadVersion,
		m.state.Compatibility,
		m.state.DriftClass,
		normalizedDecision(m.state.GovernanceDecision),
		strings.Join(unknown, ","),
		strings.Join(missing, ","),
		m.state.ReplayBinding,
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *schemaWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *schemaWorkflowModel) consumeOutcomes(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if contractID, ok := item.Result.Structured["contract_id"].(string); ok && strings.TrimSpace(contractID) != "" {
			m.state.ContractID = contractID
		}
		if contractVersion, ok := modecommon.AsInt(item.Result.Structured["contract_version"]); ok {
			m.state.ContractVersion = contractVersion
		}
		if compatibility, ok := item.Result.Structured["compatibility"].(string); ok && strings.TrimSpace(compatibility) != "" {
			m.state.Compatibility = compatibility
		}
		if driftClass, ok := item.Result.Structured["drift_class"].(string); ok && strings.TrimSpace(driftClass) != "" {
			m.state.DriftClass = driftClass
		}
		if unknown, ok := toStringSlice(item.Result.Structured["unknown_fields"]); ok {
			m.state.UnknownFields = unknown
		}
		if missing, ok := toStringSlice(item.Result.Structured["missing_fields"]); ok {
			m.state.MissingFields = missing
		}
		if decision, ok := item.Result.Structured["governance_decision"].(string); ok && strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = decision
		}
		if replayBinding, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replayBinding) != "" {
			m.state.ReplayBinding = replayBinding
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *schemaWorkflowModel) buildArgs(step schemaStep, stage int) map[string]any {
	schema := activeSchema
	payload := payloadForVariant(m.variant)
	args := map[string]any{
		"pattern":            patternName,
		"variant":            m.variant,
		"phase":              phase,
		"semantic_anchor":    semanticAnchor,
		"classification":     classification,
		"marker":             step.Marker,
		"runtime_domain":     step.RuntimeDomain,
		"semantic_intent":    step.Intent,
		"semantic_outcome":   step.Outcome,
		"contract_id":        schema.ID,
		"contract_version":   schema.Version,
		"min_compatible":     schema.MinCompatible,
		"max_compatible":     schema.MaxCompatible,
		"required_fields":    toAnySlice(schema.RequiredFields),
		"optional_fields":    toAnySlice(schema.OptionalFields),
		"payload_version":    payload.Version,
		"payload_field_keys": toAnySlice(sortedKeys(payload.Fields)),
		"stage":              stage,
	}

	switch step.Marker {
	case "schema_compat_window_checked":
		args["previous_contract_id"] = m.state.ContractID
		args["previous_contract_version"] = m.state.ContractVersion
	case "schema_drift_signal_emitted":
		args["compatibility"] = m.state.Compatibility
	case "governance_schema_gate_enforced":
		args["compatibility"] = m.state.Compatibility
		args["drift_class"] = m.state.DriftClass
	case "governance_schema_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["drift_class"] = m.state.DriftClass
		args["compatibility"] = m.state.Compatibility
	}
	return args
}

type schemaSemanticTool struct{}

func (t *schemaSemanticTool) Name() string { return semanticToolName }

func (t *schemaSemanticTool) Description() string {
	return "execute structured output schema semantic step"
}

func (t *schemaSemanticTool) JSONSchema() map[string]any {
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
			"contract_id",
			"contract_version",
			"payload_version",
			"stage",
		},
		"properties": map[string]any{
			"pattern":            map[string]any{"type": "string"},
			"variant":            map[string]any{"type": "string"},
			"phase":              map[string]any{"type": "string"},
			"semantic_anchor":    map[string]any{"type": "string"},
			"classification":     map[string]any{"type": "string"},
			"marker":             map[string]any{"type": "string"},
			"runtime_domain":     map[string]any{"type": "string"},
			"semantic_intent":    map[string]any{"type": "string"},
			"semantic_outcome":   map[string]any{"type": "string"},
			"contract_id":        map[string]any{"type": "string"},
			"contract_version":   map[string]any{"type": "integer"},
			"min_compatible":     map[string]any{"type": "integer"},
			"max_compatible":     map[string]any{"type": "integer"},
			"required_fields":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"optional_fields":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"payload_version":    map[string]any{"type": "integer"},
			"payload_field_keys": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"compatibility":      map[string]any{"type": "string"},
			"drift_class":        map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"stage": map[string]any{"type": "integer"},
		},
	}
}

func (t *schemaSemanticTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	contractID := strings.TrimSpace(fmt.Sprintf("%v", args["contract_id"]))
	contractVersion, _ := modecommon.AsInt(args["contract_version"])
	payloadVersion, _ := modecommon.AsInt(args["payload_version"])
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	requiredFields, _ := toStringSlice(args["required_fields"])
	optionalFields, _ := toStringSlice(args["optional_fields"])
	payloadFieldKeys, _ := toStringSlice(args["payload_field_keys"])

	result := map[string]any{
		"pattern":          pattern,
		"variant":          variant,
		"phase":            phaseValue,
		"semantic_anchor":  anchor,
		"classification":   classValue,
		"marker":           marker,
		"runtime_domain":   runtimeDomain,
		"semantic_intent":  intent,
		"semantic_outcome": outcome,
		"contract_id":      contractID,
		"contract_version": contractVersion,
		"payload_version":  payloadVersion,
		"stage":            stage,
		"governance":       strings.HasPrefix(marker, "governance_"),
	}

	risk := "nominal"
	compatibility := "in_window"
	driftClass := "none"
	governanceDecision := "n/a"
	missing := diffRequiredFields(requiredFields, payloadFieldKeys)
	unknown := diffUnknownFields(payloadFieldKeys, requiredFields, optionalFields)

	switch marker {
	case "schema_contract_loaded":
		result["schema_field_budget"] = len(requiredFields) + len(optionalFields)
		risk = "contract_loaded"
	case "schema_compat_window_checked":
		minCompatible, _ := modecommon.AsInt(args["min_compatible"])
		maxCompatible, _ := modecommon.AsInt(args["max_compatible"])
		if payloadVersion < minCompatible || payloadVersion > maxCompatible {
			compatibility = "out_of_window"
			risk = "compatibility_risk"
		}
		result["compatibility"] = compatibility
		result["min_compatible"] = minCompatible
		result["max_compatible"] = maxCompatible
	case "schema_drift_signal_emitted":
		if compatibilityArg, ok := args["compatibility"].(string); ok && strings.TrimSpace(compatibilityArg) != "" {
			compatibility = strings.TrimSpace(compatibilityArg)
		}
		if compatibility == "out_of_window" || len(missing) > 0 {
			driftClass = "breaking"
			risk = "breaking_drift"
		} else if len(unknown) > 0 {
			driftClass = "additive"
			risk = "additive_drift"
		}
		result["compatibility"] = compatibility
		result["drift_class"] = driftClass
		result["missing_fields"] = toAnySlice(missing)
		result["unknown_fields"] = toAnySlice(unknown)
	case "governance_schema_gate_enforced":
		compatibility = safeString(args["compatibility"], compatibility)
		driftClass = safeString(args["drift_class"], driftClass)
		switch driftClass {
		case "breaking":
			governanceDecision = "block"
			risk = "governed_block"
		case "additive":
			governanceDecision = "warn_and_record"
			risk = "governed_warn"
		default:
			governanceDecision = "allow"
			risk = "governed_allow"
		}
		result["compatibility"] = compatibility
		result["drift_class"] = driftClass
		result["governance_decision"] = governanceDecision
		result["governance"] = true
	case "governance_schema_replay_bound":
		compatibility = safeString(args["compatibility"], compatibility)
		driftClass = safeString(args["drift_class"], driftClass)
		governanceDecision = safeString(args["governance_decision"], governanceDecision)
		replayBinding := fmt.Sprintf(
			"schema-replay-%d",
			modecommon.SemanticScore(pattern, variant, contractID, fmt.Sprintf("%d", contractVersion), driftClass, governanceDecision),
		)
		result["compatibility"] = compatibility
		result["drift_class"] = driftClass
		result["governance_decision"] = governanceDecision
		result["replay_binding"] = replayBinding
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported structured schema marker: %s", marker)
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
		compatibility,
		driftClass,
		governanceDecision,
	)
	result["risk"] = risk

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s compatibility=%s drift=%s governance=%s risk=%s",
		pattern,
		variant,
		marker,
		safeString(result["compatibility"], compatibility),
		safeString(result["drift_class"], driftClass),
		safeString(result["governance_decision"], governanceDecision),
		risk,
	)

	return types.ToolResult{Content: content, Structured: result}, nil
}

func payloadForVariant(variant string) schemaPayload {
	if variant == modecommon.VariantProduction {
		return schemaPayload{Version: productionPayload.Version, Fields: cloneMap(productionPayload.Fields)}
	}
	return schemaPayload{Version: minimalPayload.Version, Fields: cloneMap(minimalPayload.Fields)}
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sortedKeys(in map[string]any) []string {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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

func diffRequiredFields(required []string, actual []string) []string {
	actualSet := make(map[string]struct{}, len(actual))
	for _, field := range actual {
		actualSet[field] = struct{}{}
	}
	missing := make([]string, 0)
	for _, field := range required {
		if _, ok := actualSet[field]; !ok {
			missing = append(missing, field)
		}
	}
	sort.Strings(missing)
	return missing
}

func diffUnknownFields(actual []string, required []string, optional []string) []string {
	known := make(map[string]struct{}, len(required)+len(optional))
	for _, field := range required {
		known[field] = struct{}{}
	}
	for _, field := range optional {
		known[field] = struct{}{}
	}
	unknown := make([]string, 0)
	for _, field := range actual {
		if _, ok := known[field]; ok {
			continue
		}
		unknown = append(unknown, field)
	}
	sort.Strings(unknown)
	return unknown
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
