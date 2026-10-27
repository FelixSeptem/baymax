package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
)

const projectionLedgerVersion = "native.history.projection-ledger.v1"

// ProjectionLedger retains source-owned commit facts and consumer acknowledgements.
// It is explicitly configured and supports only same-host sequential reopen.
type ProjectionLedger struct {
	mu               sync.RWMutex
	path             string
	source           types.SourceRef
	sequence         uint64
	floor            uint64
	retention        int
	records          []types.SourceCommitRecord
	acknowledgements map[string]types.ProjectionAcknowledgementResult
}

type ProjectionOption func(*ProjectionLedger)

// WithProjectionRetention bounds retained commit evidence for an explicitly
// configured projection ledger. It does not change Journal retention.
func WithProjectionRetention(entries int) ProjectionOption {
	return func(l *ProjectionLedger) {
		if entries > 0 && entries <= types.MaxHistoryPageSize {
			l.retention = entries
		}
	}
}

type projectionLedgerPersisted struct {
	Version          string                                           `json:"version"`
	Source           types.SourceRef                                  `json:"source"`
	Sequence         uint64                                           `json:"sequence"`
	Floor            uint64                                           `json:"floor"`
	Retention        int                                              `json:"retention"`
	Records          []types.SourceCommitRecord                       `json:"records"`
	Acknowledgements map[string]types.ProjectionAcknowledgementResult `json:"acknowledgements"`
}

func NewProjectionLedger(source types.SourceRef, options ...ProjectionOption) (*ProjectionLedger, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	}
	ledger := &ProjectionLedger{source: source, retention: types.MaxHistoryPageSize, acknowledgements: map[string]types.ProjectionAcknowledgementResult{}}
	for _, option := range options {
		if option != nil {
			option(ledger)
		}
	}
	return ledger, nil
}

func OpenProjectionLedger(path string, source types.SourceRef, options ...ProjectionOption) (*ProjectionLedger, error) {
	if path == "" {
		return nil, types.ErrSourceUnavailable
	}
	ledger, err := NewProjectionLedger(source, options...)
	if err != nil {
		return nil, err
	}
	ledger.path = filepath.Clean(path)
	raw, err := os.ReadFile(ledger.path)
	if os.IsNotExist(err) {
		return ledger, nil
	}
	if err != nil {
		return nil, types.ErrSourceUnavailable
	}
	var stored projectionLedgerPersisted
	if json.Unmarshal(raw, &stored) != nil || stored.Version != projectionLedgerVersion || stored.Source != source || stored.Retention <= 0 || stored.Floor > stored.Sequence {
		return nil, types.ErrSourceUnavailable
	}
	ledger.sequence, ledger.floor, ledger.retention = stored.Sequence, stored.Floor, stored.Retention
	ledger.records = append([]types.SourceCommitRecord(nil), stored.Records...)
	if stored.Acknowledgements != nil {
		ledger.acknowledgements = stored.Acknowledgements
	}
	if err := ledger.validateLocked(); err != nil {
		return nil, types.ErrSourceUnavailable
	}
	return ledger, nil
}

