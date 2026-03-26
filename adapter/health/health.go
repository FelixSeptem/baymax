package health

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusHealthy     Status = "healthy"
	StatusDegraded    Status = "degraded"
	StatusUnavailable Status = "unavailable"
)

const (
	CodeHealthy         = "adapter.health.healthy"
	CodeDegraded        = "adapter.health.degraded"
	CodeUnavailable     = "adapter.health.unavailable"
	CodeTargetNotFound  = "adapter.health.target_not_found"
	CodeProbeTimeout    = "adapter.health.probe_timeout"
	CodeProbeFailed     = "adapter.health.probe_failed"
	CodeUnknownStatus   = "adapter.health.unknown_status"
	CodeBackoffThrottle = "adapter.health.backoff_throttled"
	CodeCircuitOpen     = "adapter.health.circuit_open"
	CodeCircuitHalfOpen = "adapter.health.circuit_half_open"
	CodeCircuitRecover  = "adapter.health.circuit_recovered"
	CodeHalfOpenReject  = "adapter.health.half_open_probe_rejected"
)

const (
	DefaultProbeTimeout                  = 500 * time.Millisecond
	DefaultCacheTTL                      = 30 * time.Second
	DefaultBackoffInitial                = 200 * time.Millisecond
	DefaultBackoffMax                    = 5 * time.Second
	DefaultBackoffMul                    = 2.0
	DefaultBackoffJitter                 = 0.2
	DefaultCircuitFailureThreshold       = 3
	DefaultCircuitOpenDuration           = 30 * time.Second
	DefaultCircuitHalfOpenMaxProbe       = 1
	DefaultCircuitHalfOpenSuccessTrigger = 2
)

type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

type BackoffOptions struct {
	Enabled     bool
	Initial     time.Duration
	Max         time.Duration
	Multiplier  float64
	JitterRatio float64
}

type CircuitOptions struct {
	Enabled                  bool
	FailureThreshold         int
	OpenDuration             time.Duration
	HalfOpenMaxProbe         int
	HalfOpenSuccessThreshold int
}

type GovernanceSnapshot struct {
	BackoffAppliedTotal  int    `json:"backoff_applied_total,omitempty"`
	CircuitOpenTotal     int    `json:"circuit_open_total,omitempty"`
	CircuitHalfOpenTotal int    `json:"circuit_half_open_total,omitempty"`
	CircuitRecoverTotal  int    `json:"circuit_recover_total,omitempty"`
	CircuitState         string `json:"circuit_state,omitempty"`
	PrimaryCode          string `json:"primary_code,omitempty"`
}

type Result struct {
	Status     Status             `json:"status"`
	Code       string             `json:"code"`
	Message    string             `json:"message"`
	Metadata   map[string]any     `json:"metadata"`
	Governance GovernanceSnapshot `json:"governance,omitempty"`
	CheckedAt  time.Time          `json:"checked_at"`
}

type Probe interface {
	ProbeHealth(ctx context.Context) (Result, error)
}

type ProbeFunc func(context.Context) (Result, error)

func (f ProbeFunc) ProbeHealth(ctx context.Context) (Result, error) {
	return f(ctx)
}

type RunnerOptions struct {
	ProbeTimeout time.Duration
	CacheTTL     time.Duration
	Backoff      BackoffOptions
	Circuit      CircuitOptions
}

type Runner struct {
	mu      sync.RWMutex
	opts    RunnerOptions
	now     func() time.Time
	entries map[string]cacheEntry
	states  map[string]targetGovernanceState
}

type cacheEntry struct {
	result    Result
	expiresAt time.Time
}

type targetGovernanceState struct {
	CircuitState         CircuitState
	ConsecutiveFails     int
	HalfOpenSuccess      int
	HalfOpenInFlight     int
	OpenUntil            time.Time
	NextProbeAt          time.Time
	BackoffFailCount     int
	BackoffAppliedTotal  int
	CircuitOpenTotal     int
	CircuitHalfOpenTotal int
	CircuitRecoverTotal  int
	PrimaryCode          string
}

func NewRunner(opts RunnerOptions, now func() time.Time) *Runner {
	out := &Runner{
		opts:    normalizeOptions(opts),
		entries: map[string]cacheEntry{},
		states:  map[string]targetGovernanceState{},
	}
	if now == nil {
		out.now = func() time.Time { return time.Now().UTC() }
	} else {
		out.now = func() time.Time { return now().UTC() }
	}
	return out
}

