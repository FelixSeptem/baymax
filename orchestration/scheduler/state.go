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
	DelayedWaitMs   []int64                   `json:"delayed_wait_ms,omitempty"`
	Stats           Stats                     `json:"stats"`
	governance      GovernanceConfig
	lastPriority    string
	lastConsecutive int
}

func newSchedulerState(backend string) schedulerState {
	cfg := defaultGovernanceConfig()
	return schedulerState{
		Tasks:           map[string]*TaskRecord{},
		Queue:           []string{},
		TerminalCommits: map[string]TerminalCommit{},
		DelayedWaitMs:   []int64{},
		Stats: Stats{
			Backend: strings.TrimSpace(backend),
			QoSMode: string(cfg.QoS),
		},
		governance: cfg,
	}
}

func defaultGovernanceConfig() GovernanceConfig {
	return GovernanceConfig{
		QoS: QoSModeFIFO,
		Fairness: FairnessConfig{
			MaxConsecutiveClaimsPerPriority: 3,
		},
		DLQ: DLQConfig{
			Enabled: false,
		},
		Backoff: RetryBackoffConfig{
			Enabled:     false,
			Initial:     50 * time.Millisecond,
			Max:         2 * time.Second,
			Multiplier:  2.0,
			JitterRatio: 0.2,
		},
	}
}

func normalizeGovernanceConfig(cfg GovernanceConfig) GovernanceConfig {
	out := cfg
	switch out.QoS {
	case QoSModePriority:
	default:
		out.QoS = QoSModeFIFO
	}
	if out.Fairness.MaxConsecutiveClaimsPerPriority <= 0 {
		out.Fairness.MaxConsecutiveClaimsPerPriority = 3
	}
	if out.Backoff.Initial <= 0 {
		out.Backoff.Initial = 50 * time.Millisecond
	}
	if out.Backoff.Max <= 0 || out.Backoff.Max < out.Backoff.Initial {
		out.Backoff.Max = 2 * time.Second
		if out.Backoff.Max < out.Backoff.Initial {
			out.Backoff.Max = out.Backoff.Initial
		}
	}
	if out.Backoff.Multiplier < 1 {
		out.Backoff.Multiplier = 1
	}
	if out.Backoff.JitterRatio < 0 {
		out.Backoff.JitterRatio = 0
	}
	if out.Backoff.JitterRatio > 1 {
		out.Backoff.JitterRatio = 1
	}
	return out
}

func (s *schedulerState) setGovernance(cfg GovernanceConfig) {
	if s == nil {
		return
	}
	s.governance = normalizeGovernanceConfig(cfg)
	s.Stats.QoSMode = string(s.governance.QoS)
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
	if isTaskDelayed(normalized, now) {
		s.Stats.DelayedTaskTotal++
	}
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

	index, yielded := s.nextClaimableIndex(now)
	if index < 0 {
		return ClaimedTask{}, false, nil
	}
	taskID := s.Queue[index]
	s.Queue = append(s.Queue[:index], s.Queue[index+1:]...)
	record := s.Tasks[taskID]
	if record == nil || record.State != TaskStateQueued {
		return ClaimedTask{}, false, nil
	}
	delayedTask := isTaskDelayed(record.Task, record.CreatedAt)
	delayedWaitMs := delayedWaitDurationMs(record, now)

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
	record.NextEligibleAt = time.Time{}
	record.State = TaskStateRunning
	record.UpdatedAt = now

	priority := normalizedPriority(record.Task.Priority)
	if s.governance.QoS == QoSModePriority {
		s.Stats.PriorityClaimTotal++
	}
	if yielded {
		s.Stats.FairnessYieldTotal++
	}
	if priority == s.lastPriority {
		s.lastConsecutive++
	} else {
		s.lastPriority = priority
		s.lastConsecutive = 1
	}
	s.Stats.ClaimTotal++
	if delayedTask {
		s.Stats.DelayedClaimTotal++
		s.DelayedWaitMs = append(s.DelayedWaitMs, delayedWaitMs)
		s.Stats.DelayedWaitMsP95 = percentileP95Int64(s.DelayedWaitMs)
	}
	s.Stats.QoSMode = string(s.governance.QoS)
	return ClaimedTask{
		Record:          cloneTaskRecord(*record),
		Attempt:         attempt,
		TaskPriority:    priority,
		FairnessYielded: yielded,
	}, true, nil
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
		s.handleRetryTransition(taskID, record, current, now)
		s.Stats.LeaseExpiredTotal++
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
	s.handleRetryTransition(taskID, record, current, now)
	return cloneTaskRecord(*record), nil
}

