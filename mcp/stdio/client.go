package stdio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	obsTrace "github.com/FelixSeptem/baymax/observability/trace"
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
	ReadPoolSize  int
	WritePoolSize int
	CallTimeout   time.Duration
	Retry         int
	Backoff       time.Duration
	EventHandler  types.EventHandler
	RunID         string
}

type Client struct {
	transport Transport
	cfg       Config

	initialized atomic.Bool
	initMu      sync.Mutex
	initErr     error

	readPool  chan struct{}
	writePool chan struct{}
}

func NewClient(transport Transport, cfg Config) *Client {
	if cfg.ReadPoolSize <= 0 {
		cfg.ReadPoolSize = 4
	}
	if cfg.WritePoolSize <= 0 {
		cfg.WritePoolSize = 1
	}
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = 10 * time.Second
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 50 * time.Millisecond
	}

	return &Client{
		transport: transport,
		cfg:       cfg,
		readPool:  make(chan struct{}, cfg.ReadPoolSize),
		writePool: make(chan struct{}, cfg.WritePoolSize),
	}
}

func (c *Client) Warmup(ctx context.Context) error {
	return c.ensureInitialized(ctx)
}

func (c *Client) ensureInitialized(ctx context.Context) error {
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
	callID := fmt.Sprintf("mcp-stdio-%d", time.Now().UnixNano())
	c.emit(ctx, types.Event{Version: "v1", Type: "mcp.requested", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"tool": name}})

	release := c.acquirePool(ctx, isWriteCall(args), callID)
	if release == nil {
		err := context.Canceled
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		res := failedTimeout(err)
		c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(res.Error.Class)}})
		return res, err
	}
	defer release()

	attempts := c.cfg.Retry + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		stepCtx, cancel := context.WithTimeout(ctx, c.cfg.CallTimeout)
		resp, err := c.transport.CallTool(stepCtx, name, args)
		cancel()
		if err == nil {
			result := normalizeResponse(resp)
			if result.Error != nil {
				c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
				return result, nil
			}
			c.emit(ctx, types.Event{Version: "v1", Type: "mcp.completed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"tool": name}})
			return result, nil
		}
		lastErr = err
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
			result := failedTimeout(err)
			c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
			return result, err
		}
		if i < attempts-1 {
			select {
			case <-ctx.Done():
				result := failedTimeout(ctx.Err())
				c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
				return result, ctx.Err()
			case <-time.After(c.cfg.Backoff):
			}
		}
	}
	result := types.ToolResult{Error: &types.ClassifiedError{Class: types.ErrMCP, Message: lastErr.Error(), Retryable: false}}
	c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
	return result, lastErr
}

func (c *Client) acquirePool(ctx context.Context, isWrite bool, callID string) func() {
	pool := c.readPool
	if isWrite {
		pool = c.writePool
	}
	select {
	case <-ctx.Done():
		return nil
	case pool <- struct{}{}:
		return func() { <-pool }
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
	if c.cfg.EventHandler == nil {
		return
	}
	if ev.Version == "" {
		ev.Version = types.EventSchemaVersionV1
	}
	ev.TraceID = obsTrace.TraceIDFromContext(ctx)
	ev.SpanID = obsTrace.SpanIDFromContext(ctx)
	c.cfg.EventHandler.OnEvent(ctx, ev)
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

var _ types.MCPClient = (*Client)(nil)
