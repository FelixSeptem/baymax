package mailbox

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkMailboxFileStorePersistHeartbeat(b *testing.B) {
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
			path := filepath.Join(b.TempDir(), "mailbox-state.json")
			store, err := NewFileStore(path, Policy{}, tc.opts...)
			if err != nil {
				b.Fatalf("new file store: %v", err)
			}
			base := time.Unix(1_700_000_000, 0).UTC()
			if _, err := store.Publish(ctx, Envelope{
				MessageID:      "msg-bench",
				IdempotencyKey: "idem-bench",
				Kind:           KindCommand,
			}, base); err != nil {
				b.Fatalf("publish setup message: %v", err)
			}
			record, ok, err := store.Consume(ctx, "consumer-bench", base.Add(time.Millisecond), 2*time.Second, false)
			if err != nil || !ok {
				b.Fatalf("consume setup message: ok=%v err=%v", ok, err)
			}

			now := base.Add(2 * time.Millisecond)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				now = now.Add(10 * time.Millisecond)
				updated, heartbeatErr := store.Heartbeat(
					ctx,
					record.Envelope.MessageID,
					"consumer-bench",
					now,
					2*time.Second,
				)
				if heartbeatErr != nil {
					b.Fatalf("heartbeat: %v", heartbeatErr)
				}
				record = updated
			}
			b.StopTimer()
			if err := store.Flush(); err != nil {
				b.Fatalf("flush pending writes: %v", err)
			}
		})
	}
}
