package local

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
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

const (
	sandboxReasonPolicyDeny             = "sandbox.policy_deny"
	sandboxReasonLaunchFailed           = "sandbox.launch_failed"
	sandboxReasonFallbackAllowRecord    = "sandbox.fallback_allow_and_record"
	sandboxReasonCapabilityMismatch     = "sandbox.capability_mismatch"
	sandboxReasonToolNotAdapted         = "sandbox.tool_not_adapted"
	sandboxReasonSessionModeUnsupported = "sandbox.session_mode_unsupported"
	sandboxReasonEgressDeny             = "sandbox.egress_deny"
	sandboxReasonEgressAllowAndRecord   = "sandbox.egress_allow_and_record"
)

type WriteAwareTool interface {
	IsWrite() bool
}

type SandboxAdapterTool interface {
	BuildSandboxExecSpec(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error)
	HandleSandboxExecResult(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error)
}

type sandboxExecutionError struct {
	reason  string
	message string
	details map[string]any
}

type sandboxHostMarker struct {
	ReasonCode string
	Fields     map[string]any
}

func (e *sandboxExecutionError) Error() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.message)
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

type dropLowPriorityKeywordRule struct {
	keyword  string
	priority string
}

type dropLowPriorityClassifier struct {
	priorityByTool map[string]string
	keywordRules   []dropLowPriorityKeywordRule
	droppableSet   map[string]struct{}
	priorityCache  map[string]string
}

