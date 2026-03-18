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

func (s *schedulerState) snapshot() StoreSnapshot {
	if s == nil {
		return StoreSnapshot{}
	}
	taskIDs := make([]string, 0, len(s.Tasks))
	for taskID := range s.Tasks {
		taskIDs = append(taskIDs, taskID)
	}
	sort.Strings(taskIDs)
	tasks := make([]TaskRecord, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		record := s.Tasks[taskID]
		if record == nil {
			continue
		}
		tasks = append(tasks, cloneTaskRecord(*record))
	}

	commitKeys := make([]string, 0, len(s.TerminalCommits))
	for key := range s.TerminalCommits {
		commitKeys = append(commitKeys, key)
	}
	sort.Strings(commitKeys)
	commits := make([]TerminalCommit, 0, len(commitKeys))
	for _, key := range commitKeys {
		commit := s.TerminalCommits[key]
		normalized := commit
		normalized.TaskID = strings.TrimSpace(normalized.TaskID)
		normalized.AttemptID = strings.TrimSpace(normalized.AttemptID)
		normalized.ErrorMessage = strings.TrimSpace(normalized.ErrorMessage)
		normalized.ErrorLayer = strings.TrimSpace(normalized.ErrorLayer)
		normalized.OutcomeKey = strings.TrimSpace(normalized.OutcomeKey)
		normalized.Result = copyMap(normalized.Result)
		commits = append(commits, normalized)
	}

	return StoreSnapshot{
		Backend:         strings.TrimSpace(s.Stats.Backend),
		Tasks:           tasks,
		Queue:           append([]string(nil), s.Queue...),
		TerminalCommits: commits,
		Stats:           s.Stats,
	}
}

