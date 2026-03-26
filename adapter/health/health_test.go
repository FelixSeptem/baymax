package health

import (
	"context"
	"encoding/json"
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

func TestRunnerProbeGovernanceBackoffCircuitTransitionAndRecovery(t *testing.T) {
	base := time.Date(2026, time.March, 24, 9, 0, 0, 0, time.UTC)
	current := base
	var calls int32
	runner := NewRunner(RunnerOptions{
		ProbeTimeout: 100 * time.Millisecond,
		CacheTTL:     time.Millisecond,
		Backoff: BackoffOptions{
			Enabled:     true,
			Initial:     100 * time.Millisecond,
			Max:         500 * time.Millisecond,
			Multiplier:  2.0,
			JitterRatio: 0,
		},
		Circuit: CircuitOptions{
			Enabled:                  true,
			FailureThreshold:         2,
			OpenDuration:             time.Second,
			HalfOpenMaxProbe:         1,
			HalfOpenSuccessThreshold: 2,
		},
	}, func() time.Time { return current })

	probe := ProbeFunc(func(context.Context) (Result, error) {
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			return Result{Status: StatusUnavailable, Code: CodeProbeFailed, Message: "fixture unavailable"}, nil
		}
		return Result{Status: StatusHealthy, Code: CodeHealthy, Message: "fixture recovered"}, nil
	})

	first := runner.Probe(context.Background(), "adapter-a46", probe)
	if first.Governance.CircuitState != string(CircuitStateClosed) {
		t.Fatalf("first circuit state=%q, want closed", first.Governance.CircuitState)
	}

	current = current.Add(150 * time.Millisecond)
	second := runner.Probe(context.Background(), "adapter-a46", probe)
	if second.Governance.CircuitState != string(CircuitStateOpen) {
		t.Fatalf("second circuit state=%q, want open", second.Governance.CircuitState)
	}
	if second.Governance.CircuitOpenTotal != 1 {
		t.Fatalf("second open total=%d, want 1", second.Governance.CircuitOpenTotal)
	}

	third := runner.Probe(context.Background(), "adapter-a46", probe)
	if third.Code != CodeCircuitOpen {
		t.Fatalf("third code=%q, want %q", third.Code, CodeCircuitOpen)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("open circuit should short-circuit probe, calls=%d", calls)
	}

	current = current.Add(1100 * time.Millisecond)
	fourth := runner.Probe(context.Background(), "adapter-a46", probe)
	if fourth.Governance.CircuitState != string(CircuitStateHalfOpen) {
		t.Fatalf("fourth circuit state=%q, want half_open", fourth.Governance.CircuitState)
	}

	current = current.Add(150 * time.Millisecond)
	fifth := runner.Probe(context.Background(), "adapter-a46", probe)
	if fifth.Governance.CircuitState != string(CircuitStateClosed) {
		t.Fatalf("fifth circuit state=%q, want closed", fifth.Governance.CircuitState)
	}
	if fifth.Governance.CircuitRecoverTotal != 1 {
		t.Fatalf("fifth recover total=%d, want 1", fifth.Governance.CircuitRecoverTotal)
	}
	if fifth.Governance.PrimaryCode != CodeCircuitRecover {
		t.Fatalf("fifth primary code=%q, want %q", fifth.Governance.PrimaryCode, CodeCircuitRecover)
	}
}