func (r *Runner) UpdateOptions(opts RunnerOptions) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.opts = normalizeOptions(opts)
}

func (r *Runner) Probe(ctx context.Context, target string, probe Probe) Result {
	if r == nil {
		now := time.Now().UTC()
		return unavailableResult(CodeProbeFailed, "adapter health runner is nil", map[string]any{}, now)
	}

	normalizedTarget := strings.ToLower(strings.TrimSpace(target))
	now := r.now()
	if normalizedTarget == "" || probe == nil {
		return unavailableResult(
			CodeTargetNotFound,
			"adapter health target is not registered",
			map[string]any{"target": normalizedTarget},
			now,
		)
	}

	opts := r.options()
	if cached, ok := r.loadCachedGoverned(normalizedTarget, now, opts); ok {
		return cached
	}
	reservedHalfOpen := false
	if governanceEnabled(opts) {
		allow, blocked, reserved := r.beforeProbeGovernance(normalizedTarget, now, opts)
		if !allow {
			return blocked
		}
		reservedHalfOpen = reserved
	}
	out := r.execute(ctx, normalizedTarget, probe, opts.ProbeTimeout, now)
	if governanceEnabled(opts) {
		out = r.afterProbeGovernance(normalizedTarget, now, out, opts, reservedHalfOpen)
	}
	if opts.CacheTTL > 0 && shouldCacheResult(out, opts) {
		r.storeCached(normalizedTarget, out, now.Add(opts.CacheTTL))
	}
	return out
}

