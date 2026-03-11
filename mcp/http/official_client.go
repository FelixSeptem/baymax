package http

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OfficialConfig struct {
	Endpoint             string
	Headers              map[string]string
	ClientName           string
	ClientVersion        string
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
		name = "baymax-mcp-http-client"
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
	if c.cfg.Endpoint == "" {
		return errors.New("mcp endpoint is empty")
	}

	httpClient := &http.Client{Transport: &headerRoundTripper{base: http.DefaultTransport, headers: c.cfg.Headers}}
	transport := &mcp.StreamableClientTransport{Endpoint: c.cfg.Endpoint, HTTPClient: httpClient}
	session, err := c.client.Connect(ctx, transport, c.cfg.ClientSessionOptions)
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
		tools = append(tools, types.MCPToolMeta{Name: t.Name, Description: t.Description})
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
	result := types.ToolResult{}
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

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	for k, v := range h.headers {
		cloned.Header.Set(k, v)
	}
	return h.base.RoundTrip(cloned)
}

var _ types.MCPClient = (*OfficialClient)(nil)
