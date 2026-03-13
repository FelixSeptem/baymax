package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

type fanoutModel struct{ calls int32 }

func (m *fanoutModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	if atomic.AddInt32(&m.calls, 1) == 1 {
		return types.ModelResponse{
			ToolCalls: []types.ToolCall{
				{CallID: "c1", Name: "local.work_a", Args: map[string]any{"ms": 120}},
				{CallID: "c2", Name: "local.work_b", Args: map[string]any{"ms": 80}},
				{CallID: "c3", Name: "local.work_c", Args: map[string]any{"ms": 100}},
			},
		}, nil
	}
	return types.ModelResponse{FinalAnswer: "parallel fanout done"}, nil
}

func (m *fanoutModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return nil
}

type workerTool struct {
	name string
}

type multiHandler struct {
	handlers []types.EventHandler
}

func (h *multiHandler) OnEvent(ctx context.Context, ev types.Event) {
	for _, item := range h.handlers {
		item.OnEvent(ctx, ev)
	}
}

func (t *workerTool) Name() string        { return t.name }
func (t *workerTool) Description() string { return "parallel worker " + t.name }
func (t *workerTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"ms"},
		"properties": map[string]any{
			"ms": map[string]any{"type": "integer", "minimum": 1},
		},
	}
}
func (t *workerTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	ms, _ := args["ms"].(int)
	if ms == 0 {
		if f, ok := args["ms"].(float64); ok {
			ms = int(f)
		}
	}
	select {
	case <-ctx.Done():
		return types.ToolResult{}, ctx.Err()
	case <-time.After(time.Duration(ms) * time.Millisecond):
	}
	return types.ToolResult{Content: fmt.Sprintf("%s finished in %dms", t.name, ms)}, nil
}

func main() {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	_, _ = reg.Register(&workerTool{name: "work_a"})
	_, _ = reg.Register(&workerTool{name: "work_b"})
	_, _ = reg.Register(&workerTool{name: "work_c"})

	handler := &multiHandler{
		handlers: []types.EventHandler{
			event.NewJSONLoggerWithRuntimeManager(os.Stdout, mgr),
			event.NewRuntimeRecorder(mgr),
		},
	}

	eng := runner.New(&fanoutModel{}, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	res, err := eng.Run(context.Background(), types.RunRequest{RunID: "ex05-run", Input: "fanout"}, handler)
	if err != nil {
		panic(err)
	}
	fmt.Printf("final=%q tool_calls=%d\n", res.FinalAnswer, len(res.ToolCalls))
}
