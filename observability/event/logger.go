package event

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type JSONLogger struct {
	mu  sync.Mutex
	out io.Writer
}

func NewJSONLogger(out io.Writer) *JSONLogger {
	if out == nil {
		out = os.Stdout
	}
	return &JSONLogger{out: out}
}

func (l *JSONLogger) OnEvent(ctx context.Context, ev types.Event) {
	entry := map[string]any{
		"time":      ev.Time.Format(time.RFC3339Nano),
		"version":   ev.Version,
		"type":      ev.Type,
		"run_id":    ev.RunID,
		"iteration": ev.Iteration,
		"call_id":   ev.CallID,
		"trace_id":  ev.TraceID,
		"span_id":   ev.SpanID,
		"payload":   ev.Payload,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.out.Write(append(data, '\n'))
}

var _ types.EventHandler = (*JSONLogger)(nil)