func (l *ProjectionLedger) Commit(ctx context.Context, input types.SourceCommitInput) (types.SourceCommitRecord, error) {
	if err := ctx.Err(); err != nil {
		return types.SourceCommitRecord{}, err
	}
	if l == nil {
		return types.SourceCommitRecord{}, types.ErrSourceUnavailable
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	previous := l.snapshotLocked()
	l.sequence++
	identity := types.NativeHistoryIdentity{Source: l.source, Sequence: l.sequence, EventID: eventID(l.source, l.sequence)}
	event := input.Event
	event.Event.Payload = nil
	event.History = &identity
	event.Event.EventID = identity.EventID
	event.NativeVersion = l.source.Namespace
	event.Event.Source = protocolSource(l.source.Kind)
	event.Capabilities.Identity = "source_owned"
	event.Capabilities.Ordering = "source_sequence"
	event.Capabilities.Cursor = true
	event.Capabilities.DurableHistory = l.path != ""
	event.Capabilities.Deduplication = true
	event.Capabilities.GapVerification = true
	if input.TerminalEvidence != "" {
		identity.TerminalOutcome = input.TerminalEvidence
		event.History = &identity
	}
	page, err := (types.NativeEventPage{Events: []types.NativeExecutionEvent{event}}).Snapshot()
	if err != nil {
		l.restoreLocked(previous)
		return types.SourceCommitRecord{}, err
	}
	cursor := encodeCursor(l.source, identity.Sequence)
	record := types.SourceCommitRecord{Event: page.Events[0], Cursor: cursor, TerminalEvidence: input.TerminalEvidence}
	record.Digest, err = commitDigest(record)
	if err != nil {
		l.restoreLocked(previous)
		return types.SourceCommitRecord{}, types.ErrSourceUnavailable
	}
	l.records = append(l.records, record)
	if len(l.records) > l.retention {
		l.floor = l.records[0].Event.History.Sequence
		l.records = append([]types.SourceCommitRecord(nil), l.records[1:]...)
	}
	if err = l.persistLocked(); err != nil {
		l.restoreLocked(previous)
		return types.SourceCommitRecord{}, err
	}
	return record, nil
}

func (l *ProjectionLedger) commitRetained(ctx context.Context, event types.NativeExecutionEvent) (types.SourceCommitRecord, error) {
	if err := ctx.Err(); err != nil {
		return types.SourceCommitRecord{}, err
	}
	if l == nil {
		return types.SourceCommitRecord{}, types.ErrSourceUnavailable
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if event.History == nil || event.History.Source != l.source || event.History.Sequence != l.sequence+1 || event.History.EventID != eventID(l.source, event.History.Sequence) {
		return types.SourceCommitRecord{}, types.ErrHistoryConflict
	}
	page, err := (types.NativeEventPage{Events: []types.NativeExecutionEvent{event}}).Snapshot()
	if err != nil {
		return types.SourceCommitRecord{}, err
	}
	previous := l.snapshotLocked()
	record := types.SourceCommitRecord{Event: page.Events[0], Cursor: encodeCursor(l.source, event.History.Sequence), TerminalEvidence: event.History.TerminalOutcome}
	record.Digest, err = commitDigest(record)
	if err != nil {
		return types.SourceCommitRecord{}, types.ErrSourceUnavailable
	}
	l.sequence = event.History.Sequence
	l.records = append(l.records, record)
	if len(l.records) > l.retention {
		l.floor = l.records[0].Event.History.Sequence
		l.records = append([]types.SourceCommitRecord(nil), l.records[1:]...)
	}
	if err = l.persistLocked(); err != nil {
		l.restoreLocked(previous)
		return types.SourceCommitRecord{}, err
	}
	return record, nil
}

func (l *ProjectionLedger) reconcileSourceEvent(ctx context.Context, event types.NativeExecutionEvent) (types.ProjectionRecoveryResult, error) {
	if err := ctx.Err(); err != nil {
		return types.ProjectionRecoveryResult{}, err
	}
	if l == nil {
		return types.ProjectionRecoveryResult{}, types.ErrSourceUnavailable
	}
	if event.History == nil {
		return types.ProjectionRecoveryResult{}, types.ErrHistoryInvalid
	}
	if len(event.Event.Payload) != 0 {
		return types.ProjectionRecoveryResult{}, types.ErrHistoryInvalid
	}
	if event.History.Source.Validate() != nil || event.History.Sequence == 0 || event.History.Source != l.source {
		if event.History.Source != l.source {
			return types.ProjectionRecoveryResult{}, types.ErrCursorForeign
		}
		return types.ProjectionRecoveryResult{}, types.ErrHistoryInvalid
	}
	if event.History.EventID != eventID(l.source, event.History.Sequence) || event.Event.EventID != event.History.EventID {
		return types.ProjectionRecoveryResult{}, types.ErrHistoryConflict
	}
	page, err := (types.NativeEventPage{Events: []types.NativeExecutionEvent{event}}).Snapshot()
	if err != nil {
		return types.ProjectionRecoveryResult{}, err
	}
	event = page.Events[0]
	record := types.SourceCommitRecord{Event: event, Cursor: encodeCursor(l.source, event.History.Sequence), TerminalEvidence: event.History.TerminalOutcome}
	record.Digest, err = commitDigest(record)
	if err != nil {
		return types.ProjectionRecoveryResult{}, types.ErrSourceUnavailable
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.recordByEventIDLocked(event.History.EventID); ok {
		if existing.Digest != record.Digest || existing.Cursor != record.Cursor || existing.TerminalEvidence != record.TerminalEvidence || !reflect.DeepEqual(existing.Event, record.Event) {
			return types.ProjectionRecoveryResult{}, types.ErrHistoryConflict
		}
		return types.ProjectionRecoveryResult{Record: existing, Duplicate: true}, nil
	}
	sequence := event.History.Sequence
	if sequence < l.floor {
		return types.ProjectionRecoveryResult{}, types.ErrCursorExpired
	}
	if sequence != l.sequence+1 {
		return types.ProjectionRecoveryResult{}, types.ErrHistoryGap
	}
	previous := l.snapshotLocked()
	l.sequence = sequence
	l.records = append(l.records, record)
	if len(l.records) > l.retention {
		l.floor = l.records[0].Event.History.Sequence
		l.records = append([]types.SourceCommitRecord(nil), l.records[1:]...)
	}
	if err := l.persistLocked(); err != nil {
		l.restoreLocked(previous)
		return types.ProjectionRecoveryResult{}, err
	}
	return types.ProjectionRecoveryResult{Record: record}, nil
}

func (l *ProjectionLedger) AcknowledgeProjection(ctx context.Context, acknowledgement types.ProjectionAcknowledgement) (types.ProjectionAcknowledgementResult, error) {
	if err := ctx.Err(); err != nil {
		return types.ProjectionAcknowledgementResult{}, err
	}
	if l == nil {
		return types.ProjectionAcknowledgementResult{}, types.ErrSourceUnavailable
	}
	if err := acknowledgement.Validate(); err != nil {
		return types.ProjectionAcknowledgementResult{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if acknowledgement.Source != l.source {
		return types.ProjectionAcknowledgementResult{}, types.ErrCursorForeign
	}
	record, ok := l.recordByEventIDLocked(acknowledgement.EventID)
	if !ok || record.Digest != acknowledgement.Digest || record.Cursor != acknowledgement.NextCursor {
		return types.ProjectionAcknowledgementResult{}, types.ErrHistoryConflict
	}
	result := types.ProjectionAcknowledgementResult{Source: acknowledgement.Source, EventID: acknowledgement.EventID, Digest: acknowledgement.Digest, Reference: acknowledgement.Reference, NextCursor: acknowledgement.NextCursor}
	if previous, exists := l.acknowledgements[acknowledgement.EventID]; exists {
		if previous.Source != result.Source || previous.EventID != result.EventID || previous.Digest != result.Digest || previous.Reference != result.Reference || previous.NextCursor != result.NextCursor {
			return types.ProjectionAcknowledgementResult{}, types.ErrHistoryConflict
		}
		previous.Duplicate = true
		return previous, nil
	}
	previous := l.snapshotLocked()
	l.acknowledgements[acknowledgement.EventID] = result
	if err := l.persistLocked(); err != nil {
		l.restoreLocked(previous)
		return types.ProjectionAcknowledgementResult{}, err
	}
	return result, nil
}

func (l *ProjectionLedger) ReadCommittedAfter(ctx context.Context, cursor *types.OpaqueCursor, limit int) (types.SourceCommitPage, error) {
	if err := ctx.Err(); err != nil {
		return types.SourceCommitPage{}, err
	}
	if err := types.ValidateHistoryLimit(limit); err != nil {
		return types.SourceCommitPage{}, err
	}
	if l == nil {
		return types.SourceCommitPage{}, types.ErrSourceUnavailable
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	after := l.floor
	if cursor != nil {
		decoded, err := decodeCursor(cursor)
		if err != nil {
			return types.SourceCommitPage{}, err
		}
		if decoded.Source != l.source {
			return types.SourceCommitPage{}, types.ErrCursorForeign
		}
		after = decoded.After
	}
	page := types.SourceCommitPage{RetentionFloor: ptrCursor(encodeCursor(l.source, l.floor))}
	if after < l.floor {
		return page, types.ErrCursorExpired
	}
	if after > l.sequence {
		return page, types.ErrHistoryGap
	}
	for _, record := range l.records {
		if record.Event.History == nil || record.Event.History.Sequence <= after {
			continue
		}
		if len(page.Records) == limit {
			break
		}
		page.Records = append(page.Records, record)
		after = record.Event.History.Sequence
	}
	page.NextCursor = ptrCursor(encodeCursor(l.source, after))
	page.Complete = after == l.sequence
	return page, nil
}

func (l *ProjectionLedger) recordByEventIDLocked(eventID string) (types.SourceCommitRecord, bool) {
	for _, record := range l.records {
		if record.Event.History != nil && record.Event.History.EventID == eventID {
			return record, true
		}
	}
	return types.SourceCommitRecord{}, false
}

func (l *ProjectionLedger) snapshotLocked() projectionLedgerPersisted {
	acks := make(map[string]types.ProjectionAcknowledgementResult, len(l.acknowledgements))
	for key, value := range l.acknowledgements {
		acks[key] = value
	}
	return projectionLedgerPersisted{Version: projectionLedgerVersion, Source: l.source, Sequence: l.sequence, Floor: l.floor, Retention: l.retention, Records: append([]types.SourceCommitRecord(nil), l.records...), Acknowledgements: acks}
}

func (l *ProjectionLedger) restoreLocked(state projectionLedgerPersisted) {
	l.sequence, l.floor, l.retention = state.Sequence, state.Floor, state.Retention
	l.records = append([]types.SourceCommitRecord(nil), state.Records...)
	l.acknowledgements = state.Acknowledgements
}

func (l *ProjectionLedger) persistLocked() error {
	if l.path == "" {
		return nil
	}
	raw, err := json.Marshal(l.snapshotLocked())
	if err != nil {
		return types.ErrSourceUnavailable
	}
	tmp := l.path + ".tmp"
	if err = os.WriteFile(tmp, raw, 0o600); err != nil {
		return types.ErrSourceUnavailable
	}
	if err = os.Rename(tmp, l.path); err != nil {
		_ = os.Remove(tmp)
		return types.ErrSourceUnavailable
	}
	return nil
}

func (l *ProjectionLedger) validateLocked() error {
	if len(l.records) > types.MaxHistoryPageSize {
		return types.ErrHistoryGap
	}
	previous := l.floor
	for _, record := range l.records {
		if record.Event.History == nil || record.Event.History.Source != l.source || record.Event.History.Sequence != previous+1 || record.Event.History.EventID != eventID(l.source, previous+1) || record.Cursor != encodeCursor(l.source, previous+1) {
			return types.ErrHistoryGap
		}
		digest, err := commitDigest(record)
		if err != nil || digest != record.Digest {
			return types.ErrHistoryConflict
		}
		previous++
	}
	if previous != l.sequence && (len(l.records) != 0 || l.sequence != l.floor) {
		return types.ErrHistoryGap
	}
	for eventID, acknowledgement := range l.acknowledgements {
		record, ok := l.recordByEventIDLocked(eventID)
		if !ok || acknowledgement.Source != l.source || acknowledgement.EventID != eventID || acknowledgement.Digest != record.Digest || acknowledgement.NextCursor != record.Cursor || acknowledgement.Reference == "" {
			return types.ErrHistoryConflict
		}
	}
	return nil
}

func commitDigest(record types.SourceCommitRecord) (string, error) {
	copy := record
	copy.Digest = ""
	raw, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes.TrimSpace(raw))
	return hex.EncodeToString(sum[:]), nil
}
