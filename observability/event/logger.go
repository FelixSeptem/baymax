package event

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type JSONLogger struct {
	mu         sync.Mutex
	out        io.Writer
	encoder    *json.Encoder
	runtimeMgr *runtimeconfig.Manager
}

func NewJSONLogger(out io.Writer) *JSONLogger {
	if out == nil {
		out = os.Stdout
	}
	return &JSONLogger{
		out:     out,
		encoder: json.NewEncoder(out),
	}
}

func NewJSONLoggerWithRuntimeManager(out io.Writer, mgr *runtimeconfig.Manager) *JSONLogger {
	l := NewJSONLogger(out)
	l.runtimeMgr = mgr
	return l
}

func (l *JSONLogger) OnEvent(ctx context.Context, ev types.Event) {
	payload := ev.Payload
	runtimeLoadedAt := ""
	runtimeActiveProfile := ""
	if l.runtimeMgr != nil && len(payload) > 0 {
		payload = l.runtimeMgr.RedactPayload(payload)
	}
	if l.runtimeMgr != nil {
		s := l.runtimeMgr.CurrentSnapshot()
		runtimeLoadedAt = s.LoadedAt.Format(time.RFC3339Nano)
		runtimeActiveProfile = s.Config.MCP.ActiveProfile
	}
	entry := jsonLogEntry{
		Time:      ev.Time.Format(time.RFC3339Nano),
		Version:   ev.Version,
		Type:      ev.Type,
		RunID:     ev.RunID,
		Iteration: ev.Iteration,
		CallID:    ev.CallID,
		TraceID:   ev.TraceID,
		SpanID:    ev.SpanID,
		Payload:   payload,
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.runtimeMgr != nil {
		_ = l.encoder.Encode(jsonLogEntryWithRuntime{
			jsonLogEntry:         entry,
			RuntimeLoadedAt:      runtimeLoadedAt,
			RuntimeActiveProfile: runtimeActiveProfile,
		})
		return
	}
	_ = l.encoder.Encode(entry)
}

var _ types.EventHandler = (*JSONLogger)(nil)

type jsonLogEntry struct {
	Time      string         `json:"time"`
	Version   string         `json:"version"`
	Type      string         `json:"type"`
	RunID     string         `json:"run_id"`
	Iteration int            `json:"iteration"`
	CallID    string         `json:"call_id"`
	TraceID   string         `json:"trace_id"`
	SpanID    string         `json:"span_id"`
	Payload   map[string]any `json:"payload"`
}

type jsonLogEntryWithRuntime struct {
	jsonLogEntry
	RuntimeLoadedAt      string `json:"runtime_loaded_at"`
	RuntimeActiveProfile string `json:"runtime_active_profile"`
}
