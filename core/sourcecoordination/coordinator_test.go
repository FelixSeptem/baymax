package sourcecoordination

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestMemoryCoordinatorFencesStaleWriterAndPreservesLatestRevision(t *testing.T) {
	now := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	c, err := NewMemory(Options{OwnerNamespace: "runner", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := c.Acquire(ctx, "writer-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := c.CompareAndSwap(ctx, first, 0, []byte(`{"state":"one"}`))
	if err != nil {
		t.Fatal(err)
	}
	if committed.Revision != 1 {
		t.Fatalf("revision = %d", committed.Revision)
	}
	now = now.Add(time.Minute)
	second, err := c.Acquire(ctx, "writer-b", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.CompareAndSwap(ctx, first, 1, []byte(`{"state":"stale"}`)); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("stale writer error = %v", err)
	}
	if _, err := c.CompareAndSwap(ctx, second, 0, []byte(`{"state":"conflict"}`)); !errors.Is(err, types.ErrDeploymentConflict) {
		t.Fatalf("revision conflict = %v", err)
	}
	current, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current.Revision != 1 || string(current.Payload) != `{"state":"one"}` {
		t.Fatalf("current = %+v", current)
	}
}

func TestFileCoordinatorReopensAndFencesAcrossOwners(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	first, err := OpenFile(dir, Options{OwnerNamespace: "mailbox"})
	if err != nil {
		t.Fatal(err)
	}
	lease, err := first.Acquire(ctx, "writer-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = first.CompareAndSwap(ctx, lease, 0, []byte(`{"delivery":1}`)); err != nil {
		t.Fatal(err)
	}
	second, err := OpenFile(dir, Options{OwnerNamespace: "mailbox"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	current, err := second.Read(ctx)
	if err != nil || current.Revision != 1 || string(current.Payload) != `{"delivery":1}` {
		t.Fatalf("reopen = %+v %v", current, err)
	}
	newLease, err := second.Acquire(ctx, "writer-b", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = second.CompareAndSwap(ctx, lease, 1, []byte(`{}`)); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("prior lease accepted = %v", err)
	}
	if _, err = second.CompareAndSwap(ctx, newLease, 1, []byte(`{"delivery":2}`)); err != nil {
		t.Fatal(err)
	}
}

func TestFileCoordinatorRejectsCorruptAndUnavailableState(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, stateFileName), []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFile(dir, Options{OwnerNamespace: "runner"}); !errors.Is(err, types.ErrOwnerUnavailable) {
		t.Fatalf("corrupt state = %v", err)
	}
}

func TestMemoryCoordinatorRejectsForeignLeaseAndExpiredLease(t *testing.T) {
	now := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	c, err := NewMemory(Options{OwnerNamespace: "workflow", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	lease, err := c.Acquire(ctx, "writer", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	foreign := lease
	foreign.OwnerNamespace = "other"
	if _, err := c.CompareAndSwap(ctx, foreign, 0, []byte(`{}`)); !errors.Is(err, types.ErrForeignWriter) {
		t.Fatalf("foreign writer = %v", err)
	}
	now = now.Add(2 * time.Second)
	if _, err := c.CompareAndSwap(ctx, lease, 0, []byte(`{}`)); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("expired writer = %v", err)
	}
}

func TestFileCoordinatorFencesStaleWriterAcrossProcesses(t *testing.T) {
	if os.Getenv("BAYMAX_SOURCE_COORDINATION_HELPER") == "1" {
		coordinator, err := OpenFile(os.Getenv("BAYMAX_SOURCE_COORDINATION_DIR"), Options{OwnerNamespace: "process-owner"})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = coordinator.Close() }()
		lease, err := coordinator.Acquire(context.Background(), "child", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = coordinator.CompareAndSwap(context.Background(), lease, 1, []byte(`{"writer":"child"}`)); err != nil {
			t.Fatal(err)
		}
		return
	}

	dir := t.TempDir()
	parent, err := OpenFile(dir, Options{OwnerNamespace: "process-owner"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = parent.Close() }()
	lease, err := parent.Acquire(context.Background(), "parent", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parent.CompareAndSwap(context.Background(), lease, 0, []byte(`{"writer":"parent"}`)); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFileCoordinatorFencesStaleWriterAcrossProcesses$")
	cmd.Env = append(os.Environ(), "BAYMAX_SOURCE_COORDINATION_HELPER=1", "BAYMAX_SOURCE_COORDINATION_DIR="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("helper: %v %s", err, out)
	}
	if _, err = parent.CompareAndSwap(context.Background(), lease, 2, []byte(`{"writer":"stale"}`)); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("parent stale write = %v", err)
	}
	current, err := parent.Read(context.Background())
	if err != nil || current.Revision != 2 || string(current.Payload) != `{"writer":"child"}` {
		t.Fatalf("current=%+v err=%v", current, err)
	}
}
