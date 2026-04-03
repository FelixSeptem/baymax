package composer

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/memory"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	orchestrationsnapshot "github.com/FelixSeptem/baymax/orchestration/snapshot"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type UnifiedSnapshotExportRequest struct {
	RunID      string
	SessionID  string
	WorkflowID string
}

type UnifiedSnapshotExportResult struct {
	Manifest orchestrationsnapshot.Manifest
	Payload  []byte
}

type UnifiedSnapshotImportRequest struct {
	Payload      []byte
	OperationID  string
	RestoreMode  string
	CompatWindow int
}

type UnifiedSnapshotImportResult struct {
	RunID           string
	OperationID     string
	RestoreMode     string
	RestoreAction   string
	ConflictCode    string
	AppliedSegments []string
	SkippedSegments []string
}

type unifiedRunnerSessionSegmentPayload struct {
	RunID      string `json:"run_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	Status     string `json:"status,omitempty"`
}

type unifiedSchedulerMailboxSegmentPayload struct {
	Scheduler scheduler.StoreSnapshot `json:"scheduler"`
	Mailbox   mailbox.Snapshot        `json:"mailbox"`
}

type unifiedComposerRecoverySegmentPayload struct {
	Recovery RecoverySnapshot `json:"recovery"`
}

type unifiedMemorySegmentPayload struct {
	ContractVersion   string                         `json:"contract_version,omitempty"`
	Source            string                         `json:"source,omitempty"`
	Mode              string                         `json:"mode,omitempty"`
	Provider          string                         `json:"provider,omitempty"`
	Profile           string                         `json:"profile,omitempty"`
	Lifecycle         unifiedMemoryLifecycleSnapshot `json:"lifecycle"`
	Search            unifiedMemorySearchSnapshot    `json:"search"`
	RetrievalBaseline unifiedMemoryRetrievalBaseline `json:"retrieval_baseline,omitempty"`
	ExportedAt        time.Time                      `json:"exported_at,omitempty"`
}

type unifiedMemoryLifecycleSnapshot struct {
	RetentionDays    int      `json:"retention_days,omitempty"`
	TTLEnabled       bool     `json:"ttl_enabled,omitempty"`
	TTL              string   `json:"ttl,omitempty"`
	ForgetScopeAllow []string `json:"forget_scope_allow,omitempty"`
	LastAction       string   `json:"last_action,omitempty"`
}

type unifiedMemorySearchSnapshot struct {
	IndexUpdatePolicy   string `json:"index_update_policy,omitempty"`
	DriftRecoveryPolicy string `json:"drift_recovery_policy,omitempty"`
}

type unifiedMemoryRetrievalBaseline struct {
	ScopeSelected string         `json:"scope_selected,omitempty"`
	Hits          int            `json:"hits,omitempty"`
	BudgetUsed    int            `json:"budget_used,omitempty"`
	RerankStats   map[string]int `json:"rerank_stats,omitempty"`
}

type unifiedMemoryRunBaseline struct {
	RunID               string
	MemoryScopeSelected string
	MemoryHits          int
	MemoryBudgetUsed    int
	MemoryRerankStats   map[string]int
}

func (c *Composer) ExportUnifiedSnapshot(ctx context.Context, req UnifiedSnapshotExportRequest) (UnifiedSnapshotExportResult, error) {
	if c == nil {
		return UnifiedSnapshotExportResult{}, fmt.Errorf("composer is nil")
	}
	s := c.Scheduler()
	if s == nil {
		return UnifiedSnapshotExportResult{}, newRecoveryError(RecoveryErrorStoreUnavailable, "scheduler is not initialized", nil)
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return UnifiedSnapshotExportResult{}, newRecoveryError(RecoveryErrorSnapshotCorrupt, "unified snapshot export requires run_id", nil)
	}

	schedulerSnapshot, err := s.Snapshot(ctx)
	if err != nil {
		return UnifiedSnapshotExportResult{}, newRecoveryError(RecoveryErrorStoreUnavailable, "capture scheduler snapshot", err)
	}
	mailboxSnapshot, err := c.captureMailboxSnapshot(ctx)
	if err != nil {
		return UnifiedSnapshotExportResult{}, err
	}
	recoverySnapshot, err := c.CaptureRecoverySnapshot(ctx, runID, req.WorkflowID)
	if err != nil {
		return UnifiedSnapshotExportResult{}, err
	}

	manifest, err := orchestrationsnapshot.ExportManifest(orchestrationsnapshot.ExportRequest{
		ExportedAt: recoverySnapshot.UpdatedAt,
		Source: orchestrationsnapshot.Source{
			Component: "composer",
			RunID:     runID,
			SessionID: strings.TrimSpace(req.SessionID),
		},
		RunnerSessionPayload: unifiedRunnerSessionSegmentPayload{
			RunID:      runID,
			SessionID:  strings.TrimSpace(req.SessionID),
			WorkflowID: strings.TrimSpace(req.WorkflowID),
			Status:     strings.TrimSpace(recoverySnapshot.Run.Status),
		},
		SchedulerMailboxPayload: unifiedSchedulerMailboxSegmentPayload{
			Scheduler: schedulerSnapshot,
			Mailbox:   mailboxSnapshot,
		},
		ComposerRecoveryPayload: unifiedComposerRecoverySegmentPayload{
			Recovery: recoverySnapshot,
		},
		MemoryPayload: c.exportUnifiedMemoryPayload(runID),
	})
	if err != nil {
		return UnifiedSnapshotExportResult{}, newRecoveryError(RecoveryErrorSnapshotCorrupt, "export unified snapshot manifest", err)
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return UnifiedSnapshotExportResult{}, newRecoveryError(RecoveryErrorSnapshotCorrupt, "encode unified snapshot manifest", err)
	}
	return UnifiedSnapshotExportResult{
		Manifest: manifest,
		Payload:  payload,
	}, nil
}

func (c *Composer) ImportUnifiedSnapshot(ctx context.Context, req UnifiedSnapshotImportRequest) (UnifiedSnapshotImportResult, error) {
	if c == nil {
		return UnifiedSnapshotImportResult{}, fmt.Errorf("composer is nil")
	}
	c.refreshSchedulerForNextAttempt()
	c.refreshMailboxForNextAttempt()
	c.refreshRecoveryForNextAttempt()

	cfg := c.effectiveConfig()
	restoreMode := strings.TrimSpace(req.RestoreMode)
	if restoreMode == "" {
		restoreMode = strings.TrimSpace(cfg.Runtime.State.Snapshot.RestoreMode)
	}
	compatWindow := req.CompatWindow
	if compatWindow < 0 {
		compatWindow = cfg.Runtime.State.Snapshot.CompatWindow
	}

	importer := c.unifiedSnapshotImporter()
	imported, err := importer.Import(orchestrationsnapshot.ImportRequest{
		Payload:      req.Payload,
		RestoreMode:  restoreMode,
		CompatWindow: compatWindow,
		OperationID:  req.OperationID,
	})
	if err != nil {
		return UnifiedSnapshotImportResult{}, c.wrapUnifiedSnapshotImportError("", err)
	}

	runnerPayload, err := decodeRunnerSessionPayload(imported.Manifest.Segments.RunnerSession.Payload)
	if err != nil {
		return UnifiedSnapshotImportResult{}, c.wrapUnifiedSnapshotImportError("", err)
	}
	runID := strings.TrimSpace(runnerPayload.RunID)
	if runID == "" {
		runID = strings.TrimSpace(imported.Manifest.Source.RunID)
	}
	result := UnifiedSnapshotImportResult{
		RunID:         runID,
		OperationID:   imported.OperationID,
		RestoreMode:   imported.RestoreMode,
		RestoreAction: imported.RestoreAction,
	}
	if imported.Idempotent || !imported.Applied {
		return result, nil
	}

	segmentPayload, err := decodeSchedulerMailboxPayload(imported.Manifest.Segments.SchedulerMailbox.Payload)
	if err != nil {
		return result, c.wrapUnifiedSnapshotImportError(runID, err)
	}
	recoveryPayload, err := decodeComposerRecoveryPayload(imported.Manifest.Segments.ComposerRecovery.Payload)
	if err != nil {
		return result, c.wrapUnifiedSnapshotImportError(runID, err)
	}
	memoryPayload, err := decodeMemoryPayload(imported.Manifest.Segments.Memory.Payload)
	if err != nil {
		return result, c.wrapUnifiedSnapshotImportError(runID, err)
	}
	normalizedRecovery, err := normalizeRecoverySnapshot(recoveryPayload.Recovery, runID)
	if err != nil {
		return result, c.wrapUnifiedSnapshotImportError(runID, err)
	}

	restoreMode = strings.ToLower(strings.TrimSpace(imported.RestoreMode))
	allowCompatible := restoreMode == orchestrationsnapshot.RestoreModeCompatible

	currentScheduler, snapErr := c.Scheduler().Snapshot(ctx)
	if snapErr != nil {
		return result, c.wrapUnifiedSnapshotImportError(runID, snapErr)
	}
	if violation, ok := validateRecoveryBoundary(segmentPayload.Scheduler); ok {
		if !allowCompatible {
			result.ConflictCode = "snapshot_recovery_boundary_violation"
			return result, c.wrapUnifiedSnapshotConflict(runID, result.ConflictCode, "strict unified snapshot restore rejected boundary violation", violation.Err)
		}
		result.SkippedSegments = append(result.SkippedSegments, "scheduler_mailbox")
		result.ConflictCode = "snapshot_recovery_boundary_violation"
		result.RestoreAction = orchestrationsnapshot.RestoreActionCompatibleBounded
	} else if hasSchedulerSnapshotState(currentScheduler) && !schedulerSnapshotsEqual(currentScheduler, segmentPayload.Scheduler) {
		if !allowCompatible {
			result.ConflictCode = "snapshot_scheduler_conflict"
			return result, c.wrapUnifiedSnapshotConflict(runID, result.ConflictCode, "strict unified snapshot restore rejected scheduler conflict", nil)
		}
		result.SkippedSegments = append(result.SkippedSegments, "scheduler_mailbox")
		result.ConflictCode = "snapshot_scheduler_conflict"
		result.RestoreAction = orchestrationsnapshot.RestoreActionCompatibleBounded
	} else {
		if err := c.Scheduler().Restore(ctx, segmentPayload.Scheduler); err != nil {
			return result, c.wrapUnifiedSnapshotImportError(runID, err)
		}
		result.AppliedSegments = append(result.AppliedSegments, "scheduler_mailbox")
	}

	if err := c.restoreMailboxFromUnifiedSnapshot(ctx, segmentPayload.Mailbox, allowCompatible, &result); err != nil {
		return result, err
	}

	if c.recoveryEnabled && c.recoveryStore != nil {
		if err := c.recoveryStore.Save(ctx, normalizedRecovery); err != nil {
			return result, c.wrapUnifiedSnapshotImportError(runID, err)
		}
		result.AppliedSegments = append(result.AppliedSegments, "composer_recovery")
	}
	if err := c.restoreMemoryFromUnifiedSnapshot(runID, memoryPayload, allowCompatible, &result); err != nil {
		return result, err
	}

	if runID != "" && strings.TrimSpace(result.ConflictCode) == "" {
		c.rebuildCollabStatsFromSchedulerSnapshot(runID, segmentPayload.Scheduler)
		c.markRecoveryRecovered(runID, normalizedRecovery.Replay.TerminalCommitCount)
	}
	return result, nil
}

func (c *Composer) unifiedSnapshotImporter() *orchestrationsnapshot.Importer {
	c.schedulerMu.Lock()
	defer c.schedulerMu.Unlock()
	if c.stateSnapshotImporter == nil {
		c.stateSnapshotImporter = orchestrationsnapshot.NewImporter()
	}
	return c.stateSnapshotImporter
}

func (c *Composer) captureMailboxSnapshot(ctx context.Context) (mailbox.Snapshot, error) {
	c.schedulerMu.RLock()
	runtime := c.mailbox
	c.schedulerMu.RUnlock()
	if runtime == nil || runtime.mailbox == nil {
		return mailbox.Snapshot{Backend: "memory"}, nil
	}
	snapshot, err := runtime.mailbox.Snapshot(ctx)
	if err != nil {
		return mailbox.Snapshot{}, newRecoveryError(RecoveryErrorStoreUnavailable, "capture mailbox snapshot", err)
	}
	return snapshot, nil
}

func (c *Composer) restoreMailboxFromUnifiedSnapshot(
	ctx context.Context,
	target mailbox.Snapshot,
	allowCompatible bool,
	result *UnifiedSnapshotImportResult,
) error {
	c.schedulerMu.RLock()
	runtime := c.mailbox
	c.schedulerMu.RUnlock()
	if runtime == nil || runtime.mailbox == nil {
		return nil
	}
	current, err := runtime.mailbox.Snapshot(ctx)
	if err != nil {
		return c.wrapUnifiedSnapshotImportError(result.RunID, err)
	}
	if hasMailboxSnapshotState(current) && !mailboxSnapshotsEqual(current, target) {
		if !allowCompatible {
			result.ConflictCode = "snapshot_mailbox_conflict"
			return c.wrapUnifiedSnapshotConflict(result.RunID, result.ConflictCode, "strict unified snapshot restore rejected mailbox conflict", nil)
		}
		result.SkippedSegments = append(result.SkippedSegments, "scheduler_mailbox")
		result.ConflictCode = "snapshot_mailbox_conflict"
		result.RestoreAction = orchestrationsnapshot.RestoreActionCompatibleBounded
		return nil
	}
	if err := runtime.mailbox.Restore(ctx, target); err != nil {
		return c.wrapUnifiedSnapshotImportError(result.RunID, err)
	}
	if !containsString(result.AppliedSegments, "scheduler_mailbox") {
		result.AppliedSegments = append(result.AppliedSegments, "scheduler_mailbox")
	}
	return nil
}

func decodeRunnerSessionPayload(raw json.RawMessage) (unifiedRunnerSessionSegmentPayload, error) {
	var payload unifiedRunnerSessionSegmentPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return unifiedRunnerSessionSegmentPayload{}, fmt.Errorf("decode runner/session segment payload: %w", err)
	}
	payload.RunID = strings.TrimSpace(payload.RunID)
	payload.SessionID = strings.TrimSpace(payload.SessionID)
	payload.WorkflowID = strings.TrimSpace(payload.WorkflowID)
	payload.Status = strings.TrimSpace(payload.Status)
	return payload, nil
}

func decodeSchedulerMailboxPayload(raw json.RawMessage) (unifiedSchedulerMailboxSegmentPayload, error) {
	var payload unifiedSchedulerMailboxSegmentPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return unifiedSchedulerMailboxSegmentPayload{}, fmt.Errorf("decode scheduler/mailbox segment payload: %w", err)
	}
	return payload, nil
}

func decodeComposerRecoveryPayload(raw json.RawMessage) (unifiedComposerRecoverySegmentPayload, error) {
	var payload unifiedComposerRecoverySegmentPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return unifiedComposerRecoverySegmentPayload{}, fmt.Errorf("decode composer recovery segment payload: %w", err)
	}
	return payload, nil
}

func decodeMemoryPayload(raw json.RawMessage) (unifiedMemorySegmentPayload, error) {
	var payload unifiedMemorySegmentPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return unifiedMemorySegmentPayload{}, fmt.Errorf("decode memory segment payload: %w", err)
	}
	payload.ContractVersion = strings.ToLower(strings.TrimSpace(payload.ContractVersion))
	payload.Source = strings.TrimSpace(payload.Source)
	payload.Mode = strings.ToLower(strings.TrimSpace(payload.Mode))
	payload.Provider = strings.ToLower(strings.TrimSpace(payload.Provider))
	payload.Profile = strings.ToLower(strings.TrimSpace(payload.Profile))
	payload.Lifecycle.ForgetScopeAllow = normalizeScopeList(payload.Lifecycle.ForgetScopeAllow)
	payload.Lifecycle.LastAction = strings.TrimSpace(payload.Lifecycle.LastAction)
	payload.Search.IndexUpdatePolicy = strings.ToLower(strings.TrimSpace(payload.Search.IndexUpdatePolicy))
	payload.Search.DriftRecoveryPolicy = strings.ToLower(strings.TrimSpace(payload.Search.DriftRecoveryPolicy))
	payload.RetrievalBaseline.ScopeSelected = strings.ToLower(strings.TrimSpace(payload.RetrievalBaseline.ScopeSelected))
	payload.RetrievalBaseline.RerankStats = cloneIntMapCopy(payload.RetrievalBaseline.RerankStats)
	return payload, nil
}

func (c *Composer) wrapUnifiedSnapshotConflict(runID, conflictCode, message string, cause error) error {
	conflictCode = strings.TrimSpace(conflictCode)
	if runID != "" && conflictCode != "" {
		c.markRecoveryConflict(runID, conflictCode)
	}
	return newRecoveryError(RecoveryErrorConflict, message, cause)
}

func (c *Composer) wrapUnifiedSnapshotImportError(runID string, err error) error {
	if err == nil {
		return nil
	}
	var importErr *orchestrationsnapshot.ImportError
	if errorsAsImportError(err, &importErr) {
		code := strings.TrimSpace(importErr.ConflictCode)
		if runID != "" && code != "" {
			c.markRecoveryConflict(runID, code)
		}
		switch code {
		case orchestrationsnapshot.ConflictCodeInvalidRestoreMode:
			return newRecoveryError(RecoveryErrorPolicyUnsupported, "invalid unified snapshot restore mode", err)
		case orchestrationsnapshot.ConflictCodeInvalidPayload, orchestrationsnapshot.ConflictCodeDigestMismatch:
			return newRecoveryError(RecoveryErrorSnapshotCorrupt, "invalid unified snapshot payload", err)
		default:
			return newRecoveryError(RecoveryErrorConflict, "unified snapshot restore conflict", err)
		}
	}
	return newRecoveryError(RecoveryErrorSnapshotCorrupt, "unified snapshot import failed", err)
}

func hasMailboxSnapshotState(snapshot mailbox.Snapshot) bool {
	return len(snapshot.Records) > 0 || len(snapshot.Queue) > 0 || len(snapshot.Idempotency) > 0
}

func mailboxSnapshotsEqual(left, right mailbox.Snapshot) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftRaw) == string(rightRaw)
}

func containsString(items []string, target string) bool {
	for i := range items {
		if strings.TrimSpace(items[i]) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}

func errorsAsImportError(err error, out **orchestrationsnapshot.ImportError) bool {
	if err == nil {
		return false
	}
	typed, ok := err.(*orchestrationsnapshot.ImportError)
	if !ok {
		return false
	}
	*out = typed
	return true
}

func (c *Composer) exportUnifiedMemoryPayload(runID string) unifiedMemorySegmentPayload {
	cfg := c.effectiveConfig().Runtime.Memory
	out := unifiedMemorySegmentPayload{
		ContractVersion: memory.ContractVersionMemoryV1,
		Source:          "memory.spi",
		Mode:            strings.ToLower(strings.TrimSpace(cfg.Mode)),
		Provider:        strings.ToLower(strings.TrimSpace(cfg.External.Provider)),
		Profile:         strings.ToLower(strings.TrimSpace(cfg.External.Profile)),
		Lifecycle: unifiedMemoryLifecycleSnapshot{
			RetentionDays:    cfg.Lifecycle.RetentionDays,
			TTLEnabled:       cfg.Lifecycle.TTLEnabled,
			TTL:              cfg.Lifecycle.TTL.String(),
			ForgetScopeAllow: normalizeScopeList(cfg.Lifecycle.ForgetScopeAllow),
		},
		Search: unifiedMemorySearchSnapshot{
			IndexUpdatePolicy:   strings.ToLower(strings.TrimSpace(cfg.Search.IndexUpdatePolicy)),
			DriftRecoveryPolicy: strings.ToLower(strings.TrimSpace(cfg.Search.DriftRecoveryPolicy)),
		},
		ExportedAt: time.Now().UTC(),
	}
	if m := c.runtimeMgr; m != nil && strings.TrimSpace(runID) != "" {
		recent := m.RecentRuns(50)
		for i := len(recent) - 1; i >= 0; i-- {
			rec := recent[i]
			if strings.TrimSpace(rec.RunID) != strings.TrimSpace(runID) {
				continue
			}
			out.Lifecycle.LastAction = strings.TrimSpace(rec.MemoryLifecycleAction)
			out.RetrievalBaseline = unifiedMemoryRetrievalBaseline{
				ScopeSelected: strings.ToLower(strings.TrimSpace(rec.MemoryScopeSelected)),
				Hits:          rec.MemoryHits,
				BudgetUsed:    rec.MemoryBudgetUsed,
				RerankStats:   cloneIntMapCopy(rec.MemoryRerankStats),
			}
			break
		}
	}
	return out
}

func (c *Composer) restoreMemoryFromUnifiedSnapshot(
	runID string,
	payload unifiedMemorySegmentPayload,
	allowCompatible bool,
	result *UnifiedSnapshotImportResult,
) error {
	cfg := c.effectiveConfig().Runtime.Memory
	expectedContract := strings.ToLower(strings.TrimSpace(cfg.External.ContractVersion))
	if expectedContract == "" {
		expectedContract = memory.ContractVersionMemoryV1
	}
	actualContract := strings.ToLower(strings.TrimSpace(payload.ContractVersion))
	if actualContract == "" {
		actualContract = expectedContract
	}
	if expectedContract != actualContract {
		if !allowCompatible || !isCompatibleMemoryVersion(expectedContract, actualContract) {
			result.ConflictCode = "snapshot_memory_contract_mismatch"
			return c.wrapUnifiedSnapshotConflict(runID, result.ConflictCode, "memory contract version mismatch in unified snapshot", nil)
		}
		result.RestoreAction = orchestrationsnapshot.RestoreActionCompatibleBounded
		result.SkippedSegments = append(result.SkippedSegments, "memory")
		result.ConflictCode = "snapshot_memory_contract_mismatch"
		return nil
	}

	expectedLifecycle := cfg.Lifecycle
	if expectedLifecycle.RetentionDays != payload.Lifecycle.RetentionDays ||
		expectedLifecycle.TTLEnabled != payload.Lifecycle.TTLEnabled ||
		!equalDurationString(expectedLifecycle.TTL, payload.Lifecycle.TTL) ||
		!slices.Equal(normalizeScopeList(expectedLifecycle.ForgetScopeAllow), normalizeScopeList(payload.Lifecycle.ForgetScopeAllow)) {
		if !allowCompatible {
			result.ConflictCode = "snapshot_memory_lifecycle_mismatch"
			return c.wrapUnifiedSnapshotConflict(runID, result.ConflictCode, "memory lifecycle policy mismatch in strict unified snapshot restore", nil)
		}
		if !isCompatibleMemoryLifecycle(expectedLifecycle, payload.Lifecycle) {
			result.ConflictCode = "snapshot_memory_lifecycle_mismatch"
			return c.wrapUnifiedSnapshotConflict(runID, result.ConflictCode, "memory lifecycle policy outside compatible bounds", nil)
		}
		result.RestoreAction = orchestrationsnapshot.RestoreActionCompatibleBounded
	}

	if !isCompatibleSearchPolicy(cfg.Search, payload.Search) {
		if !allowCompatible {
			result.ConflictCode = "snapshot_memory_search_policy_mismatch"
			return c.wrapUnifiedSnapshotConflict(runID, result.ConflictCode, "memory search policy mismatch in strict unified snapshot restore", nil)
		}
		result.RestoreAction = orchestrationsnapshot.RestoreActionCompatibleBounded
		result.SkippedSegments = append(result.SkippedSegments, "memory")
		result.ConflictCode = "snapshot_memory_search_policy_mismatch"
		return nil
	}

	if strings.TrimSpace(runID) != "" && c.runtimeMgr != nil {
		if rec, ok := c.latestRunRecord(runID); ok {
			if !retrievalBaselineCompatible(rec, payload.RetrievalBaseline) {
				if !allowCompatible {
					result.ConflictCode = "snapshot_memory_retrieval_quality_drift"
					return c.wrapUnifiedSnapshotConflict(runID, result.ConflictCode, "memory retrieval quality drift under strict restore", nil)
				}
				result.RestoreAction = orchestrationsnapshot.RestoreActionCompatibleBounded
				result.SkippedSegments = append(result.SkippedSegments, "memory")
				result.ConflictCode = "snapshot_memory_retrieval_quality_drift"
				return nil
			}
		}
	}
	result.AppliedSegments = append(result.AppliedSegments, "memory")
	return nil
}

func latestRunIDMatch(runID string, candidate string) bool {
	return strings.TrimSpace(runID) != "" && strings.TrimSpace(runID) == strings.TrimSpace(candidate)
}

func (c *Composer) latestRunRecord(runID string) (unifiedMemoryRunBaseline, bool) {
	if c.runtimeMgr == nil || strings.TrimSpace(runID) == "" {
		return unifiedMemoryRunBaseline{}, false
	}
	records := c.runtimeMgr.RecentRuns(50)
	for i := len(records) - 1; i >= 0; i-- {
		if latestRunIDMatch(runID, records[i].RunID) {
			return unifiedMemoryRunBaseline{
				RunID:               records[i].RunID,
				MemoryScopeSelected: records[i].MemoryScopeSelected,
				MemoryHits:          records[i].MemoryHits,
				MemoryBudgetUsed:    records[i].MemoryBudgetUsed,
				MemoryRerankStats:   cloneIntMapCopy(records[i].MemoryRerankStats),
			}, true
		}
	}
	return unifiedMemoryRunBaseline{}, false
}

func retrievalBaselineCompatible(rec unifiedMemoryRunBaseline, baseline unifiedMemoryRetrievalBaseline) bool {
	if strings.TrimSpace(baseline.ScopeSelected) != "" &&
		strings.TrimSpace(strings.ToLower(rec.MemoryScopeSelected)) != strings.TrimSpace(strings.ToLower(baseline.ScopeSelected)) {
		return false
	}
	if baseline.Hits > 0 && absInt(rec.MemoryHits-baseline.Hits) > 1 {
		return false
	}
	if baseline.BudgetUsed > 0 && absInt(rec.MemoryBudgetUsed-baseline.BudgetUsed) > 1 {
		return false
	}
	if len(baseline.RerankStats) > 0 {
		for key, expected := range baseline.RerankStats {
			actual := rec.MemoryRerankStats[key]
			if absInt(actual-expected) > 1 {
				return false
			}
		}
	}
	return true
}

func isCompatibleMemoryVersion(expected, actual string) bool {
	ePrefix, eVer, ok := parseVersionSuffix(expected)
	if !ok {
		return false
	}
	aPrefix, aVer, ok := parseVersionSuffix(actual)
	if !ok {
		return false
	}
	if ePrefix != aPrefix {
		return false
	}
	diff := eVer - aVer
	if diff < 0 {
		diff = -diff
	}
	return diff <= 1
}

func parseVersionSuffix(raw string) (string, int, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	idx := strings.LastIndex(trimmed, ".v")
	if idx < 0 || idx+2 >= len(trimmed) {
		return "", 0, false
	}
	prefix := strings.TrimSpace(trimmed[:idx])
	if prefix == "" {
		return "", 0, false
	}
	ver, err := strconv.Atoi(strings.TrimSpace(trimmed[idx+2:]))
	if err != nil {
		return "", 0, false
	}
	return prefix, ver, true
}

func isCompatibleMemoryLifecycle(expected runtimeconfig.RuntimeMemoryLifecycleConfig, actual unifiedMemoryLifecycleSnapshot) bool {
	if actual.RetentionDays <= 0 {
		return false
	}
	if actual.TTLEnabled && strings.TrimSpace(actual.TTL) == "" {
		return false
	}
	actualScopes := normalizeScopeList(actual.ForgetScopeAllow)
	if len(actualScopes) == 0 {
		return false
	}
	expectedScopes := normalizeScopeList(expected.ForgetScopeAllow)
	for i := range actualScopes {
		if !slices.Contains(expectedScopes, actualScopes[i]) {
			return false
		}
	}
	return true
}

func isCompatibleSearchPolicy(expected runtimeconfig.RuntimeMemorySearchConfig, actual unifiedMemorySearchSnapshot) bool {
	if actual.IndexUpdatePolicy == "" {
		actual.IndexUpdatePolicy = strings.ToLower(strings.TrimSpace(expected.IndexUpdatePolicy))
	}
	if actual.DriftRecoveryPolicy == "" {
		actual.DriftRecoveryPolicy = strings.ToLower(strings.TrimSpace(expected.DriftRecoveryPolicy))
	}
	return strings.EqualFold(strings.TrimSpace(expected.IndexUpdatePolicy), strings.TrimSpace(actual.IndexUpdatePolicy)) &&
		strings.EqualFold(strings.TrimSpace(expected.DriftRecoveryPolicy), strings.TrimSpace(actual.DriftRecoveryPolicy))
}

func equalDurationString(expected time.Duration, actual string) bool {
	normalized := strings.TrimSpace(actual)
	if normalized == "" {
		return expected == 0
	}
	parsed, err := time.ParseDuration(normalized)
	if err != nil {
		return false
	}
	return expected == parsed
}

func normalizeScopeList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for i := range in {
		v := strings.ToLower(strings.TrimSpace(in[i]))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func cloneIntMapCopy(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
