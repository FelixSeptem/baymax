package sourceadapter_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/host/sourceadapter"
)

func TestFileStoreFreshProcess(t *testing.T) {
	if p := os.Getenv("BAYMAX_SOURCE_STORE_CHILD"); p != "" {
		s, err := sourceadapter.OpenFileStore(p, metadata())
		if err != nil {
			t.Fatal(err)
		}
		state := load(t, s)
		if state.Confirmed == nil || state.Confirmed.Token != "1" || state.Pending == nil || state.Pending.ExternalID != "two" {
			t.Fatalf("reopen=%+v", state)
		}
		return
	}
	p := filepath.Join(t.TempDir(), "state.json")
	s, err := sourceadapter.OpenFileStore(p, metadata())
	if err != nil {
		t.Fatal(err)
	}
	a := newAdapter(t, s, func(_ context.Context, f sourceadapter.Fact) (sourceadapter.Disposition, error) {
		if f.ExternalID == "two" {
			return sourceadapter.Retryable, nil
		}
		return sourceadapter.Sent, nil
	})
	err = a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "private payload"), fact("two", "2", "other payload")}})
	if !errors.Is(err, sourceadapter.ErrRetryable) {
		t.Fatal(err)
	}
	a.Dispose()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "private payload") || strings.Contains(string(b), "other payload") {
		t.Fatal("payload persisted")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFileStoreFreshProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "BAYMAX_SOURCE_STORE_CHILD="+p)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v %s", err, b)
	}
}

func TestFileStoreRejectsCorruptionWithoutOverwrite(t *testing.T) {
	for _, text := range []string{"{", "{}", "null", "{} {}", "{\"unknown\":true}", "{\"schema\":\"a\",\"schema\":\"b\"}", strings.Repeat("x", sourceadapter.MaxStateBytes+1)} {
		p := filepath.Join(t.TempDir(), "state.json")
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := sourceadapter.OpenFileStore(p, metadata()); err == nil {
			t.Fatal("corrupt state accepted")
		}
		got, err := os.ReadFile(p)
		if err != nil || string(got) != text {
			t.Fatal("corrupt state overwritten")
		}
	}
	p := filepath.Join(t.TempDir(), "state.json")
	s, err := sourceadapter.OpenFileStore(p, metadata())
	if err != nil {
		t.Fatal(err)
	}
	m := metadata()
	m.Source.ID = "foreign"
	if _, err := sourceadapter.OpenFileStore(p, m); !errors.Is(err, sourceadapter.ErrForeign) {
		t.Fatalf("foreign=%v", err)
	}
	if load(t, s).Revision != 0 {
		t.Fatal("foreign open mutated store")
	}
}

func TestFileStoreFailedWritePreservesState(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "state.json")
	s, err := sourceadapter.OpenFileStore(p, metadata())
	if err != nil {
		t.Fatal(err)
	}
	// Replace the destination with a directory to force rename failure on both
	// Windows and Unix. This tests the public atomic-write failure boundary.
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p, 0700); err != nil {
		t.Fatal(err)
	}
	a := newAdapter(t, s, func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
		t.Fatal("effect before pending persistence")
		return sourceadapter.Sent, nil
	})
	if err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "body")}}); err == nil {
		t.Fatal("failed write accepted")
	}
	if load(t, s).Revision != 0 {
		t.Fatal("failed write mutated memory")
	}
}
