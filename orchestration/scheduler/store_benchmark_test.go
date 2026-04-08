package scheduler

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkSchedulerFileStorePersistHeartbeat(b *testing.B) {
	cases := []struct {
		name string
		opts []FileStoreOption
	}{
		{
			name: "immediate",
			opts: nil,
		},
		{
			name: "group_commit_batch_8",
			opts: []FileStoreOption{
				WithPersistBatchSize(8),
				WithPersistDebounce(time.Hour),
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			ctx := context.Background()
			path := filepath.Join(b.TempDir(), "scheduler-state.json")
			store, err := NewFileStore(path, tc.opts...)
			if err != nil {
				b.Fatalf("new file store: %v", err)
			}
			base := time.Unix(1_700_000_000, 0).UTC()
			if _, err := store.Enqueue(ctx, Task{
				TaskID: "task-bench",
				RunID:  "run-bench",
			}, base); err != nil {
				b.Fatalf("enqueue setup task: %v", err)
			}
			claimed, ok, err := store.Claim(ctx, "worker-bench", base.Add(time.Millisecond), 2*time.Second)
			if err != nil || !ok {
				b.Fatalf("claim setup task: ok=%v err=%v", ok, err)
			}

			now := base.Add(2 * time.Millisecond)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				now = now.Add(10 * time.Millisecond)
				updated, heartbeatErr := store.Heartbeat(
					ctx,
					claimed.Record.Task.TaskID,
					claimed.Attempt.AttemptID,
					claimed.Attempt.LeaseToken,
					now,
					2*time.Second,
				)
				if heartbeatErr != nil {
					b.Fatalf("heartbeat: %v", heartbeatErr)
				}
				claimed = updated
			}
			b.StopTimer()
			if err := store.Flush(); err != nil {
				b.Fatalf("flush pending writes: %v", err)
			}
		})
	}
}
