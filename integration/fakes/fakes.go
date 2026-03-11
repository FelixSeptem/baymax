package fakes

import (
	"context"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
)

type ModelStep struct {
	Response types.ModelResponse
	Err      error
}

type Model struct {
	mu          sync.Mutex
	steps       []ModelStep
	stream      []types.ModelEvent
	streamErr   error
	calls       int
	lastRequest types.ModelRequest
}

func NewModel(steps []ModelStep) *Model {
	return &Model{steps: steps}
}

func (m *Model) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastRequest = req
	idx := m.calls
	m.calls++
	if idx >= len(m.steps) {
		return types.ModelResponse{FinalAnswer: "done"}, nil
	}
	return m.steps[idx].Response, m.steps[idx].Err
}

func (m *Model) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	m.mu.Lock()
	stream := append([]types.ModelEvent(nil), m.stream...)
	err := m.streamErr
	m.lastRequest = req
	m.mu.Unlock()
	for _, ev := range stream {
		if err := onEvent(ev); err != nil {
			return err
		}
	}
	return err
}

func (m *Model) SetStream(events []types.ModelEvent, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stream = append([]types.ModelEvent(nil), events...)
	m.streamErr = err
}

func (m *Model) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *Model) LastRequest() types.ModelRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastRequest
}

type Tool struct {
	NameValue   string
	SchemaValue map[string]any
	InvokeFn    func(ctx context.Context, args map[string]any) (types.ToolResult, error)
}

func (t *Tool) Name() string               { return t.NameValue }
func (t *Tool) Description() string        { return "fake tool" }
func (t *Tool) JSONSchema() map[string]any { return t.SchemaValue }
func (t *Tool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	if t.InvokeFn != nil {
		return t.InvokeFn(ctx, args)
	}
	return types.ToolResult{Content: "ok"}, nil
}

type MCP struct {
	ListFn func(ctx context.Context) ([]types.MCPToolMeta, error)
	CallFn func(ctx context.Context, name string, args map[string]any) (types.ToolResult, error)
}

func (m *MCP) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}
	return []types.MCPToolMeta{{Name: "fake.mcp"}}, nil
}

func (m *MCP) CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
	if m.CallFn != nil {
		return m.CallFn(ctx, name, args)
	}
	return types.ToolResult{Content: "mcp-ok"}, nil
}

func (m *MCP) Close() error { return nil }
