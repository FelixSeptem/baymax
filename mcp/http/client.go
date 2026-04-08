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
	mcpdiag "github.com/FelixSeptem/baymax/mcp/diag"
	mcpobs "github.com/FelixSeptem/baymax/mcp/internal/observability"
	mcpreliability "github.com/FelixSeptem/baymax/mcp/internal/reliability"
	mcpprofile "github.com/FelixSeptem/baymax/mcp/profile"
	mcpretry "github.com/FelixSeptem/baymax/mcp/retry"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
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
	Profile              mcpprofile.Name
	RuntimePolicy        *types.MCPRuntimePolicy
	RuntimeManager       *runtimeconfig.Manager
	EventHandler         types.EventHandler
	RunID                string
	Connect              ConnectFunc
	HTTPClient           *nethttp.Client
	SessionOptions       *mcp.ClientSessionOptions
}

type Client struct {
	cfg       Config
	explicit  Config
	connect   ConnectFunc
	session   Session
	sessionMu sync.Mutex
	cfgErr    error
	hasEvents bool

	seq          atomic.Uint64
	lastActivity atomic.Int64
	diag         *mcpdiag.Store
}

func NewClient(cfg Config) *Client {
	userCfg := cfg
	profile := cfg.Profile
	if profile == "" {
		profile = mcpprofile.Default
	}
	policy, policyErr := resolveStartupPolicy(cfg, profile)
	cfg = applyRuntimePolicy(cfg, policy)
	cfg = applyExplicitConfig(cfg, userCfg)
	if cfg.MaxReconnects <= 0 {
		cfg.MaxReconnects = 3
	}
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = cfg.CallTimeout
	}
	c := &Client{
		cfg:       cfg,
		explicit:  userCfg,
		cfgErr:    policyErr,
		hasEvents: cfg.EventHandler != nil,
		diag:      mcpdiag.NewStore(200),
	}
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
	if c.cfgErr != nil {
		return nil, c.cfgErr
	}
	ctx, span := otel.Tracer("baymax/mcp/http").Start(ctx, "mcp.list_tools")
	defer span.End()
	runCfg, err := c.runtimeConfig()
	if err != nil {
		return nil, err
	}
	s, err := c.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	result, err := c.withRetry(ctx, runCfg, "list_tools", func(stepCtx context.Context) (types.ToolResult, error) {
		res, callErr := s.ListTools(stepCtx, nil)
		if callErr != nil {
			return types.ToolResult{}, callErr
		}
		m := map[string]any{"tools": make([]map[string]any, 0, len(res.Tools))}
		for _, t := range res.Tools {
			m["tools"] = append(m["tools"].([]map[string]any), map[string]any{"name": t.Name, "description": t.Description, "schema": t.InputSchema})
		}
		return types.ToolResult{Structured: m}, nil
	}, nil, nil)
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
	if c.cfgErr != nil {
		return types.ToolResult{}, c.cfgErr
	}
	ctx, span := otel.Tracer("baymax/mcp/http").Start(ctx, "mcp.call", oteltrace.WithAttributes(oteltraceAttrs(name)...))
	defer span.End()
	runCfg, err := c.runtimeConfig()
	if err != nil {
		return types.ToolResult{}, err
	}
	callID := c.nextCallID()
	start := time.Now()
	reconnectCount := 0
	retryCount := 0
	if c.hasEvents {
		c.emit(ctx, types.Event{
			Version: "v1", Type: "mcp.requested", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
			Payload: map[string]any{"tool": name},
		})
	}

	if err := c.heartbeatIfNeeded(ctx, runCfg.CallTimeout, runCfg.Backoff); err != nil {
		res := types.ToolResult{Error: &types.ClassifiedError{Class: types.ErrMCP, Message: err.Error(), Retryable: true}}
		if c.hasEvents {
			c.emit(ctx, types.Event{
				Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
				Payload: map[string]any{"error_class": string(res.Error.Class)},
			})
		}
		return res, err
	}

	result, err := c.withRetry(ctx, runCfg, callID, func(stepCtx context.Context) (types.ToolResult, error) {
		s, sErr := c.ensureSession(stepCtx)
		if sErr != nil {
			return types.ToolResult{}, sErr
		}
		res, callErr := s.CallTool(stepCtx, &mcp.CallToolParams{Name: name, Arguments: args})
		if callErr != nil {
			return types.ToolResult{}, callErr
		}
		return normalizeCallResult(res), nil
	}, &retryCount, &reconnectCount)
	if err != nil {
		class := types.ErrMCP
		if errors.Is(err, context.DeadlineExceeded) {
			class = types.ErrPolicyTimeout
		}
		res := types.ToolResult{Error: &types.ClassifiedError{Class: class, Message: err.Error(), Retryable: class == types.ErrPolicyTimeout}}
		if c.hasEvents {
			c.emit(ctx, types.Event{
				Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
				Payload: map[string]any{"error_class": string(res.Error.Class)},
			})
		}
		c.recordCall(mcpdiag.CallRecord{
			Time:           time.Now(),
			Transport:      "http",
			Profile:        c.cfg.Profile,
			CallID:         callID,
			Tool:           name,
			LatencyMs:      time.Since(start).Milliseconds(),
			RetryCount:     retryCount,
			ReconnectCount: reconnectCount,
			ErrorClass:     string(class),
		})
		return res, err
	}
	if result.Error != nil {
		if c.hasEvents {
			c.emit(ctx, types.Event{
				Version: "v1", Type: "mcp.failed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
				Payload: map[string]any{"error_class": string(result.Error.Class)},
			})
		}
		c.recordCall(mcpdiag.CallRecord{
			Time:           time.Now(),
			Transport:      "http",
			Profile:        c.cfg.Profile,
			CallID:         callID,
			Tool:           name,
			LatencyMs:      time.Since(start).Milliseconds(),
			RetryCount:     retryCount,
			ReconnectCount: reconnectCount,
			ErrorClass:     string(result.Error.Class),
		})
		return result, nil
	}
	if c.hasEvents {
		c.emit(ctx, types.Event{
			Version: "v1", Type: "mcp.completed", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
			Payload: map[string]any{"tool": name},
		})
	}
	c.recordCall(mcpdiag.CallRecord{
		Time:           time.Now(),
		Transport:      "http",
		Profile:        c.cfg.Profile,
		CallID:         callID,
		Tool:           name,
		LatencyMs:      time.Since(start).Milliseconds(),
		RetryCount:     retryCount,
		ReconnectCount: reconnectCount,
	})
	return result, nil
}