func TestRunnerProbeGovernanceHalfOpenFailureReopensCircuit(t *testing.T) {
	base := time.Date(2026, time.March, 24, 9, 0, 0, 0, time.UTC)
	current := base
	var calls int32
	runner := NewRunner(RunnerOptions{
		ProbeTimeout: 100 * time.Millisecond,
		CacheTTL:     time.Millisecond,
		Circuit: CircuitOptions{
			Enabled:                  true,
			FailureThreshold:         1,
			OpenDuration:             time.Second,
			HalfOpenMaxProbe:         1,
			HalfOpenSuccessThreshold: 2,
		},
	}, func() time.Time { return current })

	probe := ProbeFunc(func(context.Context) (Result, error) {
		atomic.AddInt32(&calls, 1)
		return Result{Status: StatusUnavailable, Code: CodeProbeFailed, Message: "still unavailable"}, nil
	})

	first := runner.Probe(context.Background(), "adapter-half-open", probe)
	if first.Governance.CircuitState != string(CircuitStateOpen) {
		t.Fatalf("first circuit state=%q, want open", first.Governance.CircuitState)
	}

	second := runner.Probe(context.Background(), "adapter-half-open", probe)
	if second.Code != CodeCircuitOpen {
		t.Fatalf("second code=%q, want %q", second.Code, CodeCircuitOpen)
	}

	current = current.Add(1100 * time.Millisecond)
	third := runner.Probe(context.Background(), "adapter-half-open", probe)
	if third.Governance.CircuitState != string(CircuitStateOpen) {
		t.Fatalf("third circuit state=%q, want open (half-open probe failed)", third.Governance.CircuitState)
	}
	if third.Governance.CircuitOpenTotal < 2 {
		t.Fatalf("open total=%d, want >=2 after half-open failure", third.Governance.CircuitOpenTotal)
	}
}

func TestRunnerProbeGovernanceDeterministicForEquivalentSequence(t *testing.T) {
	run := func() []string {
		base := time.Date(2026, time.March, 24, 9, 0, 0, 0, time.UTC)
		current := base
		var calls int32
		r := NewRunner(RunnerOptions{
			ProbeTimeout: 50 * time.Millisecond,
			CacheTTL:     time.Millisecond,
			Backoff: BackoffOptions{
				Enabled:     true,
				Initial:     100 * time.Millisecond,
				Max:         300 * time.Millisecond,
				Multiplier:  2,
				JitterRatio: 0.2,
			},
			Circuit: CircuitOptions{
				Enabled:                  true,
				FailureThreshold:         2,
				OpenDuration:             500 * time.Millisecond,
				HalfOpenMaxProbe:         1,
				HalfOpenSuccessThreshold: 2,
			},
		}, func() time.Time { return current })
		probe := ProbeFunc(func(context.Context) (Result, error) {
			n := atomic.AddInt32(&calls, 1)
			switch n {
			case 1, 2:
				return Result{Status: StatusUnavailable, Code: CodeProbeFailed, Message: "failed"}, nil
			default:
				return Result{Status: StatusHealthy, Code: CodeHealthy, Message: "ok"}, nil
			}
		})
		sequence := make([]string, 0, 5)
		capture := func(res Result) {
			payload := struct {
				Code      string `json:"code"`
				State     string `json:"state"`
				OpenTotal int    `json:"open_total"`
				Recover   int    `json:"recover_total"`
				Backoff   int    `json:"backoff_total"`
				Primary   string `json:"primary"`
			}{
				Code:      res.Code,
				State:     res.Governance.CircuitState,
				OpenTotal: res.Governance.CircuitOpenTotal,
				Recover:   res.Governance.CircuitRecoverTotal,
				Backoff:   res.Governance.BackoffAppliedTotal,
				Primary:   res.Governance.PrimaryCode,
			}
			blob, _ := json.Marshal(payload)
			sequence = append(sequence, string(blob))
		}

		capture(r.Probe(context.Background(), "adapter-deterministic", probe))
		current = current.Add(150 * time.Millisecond)
		capture(r.Probe(context.Background(), "adapter-deterministic", probe))
		capture(r.Probe(context.Background(), "adapter-deterministic", probe))
		current = current.Add(600 * time.Millisecond)
		capture(r.Probe(context.Background(), "adapter-deterministic", probe))
		current = current.Add(150 * time.Millisecond)
		capture(r.Probe(context.Background(), "adapter-deterministic", probe))
		return sequence
	}

	first := run()
	second := run()
	if len(first) != len(second) {
		t.Fatalf("sequence length mismatch: first=%d second=%d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("governance sequence mismatch at step=%d first=%s second=%s", i, first[i], second[i])
		}
	}
}
