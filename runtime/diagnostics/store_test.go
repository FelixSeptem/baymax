package diagnostics

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStoreConcurrentAccess(t *testing.T) {
	d := NewStore(32, 16, 8, 20, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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

func TestStoreRunTeamsAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                     time.Now(),
		RunID:                    "run-workflow-1",
		Status:                   "failed",
		WorkflowID:               "wf-alpha",
		WorkflowStatus:           "failed",
		WorkflowStepTotal:        6,
		WorkflowStepFailed:       2,
		WorkflowRemoteStepTotal:  2,
		WorkflowRemoteStepFailed: 1,
		WorkflowResumeCount:      1,
	}
	d.AddRun(rec)
	d.AddRun(rec)

	runs := d.RecentRuns(10)
	if len(runs) != 1 {
		t.Fatalf("run records = %d, want 1", len(runs))
	}
	if runs[0].WorkflowStepTotal != 6 || runs[0].WorkflowStepFailed != 2 ||
		runs[0].WorkflowRemoteStepTotal != 2 || runs[0].WorkflowRemoteStepFailed != 1 ||
		runs[0].WorkflowResumeCount != 1 {
		t.Fatalf("workflow aggregate should stay stable under replay, got %#v", runs[0])
	}
}

func TestStoreRunA2AAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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

func TestStoreRunSchedulerSubagentAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                       time.Now(),
		RunID:                      "run-scheduler-1",
		Status:                     "success",
		SchedulerBackend:           "file",
		SchedulerQueueTotal:        3,
		SchedulerClaimTotal:        4,
		SchedulerReclaimTotal:      1,
		SchedulerDelayedTaskTotal:  2,
		SchedulerDelayedClaimTotal: 2,
		SchedulerDelayedWaitMsP95:  180,
		SubagentChildTotal:         2,
		SubagentChildFailed:        1,
		SubagentBudgetRejectTotal:  1,
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
		runs[0].SubagentBudgetRejectTotal != 1 {
		t.Fatalf("scheduler/subagent fields mismatch under replay, got %#v", runs[0])
	}
}

func TestStoreRunRecoveryAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                   time.Now(),
		RunID:                  "run-recovery-1",
		Status:                 "success",
		RecoveryEnabled:        true,
		RecoveryRecovered:      true,
		RecoveryReplayTotal:    2,
		RecoveryConflict:       false,
		RecoveryFallbackUsed:   true,
		RecoveryFallbackReason: "recovery.backend.file_init_failed",
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
	if !runs[0].RecoveryFallbackUsed || runs[0].RecoveryFallbackReason != "recovery.backend.file_init_failed" {
		t.Fatalf("recovery fallback mismatch under replay, got %#v", runs[0])
	}
}

func TestStoreSkillDedupConcurrent(t *testing.T) {
	d := NewStore(8, 8, 4, 16, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(16, 16, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 2, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(16, 16, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 1 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 1 * time.Minute})
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

	d2 := NewStore(16, 16, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 1 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 1 * time.Minute})
	d2.AddTimelineEvent("run-zero", "tool", "running", 1, base)
	d2.AddTimelineEvent("run-zero", "tool", "failed", 2, base.Add(5*time.Millisecond))
	d2.AddRun(RunRecord{RunID: "run-zero", Status: "failed"})
	empty := d2.TimelineTrends(TimelineTrendQuery{Mode: TimelineTrendModeTimeWindow, TimeWindow: 30 * time.Second})
	if len(empty) != 0 {
		t.Fatalf("empty window should return empty set, got %#v", empty)
	}
}

func TestStoreCA2ExternalTrendsThresholdSignalsAndErrorLayerExtension(t *testing.T) {
	d := NewStore(16, 64, 8, 8,
		TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 15 * time.Minute},
		CA2ExternalTrendConfig{
			Enabled: true,
			Window:  15 * time.Minute,
			Thresholds: CA2ExternalThresholds{
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

	items := d.CA2ExternalTrends(CA2ExternalTrendQuery{})
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

func TestStoreCA2ExternalTrendsEmptyWindow(t *testing.T) {
	d := NewStore(8, 16, 8, 8,
		TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 1 * time.Minute},
		CA2ExternalTrendConfig{Enabled: true, Window: 1 * time.Second},
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
	items := d.CA2ExternalTrends(CA2ExternalTrendQuery{Window: 30 * time.Second})
	if len(items) != 0 {
		t.Fatalf("expected empty CA2 trend for window, got %#v", items)
	}
}

func TestStoreCA2ExternalTrendsRunStreamSemanticEquivalent(t *testing.T) {
	d := NewStore(8, 32, 8, 8,
		TimelineTrendConfig{Enabled: true, LastNRuns: 10, TimeWindow: 15 * time.Minute},
		CA2ExternalTrendConfig{
			Enabled: true,
			Window:  15 * time.Minute,
			Thresholds: CA2ExternalThresholds{
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

	items := d.CA2ExternalTrends(CA2ExternalTrendQuery{})
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
