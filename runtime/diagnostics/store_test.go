package diagnostics

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func TestStoreConcurrentAccess(t *testing.T) {
	d := NewStore(32, 16, 8, 20, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				d.AddCall(CallRecord{
					Time:      time.Now(),
					Component: "mcp",
					Transport: "http",
					CallID:    strconv.Itoa(id*100 + j),
				})
				d.AddSkill(SkillRecord{
					Time:      time.Now(),
					SkillName: "skill-a",
					Action:    "compile",
					Status:    "success",
				})
				_ = d.RecentCalls(5)
				_ = d.RecentSkills(5)
			}
		}(i)
	}
	wg.Wait()
	if got := len(d.RecentCalls(100)); got > 32 {
		t.Fatalf("call records = %d, want <= 32", got)
	}
	if got := len(d.RecentSkills(100)); got > 20 {
		t.Fatalf("skill records = %d, want <= 20", got)
	}
}

func TestSanitizeMap(t *testing.T) {
	in := map[string]any{
		"api_key": "abc",
		"nested": map[string]any{
			"token": "x",
			"name":  "ok",
		},
	}
	out := SanitizeMap(in)
	if out["api_key"] != "***" {
		t.Fatalf("api_key should be masked")
	}
	nested, _ := out["nested"].(map[string]any)
	if nested["token"] != "***" {
		t.Fatalf("nested token should be masked")
	}
	if nested["name"] != "ok" {
		t.Fatalf("non-sensitive field should keep value")
	}
}

func TestStoreRunDedupByIdempotencyKey(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:       time.Now(),
		RunID:      "run-1",
		Status:     "success",
		Iterations: 2,
		ToolCalls:  1,
		LatencyMs:  12,
	}
	d.AddRun(rec)
	rec.LatencyMs = 99
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].LatencyMs != 99 {
		t.Fatalf("run record should be replaced on duplicate key, got %#v", runs[0])
	}
}

func TestStoreRunCardinalityTruncateAndRecordDeterministic(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.SetCardinalityConfig(CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  2,
		MaxListEntries: 2,
		MaxStringBytes: 8,
		OverflowPolicy: CardinalityOverflowTruncateAndRecord,
	})
	rec := RunRecord{
		Time:                   time.Now(),
		RunID:                  "run-a45-truncate",
		Status:                 "success",
		TimeoutResolutionTrace: "你好world你好",
		TaskBoardManualControlByReason: map[string]int{
			"gamma": 3,
			"alpha": 1,
			"beta":  2,
		},
		TimelinePhases: map[string]TimelinePhaseAggregate{
			"gamma": {CountTotal: 3},
			"alpha": {CountTotal: 1},
			"beta":  {CountTotal: 2},
		},
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records len = %d, want 1", len(runs))
	}
	got := runs[0]
	if got.DiagnosticsCardinalityOverflowPolicy != CardinalityOverflowTruncateAndRecord {
		t.Fatalf("overflow policy = %q, want %q", got.DiagnosticsCardinalityOverflowPolicy, CardinalityOverflowTruncateAndRecord)
	}
	if got.DiagnosticsCardinalityBudgetHitTotal < 2 || got.DiagnosticsCardinalityTruncatedTotal < 2 {
		t.Fatalf("expected budget/truncation counters >0, got %#v", got)
	}
	if !strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "task_board_manual_control_by_reason") {
		t.Fatalf("summary must include task_board_manual_control_by_reason, got %#v", got.DiagnosticsCardinalityTruncatedFieldSummary)
	}
	if !strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "timeout_resolution_trace") {
		t.Fatalf("summary must include timeout_resolution_trace, got %#v", got.DiagnosticsCardinalityTruncatedFieldSummary)
	}
	if len(got.TaskBoardManualControlByReason) != 2 {
		t.Fatalf("map should be truncated to 2 entries, got %#v", got.TaskBoardManualControlByReason)
	}
	if _, ok := got.TaskBoardManualControlByReason["alpha"]; !ok {
		t.Fatalf("sorted key alpha should be retained after truncation, got %#v", got.TaskBoardManualControlByReason)
	}
	if _, ok := got.TaskBoardManualControlByReason["beta"]; !ok {
		t.Fatalf("sorted key beta should be retained after truncation, got %#v", got.TaskBoardManualControlByReason)
	}
	if _, ok := got.TaskBoardManualControlByReason["gamma"]; ok {
		t.Fatalf("sorted key gamma should be truncated, got %#v", got.TaskBoardManualControlByReason)
	}
	if len(got.TimelinePhases) != 2 {
		t.Fatalf("timeline_phases should be truncated to 2 entries, got %#v", got.TimelinePhases)
	}
	if len([]byte(got.TimeoutResolutionTrace)) > 8 || !utf8.ValidString(got.TimeoutResolutionTrace) {
		t.Fatalf("timeout_resolution_trace must be utf8-safe truncated to 8 bytes, got %q", got.TimeoutResolutionTrace)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a45-truncate"})
	if err != nil {
		t.Fatalf("query runs failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len = %d, want 1", len(page.Items))
	}
	if page.Items[0].DiagnosticsCardinalityTruncatedFieldSummary != got.DiagnosticsCardinalityTruncatedFieldSummary {
		t.Fatalf("query mapping mismatch: query=%#v store=%#v", page.Items[0], got)
	}
}

func TestStoreRunCardinalityFailFastRejectsOverflowPayload(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.SetCardinalityConfig(CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  2,
		MaxListEntries: 2,
		MaxStringBytes: 4,
		OverflowPolicy: CardinalityOverflowFailFast,
	})
	rec := RunRecord{
		Time:                   time.Now(),
		RunID:                  "run-a45-fail-fast",
		Status:                 "failed",
		TimeoutResolutionTrace: "abcdefg",
		TaskBoardManualControlByReason: map[string]int{
			"gamma": 3,
			"alpha": 1,
			"beta":  2,
		},
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records len = %d, want 1", len(runs))
	}
	got := runs[0]
	if got.DiagnosticsCardinalityOverflowPolicy != CardinalityOverflowFailFast {
		t.Fatalf("overflow policy = %q, want %q", got.DiagnosticsCardinalityOverflowPolicy, CardinalityOverflowFailFast)
	}
	if got.DiagnosticsCardinalityFailFastRejectTotal != 1 {
		t.Fatalf("fail-fast reject total = %d, want 1", got.DiagnosticsCardinalityFailFastRejectTotal)
	}
	if got.DiagnosticsCardinalityBudgetHitTotal <= 0 {
		t.Fatalf("budget_hit_total should be > 0, got %#v", got)
	}
	if got.DiagnosticsCardinalityTruncatedTotal != 0 {
		t.Fatalf("truncated_total should stay 0 under fail_fast, got %#v", got)
	}
	if got.TimeoutResolutionTrace != "" {
		t.Fatalf("overflowing string should be removed under fail_fast, got %q", got.TimeoutResolutionTrace)
	}
	if got.TaskBoardManualControlByReason != nil {
		t.Fatalf("overflowing map should be removed under fail_fast, got %#v", got.TaskBoardManualControlByReason)
	}
	if !strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "task_board_manual_control_by_reason") {
		t.Fatalf("summary should include rejected field names, got %q", got.DiagnosticsCardinalityTruncatedFieldSummary)
	}
}

func TestCardinalityListGovernanceDeterministic(t *testing.T) {
	cfg := normalizeCardinalityConfig(CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  3,
		MaxListEntries: 2,
		MaxStringBytes: 16,
		OverflowPolicy: CardinalityOverflowTruncateAndRecord,
	})
	stats := &cardinalityGovernanceStats{
		budgetHitFields: map[string]struct{}{},
		truncatedFields: map[string]struct{}{},
	}
	out, overflow := governCardinalityValue([]any{"a", "b", "c"}, "field_list", cfg, stats, true)
	if !overflow {
		t.Fatal("list overflow expected for truncate_and_record")
	}
	trimmed, _ := out.([]any)
	if len(trimmed) != 2 || trimmed[0] != "a" || trimmed[1] != "b" {
		t.Fatalf("list truncation should keep first N order, got %#v", out)
	}

	failFastCfg := cfg
	failFastCfg.OverflowPolicy = CardinalityOverflowFailFast
	stats = &cardinalityGovernanceStats{
		budgetHitFields: map[string]struct{}{},
		truncatedFields: map[string]struct{}{},
	}
	out, overflow = governCardinalityValue([]any{"a", "b", "c"}, "field_list", failFastCfg, stats, true)
	if !overflow {
		t.Fatal("list overflow expected for fail_fast")
	}
	if out != nil {
		t.Fatalf("fail_fast should reject overflowing list payload, got %#v", out)
	}
	if len(stats.truncatedFields) != 0 {
		t.Fatalf("fail_fast should not report truncation, got %#v", stats.truncatedFields)
	}
}

func TestStoreRunReadinessAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                                        time.Now(),
		RunID:                                       "run-a40-readiness",
		Status:                                      "success",
		RuntimeReadinessStatus:                      "degraded",
		RuntimeReadinessFindingTotal:                2,
		RuntimeReadinessBlockingTotal:               0,
		RuntimeReadinessDegradedTotal:               2,
		RuntimePrimaryDomain:                        "scheduler",
		RuntimePrimaryCode:                          "scheduler.backend.fallback",
		RuntimePrimarySource:                        "runtime.readiness",
		RuntimePrimaryConflictTotal:                 0,
		RuntimeSecondaryReasonCodes:                 []string{"mailbox.backend.fallback", "recovery.backend.fallback"},
		RuntimeSecondaryReasonCount:                 2,
		RuntimeArbitrationRuleVersion:               "a49.v1",
		RuntimeRemediationHintCode:                  "scheduler.recover_backend",
		RuntimeRemediationHintDomain:                "scheduler",
		RuntimeReadinessPrimaryCode:                 "scheduler.backend.fallback",
		RuntimeReadinessAdmissionTotal:              1,
		RuntimeReadinessAdmissionBlockedTotal:       0,
		RuntimeReadinessAdmissionDegradedAllowTotal: 1,
		RuntimeReadinessAdmissionBypassTotal:        0,
		RuntimeReadinessAdmissionMode:               "fail_fast",
		RuntimeReadinessAdmissionPrimaryCode:        "scheduler.backend.fallback",
		AdapterHealthStatus:                         "unavailable",
		AdapterHealthProbeTotal:                     3,
		AdapterHealthDegradedTotal:                  1,
		AdapterHealthUnavailableTotal:               2,
		AdapterHealthPrimaryCode:                    "adapter.health.required_unavailable",
		AdapterHealthBackoffAppliedTotal:            4,
		AdapterHealthCircuitOpenTotal:               2,
		AdapterHealthCircuitHalfOpenTotal:           1,
		AdapterHealthCircuitRecoverTotal:            1,
		AdapterHealthCircuitState:                   "open",
		AdapterHealthGovernancePrimaryCode:          "adapter.health.circuit_open",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	if items[0].RuntimeReadinessStatus != "degraded" ||
		items[0].RuntimeReadinessFindingTotal != 2 ||
		items[0].RuntimeReadinessBlockingTotal != 0 ||
		items[0].RuntimeReadinessDegradedTotal != 2 ||
		items[0].RuntimePrimaryDomain != "scheduler" ||
		items[0].RuntimePrimaryCode != "scheduler.backend.fallback" ||
		items[0].RuntimePrimarySource != "runtime.readiness" ||
		items[0].RuntimePrimaryConflictTotal != 0 ||
		len(items[0].RuntimeSecondaryReasonCodes) != 2 ||
		items[0].RuntimeSecondaryReasonCodes[0] != "mailbox.backend.fallback" ||
		items[0].RuntimeSecondaryReasonCodes[1] != "recovery.backend.fallback" ||
		items[0].RuntimeSecondaryReasonCount != 2 ||
		items[0].RuntimeArbitrationRuleVersion != "a49.v1" ||
		items[0].RuntimeRemediationHintCode != "scheduler.recover_backend" ||
		items[0].RuntimeRemediationHintDomain != "scheduler" ||
		items[0].RuntimeReadinessPrimaryCode != "scheduler.backend.fallback" ||
		items[0].RuntimeReadinessAdmissionTotal != 1 ||
		items[0].RuntimeReadinessAdmissionBlockedTotal != 0 ||
		items[0].RuntimeReadinessAdmissionDegradedAllowTotal != 1 ||
		items[0].RuntimeReadinessAdmissionBypassTotal != 0 ||
		items[0].RuntimeReadinessAdmissionMode != "fail_fast" ||
		items[0].RuntimeReadinessAdmissionPrimaryCode != "scheduler.backend.fallback" ||
		items[0].AdapterHealthStatus != "unavailable" ||
		items[0].AdapterHealthProbeTotal != 3 ||
		items[0].AdapterHealthDegradedTotal != 1 ||
		items[0].AdapterHealthUnavailableTotal != 2 ||
		items[0].AdapterHealthPrimaryCode != "adapter.health.required_unavailable" ||
		items[0].AdapterHealthBackoffAppliedTotal != 4 ||
		items[0].AdapterHealthCircuitOpenTotal != 2 ||
		items[0].AdapterHealthCircuitHalfOpenTotal != 1 ||
		items[0].AdapterHealthCircuitRecoverTotal != 1 ||
		items[0].AdapterHealthCircuitState != "open" ||
		items[0].AdapterHealthGovernancePrimaryCode != "adapter.health.circuit_open" {
		t.Fatalf("readiness fields mismatch after dedup: %#v", items[0])
	}

	rec.RuntimeReadinessStatus = "blocked"
	rec.RuntimeReadinessFindingTotal = 3
	rec.RuntimeReadinessBlockingTotal = 2
	rec.RuntimeReadinessDegradedTotal = 1
	rec.RuntimePrimaryDomain = "timeout"
	rec.RuntimePrimaryCode = "runtime.timeout.parent_budget_rejected"
	rec.RuntimePrimarySource = "timeout.resolution.request"
	rec.RuntimePrimaryConflictTotal = 1
	rec.RuntimeSecondaryReasonCodes = []string{"runtime.timeout.exhausted", "runtime.timeout.parent_budget_clamped"}
	rec.RuntimeSecondaryReasonCount = 2
	rec.RuntimeArbitrationRuleVersion = "a49.v1"
	rec.RuntimeRemediationHintCode = "timeout.adjust_parent_budget"
	rec.RuntimeRemediationHintDomain = "timeout"
	rec.RuntimeReadinessPrimaryCode = "runtime.readiness.strict_escalated"
	rec.RuntimeReadinessAdmissionTotal = 1
	rec.RuntimeReadinessAdmissionBlockedTotal = 1
	rec.RuntimeReadinessAdmissionDegradedAllowTotal = 0
	rec.RuntimeReadinessAdmissionBypassTotal = 0
	rec.RuntimeReadinessAdmissionMode = "fail_fast"
	rec.RuntimeReadinessAdmissionPrimaryCode = "runtime.readiness.strict_escalated"
	rec.AdapterHealthStatus = "degraded"
	rec.AdapterHealthProbeTotal = 2
	rec.AdapterHealthDegradedTotal = 2
	rec.AdapterHealthUnavailableTotal = 0
	rec.AdapterHealthPrimaryCode = "adapter.health.degraded"
	rec.AdapterHealthBackoffAppliedTotal = 1
	rec.AdapterHealthCircuitOpenTotal = 1
	rec.AdapterHealthCircuitHalfOpenTotal = 1
	rec.AdapterHealthCircuitRecoverTotal = 1
	rec.AdapterHealthCircuitState = "half_open"
	rec.AdapterHealthGovernancePrimaryCode = "adapter.health.circuit_half_open"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay update, want 1", len(items))
	}
	if items[0].RuntimeReadinessStatus != "blocked" ||
		items[0].RuntimeReadinessFindingTotal != 3 ||
		items[0].RuntimeReadinessBlockingTotal != 2 ||
		items[0].RuntimeReadinessDegradedTotal != 1 ||
		items[0].RuntimePrimaryDomain != "timeout" ||
		items[0].RuntimePrimaryCode != "runtime.timeout.parent_budget_rejected" ||
		items[0].RuntimePrimarySource != "timeout.resolution.request" ||
		items[0].RuntimePrimaryConflictTotal != 1 ||
		len(items[0].RuntimeSecondaryReasonCodes) != 2 ||
		items[0].RuntimeSecondaryReasonCodes[0] != "runtime.timeout.exhausted" ||
		items[0].RuntimeSecondaryReasonCodes[1] != "runtime.timeout.parent_budget_clamped" ||
		items[0].RuntimeSecondaryReasonCount != 2 ||
		items[0].RuntimeArbitrationRuleVersion != "a49.v1" ||
		items[0].RuntimeRemediationHintCode != "timeout.adjust_parent_budget" ||
		items[0].RuntimeRemediationHintDomain != "timeout" ||
		items[0].RuntimeReadinessPrimaryCode != "runtime.readiness.strict_escalated" ||
		items[0].RuntimeReadinessAdmissionTotal != 1 ||
		items[0].RuntimeReadinessAdmissionBlockedTotal != 1 ||
		items[0].RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
		items[0].RuntimeReadinessAdmissionBypassTotal != 0 ||
		items[0].RuntimeReadinessAdmissionMode != "fail_fast" ||
		items[0].RuntimeReadinessAdmissionPrimaryCode != "runtime.readiness.strict_escalated" ||
		items[0].AdapterHealthStatus != "degraded" ||
		items[0].AdapterHealthProbeTotal != 2 ||
		items[0].AdapterHealthDegradedTotal != 2 ||
		items[0].AdapterHealthUnavailableTotal != 0 ||
		items[0].AdapterHealthPrimaryCode != "adapter.health.degraded" ||
		items[0].AdapterHealthBackoffAppliedTotal != 1 ||
		items[0].AdapterHealthCircuitOpenTotal != 1 ||
		items[0].AdapterHealthCircuitHalfOpenTotal != 1 ||
		items[0].AdapterHealthCircuitRecoverTotal != 1 ||
		items[0].AdapterHealthCircuitState != "half_open" ||
		items[0].AdapterHealthGovernancePrimaryCode != "adapter.health.circuit_half_open" {
		t.Fatalf("readiness fields mismatch after replay replacement: %#v", items[0])
	}
}

func TestStoreRunSandboxEgressAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                        time.Now(),
		RunID:                       "run-a57-additive",
		Status:                      "failed",
		SandboxEgressAction:         "deny",
		SandboxEgressViolationTotal: 2,
		SandboxEgressPolicySource:   "by_tool",
		AdapterAllowlistDecision:    "deny",
		AdapterAllowlistBlockTotal:  1,
		AdapterAllowlistPrimaryCode: "adapter.allowlist.missing_entry",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	if items[0].SandboxEgressAction != "deny" ||
		items[0].SandboxEgressViolationTotal != 2 ||
		items[0].SandboxEgressPolicySource != "by_tool" ||
		items[0].AdapterAllowlistDecision != "deny" ||
		items[0].AdapterAllowlistBlockTotal != 1 ||
		items[0].AdapterAllowlistPrimaryCode != "adapter.allowlist.missing_entry" {
		t.Fatalf("sandbox egress additive fields mismatch after dedup: %#v", items[0])
	}

	rec.SandboxEgressAction = "allow_and_record"
	rec.SandboxEgressViolationTotal = 3
	rec.SandboxEgressPolicySource = "on_violation"
	rec.AdapterAllowlistDecision = "allow"
	rec.AdapterAllowlistBlockTotal = 0
	rec.AdapterAllowlistPrimaryCode = "adapter.allowlist.signature_invalid"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	if items[0].SandboxEgressAction != "allow_and_record" ||
		items[0].SandboxEgressViolationTotal != 3 ||
		items[0].SandboxEgressPolicySource != "on_violation" ||
		items[0].AdapterAllowlistDecision != "allow" ||
		items[0].AdapterAllowlistBlockTotal != 0 ||
		items[0].AdapterAllowlistPrimaryCode != "adapter.allowlist.signature_invalid" {
		t.Fatalf("sandbox egress additive fields mismatch after replay replacement: %#v", items[0])
	}
}

func TestStoreRunMemoryGovernanceAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                  time.Now(),
		RunID:                 "run-a59-memory-governance",
		Status:                "success",
		MemoryScopeSelected:   "session",
		MemoryBudgetUsed:      3,
		MemoryHits:            3,
		MemoryRerankStats:     map[string]int{"input_total": 4, "output_total": 3},
		MemoryLifecycleAction: "ttl_expired",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	if items[0].MemoryScopeSelected != "session" ||
		items[0].MemoryBudgetUsed != 3 ||
		items[0].MemoryHits != 3 ||
		items[0].MemoryRerankStats["input_total"] != 4 ||
		items[0].MemoryRerankStats["output_total"] != 3 ||
		items[0].MemoryLifecycleAction != "ttl_expired" {
		t.Fatalf("memory governance additive fields mismatch after dedup: %#v", items[0])
	}

	rec.MemoryScopeSelected = "project"
	rec.MemoryBudgetUsed = 2
	rec.MemoryHits = 2
	rec.MemoryRerankStats = map[string]int{"input_total": 2, "output_total": 2}
	rec.MemoryLifecycleAction = "forget_applied"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	if items[0].MemoryScopeSelected != "project" ||
		items[0].MemoryBudgetUsed != 2 ||
		items[0].MemoryHits != 2 ||
		items[0].MemoryRerankStats["input_total"] != 2 ||
		items[0].MemoryRerankStats["output_total"] != 2 ||
		items[0].MemoryLifecycleAction != "forget_applied" {
		t.Fatalf("memory governance additive fields mismatch after replay replacement: %#v", items[0])
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a59-memory-governance"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len=%d, want 1", len(page.Items))
	}
	if page.Items[0].MemoryScopeSelected != "project" ||
		page.Items[0].MemoryBudgetUsed != 2 ||
		page.Items[0].MemoryHits != 2 ||
		page.Items[0].MemoryRerankStats["input_total"] != 2 ||
		page.Items[0].MemoryLifecycleAction != "forget_applied" {
		t.Fatalf("memory governance query mapping mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunHooksMiddlewareAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                              time.Now(),
		RunID:                             "run-a65-hooks-middleware",
		Status:                            "failed",
		HooksEnabled:                      true,
		HooksFailMode:                     "degrade",
		HooksPhases:                       []string{"before_reasoning", "after_reasoning"},
		ToolMiddlewareEnabled:             true,
		ToolMiddlewareFailMode:            "fail_fast",
		SkillDiscoveryMode:                "hybrid",
		SkillDiscoveryRoots:               []string{"./skills", "./agents"},
		SkillPreprocessEnabled:            true,
		SkillPreprocessPhase:              "before_run_stream",
		SkillPreprocessFailMode:           "degrade",
		SkillPreprocessStatus:             "degraded",
		SkillPreprocessReasonCode:         "skill_preprocess_failed",
		SkillPreprocessSpecCount:          2,
		SkillBundlePromptMode:             "append",
		SkillBundleWhitelistMode:          "merge",
		SkillBundleConflictPolicy:         "first_win",
		SkillBundlePromptTotal:            1,
		SkillBundleWhitelistTotal:         3,
		SkillBundleWhitelistRejectedTotal: 1,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if !got.HooksEnabled ||
		got.HooksFailMode != "degrade" ||
		len(got.HooksPhases) != 2 ||
		got.HooksPhases[0] != "before_reasoning" ||
		got.HooksPhases[1] != "after_reasoning" ||
		!got.ToolMiddlewareEnabled ||
		got.ToolMiddlewareFailMode != "fail_fast" ||
		got.SkillDiscoveryMode != "hybrid" ||
		len(got.SkillDiscoveryRoots) != 2 ||
		got.SkillDiscoveryRoots[0] != "./skills" ||
		got.SkillDiscoveryRoots[1] != "./agents" ||
		!got.SkillPreprocessEnabled ||
		got.SkillPreprocessPhase != "before_run_stream" ||
		got.SkillPreprocessFailMode != "degrade" ||
		got.SkillPreprocessStatus != "degraded" ||
		got.SkillPreprocessReasonCode != "skill_preprocess_failed" ||
		got.SkillPreprocessSpecCount != 2 ||
		got.SkillBundlePromptMode != "append" ||
		got.SkillBundleWhitelistMode != "merge" ||
		got.SkillBundleConflictPolicy != "first_win" ||
		got.SkillBundlePromptTotal != 1 ||
		got.SkillBundleWhitelistTotal != 3 ||
		got.SkillBundleWhitelistRejectedTotal != 1 {
		t.Fatalf("hooks/middleware additive fields mismatch after dedup: %#v", got)
	}

	rec.SkillPreprocessStatus = "failed"
	rec.SkillPreprocessReasonCode = "skill_bundle_whitelist_violation"
	rec.SkillBundlePromptTotal = 2
	rec.SkillBundleWhitelistRejectedTotal = 2
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	got = items[0]
	if got.SkillPreprocessStatus != "failed" ||
		got.SkillPreprocessReasonCode != "skill_bundle_whitelist_violation" ||
		got.SkillBundlePromptTotal != 2 ||
		got.SkillBundleWhitelistRejectedTotal != 2 {
		t.Fatalf("hooks/middleware additive fields mismatch after replay replacement: %#v", got)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a65-hooks-middleware"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("QueryRuns items len = %d, want 1", len(page.Items))
	}
	if page.Items[0].SkillPreprocessStatus != "failed" ||
		page.Items[0].SkillPreprocessReasonCode != "skill_bundle_whitelist_violation" ||
		page.Items[0].SkillBundlePromptTotal != 2 ||
		page.Items[0].SkillBundleWhitelistRejectedTotal != 2 {
		t.Fatalf("hooks/middleware QueryRuns additive parse mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunReactPlanNotebookAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                  time.Now(),
		RunID:                 "run-react-plan-notebook",
		Status:                "success",
		ReactPlanID:           "run-react-plan-notebook",
		ReactPlanVersion:      3,
		ReactPlanChangeTotal:  3,
		ReactPlanLastAction:   "complete",
		ReactPlanChangeReason: "run_completed",
		ReactPlanRecoverCount: 1,
		ReactPlanHookStatus:   "ok",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if got.ReactPlanID != "run-react-plan-notebook" ||
		got.ReactPlanVersion != 3 ||
		got.ReactPlanChangeTotal != 3 ||
		got.ReactPlanLastAction != "complete" ||
		got.ReactPlanChangeReason != "run_completed" ||
		got.ReactPlanRecoverCount != 1 ||
		got.ReactPlanHookStatus != "ok" {
		t.Fatalf("react plan notebook additive fields mismatch after dedup: %#v", got)
	}

	rec.ReactPlanVersion = 4
	rec.ReactPlanChangeTotal = 4
	rec.ReactPlanLastAction = "recover"
	rec.ReactPlanChangeReason = "session_resume"
	rec.ReactPlanRecoverCount = 2
	rec.ReactPlanHookStatus = "degraded"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	got = items[0]
	if got.ReactPlanVersion != 4 ||
		got.ReactPlanChangeTotal != 4 ||
		got.ReactPlanLastAction != "recover" ||
		got.ReactPlanChangeReason != "session_resume" ||
		got.ReactPlanRecoverCount != 2 ||
		got.ReactPlanHookStatus != "degraded" {
		t.Fatalf("react plan notebook additive fields mismatch after replay replacement: %#v", got)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-react-plan-notebook"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("QueryRuns items len = %d, want 1", len(page.Items))
	}
	if page.Items[0].ReactPlanVersion != 4 ||
		page.Items[0].ReactPlanChangeTotal != 4 ||
		page.Items[0].ReactPlanLastAction != "recover" ||
		page.Items[0].ReactPlanRecoverCount != 2 ||
		page.Items[0].ReactPlanHookStatus != "degraded" {
		t.Fatalf("react plan notebook QueryRuns additive parse mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunReactPlanNotebookQueryRunsParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.AddRun(RunRecord{
		Time:      time.Now(),
		RunID:     "run-react-plan-compat",
		Status:    "success",
		LatencyMs: 21,
	})

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-react-plan-compat"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("QueryRuns items len=%d, want 1", len(page.Items))
	}
	got := page.Items[0]
	if got.Status != "success" || got.LatencyMs != 21 {
		t.Fatalf("existing fields mismatch: %#v", got)
	}
	if got.ReactPlanID != "" ||
		got.ReactPlanVersion != 0 ||
		got.ReactPlanChangeTotal != 0 ||
		got.ReactPlanLastAction != "" ||
		got.ReactPlanChangeReason != "" ||
		got.ReactPlanRecoverCount != 0 ||
		got.ReactPlanHookStatus != "" {
		t.Fatalf("missing react plan notebook additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestStoreRunContextJITAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                            time.Now(),
		RunID:                           "run-context-jit",
		Status:                          "success",
		ContextRefDiscoverCount:         6,
		ContextRefResolveCount:          4,
		ContextEditEstimatedSavedTokens: 128,
		ContextEditGateDecision:         "allow.threshold_met",
		ContextSwapbackRelevanceScore:   0.82,
		ContextLifecycleTierStats:       map[string]int{"hot": 2, "warm": 3, "cold": 1, "pruned": 0},
		ContextRecapSource:              "task_aware.stage_actions.v1",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if got.ContextRefDiscoverCount != 6 ||
		got.ContextRefResolveCount != 4 ||
		got.ContextEditEstimatedSavedTokens != 128 ||
		got.ContextEditGateDecision != "allow.threshold_met" ||
		got.ContextSwapbackRelevanceScore != 0.82 ||
		got.ContextLifecycleTierStats["warm"] != 3 ||
		got.ContextRecapSource != "task_aware.stage_actions.v1" {
		t.Fatalf("context jit additive fields mismatch after dedup: %#v", got)
	}

	rec.ContextRefResolveCount = 5
	rec.ContextEditEstimatedSavedTokens = 96
	rec.ContextEditGateDecision = "deny.gain_ratio_below_threshold"
	rec.ContextSwapbackRelevanceScore = 0.65
	rec.ContextLifecycleTierStats = map[string]int{"hot": 1, "warm": 2, "cold": 2, "pruned": 1}
	rec.ContextRecapSource = "task_aware.stage_actions.v1"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	got = items[0]
	if got.ContextRefResolveCount != 5 ||
		got.ContextEditEstimatedSavedTokens != 96 ||
		got.ContextEditGateDecision != "deny.gain_ratio_below_threshold" ||
		got.ContextSwapbackRelevanceScore != 0.65 ||
		got.ContextLifecycleTierStats["pruned"] != 1 {
		t.Fatalf("context jit additive fields mismatch after replay replacement: %#v", got)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-context-jit"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("QueryRuns items len = %d, want 1", len(page.Items))
	}
	if page.Items[0].ContextRefDiscoverCount != 6 ||
		page.Items[0].ContextRefResolveCount != 5 ||
		page.Items[0].ContextEditEstimatedSavedTokens != 96 ||
		page.Items[0].ContextEditGateDecision != "deny.gain_ratio_below_threshold" ||
		page.Items[0].ContextSwapbackRelevanceScore != 0.65 ||
		page.Items[0].ContextLifecycleTierStats["hot"] != 1 ||
		page.Items[0].ContextRecapSource != "task_aware.stage_actions.v1" {
		t.Fatalf("context jit QueryRuns additive parse mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunContextJITQueryRunsParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.AddRun(RunRecord{
		Time:      time.Now(),
		RunID:     "run-context-jit-compat",
		Status:    "success",
		LatencyMs: 21,
	})

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-context-jit-compat"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("QueryRuns items len=%d, want 1", len(page.Items))
	}
	got := page.Items[0]
	if got.Status != "success" || got.LatencyMs != 21 {
		t.Fatalf("existing fields mismatch: %#v", got)
	}
	if got.ContextRefDiscoverCount != 0 ||
		got.ContextRefResolveCount != 0 ||
		got.ContextEditEstimatedSavedTokens != 0 ||
		got.ContextEditGateDecision != "" ||
		got.ContextSwapbackRelevanceScore != 0 ||
		len(got.ContextLifecycleTierStats) != 0 ||
		got.ContextRecapSource != "" {
		t.Fatalf("missing context jit additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestStoreRunRealtimeProtocolAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                          time.Now(),
		RunID:                         "run-realtime-protocol",
		Status:                        "success",
		RealtimeProtocolVersion:       "realtime_event_protocol.v1",
		RealtimeEventSeqMax:           12,
		RealtimeInterruptTotal:        1,
		RealtimeResumeTotal:           1,
		RealtimeResumeSource:          "cursor",
		RealtimeIdempotencyDedupTotal: 2,
		RealtimeLastErrorCode:         "",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if got.RealtimeProtocolVersion != "realtime_event_protocol.v1" ||
		got.RealtimeEventSeqMax != 12 ||
		got.RealtimeInterruptTotal != 1 ||
		got.RealtimeResumeTotal != 1 ||
		got.RealtimeResumeSource != "cursor" ||
		got.RealtimeIdempotencyDedupTotal != 2 {
		t.Fatalf("realtime protocol additive fields mismatch after dedup: %#v", got)
	}

	rec.RealtimeEventSeqMax = 24
	rec.RealtimeInterruptTotal = 2
	rec.RealtimeResumeTotal = 2
	rec.RealtimeResumeSource = "persisted_cursor"
	rec.RealtimeIdempotencyDedupTotal = 4
	rec.RealtimeLastErrorCode = "realtime.sequence_gap"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	got = items[0]
	if got.RealtimeEventSeqMax != 24 ||
		got.RealtimeInterruptTotal != 2 ||
		got.RealtimeResumeTotal != 2 ||
		got.RealtimeResumeSource != "persisted_cursor" ||
		got.RealtimeIdempotencyDedupTotal != 4 ||
		got.RealtimeLastErrorCode != "realtime.sequence_gap" {
		t.Fatalf("realtime protocol additive fields mismatch after replay replacement: %#v", got)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-realtime-protocol"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("QueryRuns items len = %d, want 1", len(page.Items))
	}
	if page.Items[0].RealtimeEventSeqMax != 24 ||
		page.Items[0].RealtimeInterruptTotal != 2 ||
		page.Items[0].RealtimeResumeTotal != 2 ||
		page.Items[0].RealtimeResumeSource != "persisted_cursor" ||
		page.Items[0].RealtimeIdempotencyDedupTotal != 4 ||
		page.Items[0].RealtimeLastErrorCode != "realtime.sequence_gap" {
		t.Fatalf("realtime protocol QueryRuns additive parse mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunRealtimeProtocolQueryRunsParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.AddRun(RunRecord{
		Time:      time.Now(),
		RunID:     "run-realtime-protocol-compat",
		Status:    "success",
		LatencyMs: 21,
	})

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-realtime-protocol-compat"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("QueryRuns items len=%d, want 1", len(page.Items))
	}
	got := page.Items[0]
	if got.Status != "success" || got.LatencyMs != 21 {
		t.Fatalf("existing fields mismatch: %#v", got)
	}
	if got.RealtimeProtocolVersion != "" ||
		got.RealtimeEventSeqMax != 0 ||
		got.RealtimeInterruptTotal != 0 ||
		got.RealtimeResumeTotal != 0 ||
		got.RealtimeResumeSource != "" ||
		got.RealtimeIdempotencyDedupTotal != 0 ||
		got.RealtimeLastErrorCode != "" {
		t.Fatalf("missing realtime protocol additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestStoreRunRuntimeBudgetAdmissionAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:           time.Now(),
		RunID:          "run-a60-budget-admission",
		Status:         "success",
		BudgetDecision: "degrade",
		DegradeAction:  "trim_memory_context",
		BudgetSnapshot: map[string]any{
			"version": "budget_admission.v1",
			"cost_estimate": map[string]any{
				"token":   0.20,
				"tool":    0.15,
				"sandbox": 0.10,
				"memory":  0.08,
				"total":   0.53,
			},
			"latency_estimate": map[string]any{
				"token_ms":   int64(220),
				"tool_ms":    int64(180),
				"sandbox_ms": int64(140),
				"memory_ms":  int64(90),
				"total_ms":   int64(630),
			},
		},
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	if items[0].BudgetDecision != "degrade" ||
		items[0].DegradeAction != "trim_memory_context" ||
		items[0].BudgetSnapshot["version"] != "budget_admission.v1" {
		t.Fatalf("runtime budget admission additive fields mismatch after dedup: %#v", items[0])
	}

	rec.BudgetDecision = "deny"
	rec.DegradeAction = ""
	rec.BudgetSnapshot = map[string]any{
		"version": "budget_admission.v1",
		"cost_estimate": map[string]any{
			"token":   0.42,
			"tool":    0.33,
			"sandbox": 0.25,
			"memory":  0.11,
			"total":   1.11,
		},
		"latency_estimate": map[string]any{
			"token_ms":   int64(640),
			"tool_ms":    int64(520),
			"sandbox_ms": int64(480),
			"memory_ms":  int64(210),
			"total_ms":   int64(1850),
		},
	}
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	if items[0].BudgetDecision != "deny" ||
		items[0].DegradeAction != "" ||
		items[0].BudgetSnapshot["version"] != "budget_admission.v1" {
		t.Fatalf("runtime budget admission additive fields mismatch after replay replacement: %#v", items[0])
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a60-budget-admission"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len=%d, want 1", len(page.Items))
	}
	if page.Items[0].BudgetDecision != "deny" ||
		page.Items[0].DegradeAction != "" ||
		page.Items[0].BudgetSnapshot["version"] != "budget_admission.v1" {
		t.Fatalf("runtime budget admission query mapping mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunTracingEvalAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:               time.Now(),
		RunID:              "run-a61-tracing-eval",
		Status:             "success",
		TraceExportStatus:  "degraded",
		TraceSchemaVersion: "otel_semconv.v1",
		EvalSuiteID:        "agent_eval.v1",
		EvalSummary: map[string]any{
			"task_success":     map[string]any{"pass_rate": 0.92},
			"tool_correctness": map[string]any{"pass_rate": 0.88},
		},
		EvalExecutionMode: "distributed",
		EvalJobID:         "eval-job-a61",
		EvalShardTotal:    4,
		EvalResumeCount:   1,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if got.TraceExportStatus != "degraded" ||
		got.TraceSchemaVersion != "otel_semconv.v1" ||
		got.EvalSuiteID != "agent_eval.v1" ||
		got.EvalExecutionMode != "distributed" ||
		got.EvalJobID != "eval-job-a61" ||
		got.EvalShardTotal != 4 ||
		got.EvalResumeCount != 1 {
		t.Fatalf("tracing+eval additive fields mismatch after dedup: %#v", got)
	}
	if summary, ok := got.EvalSummary["task_success"].(map[string]any); !ok || summary["pass_rate"] != 0.92 {
		t.Fatalf("tracing+eval eval_summary parse mismatch after dedup: %#v", got.EvalSummary)
	}

	rec.TraceExportStatus = "failed"
	rec.EvalExecutionMode = "local"
	rec.EvalResumeCount = 0
	rec.EvalSummary = map[string]any{
		"task_success": map[string]any{"pass_rate": 0.81},
	}
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	got = items[0]
	if got.TraceExportStatus != "failed" ||
		got.EvalExecutionMode != "local" ||
		got.EvalResumeCount != 0 {
		t.Fatalf("tracing+eval additive fields mismatch after replay replacement: %#v", got)
	}
	if summary, ok := got.EvalSummary["task_success"].(map[string]any); !ok || summary["pass_rate"] != 0.81 {
		t.Fatalf("tracing+eval eval_summary parse mismatch after replay replacement: %#v", got.EvalSummary)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a61-tracing-eval"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len=%d, want 1", len(page.Items))
	}
	if page.Items[0].TraceExportStatus != "failed" ||
		page.Items[0].EvalExecutionMode != "local" {
		t.Fatalf("tracing+eval query mapping mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunTracingEvalAdditiveFieldsBoundedCardinality(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.SetCardinalityConfig(CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  2,
		MaxListEntries: 2,
		MaxStringBytes: 24,
		OverflowPolicy: CardinalityOverflowTruncateAndRecord,
	})
	rec := RunRecord{
		Time:               time.Now(),
		RunID:              "run-a61-tracing-eval-cardinality",
		Status:             "failed",
		TraceExportStatus:  "collector_authentication_failed_and_retried",
		TraceSchemaVersion: "otel_semconv.v1.with.future.patch.suffix",
		EvalExecutionMode:  "distributed-with-unbounded-hint",
		EvalSummary: map[string]any{
			"a": 1,
			"b": 2,
			"c": 3,
		},
	}
	d.AddRun(rec)

	items := d.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if len([]byte(got.TraceExportStatus)) > 24 ||
		len([]byte(got.TraceSchemaVersion)) > 24 ||
		len([]byte(got.EvalExecutionMode)) > 24 {
		t.Fatalf("tracing+eval string fields should be bounded by cardinality config, got %#v", got)
	}
	if len(got.EvalSummary) > 2 {
		t.Fatalf("tracing+eval eval_summary map should be bounded by cardinality config, got %#v", got.EvalSummary)
	}
	if !strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "trace_export_status") ||
		!strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "trace_schema_version") ||
		!strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "eval_execution_mode") ||
		!strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "eval_summary") {
		t.Fatalf("tracing+eval bounded-cardinality summary missing expected fields: %#v", got.DiagnosticsCardinalityTruncatedFieldSummary)
	}
}

func TestStoreRunStateSnapshotRestoreAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                     time.Now(),
		RunID:                    "run-a66-snapshot-restore",
		Status:                   "failed",
		StateSnapshotVersion:     "state_session_snapshot.v1",
		StateRestoreAction:       "compatible_bounded_restore",
		StateRestoreConflictCode: "snapshot_memory_search_policy_mismatch",
		StateRestoreSource:       "composer",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if got.StateSnapshotVersion != "state_session_snapshot.v1" ||
		got.StateRestoreAction != "compatible_bounded_restore" ||
		got.StateRestoreConflictCode != "snapshot_memory_search_policy_mismatch" ||
		got.StateRestoreSource != "composer" {
		t.Fatalf("state snapshot restore additive fields mismatch after dedup: %#v", got)
	}

	rec.StateRestoreAction = "strict_exact_restore"
	rec.StateRestoreConflictCode = "state_snapshot_strict_incompatible"
	rec.StateRestoreSource = "scheduler"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	got = items[0]
	if got.StateRestoreAction != "strict_exact_restore" ||
		got.StateRestoreConflictCode != "state_snapshot_strict_incompatible" ||
		got.StateRestoreSource != "scheduler" {
		t.Fatalf("state snapshot restore additive fields mismatch after replay replacement: %#v", got)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a66-snapshot-restore"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len=%d, want 1", len(page.Items))
	}
	if page.Items[0].StateSnapshotVersion != "state_session_snapshot.v1" ||
		page.Items[0].StateRestoreAction != "strict_exact_restore" ||
		page.Items[0].StateRestoreConflictCode != "state_snapshot_strict_incompatible" ||
		page.Items[0].StateRestoreSource != "scheduler" {
		t.Fatalf("state snapshot restore query mapping mismatch: %#v", page.Items[0])
	}
}

func TestStoreRunStateSnapshotQueryRunsParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.AddRun(RunRecord{
		Time:      time.Now(),
		RunID:     "run-a66-compat",
		Status:    "success",
		LatencyMs: 19,
	})

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a66-compat"})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len=%d, want 1", len(page.Items))
	}
	got := page.Items[0]
	if got.Status != "success" || got.LatencyMs != 19 {
		t.Fatalf("existing fields should stay unchanged: %#v", got)
	}
	if got.StateSnapshotVersion != "" ||
		got.StateRestoreAction != "" ||
		got.StateRestoreConflictCode != "" ||
		got.StateRestoreSource != "" {
		t.Fatalf("missing state snapshot restore additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestStoreRunArbitrationVersionGovernanceAdditiveFieldsReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                                   time.Now(),
		RunID:                                  "run-a50-governance",
		Status:                                 "failed",
		RuntimePrimaryDomain:                   "runtime",
		RuntimePrimaryCode:                     "runtime.arbitration.version.unsupported",
		RuntimePrimarySource:                   "runtime.arbitration.version",
		RuntimeArbitrationRuleVersion:          "",
		RuntimeArbitrationRuleRequestedVersion: "a77.v9",
		RuntimeArbitrationRuleEffectiveVersion: "",
		RuntimeArbitrationRuleVersionSource:    "requested",
		RuntimeArbitrationRulePolicyAction:     "fail_fast_unsupported_version",
		RuntimeArbitrationRuleUnsupportedTotal: 1,
		RuntimeArbitrationRuleMismatchTotal:    0,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	if items[0].RuntimeArbitrationRuleRequestedVersion != "a77.v9" ||
		items[0].RuntimeArbitrationRuleEffectiveVersion != "" ||
		items[0].RuntimeArbitrationRuleVersionSource != "requested" ||
		items[0].RuntimeArbitrationRulePolicyAction != "fail_fast_unsupported_version" ||
		items[0].RuntimeArbitrationRuleUnsupportedTotal != 1 ||
		items[0].RuntimeArbitrationRuleMismatchTotal != 0 {
		t.Fatalf("a50 governance fields mismatch after replay dedup: %#v", items[0])
	}

	rec.RuntimePrimaryCode = "runtime.arbitration.version.compatibility_mismatch"
	rec.RuntimeArbitrationRuleRequestedVersion = "a48.v1"
	rec.RuntimeArbitrationRuleVersionSource = "requested"
	rec.RuntimeArbitrationRulePolicyAction = "fail_fast_version_mismatch"
	rec.RuntimeArbitrationRuleUnsupportedTotal = 0
	rec.RuntimeArbitrationRuleMismatchTotal = 1
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	if items[0].RuntimePrimaryCode != "runtime.arbitration.version.compatibility_mismatch" ||
		items[0].RuntimeArbitrationRuleRequestedVersion != "a48.v1" ||
		items[0].RuntimeArbitrationRulePolicyAction != "fail_fast_version_mismatch" ||
		items[0].RuntimeArbitrationRuleUnsupportedTotal != 0 ||
		items[0].RuntimeArbitrationRuleMismatchTotal != 1 {
		t.Fatalf("a50 governance fields mismatch after replay replacement: %#v", items[0])
	}
}

func TestStoreRunMemoryAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                     time.Now(),
		RunID:                    "run-a54-memory",
		Status:                   "success",
		MemoryMode:               "external_spi",
		MemoryProvider:           "mem0",
		MemoryProfile:            "mem0",
		MemoryContractVersion:    "memory.v1",
		MemoryQueryTotal:         3,
		MemoryUpsertTotal:        1,
		MemoryDeleteTotal:        0,
		MemoryErrorTotal:         1,
		MemoryFallbackTotal:      1,
		MemoryFallbackReasonCode: "memory.fallback.used",
		MemoryLatencyMsP95:       25,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	if items[0].MemoryMode != "external_spi" ||
		items[0].MemoryProvider != "mem0" ||
		items[0].MemoryProfile != "mem0" ||
		items[0].MemoryContractVersion != "memory.v1" ||
		items[0].MemoryQueryTotal != 3 ||
		items[0].MemoryUpsertTotal != 1 ||
		items[0].MemoryDeleteTotal != 0 ||
		items[0].MemoryErrorTotal != 1 ||
		items[0].MemoryFallbackTotal != 1 ||
		items[0].MemoryFallbackReasonCode != "memory.fallback.used" ||
		items[0].MemoryLatencyMsP95 != 25 {
		t.Fatalf("memory diagnostics fields mismatch after dedup: %#v", items[0])
	}

	rec.MemoryProvider = "zep"
	rec.MemoryProfile = "zep"
	rec.MemoryQueryTotal = 5
	rec.MemoryUpsertTotal = 2
	rec.MemoryErrorTotal = 0
	rec.MemoryFallbackTotal = 0
	rec.MemoryFallbackReasonCode = ""
	rec.MemoryLatencyMsP95 = 19
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	if items[0].MemoryProvider != "zep" ||
		items[0].MemoryProfile != "zep" ||
		items[0].MemoryQueryTotal != 5 ||
		items[0].MemoryUpsertTotal != 2 ||
		items[0].MemoryErrorTotal != 0 ||
		items[0].MemoryFallbackTotal != 0 ||
		items[0].MemoryFallbackReasonCode != "" ||
		items[0].MemoryLatencyMsP95 != 19 {
		t.Fatalf("memory diagnostics fields mismatch after replay replacement: %#v", items[0])
	}
}

func TestStoreRunObservabilityAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                               time.Now(),
		RunID:                              "run-a55-observability",
		Status:                             "failed",
		ObservabilityExportProfile:         "otlp",
		ObservabilityExportStatus:          "degraded",
		ObservabilityExportErrorTotal:      2,
		ObservabilityExportDropTotal:       1,
		ObservabilityExportQueueDepthPeak:  32,
		DiagnosticsBundleTotal:             1,
		DiagnosticsBundleLastStatus:        "failed",
		DiagnosticsBundleLastReasonCode:    "diagnostics.bundle.output_unavailable",
		DiagnosticsBundleLastSchemaVersion: "bundle.v1",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	if items[0].ObservabilityExportProfile != "otlp" ||
		items[0].ObservabilityExportStatus != "degraded" ||
		items[0].ObservabilityExportErrorTotal != 2 ||
		items[0].ObservabilityExportDropTotal != 1 ||
		items[0].ObservabilityExportQueueDepthPeak != 32 ||
		items[0].DiagnosticsBundleTotal != 1 ||
		items[0].DiagnosticsBundleLastStatus != "failed" ||
		items[0].DiagnosticsBundleLastReasonCode != "diagnostics.bundle.output_unavailable" ||
		items[0].DiagnosticsBundleLastSchemaVersion != "bundle.v1" {
		t.Fatalf("observability diagnostics fields mismatch after dedup: %#v", items[0])
	}

	rec.ObservabilityExportProfile = "langfuse"
	rec.ObservabilityExportStatus = "failed"
	rec.ObservabilityExportErrorTotal = 3
	rec.ObservabilityExportDropTotal = 4
	rec.ObservabilityExportQueueDepthPeak = 17
	rec.DiagnosticsBundleTotal = 2
	rec.DiagnosticsBundleLastStatus = "degraded"
	rec.DiagnosticsBundleLastReasonCode = "diagnostics.bundle.policy_invalid"
	rec.DiagnosticsBundleLastSchemaVersion = "bundle.v2"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	if items[0].ObservabilityExportProfile != "langfuse" ||
		items[0].ObservabilityExportStatus != "failed" ||
		items[0].ObservabilityExportErrorTotal != 3 ||
		items[0].ObservabilityExportDropTotal != 4 ||
		items[0].ObservabilityExportQueueDepthPeak != 17 ||
		items[0].DiagnosticsBundleTotal != 2 ||
		items[0].DiagnosticsBundleLastStatus != "degraded" ||
		items[0].DiagnosticsBundleLastReasonCode != "diagnostics.bundle.policy_invalid" ||
		items[0].DiagnosticsBundleLastSchemaVersion != "bundle.v2" {
		t.Fatalf("observability diagnostics fields mismatch after replay replacement: %#v", items[0])
	}
}

func TestStoreRunObservabilityAdditiveFieldsBoundedCardinality(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.SetCardinalityConfig(CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  8,
		MaxListEntries: 8,
		MaxStringBytes: 24,
		OverflowPolicy: CardinalityOverflowTruncateAndRecord,
	})
	rec := RunRecord{
		Time:                               time.Now(),
		RunID:                              "run-a55-observability-cardinality",
		Status:                             "failed",
		ObservabilityExportProfile:         "otlp-with-long-tenant-suffix-for-cardinality",
		ObservabilityExportStatus:          "degraded",
		DiagnosticsBundleLastStatus:        "failed",
		DiagnosticsBundleLastReasonCode:    "diagnostics.bundle.backend_private_reason_code_should_be_bounded",
		DiagnosticsBundleLastSchemaVersion: "bundle.v1.with.unbounded.suffix",
	}
	d.AddRun(rec)

	items := d.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if len([]byte(got.ObservabilityExportProfile)) > 24 ||
		len([]byte(got.DiagnosticsBundleLastReasonCode)) > 24 ||
		len([]byte(got.DiagnosticsBundleLastSchemaVersion)) > 24 {
		t.Fatalf("observability diagnostics string fields should be bounded by cardinality config, got %#v", got)
	}
	if !strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "observability_export_profile") ||
		!strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "diagnostics_bundle_last_reason_code") ||
		!strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "diagnostics_bundle_last_schema_version") {
		t.Fatalf("observability diagnostics bounded-cardinality summary missing expected fields: %#v", got.DiagnosticsCardinalityTruncatedFieldSummary)
	}
}

func TestStoreRunReactAdditiveFieldsBoundedCardinality(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.SetCardinalityConfig(CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  8,
		MaxListEntries: 8,
		MaxStringBytes: 24,
		OverflowPolicy: CardinalityOverflowTruncateAndRecord,
	})
	rec := RunRecord{
		Time:                        time.Now(),
		RunID:                       "run-a56-react-cardinality",
		Status:                      "failed",
		ReactEnabled:                true,
		ReactIterationTotal:         5,
		ReactToolCallTotal:          9,
		ReactToolCallBudgetHitTotal: 1,
		ReactTerminationReason:      "react.tool_call_limit_exceeded.with.private.suffix",
		ReactStreamDispatchEnabled:  true,
	}
	d.AddRun(rec)

	items := d.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if !got.ReactEnabled || !got.ReactStreamDispatchEnabled {
		t.Fatalf("react booleans mismatch: %#v", got)
	}
	if got.ReactIterationTotal != 5 || got.ReactToolCallTotal != 9 || got.ReactToolCallBudgetHitTotal != 1 {
		t.Fatalf("react numeric fields mismatch: %#v", got)
	}
	if len([]byte(got.ReactTerminationReason)) > 24 {
		t.Fatalf("react_termination_reason should be bounded by cardinality config, got %q", got.ReactTerminationReason)
	}
	if !strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "react_termination_reason") {
		t.Fatalf("missing react_termination_reason bounded summary: %#v", got.DiagnosticsCardinalityTruncatedFieldSummary)
	}
}

func TestStoreRunPolicyPrecedenceAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                    time.Now(),
		RunID:                   "run-a58-policy",
		Status:                  "failed",
		PolicyPrecedenceVersion: "policy_stack.v1",
		WinnerStage:             "sandbox_action",
		DenySource:              "sandbox_action",
		TieBreakReason:          "lexical_code_then_source_order",
		PolicyDecisionPath: []RuntimePolicyDecisionPathEntry{
			{Stage: "action_gate", Code: "action.gate.allow", Source: "action_gate", Decision: "allow"},
			{Stage: "sandbox_action", Code: "runtime.readiness.admission.sandbox_capacity_deny", Source: "sandbox_action", Decision: "deny"},
			{Stage: "readiness_admission", Code: "runtime.readiness.admission.blocked", Source: "readiness_admission", Decision: "deny"},
		},
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if got.PolicyPrecedenceVersion != "policy_stack.v1" ||
		got.WinnerStage != "sandbox_action" ||
		got.DenySource != "sandbox_action" ||
		got.TieBreakReason != "lexical_code_then_source_order" ||
		len(got.PolicyDecisionPath) != 3 ||
		got.PolicyDecisionPath[1].Stage != "sandbox_action" {
		t.Fatalf("policy precedence fields mismatch: %#v", got)
	}

	rec.PolicyDecisionPath = []RuntimePolicyDecisionPathEntry{
		{Stage: "sandbox_action", Code: "runtime.readiness.admission.sandbox_capacity_deny", Source: "sandbox_action", Decision: "deny"},
	}
	rec.DenySource = "sandbox_action"
	d.AddRun(rec)
	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replacement, want 1", len(items))
	}
	if len(items[0].PolicyDecisionPath) != 1 || items[0].PolicyDecisionPath[0].Stage != "sandbox_action" {
		t.Fatalf("policy_decision_path should update on idempotent replacement: %#v", items[0].PolicyDecisionPath)
	}
}

func TestStoreRunPolicyPrecedenceAdditiveFieldsBoundedCardinality(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	d.SetCardinalityConfig(CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  8,
		MaxListEntries: 1,
		MaxStringBytes: 16,
		OverflowPolicy: CardinalityOverflowTruncateAndRecord,
	})
	rec := RunRecord{
		Time:                    time.Now(),
		RunID:                   "run-a58-policy-cardinality",
		Status:                  "failed",
		PolicyPrecedenceVersion: "policy_stack.v1.with.long.suffix",
		WinnerStage:             "sandbox_action_with_private_suffix",
		DenySource:              "sandbox_action_with_private_suffix",
		TieBreakReason:          "lexical_code_then_source_order_with_private_suffix",
		PolicyDecisionPath: []RuntimePolicyDecisionPathEntry{
			{Stage: "action_gate", Code: "action.gate.allow.with.private.suffix", Source: "action_gate", Decision: "allow"},
			{Stage: "sandbox_action", Code: "runtime.readiness.admission.sandbox_capacity_deny.with.private.suffix", Source: "sandbox_action", Decision: "deny"},
		},
	}
	d.AddRun(rec)

	items := d.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if len(got.PolicyDecisionPath) != 1 {
		t.Fatalf("policy_decision_path should be truncated to 1 entry, got %#v", got.PolicyDecisionPath)
	}
	if len([]byte(got.PolicyPrecedenceVersion)) > 16 ||
		len([]byte(got.WinnerStage)) > 16 ||
		len([]byte(got.DenySource)) > 16 ||
		len([]byte(got.TieBreakReason)) > 16 {
		t.Fatalf("policy precedence string fields should be bounded by cardinality config, got %#v", got)
	}
	if !strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "policy_decision_path") ||
		!strings.Contains(got.DiagnosticsCardinalityTruncatedFieldSummary, "policy_precedence_version") {
		t.Fatalf("policy precedence bounded-cardinality summary missing expected fields: %#v", got.DiagnosticsCardinalityTruncatedFieldSummary)
	}
}

func TestStoreRunSandboxAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                              time.Now(),
		RunID:                             "run-a51-sandbox",
		Status:                            "failed",
		SandboxMode:                       "enforce",
		SandboxBackend:                    "windows_job",
		SandboxProfile:                    "default",
		SandboxSessionMode:                "per_call",
		SandboxRequiredCapabilities:       []string{"stdout_stderr_capture", "oom_signal"},
		SandboxDecision:                   "deny",
		SandboxReasonCode:                 "sandbox.timeout",
		SandboxFallbackUsed:               true,
		SandboxFallbackReason:             "sandbox.fallback_allow_and_record",
		SandboxTimeoutTotal:               1,
		SandboxLaunchFailedTotal:          2,
		SandboxCapabilityMismatchTotal:    3,
		SandboxQueueWaitMsP95:             5,
		SandboxExecLatencyMsP95:           7,
		SandboxExitCodeLast:               137,
		SandboxOOMTotal:                   1,
		SandboxResourceCPUMsTotal:         250,
		SandboxResourceMemoryPeakBytesP95: 2048,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	items := d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d, want 1", len(items))
	}
	got := items[0]
	if got.SandboxMode != "enforce" ||
		got.SandboxBackend != "windows_job" ||
		got.SandboxProfile != "default" ||
		got.SandboxSessionMode != "per_call" ||
		len(got.SandboxRequiredCapabilities) != 2 ||
		got.SandboxRequiredCapabilities[0] != "stdout_stderr_capture" ||
		got.SandboxRequiredCapabilities[1] != "oom_signal" ||
		got.SandboxDecision != "deny" ||
		got.SandboxReasonCode != "sandbox.timeout" ||
		!got.SandboxFallbackUsed ||
		got.SandboxFallbackReason != "sandbox.fallback_allow_and_record" ||
		got.SandboxTimeoutTotal != 1 ||
		got.SandboxLaunchFailedTotal != 2 ||
		got.SandboxCapabilityMismatchTotal != 3 ||
		got.SandboxQueueWaitMsP95 != 5 ||
		got.SandboxExecLatencyMsP95 != 7 ||
		got.SandboxExitCodeLast != 137 ||
		got.SandboxOOMTotal != 1 ||
		got.SandboxResourceCPUMsTotal != 250 ||
		got.SandboxResourceMemoryPeakBytesP95 != 2048 {
		t.Fatalf("sandbox fields mismatch after replay dedup: %#v", got)
	}

	rec.SandboxDecision = "sandbox"
	rec.SandboxReasonCode = "sandbox.launch_failed"
	rec.SandboxFallbackUsed = false
	rec.SandboxFallbackReason = ""
	rec.SandboxTimeoutTotal = 0
	rec.SandboxLaunchFailedTotal = 4
	rec.SandboxCapabilityMismatchTotal = 1
	rec.SandboxExecLatencyMsP95 = 9
	rec.SandboxResourceCPUMsTotal = 420
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay replacement, want 1", len(items))
	}
	got = items[0]
	if got.SandboxDecision != "sandbox" ||
		got.SandboxReasonCode != "sandbox.launch_failed" ||
		got.SandboxFallbackUsed ||
		got.SandboxTimeoutTotal != 0 ||
		got.SandboxLaunchFailedTotal != 4 ||
		got.SandboxCapabilityMismatchTotal != 1 ||
		got.SandboxExecLatencyMsP95 != 9 ||
		got.SandboxResourceCPUMsTotal != 420 {
		t.Fatalf("sandbox fields mismatch after replay replacement: %#v", got)
	}

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a51-sandbox"})
	if err != nil {
		t.Fatalf("query runs failed: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].SandboxReasonCode != "sandbox.launch_failed" {
		t.Fatalf("sandbox query mapping mismatch: %#v", page.Items)
	}
}

func TestStoreRunTeamsAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                 time.Now(),
		RunID:                "run-team-1",
		Status:               "failed",
		TeamID:               "team-alpha",
		TeamStrategy:         "parallel",
		TeamTaskTotal:        5,
		TeamTaskFailed:       2,
		TeamTaskCanceled:     1,
		TeamRemoteTaskTotal:  3,
		TeamRemoteTaskFailed: 1,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].TeamTaskTotal != 5 || runs[0].TeamTaskFailed != 2 || runs[0].TeamTaskCanceled != 1 ||
		runs[0].TeamRemoteTaskTotal != 3 || runs[0].TeamRemoteTaskFailed != 1 {
		t.Fatalf("team aggregate should stay stable under replay, got %#v", runs[0])
	}
}

func TestStoreRunWorkflowAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                           time.Now(),
		RunID:                          "run-workflow-1",
		Status:                         "failed",
		WorkflowID:                     "wf-alpha",
		WorkflowStatus:                 "failed",
		WorkflowStepTotal:              6,
		WorkflowStepFailed:             2,
		WorkflowRemoteStepTotal:        2,
		WorkflowRemoteStepFailed:       1,
		WorkflowSubgraphExpansionTotal: 3,
		WorkflowConditionTemplateTotal: 2,
		WorkflowGraphCompileFailed:     false,
		WorkflowResumeCount:            1,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].WorkflowStepTotal != 6 || runs[0].WorkflowStepFailed != 2 ||
		runs[0].WorkflowRemoteStepTotal != 2 || runs[0].WorkflowRemoteStepFailed != 1 ||
		runs[0].WorkflowSubgraphExpansionTotal != 3 || runs[0].WorkflowConditionTemplateTotal != 2 ||
		runs[0].WorkflowGraphCompileFailed || runs[0].WorkflowResumeCount != 1 {
		t.Fatalf("workflow aggregate should stay stable under replay, got %#v", runs[0])
	}
}

func TestStoreRunA2AAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                        time.Now(),
		RunID:                       "run-a2a-1",
		Status:                      "failed",
		A2ATaskTotal:                4,
		A2ATaskFailed:               1,
		PeerID:                      "peer-1",
		A2AErrorLayer:               "transport",
		A2ADeliveryMode:             "sse",
		A2ADeliveryFallbackUsed:     true,
		A2ADeliveryFallbackReason:   "a2a.delivery_unsupported",
		A2AVersionLocal:             "a2a.v1.2",
		A2AVersionPeer:              "a2a.v1.0",
		A2AVersionNegotiationResult: "compatible",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].A2ATaskTotal != 4 || runs[0].A2ATaskFailed != 1 {
		t.Fatalf("a2a aggregate should stay stable under replay, got %#v", runs[0])
	}
	if runs[0].PeerID != "peer-1" || runs[0].A2AErrorLayer != "transport" {
		t.Fatalf("a2a fields mismatch under replay, got %#v", runs[0])
	}
	if runs[0].A2ADeliveryMode != "sse" || !runs[0].A2ADeliveryFallbackUsed || runs[0].A2ADeliveryFallbackReason != "a2a.delivery_unsupported" {
		t.Fatalf("a2a delivery fields mismatch under replay, got %#v", runs[0])
	}
	if runs[0].A2AVersionLocal != "a2a.v1.2" || runs[0].A2AVersionPeer != "a2a.v1.0" || runs[0].A2AVersionNegotiationResult != "compatible" {
		t.Fatalf("a2a version fields mismatch under replay, got %#v", runs[0])
	}
}

func TestStoreRunAsyncDelayedAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                       time.Now(),
		RunID:                      "run-a14-async-delayed",
		Status:                     "success",
		A2AAsyncReportTotal:        3,
		A2AAsyncReportFailed:       1,
		A2AAsyncReportRetryTotal:   2,
		A2AAsyncReportDedupTotal:   1,
		AsyncAwaitTotal:            2,
		AsyncTimeoutTotal:          1,
		AsyncLateReportTotal:       1,
		AsyncReportDedupTotal:      1,
		SchedulerDelayedTaskTotal:  2,
		SchedulerDelayedClaimTotal: 2,
		SchedulerDelayedWaitMsP95:  180,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].A2AAsyncReportTotal != 3 ||
		runs[0].A2AAsyncReportFailed != 1 ||
		runs[0].A2AAsyncReportRetryTotal != 2 ||
		runs[0].A2AAsyncReportDedupTotal != 1 ||
		runs[0].AsyncAwaitTotal != 2 ||
		runs[0].AsyncTimeoutTotal != 1 ||
		runs[0].AsyncLateReportTotal != 1 ||
		runs[0].AsyncReportDedupTotal != 1 ||
		runs[0].SchedulerDelayedTaskTotal != 2 ||
		runs[0].SchedulerDelayedClaimTotal != 2 ||
		runs[0].SchedulerDelayedWaitMsP95 != 180 {
		t.Fatalf("combined async+delayed aggregate should stay stable under replay, got %#v", runs[0])
	}
}

func TestStoreRunSchedulerSubagentAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                           time.Now(),
		RunID:                          "run-scheduler-1",
		Status:                         "success",
		SchedulerBackend:               "file",
		SchedulerQueueTotal:            3,
		SchedulerClaimTotal:            4,
		SchedulerReclaimTotal:          1,
		SchedulerDelayedTaskTotal:      2,
		SchedulerDelayedClaimTotal:     2,
		SchedulerDelayedWaitMsP95:      180,
		SubagentChildTotal:             2,
		SubagentChildFailed:            1,
		SubagentBudgetRejectTotal:      1,
		EffectiveOperationProfile:      "interactive",
		TimeoutResolutionSource:        "request",
		TimeoutResolutionTrace:         `{"version":"v1","selected_source":"request"}`,
		TimeoutParentBudgetClampTotal:  1,
		TimeoutParentBudgetRejectTotal: 0,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].SchedulerBackend != "file" ||
		runs[0].SchedulerQueueTotal != 3 ||
		runs[0].SchedulerClaimTotal != 4 ||
		runs[0].SchedulerReclaimTotal != 1 ||
		runs[0].SchedulerDelayedTaskTotal != 2 ||
		runs[0].SchedulerDelayedClaimTotal != 2 ||
		runs[0].SchedulerDelayedWaitMsP95 != 180 ||
		runs[0].SubagentChildTotal != 2 ||
		runs[0].SubagentChildFailed != 1 ||
		runs[0].SubagentBudgetRejectTotal != 1 ||
		runs[0].EffectiveOperationProfile != "interactive" ||
		runs[0].TimeoutResolutionSource != "request" ||
		runs[0].TimeoutResolutionTrace == "" ||
		runs[0].TimeoutParentBudgetClampTotal != 1 ||
		runs[0].TimeoutParentBudgetRejectTotal != 0 {
		t.Fatalf("scheduler/subagent fields mismatch under replay, got %#v", runs[0])
	}
}

func TestStoreRunTimeoutResolutionFieldsQueryCompatibility(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                           time.Now(),
		RunID:                          "run-a41-query",
		Status:                         "success",
		EffectiveOperationProfile:      "background",
		TimeoutResolutionSource:        "domain",
		TimeoutResolutionTrace:         `{"version":"v1","selected_source":"domain"}`,
		TimeoutParentBudgetClampTotal:  2,
		TimeoutParentBudgetRejectTotal: 1,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	page, err := d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a41-query"})
	if err != nil {
		t.Fatalf("query runs failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len = %d, want 1", len(page.Items))
	}
	got := page.Items[0]
	if got.EffectiveOperationProfile != "background" ||
		got.TimeoutResolutionSource != "domain" ||
		got.TimeoutResolutionTrace == "" ||
		got.TimeoutParentBudgetClampTotal != 2 ||
		got.TimeoutParentBudgetRejectTotal != 1 {
		t.Fatalf("timeout resolution query fields mismatch: %#v", got)
	}
}

func TestStoreRunRecoveryAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                                 time.Now(),
		RunID:                                "run-recovery-1",
		Status:                               "success",
		RecoveryEnabled:                      true,
		RecoveryResumeBoundary:               "next_attempt_only",
		RecoveryInflightPolicy:               "no_rewind",
		RecoveryRecovered:                    true,
		RecoveryReplayTotal:                  2,
		RecoveryTimeoutReentryTotal:          1,
		RecoveryTimeoutReentryExhaustedTotal: 1,
		RecoveryConflict:                     false,
		RecoveryFallbackUsed:                 true,
		RecoveryFallbackReason:               "recovery.backend.file_init_failed",
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if !runs[0].RecoveryEnabled || !runs[0].RecoveryRecovered || runs[0].RecoveryReplayTotal != 2 {
		t.Fatalf("recovery aggregate mismatch under replay, got %#v", runs[0])
	}
	if runs[0].RecoveryResumeBoundary != "next_attempt_only" ||
		runs[0].RecoveryInflightPolicy != "no_rewind" ||
		runs[0].RecoveryTimeoutReentryTotal != 1 ||
		runs[0].RecoveryTimeoutReentryExhaustedTotal != 1 {
		t.Fatalf("recovery boundary aggregate mismatch under replay, got %#v", runs[0])
	}
	if !runs[0].RecoveryFallbackUsed || runs[0].RecoveryFallbackReason != "recovery.backend.file_init_failed" {
		t.Fatalf("recovery fallback mismatch under replay, got %#v", runs[0])
	}
}

func TestStoreRunCollabAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                      time.Now(),
		RunID:                     "run-collab-1",
		Status:                    "success",
		CollabHandoffTotal:        1,
		CollabDelegationTotal:     2,
		CollabAggregationTotal:    2,
		CollabAggregationStrategy: "all_settled",
		CollabFailFastTotal:       1,
		CollabRetryTotal:          3,
		CollabRetrySuccessTotal:   1,
		CollabRetryExhaustedTotal: 1,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].CollabHandoffTotal != 1 ||
		runs[0].CollabDelegationTotal != 2 ||
		runs[0].CollabAggregationTotal != 2 ||
		runs[0].CollabAggregationStrategy != "all_settled" ||
		runs[0].CollabFailFastTotal != 1 ||
		runs[0].CollabRetryTotal != 3 ||
		runs[0].CollabRetrySuccessTotal != 1 ||
		runs[0].CollabRetryExhaustedTotal != 1 {
		t.Fatalf("collab aggregate mismatch under replay, got %#v", runs[0])
	}
}

