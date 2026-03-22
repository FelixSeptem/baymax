package collab

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"time"
)

type Primitive string

const (
	PrimitiveHandoff     Primitive = "handoff"
	PrimitiveDelegation  Primitive = "delegation"
	PrimitiveAggregation Primitive = "aggregation"
)

type AggregationStrategy string

const (
	AggregationAllSettled   AggregationStrategy = "all_settled"
	AggregationFirstSuccess AggregationStrategy = "first_success"
)

type FailurePolicy string

const (
	FailurePolicyFailFast   FailurePolicy = "fail_fast"
	FailurePolicyBestEffort FailurePolicy = "best_effort"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusSkipped   Status = "skipped"
	StatusCanceled  Status = "canceled"
)

const (
	RetryOnTransportOnly = "transport_only"

	defaultRetryMaxAttempts    = 3
	defaultRetryBackoffInitial = 100 * time.Millisecond
	defaultRetryBackoffMax     = 2 * time.Second
	defaultRetryMultiplier     = 2.0
	defaultRetryJitterRatio    = 0.2
)

type RetryConfig struct {
	Enabled        bool          `json:"enabled"`
	MaxAttempts    int           `json:"max_attempts"`
	BackoffInitial time.Duration `json:"backoff_initial"`
	BackoffMax     time.Duration `json:"backoff_max"`
	Multiplier     float64       `json:"multiplier"`
	JitterRatio    float64       `json:"jitter_ratio"`
	RetryOn        string        `json:"retry_on"`
}

type RetryEventType string

const (
	RetryEventAttempt   RetryEventType = "retry_attempt"
	RetryEventSuccess   RetryEventType = "retry_success"
	RetryEventExhausted RetryEventType = "retry_exhausted"
)

type RetryEvent struct {
	Type         RetryEventType
	Attempt      int
	MaxAttempts  int
	Delay        time.Duration
	RetryOn      string
	Retryable    bool
	ErrorMessage string
}

type RetryObserver func(RetryEvent)