func (r *Runner) beforeProbeGovernance(target string, now time.Time, opts RunnerOptions) (bool, Result, bool) {
	if r == nil {
		return false, unavailableResult(CodeProbeFailed, "adapter health runner is nil", map[string]any{"target": target}, now), false
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.states[target]
	if opts.Circuit.Enabled {
		if state.CircuitState == "" {
			state.CircuitState = CircuitStateClosed
		}
		if state.CircuitState == CircuitStateOpen && !state.OpenUntil.IsZero() && !now.Before(state.OpenUntil) {
			state.CircuitState = CircuitStateHalfOpen
			state.HalfOpenSuccess = 0
			state.HalfOpenInFlight = 0
			state.CircuitHalfOpenTotal++
			state.PrimaryCode = CodeCircuitHalfOpen
		}
		if state.CircuitState == CircuitStateOpen && now.Before(state.OpenUntil) {
			r.states[target] = state
			meta := map[string]any{"target": target}
			if !state.OpenUntil.IsZero() {
				meta["open_until"] = state.OpenUntil.UTC().Format(time.RFC3339Nano)
			}
			blocked := unavailableResult(CodeCircuitOpen, "adapter probe skipped while circuit is open", meta, now)
			blocked = applyGovernanceSnapshot(blocked, snapshotGovernance(state, opts))
			return false, blocked, false
		}
		if state.CircuitState == CircuitStateHalfOpen &&
			opts.Circuit.HalfOpenMaxProbe > 0 &&
			state.HalfOpenInFlight >= opts.Circuit.HalfOpenMaxProbe {
			r.states[target] = state
			meta := map[string]any{"target": target, "half_open_max_probe": opts.Circuit.HalfOpenMaxProbe}
			blocked := unavailableResult(CodeHalfOpenReject, "half-open probe budget exhausted", meta, now)
			blocked = applyGovernanceSnapshot(blocked, snapshotGovernance(state, opts))
			return false, blocked, false
		}
	}
	if opts.Backoff.Enabled && !state.NextProbeAt.IsZero() && now.Before(state.NextProbeAt) {
		r.states[target] = state
		meta := map[string]any{
			"target":              target,
			"next_probe_after_ms": state.NextProbeAt.Sub(now).Milliseconds(),
		}
		blocked := unavailableResult(CodeBackoffThrottle, "adapter probe skipped due backoff throttle", meta, now)
		blocked = applyGovernanceSnapshot(blocked, snapshotGovernance(state, opts))
		return false, blocked, false
	}
	reservedHalfOpen := false
	if opts.Circuit.Enabled && state.CircuitState == CircuitStateHalfOpen {
		state.HalfOpenInFlight++
		reservedHalfOpen = true
	}
	r.states[target] = state
	return true, Result{}, reservedHalfOpen
}

func (r *Runner) afterProbeGovernance(target string, now time.Time, in Result, opts RunnerOptions, reservedHalfOpen bool) Result {
	if r == nil {
		return in
	}
	out := normalizeResult(in, now)
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.states[target]
	if opts.Circuit.Enabled && state.CircuitState == "" {
		state.CircuitState = CircuitStateClosed
	}
	if reservedHalfOpen && state.HalfOpenInFlight > 0 {
		state.HalfOpenInFlight--
	}

	status := normalizeStatus(out.Status)
	if opts.Circuit.Enabled {
		switch state.CircuitState {
		case CircuitStateOpen:
			if !state.OpenUntil.IsZero() && !now.Before(state.OpenUntil) {
				state.CircuitState = CircuitStateHalfOpen
				state.HalfOpenSuccess = 0
				state.CircuitHalfOpenTotal++
				state.PrimaryCode = CodeCircuitHalfOpen
			}
		case CircuitStateHalfOpen:
			switch status {
			case StatusHealthy:
				state.HalfOpenSuccess++
				if state.HalfOpenSuccess >= opts.Circuit.HalfOpenSuccessThreshold {
					state.CircuitState = CircuitStateClosed
					state.ConsecutiveFails = 0
					state.HalfOpenSuccess = 0
					state.OpenUntil = time.Time{}
					state.CircuitRecoverTotal++
					state.PrimaryCode = CodeCircuitRecover
				} else {
					state.PrimaryCode = CodeCircuitHalfOpen
				}
			case StatusUnavailable:
				state.CircuitState = CircuitStateOpen
				state.OpenUntil = now.Add(opts.Circuit.OpenDuration)
				state.HalfOpenSuccess = 0
				state.ConsecutiveFails = 0
				state.CircuitOpenTotal++
				state.PrimaryCode = CodeCircuitOpen
			default:
				state.PrimaryCode = CodeCircuitHalfOpen
			}
		default:
			switch status {
			case StatusUnavailable:
				state.ConsecutiveFails++
				if state.ConsecutiveFails >= opts.Circuit.FailureThreshold {
					state.CircuitState = CircuitStateOpen
					state.OpenUntil = now.Add(opts.Circuit.OpenDuration)
					state.HalfOpenSuccess = 0
					state.CircuitOpenTotal++
					state.PrimaryCode = CodeCircuitOpen
				}
			case StatusHealthy:
				state.ConsecutiveFails = 0
			case StatusDegraded:
				state.ConsecutiveFails = 0
			}
		}
	}
	if opts.Backoff.Enabled {
		if status == StatusUnavailable {
			state.BackoffFailCount++
			delay := computeBackoffDelay(target, state.BackoffFailCount, opts.Backoff)
			state.NextProbeAt = now.Add(delay)
			state.BackoffAppliedTotal++
			if strings.TrimSpace(state.PrimaryCode) == "" {
				state.PrimaryCode = CodeBackoffThrottle
			}
		} else {
			state.BackoffFailCount = 0
			state.NextProbeAt = time.Time{}
		}
	}
	r.states[target] = state
	out = applyGovernanceSnapshot(out, snapshotGovernance(state, opts))
	return out
}

func (r *Runner) execute(ctx context.Context, target string, probe Probe, timeout time.Duration, now time.Time) Result {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type probeOut struct {
		result Result
		err    error
	}
	ch := make(chan probeOut, 1)
	go func() {
		result, err := probe.ProbeHealth(probeCtx)
		ch <- probeOut{result: result, err: err}
	}()

	select {
	case <-probeCtx.Done():
		if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return unavailableResult(
				CodeProbeTimeout,
				fmt.Sprintf("adapter health probe timed out after %s", timeout),
				map[string]any{"target": target},
				now,
			)
		}
		return unavailableResult(
			CodeProbeFailed,
			"adapter health probe canceled",
			map[string]any{"target": target, "error": probeCtx.Err().Error()},
			now,
		)
	case out := <-ch:
		if out.err != nil {
			return unavailableResult(
				CodeProbeFailed,
				"adapter health probe failed",
				map[string]any{"target": target, "error": strings.TrimSpace(out.err.Error())},
				now,
			)
		}
		return normalizeResult(out.result, now)
	}
}

