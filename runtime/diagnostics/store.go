package diagnostics

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/runtime/security/redaction"
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
	Time                                 time.Time                         `json:"time"`
	RunID                                string                            `json:"run_id"`
	Status                               string                            `json:"status,omitempty"`
	Iterations                           int                               `json:"iterations"`
	ToolCalls                            int                               `json:"tool_calls"`
	LatencyMs                            int64                             `json:"latency_ms"`
	ErrorClass                           string                            `json:"error_class,omitempty"`
	PolicyKind                           string                            `json:"policy_kind,omitempty"`
	NamespaceTool                        string                            `json:"namespace_tool,omitempty"`
	FilterStage                          string                            `json:"filter_stage,omitempty"`
	Decision                             string                            `json:"decision,omitempty"`
	ReasonCode                           string                            `json:"reason_code,omitempty"`
	Severity                             string                            `json:"severity,omitempty"`
	AlertDispatchStatus                  string                            `json:"alert_dispatch_status,omitempty"`
	AlertDispatchFailureReason           string                            `json:"alert_dispatch_failure_reason,omitempty"`
	AlertDeliveryMode                    string                            `json:"alert_delivery_mode,omitempty"`
	AlertRetryCount                      int                               `json:"alert_retry_count,omitempty"`
	AlertQueueDropped                    bool                              `json:"alert_queue_dropped,omitempty"`
	AlertQueueDropCount                  int                               `json:"alert_queue_drop_count,omitempty"`
	AlertCircuitState                    string                            `json:"alert_circuit_state,omitempty"`
	AlertCircuitOpenReason               string                            `json:"alert_circuit_open_reason,omitempty"`
	ModelProvider                        string                            `json:"model_provider,omitempty"`
	FallbackUsed                         bool                              `json:"fallback_used,omitempty"`
	FallbackInitial                      string                            `json:"fallback_initial,omitempty"`
	FallbackPath                         string                            `json:"fallback_path,omitempty"`
	RequiredCapabilities                 string                            `json:"required_capabilities,omitempty"`
	FallbackReason                       string                            `json:"fallback_reason,omitempty"`
	PrefixHash                           string                            `json:"prefix_hash,omitempty"`
	AssembleLatencyMs                    int64                             `json:"assemble_latency_ms,omitempty"`
	AssembleStatus                       string                            `json:"assemble_status,omitempty"`
	GuardViolation                       string                            `json:"guard_violation,omitempty"`
	AssembleStageStatus                  string                            `json:"assemble_stage_status,omitempty"`
	Stage2SkipReason                     string                            `json:"stage2_skip_reason,omitempty"`
	Stage2RouterMode                     string                            `json:"stage2_router_mode,omitempty"`
	Stage2RouterDecision                 string                            `json:"stage2_router_decision,omitempty"`
	Stage2RouterReason                   string                            `json:"stage2_router_reason,omitempty"`
	Stage2RouterLatencyMs                int64                             `json:"stage2_router_latency_ms,omitempty"`
	Stage2RouterError                    string                            `json:"stage2_router_error,omitempty"`
	Stage1LatencyMs                      int64                             `json:"stage1_latency_ms,omitempty"`
	Stage2LatencyMs                      int64                             `json:"stage2_latency_ms,omitempty"`
	Stage2Provider                       string                            `json:"stage2_provider,omitempty"`
	Stage2Profile                        string                            `json:"stage2_profile,omitempty"`
	Stage2TemplateProfile                string                            `json:"stage2_template_profile,omitempty"`
	Stage2TemplateResolutionSource       string                            `json:"stage2_template_resolution_source,omitempty"`
	Stage2HintApplied                    bool                              `json:"stage2_hint_applied,omitempty"`
	Stage2HintMismatchReason             string                            `json:"stage2_hint_mismatch_reason,omitempty"`
	Stage2HitCount                       int                               `json:"stage2_hit_count,omitempty"`
	Stage2Source                         string                            `json:"stage2_source,omitempty"`
	Stage2Reason                         string                            `json:"stage2_reason,omitempty"`
	Stage2ReasonCode                     string                            `json:"stage2_reason_code,omitempty"`
	Stage2ErrorLayer                     string                            `json:"stage2_error_layer,omitempty"`
	CA3PressureZone                      string                            `json:"ca3_pressure_zone,omitempty"`
	CA3PressureReason                    string                            `json:"ca3_pressure_reason,omitempty"`
	CA3PressureTrigger                   string                            `json:"ca3_pressure_trigger,omitempty"`
	CA3ZoneResidencyMs                   map[string]int64                  `json:"ca3_zone_residency_ms,omitempty"`
	CA3TriggerCounts                     map[string]int                    `json:"ca3_trigger_counts,omitempty"`
	CA3CompressionRatio                  float64                           `json:"ca3_compression_ratio,omitempty"`
	CA3SpillCount                        int                               `json:"ca3_spill_count,omitempty"`
	CA3SwapBackCount                     int                               `json:"ca3_swap_back_count,omitempty"`
	CA3CompactionMode                    string                            `json:"ca3_compaction_mode,omitempty"`
	CA3CompactionFallback                bool                              `json:"ca3_compaction_fallback,omitempty"`
	CA3CompactionFallbackReason          string                            `json:"ca3_compaction_fallback_reason,omitempty"`
	CA3CompactionQualityScore            float64                           `json:"ca3_compaction_quality_score,omitempty"`
	CA3CompactionQualityReason           string                            `json:"ca3_compaction_quality_reason,omitempty"`
	CA3CompactionEmbeddingProvider       string                            `json:"ca3_compaction_embedding_provider,omitempty"`
	CA3CompactionEmbeddingSimilarity     float64                           `json:"ca3_compaction_embedding_similarity,omitempty"`
	CA3CompactionEmbeddingContribution   float64                           `json:"ca3_compaction_embedding_contribution,omitempty"`
	CA3CompactionEmbeddingStatus         string                            `json:"ca3_compaction_embedding_status,omitempty"`
	CA3CompactionEmbeddingFallbackReason string                            `json:"ca3_compaction_embedding_fallback_reason,omitempty"`
	CA3CompactionRerankerUsed            bool                              `json:"ca3_compaction_reranker_used,omitempty"`
	CA3CompactionRerankerProvider        string                            `json:"ca3_compaction_reranker_provider,omitempty"`
	CA3CompactionRerankerModel           string                            `json:"ca3_compaction_reranker_model,omitempty"`
	CA3CompactionRerankerThresholdSource string                            `json:"ca3_compaction_reranker_threshold_source,omitempty"`
	CA3CompactionRerankerThresholdHit    bool                              `json:"ca3_compaction_reranker_threshold_hit,omitempty"`
	CA3CompactionRerankerFallbackReason  string                            `json:"ca3_compaction_reranker_fallback_reason,omitempty"`
	CA3CompactionRerankerProfileVersion  string                            `json:"ca3_compaction_reranker_profile_version,omitempty"`
	CA3CompactionRerankerRolloutHit      bool                              `json:"ca3_compaction_reranker_rollout_hit,omitempty"`
	CA3CompactionRerankerThresholdDrift  float64                           `json:"ca3_compaction_reranker_threshold_drift,omitempty"`
	CA3RetainedEvidence                  int                               `json:"ca3_compaction_retained_evidence_count,omitempty"`
	RecapStatus                          string                            `json:"recap_status,omitempty"`
	TeamID                               string                            `json:"team_id,omitempty"`
	TeamStrategy                         string                            `json:"team_strategy,omitempty"`
	TeamTaskTotal                        int                               `json:"team_task_total,omitempty"`
	TeamTaskFailed                       int                               `json:"team_task_failed,omitempty"`
	TeamTaskCanceled                     int                               `json:"team_task_canceled,omitempty"`
	TeamRemoteTaskTotal                  int                               `json:"team_remote_task_total,omitempty"`
	TeamRemoteTaskFailed                 int                               `json:"team_remote_task_failed,omitempty"`
	WorkflowID                           string                            `json:"workflow_id,omitempty"`
	WorkflowStatus                       string                            `json:"workflow_status,omitempty"`
	WorkflowStepTotal                    int                               `json:"workflow_step_total,omitempty"`
	WorkflowStepFailed                   int                               `json:"workflow_step_failed,omitempty"`
	WorkflowRemoteStepTotal              int                               `json:"workflow_remote_step_total,omitempty"`
	WorkflowRemoteStepFailed             int                               `json:"workflow_remote_step_failed,omitempty"`
	WorkflowResumeCount                  int                               `json:"workflow_resume_count,omitempty"`
	A2ATaskTotal                         int                               `json:"a2a_task_total,omitempty"`
	A2ATaskFailed                        int                               `json:"a2a_task_failed,omitempty"`
	PeerID                               string                            `json:"peer_id,omitempty"`
	A2AErrorLayer                        string                            `json:"a2a_error_layer,omitempty"`
	A2ADeliveryMode                      string                            `json:"a2a_delivery_mode,omitempty"`
	A2ADeliveryFallbackUsed              bool                              `json:"a2a_delivery_fallback_used,omitempty"`
	A2ADeliveryFallbackReason            string                            `json:"a2a_delivery_fallback_reason,omitempty"`
	A2AVersionLocal                      string                            `json:"a2a_version_local,omitempty"`
	A2AVersionPeer                       string                            `json:"a2a_version_peer,omitempty"`
	A2AVersionNegotiationResult          string                            `json:"a2a_version_negotiation_result,omitempty"`
	ComposerManaged                      bool                              `json:"composer_managed,omitempty"`
	SchedulerBackend                     string                            `json:"scheduler_backend,omitempty"`
	SchedulerBackendFallback             bool                              `json:"scheduler_backend_fallback,omitempty"`
	SchedulerBackendFallbackReason       string                            `json:"scheduler_backend_fallback_reason,omitempty"`
	SchedulerQueueTotal                  int                               `json:"scheduler_queue_total,omitempty"`
	SchedulerClaimTotal                  int                               `json:"scheduler_claim_total,omitempty"`
	SchedulerReclaimTotal                int                               `json:"scheduler_reclaim_total,omitempty"`
	SubagentChildTotal                   int                               `json:"subagent_child_total,omitempty"`
	SubagentChildFailed                  int                               `json:"subagent_child_failed,omitempty"`
	SubagentBudgetRejectTotal            int                               `json:"subagent_budget_reject_total,omitempty"`
	RecoveryEnabled                      bool                              `json:"recovery_enabled,omitempty"`
	RecoveryRecovered                    bool                              `json:"recovery_recovered,omitempty"`
	RecoveryReplayTotal                  int                               `json:"recovery_replay_total,omitempty"`
	RecoveryConflict                     bool                              `json:"recovery_conflict,omitempty"`
	RecoveryConflictCode                 string                            `json:"recovery_conflict_code,omitempty"`
	RecoveryFallbackUsed                 bool                              `json:"recovery_fallback_used,omitempty"`
	RecoveryFallbackReason               string                            `json:"recovery_fallback_reason,omitempty"`
	GateChecks                           int                               `json:"gate_checks,omitempty"`
	GateDeniedCount                      int                               `json:"gate_denied_count,omitempty"`
	GateTimeoutCount                     int                               `json:"gate_timeout_count,omitempty"`
	GateRuleHitCount                     int                               `json:"gate_rule_hit_count,omitempty"`
	GateRuleLastID                       string                            `json:"gate_rule_last_id,omitempty"`
	AwaitCount                           int                               `json:"await_count,omitempty"`
	ResumeCount                          int                               `json:"resume_count,omitempty"`
	CancelByUserCount                    int                               `json:"cancel_by_user_count,omitempty"`
	CancelPropagated                     int                               `json:"cancel_propagated_count,omitempty"`
	BackpressureDrop                     int                               `json:"backpressure_drop_count,omitempty"`
	BackpressureDropByPhase              map[string]int                    `json:"backpressure_drop_count_by_phase,omitempty"`
	InflightPeak                         int                               `json:"inflight_peak,omitempty"`
	TimelinePhases                       map[string]TimelinePhaseAggregate `json:"timeline_phases,omitempty"`
}