type Dispatcher struct {
	registry    *Registry
	runtimeMgr  *runtimeconfig.Manager
	middlewares []types.ToolMiddleware
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

func (d *Dispatcher) SetMiddlewares(middlewares ...types.ToolMiddleware) {
	if d == nil {
		return
	}
	if len(middlewares) == 0 {
		d.middlewares = nil
		return
	}
	out := make([]types.ToolMiddleware, 0, len(middlewares))
	for _, middleware := range middlewares {
		if middleware == nil {
			continue
		}
		out = append(out, middleware)
	}
	d.middlewares = out
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
	priorityClassifier := compileDropLowPriorityClassifier(cfg.DropLowPriority)

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

	readErr := d.dispatchReadOnly(ctx, calls, outcomes, readIdx, cfg, priorityClassifier)
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
	rc := d.runtimeMgr.EffectiveConfigRef().Concurrency
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

func (d *Dispatcher) dispatchReadOnly(
	ctx context.Context,
	calls []types.ToolCall,
	outcomes []types.ToolCallOutcome,
	idx []int,
	cfg DispatchConfig,
	priorityClassifier *dropLowPriorityClassifier,
) error {
	if len(idx) == 0 {
		return nil
	}
	if cfg.Backpressure == types.BackpressureDropLowPriority &&
		len(idx) > cfg.QueueSize &&
		allIndicesDroppable(calls, idx, cfg.DropLowPriority, priorityClassifier) {
		for _, i := range idx {
			priority := classifyPriority(calls[i], cfg.DropLowPriority, priorityClassifier)
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
				priority := classifyPriority(calls[i], cfg.DropLowPriority, priorityClassifier)
				phase := dispatchPhaseForCall(calls[i])
				if canDropPriority(priority, cfg.DropLowPriority.DroppablePriorities, priorityClassifier) {
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

func compileDropLowPriorityClassifier(policy DropLowPriorityPolicy) *dropLowPriorityClassifier {
	classifier := &dropLowPriorityClassifier{
		priorityByTool: make(map[string]string, len(policy.PriorityByTool)),
		keywordRules:   make([]dropLowPriorityKeywordRule, 0, len(policy.PriorityByKeyword)),
		droppableSet:   make(map[string]struct{}, len(policy.DroppablePriorities)),
		priorityCache:  make(map[string]string),
	}
	for key, value := range policy.PriorityByTool {
		k := normalizePriorityToken(key)
		v := normalizePriorityToken(value)
		if k == "" || v == "" {
			continue
		}
		classifier.priorityByTool[k] = v
	}
	for key, value := range policy.PriorityByKeyword {
		k := normalizePriorityToken(key)
		v := normalizePriorityToken(value)
		if k == "" || v == "" {
			continue
		}
		classifier.keywordRules = append(classifier.keywordRules, dropLowPriorityKeywordRule{
			keyword:  k,
			priority: v,
		})
	}
	sort.SliceStable(classifier.keywordRules, func(i, j int) bool {
		if len(classifier.keywordRules[i].keyword) != len(classifier.keywordRules[j].keyword) {
			return len(classifier.keywordRules[i].keyword) > len(classifier.keywordRules[j].keyword)
		}
		return classifier.keywordRules[i].keyword < classifier.keywordRules[j].keyword
	})
	for _, priority := range policy.DroppablePriorities {
		if normalized := normalizePriorityToken(priority); normalized != "" {
			classifier.droppableSet[normalized] = struct{}{}
		}
	}
	return classifier
}

func classifyPriority(call types.ToolCall, policy DropLowPriorityPolicy, classifier *dropLowPriorityClassifier) string {
	if classifier == nil {
		classifier = compileDropLowPriorityClassifier(policy)
	}
	toolName := normalizePriorityToken(call.Name)
	if priority, ok := classifier.priorityByTool[toolName]; ok {
		return priority
	}
	if len(classifier.keywordRules) == 0 {
		return runtimeconfig.DropPriorityNormal
	}
	payload := toolName
	if len(call.Args) > 0 {
		payload += " " + lowerArgsText(call.Args)
	}
	if cached, ok := classifier.priorityCache[payload]; ok {
		return cached
	}
	priority := runtimeconfig.DropPriorityNormal
	for i := range classifier.keywordRules {
		if strings.Contains(payload, classifier.keywordRules[i].keyword) {
			priority = classifier.keywordRules[i].priority
			break
		}
	}
	classifier.priorityCache[payload] = priority
	return priority
}

func canDropPriority(priority string, droppable []string, classifier *dropLowPriorityClassifier) bool {
	p := normalizePriorityToken(priority)
	if classifier != nil && len(classifier.droppableSet) > 0 {
		_, ok := classifier.droppableSet[p]
		return ok
	}
	for _, v := range droppable {
		if normalizePriorityToken(v) == p {
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

func allIndicesDroppable(
	calls []types.ToolCall,
	idx []int,
	policy DropLowPriorityPolicy,
	classifier *dropLowPriorityClassifier,
) bool {
	if len(idx) == 0 {
		return false
	}
	for _, i := range idx {
		priority := classifyPriority(calls[i], policy, classifier)
		if !canDropPriority(priority, policy.DroppablePriorities, classifier) {
			return false
		}
	}
	return true
}

func normalizePriorityToken(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
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

	hostMarker := sandboxHostMarker{}
	if handled, err := d.trySandboxInvoke(ctx, tool, call, outcomes, i, &hostMarker); handled {
		d.recordToolDiag(call, start, outcomes[i].Result.Error)
		return err
	}

	invokeCtx := ctx
	cancelInvoke := func() {}
	if d.middlewareEnabled() {
		timeout := d.runtimeToolMiddlewareConfig().Timeout
		if timeout > 0 {
			invokeCtx, cancelInvoke = context.WithTimeout(ctx, timeout)
		}
	}
	defer cancelInvoke()

	retryCount := 0
	baseInvoke := func(invokeCtx context.Context, invokeCall types.ToolCall) (types.ToolResult, error) {
		result, attemptsUsed, err := d.invokeToolWithRetry(invokeCtx, tool, invokeCall, retry)
		retryCount = attemptsUsed
		return result, err
	}
	invoker := baseInvoke
	if d.middlewareEnabled() && len(d.middlewares) > 0 {
		invoker = buildToolMiddlewareChain(d.middlewares, baseInvoke)
	}

	result, err := invoker(invokeCtx, call)
	if err == nil {
		if strings.TrimSpace(hostMarker.ReasonCode) != "" {
			result.Structured = annotateSandboxStructured(result.Structured, hostMarker)
		}
		outcomes[i] = types.ToolCallOutcome{CallID: call.CallID, Name: call.Name, Result: result}
		if outcomes[i].Result.Error != nil {
			outcomes[i].Result.Error.Details = mergeDetails(outcomes[i].Result.Error.Details, map[string]any{"retry_count": retryCount})
		}
		d.recordToolDiag(call, start, outcomes[i].Result.Error)
		return nil
	}
	outcomes[i] = failedOutcome(call, types.ErrTool, err.Error(), false, map[string]any{"retry_count": retryCount})
	d.recordToolDiag(call, start, outcomes[i].Result.Error)
	return err
}

func (d *Dispatcher) middlewareEnabled() bool {
	if d == nil {
		return false
	}
	if len(d.middlewares) == 0 {
		return false
	}
	if d.runtimeMgr == nil {
		return true
	}
	cfg := d.runtimeToolMiddlewareConfig()
	return cfg.Enabled
}

func (d *Dispatcher) runtimeToolMiddlewareConfig() runtimeconfig.RuntimeToolMiddlewareConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.ToolMiddleware
	if d != nil && d.runtimeMgr != nil {
		cfg = d.runtimeMgr.EffectiveConfigRef().Runtime.ToolMiddleware
	}
	return cfg
}

func (d *Dispatcher) invokeToolWithRetry(
	ctx context.Context,
	tool types.Tool,
	call types.ToolCall,
	retry int,
) (types.ToolResult, int, error) {
	attempts := retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	var retryCount int
	for attempt := 0; attempt < attempts; attempt++ {
		result, err := tool.Invoke(ctx, call.Args)
		retryCount = attempt
		if err == nil {
			return result, retryCount, nil
		}
		lastErr = err
		if !shouldRetryToolError(result, err, attempt, attempts) {
			break
		}
	}
	return types.ToolResult{}, retryCount, lastErr
}

func buildToolMiddlewareChain(
	middlewares []types.ToolMiddleware,
	base types.ToolInvokeFunc,
) types.ToolInvokeFunc {
	chain := base
	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]
		if middleware == nil {
			continue
		}
		mw := middleware
		next := chain
		chain = func(ctx context.Context, call types.ToolCall) (types.ToolResult, error) {
			return mw.Invoke(ctx, call, next)
		}
	}
	return chain
}

func (d *Dispatcher) trySandboxInvoke(
	ctx context.Context,
	tool types.Tool,
	call types.ToolCall,
	outcomes []types.ToolCallOutcome,
	i int,
	hostMarker *sandboxHostMarker,
) (bool, error) {
	if d == nil || d.runtimeMgr == nil {
		return false, nil
	}
	cfg := d.runtimeMgr.EffectiveConfigRef().Security.Sandbox
	if !cfg.Enabled {
		return false, nil
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	namespaceTool, _ := sandboxNamespaceToolKey(call.Name)
	action := runtimeconfig.ResolveSandboxAction(cfg, namespaceTool)
	fallback := runtimeconfig.ResolveSandboxFallbackAction(cfg, namespaceTool)
	egressDecision := resolveSandboxEgressDecision(cfg.Egress, namespaceTool, call.Args)
	if egressDecision.Applied {
		switch egressDecision.Action {
		case runtimeconfig.SecuritySandboxEgressActionDeny:
			if mode != runtimeconfig.SecuritySandboxModeEnforce {
				if hostMarker != nil {
					*hostMarker = sandboxHostMarker{
						ReasonCode: sandboxReasonEgressAllowAndRecord,
						Fields:     egressDecision.Fields(),
					}
				}
				break
			}
			details := map[string]any{
				"reason_code":      sandboxReasonEgressDeny,
				"namespace_tool":   namespaceTool,
				"sandbox_action":   action,
				"sandbox_mode":     mode,
				"sandbox_fallback": fallback,
			}
			for key, value := range egressDecision.Fields() {
				details[key] = value
			}
			outcomes[i] = failedOutcome(call, types.ErrSecurity, "tool call denied by sandbox egress policy", false, details)
			return true, errors.New(outcomes[i].Result.Error.Message)
		case runtimeconfig.SecuritySandboxEgressActionAllowAndRecord:
			if hostMarker != nil {
				*hostMarker = sandboxHostMarker{
					ReasonCode: sandboxReasonEgressAllowAndRecord,
					Fields:     egressDecision.Fields(),
				}
			}
		}
	}

	if mode != runtimeconfig.SecuritySandboxModeEnforce {
		if action == runtimeconfig.SecuritySandboxActionSandbox {
			if _, ok := tool.(SandboxAdapterTool); !ok && hostMarker != nil {
				*hostMarker = sandboxHostMarker{ReasonCode: sandboxReasonToolNotAdapted}
			}
		}
		return false, nil
	}

	if action == runtimeconfig.SecuritySandboxActionDeny {
		outcomes[i] = failedOutcome(call, types.ErrSecurity, "tool call denied by sandbox policy", false, map[string]any{
			"reason_code":      sandboxReasonPolicyDeny,
			"namespace_tool":   namespaceTool,
			"sandbox_action":   action,
			"sandbox_mode":     mode,
			"sandbox_fallback": fallback,
		})
		return true, errors.New(outcomes[i].Result.Error.Message)
	}
	if action != runtimeconfig.SecuritySandboxActionSandbox {
		return false, nil
	}

	adapter, ok := tool.(SandboxAdapterTool)
	if !ok {
		if fallback == runtimeconfig.SecuritySandboxFallbackAllowAndRecord {
			if hostMarker != nil {
				*hostMarker = sandboxHostMarker{ReasonCode: sandboxReasonFallbackAllowRecord}
			}
			return false, nil
		}
		outcomes[i] = failedOutcome(call, types.ErrSecurity, "tool denied because sandbox adapter is not available", false, map[string]any{
			"reason_code":      sandboxReasonToolNotAdapted,
			"namespace_tool":   namespaceTool,
			"sandbox_action":   action,
			"sandbox_mode":     mode,
			"sandbox_fallback": fallback,
		})
		return true, errors.New(outcomes[i].Result.Error.Message)
	}

	result, err := d.invokeSandboxAdapter(ctx, adapter, call, namespaceTool, mode, action, fallback, cfg)
	if err == nil {
		outcomes[i] = types.ToolCallOutcome{CallID: call.CallID, Name: call.Name, Result: result}
		return true, nil
	}
	var sandboxErr *sandboxExecutionError
	if errors.As(err, &sandboxErr) {
		if fallback == runtimeconfig.SecuritySandboxFallbackAllowAndRecord {
			if hostMarker != nil {
				*hostMarker = sandboxHostMarker{ReasonCode: sandboxReasonFallbackAllowRecord}
			}
			return false, nil
		}
		details := map[string]any{
			"reason_code":      strings.ToLower(strings.TrimSpace(sandboxErr.reason)),
			"namespace_tool":   namespaceTool,
			"sandbox_action":   action,
			"sandbox_mode":     mode,
			"sandbox_fallback": fallback,
		}
		for key, value := range sandboxErr.details {
			details[key] = value
		}
		outcomes[i] = failedOutcome(call, types.ErrSecurity, sandboxErr.Error(), false, details)
		return true, err
	}
	outcomes[i] = failedOutcome(call, types.ErrSecurity, err.Error(), false, map[string]any{
		"reason_code":      sandboxReasonLaunchFailed,
		"namespace_tool":   namespaceTool,
		"sandbox_action":   action,
		"sandbox_mode":     mode,
		"sandbox_fallback": fallback,
	})
	return true, err
}

func (d *Dispatcher) invokeSandboxAdapter(
	ctx context.Context,
	adapter SandboxAdapterTool,
	call types.ToolCall,
	namespaceTool string,
	mode string,
	action string,
	fallback string,
	sandboxCfg runtimeconfig.SecuritySandboxConfig,
) (types.ToolResult, error) {
	executor := d.runtimeMgr.SandboxExecutor()
	baseDetails := func(profileName string, probeBackend string, sessionMode types.SandboxSessionMode) map[string]any {
		details := map[string]any{
			"namespace_tool": namespaceTool,
		}
		if normalized := strings.ToLower(strings.TrimSpace(mode)); normalized != "" {
			details["sandbox_mode"] = normalized
		}
		if normalized := strings.ToLower(strings.TrimSpace(action)); normalized != "" {
			details["sandbox_action"] = normalized
		}
		if normalized := strings.ToLower(strings.TrimSpace(fallback)); normalized != "" {
			details["sandbox_fallback"] = normalized
		}
		backend := strings.ToLower(strings.TrimSpace(probeBackend))
		if backend == "" {
			backend = strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.Backend))
		}
		if backend != "" {
			details["sandbox_backend"] = backend
		}
		profile := strings.ToLower(strings.TrimSpace(profileName))
		if profile == "" {
			profile = strings.ToLower(strings.TrimSpace(runtimeconfig.ResolveSandboxProfile(sandboxCfg, namespaceTool)))
		}
		if profile != "" {
			details["sandbox_profile"] = profile
		}
		session := strings.ToLower(strings.TrimSpace(string(sessionMode)))
		if session == "" {
			session = strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.SessionMode))
		}
		if session != "" {
			details["sandbox_session_mode"] = session
		}
		required := append([]string(nil), sandboxCfg.Executor.RequiredCapabilities...)
		if len(required) > 0 {
			details["sandbox_required_capabilities"] = required
		}
		return details
	}
	if executor == nil {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonLaunchFailed,
			message: "sandbox executor is unavailable",
			details: baseDetails("", "", ""),
		}
	}
	probe, err := executor.Probe(ctx)
	if err != nil {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonLaunchFailed,
			message: fmt.Sprintf("sandbox executor probe failed: %v", err),
			details: baseDetails("", "", ""),
		}
	}
	missing := make([]string, 0)
	for i := range sandboxCfg.Executor.RequiredCapabilities {
		capability := strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.RequiredCapabilities[i]))
		if capability == "" {
			continue
		}
		if !probe.Supports(capability) {
			missing = append(missing, capability)
		}
	}
	if len(missing) > 0 {
		details := baseDetails("", probe.Backend, "")
		details["missing_capabilities"] = append([]string(nil), missing...)
		details["required_capabilities"] = append([]string(nil), sandboxCfg.Executor.RequiredCapabilities...)
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonCapabilityMismatch,
			message: "sandbox executor capability mismatch",
			details: details,
		}
	}
	sessionMode := types.SandboxSessionMode(strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.SessionMode)))
	if sessionMode != "" && !probe.SupportsSessionMode(sessionMode) {
		details := baseDetails("", probe.Backend, sessionMode)
		details["configured_session_mode"] = string(sessionMode)
		details["supported_session_modes"] = append([]string(nil), probe.SupportedModes...)
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonSessionModeUnsupported,
			message: "sandbox executor does not support configured session mode",
			details: details,
		}
	}

	spec, err := adapter.BuildSandboxExecSpec(ctx, call.Args)
	if err != nil {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonToolNotAdapted,
			message: fmt.Sprintf("sandbox adapter build spec failed: %v", err),
			details: baseDetails("", probe.Backend, sessionMode),
		}
	}
	spec.NamespaceTool = namespaceTool
	spec.SessionMode = sessionMode
	profileName := runtimeconfig.ResolveSandboxProfile(sandboxCfg, namespaceTool)
	if profile, ok := sandboxCfg.Profiles[profileName]; ok {
		if strings.TrimSpace(spec.Network.Mode) == "" {
			spec.Network.Mode = strings.ToLower(strings.TrimSpace(profile.Network.Mode))
		}
		if len(spec.Network.EgressAllowlist) == 0 && len(profile.Network.EgressAllowlist) > 0 {
			spec.Network.EgressAllowlist = append([]string(nil), profile.Network.EgressAllowlist...)
		}
		if len(spec.Mounts) == 0 && len(profile.Mounts) > 0 {
			spec.Mounts = make([]types.SandboxMount, 0, len(profile.Mounts))
			for i := range profile.Mounts {
				spec.Mounts = append(spec.Mounts, types.SandboxMount{
					Source:   strings.TrimSpace(profile.Mounts[i].Source),
					Target:   strings.TrimSpace(profile.Mounts[i].Target),
					ReadOnly: profile.Mounts[i].ReadOnly,
				})
			}
		}
		if spec.ResourceLimits.CPUMilli <= 0 {
			spec.ResourceLimits.CPUMilli = profile.ResourceLimits.CPUMilli
		}
		if spec.ResourceLimits.MemoryBytes <= 0 {
			spec.ResourceLimits.MemoryBytes = profile.ResourceLimits.MemoryBytes
		}
		if spec.ResourceLimits.PIDLimit <= 0 {
			spec.ResourceLimits.PIDLimit = profile.ResourceLimits.PIDLimit
		}
		if spec.LaunchTimeout <= 0 {
			spec.LaunchTimeout = profile.Timeouts.LaunchTimeout
		}
		if spec.ExecTimeout <= 0 {
			spec.ExecTimeout = profile.Timeouts.ExecTimeout
		}
	}
	spec, err = types.NormalizeSandboxExecSpec(spec)
	if err != nil {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonLaunchFailed,
			message: fmt.Sprintf("sandbox spec normalization failed: %v", err),
			details: baseDetails(profileName, probe.Backend, sessionMode),
		}
	}
	execStartedAt := time.Now()
	execResult, err := executor.Execute(ctx, spec)
	execLatencyMs := time.Since(execStartedAt).Milliseconds()
	if execLatencyMs < 0 {
		execLatencyMs = 0
	}
	if err != nil {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonLaunchFailed,
			message: fmt.Sprintf("sandbox execution failed: %v", err),
			details: baseDetails(profileName, probe.Backend, sessionMode),
		}
	}
	if execResult.TimedOut {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  types.SandboxViolationTimeout,
			message: "sandbox execution timed out",
			details: baseDetails(profileName, probe.Backend, sessionMode),
		}
	}
	if execResult.LaunchFailed {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonLaunchFailed,
			message: "sandbox runtime launch failed",
			details: baseDetails(profileName, probe.Backend, sessionMode),
		}
	}
	result, err := adapter.HandleSandboxExecResult(ctx, execResult)
	if err != nil {
		return types.ToolResult{}, &sandboxExecutionError{
			reason:  sandboxReasonLaunchFailed,
			message: fmt.Sprintf("sandbox adapter result handling failed: %v", err),
			details: baseDetails(profileName, probe.Backend, sessionMode),
		}
	}
	result.Structured = annotateSandboxExecStructured(
		result.Structured,
		mode,
		probe.Backend,
		profileName,
		string(sessionMode),
		sandboxCfg.Executor.RequiredCapabilities,
		execLatencyMs,
		execResult,
	)
	return result, nil
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