func TestStoreUnifiedRunQueryAndSemantics(t *testing.T) {
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now()
	d.AddRun(RunRecord{
		Time:       base.Add(1 * time.Second),
		RunID:      "run-a18-1",
		Status:     "success",
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		TaskID:     "task-a18-1",
	})
	d.AddRun(RunRecord{
		Time:       base.Add(2 * time.Second),
		RunID:      "run-a18-2",
		Status:     "failed",
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		TaskID:     "task-a18-2",
	})
	d.AddRun(RunRecord{
		Time:       base.Add(3 * time.Second),
		RunID:      "run-a18-3",
		Status:     "failed",
		TeamID:     "team-a",
		WorkflowID: "wf-b",
		TaskID:     "task-a18-3",
	})

	got, err := d.QueryRuns(UnifiedRunQueryRequest{TeamID: "team-a"})
	if err != nil {
		t.Fatalf("query team filter failed: %v", err)
	}
	if got.PageSize != DefaultUnifiedQueryPageSize {
		t.Fatalf("default page_size = %d, want %d", got.PageSize, DefaultUnifiedQueryPageSize)
	}
	if got.SortField != "time" || got.SortOrder != "desc" {
		t.Fatalf("default sort mismatch: %#v", got)
	}
	if len(got.Items) != 3 {
		t.Fatalf("items len = %d, want 3", len(got.Items))
	}
	if got.Items[0].RunID != "run-a18-3" || got.Items[1].RunID != "run-a18-2" || got.Items[2].RunID != "run-a18-1" {
		t.Fatalf("default time desc order mismatch: %#v", got.Items)
	}

	got, err = d.QueryRuns(UnifiedRunQueryRequest{
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		Status:     "failed",
	})
	if err != nil {
		t.Fatalf("query AND filters failed: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].RunID != "run-a18-2" {
		t.Fatalf("AND filter result mismatch: %#v", got.Items)
	}

	got, err = d.QueryRuns(UnifiedRunQueryRequest{RunID: "run-a18-1"})
	if err != nil {
		t.Fatalf("query single run_id failed: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].RunID != "run-a18-1" {
		t.Fatalf("single run filter mismatch: %#v", got.Items)
	}

	got, err = d.QueryRuns(UnifiedRunQueryRequest{TaskID: "task-a18-missing"})
	if err != nil {
		t.Fatalf("query missing task_id should not error: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("missing task_id should return empty set, got %#v", got.Items)
	}
}

func TestStoreUnifiedRunQueryValidationAndPagingBounds(t *testing.T) {
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now()
	for i := 0; i < 5; i++ {
		d.AddRun(RunRecord{
			Time:       base.Add(time.Duration(i) * time.Second),
			RunID:      "run-a18-page-" + strconv.Itoa(i),
			Status:     "success",
			TeamID:     "team-a",
			WorkflowID: "wf-a",
			TaskID:     "task-a18-page-" + strconv.Itoa(i),
		})
	}

	pageSize2 := 2
	got, err := d.QueryRuns(UnifiedRunQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize2,
	})
	if err != nil {
		t.Fatalf("query with page_size=2 failed: %v", err)
	}
	if len(got.Items) != 2 || got.NextCursor == "" {
		t.Fatalf("page_size=2 pagination mismatch: %#v", got)
	}

	pageSizeTooLarge := 201
	if _, err := d.QueryRuns(UnifiedRunQueryRequest{PageSize: &pageSizeTooLarge}); err == nil {
		t.Fatal("expected fail-fast for page_size > 200")
	}
	pageSizeZero := 0
	if _, err := d.QueryRuns(UnifiedRunQueryRequest{PageSize: &pageSizeZero}); err == nil {
		t.Fatal("expected fail-fast for page_size lower bound")
	}
	pageSizeNegative := -1
	if _, err := d.QueryRuns(UnifiedRunQueryRequest{PageSize: &pageSizeNegative}); err == nil {
		t.Fatal("expected fail-fast for negative page_size")
	}
	if _, err := d.QueryRuns(UnifiedRunQueryRequest{Status: "pending"}); err == nil {
		t.Fatal("expected fail-fast for invalid status filter")
	}
	if _, err := d.QueryRuns(UnifiedRunQueryRequest{
		TimeRange: &UnifiedQueryTimeRange{
			Start: base.Add(10 * time.Second),
			End:   base.Add(1 * time.Second),
		},
	}); err == nil {
		t.Fatal("expected fail-fast for invalid time range")
	}
	if _, err := d.QueryRuns(UnifiedRunQueryRequest{
		Sort: UnifiedQuerySort{Field: "run_id", Order: "desc"},
	}); err == nil {
		t.Fatal("expected fail-fast for unsupported sort field")
	}
}

func TestStoreMailboxReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := MailboxRecord{
		Time:      time.Now(),
		MessageID: "msg-mailbox-1",
		Kind:      "command",
		State:     "queued",
		RunID:     "run-mailbox-1",
		TaskID:    "task-mailbox-1",
	}
	d.AddMailbox(rec)
	d.AddMailbox(rec)

	items := d.RecentMailbox(10)
	if len(items) != 1 {
		t.Fatalf("mailbox records = %d, want 1", len(items))
	}
}

func TestStoreMailboxQueryAndAggregates(t *testing.T) {
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now().UTC()
	d.AddMailbox(MailboxRecord{
		Time:       base.Add(1 * time.Second),
		MessageID:  "msg-a",
		Kind:       "command",
		State:      "queued",
		RunID:      "run-a",
		TaskID:     "task-a",
		WorkflowID: "wf-a",
		TeamID:     "team-a",
	})
	d.AddMailbox(MailboxRecord{
		Time:       base.Add(2 * time.Second),
		MessageID:  "msg-b",
		Kind:       "result",
		State:      "dead_letter",
		RunID:      "run-a",
		TaskID:     "task-a",
		WorkflowID: "wf-a",
		TeamID:     "team-a",
		Attempt:    3,
		ReasonCode: "retry_exhausted",
	})

	page, err := d.QueryMailbox(MailboxQueryRequest{RunID: "run-a"})
	if err != nil {
		t.Fatalf("QueryMailbox failed: %v", err)
	}
	if page.PageSize != DefaultMailboxQueryPageSize {
		t.Fatalf("default mailbox page_size = %d, want %d", page.PageSize, DefaultMailboxQueryPageSize)
	}
	if page.SortField != "time" || page.SortOrder != "desc" {
		t.Fatalf("default mailbox sort mismatch: %#v", page)
	}
	if len(page.Items) != 2 {
		t.Fatalf("mailbox query items len = %d, want 2", len(page.Items))
	}
	for _, rec := range page.Items {
		if rec.RunID == "" || rec.TaskID == "" || rec.WorkflowID == "" || rec.TeamID == "" {
			t.Fatalf("mailbox query should preserve correlation fields: %#v", rec)
		}
	}

	agg := d.MailboxAggregates(MailboxAggregateRequest{RunID: "run-a"})
	if agg.TotalMessages != 2 ||
		agg.ByKind["command"] != 1 ||
		agg.ByKind["result"] != 1 ||
		agg.ByState["dead_letter"] != 1 ||
		agg.ReasonCodeTotals["retry_exhausted"] != 1 ||
		agg.RetryTotal != 2 {
		t.Fatalf("mailbox aggregate mismatch: %#v", agg)
	}
}

func TestStoreMailboxQueryValidationAndCursorDeterminism(t *testing.T) {
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now().UTC()
	for i := 0; i < 3; i++ {
		d.AddMailbox(MailboxRecord{
			Time:      base.Add(time.Duration(i) * time.Second),
			MessageID: "msg-mailbox-page-" + strconv.Itoa(i),
			Kind:      "command",
			State:     "queued",
			RunID:     "run-mailbox-page",
		})
	}

	pageSize := 1
	first, err := d.QueryMailbox(MailboxQueryRequest{
		RunID:    "run-mailbox-page",
		PageSize: &pageSize,
	})
	if err != nil {
		t.Fatalf("first mailbox query failed: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first mailbox page mismatch: %#v", first)
	}
	second, err := d.QueryMailbox(MailboxQueryRequest{
		RunID:    "run-mailbox-page",
		PageSize: &pageSize,
		Cursor:   first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second mailbox query failed: %v", err)
	}
	secondReplay, err := d.QueryMailbox(MailboxQueryRequest{
		RunID:    "run-mailbox-page",
		PageSize: &pageSize,
		Cursor:   first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second mailbox replay query failed: %v", err)
	}
	if len(second.Items) != 1 || len(secondReplay.Items) != 1 || second.Items[0].MessageID != secondReplay.Items[0].MessageID {
		t.Fatalf("mailbox cursor determinism mismatch: second=%#v replay=%#v", second, secondReplay)
	}

	pageSizeTooLarge := 201
	if _, err := d.QueryMailbox(MailboxQueryRequest{PageSize: &pageSizeTooLarge}); err == nil {
		t.Fatal("expected fail-fast for mailbox page_size > 200")
	}
	pageSizeZero := 0
	if _, err := d.QueryMailbox(MailboxQueryRequest{PageSize: &pageSizeZero}); err == nil {
		t.Fatal("expected fail-fast for mailbox page_size <= 0")
	}
	if _, err := d.QueryMailbox(MailboxQueryRequest{State: "running"}); err == nil {
		t.Fatal("expected fail-fast for invalid mailbox state")
	}
	if _, err := d.QueryMailbox(MailboxQueryRequest{
		Sort: MailboxQuerySort{Field: "run_id", Order: "desc"},
	}); err == nil {
		t.Fatal("expected fail-fast for unsupported mailbox sort field")
	}
	if _, err := d.QueryMailbox(MailboxQueryRequest{
		Sort: MailboxQuerySort{Field: "time", Order: "down"},
	}); err == nil {
		t.Fatal("expected fail-fast for unsupported mailbox sort order")
	}
	if _, err := d.QueryMailbox(MailboxQueryRequest{
		TimeRange: &MailboxQueryTimeRange{
			Start: time.Now().Add(2 * time.Minute),
			End:   time.Now().Add(1 * time.Minute),
		},
	}); err == nil {
		t.Fatal("expected fail-fast for invalid mailbox time range")
	}
	if _, err := d.QueryMailbox(MailboxQueryRequest{
		RunID:    "run-mailbox-page",
		PageSize: &pageSize,
		Cursor:   "not-a-valid-cursor",
	}); err == nil {
		t.Fatal("expected fail-fast for malformed mailbox cursor")
	}
	if _, err := d.QueryMailbox(MailboxQueryRequest{
		RunID:    "run-other",
		PageSize: &pageSize,
		Cursor:   first.NextCursor,
	}); err == nil {
		t.Fatal("expected fail-fast for mailbox query boundary mismatch cursor")
	}
}

func TestStoreUnifiedRunQueryCursorDeterministicAndFailFast(t *testing.T) {
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now()
	for i := 0; i < 3; i++ {
		d.AddRun(RunRecord{
			Time:       base.Add(time.Duration(i) * time.Second),
			RunID:      "run-a18-cursor-" + strconv.Itoa(i),
			Status:     "success",
			TeamID:     "team-a",
			WorkflowID: "wf-a",
			TaskID:     "task-a18-cursor-" + strconv.Itoa(i),
		})
	}

	pageSize1 := 1
	req := UnifiedRunQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize1,
	}
	first, err := d.QueryRuns(req)
	if err != nil {
		t.Fatalf("first page query failed: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page mismatch: %#v", first)
	}
	if first.NextCursor == "1" {
		t.Fatalf("cursor must be opaque, got %#v", first.NextCursor)
	}

	second, err := d.QueryRuns(UnifiedRunQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize1,
		Cursor:   first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second page query failed: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("second page mismatch: %#v", second)
	}
	secondAgain, err := d.QueryRuns(UnifiedRunQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize1,
		Cursor:   first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second page replay failed: %v", err)
	}
	if len(secondAgain.Items) != 1 || secondAgain.Items[0].RunID != second.Items[0].RunID {
		t.Fatalf("cursor traversal must be deterministic: %#v vs %#v", secondAgain, second)
	}

	third, err := d.QueryRuns(UnifiedRunQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize1,
		Cursor:   second.NextCursor,
	})
	if err != nil {
		t.Fatalf("third page query failed: %v", err)
	}
	if len(third.Items) != 1 || third.NextCursor != "" {
		t.Fatalf("third page mismatch: %#v", third)
	}

	if _, err := d.QueryRuns(UnifiedRunQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize1,
		Cursor:   "not-a-valid-cursor",
	}); err == nil {
		t.Fatal("expected fail-fast for malformed cursor")
	}
	if _, err := d.QueryRuns(UnifiedRunQueryRequest{
		TeamID:   "team-b",
		PageSize: &pageSize1,
		Cursor:   first.NextCursor,
	}); err == nil {
		t.Fatal("expected fail-fast for cursor query boundary mismatch")
	}
}

