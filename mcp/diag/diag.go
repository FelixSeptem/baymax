package diag

import (
	"sync"
	"time"
)

type CallRecord struct {
	Time           time.Time `json:"time"`
	Transport      string    `json:"transport"`
	Profile        string    `json:"profile"`
	RunID          string    `json:"run_id,omitempty"`
	CallID         string    `json:"call_id"`
	Tool           string    `json:"tool"`
	Action         string    `json:"action,omitempty"`
	LatencyMs      int64     `json:"latency_ms"`
	RetryCount     int       `json:"retry_count"`
	ReconnectCount int       `json:"reconnect_count"`
	ErrorClass     string    `json:"error_class,omitempty"`
}

type Store struct {
	mu      sync.Mutex
	maxKeep int
	records []CallRecord
	next    int
	count   int
}

func NewStore(maxKeep int) *Store {
	if maxKeep <= 0 {
		maxKeep = 200
	}
	return &Store{
		maxKeep: maxKeep,
		records: make([]CallRecord, maxKeep),
	}
}

func (d *Store) Add(rec CallRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.maxKeep <= 0 || len(d.records) == 0 {
		return
	}
	d.records[d.next] = rec
	d.next = (d.next + 1) % d.maxKeep
	if d.count < d.maxKeep {
		d.count++
	}
}

func (d *Store) Recent(n int) []CallRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.count == 0 {
		return nil
	}
	if n <= 0 || n > d.count {
		n = d.count
	}
	out := make([]CallRecord, n)
	oldest := d.next - d.count
	if oldest < 0 {
		oldest += d.maxKeep
	}
	start := oldest + (d.count - n)
	for i := 0; i < n; i++ {
		idx := (start + i) % d.maxKeep
		out[i] = d.records[idx]
	}
	return out
}
