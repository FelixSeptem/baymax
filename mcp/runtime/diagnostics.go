package runtime

import (
	"sync"
	"time"
)

type CallRecord struct {
	Time           time.Time `json:"time"`
	Transport      string    `json:"transport"`
	Profile        string    `json:"profile"`
	CallID         string    `json:"call_id"`
	Tool           string    `json:"tool"`
	LatencyMs      int64     `json:"latency_ms"`
	RetryCount     int       `json:"retry_count"`
	ReconnectCount int       `json:"reconnect_count"`
	ErrorClass     string    `json:"error_class,omitempty"`
}

type Diagnostics struct {
	mu      sync.Mutex
	maxKeep int
	records []CallRecord
}

func NewDiagnostics(maxKeep int) *Diagnostics {
	if maxKeep <= 0 {
		maxKeep = 200
	}
	return &Diagnostics{maxKeep: maxKeep, records: make([]CallRecord, 0, maxKeep)}
}

func (d *Diagnostics) Add(rec CallRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records = append(d.records, rec)
	if len(d.records) > d.maxKeep {
		d.records = append([]CallRecord(nil), d.records[len(d.records)-d.maxKeep:]...)
	}
}

func (d *Diagnostics) Recent(n int) []CallRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	if n <= 0 || n > len(d.records) {
		n = len(d.records)
	}
	out := make([]CallRecord, n)
	copy(out, d.records[len(d.records)-n:])
	return out
}
