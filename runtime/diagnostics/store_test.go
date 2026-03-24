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

func TestStoreRunReadinessAdditiveFieldsPersistAndReplayIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := RunRecord{
		Time:                          time.Now(),
		RunID:                         "run-a40-readiness",
		Status:                        "success",
		RuntimeReadinessStatus:        "degraded",
		RuntimeReadinessFindingTotal:  2,
		RuntimeReadinessBlockingTotal: 0,
		RuntimeReadinessDegradedTotal: 2,
		RuntimeReadinessPrimaryCode:   "scheduler.backend.fallback",
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
		items[0].RuntimeReadinessPrimaryCode != "scheduler.backend.fallback" {
		t.Fatalf("readiness fields mismatch after dedup: %#v", items[0])
	}

	rec.RuntimeReadinessStatus = "blocked"
	rec.RuntimeReadinessFindingTotal = 3
	rec.RuntimeReadinessBlockingTotal = 2
	rec.RuntimeReadinessDegradedTotal = 1
	rec.RuntimeReadinessPrimaryCode = "runtime.readiness.strict_escalated"
	d.AddRun(rec)

	items = d.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len=%d after replay update, want 1", len(items))
	}
	if items[0].RuntimeReadinessStatus != "blocked" ||
		items[0].RuntimeReadinessFindingTotal != 3 ||
		items[0].RuntimeReadinessBlockingTotal != 2 ||
		items[0].RuntimeReadinessDegradedTotal != 1 ||
		items[0].RuntimeReadinessPrimaryCode != "runtime.readiness.strict_escalated" {
		t.Fatalf("readiness fields mismatch after replay replacement: %#v", items[0])
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

func TestStoreRunAsyncDelayedAggregateReplayIsIdempotent(t *testing.T) {
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 8, 4, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
	d := NewStore(8, 32, 8, 8, TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
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