type TimelinePhaseAggregate struct {
	CountTotal    int   `json:"count_total,omitempty"`
	FailedTotal   int   `json:"failed_total,omitempty"`
	CanceledTotal int   `json:"canceled_total,omitempty"`
	SkippedTotal  int   `json:"skipped_total,omitempty"`
	LatencyMs     int64 `json:"latency_ms,omitempty"`
	LatencyP95Ms  int64 `json:"latency_p95_ms,omitempty"`
}

type TimelineTrendMode string

const (
	TimelineTrendModeLastNRuns  TimelineTrendMode = "last_n_runs"
	TimelineTrendModeTimeWindow TimelineTrendMode = "time_window"
)

type TimelineTrendQuery struct {
	Mode       TimelineTrendMode
	LastNRuns  int
	TimeWindow time.Duration
}

type TimelineTrendRecord struct {
	Phase         string    `json:"phase"`
	Status        string    `json:"status"`
	CountTotal    int       `json:"count_total"`
	FailedTotal   int       `json:"failed_total"`
	CanceledTotal int       `json:"canceled_total"`
	SkippedTotal  int       `json:"skipped_total"`
	LatencyAvgMs  int64     `json:"latency_avg_ms"`
	LatencyP95Ms  int64     `json:"latency_p95_ms"`
	WindowStart   time.Time `json:"window_start"`
	WindowEnd     time.Time `json:"window_end"`
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

	timelineStates map[string]*timelineRunState
	trendConfig    TimelineTrendConfig
	ca2TrendConfig CA2ExternalTrendConfig
}