type sandboxEgressDecision struct {
	Applied      bool
	Action       string
	PolicySource string
	Target       string
}

func (d sandboxEgressDecision) Fields() map[string]any {
	fields := map[string]any{}
	if strings.TrimSpace(d.Action) != "" {
		fields["sandbox_egress_action"] = strings.ToLower(strings.TrimSpace(d.Action))
	}
	if strings.TrimSpace(d.PolicySource) != "" {
		fields["sandbox_egress_policy_source"] = strings.ToLower(strings.TrimSpace(d.PolicySource))
	}
	if strings.TrimSpace(d.Target) != "" {
		fields["sandbox_egress_target"] = strings.ToLower(strings.TrimSpace(d.Target))
	}
	return fields
}

func resolveSandboxEgressDecision(cfg runtimeconfig.SecuritySandboxEgressConfig, namespaceTool string, args map[string]any) sandboxEgressDecision {
	if !cfg.Enabled {
		return sandboxEgressDecision{}
	}
	target, ok := extractSandboxEgressTarget(args)
	if !ok {
		return sandboxEgressDecision{}
	}

	action := strings.ToLower(strings.TrimSpace(cfg.DefaultAction))
	if action == "" {
		action = runtimeconfig.SecuritySandboxEgressActionDeny
	}
	source := "default_action"

	if override, ok := cfg.ByTool[strings.ToLower(strings.TrimSpace(namespaceTool))]; ok {
		action = strings.ToLower(strings.TrimSpace(override))
		source = "by_tool"
	} else if isSandboxEgressTargetAllowed(target, cfg.Allowlist) {
		action = runtimeconfig.SecuritySandboxEgressActionAllow
		source = "allowlist"
	}

	if source != "by_tool" && action == runtimeconfig.SecuritySandboxEgressActionDeny {
		if strings.ToLower(strings.TrimSpace(cfg.OnViolation)) == runtimeconfig.SecuritySandboxEgressOnViolationAllowAndRecord {
			action = runtimeconfig.SecuritySandboxEgressActionAllowAndRecord
			source = "on_violation"
		}
	}

	return sandboxEgressDecision{
		Applied:      true,
		Action:       action,
		PolicySource: source,
		Target:       target,
	}
}

