package planbridge

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/sourcecoordination"
	"github.com/FelixSeptem/baymax/core/types"
)

func TestNativeHistoryManifestMarksFileJournalDurable(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "durable-run"}
	j, err := runner.OpenNativeEventHistory(filepath.Join(t.TempDir(), "runner.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step"}); err != nil {
		t.Fatal(err)
	}
	manifest, err := CapabilityManifestWithHistory(context.Background(), j, source)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reader.Capability(ExecutionFormDirect)
	if err != nil || entry.DurableHistory != CapabilitySupported || entry.AfterCursorReplay != CapabilitySupported {
		t.Fatalf("file journal manifest: %+v err=%v", entry, err)
	}
}

func TestNativeHistoryManifestProjectsCoordinatorScopeWithoutCrossMachineClaim(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "coordinated-run"}
	coordinator, err := sourcecoordination.OpenFile(filepath.Join(t.TempDir(), "coord"), sourcecoordination.Options{OwnerNamespace: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = coordinator.Close() }()
	journal, err := history.New(source, history.WithCoordinator(coordinator, "writer", 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = journal.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step"}); err != nil {
		t.Fatal(err)
	}
	manifest, err := CapabilityManifestWithHistory(context.Background(), journal, source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := entry.Capability(ExecutionFormDirect)
	if err != nil || direct.Deployment.Profile != types.DeploymentProfileMultiInstance || direct.Deployment.WriterScope != types.WriterScopeMultiWriter {
		t.Fatalf("coordinated manifest = %+v err=%v", direct.Deployment, err)
	}
	if direct.Deployment.Supports(types.DeploymentRequest{Profile: types.DeploymentProfileCrossMachine}) {
		t.Fatal("manifest advertised cross-machine history")
	}
}

func TestNativeHistoryManifestMapsRunnerSourceToDirectForm(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run"}
	journal, err := runner.NewNativeEventHistory(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step"}); err != nil {
		t.Fatal(err)
	}
	manifest, err := CapabilityManifestWithHistory(context.Background(), journal, source)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reader.Capability(ExecutionFormDirect)
	if err != nil || entry.DurableHistory != CapabilityUnsupported || entry.AfterCursorReplay != CapabilitySupported || entry.SourceHistory == nil {
		t.Fatalf("runner history manifest: %+v %v", entry, err)
	}
	subagent, err := reader.Capability(ExecutionFormSubagent)
	if err != nil || subagent.DurableHistory != CapabilityUnsupported {
		t.Fatalf("source capability leaked across forms: %+v %v", subagent, err)
	}
}
