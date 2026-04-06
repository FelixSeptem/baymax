package integration

import (
	"reflect"
	"strings"
	"testing"
	"time"

	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestDiagnosticsCardinalityContractBudgetAndOverflowPolicy(t *testing.T) {
	truncateStore := newDiagnosticsCardinalityStore(runtimediag.CardinalityOverflowTruncateAndRecord)
	truncateStore.AddRun(cardinalityOverflowRunRecord("run-a45-truncate-policy"))

	truncatePage, err := truncateStore.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a45-truncate-policy"})
	if err != nil {
		t.Fatalf("truncate query failed: %v", err)
	}
	if len(truncatePage.Items) != 1 {
		t.Fatalf("truncate query items len = %d, want 1", len(truncatePage.Items))
	}
	truncateRec := truncatePage.Items[0]
	if truncateRec.DiagnosticsCardinalityOverflowPolicy != runtimediag.CardinalityOverflowTruncateAndRecord {
		t.Fatalf("truncate overflow policy = %q, want %q", truncateRec.DiagnosticsCardinalityOverflowPolicy, runtimediag.CardinalityOverflowTruncateAndRecord)
	}
	if truncateRec.DiagnosticsCardinalityBudgetHitTotal <= 0 || truncateRec.DiagnosticsCardinalityTruncatedTotal <= 0 {
		t.Fatalf("truncate counters must be >0, got %#v", truncateRec)
	}
	if truncateRec.DiagnosticsCardinalityFailFastRejectTotal != 0 {
		t.Fatalf("truncate mode should not mark fail-fast reject, got %#v", truncateRec)
	}
	if len(truncateRec.TaskBoardManualControlByReason) != 2 {
		t.Fatalf("truncate mode should keep bounded map entries, got %#v", truncateRec.TaskBoardManualControlByReason)
	}
	if _, ok := truncateRec.TaskBoardManualControlByReason["alpha"]; !ok {
		t.Fatalf("truncate mode should keep sorted key alpha, got %#v", truncateRec.TaskBoardManualControlByReason)
	}
	if _, ok := truncateRec.TaskBoardManualControlByReason["beta"]; !ok {
		t.Fatalf("truncate mode should keep sorted key beta, got %#v", truncateRec.TaskBoardManualControlByReason)
	}
	if _, ok := truncateRec.TaskBoardManualControlByReason["gamma"]; ok {
		t.Fatalf("truncate mode should drop sorted tail key gamma, got %#v", truncateRec.TaskBoardManualControlByReason)
	}
	if !strings.Contains(truncateRec.DiagnosticsCardinalityTruncatedFieldSummary, "task_board_manual_control_by_reason") {
		t.Fatalf("truncate summary should include map field, got %q", truncateRec.DiagnosticsCardinalityTruncatedFieldSummary)
	}
	if !strings.Contains(truncateRec.DiagnosticsCardinalityTruncatedFieldSummary, "timeout_resolution_trace") {
		t.Fatalf("truncate summary should include string field, got %q", truncateRec.DiagnosticsCardinalityTruncatedFieldSummary)
	}

	failFastStore := newDiagnosticsCardinalityStore(runtimediag.CardinalityOverflowFailFast)
	failFastStore.AddRun(cardinalityOverflowRunRecord("run-a45-fail-fast-policy"))
	failFastPage, err := failFastStore.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a45-fail-fast-policy"})
	if err != nil {
		t.Fatalf("fail_fast query failed: %v", err)
	}
	if len(failFastPage.Items) != 1 {
		t.Fatalf("fail_fast query items len = %d, want 1", len(failFastPage.Items))
	}
	failFastRec := failFastPage.Items[0]
	if failFastRec.DiagnosticsCardinalityOverflowPolicy != runtimediag.CardinalityOverflowFailFast {
		t.Fatalf("fail_fast overflow policy = %q, want %q", failFastRec.DiagnosticsCardinalityOverflowPolicy, runtimediag.CardinalityOverflowFailFast)
	}
	if failFastRec.DiagnosticsCardinalityFailFastRejectTotal != 1 {
		t.Fatalf("fail_fast reject total = %d, want 1", failFastRec.DiagnosticsCardinalityFailFastRejectTotal)
	}
	if failFastRec.DiagnosticsCardinalityBudgetHitTotal <= 0 {
		t.Fatalf("fail_fast budget hit must be >0, got %#v", failFastRec)
	}
	if failFastRec.DiagnosticsCardinalityTruncatedTotal != 0 {
		t.Fatalf("fail_fast truncated total = %d, want 0", failFastRec.DiagnosticsCardinalityTruncatedTotal)
	}
	if failFastRec.TimeoutResolutionTrace != "" {
		t.Fatalf("fail_fast should reject overflowing string payload, got %q", failFastRec.TimeoutResolutionTrace)
	}
	if failFastRec.TaskBoardManualControlByReason != nil {
		t.Fatalf("fail_fast should reject overflowing map payload, got %#v", failFastRec.TaskBoardManualControlByReason)
	}
}

func TestDiagnosticsCardinalityContractDeterministicTruncationRunStreamEquivalent(t *testing.T) {
	store := newDiagnosticsCardinalityStore(runtimediag.CardinalityOverflowTruncateAndRecord)

	runRec := cardinalityOverflowRunRecord("run-a45-equivalent-run")
	streamRec := cardinalityOverflowRunRecord("run-a45-equivalent-stream")
	streamRec.TaskBoardManualControlByReason = map[string]int{
		"beta":  2,
		"gamma": 3,
		"alpha": 1,
	}

	store.AddRun(runRec)
	store.AddRun(streamRec)

	runPage, err := store.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a45-equivalent-run"})
	if err != nil {
		t.Fatalf("run query failed: %v", err)
	}
	streamPage, err := store.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a45-equivalent-stream"})
	if err != nil {
		t.Fatalf("stream query failed: %v", err)
	}
	if len(runPage.Items) != 1 || len(streamPage.Items) != 1 {
		t.Fatalf("run/stream query mismatch: run=%#v stream=%#v", runPage, streamPage)
	}
	if !equalCardinalityProjection(runPage.Items[0], streamPage.Items[0]) {
		t.Fatalf("run/stream truncation should be semantically equivalent, run=%#v stream=%#v", runPage.Items[0], streamPage.Items[0])
	}
}

func TestDiagnosticsCardinalityContractReplayIdempotentAggregates(t *testing.T) {
	store := newDiagnosticsCardinalityStore(runtimediag.CardinalityOverflowTruncateAndRecord)
	rec := cardinalityOverflowRunRecord("run-a45-replay")
	store.AddRun(rec)
	store.AddRun(rec)
	store.AddRun(rec)

	page, err := store.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a45-replay"})
	if err != nil {
		t.Fatalf("query replay run failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("replay should remain idempotent with single run record, got %#v", page.Items)
	}
	got := page.Items[0]
	if got.DiagnosticsCardinalityBudgetHitTotal <= 0 || got.DiagnosticsCardinalityTruncatedTotal <= 0 {
		t.Fatalf("replay record must keep cardinality counters, got %#v", got)
	}
}

func TestDiagnosticsCardinalityContractMemoryFileParity(t *testing.T) {
	memoryStore := newDiagnosticsCardinalityStore(runtimediag.CardinalityOverflowTruncateAndRecord)
	fileStore := newDiagnosticsCardinalityStore(runtimediag.CardinalityOverflowTruncateAndRecord)

	rec := cardinalityOverflowRunRecord("run-a45-parity")
	memoryStore.AddRun(rec)
	fileStore.AddRun(rec)

	memoryPage, err := memoryStore.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a45-parity"})
	if err != nil {
		t.Fatalf("memory query failed: %v", err)
	}
	filePage, err := fileStore.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a45-parity"})
	if err != nil {
		t.Fatalf("file query failed: %v", err)
	}
	if len(memoryPage.Items) != 1 || len(filePage.Items) != 1 {
		t.Fatalf("parity query size mismatch: memory=%#v file=%#v", memoryPage, filePage)
	}
	if !equalCardinalityProjection(memoryPage.Items[0], filePage.Items[0]) {
		t.Fatalf("memory/file parity mismatch: memory=%#v file=%#v", memoryPage.Items[0], filePage.Items[0])
	}
}

func newDiagnosticsCardinalityStore(policy string) *runtimediag.Store {
	store := runtimediag.NewStore(
		64,
		64,
		32,
		32,
		runtimediag.TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute},
		runtimediag.ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute},
	)
	store.SetCardinalityConfig(runtimediag.CardinalityConfig{
		Enabled:        true,
		MaxMapEntries:  2,
		MaxListEntries: 2,
		MaxStringBytes: 8,
		OverflowPolicy: policy,
	})
	return store
}