func extractSandboxEgressTarget(args map[string]any) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	keys := []string{
		"egress_target",
		"url",
		"uri",
		"endpoint",
		"host",
		"hostname",
		"domain",
		"address",
	}
	for _, key := range keys {
		value, ok := args[key]
		if !ok {
			continue
		}
		typed, ok := value.(string)
		if !ok {
			continue
		}
		target := normalizeSandboxEgressTarget(typed)
		if target != "" {
			return target, true
		}
	}
	return "", false
}

func normalizeSandboxEgressTarget(raw string) string {
	target := strings.ToLower(strings.TrimSpace(raw))
	if target == "" {
		return ""
	}
	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)
		if err != nil {
			return ""
		}
		target = strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	}
	if host, _, err := net.SplitHostPort(target); err == nil {
		target = strings.ToLower(strings.TrimSpace(host))
	}
	if idx := strings.Index(target, "/"); idx > -1 {
		target = target[:idx]
	}
	target = strings.TrimPrefix(target, "[")
	target = strings.TrimSuffix(target, "]")
	target = strings.TrimSpace(target)
	return target
}

func isSandboxEgressTargetAllowed(target string, allowlist []string) bool {
	normalizedTarget := normalizeSandboxEgressTarget(target)
	if normalizedTarget == "" || len(allowlist) == 0 {
		return false
	}
	for _, entry := range allowlist {
		pattern := strings.ToLower(strings.TrimSpace(entry))
		if pattern == "" {
			continue
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			if suffix != "" && strings.HasSuffix(normalizedTarget, "."+suffix) {
				return true
			}
			continue
		}
		if normalizedTarget == pattern {
			return true
		}
	}
	return false
}

