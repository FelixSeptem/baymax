package health

import (
	"context"
	"errors"
	"fmt"
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
	CodeHealthy        = "adapter.health.healthy"
	CodeDegraded       = "adapter.health.degraded"
	CodeUnavailable    = "adapter.health.unavailable"
	CodeTargetNotFound = "adapter.health.target_not_found"
	CodeProbeTimeout   = "adapter.health.probe_timeout"
	CodeProbeFailed    = "adapter.health.probe_failed"
	CodeUnknownStatus  = "adapter.health.unknown_status"
)

const (
	DefaultProbeTimeout = 500 * time.Millisecond
	DefaultCacheTTL     = 30 * time.Second
)

type Result struct {
	Status    Status         `json:"status"`
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	CheckedAt time.Time      `json:"checked_at"`
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
}

type Runner struct {
	mu      sync.RWMutex
	opts    RunnerOptions
	now     func() time.Time
	entries map[string]cacheEntry
}

type cacheEntry struct {
	result    Result
	expiresAt time.Time
}

func NewRunner(opts RunnerOptions, now func() time.Time) *Runner {
	out := &Runner{
		opts:    normalizeOptions(opts),
		entries: map[string]cacheEntry{},
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

	if cached, ok := r.loadCached(normalizedTarget, now); ok {
		return cached
	}

	opts := r.options()
	out := r.execute(ctx, normalizedTarget, probe, opts.ProbeTimeout, now)
	if opts.CacheTTL > 0 {
		r.storeCached(normalizedTarget, out, now.Add(opts.CacheTTL))
	}
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
		Status:    normalizeStatus(in.Status),
		Code:      strings.TrimSpace(in.Code),
		Message:   strings.TrimSpace(in.Message),
		Metadata:  cloneMap(in.Metadata),
		CheckedAt: in.CheckedAt.UTC(),
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
	return opts
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
