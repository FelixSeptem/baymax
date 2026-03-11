package http

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	obsTrace "github.com/FelixSeptem/baymax/observability/trace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Session interface {
	ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
	Close() error
}

type ConnectFunc func(ctx context.Context) (Session, error)

type Config struct {
	Endpoint             string
	Headers              map[string]string
	ClientName           string
	ClientVersion        string
	CallTimeout          time.Duration
	Retry                int
	Backoff              time.Duration
	MaxReconnects        int
	HeartbeatInterval    time.Duration
	HeartbeatTimeout     time.Duration
	DisableStandaloneSSE bool
	EventHandler         types.EventHandler
	RunID                string
	Connect              ConnectFunc
	HTTPClient           *nethttp.Client
	SessionOptions       *mcp.ClientSessionOptions
}

type Client struct {
	cfg       Config
	connect   ConnectFunc
	session   Session
	sessionMu sync.Mutex

	seq          atomic.Uint64
	lastActivity atomic.Int64
}

func NewClient(cfg Config) *Client {
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = 10 * time.Second
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 50 * time.Millisecond
	}
	if cfg.MaxReconnects <= 0 {
		cfg.MaxReconnects = 3
	}
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = cfg.CallTimeout
	}
	c := &Client{cfg: cfg}
	if cfg.Connect != nil {
		c.connect = cfg.Connect
	} else {
		c.connect = c.defaultConnect
	}
	c.lastActivity.Store(time.Now().UnixNano())
	return c
}

func (c *Client) defaultConnect(ctx context.Context) (Session, error) {
	if c.cfg.Endpoint == "" {
		return nil, errors.New("mcp endpoint is empty")
	}
	name := c.cfg.ClientName
	if name == "" {
		name = "baymax-mcp-http-client"
	}
	version := c.cfg.ClientVersion
	if version == "" {
		version = "v1"
	}
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: name, Version: version}, nil)
	httpClient := c.cfg.HTTPClient
	if httpClient == nil {
		httpClient = &nethttp.Client{Transport: &headerRoundTripper{base: nethttp.DefaultTransport, headers: c.cfg.Headers}}
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint:             c.cfg.Endpoint,
		HTTPClient:           httpClient,
		MaxRetries:           c.cfg.MaxReconnects,
		DisableStandaloneSSE: c.cfg.DisableStandaloneSSE,
	}
	s, err := mcpClient.Connect(ctx, transport, c.cfg.SessionOptions)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *Client) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	ctx, span := otel.Tracer("baymax/mcp/http").Start(ctx, "mcp.list_tools")
	defer span.End()
	s, err := c.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	result, err := c.withRetry(ctx, "list_tools", func(stepCtx context.Context) (types.ToolResult, error) {
		res, callErr := s.ListTools(stepCtx, nil)
		if callErr != nil {
			return types.ToolResult{}, callErr
		}
		m := map[string]any{"tools": make([]map[string]any, 0, len(res.Tools))}
		for _, t := range res.Tools {
			m["tools"] = append(m["tools"].([]map[string]any), map[string]any{"name": t.Name, "description": t.Description, "schema": t.InputSchema})
		}
		return types.ToolResult{Structured: m}, nil
	})
	if err != nil {
		return nil, err
	}
	toolsRaw, _ := result.Structured["tools"].([]map[string]any)
	tools := make([]types.MCPToolMeta, 0, len(toolsRaw))
	for _, item := range toolsRaw {
		tools = append(tools, types.MCPToolMeta{
			Name:        toString(item["name"]),
			Description: toString(item["description"]),
			InputSchema: asMap(item["schema"]),
		})
	}
	return tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
	ctx, span := otel.Tracer("baymax/mcp/http").Start(ctx, "mcp.call", oteltrace.WithAttributes(oteltraceAttrs(name)...))
	defer span.End()
	callID := c.nextCallID()
	c.emit(ctx, types.Event{Version: "v1", Type: "mcp.requested", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"tool": name}})

	if err := c.heartbeatIfNeeded(ctx); err != nil {
		res := types.ToolResult{Error: &types.ClassifiedError{Class: types.ErrMCP, Message: err.Error(), Retryable: true}}
		c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(res.Error.Class)}})
		return res, err
	}

	result, err := c.withRetry(ctx, callID, func(stepCtx context.Context) (types.ToolResult, error) {
		s, sErr := c.ensureSession(stepCtx)
		if sErr != nil {
			return types.ToolResult{}, sErr
		}
		res, callErr := s.CallTool(stepCtx, &mcp.CallToolParams{Name: name, Arguments: args})
		if callErr != nil {
			return types.ToolResult{}, callErr
		}
		return normalizeCallResult(res), nil
	})
	if err != nil {
		class := types.ErrMCP
		if errors.Is(err, context.DeadlineExceeded) {
			class = types.ErrPolicyTimeout
		}
		res := types.ToolResult{Error: &types.ClassifiedError{Class: class, Message: err.Error(), Retryable: class == types.ErrPolicyTimeout}}
		c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(res.Error.Class)}})
		return res, err
	}
	if result.Error != nil {
		c.emit(ctx, types.Event{Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"error_class": string(result.Error.Class)}})
		return result, nil
	}
	c.emit(ctx, types.Event{Version: "v1", Type: "mcp.completed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(), Payload: map[string]any{"tool": name}})
	return result, nil
}

func (c *Client) withRetry(ctx context.Context, callID string, invoke func(stepCtx context.Context) (types.ToolResult, error)) (types.ToolResult, error) {
	attempts := c.cfg.Retry + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		stepCtx, cancel := context.WithTimeout(ctx, c.cfg.CallTimeout)
		res, err := invoke(stepCtx)
		cancel()
		if err == nil {
			c.lastActivity.Store(time.Now().UnixNano())
			return res, nil
		}
		lastErr = err
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
			return types.ToolResult{}, context.DeadlineExceeded
		}
		if i < attempts-1 {
			if recErr := c.reconnect(ctx, err); recErr != nil {
				lastErr = recErr
			}
			select {
			case <-ctx.Done():
				return types.ToolResult{}, ctx.Err()
			case <-time.After(backoffAt(c.cfg.Backoff, i)):
			}
		}
	}
	return types.ToolResult{}, lastErr
}

