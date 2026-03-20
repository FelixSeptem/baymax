package mailbox

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type QueryTimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

type QuerySort struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type QueryRequest struct {
	MessageID      string          `json:"message_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	Kind           string          `json:"kind,omitempty"`
	State          string          `json:"state,omitempty"`
	FromAgent      string          `json:"from_agent,omitempty"`
	ToAgent        string          `json:"to_agent,omitempty"`
	TaskID         string          `json:"task_id,omitempty"`
	RunID          string          `json:"run_id,omitempty"`
	WorkflowID     string          `json:"workflow_id,omitempty"`
	TeamID         string          `json:"team_id,omitempty"`
	TimeRange      *QueryTimeRange `json:"time_range,omitempty"`
	PageSize       *int            `json:"page_size,omitempty"`
	Sort           QuerySort       `json:"sort,omitempty"`
	Cursor         string          `json:"cursor,omitempty"`
}

type QueryResult struct {
	Items      []Record `json:"items"`
	NextCursor string   `json:"next_cursor,omitempty"`
	PageSize   int      `json:"page_size"`
	SortField  string   `json:"sort_field"`
	SortOrder  string   `json:"sort_order"`
}

type normalizedQuery struct {
	MessageID      string
	IdempotencyKey string
	CorrelationID  string
	Kind           string
	State          string
	FromAgent      string
	ToAgent        string
	TaskID         string
	RunID          string
	WorkflowID     string
	TeamID         string
	TimeRange      *QueryTimeRange
	PageSize       int
	SortField      string
	SortOrder      string
	Cursor         string
}

type queryCursor struct {
	Offset    int    `json:"offset"`
	QueryHash string `json:"query_hash"`
}

func querySnapshot(snapshot Snapshot, req QueryRequest) (QueryResult, error) {
	q, err := normalizeQuery(req)
	if err != nil {
		return QueryResult{}, err
	}
	queryHash := queryHash(q)
	start, err := decodeCursor(q.Cursor, queryHash)
	if err != nil {
		return QueryResult{}, err
	}

	filtered := make([]Record, 0, len(snapshot.Records))
	for i := range snapshot.Records {
		if matchesQuery(snapshot.Records[i], q) {
			filtered = append(filtered, cloneRecord(snapshot.Records[i]))
		}
	}
	sortQuery(filtered, q.SortField, q.SortOrder)

	if start > len(filtered) {
		return QueryResult{}, fmt.Errorf("invalid query cursor")
	}
	end := start + q.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	items := append([]Record(nil), filtered[start:end]...)
	nextCursor := ""
	if end < len(filtered) {
		encoded, err := encodeCursor(queryCursor{
			Offset:    end,
			QueryHash: queryHash,
		})
		if err != nil {
			return QueryResult{}, err
		}
		nextCursor = encoded
	}
	return QueryResult{
		Items:      items,
		NextCursor: nextCursor,
		PageSize:   q.PageSize,
		SortField:  q.SortField,
		SortOrder:  q.SortOrder,
	}, nil
}

func normalizeQuery(req QueryRequest) (normalizedQuery, error) {
	pageSize := DefaultQueryPageSize
	if req.PageSize != nil {
		if *req.PageSize <= 0 || *req.PageSize > MaxQueryPageSize {
			return normalizedQuery{}, fmt.Errorf("page_size must be within [1,%d]", MaxQueryPageSize)
		}
		pageSize = *req.PageSize
	}
	sortField := strings.ToLower(strings.TrimSpace(req.Sort.Field))
	if sortField == "" {
		sortField = "updated_at"
	}
	if sortField != "updated_at" && sortField != "created_at" {
		return normalizedQuery{}, fmt.Errorf("unsupported sort.field %q", req.Sort.Field)
	}
	sortOrder := strings.ToLower(strings.TrimSpace(req.Sort.Order))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return normalizedQuery{}, fmt.Errorf("unsupported sort.order %q", req.Sort.Order)
	}

	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind != "" {
		switch kind {
		case string(KindCommand), string(KindEvent), string(KindResult):
		default:
			return normalizedQuery{}, fmt.Errorf("unsupported kind filter %q", req.Kind)
		}
	}
	state := strings.ToLower(strings.TrimSpace(req.State))
	if state != "" {
		switch state {
		case string(StateQueued), string(StateInFlight), string(StateAcked), string(StateNacked), string(StateDeadLetter), string(StateExpired):
		default:
			return normalizedQuery{}, fmt.Errorf("unsupported state filter %q", req.State)
		}
	}
	var tr *QueryTimeRange
	if req.TimeRange != nil {
		start := req.TimeRange.Start
		end := req.TimeRange.End
		if !start.IsZero() {
			start = start.UTC()
		}
		if !end.IsZero() {
			end = end.UTC()
		}
		if !start.IsZero() && !end.IsZero() && start.After(end) {
			return normalizedQuery{}, fmt.Errorf("time_range.start must be <= time_range.end")
		}
		tr = &QueryTimeRange{Start: start, End: end}
	}
	return normalizedQuery{
		MessageID:      strings.TrimSpace(req.MessageID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(req.CorrelationID),
		Kind:           kind,
		State:          state,
		FromAgent:      strings.TrimSpace(req.FromAgent),
		ToAgent:        strings.TrimSpace(req.ToAgent),
		TaskID:         strings.TrimSpace(req.TaskID),
		RunID:          strings.TrimSpace(req.RunID),
		WorkflowID:     strings.TrimSpace(req.WorkflowID),
		TeamID:         strings.TrimSpace(req.TeamID),
		TimeRange:      tr,
		PageSize:       pageSize,
		SortField:      sortField,
		SortOrder:      sortOrder,
		Cursor:         strings.TrimSpace(req.Cursor),
	}, nil
}

func queryHash(q normalizedQuery) string {
	start := int64(0)
	end := int64(0)
	if q.TimeRange != nil {
		if !q.TimeRange.Start.IsZero() {
			start = q.TimeRange.Start.UnixNano()
		}
		if !q.TimeRange.End.IsZero() {
			end = q.TimeRange.End.UnixNano()
		}
	}
	raw := strings.Join([]string{
		q.MessageID,
		q.IdempotencyKey,
		q.CorrelationID,
		q.Kind,
		q.State,
		q.FromAgent,
		q.ToAgent,
		q.TaskID,
		q.RunID,
		q.WorkflowID,
		q.TeamID,
		fmt.Sprintf("%d", start),
		fmt.Sprintf("%d", end),
		q.SortField,
		q.SortOrder,
		fmt.Sprintf("%d", q.PageSize),
	}, "|")
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func matchesQuery(record Record, q normalizedQuery) bool {
	env := record.Envelope
	if q.MessageID != "" && strings.TrimSpace(env.MessageID) != q.MessageID {
		return false
	}
	if q.IdempotencyKey != "" && strings.TrimSpace(env.IdempotencyKey) != q.IdempotencyKey {
		return false
	}
	if q.CorrelationID != "" && strings.TrimSpace(env.CorrelationID) != q.CorrelationID {
		return false
	}
	if q.Kind != "" && strings.TrimSpace(string(env.Kind)) != q.Kind {
		return false
	}
	if q.State != "" && strings.TrimSpace(string(record.State)) != q.State {
		return false
	}
	if q.FromAgent != "" && strings.TrimSpace(env.FromAgent) != q.FromAgent {
		return false
	}
	if q.ToAgent != "" && strings.TrimSpace(env.ToAgent) != q.ToAgent {
		return false
	}
	if q.TaskID != "" && strings.TrimSpace(env.TaskID) != q.TaskID {
		return false
	}
	if q.RunID != "" && strings.TrimSpace(env.RunID) != q.RunID {
		return false
	}
	if q.WorkflowID != "" && strings.TrimSpace(env.WorkflowID) != q.WorkflowID {
		return false
	}
	if q.TeamID != "" && strings.TrimSpace(env.TeamID) != q.TeamID {
		return false
	}
	if q.TimeRange != nil {
		ts := record.UpdatedAt
		if !q.TimeRange.Start.IsZero() {
			if ts.IsZero() || ts.Before(q.TimeRange.Start) {
				return false
			}
		}
		if !q.TimeRange.End.IsZero() {
			if ts.IsZero() || ts.After(q.TimeRange.End) {
				return false
			}
		}
	}
	return true
}

func sortQuery(records []Record, field, order string) {
	desc := strings.ToLower(strings.TrimSpace(order)) != "asc"
	sort.SliceStable(records, func(i, j int) bool {
		left := records[i]
		right := records[j]
		leftTS := querySortTimestamp(left, field)
		rightTS := querySortTimestamp(right, field)
		if leftTS.Equal(rightTS) {
			return strings.TrimSpace(left.Envelope.MessageID) < strings.TrimSpace(right.Envelope.MessageID)
		}
		if desc {
			return leftTS.After(rightTS)
		}
		return leftTS.Before(rightTS)
	})
}

func querySortTimestamp(record Record, field string) time.Time {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "created_at":
		return record.CreatedAt
	default:
		return record.UpdatedAt
	}
}

func encodeCursor(cursor queryCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("encode query cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCursor(cursor, expectedHash string) (int, error) {
	if strings.TrimSpace(cursor) == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return 0, fmt.Errorf("invalid query cursor")
	}
	var decoded queryCursor
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, fmt.Errorf("invalid query cursor")
	}
	if decoded.Offset < 0 || strings.TrimSpace(decoded.QueryHash) == "" {
		return 0, fmt.Errorf("invalid query cursor")
	}
	if strings.TrimSpace(decoded.QueryHash) != strings.TrimSpace(expectedHash) {
		return 0, fmt.Errorf("invalid query cursor")
	}
	return decoded.Offset, nil
}
