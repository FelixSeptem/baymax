package stdio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	mcpdiag "github.com/FelixSeptem/baymax/mcp/diag"
	mcpobs "github.com/FelixSeptem/baymax/mcp/internal/observability"
	mcpreliability "github.com/FelixSeptem/baymax/mcp/internal/reliability"
	mcpprofile "github.com/FelixSeptem/baymax/mcp/profile"
	mcpretry "github.com/FelixSeptem/baymax/mcp/retry"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Transport interface {
	Initialize(ctx context.Context) error
	ListTools(ctx context.Context) ([]types.MCPToolMeta, error)
	CallTool(ctx context.Context, name string, args map[string]any) (Response, error)
	Close() error
}

type Response struct {
	Content    string
	Structured map[string]any
	Error      string
}

type Config struct {
	ReadPoolSize   int
	WritePoolSize  int
	CallTimeout    time.Duration
	Retry          int
	Backoff        time.Duration
	QueueSize      int
	Backpressure   types.BackpressureMode
	Profile        mcpprofile.Name
	RuntimePolicy  *types.MCPRuntimePolicy
	RuntimeManager *runtimeconfig.Manager
	EventHandler   types.EventHandler
	RunID          string
}

type Client struct {
	transport Transport
	cfg       Config
	explicit  Config

	initialized atomic.Bool
	initMu      sync.Mutex
	initErr     error

	readPool  chan struct{}
	writePool chan struct{}
	diag      *mcpdiag.Store
}

func NewClient(transport Transport, cfg Config) *Client {
	userCfg := cfg
	profile := cfg.Profile
	if profile == "" {
		profile = mcpprofile.Default
	}
	policy, policyErr := resolveStartupPolicy(cfg, profile)
	cfg = applyRuntimePolicy(cfg, policy)
	cfg = applyExplicitConfig(cfg, userCfg)
	if cfg.Profile == "" {
		cfg.Profile = profile
	}

	return &Client{
		transport: transport,
		cfg:       cfg,
		explicit:  userCfg,
		initErr:   policyErr,
		readPool:  make(chan struct{}, cfg.ReadPoolSize),
		writePool: make(chan struct{}, cfg.WritePoolSize),
		diag:      mcpdiag.NewStore(200),
	}
}

func (c *Client) Warmup(ctx context.Context) error {
	return c.ensureInitialized(ctx)
}

func (c *Client) ensureInitialized(ctx context.Context) error {
	if c.initErr != nil {
		return c.initErr
	}
	if c.initialized.Load() {
		return c.initErr
	}
	c.initMu.Lock()
	defer c.initMu.Unlock()
	if c.initialized.Load() {
		return c.initErr
	}
	if c.transport == nil {
		c.initErr = errors.New("stdio transport is nil")
		c.initialized.Store(true)
		return c.initErr
	}
	if err := c.transport.Initialize(ctx); err != nil {
		c.initErr = err
		c.initialized.Store(true)
		return err
	}
	_, err := c.transport.ListTools(ctx)
	c.initErr = err
	c.initialized.Store(true)
	return err
}

