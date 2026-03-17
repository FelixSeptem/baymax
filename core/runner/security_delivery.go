package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type securityAlertDispatchResult struct {
	Status            string
	FailureReason     string
	DeliveryMode      string
	RetryCount        int
	QueueDropped      bool
	QueueDropCount    int
	CircuitState      string
	CircuitOpenReason string
}

type securityAlertDeliveryRequest struct {
	Config   runtimeconfig.SecurityEventDeliveryConfig
	Callback types.SecurityAlertCallback
	Event    types.SecurityEvent
}

type securityAlertDeliveryTask struct {
	request securityAlertDeliveryRequest
}

type securityAlertDeliveryExecutor struct {
	now func() time.Time

	mu            sync.Mutex
	cond          *sync.Cond
	queue         []securityAlertDeliveryTask
	workerStarted bool

	circuit securityAlertCircuitBreaker
}

func newSecurityAlertDeliveryExecutor(now func() time.Time) *securityAlertDeliveryExecutor {
	if now == nil {
		now = time.Now
	}
	executor := &securityAlertDeliveryExecutor{
		now:   now,
		queue: make([]securityAlertDeliveryTask, 0, 16),
	}
	executor.cond = sync.NewCond(&executor.mu)
	return executor
}

func (e *securityAlertDeliveryExecutor) dispatch(ctx context.Context, req securityAlertDeliveryRequest) securityAlertDispatchResult {
	cfg := normalizeSecurityDeliveryConfig(req.Config)
	result := securityAlertDispatchResult{
		Status:       securityAlertDispatchFailed,
		DeliveryMode: normalizeSecurityAlertDeliveryMode(cfg.Mode),
		CircuitState: runtimeconfig.SecurityEventCircuitStateClosed,
	}
	if req.Callback == nil {
		result.FailureReason = securityAlertFailureCallbackMissing
		return result
	}
	if result.DeliveryMode == runtimeconfig.SecurityEventDeliveryModeSync {
		return e.dispatchSync(ctx, cfg, req.Callback, req.Event)
	}
	return e.dispatchAsync(cfg, req.Callback, req.Event)
}

func (e *securityAlertDeliveryExecutor) dispatchSync(
	ctx context.Context,
	cfg runtimeconfig.SecurityEventDeliveryConfig,
	callback types.SecurityAlertCallback,
	event types.SecurityEvent,
) securityAlertDispatchResult {
	return e.executeManagedDelivery(ctx, cfg, callback, event)
}

func (e *securityAlertDeliveryExecutor) dispatchAsync(
	cfg runtimeconfig.SecurityEventDeliveryConfig,
	callback types.SecurityAlertCallback,
	event types.SecurityEvent,
) securityAlertDispatchResult {
	now := e.now()
	fastFail, snapshot := e.circuit.shouldFastFail(now, cfg.CircuitBreaker)
	if fastFail {
		return securityAlertDispatchResult{
			Status:            securityAlertDispatchFailed,
			FailureReason:     securityAlertFailureCircuitOpen,
			DeliveryMode:      runtimeconfig.SecurityEventDeliveryModeAsync,
			CircuitState:      snapshot.state,
			CircuitOpenReason: snapshot.reason,
		}
	}

	dropped := 0
	e.mu.Lock()
	if !e.workerStarted {
		e.workerStarted = true
		go e.workerLoop()
	}
	limit := cfg.Queue.Size
	if limit <= 0 {
		limit = 1
	}
	for len(e.queue) >= limit {
		// Keep the latest deny signal under pressure.
		e.queue = e.queue[1:]
		dropped++
	}
	e.queue = append(e.queue, securityAlertDeliveryTask{
		request: securityAlertDeliveryRequest{
			Config:   cfg,
			Callback: callback,
			Event:    event,
		},
	})
	e.cond.Signal()
	e.mu.Unlock()

	snapshot = e.circuit.snapshot(e.now(), cfg.CircuitBreaker)
	return securityAlertDispatchResult{
		Status:         securityAlertDispatchQueued,
		DeliveryMode:   runtimeconfig.SecurityEventDeliveryModeAsync,
		QueueDropped:   dropped > 0,
		QueueDropCount: dropped,
		CircuitState:   snapshot.state,
	}
}