func normalizeResult(in Result, fallbackCheckedAt time.Time) Result {
	out := Result{
		Status:     normalizeStatus(in.Status),
		Code:       strings.TrimSpace(in.Code),
		Message:    strings.TrimSpace(in.Message),
		Metadata:   cloneMap(in.Metadata),
		Governance: normalizeGovernance(in.Governance),
		CheckedAt:  in.CheckedAt.UTC(),
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	if out.CheckedAt.IsZero() {
		out.CheckedAt = fallbackCheckedAt
	}

	switch out.Status {
	case StatusHealthy:
		if out.Code == "" {
			out.Code = CodeHealthy
		}
		if out.Message == "" {
			out.Message = "adapter is healthy"
		}
	case StatusDegraded:
		if out.Code == "" {
			out.Code = CodeDegraded
		}
		if out.Message == "" {
			out.Message = "adapter is degraded"
		}
	default:
		// Unknown status is normalized to unavailable with canonical reason code.
		if normalizeStatus(in.Status) != StatusUnavailable {
			out.Code = CodeUnknownStatus
			out.Message = "adapter probe returned unknown status"
			out.Metadata["raw_status"] = strings.TrimSpace(strings.ToLower(string(in.Status)))
		} else {
			if out.Code == "" {
				out.Code = CodeUnavailable
			}
			if out.Message == "" {
				out.Message = "adapter is unavailable"
			}
		}
		out.Status = StatusUnavailable
	}
	return out
}

func normalizeGovernance(in GovernanceSnapshot) GovernanceSnapshot {
	out := in
	out.CircuitState = strings.ToLower(strings.TrimSpace(out.CircuitState))
	out.PrimaryCode = strings.TrimSpace(out.PrimaryCode)
	return out
}

func unavailableResult(code, message string, metadata map[string]any, checkedAt time.Time) Result {
	return normalizeResult(Result{
		Status:    StatusUnavailable,
		Code:      code,
		Message:   message,
		Metadata:  metadata,
		CheckedAt: checkedAt,
	}, checkedAt)
}

func normalizeStatus(in Status) Status {
	switch Status(strings.ToLower(strings.TrimSpace(string(in)))) {
	case StatusHealthy:
		return StatusHealthy
	case StatusDegraded:
		return StatusDegraded
	case StatusUnavailable:
		return StatusUnavailable
	default:
		return Status("")
	}
}

func (r *Runner) loadCached(target string, now time.Time) (Result, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[target]
	if !ok || now.After(entry.expiresAt) {
		return Result{}, false
	}
	out := entry.result
	out.Metadata = cloneMap(out.Metadata)
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	out.Metadata["cache_hit"] = true
	return out, true
}

func (r *Runner) loadCachedGoverned(target string, now time.Time, opts RunnerOptions) (Result, bool) {
	if !governanceEnabled(opts) {
		return r.loadCached(target, now)
	}
	state := r.governanceSnapshot(target, opts)
	if strings.TrimSpace(state.CircuitState) == string(CircuitStateOpen) || strings.TrimSpace(state.CircuitState) == string(CircuitStateHalfOpen) {
		return Result{}, false
	}
	out, ok := r.loadCached(target, now)
	if !ok {
		return Result{}, false
	}
	out = applyGovernanceSnapshot(out, state)
	return out, true
}

func (r *Runner) storeCached(target string, result Result, expiresAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[target] = cacheEntry{
		result:    normalizeResult(result, result.CheckedAt),
		expiresAt: expiresAt.UTC(),
	}
}

func (r *Runner) options() RunnerOptions {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.opts
}

func normalizeOptions(opts RunnerOptions) RunnerOptions {
	if opts.ProbeTimeout <= 0 {
		opts.ProbeTimeout = DefaultProbeTimeout
	}
	if opts.CacheTTL <= 0 {
		opts.CacheTTL = DefaultCacheTTL
	}
	if opts.Backoff.Enabled {
		if opts.Backoff.Initial <= 0 {
			opts.Backoff.Initial = DefaultBackoffInitial
		}
		if opts.Backoff.Max < opts.Backoff.Initial {
			opts.Backoff.Max = DefaultBackoffMax
		}
		if opts.Backoff.Multiplier <= 1 {
			opts.Backoff.Multiplier = DefaultBackoffMul
		}
		if opts.Backoff.JitterRatio < 0 || opts.Backoff.JitterRatio > 1 {
			opts.Backoff.JitterRatio = DefaultBackoffJitter
		}
	}
	if opts.Circuit.Enabled {
		if opts.Circuit.FailureThreshold <= 0 {
			opts.Circuit.FailureThreshold = DefaultCircuitFailureThreshold
		}
		if opts.Circuit.OpenDuration <= 0 {
			opts.Circuit.OpenDuration = DefaultCircuitOpenDuration
		}
		if opts.Circuit.HalfOpenMaxProbe <= 0 {
			opts.Circuit.HalfOpenMaxProbe = DefaultCircuitHalfOpenMaxProbe
		}
		if opts.Circuit.HalfOpenSuccessThreshold <= 0 {
			opts.Circuit.HalfOpenSuccessThreshold = DefaultCircuitHalfOpenSuccessTrigger
		}
	}
	return opts
}

func governanceEnabled(opts RunnerOptions) bool {
	return opts.Backoff.Enabled || opts.Circuit.Enabled
}

func shouldCacheResult(result Result, opts RunnerOptions) bool {
	if !governanceEnabled(opts) {
		return true
	}
	if strings.TrimSpace(result.Governance.CircuitState) == string(CircuitStateOpen) {
		return false
	}
	return true
}

func snapshotGovernance(state targetGovernanceState, opts RunnerOptions) GovernanceSnapshot {
	out := GovernanceSnapshot{
		BackoffAppliedTotal:  state.BackoffAppliedTotal,
		CircuitOpenTotal:     state.CircuitOpenTotal,
		CircuitHalfOpenTotal: state.CircuitHalfOpenTotal,
		CircuitRecoverTotal:  state.CircuitRecoverTotal,
		PrimaryCode:          strings.TrimSpace(state.PrimaryCode),
	}
	if opts.Circuit.Enabled {
		circuit := state.CircuitState
		if circuit == "" {
			circuit = CircuitStateClosed
		}
		out.CircuitState = string(circuit)
	}
	return normalizeGovernance(out)
}

func (r *Runner) governanceSnapshot(target string, opts RunnerOptions) GovernanceSnapshot {
	if r == nil {
		return GovernanceSnapshot{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	state := r.states[target]
	return snapshotGovernance(state, opts)
}

func applyGovernanceSnapshot(in Result, governance GovernanceSnapshot) Result {
	out := normalizeResult(in, in.CheckedAt)
	out.Governance = normalizeGovernance(governance)
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	out.Metadata["governance_backoff_applied_total"] = out.Governance.BackoffAppliedTotal
	out.Metadata["governance_circuit_open_total"] = out.Governance.CircuitOpenTotal
	out.Metadata["governance_circuit_half_open_total"] = out.Governance.CircuitHalfOpenTotal
	out.Metadata["governance_circuit_recover_total"] = out.Governance.CircuitRecoverTotal
	if strings.TrimSpace(out.Governance.CircuitState) != "" {
		out.Metadata["governance_circuit_state"] = out.Governance.CircuitState
	}
	if strings.TrimSpace(out.Governance.PrimaryCode) != "" {
		out.Metadata["governance_primary_code"] = out.Governance.PrimaryCode
	}
	return out
}

func computeBackoffDelay(target string, failures int, cfg BackoffOptions) time.Duration {
	if failures <= 0 {
		return cfg.Initial
	}
	base := float64(cfg.Initial)
	maxVal := float64(cfg.Max)
	if base <= 0 {
		base = float64(DefaultBackoffInitial)
	}
	if maxVal < base {
		maxVal = float64(DefaultBackoffMax)
	}
	multiplier := cfg.Multiplier
	if multiplier <= 1 {
		multiplier = DefaultBackoffMul
	}
	value := base
	for i := 1; i < failures; i++ {
		value *= multiplier
		if value >= maxVal {
			value = maxVal
			break
		}
	}
	if cfg.JitterRatio > 0 {
		hash := fnv.New32a()
		_, _ = fmt.Fprintf(hash, "%s:%d", target, failures)
		unit := float64(hash.Sum32()%10000) / 9999.0
		scale := 1 + ((unit*2)-1)*cfg.JitterRatio
		value *= scale
	}
	value = math.Max(base, value)
	value = math.Min(maxVal, value)
	if value < 0 {
		value = 0
	}
	return time.Duration(value)
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
