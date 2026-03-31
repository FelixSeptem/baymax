package memory

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFilesystemEngineRoundTripRecovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory-store")
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: root,
		Compaction: FilesystemCompactionConfig{
			Enabled:     true,
			MinOps:      2,
			MaxWALBytes: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	_, err = engine.Upsert(UpsertRequest{
		Namespace: "s1",
		Records: []Record{
			{ID: "r1", Content: "hello memory", SessionID: "session-1"},
			{ID: "r2", Content: "world memory", SessionID: "session-1"},
		},
	})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	_, err = engine.Delete(DeleteRequest{Namespace: "s1", IDs: []string{"r2"}})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	recovered, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: root,
		Compaction: FilesystemCompactionConfig{
			Enabled:     true,
			MinOps:      2,
			MaxWALBytes: 1,
		},
	})
	if err != nil {
		t.Fatalf("recover NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = recovered.Close() }()

	resp, err := recovered.Query(QueryRequest{
		Namespace: "s1",
		SessionID: "session-1",
		Query:     "hello",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("query total = %d, want 1", resp.Total)
	}
	if len(resp.Records) != 1 || resp.Records[0].ID != "r1" {
		t.Fatalf("records mismatch: %#v", resp.Records)
	}
}

func TestFilesystemEngineReplayIdempotentBySeq(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory-store")
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: root,
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	_, err = engine.Upsert(UpsertRequest{
		Namespace: "s1",
		Records: []Record{
			{ID: "r1", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	walPath := filepath.Join(root, walFileName)
	raw, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("read wal failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatalf("wal lines = %d, want 1", len(lines))
	}
	duplicate := strings.TrimSpace(lines[0]) + "\n" + strings.TrimSpace(lines[0]) + "\n"
	if err := os.WriteFile(walPath, []byte(duplicate), 0o600); err != nil {
		t.Fatalf("write duplicated wal failed: %v", err)
	}

	recovered, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: root,
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("recover NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = recovered.Close() }()
	resp, err := recovered.Query(QueryRequest{Namespace: "s1"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp.Total != 1 || len(resp.Records) != 1 || resp.Records[0].ID != "r1" {
		t.Fatalf("idempotent replay mismatch: %#v", resp.Records)
	}
}

func TestFilesystemEngineRecoverFromSnapshotNext(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory-store")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	state := `{"last_seq":3,"records":[{"id":"r1","namespace":"ns","content":"from-next"}]}`
	if err := os.WriteFile(filepath.Join(root, snapshotNextFileName), []byte(state), 0o600); err != nil {
		t.Fatalf("write snapshot.next failed: %v", err)
	}
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: root,
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = engine.Close() }()
	resp, err := engine.Query(QueryRequest{Namespace: "ns"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp.Total != 1 || len(resp.Records) != 1 || resp.Records[0].Content != "from-next" {
		t.Fatalf("recover from snapshot.next mismatch: %#v", resp.Records)
	}
}

func TestFilesystemEngineConcurrentQueryUpsert(t *testing.T) {
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: filepath.Join(t.TempDir(), "memory-store"),
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = engine.Close() }()

	const workers = 8
	const loops = 50
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				id := "r-" + string(rune('a'+worker))
				_, _ = engine.Upsert(UpsertRequest{
					Namespace: "ns",
					Records: []Record{
						{ID: id, Content: "content"},
					},
				})
				_, _ = engine.Query(QueryRequest{Namespace: "ns", MaxItems: 8})
			}
		}(i)
	}
	wg.Wait()
	resp, err := engine.Query(QueryRequest{Namespace: "ns"})
	if err != nil {
		t.Fatalf("final Query failed: %v", err)
	}
	if resp.Total == 0 {
		t.Fatalf("expected non-empty records after concurrent upsert/query")
	}
}
