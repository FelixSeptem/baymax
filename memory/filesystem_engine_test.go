package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestFilesystemEngineCompactWritesSnapshotAndIndexJSON(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory-store")
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: root,
		Compaction: FilesystemCompactionConfig{
			Enabled:     true,
			MinOps:      1,
			MaxWALBytes: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = engine.Close() }()

	if _, err := engine.Upsert(UpsertRequest{
		Namespace: "ns",
		Records: []Record{
			{ID: "r1", Content: "snapshot-compact"},
		},
	}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	rawSnapshot, err := os.ReadFile(filepath.Join(root, snapshotFileName))
	if err != nil {
		t.Fatalf("read snapshot failed: %v", err)
	}
	state := snapshotState{}
	if err := json.Unmarshal(rawSnapshot, &state); err != nil {
		t.Fatalf("decode snapshot failed: %v", err)
	}
	if state.LastSeq <= 0 || len(state.Records) == 0 {
		t.Fatalf("snapshot content mismatch: %#v", state)
	}

	rawIndex, err := os.ReadFile(filepath.Join(root, indexFileName))
	if err != nil {
		t.Fatalf("read index failed: %v", err)
	}
	index := indexState{}
	if err := json.Unmarshal(rawIndex, &index); err != nil {
		t.Fatalf("decode index failed: %v", err)
	}
	if strings.TrimSpace(index.Checksum) == "" || strings.TrimSpace(index.SchemaVersion) == "" {
		t.Fatalf("index content mismatch: %#v", index)
	}
}

func TestFilesystemEngineCompactionContractWALAndRecovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory-store")
	cfg := FilesystemEngineConfig{
		RootDir: root,
		Compaction: FilesystemCompactionConfig{
			Enabled:     true,
			MinOps:      1,
			MaxWALBytes: 1,
		},
	}
	engine, err := NewFilesystemEngine(cfg)
	if err != nil {
		t.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	if _, err := engine.Upsert(UpsertRequest{
		Namespace: "ns",
		Records: []Record{
			{ID: "r1", Content: "one"},
		},
	}); err != nil {
		t.Fatalf("first Upsert failed: %v", err)
	}
	if _, err := engine.Upsert(UpsertRequest{
		Namespace: "ns",
		Records: []Record{
			{ID: "r2", Content: "two"},
		},
	}); err != nil {
		t.Fatalf("second Upsert failed: %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	walRaw, err := os.ReadFile(filepath.Join(root, walFileName))
	if err != nil {
		t.Fatalf("read WAL failed: %v", err)
	}
	if strings.TrimSpace(string(walRaw)) != "" {
		t.Fatalf("WAL should be compacted/truncated, got %q", string(walRaw))
	}

	recovered, err := NewFilesystemEngine(cfg)
	if err != nil {
		t.Fatalf("recover NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = recovered.Close() }()
	resp, err := recovered.Query(QueryRequest{Namespace: "ns"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("recovered total = %d, want 2", resp.Total)
	}
	got := map[string]bool{}
	for _, record := range resp.Records {
		got[record.ID] = true
	}
	if !got["r1"] || !got["r2"] {
		t.Fatalf("recovered records mismatch: %#v", resp.Records)
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

func TestFilesystemEngineQueryTTLReadPathAndWritePathCleanup(t *testing.T) {
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: filepath.Join(t.TempDir(), "memory-store"),
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
		Lifecycle: LifecycleConfig{
			TTLEnabled: true,
			TTL:        time.Second,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = engine.Close() }()

	now := time.Now().UTC()
	engine.mu.Lock()
	engine.records["ns"] = map[string]Record{
		"expired": {
			ID:        "expired",
			Namespace: "ns",
			Content:   "old",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		"fresh": {
			ID:        "fresh",
			Namespace: "ns",
			Content:   "new",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	engine.mu.Unlock()

	first, err := engine.Query(QueryRequest{Namespace: "ns"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if first.Total != 1 || len(first.Records) != 1 || first.Records[0].ID != "fresh" {
		t.Fatalf("query should filter expired record, got %#v", first.Records)
	}
	if first.MemoryLifecycleAction != LifecycleActionTTL {
		t.Fatalf("lifecycle action = %q, want %q", first.MemoryLifecycleAction, LifecycleActionTTL)
	}

	engine.mu.RLock()
	_, expiredStillPresent := engine.records["ns"]["expired"]
	engine.mu.RUnlock()
	if !expiredStillPresent {
		t.Fatal("query path should not mutate storage by deleting expired records")
	}

	if _, err := engine.Upsert(UpsertRequest{
		Namespace: "ns",
		Records: []Record{
			{ID: "writer", Content: "keep"},
		},
	}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	engine.mu.RLock()
	_, expiredAfterWrite := engine.records["ns"]["expired"]
	engine.mu.RUnlock()
	if expiredAfterWrite {
		t.Fatal("write path should cleanup expired records via TTL enforcement")
	}
}

func TestFilesystemEngineWALFsyncBatchSizeDefaultAndOptional(t *testing.T) {
	defaultEngine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: filepath.Join(t.TempDir(), "memory-default-sync"),
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine default failed: %v", err)
	}
	if defaultEngine.cfg.Compaction.FsyncBatchSize != 1 {
		t.Fatalf("default fsync_batch_size = %d, want 1", defaultEngine.cfg.Compaction.FsyncBatchSize)
	}
	if _, err := defaultEngine.Upsert(UpsertRequest{
		Namespace: "ns",
		Records: []Record{
			{ID: "r-default", Content: "default"},
		},
	}); err != nil {
		t.Fatalf("default upsert failed: %v", err)
	}
	defaultEngine.mu.RLock()
	defaultPending := defaultEngine.pendingFsyncOps
	defaultEngine.mu.RUnlock()
	if defaultPending != 0 {
		t.Fatalf("default sync mode should flush each write, pending=%d", defaultPending)
	}
	if err := defaultEngine.Close(); err != nil {
		t.Fatalf("default engine close failed: %v", err)
	}

	batchEngine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: filepath.Join(t.TempDir(), "memory-batch-sync"),
		Compaction: FilesystemCompactionConfig{
			Enabled:        false,
			FsyncBatchSize: 3,
		},
	})
	if err != nil {
		t.Fatalf("NewFilesystemEngine batch failed: %v", err)
	}
	defer func() { _ = batchEngine.Close() }()

	upsertAndPending := func(id string) int {
		t.Helper()
		if _, err := batchEngine.Upsert(UpsertRequest{
			Namespace: "ns",
			Records: []Record{
				{ID: id, Content: "batch"},
			},
		}); err != nil {
			t.Fatalf("batch upsert failed: %v", err)
		}
		batchEngine.mu.RLock()
		defer batchEngine.mu.RUnlock()
		return batchEngine.pendingFsyncOps
	}

	if pending := upsertAndPending("r-1"); pending != 1 {
		t.Fatalf("pending after first batch write = %d, want 1", pending)
	}
	if pending := upsertAndPending("r-2"); pending != 2 {
		t.Fatalf("pending after second batch write = %d, want 2", pending)
	}
	if pending := upsertAndPending("r-3"); pending != 0 {
		t.Fatalf("pending after third batch write = %d, want 0 (batched fsync)", pending)
	}
}

func BenchmarkMemoryFilesystemWriteUpsert(b *testing.B) {
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: filepath.Join(b.TempDir(), "memory-bench-write"),
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
	})
	if err != nil {
		b.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = engine.Close() }()

	record := Record{Content: "benchmark payload"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record.ID = fmt.Sprintf("record-%d", i%256)
		if _, err := engine.Upsert(UpsertRequest{
			Namespace: "bench",
			Records:   []Record{record},
		}); err != nil {
			b.Fatalf("Upsert failed: %v", err)
		}
	}
}

func BenchmarkMemoryFilesystemQueryReadPath(b *testing.B) {
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: filepath.Join(b.TempDir(), "memory-bench-query"),
		Compaction: FilesystemCompactionConfig{
			Enabled: false,
		},
		Lifecycle: LifecycleConfig{
			TTLEnabled: true,
			TTL:        24 * time.Hour,
		},
	})
	if err != nil {
		b.Fatalf("NewFilesystemEngine failed: %v", err)
	}
	defer func() { _ = engine.Close() }()

	batch := make([]Record, 0, 512)
	for i := 0; i < 512; i++ {
		batch = append(batch, Record{
			ID:      fmt.Sprintf("record-%d", i),
			Content: fmt.Sprintf("benchmark query payload %d", i),
		})
	}
	if _, err := engine.Upsert(UpsertRequest{
		Namespace: "bench",
		Records:   batch,
	}); err != nil {
		b.Fatalf("seed upsert failed: %v", err)
	}
	queryReq := QueryRequest{Namespace: "bench", Query: "benchmark query", MaxItems: 64}
	if warmup, err := engine.Query(queryReq); err != nil || warmup.Total == 0 {
		b.Fatalf("query warmup failed: total=%d err=%v", warmup.Total, err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, queryErr := engine.Query(queryReq)
		if queryErr != nil {
			b.Fatalf("Query failed: %v", queryErr)
		}
		if resp.Total == 0 {
			b.Fatal("Query returned no records")
		}
	}
}

func BenchmarkMemoryFilesystemCompactionCycle(b *testing.B) {
	baseRoot := filepath.Join(b.TempDir(), "memory-bench-compaction")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := filepath.Join(baseRoot, fmt.Sprintf("run-%d", i))
		engine, err := NewFilesystemEngine(FilesystemEngineConfig{
			RootDir: root,
			Compaction: FilesystemCompactionConfig{
				Enabled:     true,
				MinOps:      1,
				MaxWALBytes: 1,
			},
		})
		if err != nil {
			b.Fatalf("NewFilesystemEngine failed: %v", err)
		}
		if _, err := engine.Upsert(UpsertRequest{
			Namespace: "bench",
			Records: []Record{
				{ID: "r1", Content: "compaction benchmark payload"},
			},
		}); err != nil {
			_ = engine.Close()
			b.Fatalf("Upsert failed: %v", err)
		}
		if err := engine.Close(); err != nil {
			b.Fatalf("Close failed: %v", err)
		}
	}
}
