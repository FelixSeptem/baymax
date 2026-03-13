package diagnostics

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStoreConcurrentAccess(t *testing.T) {
	d := NewStore(32, 16, 8, 20)
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
	d := NewStore(8, 8, 4, 8)
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

func TestStoreSkillDedupConcurrent(t *testing.T) {
	d := NewStore(8, 8, 4, 16)
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
	d := NewStore(8, 8, 4, 8)
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
	d := NewStore(8, 8, 4, 8)
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
