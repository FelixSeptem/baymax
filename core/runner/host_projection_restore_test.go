package runner

import (
	"context"
	"reflect"
	"testing"

	"github.com/FelixSeptem/baymax/context/assembler"
	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestEngineRestoreProjectedHandoffRequiresExplicitCurrentResolver(t *testing.T) {
	descriptor := runnerHostProjection(t)
	contextAssembler := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	record, err := contextAssembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: "run-engine-projected-restore", Input: "restore", HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "run-engine-projected-restore", Input: "restore"}, types.ContextAssembleResult{Status: assembler.StatusSuccess})
	if err != nil {
		t.Fatalf("BuildHandoff() error = %v", err)
	}
	engine := New(nil, WithHandoffResolver(handoff.ResolverFunc(func(handoff.Reference) error { return nil })))
	if _, err := engine.RestoreHandoff(context.Background(), record, nil); err == nil {
		t.Fatal("RestoreHandoff() accepted projected record without explicit current host resolver")
	}
	resolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current restore context"}}
	result, resolution, err := engine.RestoreHandoffWithHostProjection(context.Background(), record, resolver, nil)
	if err != nil {
		t.Fatalf("RestoreHandoffWithHostProjection() error = %v", err)
	}
	if !result.Restored || resolver.calls != 1 || len(resolution.Materials()) != 1 {
		t.Fatalf("result=%#v calls=%d resolution=%#v, want current reauthorized restore", result, resolver.calls, resolution.Materials())
	}
}

func TestEquivalentRunStreamProjectedHandoffsRestoreToSameNormalizedOutcome(t *testing.T) {
	descriptor := runnerHostProjection(t)
	newEngine := func(t *testing.T, stream bool) *Engine {
		t.Helper()
		resolver := handoff.ResolverFunc(func(handoff.Reference) error { return nil })
		if stream {
			return New(&fakeModel{stream: func(_ context.Context, _ types.ModelRequest, emit func(types.ModelEvent) error) error {
				return emit(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "done"})
			}}, WithRuntimeManager(newHostProjectionHandoffRuntimeManager(t, "BAYMAX_HOST_PROJECTION_RESTORE_STREAM_TEST")), WithHandoffResolver(resolver))
		}
		return New(&fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "done"}, nil
		}}, WithRuntimeManager(newHostProjectionHandoffRuntimeManager(t, "BAYMAX_HOST_PROJECTION_RESTORE_RUN_TEST")), WithHandoffResolver(resolver))
	}
	runEngine, streamEngine := newEngine(t, false), newEngine(t, true)
	base := types.RunRequest{
		RunID: "run-stream-projected-handoff", SessionID: "projected-handoff-session", Input: "restore objective",
		Messages: []types.Message{{Role: "user", Content: "original request"}}, HostContextProjection: &descriptor,
	}
	runRequest := base
	runRequest.HostContextResolver = &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current run context"}}
	streamRequest := base
	streamRequest.HostContextResolver = &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current stream context"}}
	runResult, runErr := runEngine.Run(context.Background(), runRequest, nil)
	streamResult, streamErr := streamEngine.Stream(context.Background(), streamRequest, nil)
	if runErr != nil || streamErr != nil || runResult.FinalAnswer != "done" || streamResult.FinalAnswer != "done" {
		t.Fatalf("Run/Stream results=%#v/%#v errors=%v/%v", runResult, streamResult, runErr, streamErr)
	}

	runRecord, err := runEngine.assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: base.RunID, SessionID: base.SessionID, Input: base.Input, Messages: base.Messages, HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: base.RunID, Input: base.Input, Messages: base.Messages}, types.ContextAssembleResult{Status: assembler.StatusSuccess})
	if err != nil {
		t.Fatalf("build Run handoff: %v", err)
	}
	streamRecord, err := streamEngine.assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: base.RunID, SessionID: base.SessionID, Input: base.Input, Messages: base.Messages, HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: base.RunID, Input: base.Input, Messages: base.Messages}, types.ContextAssembleResult{Status: assembler.StatusSuccess})
	if err != nil {
		t.Fatalf("build Stream handoff: %v", err)
	}
	if !reflect.DeepEqual(runRecord, streamRecord) {
		t.Fatalf("Run/Stream handoffs differ: %#v/%#v", runRecord, streamRecord)
	}
	runRestoreResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "fresh restore context"}}
	streamRestoreResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "fresh restore context"}}
	runRestore, runResolution, runErr := runEngine.RestoreHandoffWithHostProjection(context.Background(), runRecord, runRestoreResolver, nil)
	streamRestore, streamResolution, streamErr := streamEngine.RestoreHandoffWithHostProjection(context.Background(), streamRecord, streamRestoreResolver, nil)
	if runErr != nil || streamErr != nil || !runRestore.Restored || !streamRestore.Restored || runRestoreResolver.calls != 1 || streamRestoreResolver.calls != 1 {
		t.Fatalf("Run/Stream restore=%#v/%#v resolution=%#v/%#v errors=%v/%v resolver_calls=%d/%d", runRestore, streamRestore, runResolution.Materials(), streamResolution.Materials(), runErr, streamErr, runRestoreResolver.calls, streamRestoreResolver.calls)
	}
	if !reflect.DeepEqual(runRestore, streamRestore) || !reflect.DeepEqual(runResolution.Materials(), streamResolution.Materials()) {
		t.Fatalf("Run/Stream normalized restore=%#v/%#v resolution=%#v/%#v", runRestore, streamRestore, runResolution.Materials(), streamResolution.Materials())
	}
	denied := &runnerHostProjectionResolver{err: hostprojection.ErrSourceNotFound}
	if _, _, err := runEngine.RestoreHandoffWithHostProjection(context.Background(), runRecord, denied, nil); hostProjectionCode(err) != hostprojection.CodeSourceNotFound || denied.calls != 1 {
		t.Fatalf("matching historical digest bypassed current authorization: error=%v code=%q resolver_calls=%d", err, hostProjectionCode(err), denied.calls)
	}
}
