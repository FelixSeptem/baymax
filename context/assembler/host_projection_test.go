package assembler

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestAssemblerRejectsInvalidHostProjectionBeforeResolverOrJournal(t *testing.T) {
	journalPath := filepath.Join(t.TempDir(), "journal.jsonl")
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = journalPath
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	descriptor := validHostProjection(t)
	descriptor.SchemaVersion = "host_context_projection.v2"
	resolver := &hostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "must not resolve"}}

	_, _, err := assembler.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:                 "run-host-projection-invalid",
		SessionID:             "session-host-projection-invalid",
		PrefixVersion:         semanticPrefixVersion,
		HostContextProjection: &descriptor,
		HostContextResolver:   resolver,
	}, types.ModelRequest{RunID: "run-host-projection-invalid"})
	if err == nil {
		t.Fatal("Assemble() error = nil, want invalid projection rejection")
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls = %d, want 0 before validation", resolver.calls)
	}
	if _, statErr := os.Stat(journalPath); !os.IsNotExist(statErr) {
		t.Fatalf("journal stat error = %v, want no journal side effect", statErr)
	}
}

func TestAssemblerInjectsOnlyCurrentAuthorizedHostProjectionMaterial(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	descriptor := validHostProjection(t)
	resolver := &hostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current authorized context"}}
	baseMessages := []types.Message{{Role: "user", Content: "original request"}}

	assembled, _, err := assembler.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:                 "run-host-projection-valid",
		SessionID:             "session-host-projection-valid",
		PrefixVersion:         semanticPrefixVersion,
		Messages:              baseMessages,
		HostContextProjection: &descriptor,
		HostContextResolver:   resolver,
	}, types.ModelRequest{
		RunID:    "run-host-projection-valid",
		Messages: baseMessages,
	})
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", resolver.calls)
	}
	if len(assembled.Messages) != 2 || assembled.Messages[1].Content != "current authorized context" {
		t.Fatalf("assembled messages = %#v, want original plus current authorized material", assembled.Messages)
	}
}

func TestAssemblerBuildHandoffRetainsOnlyHostProjectionCorrelation(t *testing.T) {
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	descriptor := validHostProjection(t)
	h, err := assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID:                 "run-host-projection-handoff",
		Input:                 "handoff objective",
		HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "run-host-projection-handoff", Input: "handoff objective"}, types.ContextAssembleResult{Status: StatusSuccess})
	if err != nil {
		t.Fatalf("BuildHandoff() error = %v", err)
	}
	if h.HostContextProjection == nil {
		t.Fatal("handoff correlation = nil, want bounded projection evidence")
	}
	correlation := h.HostContextProjection
	if correlation.SchemaVersion != hostprojection.SchemaVersionV1 || correlation.ProjectionID != descriptor.ProjectionID || correlation.Digest != descriptor.Digest {
		t.Fatalf("handoff correlation = %#v, want descriptor identity/digest", correlation)
	}
	if len(correlation.SourceReferences) != 1 || correlation.SourceReferences[0].Reference != "opaque://source/assembler" {
		t.Fatalf("handoff source evidence = %#v, want opaque reference", correlation.SourceReferences)
	}
}

func TestAssemblerBuildHandoffCorrelatesProjectionAtEveryLegalCut(t *testing.T) {
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	descriptor := validHostProjection(t)
	tests := []struct {
		name string
		req  types.ContextAssembleRequest
		cut  handoff.Cut
	}{
		{name: "finalized", req: types.ContextAssembleRequest{RunID: "run-finalized", Input: "objective", HostContextProjection: &descriptor}, cut: handoff.CutFinalizedEvent},
		{name: "checkpoint", req: types.ContextAssembleRequest{RunID: "run-checkpoint", Input: "objective", FinalizedCheckpointID: "cp-1", HostContextProjection: &descriptor}, cut: handoff.CutCheckpoint},
		{name: "flushed stream", req: types.ContextAssembleRequest{RunID: "run-stream", Input: "objective", StreamFlushed: true, HostContextProjection: &descriptor}, cut: handoff.CutFlushedStream},
		{name: "tool finalized", req: types.ContextAssembleRequest{RunID: "run-tool", Input: "objective", HostContextProjection: &descriptor, ToolResult: []types.ToolCallOutcome{{Lifecycle: &types.ToolLifecycleProjection{Finalized: true}}}}, cut: handoff.CutToolFinalized},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record, err := assembler.BuildHandoff(tc.req, types.ModelRequest{RunID: tc.req.RunID, Input: tc.req.Input}, types.ContextAssembleResult{Status: StatusSuccess})
			if err != nil {
				t.Fatalf("BuildHandoff() error = %v", err)
			}
			if record.Cut != tc.cut || record.HostContextProjection == nil || record.HostContextProjection.ProjectionID != descriptor.ProjectionID {
				t.Fatalf("handoff = %#v, want cut=%q bounded correlation", record, tc.cut)
			}
		})
	}
	legacy, err := assembler.BuildHandoff(types.ContextAssembleRequest{RunID: "run-legacy-handoff", Input: "objective"}, types.ModelRequest{RunID: "run-legacy-handoff", Input: "objective"}, types.ContextAssembleResult{Status: StatusSuccess})
	if err != nil || legacy.HostContextProjection != nil {
		t.Fatalf("legacy handoff=%#v error=%v, want compatible nullable correlation", legacy, err)
	}
}

