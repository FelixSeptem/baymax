package event

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestJSONLoggerIncludesCorrelationFields(t *testing.T) {
	var b strings.Builder
	l := NewJSONLogger(&b)
	ev := types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.started",
		RunID:   "run-1",
		TraceID: "trace-1",
		SpanID:  "span-1",
		Time:    time.Unix(0, 0).UTC(),
	}
	l.OnEvent(context.Background(), ev)

	line := strings.TrimSpace(b.String())
	if line == "" {
		t.Fatal("expected log line")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["run_id"] != "run-1" || got["trace_id"] != "trace-1" || got["span_id"] != "span-1" {
		t.Fatalf("missing correlation fields: %#v", got)
	}
}
