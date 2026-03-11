package runtime

import (
	"strings"
	"sync"
	"time"
)

type RunRecord struct {
	Time       time.Time `json:"time"`
	RunID      string    `json:"run_id"`
	Iterations int       `json:"iterations"`
	ToolCalls  int       `json:"tool_calls"`
	LatencyMs  int64     `json:"latency_ms"`
	ErrorClass string    `json:"error_class,omitempty"`
}

type ReloadRecord struct {
	Time    time.Time `json:"time"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

type RuntimeDiagnostics struct {
	mu sync.RWMutex

	maxCallRecords  int
	maxRunRecords   int
	maxReloadErrors int

	calls   []CallRecord
	runs    []RunRecord
	reloads []ReloadRecord
}

func NewRuntimeDiagnostics(maxCalls, maxRuns, maxReloads int) *RuntimeDiagnostics {
	if maxCalls <= 0 {
		maxCalls = 200
	}
	if maxRuns <= 0 {
		maxRuns = 200
	}
	if maxReloads <= 0 {
		maxReloads = 100
	}
	return &RuntimeDiagnostics{
		maxCallRecords:  maxCalls,
		maxRunRecords:   maxRuns,
		maxReloadErrors: maxReloads,
		calls:           make([]CallRecord, 0, maxCalls),
		runs:            make([]RunRecord, 0, maxRuns),
		reloads:         make([]ReloadRecord, 0, maxReloads),
	}
}

func (d *RuntimeDiagnostics) Resize(maxCalls, maxRuns, maxReloads int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if maxCalls > 0 {
		d.maxCallRecords = maxCalls
		d.calls = trimTail(d.calls, d.maxCallRecords)
	}
	if maxRuns > 0 {
		d.maxRunRecords = maxRuns
		d.runs = trimTail(d.runs, d.maxRunRecords)
	}
	if maxReloads > 0 {
		d.maxReloadErrors = maxReloads
		d.reloads = trimTail(d.reloads, d.maxReloadErrors)
	}
}

func (d *RuntimeDiagnostics) AddCall(rec CallRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, rec)
	d.calls = trimTail(d.calls, d.maxCallRecords)
}

func (d *RuntimeDiagnostics) AddRun(rec RunRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.runs = append(d.runs, rec)
	d.runs = trimTail(d.runs, d.maxRunRecords)
}

func (d *RuntimeDiagnostics) AddReload(rec ReloadRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reloads = append(d.reloads, rec)
	d.reloads = trimTail(d.reloads, d.maxReloadErrors)
}

func (d *RuntimeDiagnostics) RecentCalls(n int) []CallRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.calls, n)
}

func (d *RuntimeDiagnostics) RecentRuns(n int) []RunRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.runs, n)
}

func (d *RuntimeDiagnostics) RecentReloads(n int) []ReloadRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.reloads, n)
}

func sanitizeMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		if isSensitiveKey(k) {
			out[k] = "***"
			continue
		}
		switch tv := v.(type) {
		case map[string]any:
			out[k] = sanitizeMap(tv)
		case []any:
			out[k] = sanitizeSlice(tv)
		default:
			out[k] = v
		}
	}
	return out
}

func sanitizeSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out = append(out, sanitizeMap(tv))
		case []any:
			out = append(out, sanitizeSlice(tv))
		default:
			out = append(out, v)
		}
	}
	return out
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(k, "secret") ||
		strings.Contains(k, "token") ||
		strings.Contains(k, "password") ||
		strings.Contains(k, "api_key") ||
		strings.Contains(k, "apikey")
}

func trimTail[T any](src []T, n int) []T {
	if n <= 0 || len(src) <= n {
		return src
	}
	dst := make([]T, n)
	copy(dst, src[len(src)-n:])
	return dst
}

func tailCopy[T any](src []T, n int) []T {
	if n <= 0 || n > len(src) {
		n = len(src)
	}
	dst := make([]T, n)
	copy(dst, src[len(src)-n:])
	return dst
}
