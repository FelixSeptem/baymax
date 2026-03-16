package local

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const namespace = "local."

type WriteAwareTool interface {
	IsWrite() bool
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]types.Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]types.Tool)}
}

func NormalizeName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("tool name is empty")
	}
	if strings.Contains(trimmed, " ") {
		return "", fmt.Errorf("tool name %q must not contain spaces", trimmed)
	}
	if strings.HasPrefix(trimmed, namespace) {
		return trimmed, nil
	}
	if strings.Contains(trimmed, ".") {
		return "", fmt.Errorf("tool name %q must be local namespace or simple name", trimmed)
	}
	return namespace + trimmed, nil
}

func (r *Registry) Register(tool types.Tool) (string, error) {
	if tool == nil {
		return "", fmt.Errorf("tool is nil")
	}
	fqName, err := NormalizeName(tool.Name())
	if err != nil {
		return "", err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[fqName]; exists {
		return "", fmt.Errorf("tool %q already registered", fqName)
	}
	r.tools[fqName] = tool
	return fqName, nil
}

func (r *Registry) Get(name string) (types.Tool, bool) {
	fqName, err := NormalizeName(name)
	if err != nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[fqName]
	return tool, ok
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.tools))
	for name := range r.tools {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

type DispatchConfig struct {
	MaxCalls        int
	Concurrency     int
	FailFast        bool
	QueueSize       int
	Backpressure    types.BackpressureMode
	Retry           int
	DropLowPriority DropLowPriorityPolicy
}

type DropLowPriorityPolicy struct {
	PriorityByTool      map[string]string
	PriorityByKeyword   map[string]string
	DroppablePriorities []string
}

type Dispatcher struct {
	registry   *Registry
	runtimeMgr *runtimeconfig.Manager
}

func NewDispatcher(registry *Registry) *Dispatcher {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Dispatcher{registry: registry}
}

func NewDispatcherWithRuntimeManager(registry *Registry, mgr *runtimeconfig.Manager) *Dispatcher {
	d := NewDispatcher(registry)
	d.runtimeMgr = mgr
	return d
}

func (d *Dispatcher) SetRuntimeManager(mgr *runtimeconfig.Manager) {
	d.runtimeMgr = mgr
}

func (d *Dispatcher) Dispatch(ctx context.Context, calls []types.ToolCall, cfg DispatchConfig) ([]types.ToolCallOutcome, error) {
	cfg = d.applyRuntimeDefaults(cfg)
	if cfg.MaxCalls <= 0 {
		cfg.MaxCalls = len(calls)
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = len(calls)
	}
	if cfg.Backpressure == "" {
		cfg.Backpressure = types.BackpressureBlock
	}
	if len(calls) > cfg.MaxCalls {
		return nil, fmt.Errorf("tool calls %d exceed max %d", len(calls), cfg.MaxCalls)
	}

	readIdx := make([]int, 0, len(calls))
	writeIdx := make([]int, 0, len(calls))
	outcomes := make([]types.ToolCallOutcome, len(calls))

	for i, call := range calls {
		tool, ok := d.registry.Get(call.Name)
		if !ok {
			outcomes[i] = failedOutcome(call, types.ErrTool, fmt.Sprintf("tool %q not found", call.Name), false, map[string]any{"name": call.Name})
			if cfg.FailFast {
				return outcomes[:i+1], errors.New(outcomes[i].Result.Error.Message)
			}
			continue
		}
		if wt, ok := tool.(WriteAwareTool); ok && wt.IsWrite() {
			writeIdx = append(writeIdx, i)
			continue
		}
		readIdx = append(readIdx, i)
	}

	readErr := d.dispatchReadOnly(ctx, calls, outcomes, readIdx, cfg)
	if readErr != nil && cfg.FailFast {
		return outcomes, readErr
	}

	for _, i := range writeIdx {
		if err := d.invokeOne(ctx, calls[i], outcomes, i, cfg.Retry); err != nil && cfg.FailFast {
			return outcomes[:i+1], err
		}
	}
	return outcomes, nil
}

func (d *Dispatcher) applyRuntimeDefaults(cfg DispatchConfig) DispatchConfig {
	if d.runtimeMgr == nil {
		return cfg
	}
	rc := d.runtimeMgr.EffectiveConfig().Concurrency
	if cfg.Concurrency <= 0 && rc.LocalMaxWorkers > 0 {
		cfg.Concurrency = rc.LocalMaxWorkers
	}
	if cfg.QueueSize <= 0 && rc.LocalQueueSize > 0 {
		cfg.QueueSize = rc.LocalQueueSize
	}
	if cfg.Backpressure == "" && rc.Backpressure != "" {
		cfg.Backpressure = rc.Backpressure
	}
	if len(cfg.DropLowPriority.PriorityByTool) == 0 && len(rc.DropLowPriority.PriorityByTool) > 0 {
		cfg.DropLowPriority.PriorityByTool = copyStringMap(rc.DropLowPriority.PriorityByTool)
	}
	if len(cfg.DropLowPriority.PriorityByKeyword) == 0 && len(rc.DropLowPriority.PriorityByKeyword) > 0 {
		cfg.DropLowPriority.PriorityByKeyword = copyStringMap(rc.DropLowPriority.PriorityByKeyword)
	}
	if len(cfg.DropLowPriority.DroppablePriorities) == 0 && len(rc.DropLowPriority.DroppablePriorities) > 0 {
		cfg.DropLowPriority.DroppablePriorities = append([]string(nil), rc.DropLowPriority.DroppablePriorities...)
	}
	return cfg
}

func (d *Dispatcher) dispatchReadOnly(ctx context.Context, calls []types.ToolCall, outcomes []types.ToolCallOutcome, idx []int, cfg DispatchConfig) error {
	if len(idx) == 0 {
		return nil
	}
	if cfg.Backpressure == types.BackpressureDropLowPriority &&
		len(idx) > cfg.QueueSize &&
		allIndicesDroppable(calls, idx, cfg.DropLowPriority) {
		for _, i := range idx {
			priority := classifyPriority(calls[i], cfg.DropLowPriority)
			phase := dispatchPhaseForCall(calls[i])
			outcomes[i] = failedOutcome(calls[i], types.ErrTool, "tool call dropped by low-priority backpressure", true, map[string]any{
				"reason":         "queue_full",
				"drop_reason":    "low_priority_dropped",
				"priority":       priority,
				"dispatch_phase": string(phase),
			})
		}
		return nil
	}
	jobs := make(chan int, cfg.QueueSize)
	type asyncResult struct {
		index int
		err   error
	}
	resultCh := make(chan asyncResult, len(idx))
	var wg sync.WaitGroup
	workers := cfg.Concurrency
	if workers > len(idx) {
		workers = len(idx)
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				err := d.invokeOne(ctx, calls[i], outcomes, i, cfg.Retry)
				resultCh <- asyncResult{index: i, err: err}
			}
		}()
	}

	queued := 0
	for _, i := range idx {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		default:
		}
		switch cfg.Backpressure {
		case types.BackpressureReject:
			select {
			case jobs <- i:
				queued++
			default:
				phase := dispatchPhaseForCall(calls[i])
				outcomes[i] = failedOutcome(calls[i], types.ErrTool, "tool queue is full", true, map[string]any{
					"reason":         "queue_full",
					"dispatch_phase": string(phase),
				})
				if cfg.FailFast {
					close(jobs)
					wg.Wait()
					return errors.New(outcomes[i].Result.Error.Message)
				}
			}
		case types.BackpressureDropLowPriority:
			select {
			case jobs <- i:
				queued++
			default:
				priority := classifyPriority(calls[i], cfg.DropLowPriority)
				phase := dispatchPhaseForCall(calls[i])
				if canDropPriority(priority, cfg.DropLowPriority.DroppablePriorities) {
					outcomes[i] = failedOutcome(calls[i], types.ErrTool, "tool call dropped by low-priority backpressure", true, map[string]any{
						"reason":         "queue_full",
						"drop_reason":    "low_priority_dropped",
						"priority":       priority,
						"dispatch_phase": string(phase),
					})
					continue
				}
				select {
				case <-ctx.Done():
					close(jobs)
					wg.Wait()
					return ctx.Err()
				case jobs <- i:
					queued++
				}
			}
		default:
			select {
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				return ctx.Err()
			case jobs <- i:
				queued++
			}
		}
	}
	close(jobs)
	wg.Wait()
	for i := 0; i < queued; i++ {
		res := <-resultCh
		if res.err == nil {
			continue
		}
		if cfg.FailFast {
			return res.err
		}
	}
	return nil
}