func TestStoreSkillDedupConcurrent(t *testing.T) {
	d := NewStore(8, 8, 4, 16, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := SkillRecord{
		Time:       time.Now(),
		RunID:      "run-1",
		SkillName:  "skill-a",
		Action:     "compile",
		Status:     "warning",
		ErrorClass: "ErrSkill",
		Payload: map[string]any{
			"reason": "compile read failed",
			"path":   "/tmp/skill",
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.AddSkill(rec)
		}()
	}
	wg.Wait()

	items := d.RecentSkills(20)
	if len(items) != 1 {
		t.Fatalf("skill records = %d, want 1", len(items))
	}
}

func TestIdempotencyKeyDeterministic(t *testing.T) {
	runA := RunRecord{RunID: "run-1", Status: "success", Iterations: 1}
	runB := RunRecord{RunID: "run-1", Status: "success", Iterations: 99}
	if RunIdempotencyKey(runA) != RunIdempotencyKey(runB) {
		t.Fatalf("run key should be stable for same run/status")
	}

	skillA := SkillRecord{
		RunID:     "run-1",
		SkillName: "a",
		Action:    "compile",
		Status:    "warning",
		Payload:   map[string]any{"reason": "x", "path": "p"},
	}
	skillB := SkillRecord{
		RunID:     "run-1",
		SkillName: "a",
		Action:    "compile",
		Status:    "warning",
		Payload:   map[string]any{"path": "p", "reason": "x"},
	}
	if SkillIdempotencyKey(skillA) != SkillIdempotencyKey(skillB) {
		t.Fatalf("skill key should be deterministic for semantically equal payload")
	}
}

func TestStoreTimelineAggregationWithP95(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now()

	d.AddTimelineEvent("run-1", "model", "running", 1, base)
	d.AddTimelineEvent("run-1", "model", "succeeded", 2, base.Add(10*time.Millisecond))
	d.AddTimelineEvent("run-1", "model", "running", 3, base.Add(20*time.Millisecond))
	d.AddTimelineEvent("run-1", "model", "failed", 4, base.Add(60*time.Millisecond))
	d.AddTimelineEvent("run-1", "model", "running", 5, base.Add(70*time.Millisecond))
	d.AddTimelineEvent("run-1", "model", "canceled", 6, base.Add(170*time.Millisecond))
	d.AddTimelineEvent("run-1", "tool", "skipped", 7, base.Add(180*time.Millisecond))

	d.AddRun(RunRecord{
		Time:      base.Add(190 * time.Millisecond),
		RunID:     "run-1",
		Status:    "success",
		LatencyMs: 190,
	})

	runs := d.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	modelAgg, ok := runs[0].TimelinePhases["model"]
	if !ok {
		t.Fatalf("missing model phase aggregate: %#v", runs[0].TimelinePhases)
	}
	if modelAgg.CountTotal != 3 || modelAgg.FailedTotal != 1 || modelAgg.CanceledTotal != 1 {
		t.Fatalf("unexpected model aggregate counts: %#v", modelAgg)
	}
	if modelAgg.LatencyMs != 150 {
		t.Fatalf("model latency_ms = %d, want 150", modelAgg.LatencyMs)
	}
	if modelAgg.LatencyP95Ms != 100 {
		t.Fatalf("model latency_p95_ms = %d, want 100", modelAgg.LatencyP95Ms)
	}
	toolAgg, ok := runs[0].TimelinePhases["tool"]
	if !ok || toolAgg.SkippedTotal != 1 {
		t.Fatalf("unexpected tool aggregate: %#v", toolAgg)
	}
}

func TestStoreTimelineAggregationIdempotentReplay(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now()

	d.AddTimelineEvent("run-1", "run", "running", 1, base)
	d.AddTimelineEvent("run-1", "run", "succeeded", 2, base.Add(10*time.Millisecond))
	d.AddTimelineEvent("run-1", "run", "running", 1, base)
	d.AddTimelineEvent("run-1", "run", "succeeded", 2, base.Add(10*time.Millisecond))

	d.AddRun(RunRecord{Time: base.Add(20 * time.Millisecond), RunID: "run-1", Status: "success"})
	runs := d.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	agg := runs[0].TimelinePhases["run"]
	if agg.CountTotal != 1 {
		t.Fatalf("count_total = %d, want 1", agg.CountTotal)
	}
	if agg.LatencyMs != 10 {
		t.Fatalf("latency_ms = %d, want 10", agg.LatencyMs)
	}
}

func TestStoreTimelineTrendsLastNRunsAndTimeWindow(t *testing.T) {
	d := NewStore(16, 16, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 2, TimeWindow: 15 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	base := time.Now().Add(-2 * time.Minute)

	addRunTimeline := func(runID string, seq int64, started, ended time.Time, status string) {
		d.AddTimelineEvent(runID, "model", "running", seq, started)
		d.AddTimelineEvent(runID, "model", status, seq+1, ended)
		d.AddRun(RunRecord{Time: ended, RunID: runID, Status: "success", LatencyMs: ended.Sub(started).Milliseconds()})
	}

	addRunTimeline("run-1", 1, base, base.Add(10*time.Millisecond), "succeeded")
	addRunTimeline("run-2", 10, base.Add(30*time.Second), base.Add(30*time.Second+20*time.Millisecond), "failed")
	addRunTimeline("run-3", 20, base.Add(90*time.Second), base.Add(90*time.Second+30*time.Millisecond), "canceled")

	lastN := d.TimelineTrends(TimelineTrendQuery{Mode: TimelineTrendModeLastNRuns})
	if len(lastN) == 0 {
		t.Fatal("last_n_runs trend should not be empty")
	}
	contains := func(items []TimelineTrendRecord, phase, status string) bool {
		for _, item := range items {
			if item.Phase == phase && item.Status == status {
				return true
			}
		}
		return false
	}
	if contains(lastN, "model", "succeeded") {
		t.Fatalf("last_n_runs should include only latest 2 runs, got %#v", lastN)
	}
	if !contains(lastN, "model", "failed") || !contains(lastN, "model", "canceled") {
		t.Fatalf("last_n_runs missing expected buckets: %#v", lastN)
	}
	for _, item := range lastN {
		if item.LatencyP95Ms <= 0 {
			t.Fatalf("latency_p95_ms must be >0, got %#v", item)
		}
		if item.WindowStart.IsZero() || item.WindowEnd.IsZero() {
			t.Fatalf("window bounds should be set, got %#v", item)
		}
	}

	window := d.TimelineTrends(TimelineTrendQuery{
		Mode:       TimelineTrendModeTimeWindow,
		TimeWindow: 45 * time.Second,
	})
	if len(window) == 0 {
		t.Fatal("time_window trend should not be empty")
	}
	if contains(window, "model", "succeeded") || contains(window, "model", "failed") {
		t.Fatalf("time_window should include only latest canceled bucket, got %#v", window)
	}
	if !contains(window, "model", "canceled") {
		t.Fatalf("time_window missing canceled bucket: %#v", window)
	}
}

func TestStoreTimelineTrendsIdempotentReplayAndEmptyWindow(t *testing.T) {
	d := NewStore(16, 16, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 1 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 1 * time.Minute})
	base := time.Now()
	d.AddTimelineEvent("run-1", "tool", "running", 1, base)
	d.AddTimelineEvent("run-1", "tool", "failed", 2, base.Add(10*time.Millisecond))
	d.AddTimelineEvent("run-1", "tool", "running", 1, base)
	d.AddTimelineEvent("run-1", "tool", "failed", 2, base.Add(10*time.Millisecond))
	d.AddRun(RunRecord{Time: base.Add(20 * time.Millisecond), RunID: "run-1", Status: "failed"})

	trends := d.TimelineTrends(TimelineTrendQuery{Mode: TimelineTrendModeLastNRuns, LastNRuns: 1})
	if len(trends) != 1 {
		t.Fatalf("trend len = %d, want 1", len(trends))
	}
	if trends[0].CountTotal != 1 || trends[0].FailedTotal != 1 {
		t.Fatalf("duplicate replay should not increase trend counts: %#v", trends[0])
	}

	d2 := NewStore(16, 16, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 1 * time.Minute}, ContextStage2ExternalTrendConfig{Enabled: true, Window: 1 * time.Minute})
	d2.AddTimelineEvent("run-zero", "tool", "running", 1, base)
	d2.AddTimelineEvent("run-zero", "tool", "failed", 2, base.Add(5*time.Millisecond))
	d2.AddRun(RunRecord{RunID: "run-zero", Status: "failed"})
	empty := d2.TimelineTrends(TimelineTrendQuery{Mode: TimelineTrendModeTimeWindow, TimeWindow: 30 * time.Second})
	if len(empty) != 0 {
		t.Fatalf("empty window should return empty set, got %#v", empty)
	}
}

func TestStoreContextStage2ExternalTrendsThresholdSignalsAndErrorLayerExtension(t *testing.T) {
	d := NewStore(16, 64, 8, 8,
		TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 15 * time.Minute},
		ContextStage2ExternalTrendConfig{
			Enabled: true,
			Window:  15 * time.Minute,
			Thresholds: ContextStage2ExternalThresholds{
				P95LatencyMs: 50,
				ErrorRate:    0.25,
				HitRate:      0.80,
			},
		},
	)
	base := time.Now().Add(-30 * time.Second)
	d.AddRun(RunRecord{
		Time:             base.Add(1 * time.Second),
		RunID:            "run-1",
		Stage2Provider:   "http",
		Stage2LatencyMs:  80,
		Stage2HitCount:   0,
		Stage2ReasonCode: "timeout",
		Stage2ErrorLayer: "transport",
	})
	d.AddRun(RunRecord{
		Time:             base.Add(2 * time.Second),
		RunID:            "run-2",
		Stage2Provider:   "http",
		Stage2LatencyMs:  60,
		Stage2HitCount:   1,
		Stage2ReasonCode: "upstream_custom",
		Stage2ErrorLayer: "vendor_limit",
	})
	d.AddRun(RunRecord{
		Time:             base.Add(3 * time.Second),
		RunID:            "run-3",
		Stage2Provider:   "http",
		Stage2LatencyMs:  20,
		Stage2HitCount:   1,
		Stage2ReasonCode: "ok",
	})

	items := d.ContextStage2ExternalTrends(ContextStage2ExternalTrendQuery{})
	if len(items) != 1 {
		t.Fatalf("trend len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Provider != "http" {
		t.Fatalf("provider = %q, want http", got.Provider)
	}
	if got.P95LatencyMs <= 50 {
		t.Fatalf("p95_latency_ms = %d, want > 50", got.P95LatencyMs)
	}
	if got.ErrorRate <= 0.25 {
		t.Fatalf("error_rate = %v, want > 0.25", got.ErrorRate)
	}
	if got.HitRate >= 0.80 {
		t.Fatalf("hit_rate = %v, want < 0.80", got.HitRate)
	}
	expectHits := map[string]bool{"p95_latency_ms": true, "error_rate": true, "hit_rate": true}
	for _, key := range got.ThresholdHits {
		delete(expectHits, key)
	}
	if len(expectHits) != 0 {
		t.Fatalf("threshold_hits missing = %#v, got %#v", expectHits, got.ThresholdHits)
	}
	if got.ErrorLayerDistribution["transport"] != 1 || got.ErrorLayerDistribution["vendor_limit"] != 1 {
		t.Fatalf("error layer distribution mismatch: %#v", got.ErrorLayerDistribution)
	}
}

func TestStoreContextStage2ExternalTrendsEmptyWindow(t *testing.T) {
	d := NewStore(8, 16, 8, 8,
		TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 1 * time.Minute},
		ContextStage2ExternalTrendConfig{Enabled: true, Window: 1 * time.Second},
	)
	base := time.Now()
	d.AddRun(RunRecord{
		Time:             base.Add(-2 * time.Minute),
		RunID:            "run-old",
		Stage2Provider:   "http",
		Stage2LatencyMs:  10,
		Stage2ReasonCode: "ok",
	})
	d.AddRun(RunRecord{
		RunID:            "run-no-time",
		Stage2Provider:   "http",
		Stage2LatencyMs:  10,
		Stage2ReasonCode: "ok",
	})
	items := d.ContextStage2ExternalTrends(ContextStage2ExternalTrendQuery{Window: 30 * time.Second})
	if len(items) != 0 {
		t.Fatalf("expected empty CA2 trend for window, got %#v", items)
	}
}

func TestStoreContextStage2ExternalTrendsRunStreamSemanticEquivalent(t *testing.T) {
	d := NewStore(8, 32, 8, 8,
		TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 15 * time.Minute},
		ContextStage2ExternalTrendConfig{
			Enabled: true,
			Window:  15 * time.Minute,
			Thresholds: ContextStage2ExternalThresholds{
				P95LatencyMs: 200,
				ErrorRate:    0.8,
				HitRate:      0.1,
			},
		},
	)
	base := time.Now()
	d.AddRun(RunRecord{
		Time:             base.Add(10 * time.Millisecond),
		RunID:            "run-equivalent",
		Stage2Provider:   "http",
		Stage2LatencyMs:  90,
		Stage2HitCount:   1,
		Stage2ReasonCode: "ok",
	})
	d.AddRun(RunRecord{
		Time:             base.Add(20 * time.Millisecond),
		RunID:            "stream-equivalent",
		Stage2Provider:   "http",
		Stage2LatencyMs:  90,
		Stage2HitCount:   1,
		Stage2ReasonCode: "ok",
	})

	items := d.ContextStage2ExternalTrends(ContextStage2ExternalTrendQuery{})
	if len(items) != 1 {
		t.Fatalf("trend len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Provider != "http" {
		t.Fatalf("provider = %q, want http", got.Provider)
	}
	if got.ErrorRate != 0 {
		t.Fatalf("error_rate = %v, want 0", got.ErrorRate)
	}
	if got.HitRate != 1 {
		t.Fatalf("hit_rate = %v, want 1", got.HitRate)
	}
	if len(got.ThresholdHits) != 0 {
		t.Fatalf("threshold hits should be empty for equivalent healthy run/stream sample, got %#v", got.ThresholdHits)
	}
}