type timelineRunState struct {
	seen           map[string]struct{}
	runningSince   map[string]time.Time
	phaseLatencyMs map[string][]int64
	phases         map[string]TimelinePhaseAggregate
	buckets        map[string]timelineTrendBucket
}

type timelineTrendBucket struct {
	CountTotal    int
	FailedTotal   int
	CanceledTotal int
	SkippedTotal  int
	LatencyTotal  int64
	Latencies     []int64
}

type TimelineTrendConfig struct {
	Enabled    bool
	LastNRuns  int
	TimeWindow time.Duration
}

type CA2ExternalTrendConfig struct {
	Enabled    bool
	Window     time.Duration
	Thresholds CA2ExternalThresholds
}

type CA2ExternalThresholds struct {
	P95LatencyMs int64
	ErrorRate    float64
	HitRate      float64
}

type CA2ExternalTrendQuery struct {
	Window time.Duration
}

type CA2ExternalTrendRecord struct {
	Provider               string         `json:"provider"`
	WindowStart            time.Time      `json:"window_start"`
	WindowEnd              time.Time      `json:"window_end"`
	P95LatencyMs           int64          `json:"p95_latency_ms"`
	ErrorRate              float64        `json:"error_rate"`
	HitRate                float64        `json:"hit_rate"`
	ThresholdHits          []string       `json:"threshold_hits,omitempty"`
	ErrorLayerDistribution map[string]int `json:"error_layer_distribution,omitempty"`
}