func annotateSandboxStructured(base map[string]any, marker sandboxHostMarker) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	reason := strings.ToLower(strings.TrimSpace(marker.ReasonCode))
	if reason == "" {
		return out
	}
	for key, value := range marker.Fields {
		out[key] = value
	}
	out["sandbox_decision"] = runtimeconfig.SecuritySandboxActionHost
	out["sandbox_reason_code"] = reason
	if reason == sandboxReasonFallbackAllowRecord {
		out["sandbox_fallback_reason"] = reason
		out["sandbox_fallback"] = true
		out["sandbox_fallback_used"] = true
	}
	return out
}

func annotateSandboxExecStructured(
	base map[string]any,
	mode string,
	backend string,
	profile string,
	sessionMode string,
	requiredCapabilities []string,
	execLatencyMs int64,
	execResult types.SandboxExecResult,
) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	if normalized := strings.ToLower(strings.TrimSpace(mode)); normalized != "" {
		out["sandbox_mode"] = normalized
	}
	if normalized := strings.ToLower(strings.TrimSpace(backend)); normalized != "" {
		out["sandbox_backend"] = normalized
	}
	if normalized := strings.ToLower(strings.TrimSpace(profile)); normalized != "" {
		out["sandbox_profile"] = normalized
	}
	if normalized := strings.ToLower(strings.TrimSpace(sessionMode)); normalized != "" {
		out["sandbox_session_mode"] = normalized
	}
	if len(requiredCapabilities) > 0 {
		out["sandbox_required_capabilities"] = append([]string(nil), requiredCapabilities...)
	}
	out["sandbox_decision"] = runtimeconfig.SecuritySandboxActionSandbox
	out["sandbox_exec_latency_ms"] = execLatencyMs
	out["sandbox_exit_code"] = execResult.ExitCode
	if execResult.OOMKilled {
		out["sandbox_oom"] = true
	}
	if execResult.ResourceUsage.CPUTimeMs > 0 {
		out["sandbox_resource_cpu_ms"] = execResult.ResourceUsage.CPUTimeMs
	}
	if execResult.ResourceUsage.MemoryPeakBytes > 0 {
		out["sandbox_resource_memory_peak_bytes"] = execResult.ResourceUsage.MemoryPeakBytes
	}
	if len(execResult.ViolationCodes) > 0 {
		out["sandbox_violation_codes"] = append([]string(nil), execResult.ViolationCodes...)
	}
	return out
}

func sandboxNamespaceToolKey(toolName string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(toolName))
	if normalized == "" {
		return "", false
	}
	if strings.Contains(normalized, "+") {
		parts := strings.Split(normalized, "+")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", false
		}
		return normalized, true
	}
	namespace := "local"
	tool := normalized
	if idx := strings.Index(normalized, "."); idx >= 0 {
		namespace = strings.TrimSpace(normalized[:idx])
		tool = strings.TrimSpace(normalized[idx+1:])
	}
	if namespace == "" || tool == "" {
		return "", false
	}
	return namespace + "+" + tool, true
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
