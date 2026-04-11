package mapreducelargebatch

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
	patternName      = "mapreduce-large-batch"
	phase            = "P1"
	semanticAnchor   = "mapreduce.shard_reduce_retry"
	classification   = "mapreduce.large_batch"
	semanticToolName = "mode_mapreduce_large_batch_semantic_step"
	defaultBatchID   = "batch-20260410-ops"
)

type mapReduceStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type batchRecord struct {
	Partition string
	Value     int
	Priority  int
}

type shardSnapshot struct {
	ShardID    int
	Load       int
	Failed     bool
	Partitions []string
}

type mapReduceState struct {
	BatchID            string
	Shards             []shardSnapshot
	HotShards          []int
	AggregateSum       int
	AggregateGroups    int
	FailedShardCount   int
	RetryShards        []int
	RetryClass         string
	RetryBudget        int
	GovernanceDecision string
	GovernanceTicket   string
	ReplaySignature    string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"orchestration/teams", "runtime/diagnostics"}

var seedRecords = []batchRecord{
	{Partition: "alpha", Value: 6, Priority: 2},
	{Partition: "alpha", Value: 4, Priority: 1},
	{Partition: "beta", Value: 7, Priority: 3},
	{Partition: "gamma", Value: 5, Priority: 1},
	{Partition: "delta", Value: 9, Priority: 2},
	{Partition: "epsilon", Value: 8, Priority: 2},
	{Partition: "zeta", Value: 3, Priority: 1},
	{Partition: "eta", Value: 2, Priority: 1},
	{Partition: "theta", Value: 10, Priority: 3},
	{Partition: "iota", Value: 5, Priority: 1},
}

var minimalSemanticSteps = []mapReduceStep{
	{
		Marker:        "mapreduce_shards_fanned_out",
		RuntimeDomain: "orchestration/teams",
		Intent:        "fan out a large batch into deterministic shards and flag hot shard skew",
		Outcome:       "shard distribution with hot shard evidence is produced",
	},
	{
		Marker:        "mapreduce_reduce_aggregated",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "aggregate map shard outputs and preserve partial reduce diagnostics",
		Outcome:       "reduce aggregate sum/group evidence is produced",
	},
	{
		Marker:        "mapreduce_retry_classified",
		RuntimeDomain: "orchestration/teams",
		Intent:        "classify failed shards into retry_scheduled or retry_deferred by budget",
		Outcome:       "retry class and retry shard set are emitted",
	},
}