func cardinalityOverflowRunRecord(runID string) runtimediag.RunRecord {
	return runtimediag.RunRecord{
		Time:                   time.Now().UTC(),
		RunID:                  runID,
		Status:                 "success",
		TimeoutResolutionTrace: "你好world你好",
		TaskBoardManualControlByReason: map[string]int{
			"gamma": 3,
			"alpha": 1,
			"beta":  2,
		},
	}
}

func equalCardinalityProjection(a, b runtimediag.RunRecord) bool {
	if a.DiagnosticsCardinalityBudgetHitTotal != b.DiagnosticsCardinalityBudgetHitTotal ||
		a.DiagnosticsCardinalityTruncatedTotal != b.DiagnosticsCardinalityTruncatedTotal ||
		a.DiagnosticsCardinalityFailFastRejectTotal != b.DiagnosticsCardinalityFailFastRejectTotal ||
		a.DiagnosticsCardinalityOverflowPolicy != b.DiagnosticsCardinalityOverflowPolicy ||
		a.DiagnosticsCardinalityTruncatedFieldSummary != b.DiagnosticsCardinalityTruncatedFieldSummary ||
		a.TimeoutResolutionTrace != b.TimeoutResolutionTrace {
		return false
	}
	return reflect.DeepEqual(a.TaskBoardManualControlByReason, b.TaskBoardManualControlByReason)
}
