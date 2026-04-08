package memory

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultCompactionMinOps      = 32
	defaultCompactionMaxWALBytes = 4 << 20
	snapshotFileName             = "memory.snapshot.json"
	snapshotNextFileName         = "memory.snapshot.next.json"
	snapshotBackupFileName       = "memory.snapshot.bak.json"
	walFileName                  = "memory.wal.jsonl"
	indexFileName                = "memory.index.json"
	indexSchemaVersionV2         = "memory.index.v2"
)

type FilesystemCompactionConfig struct {
	Enabled        bool  `json:"enabled"`
	MinOps         int   `json:"min_ops"`
	MaxWALBytes    int64 `json:"max_wal_bytes"`
	FsyncBatchSize int   `json:"fsync_batch_size"`
}

type FilesystemEngineConfig struct {
	RootDir            string                     `json:"root_dir"`
	Compaction         FilesystemCompactionConfig `json:"compaction"`
	Lifecycle          LifecycleConfig            `json:"lifecycle"`
	Search             SearchConfig               `json:"search"`
	Profile            string                     `json:"profile,omitempty"`
	Model              string                     `json:"model,omitempty"`
	IndexSchemaVersion string                     `json:"index_schema_version,omitempty"`
}

type FilesystemEngine struct {
	mu                  sync.RWMutex
	cfg                 FilesystemEngineConfig
	records             map[string]map[string]Record
	lastSeq             int64
	walFile             *os.File
	pendingFsyncOps     int
	opsSinceCompaction  int
	lastLifecycleAction string
	recoveryDrift       bool
}

type walEntry struct {
	Seq       int64    `json:"seq"`
	Op        string   `json:"op"`
	Namespace string   `json:"namespace"`
	Records   []Record `json:"records,omitempty"`
	IDs       []string `json:"ids,omitempty"`
}

type snapshotState struct {
	LastSeq int64    `json:"last_seq"`
	Records []Record `json:"records"`
}

