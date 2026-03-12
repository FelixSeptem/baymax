package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProviderRejectsPlaceholderInCA2(t *testing.T) {
	_, err := New("rag", "")
	if !errors.Is(err, ErrProviderNotReady) {
		t.Fatalf("err = %v, want ErrProviderNotReady", err)
	}
}

func TestFileProviderFetch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"s1","content":"c1"}`,
		`{"session_id":"s1","content":"c2"}`,
		`{"session_id":"s2","content":"x"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	p, err := New("file", path)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	resp, err := p.Fetch(context.Background(), Request{SessionID: "s1", MaxItems: 2})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(resp.Chunks) != 2 || resp.Chunks[0] != "c1" || resp.Chunks[1] != "c2" {
		t.Fatalf("unexpected chunks: %#v", resp.Chunks)
	}
}