func (c *Client) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.CallTimeout)
	defer cancel()
	return c.transport.ListTools(ctx)
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
	ctx, span := otel.Tracer("baymax/mcp/stdio").Start(ctx, "mcp.call", oteltrace.WithAttributes(traceAttrs(name)...))
	defer span.End()
	if err := c.ensureInitialized(ctx); err != nil {
		return types.ToolResult{}, err
	}
	runCfg, err := c.runtimeConfig()
	if err != nil {
		return types.ToolResult{}, err
	}
	callID := fmt.Sprintf("mcp-stdio-%d", time.Now().UnixNano())
	start := time.Now()
	retryCount := 0
	c.emit(ctx, types.Event{Version: "v1", Type: "mcp.requested", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"tool": name}})

	queueStart := time.Now()
	release := c.acquirePool(ctx, isWriteCall(args), callID)
	if release == nil {
		err := context.Canceled
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			err = errors.New("queue rejected by backpressure policy")
		}
		res := failedTimeout(err)
		c.emit(ctx, types.Event{
			Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
			Payload: map[string]any{
				"error_class":   string(res.Error.Class),
				"queue_wait_ms": time.Since(queueStart).Milliseconds(),
				"backpressure":  runCfg.Backpressure,
			},
		})
		c.recordCall(mcpdiag.CallRecord{
			Time:           time.Now(),
			Transport:      "stdio",
			Profile:        c.cfg.Profile,
			CallID:         callID,
			Tool:           name,
			LatencyMs:      time.Since(start).Milliseconds(),
			RetryCount:     retryCount,
			ReconnectCount: 0,
			ErrorClass:     string(res.Error.Class),
		})
		return res, err
	}
	defer release()

	resp, finalAttempt, err := mcpreliability.Execute(ctx, mcpreliability.RetryConfig{
		Attempts: runCfg.Retry + 1,
		Timeout:  runCfg.CallTimeout,
		Backoff:  runCfg.Backoff,
	}, mcpreliability.RetryHooks[Response]{
		Invoke: func(stepCtx context.Context, attempt int) (Response, error) {
			return c.invokeAsync(stepCtx, name, args)
		},
		ShouldRetry: mcpretry.ShouldRetry,
	})
	retryCount = finalAttempt
	if err == nil {
		result := normalizeResponse(resp)
		if result.Error != nil {
			c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
			return result, nil
		}
		c.emit(ctx, types.Event{
			Version: "v1", Type: "mcp.completed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
			Payload: map[string]any{
				"tool": name, "retry_count": finalAttempt, "queue_wait_ms": time.Since(queueStart).Milliseconds(),
			},
		})
		c.recordCall(mcpdiag.CallRecord{
			Time:           time.Now(),
			Transport:      "stdio",
			Profile:        c.cfg.Profile,
			CallID:         callID,
			Tool:           name,
			LatencyMs:      time.Since(start).Milliseconds(),
			RetryCount:     finalAttempt,
			ReconnectCount: 0,
		})
		return result, nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		result := failedTimeout(err)
		c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
		return result, err
	}
	result := types.ToolResult{Error: &types.ClassifiedError{Class: types.ErrMCP, Message: err.Error(), Retryable: false}}
	c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
	c.recordCall(mcpdiag.CallRecord{
		Time:           time.Now(),
		Transport:      "stdio",
		Profile:        c.cfg.Profile,
		CallID:         callID,
		Tool:           name,
		LatencyMs:      time.Since(start).Milliseconds(),
		RetryCount:     retryCount,
		ReconnectCount: 0,
		ErrorClass:     string(result.Error.Class),
	})
	return result, err
}

func (c *Client) acquirePool(ctx context.Context, isWrite bool, callID string) func() {
	pool := c.readPool
	if isWrite {
		pool = c.writePool
	}
	runCfg, err := c.runtimeConfig()
	if err != nil {
		return nil
	}
	if runCfg.Backpressure == types.BackpressureReject {
		select {
		case pool <- struct{}{}:
			return func() { <-pool }
		default:
			c.emit(ctx, types.Event{
				Version: "v1", Type: "mcp.queue_rejected", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
				Payload: map[string]any{"backpressure": runCfg.Backpressure},
			})
			return nil
		}
	}
	select {
	case <-ctx.Done():
		return nil
	case pool <- struct{}{}:
		return func() { <-pool }
	}
}

func (c *Client) invokeAsync(ctx context.Context, name string, args map[string]any) (Response, error) {
	type callResult struct {
		resp Response
		err  error
	}
	ch := make(chan callResult, 1)
	go func() {
		resp, err := c.transport.CallTool(ctx, name, args)
		select {
		case ch <- callResult{resp: resp, err: err}:
		case <-ctx.Done():
		}
	}()
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case out := <-ch:
		return out.resp, out.err
	}
}

