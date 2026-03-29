package stdio

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OfficialConfig struct {
	Command              *exec.Cmd
	ClientName           string
	ClientVersion        string
	RunID                string
	EventHandler         types.EventHandler
	ClientSessionOptions *mcp.ClientSessionOptions
	RuntimeManager       *runtimeconfig.Manager
	SandboxSelector      string
}

type officialSession interface {
	ListTools(ctx context.Context) ([]types.MCPToolMeta, error)
	CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error)
	Close() error
}

type mcpSDKSession struct {
	session *mcp.ClientSession
}

func (s *mcpSDKSession) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	res, err := s.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	tools := make([]types.MCPToolMeta, 0, len(res.Tools))
	for _, item := range res.Tools {
		tools = append(tools, types.MCPToolMeta{
			Name:        item.Name,
			Description: item.Description,
			InputSchema: toMap(item.InputSchema),
		})
	}
	return tools, nil
}

func (s *mcpSDKSession) CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
	res, err := s.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return types.ToolResult{Error: &types.ClassifiedError{Class: types.ErrMCP, Message: err.Error()}}, err
	}
	result := types.ToolResult{Structured: toMap(res.StructuredContent)}
	for _, content := range res.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			if result.Content == "" {
				result.Content = text.Text
			} else {
				result.Content += "\n" + text.Text
			}
		}
	}
	if res.IsError {
		result.Error = &types.ClassifiedError{Class: types.ErrMCP, Message: result.Content}
	}
	return result, nil
}

func (s *mcpSDKSession) Close() error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Close()
}

type officialSandboxPolicy struct {
	Mode        string
	Action      string
	Fallback    string
	SessionMode string
}

type OfficialClient struct {
	cfg OfficialConfig

	connect func(ctx context.Context, command *exec.Cmd, opts *mcp.ClientSessionOptions) (officialSession, error)

	session officialSession
	mu      sync.Mutex
}

func NewOfficialClient(cfg OfficialConfig) *OfficialClient {
	name := strings.TrimSpace(cfg.ClientName)
	if name == "" {
		name = "baymax-mcp-stdio-client"
	}
	version := strings.TrimSpace(cfg.ClientVersion)
	if version == "" {
		version = "v1"
	}
	client := mcp.NewClient(&mcp.Implementation{Name: name, Version: version}, nil)
	return &OfficialClient{
		cfg: cfg,
		connect: func(ctx context.Context, command *exec.Cmd, opts *mcp.ClientSessionOptions) (officialSession, error) {
			session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, opts)
			if err != nil {
				return nil, err
			}
			return &mcpSDKSession{session: session}, nil
		},
	}
}

func (c *OfficialClient) resolveSandboxPolicy() officialSandboxPolicy {
	policy := officialSandboxPolicy{
		Mode:        runtimeconfig.SecuritySandboxModeObserve,
		Action:      runtimeconfig.SecuritySandboxActionHost,
		Fallback:    runtimeconfig.SecuritySandboxFallbackAllowAndRecord,
		SessionMode: runtimeconfig.SecuritySandboxSessionModePerSession,
	}
	if c == nil || c.cfg.RuntimeManager == nil {
		return policy
	}
	cfg := c.cfg.RuntimeManager.EffectiveConfig().Security.Sandbox
	policy.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	policy.SessionMode = strings.ToLower(strings.TrimSpace(cfg.Executor.SessionMode))
	if policy.SessionMode != runtimeconfig.SecuritySandboxSessionModePerCall &&
		policy.SessionMode != runtimeconfig.SecuritySandboxSessionModePerSession {
		policy.SessionMode = runtimeconfig.SecuritySandboxSessionModePerSession
	}
	selector := strings.ToLower(strings.TrimSpace(c.cfg.SandboxSelector))
	if selector == "" {
		selector = runtimeconfig.SecuritySandboxSelectorMCPStdioCommand
	}
	if !cfg.Enabled {
		return policy
	}
	if policy.Mode != runtimeconfig.SecuritySandboxModeEnforce {
		policy.Action = runtimeconfig.SecuritySandboxActionHost
		return policy
	}
	policy.Action = runtimeconfig.ResolveSandboxAction(cfg, selector)
	policy.Fallback = runtimeconfig.ResolveSandboxFallbackAction(cfg, selector)
	return policy
}

