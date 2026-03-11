package stdio

import (
	"context"
	"errors"
	"os/exec"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OfficialConfig struct {
	Command              *exec.Cmd
	ClientName           string
	ClientVersion        string
	RunID                string
	EventHandler         types.EventHandler
	ClientSessionOptions *mcp.ClientSessionOptions
}

type OfficialClient struct {
	cfg     OfficialConfig
	client  *mcp.Client
	session *mcp.ClientSession
	mu      sync.Mutex
}

func NewOfficialClient(cfg OfficialConfig) *OfficialClient {
	name := cfg.ClientName
	if name == "" {
		name = "baymax-mcp-stdio-client"
	}
	version := cfg.ClientVersion
	if version == "" {
		version = "v1"
	}
	return &OfficialClient{
		cfg: cfg,
		client: mcp.NewClient(&mcp.Implementation{
			Name:    name,
			Version: version,
		}, nil),
	}
}

func (c *OfficialClient) ensureSession(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != nil {
		return nil
	}
	if c.cfg.Command == nil {
		return errors.New("stdio command is nil")
	}
	session, err := c.client.Connect(ctx, &mcp.CommandTransport{Command: c.cfg.Command}, c.cfg.ClientSessionOptions)
	if err != nil {
		return err
	}
	c.session = session
	return nil
}

func (c *OfficialClient) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}
	res, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	tools := make([]types.MCPToolMeta, 0, len(res.Tools))
	for _, t := range res.Tools {
		tools = append(tools, types.MCPToolMeta{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: toMap(t.InputSchema),
		})
	}
	return tools, nil
}

func (c *OfficialClient) CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
	if err := c.ensureSession(ctx); err != nil {
		return types.ToolResult{}, err
	}
	res, err := c.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return types.ToolResult{Error: &types.ClassifiedError{Class: types.ErrMCP, Message: err.Error()}}, err
	}
	result := types.ToolResult{Structured: toMap(res.StructuredContent)}
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
		result.Error = &types.ClassifiedError{Class: types.ErrMCP, Message: result.Content}
	}
	return result, nil
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
