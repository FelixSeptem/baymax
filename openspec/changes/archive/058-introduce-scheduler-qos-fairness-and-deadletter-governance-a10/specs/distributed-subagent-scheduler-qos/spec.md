## ADDED Requirements

### Requirement: Scheduler SHALL support explicit QoS mode with FIFO default
Scheduler MUST expose QoS mode controls and default to `fifo`; priority scheduling MUST only activate when explicitly enabled.

#### Scenario: Runtime starts without QoS override
- **WHEN** scheduler loads default configuration
- **THEN** claim order follows FIFO semantics

### Requirement: Priority scheduling SHALL use task-level priority fields
When priority mode is enabled, scheduler MUST derive priority from task-level fields and MUST NOT require host callback for priority resolution.

#### Scenario: Task batch contains mixed priorities
- **WHEN** priority mode is enabled and tasks have high/normal/low priority fields
- **THEN** scheduler claim order follows configured priority precedence

### Requirement: Scheduler SHALL enforce fairness window in priority mode
Scheduler priority mode MUST enforce `max_consecutive_claims_per_priority` and yield to lower-priority claimable tasks after threshold is reached.

#### Scenario: High-priority queue remains continuously non-empty
- **WHEN** scheduler reaches fairness threshold for high-priority claims
- **THEN** scheduler yields claim opportunities to other priority levels when claimable tasks exist

### Requirement: Scheduler SHALL support dead-letter transfer with default disabled
Scheduler MUST support dead-letter transfer for retry-exhausted tasks, and dead-letter behavior MUST be disabled by default.

#### Scenario: Task exhausts retry budget with DLQ enabled
- **WHEN** a task exceeds retry limits under enabled dead-letter policy
- **THEN** scheduler moves task to dead-letter state and stops normal retry/requeue flow

### Requirement: Retry governance SHALL use exponential backoff with jitter
Scheduler retry governance MUST support exponential backoff and bounded jitter for retry scheduling.

#### Scenario: Consecutive retry failures occur
- **WHEN** same task attempt repeatedly fails and remains retryable
- **THEN** retry delays grow exponentially and include bounded jitter before next eligible claim
