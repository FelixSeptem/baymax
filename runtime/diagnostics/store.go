package diagnostics

import (
	"strings"
	"sync"
	"time"
)

type CallRecord struct {
	Time           time.Time `json:"time"`
	Component      string    `json:"component"`
	Transport      string    `json:"transport,omitempty"`
	Profile        string    `json:"profile,omitempty"`
	RunID          string    `json:"run_id,omitempty"`
	CallID         string    `json:"call_id,omitempty"`
	Name           string    `json:"name,omitempty"`
	Action         string    `json:"action,omitempty"`
	LatencyMs      int64     `json:"latency_ms"`
	RetryCount     int       `json:"retry_count"`
	ReconnectCount int       `json:"reconnect_count"`
	ErrorClass     string    `json:"error_class,omitempty"`
}

type RunRecord struct {
	Time       time.Time `json:"time"`
	RunID      string    `json:"run_id"`
	Iterations int       `json:"iterations"`
	ToolCalls  int       `json:"tool_calls"`
	LatencyMs  int64     `json:"latency_ms"`
	ErrorClass string    `json:"error_class,omitempty"`
}

type SkillRecord struct {
	Time       time.Time      `json:"time"`
	RunID      string         `json:"run_id,omitempty"`
	SkillName  string         `json:"skill_name,omitempty"`
	Action     string         `json:"action"`
	Status     string         `json:"status"`
	LatencyMs  int64          `json:"latency_ms,omitempty"`
	ErrorClass string         `json:"error_class,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type ReloadRecord struct {
	Time    time.Time `json:"time"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

type Store struct {
	mu sync.RWMutex

	maxCallRecords  int
	maxRunRecords   int
	maxReloadErrors int
	maxSkillRecords int

	calls   []CallRecord
	runs    []RunRecord
	reloads []ReloadRecord
	skills  []SkillRecord
}

func NewStore(maxCalls, maxRuns, maxReloads, maxSkills int) *Store {
	if maxCalls <= 0 {
		maxCalls = 200
	}
	if maxRuns <= 0 {
		maxRuns = 200
	}
	if maxReloads <= 0 {
		maxReloads = 100
	}
	if maxSkills <= 0 {
		maxSkills = 200
	}
	return &Store{
		maxCallRecords:  maxCalls,
		maxRunRecords:   maxRuns,
		maxReloadErrors: maxReloads,
		maxSkillRecords: maxSkills,
		calls:           make([]CallRecord, 0, maxCalls),
		runs:            make([]RunRecord, 0, maxRuns),
		reloads:         make([]ReloadRecord, 0, maxReloads),
		skills:          make([]SkillRecord, 0, maxSkills),
	}
}

func (d *Store) Resize(maxCalls, maxRuns, maxReloads, maxSkills int) {
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
	if maxSkills > 0 {
		d.maxSkillRecords = maxSkills
		d.skills = trimTail(d.skills, d.maxSkillRecords)
	}
}

func (d *Store) AddCall(rec CallRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, rec)
	d.calls = trimTail(d.calls, d.maxCallRecords)
}

func (d *Store) AddRun(rec RunRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.runs = append(d.runs, rec)
	d.runs = trimTail(d.runs, d.maxRunRecords)
}

func (d *Store) AddReload(rec ReloadRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reloads = append(d.reloads, rec)
	d.reloads = trimTail(d.reloads, d.maxReloadErrors)
}

func (d *Store) AddSkill(rec SkillRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.skills = append(d.skills, rec)
	d.skills = trimTail(d.skills, d.maxSkillRecords)
}

func (d *Store) RecentCalls(n int) []CallRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.calls, n)
}

func (d *Store) RecentRuns(n int) []RunRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.runs, n)
}

func (d *Store) RecentReloads(n int) []ReloadRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.reloads, n)
}

func (d *Store) RecentSkills(n int) []SkillRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.skills, n)
}

func SanitizeMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		if isSensitiveKey(k) {
			out[k] = "***"
			continue
		}
		switch tv := v.(type) {
		case map[string]any:
			out[k] = SanitizeMap(tv)
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
			out = append(out, SanitizeMap(tv))
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