func (e *securityAlertDeliveryExecutor) workerLoop() {
	for {
		e.mu.Lock()
		for len(e.queue) == 0 {
			e.cond.Wait()
		}
		task := e.queue[0]
		e.queue = e.queue[1:]
		e.mu.Unlock()

		_ = e.executeManagedDelivery(context.Background(), task.request.Config, task.request.Callback, task.request.Event)
	}
}

func (e *securityAlertDeliveryExecutor) executeManagedDelivery(
	ctx context.Context,
	cfg runtimeconfig.SecurityEventDeliveryConfig,
	callback types.SecurityAlertCallback,
	event types.SecurityEvent,
) securityAlertDispatchResult {
	cfg = normalizeSecurityDeliveryConfig(cfg)
	result := securityAlertDispatchResult{
		Status:       securityAlertDispatchFailed,
		DeliveryMode: normalizeSecurityAlertDeliveryMode(cfg.Mode),
		CircuitState: runtimeconfig.SecurityEventCircuitStateClosed,
	}

	attempts := cfg.Retry.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	if attempts > 3 {
		attempts = 3
	}
	lastFailure := ""
	for attempt := 1; attempt <= attempts; attempt++ {
		now := e.now()
		allowed, state, reason := e.circuit.startAttempt(now, cfg.CircuitBreaker)
		if !allowed {
			return securityAlertDispatchResult{
				Status:            securityAlertDispatchFailed,
				FailureReason:     securityAlertFailureCircuitOpen,
				DeliveryMode:      normalizeSecurityAlertDeliveryMode(cfg.Mode),
				RetryCount:        attempt - 1,
				CircuitState:      state,
				CircuitOpenReason: reason,
			}
		}

		callCtx := ctx
		cancel := func() {}
		if cfg.Timeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		}
		err := callback(callCtx, event)
		cancel()
		if err == nil {
			e.circuit.finishAttempt(e.now(), cfg.CircuitBreaker, true, "")
			snapshot := e.circuit.snapshot(e.now(), cfg.CircuitBreaker)
			return securityAlertDispatchResult{
				Status:            securityAlertDispatchSucceeded,
				DeliveryMode:      normalizeSecurityAlertDeliveryMode(cfg.Mode),
				RetryCount:        attempt - 1,
				CircuitState:      snapshot.state,
				CircuitOpenReason: snapshot.reason,
			}
		}

		failureReason := normalizeSecurityAlertFailure(err)
		lastFailure = failureReason
		e.circuit.finishAttempt(e.now(), cfg.CircuitBreaker, false, failureReason)

		if attempt >= attempts {
			snapshot := e.circuit.snapshot(e.now(), cfg.CircuitBreaker)
			result.RetryCount = attempt - 1
			result.CircuitState = snapshot.state
			result.CircuitOpenReason = snapshot.reason
			result.FailureReason = securityAlertFailureRetryExhausted
			if result.CircuitOpenReason == "" {
				result.CircuitOpenReason = lastFailure
			}
			return result
		}

		backoff := retryBackoff(attempt, cfg.Retry)
		if backoff <= 0 {
			continue
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			snapshot := e.circuit.snapshot(e.now(), cfg.CircuitBreaker)
			return securityAlertDispatchResult{
				Status:            securityAlertDispatchFailed,
				FailureReason:     normalizeSecurityAlertFailure(ctx.Err()),
				DeliveryMode:      normalizeSecurityAlertDeliveryMode(cfg.Mode),
				RetryCount:        attempt - 1,
				CircuitState:      snapshot.state,
				CircuitOpenReason: snapshot.reason,
			}
		case <-timer.C:
		}
	}

	snapshot := e.circuit.snapshot(e.now(), cfg.CircuitBreaker)
	result.CircuitState = snapshot.state
	result.CircuitOpenReason = snapshot.reason
	result.FailureReason = securityAlertFailureCallbackError
	return result
}

