package diagnostics

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	Time                 time.Time `json:"time"`
	RunID                string    `json:"run_id"`
	Status               string    `json:"status,omitempty"`
	Iterations           int       `json:"iterations"`
	ToolCalls            int       `json:"tool_calls"`
	LatencyMs            int64     `json:"latency_ms"`
	ErrorClass           string    `json:"error_class,omitempty"`
	ModelProvider        string    `json:"model_provider,omitempty"`
	FallbackUsed         bool      `json:"fallback_used,omitempty"`
	FallbackInitial      string    `json:"fallback_initial,omitempty"`
	FallbackPath         string    `json:"fallback_path,omitempty"`
	RequiredCapabilities string    `json:"required_capabilities,omitempty"`
	FallbackReason       string    `json:"fallback_reason,omitempty"`
	PrefixHash           string    `json:"prefix_hash,omitempty"`
	AssembleLatencyMs    int64     `json:"assemble_latency_ms,omitempty"`
	AssembleStatus       string    `json:"assemble_status,omitempty"`
	GuardViolation       string    `json:"guard_violation,omitempty"`
	AssembleStageStatus  string    `json:"assemble_stage_status,omitempty"`
	Stage2SkipReason     string    `json:"stage2_skip_reason,omitempty"`
	Stage1LatencyMs      int64     `json:"stage1_latency_ms,omitempty"`
	Stage2LatencyMs      int64     `json:"stage2_latency_ms,omitempty"`
	Stage2Provider       string    `json:"stage2_provider,omitempty"`
	RecapStatus          string    `json:"recap_status,omitempty"`
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
	runKeys map[string]int
	sklKeys map[string]int
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
		runKeys:         make(map[string]int, maxRuns),
		sklKeys:         make(map[string]int, maxSkills),
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
		d.rebuildRunKeys()
	}
	if maxReloads > 0 {
		d.maxReloadErrors = maxReloads
		d.reloads = trimTail(d.reloads, d.maxReloadErrors)
	}
	if maxSkills > 0 {
		d.maxSkillRecords = maxSkills
		d.skills = trimTail(d.skills, d.maxSkillRecords)
		d.rebuildSkillKeys()
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
	rec.Status = normalizeRunStatus(rec.Status, rec.ErrorClass)
	key := RunIdempotencyKey(rec)
	if idx, ok := d.runKeys[key]; ok && idx >= 0 && idx < len(d.runs) {
		d.runs[idx] = rec
		return
	}
	d.runs = append(d.runs, rec)
	d.runs = trimTail(d.runs, d.maxRunRecords)
	d.rebuildRunKeys()
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
	rec.Status = normalizeSkillStatus(rec.Status)
	key := SkillIdempotencyKey(rec)
	if idx, ok := d.sklKeys[key]; ok && idx >= 0 && idx < len(d.skills) {
		d.skills[idx] = rec
		return
	}
	d.skills = append(d.skills, rec)
	d.skills = trimTail(d.skills, d.maxSkillRecords)
	d.rebuildSkillKeys()
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

func RunIdempotencyKey(rec RunRecord) string {
	status := normalizeRunStatus(rec.Status, rec.ErrorClass)
	if strings.TrimSpace(rec.RunID) != "" {
		return fmt.Sprintf("run:%s:%s", strings.TrimSpace(rec.RunID), status)
	}
	return fmt.Sprintf(
		"run:anon:%d:%d:%d:%s:%s",
		rec.Iterations,
		rec.ToolCalls,
		rec.LatencyMs,
		status,
		strings.TrimSpace(rec.ErrorClass),
	)
}

func SkillIdempotencyKey(rec SkillRecord) string {
	return fmt.Sprintf(
		"skill:%s:%s:%s:%s:%s:%s",
		strings.TrimSpace(rec.RunID),
		strings.TrimSpace(rec.SkillName),
		strings.TrimSpace(rec.Action),
		normalizeSkillStatus(rec.Status),
		strings.TrimSpace(rec.ErrorClass),
		payloadDigest(rec.Payload),
	)
}

func normalizeRunStatus(status, errorClass string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "success", "failed":
		return s
	}
	if strings.TrimSpace(errorClass) != "" {
		return "failed"
	}
	return "success"
}

func normalizeSkillStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "success", "failed", "warning":
		return s
	default:
		return "warning"
	}
}

func payloadDigest(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	raw, err := json.Marshal(normalizePayloadForKey(payload))
	if err != nil {
		return "marshal_error"
	}
	sum := sha1.Sum(raw)
	return hex.EncodeToString(sum[:])
}

func normalizePayloadForKey(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := strings.ToLower(strings.TrimSpace(k))
		if lk == "latency_ms" || lk == "time" || lk == "timestamp" {
			continue
		}
		switch tv := v.(type) {
		case map[string]any:
			out[k] = normalizePayloadForKey(tv)
		case []any:
			out[k] = normalizeSliceForKey(tv)
		default:
			out[k] = v
		}
	}
	return out
}

func normalizeSliceForKey(in []any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out = append(out, normalizePayloadForKey(tv))
		case []any:
			out = append(out, normalizeSliceForKey(tv))
		default:
			out = append(out, v)
		}
	}
	return out
}

func (d *Store) rebuildRunKeys() {
	d.runKeys = make(map[string]int, len(d.runs))
	for i := range d.runs {
		d.runKeys[RunIdempotencyKey(d.runs[i])] = i
	}
}

func (d *Store) rebuildSkillKeys() {
	d.sklKeys = make(map[string]int, len(d.skills))
	for i := range d.skills {
		d.sklKeys[SkillIdempotencyKey(d.skills[i])] = i
	}
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