func TestProjectedHandoffDoesNotPersistResolvedBody(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig()
	cfg.ContextAssembler.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.Runtime.Context.JIT.Compaction.Handoff.Enabled = true
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg.ContextAssembler }, WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig { return cfg.Runtime.Context }))
	d := validHostProjection(t)
	r := &hostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "private-resolved-body-marker"}}
	req := types.ContextAssembleRequest{RunID: "privacy-run", Input: "objective", HostContextProjection: &d, HostContextResolver: r}
	_, _, record, err := a.AssembleWithHandoff(context.Background(), req, types.ModelRequest{RunID: req.RunID, Input: req.Input})
	if err != nil || record == nil {
		t.Fatalf("AssembleWithHandoff() record=%#v error=%v", record, err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private-resolved-body-marker") {
		t.Fatal("resolved host body leaked into handoff")
	}
}

func TestAssemblerRestoreProjectedHandoffRequiresCurrentHostResolver(t *testing.T) {
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	descriptor := validHostProjection(t)
	h, err := assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: "run-host-projection-restore", Input: "restore objective", HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "run-host-projection-restore", Input: "restore objective"}, types.ContextAssembleResult{Status: StatusSuccess})
	if err != nil {
		t.Fatalf("BuildHandoff() error = %v", err)
	}
	if _, err := assembler.RestoreHandoff(h, handoff.ResolverFunc(func(handoff.Reference) error { return nil })); err == nil {
		t.Fatal("RestoreHandoff() accepted projected handoff without current host resolver")
	}

	resolver := &hostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current restore context"}}
	result, _, err := assembler.RestoreHandoffWithHostProjection(context.Background(), h, handoff.ResolverFunc(func(handoff.Reference) error { return nil }), resolver, nil)
	if err != nil {
		t.Fatalf("RestoreHandoffWithHostProjection() error = %v", err)
	}
	if !result.Restored || resolver.calls != 1 {
		t.Fatalf("restore = %#v resolver calls=%d, want restored with reauthorization", result, resolver.calls)
	}
}

func TestAssemblerRestoreProjectedHandoffFailsClosedAfterAuthorizationRevocation(t *testing.T) {
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	descriptor := validHostProjection(t)
	h, err := assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: "run-host-projection-revoked", Input: "restore objective", HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "run-host-projection-revoked", Input: "restore objective"}, types.ContextAssembleResult{Status: StatusSuccess})
	if err != nil {
		t.Fatalf("BuildHandoff() error = %v", err)
	}
	resolver := &hostProjectionResolver{err: hostprojection.ErrSourceNotFound}
	result, _, err := assembler.RestoreHandoffWithHostProjection(context.Background(), h, handoff.ResolverFunc(func(handoff.Reference) error { return nil }), resolver, nil)
	if err == nil {
		t.Fatal("RestoreHandoffWithHostProjection() error = nil, want authorization rejection")
	}
	if result.Restored || resolver.calls != 1 || !errors.As(err, new(*hostprojection.Error)) {
		t.Fatalf("restore=%#v resolver calls=%d error=%v, want fail-closed classified rejection", result, resolver.calls, err)
	}
}

func TestAssemblerRestoreProjectedHandoffRejectsCorrelationDriftBeforeHostResolution(t *testing.T) {
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	descriptor := validHostProjection(t)
	h, err := assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: "run-host-projection-drift", Input: "restore objective", HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "run-host-projection-drift", Input: "restore objective"}, types.ContextAssembleResult{Status: StatusSuccess})
	if err != nil {
		t.Fatalf("BuildHandoff() error = %v", err)
	}
	h.HostContextProjection.Digest = "sha256:drift"
	resolver := &hostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "must not resolve"}}
	_, _, err = assembler.RestoreHandoffWithHostProjection(context.Background(), h, handoff.ResolverFunc(func(handoff.Reference) error { return nil }), resolver, nil)
	if err == nil {
		t.Fatal("RestoreHandoffWithHostProjection() accepted correlation drift")
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls=%d, want 0 before drift rejection", resolver.calls)
	}
	if code := hostProjectionCode(t, err); code != hostprojection.CodeDigestMismatch {
		t.Fatalf("error code=%q, want %q", code, hostprojection.CodeDigestMismatch)
	}
}

