package diag

import (
	"testing"
	"time"
)

func TestStoreRecentWithoutOverflow(t *testing.T) {
	store := NewStore(4)
	base := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		store.Add(CallRecord{
			Time:      base.Add(time.Duration(i) * time.Second),
			Transport: "http",
			CallID:    "call-" + string(rune('a'+i)),
			Tool:      "tool",
		})
	}

	items := store.Recent(10)
	if len(items) != 3 {
		t.Fatalf("recent len=%d, want 3", len(items))
	}
	if items[0].CallID != "call-a" || items[1].CallID != "call-b" || items[2].CallID != "call-c" {
		t.Fatalf("recent order mismatch: %#v", items)
	}
}

func TestStoreRecentOverflowKeepsTailInOrder(t *testing.T) {
	store := NewStore(3)
	base := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 6; i++ {
		store.Add(CallRecord{
			Time:      base.Add(time.Duration(i) * time.Second),
			Transport: "stdio",
			CallID:    "call-" + string(rune('0'+i)),
			Tool:      "tool",
		})
	}

	items := store.Recent(3)
	if len(items) != 3 {
		t.Fatalf("recent len=%d, want 3", len(items))
	}
	if items[0].CallID != "call-3" || items[1].CallID != "call-4" || items[2].CallID != "call-5" {
		t.Fatalf("recent tail order mismatch: %#v", items)
	}

	lastTwo := store.Recent(2)
	if len(lastTwo) != 2 {
		t.Fatalf("recent(2) len=%d, want 2", len(lastTwo))
	}
	if lastTwo[0].CallID != "call-4" || lastTwo[1].CallID != "call-5" {
		t.Fatalf("recent(2) order mismatch: %#v", lastTwo)
	}
}