func normalizeSecurityDeliveryConfig(cfg runtimeconfig.SecurityEventDeliveryConfig) runtimeconfig.SecurityEventDeliveryConfig {
	out := cfg
	out.Mode = normalizeSecurityAlertDeliveryMode(out.Mode)
	if out.Queue.Size <= 0 {
		out.Queue.Size = 1
	}
	if strings.TrimSpace(out.Queue.OverflowPolicy) == "" {
		out.Queue.OverflowPolicy = runtimeconfig.SecurityEventDeliveryOverflowDropOld
	}
	if out.Timeout <= 0 {
		out.Timeout = 1200 * time.Millisecond
	}
	if out.Retry.MaxAttempts <= 0 {
		out.Retry.MaxAttempts = 1
	}
	if out.Retry.MaxAttempts > 3 {
		out.Retry.MaxAttempts = 3
	}
	if out.Retry.BackoffInitial <= 0 {
		out.Retry.BackoffInitial = 100 * time.Millisecond
	}
	if out.Retry.BackoffMax <= 0 {
		out.Retry.BackoffMax = out.Retry.BackoffInitial
	}
	if out.Retry.BackoffMax < out.Retry.BackoffInitial {
		out.Retry.BackoffMax = out.Retry.BackoffInitial
	}
	if out.CircuitBreaker.FailureThreshold <= 0 {
		out.CircuitBreaker.FailureThreshold = 1
	}
	if out.CircuitBreaker.OpenWindow <= 0 {
		out.CircuitBreaker.OpenWindow = time.Second
	}
	if out.CircuitBreaker.HalfOpenProbes <= 0 {
		out.CircuitBreaker.HalfOpenProbes = 1
	}
	return out
}

func retryBackoff(attempt int, cfg runtimeconfig.SecurityEventDeliveryRetryConfig) time.Duration {
	if attempt <= 0 {
		return 0
	}
	backoff := cfg.BackoffInitial
	if backoff <= 0 {
		return 0
	}
	maxBackoff := cfg.BackoffMax
	if maxBackoff <= 0 || maxBackoff < backoff {
		maxBackoff = backoff
	}
	for i := 1; i < attempt; i++ {
		if backoff >= maxBackoff {
			return maxBackoff
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
	return backoff
}

func normalizeSecurityAlertFailure(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return securityAlertFailureCallbackTimeout
	case errors.Is(err, context.Canceled):
		return securityAlertFailureCallbackTimeout
	default:
		return securityAlertFailureCallbackError
	}
}

type securityAlertCircuitSnapshot struct {
	state  string
	reason string
}

type securityAlertCircuitBreaker struct {
	mu                  sync.Mutex
	state               string
	consecutiveFailures int
	openedAt            time.Time
	openReason          string
	halfOpenInFlight    int
}

func (c *securityAlertCircuitBreaker) snapshot(
	now time.Time,
	cfg runtimeconfig.SecurityEventDeliveryCircuitConfig,
) securityAlertCircuitSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.currentStateLocked(now, cfg)
	reason := strings.TrimSpace(c.openReason)
	if state != runtimeconfig.SecurityEventCircuitStateOpen && state != runtimeconfig.SecurityEventCircuitStateHalfOpen {
		reason = ""
	}
	return securityAlertCircuitSnapshot{
		state:  state,
		reason: reason,
	}
}

