package collab

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type RetryConfig struct {
	Enabled bool `json:"enabled"`
}

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
		Retry: RetryConfig{
			Enabled: false,
		},
	}
}

func ValidateConfig(cfg Config) error {
	if _, err := ParseAggregationStrategy(cfg.DefaultAggregation); err != nil {
		return err
	}
	if _, err := ParseFailurePolicy(cfg.FailurePolicy); err != nil {
		return err
	}
	if cfg.Retry.Enabled {
		return errors.New("composer.collab.retry.enabled must be false: primitive-layer retry is not supported")
	}
	return nil
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
	if req.Retry.Enabled || cfg.Retry.Enabled {
		return Result{}, errors.New("primitive-layer retry is disabled")
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
		outcome, execErr := exec(ctx)
		if execErr != nil {
			if strings.TrimSpace(outcome.Error) == "" {
				outcome.Error = strings.TrimSpace(execErr.Error())
			}
			if outcome.Status == "" || outcome.Status == StatusRunning || outcome.Status == StatusPending {
				outcome.Status = StatusFailed
			}
		}
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