type indexState struct {
	LastSeq        int64  `json:"last_seq"`
	Checksum       string `json:"checksum"`
	Profile        string `json:"profile,omitempty"`
	Model          string `json:"model,omitempty"`
	SchemaVersion  string `json:"schema_version"`
	IndexPolicy    string `json:"index_policy,omitempty"`
	RecoveryPolicy string `json:"recovery_policy,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

func NewFilesystemEngine(cfg FilesystemEngineConfig) (*FilesystemEngine, error) {
	normalized := normalizeFilesystemConfig(cfg)
	if strings.TrimSpace(normalized.RootDir) == "" {
		return nil, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "builtin filesystem root_dir is required",
		}
	}
	if err := os.MkdirAll(normalized.RootDir, 0o755); err != nil {
		return nil, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "create builtin filesystem root_dir failed",
			Cause:     err,
		}
	}
	engine := &FilesystemEngine{
		cfg:                 normalized,
		records:             map[string]map[string]Record{},
		lastLifecycleAction: LifecycleActionNone,
	}
	if err := engine.recover(); err != nil {
		return nil, err
	}
	walPath := engine.walPath()
	walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o600)
	if err != nil {
		return nil, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "open builtin memory WAL failed",
			Cause:     err,
		}
	}
	engine.walFile = walFile
	return engine, nil
}

func (e *FilesystemEngine) Close() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.walFile == nil {
		return nil
	}
	var err error
	if e.pendingFsyncOps > 0 {
		if syncErr := e.walFile.Sync(); syncErr != nil {
			err = syncErr
		} else {
			e.pendingFsyncOps = 0
		}
	}
	if closeErr := e.walFile.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	e.walFile = nil
	return err
}

func (e *FilesystemEngine) Query(req QueryRequest) (QueryResponse, error) {
	start := time.Now()
	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		return QueryResponse{}, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerSemantic,
			Message:   "query namespace is required",
		}
	}
	now := start.UTC()
	e.mu.RLock()
	lifecycleAction := LifecycleActionNone
	items, expired := e.collectQueryRecordsReadLocked(namespace, req, now)
	if expired > 0 {
		lifecycleAction = LifecycleActionTTL
	}
	if lifecycleAction == LifecycleActionNone && e.recoveryDrift {
		lifecycleAction = LifecycleActionRecoveryConsistencyDrift
	}
	e.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].ID < items[j].ID
	})
	if req.MaxItems > 0 && len(items) > req.MaxItems {
		items = items[:req.MaxItems]
	}
	return QueryResponse{
		OperationID:           req.OperationID,
		Namespace:             namespace,
		Records:               items,
		Total:                 len(items),
		ReasonCode:            ReasonCodeOK,
		LatencyMs:             time.Since(start).Milliseconds(),
		MemoryLifecycleAction: lifecycleAction,
	}, nil
}

func (e *FilesystemEngine) collectQueryRecordsReadLocked(namespace string, req QueryRequest, now time.Time) ([]Record, int) {
	items := make([]Record, 0)
	expired := 0
	byID := e.records[namespace]
	if len(byID) == 0 {
		return items, expired
	}
	for _, item := range byID {
		if e.isExpiredRecord(item, now) {
			expired++
			continue
		}
		if !matchRecord(item, req) {
			continue
		}
		items = append(items, cloneRecord(item))
	}
	return items, expired
}

func (e *FilesystemEngine) Upsert(req UpsertRequest) (UpsertResponse, error) {
	start := time.Now()
	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		return UpsertResponse{}, &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerSemantic,
			Message:   "upsert namespace is required",
		}
	}
	if len(req.Records) == 0 {
		return UpsertResponse{}, &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerSemantic,
			Message:   "upsert records are required",
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	lifecycleAction := LifecycleActionNone
	if expired := e.applyTTLLocked(time.Now().UTC()); expired > 0 {
		lifecycleAction = LifecycleActionTTL
	}
	space := e.records[namespace]
	if space == nil {
		space = map[string]Record{}
		e.records[namespace] = space
	}

	now := time.Now().UTC()
	updated := make([]Record, 0, len(req.Records))
	for _, item := range req.Records {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return UpsertResponse{}, &Error{
				Operation: OperationUpsert,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerSemantic,
				Message:   "upsert record id is required",
			}
		}
		record := cloneRecord(item)
		record.ID = id
		record.Namespace = namespace
		record.SessionID = strings.TrimSpace(record.SessionID)
		record.RunID = strings.TrimSpace(record.RunID)
		if record.Metadata == nil {
			record.Metadata = map[string]string{}
		}
		if existing, ok := space[id]; ok && !existing.CreatedAt.IsZero() {
			record.CreatedAt = existing.CreatedAt
		}
		if record.CreatedAt.IsZero() {
			record.CreatedAt = now
		}
		record.UpdatedAt = now
		space[id] = record
		updated = append(updated, record)
	}
	e.lastSeq++
	entry := walEntry{
		Seq:       e.lastSeq,
		Op:        OperationUpsert,
		Namespace: namespace,
		Records:   updated,
	}
	if err := e.appendWALEntryLocked(entry); err != nil {
		return UpsertResponse{}, err
	}
	if trimmed := e.applyRetentionLocked(namespace); len(trimmed) > 0 {
		lifecycleAction = LifecycleActionRetention
		e.lastSeq++
		trimEntry := walEntry{
			Seq:       e.lastSeq,
			Op:        OperationDelete,
			Namespace: namespace,
			IDs:       trimmed,
		}
		if err := e.appendWALEntryLocked(trimEntry); err != nil {
			return UpsertResponse{}, err
		}
	}
	if err := e.maintainIndexLocked(); err != nil {
		return UpsertResponse{}, err
	}
	if err := e.maybeCompactLocked(); err != nil {
		return UpsertResponse{}, err
	}
	return UpsertResponse{
		OperationID:           req.OperationID,
		Namespace:             namespace,
		Upserted:              len(updated),
		ReasonCode:            ReasonCodeOK,
		LatencyMs:             time.Since(start).Milliseconds(),
		MemoryLifecycleAction: lifecycleAction,
	}, nil
}

func (e *FilesystemEngine) Delete(req DeleteRequest) (DeleteResponse, error) {
	start := time.Now()
	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		return DeleteResponse{}, &Error{
			Operation: OperationDelete,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerSemantic,
			Message:   "delete namespace is required",
		}
	}
	if len(req.IDs) == 0 {
		return DeleteResponse{}, &Error{
			Operation: OperationDelete,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerSemantic,
			Message:   "delete ids are required",
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	lifecycleAction := LifecycleActionNone
	if expired := e.applyTTLLocked(time.Now().UTC()); expired > 0 {
		lifecycleAction = LifecycleActionTTL
	}
	space := e.records[namespace]
	if space == nil {
		space = map[string]Record{}
		e.records[namespace] = space
	}
	ids := make([]string, 0, len(req.IDs))
	seen := map[string]struct{}{}
	deleted := 0
	for _, raw := range req.IDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if _, ok := space[id]; ok {
			delete(space, id)
			deleted++
		}
	}
	e.lastSeq++
	entry := walEntry{
		Seq:       e.lastSeq,
		Op:        OperationDelete,
		Namespace: namespace,
		IDs:       ids,
	}
	if err := e.appendWALEntryLocked(entry); err != nil {
		return DeleteResponse{}, err
	}
	if scope := strings.ToLower(strings.TrimSpace(req.Scope)); scope != "" {
		lifecycleAction = LifecycleActionForget
	}
	if err := e.maintainIndexLocked(); err != nil {
		return DeleteResponse{}, err
	}
	if err := e.maybeCompactLocked(); err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{
		OperationID:           req.OperationID,
		Namespace:             namespace,
		Deleted:               deleted,
		ReasonCode:            ReasonCodeOK,
		LatencyMs:             time.Since(start).Milliseconds(),
		MemoryLifecycleAction: lifecycleAction,
	}, nil
}

func (e *FilesystemEngine) Compact() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.compactLocked()
}

func (e *FilesystemEngine) recover() error {
	if err := e.prepareSnapshotFiles(); err != nil {
		return err
	}
	if err := e.recoverSnapshot(); err != nil {
		return err
	}
	if err := e.recoverWAL(); err != nil {
		return err
	}
	return e.recoverIndex()
}

func (e *FilesystemEngine) prepareSnapshotFiles() error {
	snapshot := e.snapshotPath()
	next := e.snapshotNextPath()
	backup := e.snapshotBackupPath()
	if _, err := os.Stat(snapshot); err == nil {
		_ = os.Remove(next)
		_ = os.Remove(backup)
		return nil
	}
	if _, err := os.Stat(next); err == nil {
		if renameErr := os.Rename(next, snapshot); renameErr != nil {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeStorageUnavailable,
				Layer:     LayerStorage,
				Message:   "recover snapshot.next failed",
				Cause:     renameErr,
			}
		}
		return nil
	}
	if _, err := os.Stat(backup); err == nil {
		if renameErr := os.Rename(backup, snapshot); renameErr != nil {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeStorageUnavailable,
				Layer:     LayerStorage,
				Message:   "recover snapshot.bak failed",
				Cause:     renameErr,
			}
		}
	}
	return nil
}

func (e *FilesystemEngine) recoverSnapshot() error {
	raw, err := os.ReadFile(e.snapshotPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "read snapshot failed",
			Cause:     err,
		}
	}
	state := snapshotState{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "decode snapshot failed",
			Cause:     err,
		}
	}
	e.lastSeq = state.LastSeq
	for _, item := range state.Records {
		namespace := strings.TrimSpace(item.Namespace)
		id := strings.TrimSpace(item.ID)
		if namespace == "" || id == "" {
			continue
		}
		if e.records[namespace] == nil {
			e.records[namespace] = map[string]Record{}
		}
		e.records[namespace][id] = cloneRecord(item)
	}
	return nil
}

func (e *FilesystemEngine) recoverWAL() error {
	file, err := os.OpenFile(e.walPath(), os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "open WAL for replay failed",
			Cause:     err,
		}
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		entry := walEntry{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeStorageUnavailable,
				Layer:     LayerStorage,
				Message:   "decode WAL entry failed",
				Cause:     err,
			}
		}
		if entry.Seq <= e.lastSeq {
			continue
		}
		if err := e.applyWALEntry(entry); err != nil {
			return err
		}
		e.lastSeq = entry.Seq
	}
	if err := scanner.Err(); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "scan WAL failed",
			Cause:     err,
		}
	}
	return nil
}

func (e *FilesystemEngine) recoverIndex() error {
	indexPath := e.indexPath()
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return e.writeIndexState()
		}
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "read index failed",
			Cause:     err,
		}
	}
	state := indexState{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "decode index failed",
			Cause:     err,
		}
	}
	if needsIndexDriftRecovery(e.cfg, state, e.currentChecksum()) {
		e.recoveryDrift = true
		e.lastLifecycleAction = LifecycleActionRecoveryConsistencyDrift
		if err := e.recoverFromDrift(); err != nil {
			return err
		}
	}
	return nil
}

func needsIndexDriftRecovery(cfg FilesystemEngineConfig, state indexState, checksum string) bool {
	if strings.TrimSpace(state.Checksum) == "" {
		return true
	}
	if strings.TrimSpace(state.Checksum) != strings.TrimSpace(checksum) {
		return true
	}
	if strings.TrimSpace(state.SchemaVersion) != strings.TrimSpace(cfg.IndexSchemaVersion) {
		return true
	}
	if strings.TrimSpace(state.Profile) != strings.TrimSpace(cfg.Profile) {
		return true
	}
	if strings.TrimSpace(state.Model) != strings.TrimSpace(cfg.Model) {
		return true
	}
	if strings.TrimSpace(state.IndexPolicy) != strings.TrimSpace(cfg.Search.IndexUpdatePolicy) {
		return true
	}
	if strings.TrimSpace(state.RecoveryPolicy) != strings.TrimSpace(cfg.Search.DriftRecoveryPolicy) {
		return true
	}
	return false
}

func (e *FilesystemEngine) recoverFromDrift() error {
	if e.cfg.Search.DriftRecoveryPolicy == DriftRecoveryPolicyFullRebuild {
		return e.rebuildIndexFull()
	}
	if err := e.rebuildIndexIncremental(); err != nil {
		return e.rebuildIndexFull()
	}
	return nil
}

func (e *FilesystemEngine) rebuildIndexIncremental() error {
	return e.writeIndexState()
}

func (e *FilesystemEngine) rebuildIndexFull() error {
	return e.writeIndexState()
}

func (e *FilesystemEngine) writeIndexState() error {
	state := indexState{
		LastSeq:        e.lastSeq,
		Checksum:       e.currentChecksum(),
		Profile:        strings.TrimSpace(e.cfg.Profile),
		Model:          strings.TrimSpace(e.cfg.Model),
		SchemaVersion:  strings.TrimSpace(e.cfg.IndexSchemaVersion),
		IndexPolicy:    strings.TrimSpace(e.cfg.Search.IndexUpdatePolicy),
		RecoveryPolicy: strings.TrimSpace(e.cfg.Search.DriftRecoveryPolicy),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}
	indexFile, err := os.OpenFile(e.indexPath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write index state failed",
			Cause:     err,
		}
	}
	defer func() { _ = indexFile.Close() }()
	writer := bufio.NewWriter(indexFile)
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(state); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "encode index state failed",
			Cause:     err,
		}
	}
	if err := writer.Flush(); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write index state failed",
			Cause:     err,
		}
	}
	if err := indexFile.Sync(); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write index state failed",
			Cause:     err,
		}
	}
	return nil
}

func (e *FilesystemEngine) currentChecksum() string {
	state := snapshotState{
		LastSeq: e.lastSeq,
		Records: make([]Record, 0),
	}
	namespaces := make([]string, 0, len(e.records))
	for namespace := range e.records {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	for _, namespace := range namespaces {
		ids := make([]string, 0, len(e.records[namespace]))
		for id := range e.records[namespace] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			state.Records = append(state.Records, cloneRecord(e.records[namespace][id]))
		}
	}
	raw, err := json.Marshal(state)
	if err != nil {
		sum := sha1.Sum([]byte(fmt.Sprintf("%d:%d", e.lastSeq, len(state.Records))))
		return hex.EncodeToString(sum[:])
	}
	sum := sha1.Sum(raw)
	return hex.EncodeToString(sum[:])
}

func (e *FilesystemEngine) applyWALEntry(entry walEntry) error {
	namespace := strings.TrimSpace(entry.Namespace)
	if namespace == "" {
		return &Error{
			Operation: entry.Op,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "WAL entry namespace is empty",
		}
	}
	if e.records[namespace] == nil {
		e.records[namespace] = map[string]Record{}
	}
	switch strings.ToLower(strings.TrimSpace(entry.Op)) {
	case OperationUpsert:
		for _, record := range entry.Records {
			id := strings.TrimSpace(record.ID)
			if id == "" {
				continue
			}
			next := cloneRecord(record)
			next.Namespace = namespace
			e.records[namespace][id] = next
		}
	case OperationDelete:
		for _, rawID := range entry.IDs {
			id := strings.TrimSpace(rawID)
			if id == "" {
				continue
			}
			delete(e.records[namespace], id)
		}
	default:
		return &Error{
			Operation: entry.Op,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "WAL entry operation is unsupported",
		}
	}
	return nil
}

func (e *FilesystemEngine) appendWALEntryLocked(entry walEntry) error {
	if e.walFile == nil {
		return &Error{
			Operation: entry.Op,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "WAL file is not initialized",
		}
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return &Error{
			Operation: entry.Op,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "encode WAL entry failed",
			Cause:     err,
		}
	}
	if _, err := e.walFile.Write(append(raw, '\n')); err != nil {
		return &Error{
			Operation: entry.Op,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "append WAL entry failed",
			Cause:     err,
		}
	}
	e.opsSinceCompaction++
	e.pendingFsyncOps++
	batchSize := e.cfg.Compaction.FsyncBatchSize
	if batchSize <= 1 {
		return e.syncWALLocked(entry.Op)
	}
	if e.pendingFsyncOps >= batchSize {
		if err := e.syncWALLocked(entry.Op); err != nil {
			return err
		}
	}
	return nil
}

func (e *FilesystemEngine) syncWALLocked(operation string) error {
	if e.walFile == nil || e.pendingFsyncOps <= 0 {
		return nil
	}
	if err := e.walFile.Sync(); err != nil {
		return &Error{
			Operation: operation,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "sync WAL entry failed",
			Cause:     err,
		}
	}
	e.pendingFsyncOps = 0
	return nil
}

func (e *FilesystemEngine) applyTTLLocked(now time.Time) int {
	removed := 0
	for namespace := range e.records {
		for id, record := range e.records[namespace] {
			if e.isExpiredRecord(record, now) {
				delete(e.records[namespace], id)
				removed++
			}
		}
	}
	return removed
}

func (e *FilesystemEngine) isExpiredRecord(record Record, now time.Time) bool {
	if !e.cfg.Lifecycle.TTLEnabled || e.cfg.Lifecycle.TTL <= 0 {
		return false
	}
	if record.UpdatedAt.IsZero() {
		return false
	}
	expiredBefore := now.Add(-e.cfg.Lifecycle.TTL)
	return record.UpdatedAt.Before(expiredBefore)
}

func (e *FilesystemEngine) applyRetentionLocked(namespace string) []string {
	if e.cfg.Lifecycle.RetentionDays <= 0 {
		return nil
	}
	space := e.records[namespace]
	if len(space) <= e.cfg.Lifecycle.RetentionDays {
		return nil
	}
	records := make([]Record, 0, len(space))
	for _, record := range space {
		records = append(records, cloneRecord(record))
	}
	sort.Slice(records, func(i, j int) bool {
		if !records[i].UpdatedAt.Equal(records[j].UpdatedAt) {
			return records[i].UpdatedAt.After(records[j].UpdatedAt)
		}
		return records[i].ID < records[j].ID
	})
	trimmed := make([]string, 0)
	for i := e.cfg.Lifecycle.RetentionDays; i < len(records); i++ {
		id := strings.TrimSpace(records[i].ID)
		if id == "" {
			continue
		}
		if _, ok := space[id]; ok {
			delete(space, id)
			trimmed = append(trimmed, id)
		}
	}
	sort.Strings(trimmed)
	return trimmed
}

func (e *FilesystemEngine) maintainIndexLocked() error {
	switch strings.TrimSpace(e.cfg.Search.IndexUpdatePolicy) {
	case IndexUpdatePolicyIncremental:
		return e.writeIndexState()
	case IndexUpdatePolicyFullRebuildOnProfileDrift:
		return e.rebuildIndexFull()
	default:
		return e.writeIndexState()
	}
}

func (e *FilesystemEngine) maybeCompactLocked() error {
	if !e.cfg.Compaction.Enabled {
		return nil
	}
	if e.opsSinceCompaction < e.cfg.Compaction.MinOps {
		stat, err := e.walFile.Stat()
		if err != nil {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeStorageUnavailable,
				Layer:     LayerStorage,
				Message:   "stat WAL failed",
				Cause:     err,
			}
		}
		if stat.Size() < e.cfg.Compaction.MaxWALBytes {
			return nil
		}
	}
	return e.compactLocked()
}

func (e *FilesystemEngine) compactLocked() error {
	nextPath := e.snapshotNextPath()
	if err := e.writeSnapshotNextLocked(nextPath); err != nil {
		return err
	}

	snapshotPath := e.snapshotPath()
	backupPath := e.snapshotBackupPath()
	if _, err := os.Stat(snapshotPath); err == nil {
		_ = os.Remove(backupPath)
		if err := os.Rename(snapshotPath, backupPath); err != nil {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeStorageUnavailable,
				Layer:     LayerStorage,
				Message:   "rotate snapshot to backup failed",
				Cause:     err,
			}
		}
	}
	if err := os.Rename(nextPath, snapshotPath); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "promote snapshot.next failed",
			Cause:     err,
		}
	}
	_ = os.Remove(backupPath)

	if e.walFile != nil {
		_ = e.walFile.Close()
	}
	if err := os.WriteFile(e.walPath(), []byte{}, 0o600); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "truncate WAL failed",
			Cause:     err,
		}
	}
	walFile, err := os.OpenFile(e.walPath(), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o600)
	if err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "reopen WAL failed",
			Cause:     err,
		}
	}
	e.walFile = walFile
	e.pendingFsyncOps = 0
	e.opsSinceCompaction = 0
	return e.writeIndexState()
}

func (e *FilesystemEngine) writeSnapshotNextLocked(nextPath string) error {
	snapshotFile, err := os.OpenFile(nextPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write snapshot.next failed",
			Cause:     err,
		}
	}
	defer func() { _ = snapshotFile.Close() }()

	writer := bufio.NewWriter(snapshotFile)
	if _, err := writer.WriteString(`{"last_seq":`); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write snapshot.next failed",
			Cause:     err,
		}
	}
	if _, err := writer.WriteString(strconv.FormatInt(e.lastSeq, 10)); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write snapshot.next failed",
			Cause:     err,
		}
	}
	if _, err := writer.WriteString(`,"records":[`); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write snapshot.next failed",
			Cause:     err,
		}
	}

	first := true
	namespaces := make([]string, 0, len(e.records))
	for namespace := range e.records {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	for _, namespace := range namespaces {
		ids := make([]string, 0, len(e.records[namespace]))
		for id := range e.records[namespace] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			raw, marshalErr := json.Marshal(cloneRecord(e.records[namespace][id]))
			if marshalErr != nil {
				return &Error{
					Operation: OperationQuery,
					Code:      ReasonCodeStorageUnavailable,
					Layer:     LayerStorage,
					Message:   "encode snapshot failed",
					Cause:     marshalErr,
				}
			}
			if !first {
				if err := writer.WriteByte(','); err != nil {
					return &Error{
						Operation: OperationQuery,
						Code:      ReasonCodeStorageUnavailable,
						Layer:     LayerStorage,
						Message:   "write snapshot.next failed",
						Cause:     err,
					}
				}
			}
			first = false
			if _, err := writer.Write(raw); err != nil {
				return &Error{
					Operation: OperationQuery,
					Code:      ReasonCodeStorageUnavailable,
					Layer:     LayerStorage,
					Message:   "write snapshot.next failed",
					Cause:     err,
				}
			}
		}
	}

	if _, err := writer.WriteString(`]}`); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write snapshot.next failed",
			Cause:     err,
		}
	}
	if err := writer.Flush(); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write snapshot.next failed",
			Cause:     err,
		}
	}
	if err := snapshotFile.Sync(); err != nil {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeStorageUnavailable,
			Layer:     LayerStorage,
			Message:   "write snapshot.next failed",
			Cause:     err,
		}
	}
	return nil
}

func (e *FilesystemEngine) snapshotPath() string {
	return filepath.Join(e.cfg.RootDir, snapshotFileName)
}

func (e *FilesystemEngine) snapshotNextPath() string {
	return filepath.Join(e.cfg.RootDir, snapshotNextFileName)
}

func (e *FilesystemEngine) snapshotBackupPath() string {
	return filepath.Join(e.cfg.RootDir, snapshotBackupFileName)
}

func (e *FilesystemEngine) walPath() string {
	return filepath.Join(e.cfg.RootDir, walFileName)
}

func (e *FilesystemEngine) indexPath() string {
	return filepath.Join(e.cfg.RootDir, indexFileName)
}

func normalizeFilesystemConfig(cfg FilesystemEngineConfig) FilesystemEngineConfig {
	out := cfg
	out.RootDir = strings.TrimSpace(out.RootDir)
	if out.Compaction.MinOps <= 0 {
		out.Compaction.MinOps = defaultCompactionMinOps
	}
	if out.Compaction.MaxWALBytes <= 0 {
		out.Compaction.MaxWALBytes = defaultCompactionMaxWALBytes
	}
	if out.Compaction.FsyncBatchSize <= 0 {
		out.Compaction.FsyncBatchSize = 1
	}
	if out.Lifecycle.RetentionDays <= 0 {
		out.Lifecycle.RetentionDays = 30
	}
	if out.Lifecycle.TTL <= 0 {
		out.Lifecycle.TTL = 7 * 24 * time.Hour
	}
	out.Lifecycle.ForgetScopeAllow = normalizeScopeList(out.Lifecycle.ForgetScopeAllow)
	if len(out.Lifecycle.ForgetScopeAllow) == 0 {
		out.Lifecycle.ForgetScopeAllow = []string{ScopeSession, ScopeProject, ScopeGlobal}
	}
	out.Search.IndexUpdatePolicy = strings.ToLower(strings.TrimSpace(out.Search.IndexUpdatePolicy))
	if out.Search.IndexUpdatePolicy == "" {
		out.Search.IndexUpdatePolicy = IndexUpdatePolicyIncremental
	}
	out.Search.DriftRecoveryPolicy = strings.ToLower(strings.TrimSpace(out.Search.DriftRecoveryPolicy))
	if out.Search.DriftRecoveryPolicy == "" {
		out.Search.DriftRecoveryPolicy = DriftRecoveryPolicyIncrementalThenFull
	}
	if strings.TrimSpace(out.IndexSchemaVersion) == "" {
		out.IndexSchemaVersion = indexSchemaVersionV2
	}
	out.Profile = strings.TrimSpace(out.Profile)
	if out.Profile == "" {
		out.Profile = ProfileGeneric
	}
	out.Model = strings.TrimSpace(out.Model)
	if out.Model == "" {
		out.Model = "generic"
	}
	return out
}

func matchRecord(record Record, req QueryRequest) bool {
	if len(req.IDs) > 0 {
		seen := false
		for _, raw := range req.IDs {
			id := strings.TrimSpace(raw)
			if id == "" {
				continue
			}
			if id == record.ID {
				seen = true
				break
			}
		}
		if !seen {
			return false
		}
	}
	if sessionID := strings.TrimSpace(req.SessionID); sessionID != "" && sessionID != record.SessionID {
		return false
	}
	if runID := strings.TrimSpace(req.RunID); runID != "" && runID != record.RunID {
		return false
	}
	query := strings.ToLower(strings.TrimSpace(req.Query))
	if query != "" {
		content := strings.ToLower(record.Content)
		if !strings.Contains(content, query) {
			return false
		}
	}
	return true
}

func cloneRecord(in Record) Record {
	out := in
	if in.Metadata == nil {
		out.Metadata = map[string]string{}
		return out
	}
	out.Metadata = make(map[string]string, len(in.Metadata))
	for key, value := range in.Metadata {
		out.Metadata[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func (e *FilesystemEngine) String() string {
	if e == nil {
		return "FilesystemEngine(<nil>)"
	}
	return fmt.Sprintf("FilesystemEngine(root=%s)", e.cfg.RootDir)
}