func (s *schedulerState) nextClaimableIndex(now time.Time) (int, bool) {
	firstEligible := -1
	byPriority := map[string]int{}
	for idx := range s.Queue {
		taskID := strings.TrimSpace(s.Queue[idx])
		record := s.Tasks[taskID]
		if !isClaimableRecord(record, now) {
			continue
		}
		if firstEligible < 0 {
			firstEligible = idx
		}
		priority := normalizedPriority(record.Task.Priority)
		if _, exists := byPriority[priority]; !exists {
			byPriority[priority] = idx
		}
	}
	if firstEligible < 0 {
		return -1, false
	}
	if s.governance.QoS != QoSModePriority {
		return firstEligible, false
	}

	priorityOrder := []string{TaskPriorityHigh, TaskPriorityNormal, TaskPriorityLow}
	selectedPriority := ""
	selectedIndex := -1
	for _, priority := range priorityOrder {
		if idx, ok := byPriority[priority]; ok {
			selectedPriority = priority
			selectedIndex = idx
			break
		}
	}
	if selectedIndex < 0 {
		return firstEligible, false
	}
	threshold := s.governance.Fairness.MaxConsecutiveClaimsPerPriority
	if threshold > 0 && s.lastPriority == selectedPriority && s.lastConsecutive >= threshold {
		for _, priority := range priorityOrder {
			if priority == selectedPriority {
				continue
			}
			if idx, ok := byPriority[priority]; ok {
				return idx, true
			}
		}
	}
	return selectedIndex, false
}

func isClaimableRecord(record *TaskRecord, now time.Time) bool {
	if record == nil || record.State != TaskStateQueued {
		return false
	}
	if !record.Task.NotBefore.IsZero() && record.Task.NotBefore.After(now) {
		return false
	}
	if record.NextEligibleAt.IsZero() {
		return true
	}
	return !record.NextEligibleAt.After(now)
}

func normalizedPriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case TaskPriorityHigh:
		return TaskPriorityHigh
	case TaskPriorityLow:
		return TaskPriorityLow
	default:
		return TaskPriorityNormal
	}
}

func (s *schedulerState) handleRetryTransition(taskID string, record *TaskRecord, current Attempt, now time.Time) {
	current.Status = AttemptStatusExpired
	current.TerminalAt = now
	for i := range record.Attempts {
		if record.Attempts[i].AttemptID == current.AttemptID {
			record.Attempts[i] = current
			break
		}
	}
	record.CurrentAttempt = ""
	record.UpdatedAt = now

	maxAttempts := record.Task.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if len(record.Attempts) >= maxAttempts {
		record.NextEligibleAt = time.Time{}
		if s.governance.DLQ.Enabled {
			record.State = TaskStateDeadLetter
			record.DeadLetterCode = "retry_exhausted"
			record.ErrorMessage = "retry attempts exhausted"
			s.Stats.DeadLetterTotal++
			return
		}
		record.State = TaskStateFailed
		record.ErrorMessage = "retry attempts exhausted"
		s.Stats.FailTotal++
		return
	}

	record.State = TaskStateQueued
	record.DeadLetterCode = ""
	backoff := s.retryDelay(taskID, current)
	record.NextEligibleAt = now.Add(backoff)
	s.Queue = append(s.Queue, taskID)
	s.Stats.ReclaimTotal++
	if backoff > 0 {
		s.Stats.RetryBackoffTotal++
	}
}