func TestAssemblerRestoreProjectedHandoffRejectsCapabilityExpansionBeforeHostResolution(t *testing.T) {
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	descriptor := validHostProjection(t)
	h, err := assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: "run-host-projection-expansion", Input: "restore objective", HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "run-host-projection-expansion", Input: "restore objective"}, types.ContextAssembleResult{Status: StatusSuccess})
	if err != nil {
		t.Fatalf("BuildHandoff() error = %v", err)
	}
	correlation := h.HostContextProjection
	correlation.Capabilities = []string{string(hostprojection.CapabilityRead)}
	correlation.Parent = &handoff.HostContextParentCorrelation{
		ProjectionID:     "parent-projection",
		CapabilityDigest: "parent-capability-digest",
		Capabilities:     []string{string(hostprojection.CapabilityReference)},
	}
	drifted := hostprojection.Descriptor{
		SchemaVersion: correlation.SchemaVersion, ProjectionID: correlation.ProjectionID, ProjectionVersion: correlation.ProjectionVersion,
		CapabilityDigest: correlation.CapabilityDigest, Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead},
		SourceReferences: []hostprojection.SourceReference{{Reference: correlation.SourceReferences[0].Reference, Digest: correlation.SourceReferences[0].Digest}},
		Parent:           &hostprojection.ParentCapabilityEvidence{ProjectionID: correlation.Parent.ProjectionID, CapabilityDigest: correlation.Parent.CapabilityDigest, Capabilities: []hostprojection.Capability{hostprojection.CapabilityReference}},
	}
	digest, err := hostprojection.CanonicalDigest(drifted)
	if err != nil {
		t.Fatalf("CanonicalDigest() error = %v", err)
	}
	correlation.Digest = digest
	resolver := &hostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "must not resolve"}}
	_, _, err = assembler.RestoreHandoffWithHostProjection(context.Background(), h, handoff.ResolverFunc(func(handoff.Reference) error { return nil }), resolver, nil)
	if err == nil {
		t.Fatal("RestoreHandoffWithHostProjection() accepted capability expansion")
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls=%d, want 0 before capability rejection", resolver.calls)
	}
	if code := hostProjectionCode(t, err); code != hostprojection.CodeCapabilityExpansion {
		t.Fatalf("error code=%q, want %q", code, hostprojection.CodeCapabilityExpansion)
	}
}

func hostProjectionCode(t *testing.T, err error) hostprojection.Code {
	t.Helper()
	classified, ok := err.(*hostprojection.Error)
	if !ok {
		t.Fatalf("error=%#v, want host projection classification", err)
	}
	return classified.Code
}

func TestAssemblerRestoreProjectedHandoffIsIdempotentAfterReauthorization(t *testing.T) {
	assembler := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	descriptor := validHostProjection(t)
	h, err := assembler.BuildHandoff(types.ContextAssembleRequest{
		RunID: "run-host-projection-idempotent", Input: "restore objective", HostContextProjection: &descriptor,
	}, types.ModelRequest{RunID: "run-host-projection-idempotent", Input: "restore objective"}, types.ContextAssembleResult{Status: StatusSuccess})
	if err != nil {
		t.Fatalf("BuildHandoff() error = %v", err)
	}
	resolver := &hostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current restore context"}}
	first, firstResolution, err := assembler.RestoreHandoffWithHostProjection(context.Background(), h, handoff.ResolverFunc(func(handoff.Reference) error { return nil }), resolver, nil)
	if err != nil {
		t.Fatalf("first restore error = %v", err)
	}
	second, secondResolution, err := assembler.RestoreHandoffWithHostProjection(context.Background(), h, handoff.ResolverFunc(func(handoff.Reference) error { return nil }), resolver, nil)
	if err != nil {
		t.Fatalf("second restore error = %v", err)
	}
	if first != second || len(firstResolution.Materials()) != 1 || len(secondResolution.Materials()) != 1 || resolver.calls != 2 {
		t.Fatalf("first=%#v second=%#v resolutions=%#v/%#v calls=%d, want stable result with fresh reauthorization", first, second, firstResolution.Materials(), secondResolution.Materials(), resolver.calls)
	}
}

func validHostProjection(t *testing.T) hostprojection.Descriptor {
	t.Helper()
	descriptor := hostprojection.Descriptor{
		SchemaVersion:     hostprojection.SchemaVersionV1,
		ProjectionID:      "projection-assembler",
		ProjectionVersion: 1,
		CapabilityDigest:  "capability-assembler",
		Capabilities:      []hostprojection.Capability{hostprojection.CapabilityReference},
		SourceReferences:  []hostprojection.SourceReference{{Reference: "opaque://source/assembler", Digest: "source-digest-assembler"}},
	}
	digest, err := hostprojection.CanonicalDigest(descriptor)
	if err != nil {
		t.Fatalf("CanonicalDigest() error = %v", err)
	}
	descriptor.Digest = digest
	return descriptor
}

type hostProjectionResolver struct {
	calls    int
	material hostprojection.Material
	err      error
}

func (r *hostProjectionResolver) ResolveHostContext(_ context.Context, _ hostprojection.SourceReference) (hostprojection.Material, error) {
	r.calls++
	if r.err != nil {
		return hostprojection.Material{}, r.err
	}
	return r.material, nil
}