func (c *OfficialClient) ensureSession(ctx context.Context, policy officialSandboxPolicy) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if policy.Action == runtimeconfig.SecuritySandboxActionDeny {
		return errors.New("sandbox.policy_deny: mcp stdio command startup denied by policy")
	}
	if policy.Action == runtimeconfig.SecuritySandboxActionSandbox && policy.Fallback == runtimeconfig.SecuritySandboxFallbackDeny {
		return errors.New("sandbox.tool_not_adapted: mcp stdio sandbox launcher is not available")
	}

	perCall := policy.SessionMode == runtimeconfig.SecuritySandboxSessionModePerCall
	if perCall && c.session != nil {
		_ = c.session.Close()
		c.session = nil
	}
	if c.session != nil {
		return nil
	}
	if c.cfg.Command == nil {
		return errors.New("stdio command is nil")
	}
	session, err := c.connect(ctx, c.cfg.Command, c.cfg.ClientSessionOptions)
	if err != nil {
		return err
	}
	c.session = session
	return nil
}

func (c *OfficialClient) sessionSnapshot() officialSession {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session
}

func (c *OfficialClient) resetSession() {
	c.mu.Lock()
	if c.session != nil {
		_ = c.session.Close()
		c.session = nil
	}
	c.mu.Unlock()
}

func (c *OfficialClient) cleanupPerCallSession(policy officialSandboxPolicy) {
	if policy.SessionMode != runtimeconfig.SecuritySandboxSessionModePerCall {
		return
	}
	c.mu.Lock()
	if c.session != nil {
		_ = c.session.Close()
		c.session = nil
	}
	c.mu.Unlock()
}

func (c *OfficialClient) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	policy := c.resolveSandboxPolicy()
	if err := c.ensureSession(ctx, policy); err != nil {
		return nil, err
	}
	session := c.sessionSnapshot()
	if session == nil {
		return nil, errors.New("stdio session is unavailable")
	}
	tools, err := session.ListTools(ctx)
	if err != nil && policy.SessionMode == runtimeconfig.SecuritySandboxSessionModePerSession && shouldReconnectSessionError(err) {
		c.resetSession()
		if reconnectErr := c.ensureSession(ctx, policy); reconnectErr != nil {
			return nil, reconnectErr
		}
		session = c.sessionSnapshot()
		if session == nil {
			return nil, errors.New("stdio session reconnect failed")
		}
		tools, err = session.ListTools(ctx)
	}
	c.cleanupPerCallSession(policy)
	return tools, err
}

func (c *OfficialClient) CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
	policy := c.resolveSandboxPolicy()
	if err := c.ensureSession(ctx, policy); err != nil {
		return types.ToolResult{}, err
	}
	session := c.sessionSnapshot()
	if session == nil {
		return types.ToolResult{}, errors.New("stdio session is unavailable")
	}
	result, err := session.CallTool(ctx, name, args)
	if err != nil && policy.SessionMode == runtimeconfig.SecuritySandboxSessionModePerSession && shouldReconnectSessionError(err) {
		c.resetSession()
		if reconnectErr := c.ensureSession(ctx, policy); reconnectErr != nil {
			return types.ToolResult{}, reconnectErr
		}
		session = c.sessionSnapshot()
		if session == nil {
			return types.ToolResult{}, errors.New("stdio session reconnect failed")
		}
		result, err = session.CallTool(ctx, name, args)
	}
	c.cleanupPerCallSession(policy)
	return result, err
}

func (c *OfficialClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		return nil
	}
	err := c.session.Close()
	c.session = nil
	return err
}

func shouldReconnectSessionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "eof"),
		strings.Contains(msg, "closed"):
		return true
	default:
		return false
	}
}

func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if ok {
		return m
	}
	return map[string]any{"value": v}
}

var _ types.MCPClient = (*OfficialClient)(nil)