func normalizeResponse(resp Response) types.ToolResult {
	result := types.ToolResult{Content: resp.Content, Structured: resp.Structured}
	if resp.Error != "" {
		result.Error = &types.ClassifiedError{Class: types.ErrMCP, Message: resp.Error, Retryable: false}
	}
	return result
}

func failedTimeout(err error) types.ToolResult {
	msg := "timeout"
	if err != nil {
		msg = err.Error()
	}
	return types.ToolResult{Error: &types.ClassifiedError{Class: types.ErrPolicyTimeout, Message: msg, Retryable: true}}
}

func isWriteCall(args map[string]any) bool {
	if args == nil {
		return false
	}
	if v, ok := args["_write"].(bool); ok {
		return v
	}
	return false
}

func (c *Client) emit(ctx context.Context, ev types.Event) {
	mcpobs.EmitEvent(ctx, c.cfg.EventHandler, ev)
}

func traceAttrs(tool string) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String("tool.name", tool)}
}

func (c *Client) Close() error {
	if c.transport == nil {
		return nil
	}
	return c.transport.Close()
}

func applyRuntimePolicy(cfg Config, p types.MCPRuntimePolicy) Config {
	if cfg.Profile == "" {
		cfg.Profile = mcpprofile.Default
	}
	if p.ReadPoolSize > 0 {
		cfg.ReadPoolSize = p.ReadPoolSize
	}
	if p.WritePoolSize > 0 {
		cfg.WritePoolSize = p.WritePoolSize
	}
	if p.CallTimeout > 0 {
		cfg.CallTimeout = p.CallTimeout
	}
	if p.Retry >= 0 {
		cfg.Retry = p.Retry
	}
	if p.Backoff > 0 {
		cfg.Backoff = p.Backoff
	}
	if p.QueueSize > 0 {
		cfg.QueueSize = p.QueueSize
	}
	if p.Backpressure != "" {
		cfg.Backpressure = p.Backpressure
	}
	return cfg
}

func (c *Client) RecentCallSummary(n int) []mcpdiag.CallRecord {
	return mcpobs.RecentCalls(c.diag, c.cfg.RuntimeManager, n)
}

func applyExplicitConfig(cfg Config, user Config) Config {
	if user.ReadPoolSize > 0 {
		cfg.ReadPoolSize = user.ReadPoolSize
	}
	if user.WritePoolSize > 0 {
		cfg.WritePoolSize = user.WritePoolSize
	}
	if user.CallTimeout > 0 {
		cfg.CallTimeout = user.CallTimeout
	}
	if user.Retry > 0 {
		cfg.Retry = user.Retry
	}
	if user.Backoff > 0 {
		cfg.Backoff = user.Backoff
	}
	if user.QueueSize > 0 {
		cfg.QueueSize = user.QueueSize
	}
	if user.Backpressure != "" {
		cfg.Backpressure = user.Backpressure
	}
	return cfg
}

func (c *Client) runtimeConfig() (Config, error) {
	policy, err := resolveRuntimePolicy(c.cfg, c.explicit)
	if err != nil {
		return Config{}, err
	}
	out := applyRuntimePolicy(c.cfg, policy)
	out = applyExplicitConfig(out, c.explicit)
	return out, nil
}

func resolveStartupPolicy(cfg Config, profile mcpprofile.Name) (types.MCPRuntimePolicy, error) {
	return mcpreliability.ResolveStartupPolicy(profile, cfg.RuntimeManager, cfg.RuntimePolicy)
}

func resolveRuntimePolicy(cfg Config, explicit Config) (types.MCPRuntimePolicy, error) {
	return mcpreliability.ResolveRuntimePolicy(cfg.Profile, cfg.RuntimeManager, explicit.RuntimePolicy)
}

func (c *Client) recordCall(rec mcpdiag.CallRecord) {
	mcpobs.RecordCall(c.diag, c.cfg.RuntimeManager, c.cfg.RunID, rec)
}

var _ types.MCPClient = (*Client)(nil)