func (s *schedulerState) retryDelay(taskID string, current Attempt) time.Duration {
	if !s.governance.Backoff.Enabled {
		return 0
	}
	base := float64(s.governance.Backoff.Initial)
	if base <= 0 {
		base = float64(50 * time.Millisecond)
	}
	multiplier := s.governance.Backoff.Multiplier
	if multiplier < 1 {
		multiplier = 1
	}
	attemptNum := current.Attempt
	if attemptNum <= 0 {
		attemptNum = 1
	}
	delayFloat := base
	for i := 1; i < attemptNum; i++ {
		delayFloat *= multiplier
	}
	maxDelay := float64(s.governance.Backoff.Max)
	if maxDelay <= 0 {
		maxDelay = float64(2 * time.Second)
	}
	if delayFloat > maxDelay {
		delayFloat = maxDelay
	}
	delay := time.Duration(delayFloat)
	jitterRatio := s.governance.Backoff.JitterRatio
	if jitterRatio <= 0 {
		return delay
	}
	jitterRange := int64(float64(delay) * jitterRatio)
	if jitterRange <= 0 {
		return delay
	}
	seed := stableRetryJitterSeed(taskID, current.AttemptID, current.Attempt)
	jitter := (seed % (2*jitterRange + 1)) - jitterRange
	withJitter := int64(delay) + jitter
	if withJitter < 0 {
		withJitter = 0
	}
	if withJitter > int64(s.governance.Backoff.Max) {
		withJitter = int64(s.governance.Backoff.Max)
	}
	return time.Duration(withJitter)
}

func stableRetryJitterSeed(taskID, attemptID string, attempt int) int64 {
	raw := strings.TrimSpace(taskID) + "|" + strings.TrimSpace(attemptID) + "|" + fmt.Sprintf("%d", attempt)
	var h uint64 = 1469598103934665603
	const prime uint64 = 1099511628211
	for i := 0; i < len(raw); i++ {
		h ^= uint64(raw[i])
		h *= prime
	}
	return int64(h & 0x7fffffffffffffff)
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
	record.NextEligibleAt = time.Time{}
	record.DeadLetterCode = ""
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
		DelayedWaitMs:   append([]int64(nil), s.DelayedWaitMs...),
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
		normalizedTask, err := normalizeTask(record.Task)
		if err != nil {
			return fmt.Errorf("%w: tasks[%d].task is invalid: %v", ErrSnapshotCorrupt, i, err)
		}
		record.Task = normalizedTask

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
		case TaskStateSucceeded, TaskStateFailed, TaskStateDeadLetter:
			if currentAttemptID != "" {
				return fmt.Errorf("%w: terminal task %q must not have current_attempt_id", ErrSnapshotCorrupt, taskID)
			}
		default:
			return fmt.Errorf("%w: unsupported task state %q for task %q", ErrSnapshotCorrupt, record.State, taskID)
		}
		record.ErrorMessage = strings.TrimSpace(record.ErrorMessage)
		record.ErrorLayer = strings.TrimSpace(record.ErrorLayer)
		record.DeadLetterCode = strings.TrimSpace(record.DeadLetterCode)
		if record.State != TaskStateQueued {
			record.NextEligibleAt = time.Time{}
		}
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
	if strings.TrimSpace(stats.QoSMode) == "" {
		stats.QoSMode = string(s.governance.QoS)
	}
	delayedWaitMs := append([]int64(nil), snapshot.DelayedWaitMs...)
	if len(delayedWaitMs) > 0 {
		stats.DelayedWaitMsP95 = percentileP95Int64(delayedWaitMs)
	}

	s.Tasks = tasks
	s.Queue = queue
	s.TerminalCommits = terminalCommits
	s.DelayedWaitMs = delayedWaitMs
	s.Stats = stats
	s.lastPriority = ""
	s.lastConsecutive = 0
	return nil
}

func isTaskDelayed(task Task, createdAt time.Time) bool {
	if task.NotBefore.IsZero() {
		return false
	}
	if createdAt.IsZero() {
		return true
	}
	return task.NotBefore.After(createdAt)
}

func delayedWaitDurationMs(record *TaskRecord, now time.Time) int64 {
	if record == nil || now.IsZero() {
		return 0
	}
	wait := now.Sub(record.CreatedAt).Milliseconds()
	if wait < 0 {
		return 0
	}
	return wait
}

func percentileP95Int64(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	index := int(float64(len(cp))*0.95 + 0.9999999)
	if index <= 0 {
		index = 1
	}
	if index > len(cp) {
		index = len(cp)
	}
	return cp[index-1]
}
