package local

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
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
	MaxCalls    int
	Concurrency int
	FailFast    bool
}

type Dispatcher struct {
	registry *Registry
}

func NewDispatcher(registry *Registry) *Dispatcher {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Dispatcher{registry: registry}
}

func (d *Dispatcher) Dispatch(ctx context.Context, calls []types.ToolCall, cfg DispatchConfig) ([]types.ToolCallOutcome, error) {
	if cfg.MaxCalls <= 0 {
		cfg.MaxCalls = len(calls)
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
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
		if err := d.invokeOne(ctx, calls[i], outcomes, i); err != nil && cfg.FailFast {
			return outcomes[:i+1], err
		}
	}
	return outcomes, nil
}

func (d *Dispatcher) dispatchReadOnly(ctx context.Context, calls []types.ToolCall, outcomes []types.ToolCallOutcome, idx []int, cfg DispatchConfig) error {
	if len(idx) == 0 {
		return nil
	}
	jobs := make(chan int)
	errCh := make(chan error, 1)
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
				if err := d.invokeOne(ctx, calls[i], outcomes, i); err != nil && cfg.FailFast {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}()
	}

	for _, i := range idx {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (d *Dispatcher) invokeOne(ctx context.Context, call types.ToolCall, outcomes []types.ToolCallOutcome, i int) error {
	ctx, span := otel.Tracer("baymax/tool/local").Start(ctx, "tool.invoke", oteltrace.WithAttributes(oteltraceAttrs(call.Name)...))
	defer span.End()
	tool, ok := d.registry.Get(call.Name)
	if !ok {
		outcomes[i] = failedOutcome(call, types.ErrTool, fmt.Sprintf("tool %q not found", call.Name), false, map[string]any{"name": call.Name})
		return errors.New(outcomes[i].Result.Error.Message)
	}
	if err := ValidateArgs(tool.JSONSchema(), call.Args); err != nil {
		outcomes[i] = failedOutcome(call, types.ErrTool, "input validation failed", false, map[string]any{"validation": err.Error()})
		return errors.New(outcomes[i].Result.Error.Message)
	}
	result, err := tool.Invoke(ctx, call.Args)
	if err != nil {
		outcomes[i] = failedOutcome(call, types.ErrTool, err.Error(), false, nil)
		return err
	}
	outcomes[i] = types.ToolCallOutcome{CallID: call.CallID, Name: call.Name, Result: result}
	return nil
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
