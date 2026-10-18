// This offline example uses a real runner with only synthetic opaque host references.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/context/assembler"
	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

const (
	projectionMaterial = "current host-authorized context"
	userMessage        = "summarize the synthetic source"
)

type offlineModel struct {
	messages []types.Message
}

func (m *offlineModel) Generate(_ context.Context, request types.ModelRequest) (types.ModelResponse, error) {
	m.messages = append([]types.Message(nil), request.Messages...)
	if len(request.ToolResult) == 0 {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "offline-tool-call", Name: "local.echo"}}}, nil
	}
	return types.ModelResponse{FinalAnswer: "offline completion"}, nil
}

func (m *offlineModel) Stream(ctx context.Context, request types.ModelRequest, emit func(types.ModelEvent) error) error {
	response, err := m.Generate(ctx, request)
	if err != nil {
		return err
	}
	for index := range response.ToolCalls {
		if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &response.ToolCalls[index]}); err != nil {
			return err
		}
	}
	if response.FinalAnswer != "" {
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: response.FinalAnswer})
	}
	return nil
}

type offlineTool struct{ calls int }

func (*offlineTool) Name() string               { return "echo" }
func (*offlineTool) Description() string        { return "offline deterministic echo" }
func (*offlineTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *offlineTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	t.calls++
	return types.ToolResult{Content: "offline tool result"}, nil
}

type offlineResolver struct {
	calls    int
	material hostprojection.Material
	err      error
}

func (r *offlineResolver) ResolveHostContext(_ context.Context, _ hostprojection.SourceReference) (hostprojection.Material, error) {
	r.calls++
	if r.err != nil {
		return hostprojection.Material{}, r.err
	}
	return r.material, nil
}

func main() {
	markers, err := execute()
	if err != nil {
		panic(err)
	}
	for _, marker := range markers {
		fmt.Println(marker)
	}
}

func execute() ([]string, error) {
	manager, closeManager, err := offlineRuntimeManager()
	if err != nil {
		return nil, err
	}
	defer closeManager()

	descriptor, err := offlineDescriptor()
	if err != nil {
		return nil, err
	}
	runModel, runResolver, runTool, runEngine, err := offlineEngine(manager, nil)
	if err != nil {
		return nil, err
	}
	streamModel, streamResolver, streamTool, streamEngine, err := offlineEngine(manager, nil)
	if err != nil {
		return nil, err
	}
	runResolver.material = hostprojection.Material{Role: "system", Content: projectionMaterial}
	streamResolver.material = hostprojection.Material{Role: "system", Content: projectionMaterial}

	base := types.RunRequest{
		SessionID:             "offline-host-context-session",
		Input:                 userMessage,
		Messages:              []types.Message{{Role: "user", Content: userMessage}},
		HostContextProjection: &descriptor,
	}
	runRequest := base
	runRequest.RunID = "offline-host-context-run"
	runRequest.HostContextResolver = runResolver
	streamRequest := base
	streamRequest.RunID = "offline-host-context-stream"
	streamRequest.HostContextResolver = streamResolver
	runResult, err := runEngine.Run(context.Background(), runRequest, nil)
	if err != nil || runResult.FinalAnswer != "offline completion" || runResolver.calls != 2 || runTool.calls != 1 {
		return nil, fmt.Errorf("valid Run did not use the admitted projection: result=%#v resolver_calls=%d tool_calls=%d err=%w", runResult, runResolver.calls, runTool.calls, err)
	}
	streamResult, err := streamEngine.Stream(context.Background(), streamRequest, nil)
	if err != nil || streamResult.FinalAnswer != "offline completion" || streamResolver.calls != 2 || streamTool.calls != 1 {
		return nil, fmt.Errorf("valid Stream did not use the admitted projection: result=%#v resolver_calls=%d tool_calls=%d err=%w", streamResult, streamResolver.calls, streamTool.calls, err)
	}
	wantMessages := []types.Message{{Role: "user", Content: userMessage}, {Role: "system", Content: projectionMaterial}}
	if !sameMessages(runModel.messages, wantMessages) || !sameMessages(streamModel.messages, wantMessages) {
		return nil, fmt.Errorf("Run/Stream assembled messages=%#v/%#v, want %#v", runModel.messages, streamModel.messages, wantMessages)
	}
	markers := []string{"host_context.projection_admitted", "host_context.run_stream_equivalent"}

	widened := descriptor
	widened.Capabilities = []hostprojection.Capability{hostprojection.CapabilityRead}
	widened.Parent = &hostprojection.ParentCapabilityEvidence{
		ProjectionID:     "offline-parent",
		CapabilityDigest: "offline-parent-capabilities",
		Capabilities:     []hostprojection.Capability{hostprojection.CapabilityReference},
	}
	widened.Digest, err = hostprojection.CanonicalDigest(widened)
	if err != nil {
		return nil, err
	}
	_, wideningResolver, _, wideningEngine, err := offlineEngine(manager, nil)
	if err != nil {
		return nil, err
	}
	wideningResolver.material = hostprojection.Material{Role: "system", Content: "must not resolve"}
	_, err = wideningEngine.Run(context.Background(), types.RunRequest{
		RunID: "offline-host-context-widening", SessionID: "offline-host-context-session", Input: userMessage,
		Messages: []types.Message{{Role: "user", Content: userMessage}}, HostContextProjection: &widened, HostContextResolver: wideningResolver,
	}, nil)
	if hostProjectionCode(err) != hostprojection.CodeCapabilityExpansion || wideningResolver.calls != 0 {
		return nil, fmt.Errorf("capability widening result=%v resolver_calls=%d, want pre-resolution rejection", err, wideningResolver.calls)
	}
	markers = append(markers, "host_context.capability_expansion")

	contextAssembler := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	record, err := contextAssembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: "offline-host-context-restore", Input: userMessage, HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "offline-host-context-restore", Input: userMessage}, types.ContextAssembleResult{Status: assembler.StatusSuccess})
	if err != nil {
		return nil, fmt.Errorf("build handoff: %w", err)
	}
	_, revokedResolver, _, restoreEngine, err := offlineEngine(manager, handoff.ResolverFunc(func(handoff.Reference) error { return nil }))
	if err != nil {
		return nil, err
	}
	revokedResolver.err = hostprojection.ErrSourceNotFound
	restored, _, err := restoreEngine.RestoreHandoffWithHostProjection(context.Background(), record, revokedResolver, nil)
	if hostProjectionCode(err) != hostprojection.CodeSourceNotFound || restored.Restored || revokedResolver.calls != 1 {
		return nil, fmt.Errorf("revoked restore result=%#v resolver_calls=%d err=%v, want current-authorization rejection", restored, revokedResolver.calls, err)
	}
	markers = append(markers, "host_context.restore_revalidation_failed")

	legacyModel, _, legacyTool, legacyEngine, err := offlineEngine(manager, nil)
	if err != nil {
		return nil, err
	}
	legacyResult, err := legacyEngine.Run(context.Background(), types.RunRequest{
		RunID: "offline-host-context-legacy", SessionID: "offline-host-context-session", Input: userMessage,
		Messages: []types.Message{{Role: "user", Content: userMessage}},
	}, nil)
	if err != nil || legacyResult.FinalAnswer != "offline completion" || legacyTool.calls != 1 || !sameMessages(legacyModel.messages, []types.Message{{Role: "user", Content: userMessage}}) {
		return nil, fmt.Errorf("legacy rollback result=%#v tool_calls=%d messages=%#v err=%w", legacyResult, legacyTool.calls, legacyModel.messages, err)
	}
	return append(markers, "host_context.legacy_rollback"), nil
}