type Config struct {
	Enabled            bool                `json:"enabled"`
	DefaultAggregation AggregationStrategy `json:"default_aggregation"`
	FailurePolicy      FailurePolicy       `json:"failure_policy"`
	Retry              RetryConfig         `json:"retry"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:            false,
		DefaultAggregation: AggregationAllSettled,
		FailurePolicy:      FailurePolicyFailFast,
		Retry:              normalizeRetryConfig(RetryConfig{Enabled: false}),
	}
}

func ValidateConfig(cfg Config) error {
	if _, err := ParseAggregationStrategy(cfg.DefaultAggregation); err != nil {
		return err
	}
	if _, err := ParseFailurePolicy(cfg.FailurePolicy); err != nil {
		return err
	}
	return ValidateRetryConfig(cfg.Retry)
}

func ValidateRetryConfig(cfg RetryConfig) error {
	if cfg.MaxAttempts <= 0 {
		return errors.New("composer.collab.retry.max_attempts must be > 0")
	}
	if cfg.BackoffInitial <= 0 {
		return errors.New("composer.collab.retry.backoff_initial must be > 0")
	}
	if cfg.BackoffMax < cfg.BackoffInitial {
		return errors.New("composer.collab.retry.backoff_max must be >= composer.collab.retry.backoff_initial")
	}
	if cfg.Multiplier <= 1 {
		return errors.New("composer.collab.retry.multiplier must be > 1")
	}
	if cfg.JitterRatio < 0 || cfg.JitterRatio > 1 {
		return errors.New("composer.collab.retry.jitter_ratio must be in [0,1]")
	}
	retryOn := strings.ToLower(strings.TrimSpace(cfg.RetryOn))
	switch retryOn {
	case RetryOnTransportOnly:
		return nil
	default:
		return fmt.Errorf("composer.collab.retry.retry_on must be one of [%s], got %q", RetryOnTransportOnly, cfg.RetryOn)
	}
}

type BranchExecutor func(context.Context) (Outcome, error)

type Branch struct {
	ID       string
	Required bool
	Execute  BranchExecutor
}

type Request struct {
	Primitive   Primitive
	Strategy    AggregationStrategy
	Policy      FailurePolicy
	Retry       RetryConfig
	Aggregation []Branch
}

type Outcome struct {
	Status    Status
	Retryable bool
	Error     string
	Payload   map[string]any
}

type BranchResult struct {
	ID       string
	Required bool
	Outcome  Outcome
}

type Result struct {
	Primitive Primitive
	Strategy  AggregationStrategy
	Policy    FailurePolicy
	Outcome   Outcome
	Branches  []BranchResult
}

func Execute(ctx context.Context, cfg Config, req Request) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	primitive, err := ParsePrimitive(req.Primitive)
	if err != nil {
		return Result{}, err
	}
	strategy, err := resolveStrategy(cfg, req)
	if err != nil {
		return Result{}, err
	}
	policy, err := resolvePolicy(cfg, req)
	if err != nil {
		return Result{}, err
	}
	retryCfg, err := resolveRetryConfig(cfg.Retry, req.Retry)
	if err != nil {
		return Result{}, err
	}
	out := Result{
		Primitive: primitive,
		Strategy:  strategy,
		Policy:    policy,
		Branches:  make([]BranchResult, 0, len(req.Aggregation)),
	}
	if primitive == PrimitiveHandoff {
		out.Outcome = Outcome{Status: StatusSucceeded}
		return out, nil
	}
	if len(req.Aggregation) == 0 {
		out.Outcome = Outcome{Status: StatusSucceeded}
		return out, nil
	}
	for _, branch := range req.Aggregation {
		exec := branch.Execute
		if exec == nil {
			outcome := Outcome{Status: StatusFailed, Error: "branch executor is nil"}
			out.Branches = append(out.Branches, BranchResult{
				ID:       strings.TrimSpace(branch.ID),
				Required: branch.Required,
				Outcome:  outcome,
			})
			if policy == FailurePolicyFailFast && branch.Required {
				out.Outcome = outcome
				return out, nil
			}
			continue
		}
		outcome, retryAttempts, exhausted, execErr := executeBranchWithRetry(
			ctx,
			retryCfg,
			exec,
			strings.TrimSpace(branch.ID),
		)
		if execErr != nil {
			if strings.TrimSpace(outcome.Error) == "" {
				outcome.Error = strings.TrimSpace(execErr.Error())
			}
			if outcome.Status == "" || outcome.Status == StatusRunning || outcome.Status == StatusPending {
				outcome.Status = StatusFailed
			}
		}
		outcome = withRetryPayload(outcome, retryAttempts, exhausted)
		outcome.Status = NormalizeStatus(outcome.Status)
		out.Branches = append(out.Branches, BranchResult{
			ID:       strings.TrimSpace(branch.ID),
			Required: branch.Required,
			Outcome:  outcome,
		})
		if strategy == AggregationFirstSuccess && outcome.Status == StatusSucceeded {
			out.Outcome = Outcome{Status: StatusSucceeded}
			return out, nil
		}
		if policy == FailurePolicyFailFast && branch.Required &&
			(outcome.Status == StatusFailed || outcome.Status == StatusCanceled) {
			out.Outcome = Outcome{
				Status:    StatusFailed,
				Retryable: outcome.Retryable,
				Error:     outcome.Error,
			}
			return out, nil
		}
	}
	out.Outcome = summarize(strategy, out.Branches)
	return out, nil
}

func executeBranchWithRetry(
	ctx context.Context,
	cfg RetryConfig,
	exec BranchExecutor,
	branchID string,
) (Outcome, int, bool, error) {
	attempt := 0
	retries := 0
	for {
		attempt++
		outcome, err := exec(ctx)
		if err != nil {
			if strings.TrimSpace(outcome.Error) == "" {
				outcome.Error = strings.TrimSpace(err.Error())
			}
			if outcome.Status == "" || outcome.Status == StatusRunning || outcome.Status == StatusPending {
				outcome.Status = StatusFailed
			}
		}
		if !shouldRetryOutcome(cfg, outcome) {
			return outcome, retries, false, err
		}
		if attempt >= cfg.MaxAttempts {
			return outcome, retries, true, err
		}
		delay := RetryDelay(cfg, attempt, "branch", branchID)
		if waitErr := waitWithContext(ctx, delay); waitErr != nil {
			return outcome, retries, false, waitErr
		}
		retries++
	}
}

func summarize(strategy AggregationStrategy, branches []BranchResult) Outcome {
	if strategy == AggregationFirstSuccess {
		for _, b := range branches {
			if b.Outcome.Status == StatusSucceeded {
				return Outcome{Status: StatusSucceeded}
			}
		}
	}
	hasSuccess := false
	allSkipped := true
	for _, b := range branches {
		switch b.Outcome.Status {
		case StatusSucceeded:
			hasSuccess = true
			allSkipped = false
		case StatusSkipped:
		case StatusFailed, StatusCanceled:
			return Outcome{
				Status:    StatusFailed,
				Retryable: b.Outcome.Retryable,
				Error:     b.Outcome.Error,
			}
		default:
			allSkipped = false
		}
	}
	if hasSuccess {
		return Outcome{Status: StatusSucceeded}
	}
	if allSkipped {
		return Outcome{Status: StatusSkipped}
	}
	return Outcome{Status: StatusFailed}
}

func resolveStrategy(cfg Config, req Request) (AggregationStrategy, error) {
	v := req.Strategy
	if strings.TrimSpace(string(v)) == "" {
		v = cfg.DefaultAggregation
	}
	return ParseAggregationStrategy(v)
}

func resolvePolicy(cfg Config, req Request) (FailurePolicy, error) {
	v := req.Policy
	if strings.TrimSpace(string(v)) == "" {
		v = cfg.FailurePolicy
	}
	return ParseFailurePolicy(v)
}

func resolveRetryConfig(base, override RetryConfig) (RetryConfig, error) {
	out := normalizeRetryConfig(base)
	if override.Enabled {
		out.Enabled = true
	}
	if override.MaxAttempts > 0 {
		out.MaxAttempts = override.MaxAttempts
	}
	if override.BackoffInitial > 0 {
		out.BackoffInitial = override.BackoffInitial
	}
	if override.BackoffMax > 0 {
		out.BackoffMax = override.BackoffMax
	}
	if override.Multiplier > 0 {
		out.Multiplier = override.Multiplier
	}
	if override.JitterRatio > 0 {
		out.JitterRatio = override.JitterRatio
	}
	if v := strings.TrimSpace(strings.ToLower(override.RetryOn)); v != "" {
		out.RetryOn = v
	}
	if err := ValidateRetryConfig(out); err != nil {
		return RetryConfig{}, err
	}
	return out, nil
}

func normalizeRetryConfig(cfg RetryConfig) RetryConfig {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultRetryMaxAttempts
	}
	if cfg.BackoffInitial <= 0 {
		cfg.BackoffInitial = defaultRetryBackoffInitial
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = defaultRetryBackoffMax
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = defaultRetryMultiplier
	}
	if cfg.JitterRatio == 0 {
		cfg.JitterRatio = defaultRetryJitterRatio
	}
	cfg.RetryOn = strings.ToLower(strings.TrimSpace(cfg.RetryOn))
	if cfg.RetryOn == "" {
		cfg.RetryOn = RetryOnTransportOnly
	}
	return cfg
}

func shouldRetryOutcome(cfg RetryConfig, outcome Outcome) bool {
	if !cfg.Enabled || !outcome.Retryable {
		return false
	}
	switch cfg.RetryOn {
	case RetryOnTransportOnly:
		return true
	default:
		return false
	}
}

func withRetryPayload(outcome Outcome, retryAttempts int, exhausted bool) Outcome {
	if retryAttempts <= 0 && !exhausted {
		return outcome
	}
	payload := cloneMap(outcome.Payload)
	payload["collab_retry_attempts"] = retryAttempts
	if exhausted {
		payload["collab_retry_exhausted"] = true
	}
	outcome.Payload = payload
	return outcome
}

func RetryDelay(cfg RetryConfig, failedAttempt int, seedParts ...string) time.Duration {
	if failedAttempt <= 0 {
		return 0
	}
	base := float64(cfg.BackoffInitial) * math.Pow(cfg.Multiplier, float64(failedAttempt-1))
	max := float64(cfg.BackoffMax)
	if base > max {
		base = max
	}
	if cfg.JitterRatio > 0 {
		jitterBound := base * cfg.JitterRatio
		seed := stableRetrySeed(seedParts, failedAttempt)
		ratio := stableRetryUnitFloat(seed)
		offset := (ratio*2 - 1) * jitterBound
		base = math.Max(0, base+offset)
	}
	if base > max {
		base = max
	}
	return time.Duration(base)
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func stableRetrySeed(parts []string, attempt int) uint64 {
	h := fnv.New64a()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(part)))
		_, _ = h.Write([]byte{'|'})
	}
	_, _ = fmt.Fprintf(h, "%d", attempt)
	return h.Sum64()
}

func stableRetryUnitFloat(seed uint64) float64 {
	const denom = float64((1 << 53) - 1)
	masked := float64(seed & ((1 << 53) - 1))
	if masked <= 0 {
		return 0.5
	}
	return masked / denom
}

func ParsePrimitive(v Primitive) (Primitive, error) {
	switch Primitive(strings.ToLower(strings.TrimSpace(string(v)))) {
	case PrimitiveHandoff:
		return PrimitiveHandoff, nil
	case PrimitiveDelegation:
		return PrimitiveDelegation, nil
	case PrimitiveAggregation:
		return PrimitiveAggregation, nil
	default:
		return "", fmt.Errorf("unsupported collaboration primitive %q", v)
	}
}

func ParseAggregationStrategy(v AggregationStrategy) (AggregationStrategy, error) {
	switch AggregationStrategy(strings.ToLower(strings.TrimSpace(string(v)))) {
	case AggregationAllSettled:
		return AggregationAllSettled, nil
	case AggregationFirstSuccess:
		return AggregationFirstSuccess, nil
	default:
		return "", fmt.Errorf("unsupported collaboration aggregation strategy %q", v)
	}
}

func ParseFailurePolicy(v FailurePolicy) (FailurePolicy, error) {
	switch FailurePolicy(strings.ToLower(strings.TrimSpace(string(v)))) {
	case FailurePolicyFailFast:
		return FailurePolicyFailFast, nil
	case FailurePolicyBestEffort:
		return FailurePolicyBestEffort, nil
	default:
		return "", fmt.Errorf("unsupported collaboration failure policy %q", v)
	}
}

func NormalizeStatus(v Status) Status {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case "submitted":
		return StatusPending
	case "pending":
		return StatusPending
	case "running":
		return StatusRunning
	case "succeeded", "success":
		return StatusSucceeded
	case "failed":
		return StatusFailed
	case "skipped":
		return StatusSkipped
	case "canceled", "cancelled":
		return StatusCanceled
	default:
		return StatusFailed
	}
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
