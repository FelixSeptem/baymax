package health

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerProbeUnknownTargetDeterministicUnavailable(t *testing.T) {
	runner := NewRunner(RunnerOptions{}, fixedNow())
	got := runner.Probe(context.Background(), "unknown-adapter", nil)
	if got.Status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", got.Status, StatusUnavailable)
	}
	if got.Code != CodeTargetNotFound {
		t.Fatalf("code = %q, want %q", got.Code, CodeTargetNotFound)
	}
	if strings.TrimSpace(got.Message) == "" {
		t.Fatalf("message should not be empty: %#v", got)
	}
	if got.Metadata == nil {
		t.Fatalf("metadata should not be nil: %#v", got)
	}
}

func TestRunnerProbeTimeoutClassification(t *testing.T) {
	runner := NewRunner(RunnerOptions{
		ProbeTimeout: 20 * time.Millisecond,
		CacheTTL:     5 * time.Second,
	}, fixedNow())
	probe := ProbeFunc(func(ctx context.Context) (Result, error) {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return Result{Status: StatusHealthy, Code: CodeHealthy, Message: "late"}, nil
		}
	})
	got := runner.Probe(context.Background(), "adapter-timeout", probe)
	if got.Status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", got.Status, StatusUnavailable)
	}
	if got.Code != CodeProbeTimeout {
		t.Fatalf("code = %q, want %q", got.Code, CodeProbeTimeout)
	}
}

func TestRunnerProbeCacheReuseWithinTTL(t *testing.T) {
	var calls int32
	now := time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC)
	current := now
	runner := NewRunner(RunnerOptions{
		ProbeTimeout: 200 * time.Millisecond,
		CacheTTL:     30 * time.Second,
	}, func() time.Time {
		return current
	})

	probe := ProbeFunc(func(context.Context) (Result, error) {
		atomic.AddInt32(&calls, 1)
		return Result{
			Status:   StatusDegraded,
			Code:     CodeDegraded,
			Message:  "adapter backend latency elevated",
			Metadata: map[string]any{"source": "fixture"},
		}, nil
	})

	first := runner.Probe(context.Background(), "adapter-cache", probe)
	second := runner.Probe(context.Background(), "adapter-cache", probe)
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("probe call count = %d, want 1", calls)
	}
	if first.Status != second.Status || first.Code != second.Code {
		t.Fatalf("cached semantic drift first=%#v second=%#v", first, second)
	}
	if second.Metadata["cache_hit"] != true {
		t.Fatalf("cache_hit marker missing in cached result: %#v", second.Metadata)
	}
	if !first.CheckedAt.Equal(second.CheckedAt) {
		t.Fatalf("checked_at should stay stable within cache ttl: first=%s second=%s", first.CheckedAt, second.CheckedAt)
	}

	current = current.Add(31 * time.Second)
	third := runner.Probe(context.Background(), "adapter-cache", probe)
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("probe call count after ttl = %d, want 2", calls)
	}
	if third.Metadata["cache_hit"] == true {
		t.Fatalf("post-ttl probe should not be cache hit: %#v", third.Metadata)
	}
}

func TestRunnerProbeCanonicalUnknownUnavailableAndDegradedBranches(t *testing.T) {
	now := fixedNow()
	runner := NewRunner(RunnerOptions{}, now)

	unknownStatus := runner.Probe(context.Background(), "adapter-unknown", ProbeFunc(func(context.Context) (Result, error) {
		return Result{
			Status:   Status("mystery"),
			Code:     "custom.non_canonical",
			Message:  "mystery status",
			Metadata: map[string]any{"detail": "unknown"},
		}, nil
	}))
	if unknownStatus.Status != StatusUnavailable || unknownStatus.Code != CodeUnknownStatus {
		t.Fatalf("unknown status mapping mismatch: %#v", unknownStatus)
	}

	unavailable := runner.Probe(context.Background(), "adapter-unavailable", ProbeFunc(func(context.Context) (Result, error) {
		return Result{
			Status:   StatusUnavailable,
			Code:     "",
			Message:  "",
			Metadata: map[string]any{"detail": "down"},
		}, nil
	}))
	if unavailable.Status != StatusUnavailable || unavailable.Code != CodeUnavailable {
		t.Fatalf("unavailable mapping mismatch: %#v", unavailable)
	}

	degraded := runner.Probe(context.Background(), "adapter-degraded", ProbeFunc(func(context.Context) (Result, error) {
		return Result{
			Status:  StatusDegraded,
			Message: "degraded branch",
		}, nil
	}))
	if degraded.Status != StatusDegraded || degraded.Code != CodeDegraded {
		t.Fatalf("degraded mapping mismatch: %#v", degraded)
	}
}

func fixedNow() func() time.Time {
	return func() time.Time {
		return time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC)
	}
}