func (c *Client) withRetry(ctx context.Context, runCfg Config, callID string, invoke func(stepCtx context.Context) (types.ToolResult, error), retryCount *int, reconnectCount *int) (types.ToolResult, error) {
	result, finalAttempt, err := mcpreliability.Execute(ctx, mcpreliability.RetryConfig{
		Attempts: runCfg.Retry + 1,
		Timeout:  runCfg.CallTimeout,
		Backoff:  runCfg.Backoff,
	}, mcpreliability.RetryHooks[types.ToolResult]{
		Invoke: func(stepCtx context.Context, attempt int) (types.ToolResult, error) {
			return invoke(stepCtx)
		},
		ShouldRetry: mcpretry.ShouldRetry,
		OnRetry: func(retryCtx context.Context, attempt int, err error) error {
			rc, recErr := c.reconnect(retryCtx, err, runCfg.Backoff, runCfg.MaxReconnects)
			if reconnectCount != nil {
				*reconnectCount += rc
			}
			return recErr
		},
	})
	if retryCount != nil {
		*retryCount = finalAttempt
	}
	if err == nil {
		c.lastActivity.Store(time.Now().UnixNano())
		if c.hasEvents && finalAttempt > 0 {
			c.emit(ctx, types.Event{
				Version: "v1", Type: "mcp.retry", RunID: c.cfg.RunID, CallID: callID, Time: time.Now(),
				Payload: map[string]any{"retry_count": finalAttempt},
			})
		}
	}
	return result, err
}

func (c *Client) heartbeatIfNeeded(ctx context.Context, defaultTimeout time.Duration, backoff time.Duration) error {
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
	timeout := c.cfg.HeartbeatTimeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	hbCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, hbErr := s.ListTools(hbCtx, nil)
	if hbErr == nil {
		c.lastActivity.Store(time.Now().UnixNano())
		return nil
	}
	_, recErr := c.reconnect(ctx, hbErr, backoff, c.cfg.MaxReconnects)
	return recErr
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

func (c *Client) reconnect(ctx context.Context, reason error, backoff time.Duration, maxReconnects int) (int, error) {
	attempted := 0
	c.sessionMu.Lock()
	if c.session != nil {
		_ = c.session.Close()
		c.session = nil
	}
	c.sessionMu.Unlock()
	if c.hasEvents {
		c.emit(ctx, types.Event{
			Version: "v1", Type: "mcp.reconnected", RunID: c.cfg.RunID, Time: time.Now(),
			Payload: map[string]any{"reason": reason.Error()},
		})
	}

	var lastErr error
	for i := 0; i <= maxReconnects; i++ {
		attempted++
		s, err := c.connect(ctx)
		if err == nil {
			c.sessionMu.Lock()
			c.session = s
			c.sessionMu.Unlock()
			c.lastActivity.Store(time.Now().UnixNano())
			return attempted, nil
		}
		lastErr = err
		if i < maxReconnects {
			select {
			case <-ctx.Done():
				return attempted, ctx.Err()
			case <-time.After(mcpretry.BackoffAt(backoff, i)):
			}
		}
	}
	return attempted, lastErr
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
	mcpobs.EmitEvent(ctx, c.cfg.EventHandler, ev)
}

func oteltraceAttrs(tool string) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String("tool.name", tool)}
}

func applyRuntimePolicy(cfg Config, p types.MCPRuntimePolicy) Config {
	if cfg.Profile == "" {
		cfg.Profile = mcpprofile.Default
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
	return cfg
}

func (c *Client) RecentCallSummary(n int) []mcpdiag.CallRecord {
	return mcpobs.RecentCalls(c.diag, c.cfg.RuntimeManager, n)
}

func applyExplicitConfig(cfg Config, user Config) Config {
	if user.CallTimeout > 0 {
		cfg.CallTimeout = user.CallTimeout
	}
	if user.Retry > 0 {
		cfg.Retry = user.Retry
	}
	if user.Backoff > 0 {
		cfg.Backoff = user.Backoff
	}
	if user.MaxReconnects > 0 {
		cfg.MaxReconnects = user.MaxReconnects
	}
	if user.HeartbeatInterval > 0 {
		cfg.HeartbeatInterval = user.HeartbeatInterval
	}
	if user.HeartbeatTimeout > 0 {
		cfg.HeartbeatTimeout = user.HeartbeatTimeout
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
	if out.MaxReconnects <= 0 {
		out.MaxReconnects = 3
	}
	if out.HeartbeatTimeout <= 0 {
		out.HeartbeatTimeout = out.CallTimeout
	}
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
