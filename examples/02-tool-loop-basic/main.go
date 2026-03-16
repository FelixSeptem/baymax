package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

type toolLoopModel struct{ calls int }

func (m *toolLoopModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	m.calls++
	if m.calls == 1 {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo", Args: map[string]any{"q": "tool-loop"}}}}, nil
	}
	return types.ModelResponse{FinalAnswer: "tool result received"}, nil
}

func (m *toolLoopModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return nil
}

type echoTool struct{}

func (t *echoTool) Name() string        { return "echo" }
func (t *echoTool) Description() string { return "echo text" }
func (t *echoTool) JSONSchema() map[string]any {
	return map[string]any{"type": "object", "required": []any{"q"}, "properties": map[string]any{"q": map[string]any{"type": "string"}}}
}
func (t *echoTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	return types.ToolResult{Content: fmt.Sprintf("echo=%v", args["q"])}, nil
}

type timelinePrinter struct{}

func (p timelinePrinter) OnEvent(ctx context.Context, ev types.Event) {
	_ = ctx
	if ev.Type != types.EventTypeActionTimeline {
		return
	}
	reason, _ := ev.Payload["reason"].(string)
	if strings.TrimSpace(reason) != "gate.rule_match" {
		return
	}
	fmt.Printf("timeline reason=%s iteration=%d\n", reason, ev.Iteration)
}

func main() {
	cfgPath := filepath.Join(os.TempDir(), fmt.Sprintf("baymax-example-02-%d.yaml", time.Now().UnixNano()))
	cfg := `
action_gate:
  enabled: true
  policy: require_confirm
  parameter_rules:
    - id: allow-tool-loop-q
      tool_names: [echo]
      action: allow
      condition:
        path: q
        operator: contains
        expected: tool-loop
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		panic(err)
	}
	defer func() { _ = os.Remove(cfgPath) }()
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	_, _ = reg.Register(&echoTool{})
	eng := runner.New(&toolLoopModel{}, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	res, err := eng.Run(context.Background(), types.RunRequest{Input: "run tool"}, timelinePrinter{})
	if err != nil {
		panic(err)
	}
	fmt.Println(res.FinalAnswer)
}
