package extension

import (
	"context"
	"sync"
	"testing"
)

func TestGenerationManagerReloadCreatesNewGenerationAndSuppressesStaleEvents(t *testing.T) {
	m := NewGenerationManager()
	d := Descriptor{Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:a", Source: SourceProjectExplicit}
	first, err := m.Reload(d)
	if err != nil || first.Generation != 1 {
		t.Fatalf("first reload=%#v err=%v", first, err)
	}
	d.Digest = "sha256:b"
	second, err := m.Reload(d)
	if err != nil || second.Generation != 2 {
		t.Fatalf("second reload=%#v err=%v", second, err)
	}
	if m.AcceptsEvent(first.Generation) {
		t.Fatal("stale generation must not receive new events")
	}
	if !m.AcceptsEvent(second.Generation) {
		t.Fatal("active generation should receive events")
	}
}

func TestGenerationManagerFailedReloadKeepsPreviousGeneration(t *testing.T) {
	m := NewGenerationManager()
	d := Descriptor{Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:a", Source: SourceProjectExplicit}
	active, err := m.Reload(d)
	if err != nil {
		t.Fatal(err)
	}
	bad := d
	bad.Digest = ""
	if _, err := m.Reload(bad); err == nil {
		t.Fatal("expected failed reload")
	}
	if got := m.ActiveGeneration(); got != active.Generation {
		t.Fatalf("active generation=%d, want %d", got, active.Generation)
	}
}

func TestTurnSnapshotAndSavePointCommitAreIsolated(t *testing.T) {
	m := NewGenerationManager()
	m.SetState("count", 1)
	snapshot := m.Snapshot()
	snapshot["count"] = 9
	if got := m.State("count"); got != 1 {
		t.Fatalf("snapshot mutation leaked into manager: %v", got)
	}
	if err := m.Commit(map[string]any{"count": 2}); err != nil {
		t.Fatal(err)
	}
	if got := m.State("count"); got != 2 {
		t.Fatalf("committed state=%v, want 2", got)
	}
}

func TestExtensionFailureDoesNotRewriteCoreState(t *testing.T) {
	m := NewGenerationManager()
	m.SetState("terminal", "succeeded")
	result := Execute(context.Background(), ExecutionOptions{FailurePolicy: FailurePolicyDegrade}, func(context.Context) (any, error) {
		panic("extension failure")
	})
	if result.Outcome != ExecutionDegraded {
		t.Fatalf("result=%#v", result)
	}
	if got := m.State("terminal"); got != "succeeded" {
		t.Fatalf("extension failure rewrote core terminal state: %v", got)
	}
}

func TestGenerationManagerConcurrentReloadLeavesOnlyLatestGenerationActive(t *testing.T) {
	m := NewGenerationManager()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := Descriptor{Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:" + string(rune('a'+i)), Source: SourceProjectExplicit}
			_, _ = m.Reload(d)
		}(i)
	}
	wg.Wait()
	if got := m.ActiveGeneration(); got != 8 {
		t.Fatalf("active generation=%d, want 8", got)
	}
	if !m.AcceptsEvent(8) {
		t.Fatal("latest generation should remain active after concurrent reloads")
	}
}
