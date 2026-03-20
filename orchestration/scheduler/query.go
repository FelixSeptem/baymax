package scheduler

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	DefaultTaskBoardQueryPageSize = 50
	MaxTaskBoardQueryPageSize     = 200
)

type TaskBoardQueryTimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

type TaskBoardQuerySort struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type TaskBoardQueryRequest struct {
	TaskID      string                   `json:"task_id,omitempty"`
	RunID       string                   `json:"run_id,omitempty"`
	WorkflowID  string                   `json:"workflow_id,omitempty"`
	TeamID      string                   `json:"team_id,omitempty"`
	State       string                   `json:"state,omitempty"`
	Priority    string                   `json:"priority,omitempty"`
	AgentID     string                   `json:"agent_id,omitempty"`
	PeerID      string                   `json:"peer_id,omitempty"`
	ParentRunID string                   `json:"parent_run_id,omitempty"`
	TimeRange   *TaskBoardQueryTimeRange `json:"time_range,omitempty"`
	PageSize    *int                     `json:"page_size,omitempty"`
	Sort        TaskBoardQuerySort       `json:"sort,omitempty"`
	Cursor      string                   `json:"cursor,omitempty"`
}

type TaskBoardQueryResult struct {
	Items      []TaskRecord `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
	PageSize   int          `json:"page_size"`
	SortField  string       `json:"sort_field"`
	SortOrder  string       `json:"sort_order"`
}

type normalizedTaskBoardQuery struct {
	TaskID      string
	RunID       string
	WorkflowID  string
	TeamID      string
	State       string
	Priority    string
	AgentID     string
	PeerID      string
	ParentRunID string
	TimeRange   *TaskBoardQueryTimeRange
	PageSize    int
	SortField   string
	SortOrder   string
	Cursor      string
}

type taskBoardQueryCursor struct {
	Offset    int    `json:"offset"`
	QueryHash string `json:"query_hash"`
}

func (s *Scheduler) QueryTasks(ctx context.Context, req TaskBoardQueryRequest) (TaskBoardQueryResult, error) {
	q, err := normalizeTaskBoardQuery(req)
	if err != nil {
		return TaskBoardQueryResult{}, err
	}
	queryHash := taskBoardQueryHash(q)
	start, err := decodeTaskBoardCursor(q.Cursor, queryHash)
	if err != nil {
		return TaskBoardQueryResult{}, err
	}

	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return TaskBoardQueryResult{}, err
	}

	filtered := make([]TaskRecord, 0, len(snapshot.Tasks))
	for i := range snapshot.Tasks {
		if matchesTaskBoardQuery(snapshot.Tasks[i], q) {
			filtered = append(filtered, snapshot.Tasks[i])
		}
	}
	sortTaskBoardQuery(filtered, q.SortField, q.SortOrder)

	if start > len(filtered) {
		return TaskBoardQueryResult{}, fmt.Errorf("invalid query cursor")
	}
	end := start + q.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	items := append([]TaskRecord(nil), filtered[start:end]...)

	nextCursor := ""
	if end < len(filtered) {
		nextCursor, err = encodeTaskBoardCursor(taskBoardQueryCursor{
			Offset:    end,
			QueryHash: queryHash,
		})
		if err != nil {
			return TaskBoardQueryResult{}, err
		}
	}

	return TaskBoardQueryResult{
		Items:      items,
		NextCursor: nextCursor,
		PageSize:   q.PageSize,
		SortField:  q.SortField,
		SortOrder:  q.SortOrder,
	}, nil
}

func normalizeTaskBoardQuery(req TaskBoardQueryRequest) (normalizedTaskBoardQuery, error) {
	pageSize := DefaultTaskBoardQueryPageSize
	if req.PageSize != nil {
		if *req.PageSize <= 0 || *req.PageSize > MaxTaskBoardQueryPageSize {
			return normalizedTaskBoardQuery{}, fmt.Errorf("page_size must be within [1,%d]", MaxTaskBoardQueryPageSize)
		}
		pageSize = *req.PageSize
	}

	sortField := strings.ToLower(strings.TrimSpace(req.Sort.Field))
	if sortField == "" {
		sortField = "updated_at"
	}
	if sortField != "updated_at" && sortField != "created_at" {
		return normalizedTaskBoardQuery{}, fmt.Errorf("unsupported sort.field %q", req.Sort.Field)
	}

	sortOrder := strings.ToLower(strings.TrimSpace(req.Sort.Order))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return normalizedTaskBoardQuery{}, fmt.Errorf("unsupported sort.order %q", req.Sort.Order)
	}

	state := strings.ToLower(strings.TrimSpace(req.State))
	if state != "" {
		switch state {
		case string(TaskStateQueued), string(TaskStateRunning), string(TaskStateSucceeded), string(TaskStateFailed), string(TaskStateDeadLetter):
		default:
			return normalizedTaskBoardQuery{}, fmt.Errorf("unsupported state filter %q", req.State)
		}
	}

	var tr *TaskBoardQueryTimeRange
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
			return normalizedTaskBoardQuery{}, fmt.Errorf("time_range.start must be <= time_range.end")
		}
		tr = &TaskBoardQueryTimeRange{
			Start: start,
			End:   end,
		}
	}

	return normalizedTaskBoardQuery{
		TaskID:      strings.TrimSpace(req.TaskID),
		RunID:       strings.TrimSpace(req.RunID),
		WorkflowID:  strings.TrimSpace(req.WorkflowID),
		TeamID:      strings.TrimSpace(req.TeamID),
		State:       state,
		Priority:    strings.ToLower(strings.TrimSpace(req.Priority)),
		AgentID:     strings.TrimSpace(req.AgentID),
		PeerID:      strings.TrimSpace(req.PeerID),
		ParentRunID: strings.TrimSpace(req.ParentRunID),
		TimeRange:   tr,
		PageSize:    pageSize,
		SortField:   sortField,
		SortOrder:   sortOrder,
		Cursor:      strings.TrimSpace(req.Cursor),
	}, nil
}

func taskBoardQueryHash(q normalizedTaskBoardQuery) string {
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
		q.TaskID,
		q.RunID,
		q.WorkflowID,
		q.TeamID,
		q.State,
		q.Priority,
		q.AgentID,
		q.PeerID,
		q.ParentRunID,
		fmt.Sprintf("%d", start),
		fmt.Sprintf("%d", end),
		q.SortField,
		q.SortOrder,
		fmt.Sprintf("%d", q.PageSize),
	}, "|")
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func encodeTaskBoardCursor(c taskBoardQueryCursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("encode query cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeTaskBoardCursor(cursor, expectedHash string) (int, error) {
	if strings.TrimSpace(cursor) == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return 0, fmt.Errorf("invalid query cursor")
	}
	var decoded taskBoardQueryCursor
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

func matchesTaskBoardQuery(record TaskRecord, q normalizedTaskBoardQuery) bool {
	if q.TaskID != "" && strings.TrimSpace(record.Task.TaskID) != q.TaskID {
		return false
	}
	if q.RunID != "" && strings.TrimSpace(record.Task.RunID) != q.RunID {
		return false
	}
	if q.WorkflowID != "" && strings.TrimSpace(record.Task.WorkflowID) != q.WorkflowID {
		return false
	}
	if q.TeamID != "" && strings.TrimSpace(record.Task.TeamID) != q.TeamID {
		return false
	}
	if q.State != "" && strings.TrimSpace(string(record.State)) != q.State {
		return false
	}
	if q.Priority != "" && normalizedPriority(record.Task.Priority) != q.Priority {
		return false
	}
	if q.AgentID != "" && strings.TrimSpace(record.Task.AgentID) != q.AgentID {
		return false
	}
	if q.PeerID != "" && strings.TrimSpace(record.Task.PeerID) != q.PeerID {
		return false
	}
	if q.ParentRunID != "" && strings.TrimSpace(record.Task.ParentRunID) != q.ParentRunID {
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

func sortTaskBoardQuery(items []TaskRecord, field, order string) {
	desc := strings.TrimSpace(strings.ToLower(order)) != "asc"
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		leftTS := taskBoardSortTimestamp(left, field)
		rightTS := taskBoardSortTimestamp(right, field)
		if leftTS.Equal(rightTS) {
			leftTaskID := strings.TrimSpace(left.Task.TaskID)
			rightTaskID := strings.TrimSpace(right.Task.TaskID)
			if leftTaskID == rightTaskID {
				return strings.TrimSpace(left.Task.RunID) < strings.TrimSpace(right.Task.RunID)
			}
			return leftTaskID < rightTaskID
		}
		if desc {
			return leftTS.After(rightTS)
		}
		return leftTS.Before(rightTS)
	})
}

func taskBoardSortTimestamp(record TaskRecord, field string) time.Time {
	switch strings.TrimSpace(strings.ToLower(field)) {
	case "created_at":
		return record.CreatedAt
	default:
		return record.UpdatedAt
	}
}