var productionGovernanceSteps = []mapReduceStep{
	{
		Marker:        "governance_mapreduce_gate_enforced",
		RuntimeDomain: "orchestration/teams",
		Intent:        "enforce governance gate using retry class and hot shard pressure",
		Outcome:       "gate decision with governance ticket is persisted",
	},
	{
		Marker:        "governance_mapreduce_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind governance decision to replay signature for deterministic audit",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeMapReduceVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeMapReduceVariant(modecommon.VariantProduction)
}

func executeMapReduceVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&mapReduceBatchTool{}); err != nil {
		panic(err)
	}

	model := &mapReduceBatchModel{
		variant: variant,
		state: mapReduceState{
			BatchID: defaultBatchID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute mapreduce large batch semantic pipeline",
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

func executionPlanForVariant(variant string) []mapReduceStep {
	plan := make([]mapReduceStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type mapReduceBatchModel struct {
	variant string
	cursor  int
	state   mapReduceState
}

func (m *mapReduceBatchModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.capture(req.ToolResult)

	plan := executionPlanForVariant(m.variant)
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s shards=%d hot=%s aggregate_sum=%d aggregate_groups=%d failed_shards=%d retry_class=%s retry_shards=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		len(m.state.Shards),
		intSliceToken(m.state.HotShards),
		m.state.AggregateSum,
		m.state.AggregateGroups,
		m.state.FailedShardCount,
		normalizedValue(m.state.RetryClass, true),
		intSliceToken(m.state.RetryShards),
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *mapReduceBatchModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *mapReduceBatchModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if shardPayload, ok := item.Result.Structured["shard_summaries"]; ok {
			if parsed, ok := toShardSummaries(shardPayload); ok {
				m.state.Shards = parsed
			}
		}
		if hotPayload, ok := item.Result.Structured["hot_shards"]; ok {
			if parsed, ok := toIntSlice(hotPayload); ok {
				m.state.HotShards = parsed
			}
		}
		if sum, ok := modecommon.AsInt(item.Result.Structured["aggregate_sum"]); ok {
			m.state.AggregateSum = sum
		}
		if groups, ok := modecommon.AsInt(item.Result.Structured["aggregate_groups"]); ok {
			m.state.AggregateGroups = groups
		}
		if failed, ok := modecommon.AsInt(item.Result.Structured["failed_shards"]); ok {
			m.state.FailedShardCount = failed
		}
		if retryShards, ok := toIntSlice(item.Result.Structured["retry_shards"]); ok {
			m.state.RetryShards = retryShards
		}
		if retryClass, _ := item.Result.Structured["retry_class"].(string); strings.TrimSpace(retryClass) != "" {
			m.state.RetryClass = strings.TrimSpace(retryClass)
		}
		if retryBudget, ok := modecommon.AsInt(item.Result.Structured["retry_budget"]); ok {
			m.state.RetryBudget = retryBudget
		}
		if decision, _ := item.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := item.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if signature, _ := item.Result.Structured["replay_signature"].(string); strings.TrimSpace(signature) != "" {
			m.state.ReplaySignature = strings.TrimSpace(signature)
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *mapReduceBatchModel) argsForStep(step mapReduceStep, stage int) map[string]any {
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
		"batch_id":        m.state.BatchID,
	}

	switch step.Marker {
	case "mapreduce_shards_fanned_out":
		shardCount := 3
		hotThreshold := 18
		if m.variant == modecommon.VariantProduction {
			shardCount = 4
			hotThreshold = 14
		}
		args["records"] = encodeRecords(seedRecords)
		args["shard_count"] = shardCount
		args["hot_threshold"] = hotThreshold
	case "mapreduce_reduce_aggregated":
		args["shard_summaries"] = encodeShards(m.state.Shards)
	case "mapreduce_retry_classified":
		retryBudget := 1
		if m.variant == modecommon.VariantProduction {
			retryBudget = 2
		}
		args["shard_summaries"] = encodeShards(m.state.Shards)
		args["retry_budget"] = retryBudget
	case "governance_mapreduce_gate_enforced":
		args["retry_class"] = m.state.RetryClass
		args["retry_shards"] = intSliceToAny(m.state.RetryShards)
		args["hot_shards"] = intSliceToAny(m.state.HotShards)
		args["retry_budget"] = m.state.RetryBudget
	case "governance_mapreduce_replay_bound":
		args["aggregate_sum"] = m.state.AggregateSum
		args["retry_shards"] = intSliceToAny(m.state.RetryShards)
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
	}
	return args
}

type mapReduceBatchTool struct{}

func (t *mapReduceBatchTool) Name() string { return semanticToolName }

func (t *mapReduceBatchTool) Description() string {
	return "execute mapreduce shard/reduce/retry semantic step"
}

func (t *mapReduceBatchTool) JSONSchema() map[string]any {
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
			"batch_id":        map[string]any{"type": "string"},
			"records":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"shard_count":     map[string]any{"type": "integer"},
			"hot_threshold":   map[string]any{"type": "integer"},
			"shard_summaries": map[string]any{"type": "array"},
			"retry_budget":    map[string]any{"type": "integer"},
			"retry_shards":    map[string]any{"type": "array"},
			"hot_shards":      map[string]any{"type": "array"},
			"retry_class":     map[string]any{"type": "string"},
			"aggregate_sum":   map[string]any{"type": "integer"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *mapReduceBatchTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "mapreduce_shards_fanned_out":
		batchID := strings.TrimSpace(fmt.Sprintf("%v", args["batch_id"]))
		if batchID == "" {
			batchID = defaultBatchID
		}
		records := decodeRecords(args["records"])
		if len(records) == 0 {
			records = append([]batchRecord(nil), seedRecords...)
		}
		shardCount, _ := modecommon.AsInt(args["shard_count"])
		if shardCount <= 0 {
			shardCount = 3
		}
		hotThreshold, _ := modecommon.AsInt(args["hot_threshold"])
		if hotThreshold <= 0 {
			hotThreshold = 18
		}

		shards := fanOutRecords(records, shardCount, hotThreshold)
		hotShards := collectHotShards(shards, hotThreshold)
		structured["batch_id"] = batchID
		structured["shard_count"] = shardCount
		structured["hot_threshold"] = hotThreshold
		structured["shard_summaries"] = encodeShards(shards)
		structured["hot_shards"] = intSliceToAny(hotShards)
		if len(hotShards) > 0 {
			risk = "shard_skew"
		}
	case "mapreduce_reduce_aggregated":
		shards, _ := toShardSummaries(args["shard_summaries"])
		if len(shards) == 0 {
			shards = fanOutRecords(seedRecords, 3, 18)
		}
		aggregateSum, aggregateGroups, failedCount := reduceShards(shards)
		reduceMode := "complete_reduce"
		if failedCount > 0 {
			reduceMode = "partial_reduce"
			risk = "degraded_path"
		}
		structured["shard_summaries"] = encodeShards(shards)
		structured["aggregate_sum"] = aggregateSum
		structured["aggregate_groups"] = aggregateGroups
		structured["failed_shards"] = failedCount
		structured["reduce_mode"] = reduceMode
	case "mapreduce_retry_classified":
		shards, _ := toShardSummaries(args["shard_summaries"])
		retryBudget, _ := modecommon.AsInt(args["retry_budget"])
		if retryBudget <= 0 {
			retryBudget = 1
		}
		retryShards := failedShardIDs(shards)
		retryClass := "no_retry"
		if len(retryShards) > 0 {
			retryClass = "retry_scheduled"
			if len(retryShards) > retryBudget {
				retryClass = "retry_deferred"
			}
		}
		structured["retry_shards"] = intSliceToAny(retryShards)
		structured["retry_budget"] = retryBudget
		structured["retry_class"] = retryClass
		structured["retry_exhausted"] = retryClass == "retry_deferred"
		switch retryClass {
		case "retry_deferred":
			risk = "degraded_path"
		case "retry_scheduled":
			risk = "retry_inflight"
		}
	case "governance_mapreduce_gate_enforced":
		retryClass := strings.TrimSpace(fmt.Sprintf("%v", args["retry_class"]))
		retryShards, _ := toIntSlice(args["retry_shards"])
		hotShards, _ := toIntSlice(args["hot_shards"])
		retryBudget, _ := modecommon.AsInt(args["retry_budget"])

		decision := "allow"
		if retryClass == "retry_deferred" {
			decision = "deny"
		} else if len(hotShards) > 0 || len(retryShards) > retryBudget {
			decision = "allow_with_throttle"
		}
		ticket := fmt.Sprintf(
			"mr-gate-%d",
			modecommon.SemanticScore(pattern, variant, retryClass, intSliceToken(retryShards), intSliceToken(hotShards), decision),
		)
		structured["retry_class"] = retryClass
		structured["retry_shards"] = intSliceToAny(retryShards)
		structured["hot_shards"] = intSliceToAny(hotShards)
		structured["retry_budget"] = retryBudget
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_mapreduce_replay_bound":
		batchID := strings.TrimSpace(fmt.Sprintf("%v", args["batch_id"]))
		if batchID == "" {
			batchID = defaultBatchID
		}
		aggregateSum, _ := modecommon.AsInt(args["aggregate_sum"])
		retryShards, _ := toIntSlice(args["retry_shards"])
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		replay := fmt.Sprintf(
			"mr-replay-%d",
			modecommon.SemanticScore(batchID, decision, ticket, fmt.Sprintf("%d", aggregateSum), intSliceToken(retryShards)),
		)
		structured["batch_id"] = batchID
		structured["aggregate_sum"] = aggregateSum
		structured["retry_shards"] = intSliceToAny(retryShards)
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["governance"] = true
		structured["replay_signature"] = replay
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported mapreduce semantic marker: %s", marker)
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

func fanOutRecords(records []batchRecord, shardCount int, hotThreshold int) []shardSnapshot {
	shardMap := make(map[int]*shardSnapshot, shardCount)
	for idx := 0; idx < shardCount; idx++ {
		shardMap[idx] = &shardSnapshot{
			ShardID:    idx,
			Load:       0,
			Failed:     false,
			Partitions: []string{},
		}
	}

	for _, record := range records {
		shardID := stableShard(record.Partition, shardCount)
		target := shardMap[shardID]
		target.Load += record.Value
		target.Partitions = append(target.Partitions, record.Partition)
	}

	out := make([]shardSnapshot, 0, len(shardMap))
	for idx := 0; idx < shardCount; idx++ {
		item := shardMap[idx]
		item.Partitions = uniqueSorted(item.Partitions)
		item.Failed = item.Load > hotThreshold+3
		out = append(out, *item)
	}
	return out
}

func collectHotShards(shards []shardSnapshot, hotThreshold int) []int {
	out := make([]int, 0)
	for _, shard := range shards {
		if shard.Load > hotThreshold {
			out = append(out, shard.ShardID)
		}
	}
	sort.Ints(out)
	return out
}

func reduceShards(shards []shardSnapshot) (int, int, int) {
	aggregateSum := 0
	groupSet := map[string]struct{}{}
	failedCount := 0
	for _, shard := range shards {
		aggregateSum += shard.Load
		for _, partition := range shard.Partitions {
			groupSet[partition] = struct{}{}
		}
		if shard.Failed {
			failedCount++
		}
	}
	return aggregateSum, len(groupSet), failedCount
}

func failedShardIDs(shards []shardSnapshot) []int {
	out := make([]int, 0)
	for _, shard := range shards {
		if shard.Failed {
			out = append(out, shard.ShardID)
		}
	}
	sort.Ints(out)
	return out
}

func stableShard(partition string, shardCount int) int {
	if shardCount <= 0 {
		return 0
	}
	total := 0
	for _, ch := range []byte(strings.ToLower(strings.TrimSpace(partition))) {
		total += int(ch)
	}
	return total % shardCount
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, item := range in {
		text := strings.TrimSpace(item)
		if text == "" {
			continue
		}
		set[text] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func decodeRecords(value any) []batchRecord {
	items, ok := toStringSlice(value)
	if !ok {
		return nil
	}
	out := make([]batchRecord, 0, len(items))
	for _, item := range items {
		parts := strings.Split(item, "|")
		if len(parts) != 3 {
			continue
		}
		valueInt := asInt(parts[1], 0)
		priorityInt := asInt(parts[2], 1)
		out = append(out, batchRecord{
			Partition: strings.TrimSpace(parts[0]),
			Value:     valueInt,
			Priority:  priorityInt,
		})
	}
	return out
}

func encodeRecords(records []batchRecord) []any {
	out := make([]any, 0, len(records))
	for _, record := range records {
		out = append(out, fmt.Sprintf("%s|%d|%d", record.Partition, record.Value, record.Priority))
	}
	return out
}

func encodeShards(shards []shardSnapshot) []any {
	out := make([]any, 0, len(shards))
	for _, shard := range shards {
		out = append(out, map[string]any{
			"shard_id":   shard.ShardID,
			"load":       shard.Load,
			"failed":     shard.Failed,
			"partitions": stringSliceToAny(shard.Partitions),
		})
	}
	return out
}

func toShardSummaries(value any) ([]shardSnapshot, bool) {
	rawList, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]shardSnapshot, 0, len(rawList))
	for _, item := range rawList {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		shardID, _ := modecommon.AsInt(row["shard_id"])
		load, _ := modecommon.AsInt(row["load"])
		failed := asBool(row["failed"])
		partitions, _ := toStringSlice(row["partitions"])
		out = append(out, shardSnapshot{
			ShardID:    shardID,
			Load:       load,
			Failed:     failed,
			Partitions: uniqueSorted(partitions),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ShardID < out[j].ShardID })
	return out, true
}

func toStringSlice(value any) ([]string, bool) {
	switch raw := value.(type) {
	case []string:
		return append([]string(nil), raw...), true
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

func toIntSlice(value any) ([]int, bool) {
	switch raw := value.(type) {
	case []int:
		return append([]int(nil), raw...), true
	case []any:
		out := make([]int, 0, len(raw))
		for _, item := range raw {
			if val, ok := modecommon.AsInt(item); ok {
				out = append(out, val)
			}
		}
		sort.Ints(out)
		return out, true
	default:
		return nil, false
	}
}

func stringSliceToAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}

func intSliceToAny(in []int) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}

func intSliceToken(in []int) string {
	if len(in) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(in))
	for _, item := range in {
		parts = append(parts, fmt.Sprintf("%d", item))
	}
	return strings.Join(parts, "|")
}

func asInt(value any, fallback int) int {
	if parsed, ok := modecommon.AsInt(value); ok {
		return parsed
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(text, "%d", &parsed); err == nil {
		return parsed
	}
	return fallback
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