func NewStore(maxCalls, maxRuns, maxReloads, maxSkills int, trend TimelineTrendConfig, ca2 CA2ExternalTrendConfig) *Store {
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
	if trend.LastNRuns <= 0 {
		trend.LastNRuns = 100
	}
	if trend.TimeWindow <= 0 {
		trend.TimeWindow = 15 * time.Minute
	}
	if ca2.Window <= 0 {
		ca2.Window = 15 * time.Minute
	}
	if ca2.Thresholds.P95LatencyMs <= 0 {
		ca2.Thresholds.P95LatencyMs = 1500
	}
	if ca2.Thresholds.ErrorRate < 0 || ca2.Thresholds.ErrorRate > 1 {
		ca2.Thresholds.ErrorRate = 0.1
	}
	if ca2.Thresholds.HitRate < 0 || ca2.Thresholds.HitRate > 1 {
		ca2.Thresholds.HitRate = 0.2
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
		timelineStates:  make(map[string]*timelineRunState, maxRuns),
		trendConfig:     trend,
		ca2TrendConfig:  ca2,
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
		d.pruneTimelineStates()
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

func (d *Store) SetTrendConfig(cfg TimelineTrendConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if cfg.LastNRuns <= 0 {
		cfg.LastNRuns = 100
	}
	if cfg.TimeWindow <= 0 {
		cfg.TimeWindow = 15 * time.Minute
	}
	d.trendConfig = cfg
}

func (d *Store) SetCA2ExternalTrendConfig(cfg CA2ExternalTrendConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if cfg.Window <= 0 {
		cfg.Window = 15 * time.Minute
	}
	if cfg.Thresholds.P95LatencyMs <= 0 {
		cfg.Thresholds.P95LatencyMs = 1500
	}
	if cfg.Thresholds.ErrorRate < 0 || cfg.Thresholds.ErrorRate > 1 {
		cfg.Thresholds.ErrorRate = 0.1
	}
	if cfg.Thresholds.HitRate < 0 || cfg.Thresholds.HitRate > 1 {
		cfg.Thresholds.HitRate = 0.2
	}
	d.ca2TrendConfig = cfg
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
	if len(rec.TimelinePhases) == 0 {
		rec.TimelinePhases = d.timelinePhasesForRun(rec.RunID)
	}
	key := RunIdempotencyKey(rec)
	if idx, ok := d.runKeys[key]; ok && idx >= 0 && idx < len(d.runs) {
		d.runs[idx] = rec
		return
	}
	d.runs = append(d.runs, rec)
	d.runs = trimTail(d.runs, d.maxRunRecords)
	d.rebuildRunKeys()
	d.pruneTimelineStates()
}

func (d *Store) AddTimelineEvent(runID, phase, status string, sequence int64, ts time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	runID = strings.TrimSpace(runID)
	phase = strings.TrimSpace(phase)
	status = strings.ToLower(strings.TrimSpace(status))
	if runID == "" || phase == "" || sequence <= 0 {
		return
	}
	state := d.timelineStates[runID]
	if state == nil {
		state = &timelineRunState{
			seen:           map[string]struct{}{},
			runningSince:   map[string]time.Time{},
			phaseLatencyMs: map[string][]int64{},
			phases:         map[string]TimelinePhaseAggregate{},
			buckets:        map[string]timelineTrendBucket{},
		}
		d.timelineStates[runID] = state
	}
	key := fmt.Sprintf("%d:%s:%s", sequence, phase, status)
	if _, ok := state.seen[key]; ok {
		return
	}
	state.seen[key] = struct{}{}
	if ts.IsZero() {
		ts = time.Now()
	}
	switch status {
	case "running":
		state.runningSince[phase] = ts
	case "succeeded", "failed", "canceled", "skipped":
		agg := state.phases[phase]
		agg.CountTotal++
		switch status {
		case "failed":
			agg.FailedTotal++
		case "canceled":
			agg.CanceledTotal++
		case "skipped":
			agg.SkippedTotal++
		}
		if startedAt, ok := state.runningSince[phase]; ok && !startedAt.IsZero() {
			lat := ts.Sub(startedAt).Milliseconds()
			if lat < 0 {
				lat = 0
			}
			agg.LatencyMs += lat
			phaseSamples := state.phaseLatencyMs[phase]
			phaseSamples = append(phaseSamples, lat)
			state.phaseLatencyMs[phase] = phaseSamples
			agg.LatencyP95Ms = percentileP95(phaseSamples)
			delete(state.runningSince, phase)
		}
		state.phases[phase] = agg
		state.recordBucket(phase, status, state.phaseLatencyMs[phase])
	}
}

func (s *timelineRunState) recordBucket(phase, status string, phaseSamples []int64) {
	if s == nil {
		return
	}
	key := trendBucketKey(phase, status)
	b := s.buckets[key]
	b.CountTotal++
	switch status {
	case "failed":
		b.FailedTotal++
	case "canceled":
		b.CanceledTotal++
	case "skipped":
		b.SkippedTotal++
	}
	if len(phaseSamples) > 0 {
		lat := phaseSamples[len(phaseSamples)-1]
		b.LatencyTotal += lat
		b.Latencies = append(b.Latencies, lat)
	}
	s.buckets[key] = b
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

func (d *Store) TimelineTrends(query TimelineTrendQuery) []TimelineTrendRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.trendConfig.Enabled {
		return []TimelineTrendRecord{}
	}
	selected, start, end := d.selectTrendRuns(query)
	if len(selected) == 0 {
		return []TimelineTrendRecord{}
	}
	type aggState struct {
		countTotal    int
		failedTotal   int
		canceledTotal int
		skippedTotal  int
		latencyTotal  int64
		latencies     []int64
	}
	agg := map[string]*aggState{}
	for _, rec := range selected {
		runID := strings.TrimSpace(rec.RunID)
		if runID == "" {
			continue
		}
		state := d.timelineStates[runID]
		if state == nil {
			continue
		}
		for bucketKey, bucket := range state.buckets {
			if bucket.CountTotal == 0 {
				continue
			}
			s := agg[bucketKey]
			if s == nil {
				s = &aggState{}
				agg[bucketKey] = s
			}
			s.countTotal += bucket.CountTotal
			s.failedTotal += bucket.FailedTotal
			s.canceledTotal += bucket.CanceledTotal
			s.skippedTotal += bucket.SkippedTotal
			s.latencyTotal += bucket.LatencyTotal
			if len(bucket.Latencies) > 0 {
				s.latencies = append(s.latencies, bucket.Latencies...)
			}
		}
	}
	if len(agg) == 0 {
		return []TimelineTrendRecord{}
	}
	out := make([]TimelineTrendRecord, 0, len(agg))
	for key, s := range agg {
		phase, status := splitTrendBucketKey(key)
		latAvg := int64(0)
		if s.countTotal > 0 {
			latAvg = s.latencyTotal / int64(s.countTotal)
		}
		out = append(out, TimelineTrendRecord{
			Phase:         phase,
			Status:        status,
			CountTotal:    s.countTotal,
			FailedTotal:   s.failedTotal,
			CanceledTotal: s.canceledTotal,
			SkippedTotal:  s.skippedTotal,
			LatencyAvgMs:  latAvg,
			LatencyP95Ms:  percentileP95(s.latencies),
			WindowStart:   start,
			WindowEnd:     end,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Phase == out[j].Phase {
			return out[i].Status < out[j].Status
		}
		return out[i].Phase < out[j].Phase
	})
	return out
}

func (d *Store) CA2ExternalTrends(query CA2ExternalTrendQuery) []CA2ExternalTrendRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.ca2TrendConfig.Enabled {
		return []CA2ExternalTrendRecord{}
	}
	selected, start, end := d.selectCA2Runs(query)
	if len(selected) == 0 {
		return []CA2ExternalTrendRecord{}
	}
	type agg struct {
		total      int
		hits       int
		errors     int
		latencies  []int64
		layerCount map[string]int
	}
	byProvider := map[string]*agg{}
	for i := range selected {
		provider := strings.ToLower(strings.TrimSpace(selected[i].Stage2Provider))
		if provider == "" {
			continue
		}
		item := byProvider[provider]
		if item == nil {
			item = &agg{layerCount: map[string]int{}}
			byProvider[provider] = item
		}
		item.total++
		if selected[i].Stage2LatencyMs > 0 {
			item.latencies = append(item.latencies, selected[i].Stage2LatencyMs)
		}
		if selected[i].Stage2HitCount > 0 {
			item.hits++
		}
		if isCA2ExternalError(selected[i]) {
			item.errors++
			layer := strings.ToLower(strings.TrimSpace(selected[i].Stage2ErrorLayer))
			if layer == "" {
				layer = "unknown"
			}
			item.layerCount[layer]++
		}
	}
	if len(byProvider) == 0 {
		return []CA2ExternalTrendRecord{}
	}
	out := make([]CA2ExternalTrendRecord, 0, len(byProvider))
	for provider, item := range byProvider {
		if item.total == 0 {
			continue
		}
		errorRate := float64(item.errors) / float64(item.total)
		hitRate := float64(item.hits) / float64(item.total)
		p95 := percentileP95(item.latencies)
		thresholdHits := make([]string, 0, 3)
		if p95 > d.ca2TrendConfig.Thresholds.P95LatencyMs {
			thresholdHits = append(thresholdHits, "p95_latency_ms")
		}
		if errorRate > d.ca2TrendConfig.Thresholds.ErrorRate {
			thresholdHits = append(thresholdHits, "error_rate")
		}
		if hitRate < d.ca2TrendConfig.Thresholds.HitRate {
			thresholdHits = append(thresholdHits, "hit_rate")
		}
		sort.Strings(thresholdHits)
		out = append(out, CA2ExternalTrendRecord{
			Provider:               provider,
			WindowStart:            start,
			WindowEnd:              end,
			P95LatencyMs:           p95,
			ErrorRate:              errorRate,
			HitRate:                hitRate,
			ThresholdHits:          thresholdHits,
			ErrorLayerDistribution: item.layerCount,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
	return out
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
	return redaction.New(true, redaction.DefaultKeywords()).SanitizeMap(in)
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

func (d *Store) timelinePhasesForRun(runID string) map[string]TimelinePhaseAggregate {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil
	}
	state := d.timelineStates[runID]
	if state == nil || len(state.phases) == 0 {
		return nil
	}
	out := make(map[string]TimelinePhaseAggregate, len(state.phases))
	for phase, agg := range state.phases {
		out[phase] = agg
	}
	return out
}

func (d *Store) selectTrendRuns(query TimelineTrendQuery) ([]RunRecord, time.Time, time.Time) {
	if len(d.runs) == 0 {
		return nil, time.Time{}, time.Time{}
	}
	mode := query.Mode
	if mode == "" {
		mode = TimelineTrendModeLastNRuns
	}
	switch mode {
	case TimelineTrendModeTimeWindow:
		window := query.TimeWindow
		if window <= 0 {
			window = d.trendConfig.TimeWindow
		}
		if window <= 0 {
			return nil, time.Time{}, time.Time{}
		}
		end := d.runs[len(d.runs)-1].Time
		if end.IsZero() {
			end = time.Now()
		}
		start := end.Add(-window)
		selected := make([]RunRecord, 0, len(d.runs))
		for i := range d.runs {
			ts := d.runs[i].Time
			if ts.IsZero() {
				continue
			}
			if ts.Before(start) || ts.After(end) {
				continue
			}
			selected = append(selected, d.runs[i])
		}
		return selected, start, end
	default:
		n := query.LastNRuns
		if n <= 0 {
			n = d.trendConfig.LastNRuns
		}
		selected := tailCopy(d.runs, n)
		if len(selected) == 0 {
			return nil, time.Time{}, time.Time{}
		}
		start := selected[0].Time
		end := selected[len(selected)-1].Time
		return selected, start, end
	}
}

func (d *Store) selectCA2Runs(query CA2ExternalTrendQuery) ([]RunRecord, time.Time, time.Time) {
	if len(d.runs) == 0 {
		return nil, time.Time{}, time.Time{}
	}
	window := query.Window
	if window <= 0 {
		window = d.ca2TrendConfig.Window
	}
	if window <= 0 {
		return nil, time.Time{}, time.Time{}
	}
	end := d.runs[len(d.runs)-1].Time
	if end.IsZero() {
		end = time.Now()
	}
	start := end.Add(-window)
	selected := make([]RunRecord, 0, len(d.runs))
	for i := range d.runs {
		rec := d.runs[i]
		ts := rec.Time
		if ts.IsZero() {
			continue
		}
		if ts.Before(start) || ts.After(end) {
			continue
		}
		if strings.TrimSpace(rec.Stage2Provider) == "" {
			continue
		}
		selected = append(selected, rec)
	}
	return selected, start, end
}

func trendBucketKey(phase, status string) string {
	return strings.TrimSpace(phase) + "|" + strings.ToLower(strings.TrimSpace(status))
}

func splitTrendBucketKey(key string) (string, string) {
	parts := strings.SplitN(key, "|", 2)
	if len(parts) != 2 {
		return key, ""
	}
	return parts[0], parts[1]
}

func isCA2ExternalError(rec RunRecord) bool {
	if strings.TrimSpace(rec.Stage2ErrorLayer) != "" {
		return true
	}
	code := strings.ToLower(strings.TrimSpace(rec.Stage2ReasonCode))
	return code != "" && code != "ok"
}

func (d *Store) pruneTimelineStates() {
	if len(d.timelineStates) == 0 {
		return
	}
	keep := make(map[string]struct{}, len(d.runs))
	for i := range d.runs {
		runID := strings.TrimSpace(d.runs[i].RunID)
		if runID == "" {
			continue
		}
		keep[runID] = struct{}{}
	}
	for runID := range d.timelineStates {
		if _, ok := keep[runID]; ok {
			continue
		}
		delete(d.timelineStates, runID)
	}
}

func percentileP95(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	if len(samples) == 1 {
		return samples[0]
	}
	cp := make([]int64, len(samples))
	copy(cp, samples)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(math.Ceil(0.95*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
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