func (c *securityAlertCircuitBreaker) shouldFastFail(
	now time.Time,
	cfg runtimeconfig.SecurityEventDeliveryCircuitConfig,
) (bool, securityAlertCircuitSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.currentStateLocked(now, cfg)
	reason := strings.TrimSpace(c.openReason)
	switch state {
	case runtimeconfig.SecurityEventCircuitStateOpen:
		return true, securityAlertCircuitSnapshot{state: state, reason: reason}
	case runtimeconfig.SecurityEventCircuitStateHalfOpen:
		probes := cfg.HalfOpenProbes
		if probes <= 0 {
			probes = 1
		}
		if c.halfOpenInFlight >= probes {
			return true, securityAlertCircuitSnapshot{state: state, reason: reason}
		}
	}
	return false, securityAlertCircuitSnapshot{state: state, reason: reason}
}

func (c *securityAlertCircuitBreaker) startAttempt(
	now time.Time,
	cfg runtimeconfig.SecurityEventDeliveryCircuitConfig,
) (bool, string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.currentStateLocked(now, cfg)
	if state == runtimeconfig.SecurityEventCircuitStateOpen {
		return false, state, strings.TrimSpace(c.openReason)
	}
	if state == runtimeconfig.SecurityEventCircuitStateHalfOpen {
		probes := cfg.HalfOpenProbes
		if probes <= 0 {
			probes = 1
		}
		if c.halfOpenInFlight >= probes {
			return false, state, strings.TrimSpace(c.openReason)
		}
		c.halfOpenInFlight++
		return true, state, strings.TrimSpace(c.openReason)
	}
	return true, runtimeconfig.SecurityEventCircuitStateClosed, ""
}

func (c *securityAlertCircuitBreaker) finishAttempt(
	now time.Time,
	cfg runtimeconfig.SecurityEventDeliveryCircuitConfig,
	success bool,
	failureReason string,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	threshold := cfg.FailureThreshold
	if threshold <= 0 {
		threshold = 1
	}
	state := c.currentStateLocked(now, cfg)

	switch state {
	case runtimeconfig.SecurityEventCircuitStateHalfOpen:
		if c.halfOpenInFlight > 0 {
			c.halfOpenInFlight--
		}
		if success {
			if c.halfOpenInFlight == 0 {
				c.state = runtimeconfig.SecurityEventCircuitStateClosed
				c.consecutiveFailures = 0
				c.openReason = ""
			}
			return
		}
		c.state = runtimeconfig.SecurityEventCircuitStateOpen
		c.openedAt = now
		c.openReason = strings.TrimSpace(failureReason)
		c.consecutiveFailures = 0
		c.halfOpenInFlight = 0
		return
	case runtimeconfig.SecurityEventCircuitStateOpen:
		if success {
			c.state = runtimeconfig.SecurityEventCircuitStateClosed
			c.consecutiveFailures = 0
			c.openReason = ""
		} else if strings.TrimSpace(failureReason) != "" {
			c.openReason = strings.TrimSpace(failureReason)
		}
		return
	default:
		if success {
			c.state = runtimeconfig.SecurityEventCircuitStateClosed
			c.consecutiveFailures = 0
			c.openReason = ""
			return
		}
		c.consecutiveFailures++
		if c.consecutiveFailures >= threshold {
			c.state = runtimeconfig.SecurityEventCircuitStateOpen
			c.openedAt = now
			c.openReason = strings.TrimSpace(failureReason)
			c.consecutiveFailures = 0
		}
	}
}

func (c *securityAlertCircuitBreaker) currentStateLocked(
	now time.Time,
	cfg runtimeconfig.SecurityEventDeliveryCircuitConfig,
) string {
	state := strings.ToLower(strings.TrimSpace(c.state))
	if state == "" {
		state = runtimeconfig.SecurityEventCircuitStateClosed
		c.state = state
		return state
	}
	if state == runtimeconfig.SecurityEventCircuitStateOpen {
		openWindow := cfg.OpenWindow
		if openWindow <= 0 {
			openWindow = time.Second
		}
		if !c.openedAt.IsZero() && now.Sub(c.openedAt) >= openWindow {
			c.state = runtimeconfig.SecurityEventCircuitStateHalfOpen
			return runtimeconfig.SecurityEventCircuitStateHalfOpen
		}
	}
	return state
}