func classifyPriority(call types.ToolCall, policy DropLowPriorityPolicy) string {
	toolName := strings.ToLower(strings.TrimSpace(call.Name))
	if v, ok := policy.PriorityByTool[toolName]; ok && strings.TrimSpace(v) != "" {
		return strings.ToLower(strings.TrimSpace(v))
	}
	payload := toolName
	if len(call.Args) > 0 {
		payload += " " + lowerArgsText(call.Args)
	}
	for _, keyword := range sortedKeywords(policy.PriorityByKeyword) {
		if strings.Contains(payload, keyword) {
			return strings.ToLower(strings.TrimSpace(policy.PriorityByKeyword[keyword]))
		}
	}
	return runtimeconfig.DropPriorityNormal
}

func canDropPriority(priority string, droppable []string) bool {
	p := strings.ToLower(strings.TrimSpace(priority))
	for _, v := range droppable {
		if strings.ToLower(strings.TrimSpace(v)) == p {
			return true
		}
	}
	return false
}

func lowerArgsText(args map[string]any) string {
	parts := make([]string, 0, len(args))
	for key, value := range args {
		parts = append(parts, strings.ToLower(strings.TrimSpace(key))+"="+strings.ToLower(fmt.Sprint(value)))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func sortedKeywords(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		k := strings.ToLower(strings.TrimSpace(key))
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	return keys
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func allIndicesDroppable(calls []types.ToolCall, idx []int, policy DropLowPriorityPolicy) bool {
	if len(idx) == 0 {
		return false
	}
	for _, i := range idx {
		priority := classifyPriority(calls[i], policy)
		if !canDropPriority(priority, policy.DroppablePriorities) {
			return false
		}
	}
	return true
}

func dispatchPhaseForCall(call types.ToolCall) types.ActionPhase {
	name := strings.ToLower(strings.TrimSpace(call.Name))
	switch {
	case strings.HasPrefix(name, "mcp."), strings.HasPrefix(name, "local.mcp_"), strings.HasPrefix(name, "local.mcp."):
		return types.ActionPhaseMCP
	case strings.HasPrefix(name, "skill."), strings.HasPrefix(name, "local.skill_"), strings.HasPrefix(name, "local.skill."):
		return types.ActionPhaseSkill
	default:
		return types.ActionPhaseTool
	}
}

func (d *Dispatcher) invokeOne(ctx context.Context, call types.ToolCall, outcomes []types.ToolCallOutcome, i int, retry int) error {
	ctx, span := otel.Tracer("baymax/tool/local").Start(ctx, "tool.invoke", oteltrace.WithAttributes(oteltraceAttrs(call.Name)...))
	defer span.End()
	start := time.Now()
	tool, ok := d.registry.Get(call.Name)
	if !ok {
		outcomes[i] = failedOutcome(call, types.ErrTool, fmt.Sprintf("tool %q not found", call.Name), false, map[string]any{"name": call.Name})
		d.recordToolDiag(call, start, outcomes[i].Result.Error)
		return errors.New(outcomes[i].Result.Error.Message)
	}
	if err := ValidateArgs(tool.JSONSchema(), call.Args); err != nil {
		outcomes[i] = failedOutcome(call, types.ErrTool, "input validation failed", false, map[string]any{"validation": err.Error()})
		d.recordToolDiag(call, start, outcomes[i].Result.Error)
		return errors.New(outcomes[i].Result.Error.Message)
	}
	attempts := retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		result, err := tool.Invoke(ctx, call.Args)
		if err == nil {
			outcomes[i] = types.ToolCallOutcome{CallID: call.CallID, Name: call.Name, Result: result}
			if outcomes[i].Result.Error != nil {
				outcomes[i].Result.Error.Details = mergeDetails(outcomes[i].Result.Error.Details, map[string]any{"retry_count": attempt})
			}
			d.recordToolDiag(call, start, outcomes[i].Result.Error)
			return nil
		}
		lastErr = err
		if !shouldRetryToolError(result, err, attempt, attempts) {
			break
		}
	}
	outcomes[i] = failedOutcome(call, types.ErrTool, lastErr.Error(), false, map[string]any{"retry_count": attempts - 1})
	d.recordToolDiag(call, start, outcomes[i].Result.Error)
	return lastErr
}

func shouldRetryToolError(result types.ToolResult, err error, attempt int, attempts int) bool {
	if attempt >= attempts-1 || err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if result.Error != nil {
		return result.Error.Retryable
	}
	return false
}

func mergeDetails(base map[string]any, extra map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range extra {
		base[k] = v
	}
	return base
}

func oteltraceAttrs(name string) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String("tool.name", name)}
}

func failedOutcome(call types.ToolCall, class types.ErrorClass, message string, retryable bool, details map[string]any) types.ToolCallOutcome {
	return types.ToolCallOutcome{
		CallID: call.CallID,
		Name:   call.Name,
		Result: types.ToolResult{
			Error: &types.ClassifiedError{Class: class, Message: message, Retryable: retryable, Details: details},
		},
	}
}

func (d *Dispatcher) recordToolDiag(call types.ToolCall, start time.Time, classifiedErr *types.ClassifiedError) {
	if d.runtimeMgr == nil {
		return
	}
	errorClass := ""
	if classifiedErr != nil {
		errorClass = string(classifiedErr.Class)
	}
	d.runtimeMgr.RecordCall(runtimediag.CallRecord{
		Time:       time.Now(),
		Component:  "tool",
		Transport:  "local",
		CallID:     call.CallID,
		Name:       call.Name,
		Action:     "invoke",
		LatencyMs:  time.Since(start).Milliseconds(),
		ErrorClass: errorClass,
	})
}