func (c *Client) heartbeatIfNeeded(ctx context.Context) error {
	if c.cfg.HeartbeatInterval <= 0 {
		return nil
	}
	last := time.Unix(0, c.lastActivity.Load())
	if time.Since(last) < c.cfg.HeartbeatInterval {
		return nil
	}
	s, err := c.ensureSession(ctx)
	if err != nil {
		return err
	}
	hbCtx, cancel := context.WithTimeout(ctx, c.cfg.HeartbeatTimeout)
	defer cancel()
	_, hbErr := s.ListTools(hbCtx, nil)
	if hbErr == nil {
		c.lastActivity.Store(time.Now().UnixNano())
		return nil
	}
	return c.reconnect(ctx, hbErr)
}

func (c *Client) ensureSession(ctx context.Context) (Session, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if c.session != nil {
		return c.session, nil
	}
	s, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	c.session = s
	return s, nil
}

func (c *Client) reconnect(ctx context.Context, reason error) error {
	c.sessionMu.Lock()
	if c.session != nil {
		_ = c.session.Close()
		c.session = nil
	}
	c.sessionMu.Unlock()
	c.emit(ctx, types.Event{Version: "v1", Type: "mcp.reconnected", RunID: c.cfg.RunID, Time: time.Now(), Payload: map[string]any{"reason": reason.Error()}})

	var lastErr error
	for i := 0; i <= c.cfg.MaxReconnects; i++ {
		s, err := c.connect(ctx)
		if err == nil {
			c.sessionMu.Lock()
			c.session = s
			c.sessionMu.Unlock()
			c.lastActivity.Store(time.Now().UnixNano())
			return nil
		}
		lastErr = err
		if i < c.cfg.MaxReconnects {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffAt(c.cfg.Backoff, i)):
			}
		}
	}
	return lastErr
}

func (c *Client) nextCallID() string {
	id := c.seq.Add(1)
	return fmt.Sprintf("mcp-http-%d", id)
}

func (c *Client) Close() error {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if c.session == nil {
		return nil
	}
	err := c.session.Close()
	c.session = nil
	return err
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

func oteltraceAttrs(tool string) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String("tool.name", tool)}
}

func backoffAt(base time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return base
	}
	d := base
	for i := 0; i < attempt; i++ {
		d *= 2
		if d > 2*time.Second {
			return 2 * time.Second
		}
	}
	return d
}

func normalizeCallResult(res *mcp.CallToolResult) types.ToolResult {
	result := types.ToolResult{Structured: asMap(res.StructuredContent)}
	for _, ctn := range res.Content {
		if text, ok := ctn.(*mcp.TextContent); ok {
			if result.Content == "" {
				result.Content = text.Text
			} else {
				result.Content += "\n" + text.Text
			}
		}
	}
	if res.IsError {
		result.Error = &types.ClassifiedError{Class: types.ErrMCP, Message: result.Content, Retryable: false}
	}
	return result
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}

func asMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if ok {
		return m
	}
	return map[string]any{"value": v}
}

var _ types.MCPClient = (*Client)(nil)
