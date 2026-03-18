package scheduler

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type schedulerState struct {
	Tasks           map[string]*TaskRecord    `json:"tasks"`
	Queue           []string                  `json:"queue"`
	TerminalCommits map[string]TerminalCommit `json:"terminal_commits"`
	Stats           Stats                     `json:"stats"`
}

func newSchedulerState(backend string) schedulerState {
	return schedulerState{
		Tasks:           map[string]*TaskRecord{},
		Queue:           []string{},
		TerminalCommits: map[string]TerminalCommit{},
		Stats:           Stats{Backend: strings.TrimSpace(backend)},
	}
}

func (s *schedulerState) enqueue(task Task, now time.Time) (TaskRecord, error) {
	normalized, err := normalizeTask(task)
	if err != nil {
		return TaskRecord{}, err
	}
	if existing, ok := s.Tasks[normalized.TaskID]; ok && existing != nil {
		return cloneTaskRecord(*existing), nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	record := TaskRecord{
		Task:      normalized,
		State:     TaskStateQueued,
		Attempts:  []Attempt{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.Tasks[normalized.TaskID] = &record
	s.Queue = append(s.Queue, normalized.TaskID)
	s.Stats.QueueTotal++
	return cloneTaskRecord(record), nil
}

func (s *schedulerState) claim(workerID string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, bool, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return ClaimedTask{}, false, fmt.Errorf("worker_id is required")
	}
	if leaseTimeout <= 0 {
		return ClaimedTask{}, false, fmt.Errorf("lease_timeout must be > 0")
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.expireLeases(now)

	for len(s.Queue) > 0 {
		taskID := s.Queue[0]
		s.Queue = s.Queue[1:]
		record := s.Tasks[taskID]
		if record == nil {
			continue
		}
		if record.State != TaskStateQueued {
			continue
		}

		nextAttempt := len(record.Attempts) + 1
		attempt := Attempt{
			AttemptID:      fmt.Sprintf("%s-attempt-%d", taskID, nextAttempt),
			Attempt:        nextAttempt,
			WorkerID:       workerID,
			LeaseToken:     fmt.Sprintf("%s-lease-%d", taskID, now.UnixNano()),
			Status:         AttemptStatusRunning,
			StartedAt:      now,
			HeartbeatAt:    now,
			LeaseExpiresAt: now.Add(leaseTimeout),
		}
		record.Attempts = append(record.Attempts, attempt)
		record.CurrentAttempt = attempt.AttemptID
		record.State = TaskStateRunning
		record.UpdatedAt = now
		s.Stats.ClaimTotal++
		return ClaimedTask{Record: cloneTaskRecord(*record), Attempt: attempt}, true, nil
	}
	return ClaimedTask{}, false, nil
}

func (s *schedulerState) heartbeat(taskID, attemptID, leaseToken string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, error) {
	taskID = strings.TrimSpace(taskID)
	attemptID = strings.TrimSpace(attemptID)
	leaseToken = strings.TrimSpace(leaseToken)
	if taskID == "" || attemptID == "" || leaseToken == "" {
		return ClaimedTask{}, fmt.Errorf("task_id/attempt_id/lease_token are required")
	}
	if leaseTimeout <= 0 {
		return ClaimedTask{}, fmt.Errorf("lease_timeout must be > 0")
	}
	record := s.Tasks[taskID]
	if record == nil {
		return ClaimedTask{}, ErrTaskNotFound
	}
	if record.State != TaskStateRunning {
		return ClaimedTask{}, ErrTaskNotRunning
	}
	current, ok := record.currentAttempt()
	if !ok || current.AttemptID != attemptID {
		return ClaimedTask{}, ErrAttemptNotFound
	}
	if current.LeaseToken != leaseToken {
		return ClaimedTask{}, ErrLeaseTokenMismatch
	}
	if now.IsZero() {
		now = time.Now()
	}
	if current.LeaseExpiresAt.Before(now) {
		return ClaimedTask{}, ErrLeaseExpired
	}
	current.HeartbeatAt = now
	current.LeaseExpiresAt = now.Add(leaseTimeout)
	for i := range record.Attempts {
		if record.Attempts[i].AttemptID == current.AttemptID {
			record.Attempts[i] = current
			break
		}
	}
	record.UpdatedAt = now
	return ClaimedTask{Record: cloneTaskRecord(*record), Attempt: current}, nil
}

func (s *schedulerState) expireLeases(now time.Time) []ClaimedTask {
	if now.IsZero() {
		now = time.Now()
	}
	expiredTaskIDs := make([]string, 0)
	for taskID, record := range s.Tasks {
		if record == nil || record.State != TaskStateRunning {
			continue
		}
		current, ok := record.currentAttempt()
		if !ok || current.Status != AttemptStatusRunning {
			continue
		}
		if current.LeaseExpiresAt.After(now) {
			continue
		}
		expiredTaskIDs = append(expiredTaskIDs, taskID)
	}
	sort.Strings(expiredTaskIDs)

	out := make([]ClaimedTask, 0, len(expiredTaskIDs))
	for _, taskID := range expiredTaskIDs {
		record := s.Tasks[taskID]
		if record == nil || record.State != TaskStateRunning {
			continue
		}
		current, ok := record.currentAttempt()
		if !ok {
			continue
		}
		current.Status = AttemptStatusExpired
		current.TerminalAt = now
		for i := range record.Attempts {
			if record.Attempts[i].AttemptID == current.AttemptID {
				record.Attempts[i] = current
				break
			}
		}
		record.CurrentAttempt = ""
		record.State = TaskStateQueued
		record.UpdatedAt = now
		s.Queue = append(s.Queue, taskID)
		s.Stats.LeaseExpiredTotal++
		s.Stats.ReclaimTotal++
		out = append(out, ClaimedTask{Record: cloneTaskRecord(*record), Attempt: current})
	}
	return out
}

func (s *schedulerState) requeue(taskID string, now time.Time) (TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return TaskRecord{}, fmt.Errorf("task_id is required")
	}
	record := s.Tasks[taskID]
	if record == nil {
		return TaskRecord{}, ErrTaskNotFound
	}
	if now.IsZero() {
		now = time.Now()
	}
	if record.State != TaskStateRunning {
		return TaskRecord{}, ErrTaskNotRunning
	}
	current, ok := record.currentAttempt()
	if !ok {
		return TaskRecord{}, ErrAttemptNotFound
	}
	current.Status = AttemptStatusExpired
	current.TerminalAt = now
	for i := range record.Attempts {
		if record.Attempts[i].AttemptID == current.AttemptID {
			record.Attempts[i] = current
			break
		}
	}
	record.CurrentAttempt = ""
	record.State = TaskStateQueued
	record.UpdatedAt = now
	s.Queue = append(s.Queue, taskID)
	s.Stats.ReclaimTotal++
	return cloneTaskRecord(*record), nil
}

func (s *schedulerState) commitTerminal(commit TerminalCommit) (CommitResult, error) {
	normalized, err := normalizeCommit(commit)
	if err != nil {
		return CommitResult{}, err
	}
	record := s.Tasks[normalized.TaskID]
	if record == nil {
		return CommitResult{}, ErrTaskNotFound
	}
	key := terminalCommitKey(normalized.TaskID, normalized.AttemptID)
	if existing, ok := s.TerminalCommits[key]; ok {
		_ = existing
		s.Stats.DuplicateTerminalCommitTotal++
		return CommitResult{Record: cloneTaskRecord(*record), Duplicate: true}, nil
	}
	if record.State != TaskStateRunning {
		return CommitResult{}, ErrTaskNotRunning
	}
	current, ok := record.currentAttempt()
	if !ok {
		return CommitResult{}, ErrAttemptNotFound
	}
	if current.AttemptID != normalized.AttemptID {
		return CommitResult{}, ErrStaleAttempt
	}

	switch normalized.Status {
	case TaskStateSucceeded:
		current.Status = AttemptStatusSucceeded
	case TaskStateFailed:
		current.Status = AttemptStatusFailed
	default:
		return CommitResult{}, fmt.Errorf("unsupported terminal status %q", normalized.Status)
	}
	current.TerminalAt = normalized.CommittedAt
	for i := range record.Attempts {
		if record.Attempts[i].AttemptID == current.AttemptID {
			record.Attempts[i] = current
			break
		}
	}
	record.CurrentAttempt = ""
	record.State = normalized.Status
	record.UpdatedAt = normalized.CommittedAt
	record.ErrorMessage = normalized.ErrorMessage
	record.ErrorClass = normalized.ErrorClass
	record.ErrorLayer = normalized.ErrorLayer
	if normalized.Status == TaskStateSucceeded {
		record.Result = copyMap(normalized.Result)
	} else {
		record.Result = map[string]any{}
	}
	s.TerminalCommits[key] = normalized
	if normalized.Status == TaskStateSucceeded {
		s.Stats.CompleteTotal++
	} else {
		s.Stats.FailTotal++
	}
	return CommitResult{Record: cloneTaskRecord(*record)}, nil
}

func (s *schedulerState) get(taskID string) (TaskRecord, bool) {
	record := s.Tasks[strings.TrimSpace(taskID)]
	if record == nil {
		return TaskRecord{}, false
	}
	return cloneTaskRecord(*record), true
}

func cloneTaskRecord(in TaskRecord) TaskRecord {
	out := in
	out.Task = in.Task
	out.Task.Payload = copyMap(in.Task.Payload)
	out.Attempts = append([]Attempt(nil), in.Attempts...)
	out.Result = copyMap(in.Result)
	return out
}

func terminalCommitKey(taskID, attemptID string) string {
	return strings.TrimSpace(taskID) + "|" + strings.TrimSpace(attemptID)
}