func (s *schedulerState) restore(snapshot StoreSnapshot) error {
	if s == nil {
		return fmt.Errorf("%w: scheduler state is nil", ErrSnapshotCorrupt)
	}
	tasks := make(map[string]*TaskRecord, len(snapshot.Tasks))
	for i := range snapshot.Tasks {
		record := cloneTaskRecord(snapshot.Tasks[i])
		taskID := strings.TrimSpace(record.Task.TaskID)
		if taskID == "" {
			return fmt.Errorf("%w: tasks[%d].task.task_id is required", ErrSnapshotCorrupt, i)
		}
		if _, exists := tasks[taskID]; exists {
			return fmt.Errorf("%w: duplicate task_id %q", ErrSnapshotCorrupt, taskID)
		}
		record.Task.TaskID = taskID
		record.Task.RunID = strings.TrimSpace(record.Task.RunID)
		record.Task.WorkflowID = strings.TrimSpace(record.Task.WorkflowID)
		record.Task.TeamID = strings.TrimSpace(record.Task.TeamID)
		record.Task.StepID = strings.TrimSpace(record.Task.StepID)
		record.Task.AgentID = strings.TrimSpace(record.Task.AgentID)
		record.Task.PeerID = strings.TrimSpace(record.Task.PeerID)
		record.Task.ParentRunID = strings.TrimSpace(record.Task.ParentRunID)

		currentAttemptID := strings.TrimSpace(record.CurrentAttempt)
		record.CurrentAttempt = currentAttemptID
		attemptSeen := map[string]struct{}{}
		currentExists := false
		for idx := range record.Attempts {
			attemptID := strings.TrimSpace(record.Attempts[idx].AttemptID)
			if attemptID == "" {
				return fmt.Errorf("%w: tasks[%d].attempts[%d].attempt_id is required", ErrSnapshotCorrupt, i, idx)
			}
			if _, exists := attemptSeen[attemptID]; exists {
				return fmt.Errorf("%w: duplicate attempt_id %q for task %q", ErrSnapshotCorrupt, attemptID, taskID)
			}
			attemptSeen[attemptID] = struct{}{}
			record.Attempts[idx].AttemptID = attemptID
			record.Attempts[idx].WorkerID = strings.TrimSpace(record.Attempts[idx].WorkerID)
			record.Attempts[idx].LeaseToken = strings.TrimSpace(record.Attempts[idx].LeaseToken)
			if attemptID == currentAttemptID {
				currentExists = true
			}
		}
		switch record.State {
		case TaskStateQueued:
			if currentAttemptID != "" {
				return fmt.Errorf("%w: queued task %q must not have current_attempt_id", ErrSnapshotCorrupt, taskID)
			}
		case TaskStateRunning:
			if currentAttemptID == "" || !currentExists {
				return fmt.Errorf("%w: running task %q requires current_attempt_id", ErrSnapshotCorrupt, taskID)
			}
		case TaskStateSucceeded, TaskStateFailed:
			if currentAttemptID != "" {
				return fmt.Errorf("%w: terminal task %q must not have current_attempt_id", ErrSnapshotCorrupt, taskID)
			}
		default:
			return fmt.Errorf("%w: unsupported task state %q for task %q", ErrSnapshotCorrupt, record.State, taskID)
		}
		record.ErrorMessage = strings.TrimSpace(record.ErrorMessage)
		record.ErrorLayer = strings.TrimSpace(record.ErrorLayer)
		record.Result = copyMap(record.Result)
		tasks[taskID] = &record
	}

	queue := make([]string, 0, len(snapshot.Queue))
	queueSeen := map[string]struct{}{}
	for i := range snapshot.Queue {
		taskID := strings.TrimSpace(snapshot.Queue[i])
		if taskID == "" {
			continue
		}
		record := tasks[taskID]
		if record == nil {
			return fmt.Errorf("%w: queue references unknown task %q", ErrSnapshotCorrupt, taskID)
		}
		if record.State != TaskStateQueued {
			return fmt.Errorf("%w: queue task %q must be queued", ErrSnapshotCorrupt, taskID)
		}
		if _, exists := queueSeen[taskID]; exists {
			return fmt.Errorf("%w: duplicate queue task_id %q", ErrSnapshotCorrupt, taskID)
		}
		queueSeen[taskID] = struct{}{}
		queue = append(queue, taskID)
	}

	terminalCommits := make(map[string]TerminalCommit, len(snapshot.TerminalCommits))
	for i := range snapshot.TerminalCommits {
		commit := snapshot.TerminalCommits[i]
		taskID := strings.TrimSpace(commit.TaskID)
		attemptID := strings.TrimSpace(commit.AttemptID)
		if taskID == "" || attemptID == "" {
			return fmt.Errorf("%w: terminal_commits[%d] requires task_id and attempt_id", ErrSnapshotCorrupt, i)
		}
		switch commit.Status {
		case TaskStateSucceeded, TaskStateFailed:
		default:
			return fmt.Errorf("%w: terminal_commits[%d] has unsupported status %q", ErrSnapshotCorrupt, i, commit.Status)
		}
		key := terminalCommitKey(taskID, attemptID)
		if _, exists := terminalCommits[key]; exists {
			return fmt.Errorf("%w: duplicate terminal commit %q", ErrSnapshotCorrupt, key)
		}
		normalized := commit
		normalized.TaskID = taskID
		normalized.AttemptID = attemptID
		normalized.ErrorMessage = strings.TrimSpace(normalized.ErrorMessage)
		normalized.ErrorLayer = strings.TrimSpace(normalized.ErrorLayer)
		normalized.OutcomeKey = strings.TrimSpace(normalized.OutcomeKey)
		normalized.Result = copyMap(normalized.Result)
		terminalCommits[key] = normalized
	}

	backend := strings.TrimSpace(snapshot.Stats.Backend)
	if backend == "" {
		backend = strings.TrimSpace(snapshot.Backend)
	}
	if backend == "" {
		backend = strings.TrimSpace(s.Stats.Backend)
	}
	if backend == "" {
		backend = "memory"
	}
	stats := snapshot.Stats
	stats.Backend = backend

	s.Tasks = tasks
	s.Queue = queue
	s.TerminalCommits = terminalCommits
	s.Stats = stats
	return nil
}