func offlineDescriptor() (hostprojection.Descriptor, error) {
	descriptor := hostprojection.Descriptor{
		SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "offline-host-context-projection", ProjectionVersion: 1,
		CapabilityDigest: "offline-host-context-capabilities", Capabilities: []hostprojection.Capability{hostprojection.CapabilityReference},
		SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://offline/host-context", Digest: "offline-host-context-source"}},
	}
	digest, err := hostprojection.CanonicalDigest(descriptor)
	if err != nil {
		return hostprojection.Descriptor{}, err
	}
	descriptor.Digest = digest
	return descriptor, nil
}

func offlineEngine(manager *runtimeconfig.Manager, resolver handoff.Resolver) (*offlineModel, *offlineResolver, *offlineTool, *runner.Engine, error) {
	base := &offlineModel{}
	model, err := modelcapability.Wrap(base, modelcapability.Config{Provider: "offline", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	tool := &offlineTool{}
	registry := local.NewRegistry()
	if _, err := registry.Register(tool); err != nil {
		return nil, nil, nil, nil, err
	}
	hostResolver := &offlineResolver{}
	options := []runner.Option{runner.WithRuntimeManager(manager), runner.WithLocalRegistry(registry)}
	if resolver != nil {
		options = append(options, runner.WithHandoffResolver(resolver))
	}
	return base, hostResolver, tool, runner.New(model, options...), nil
}

func offlineRuntimeManager() (*runtimeconfig.Manager, func(), error) {
	directory, err := os.MkdirTemp("", "baymax-host-context-assembly-")
	if err != nil {
		return nil, nil, err
	}
	configPath := filepath.Join(directory, "runtime.yaml")
	journalPath := filepath.ToSlash(filepath.Join(directory, "journal.jsonl"))
	config := "context_assembler:\n  enabled: true\n  journal_path: '" + journalPath + "'\n  stage2_routing_and_disclosure:\n    enabled: false\n  pressure_compaction_and_swapback:\n    enabled: false\n"
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		_ = os.RemoveAll(directory)
		return nil, nil, err
	}
	manager, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: configPath, EnvPrefix: "BAYMAX_HOST_CONTEXT_ASSEMBLY_EXAMPLE"})
	if err != nil {
		_ = os.RemoveAll(directory)
		return nil, nil, err
	}
	return manager, func() {
		_ = manager.Close()
		_ = os.RemoveAll(directory)
	}, nil
}

func hostProjectionCode(err error) hostprojection.Code {
	var classified *hostprojection.Error
	if errors.As(err, &classified) {
		return classified.Code
	}
	return ""
}

func sameMessages(got, want []types.Message) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
