package journal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewStorageRejectsDBBackendInCA1(t *testing.T) {
	_, err := NewStorage("db", filepath.Join(t.TempDir(), "journal.jsonl"))
	if !errors.Is(err, ErrBackendNotReady) {
		t.Fatalf("err = %v, want ErrBackendNotReady", err)
	}
}

func TestFileStorageAppendOnlyConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	s := NewFileStorage(path)
	const goroutines = 8
	const writes = 20

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for i := 0; i < writes; i++ {
				err := s.Append(context.Background(), Entry{
					Time:      time.Now(),
					RunID:     "run-1",
					SessionID: "s-1",
					Phase:     "intent",
					Status:    "success",
				})
				if err != nil {
					t.Errorf("append failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	want := goroutines * writes
	if len(lines) != want {
		t.Fatalf("line count = %d, want %d", len(lines), want)
	}
}
