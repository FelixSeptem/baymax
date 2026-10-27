// Package history provides a source-owned, read-only journal primitive for
// execution owners that need retained native event replay.
package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
)

const cursorVersion = "native.history.cursor.v1"

type Option func(*Journal)

func WithRetention(entries int) Option {
	return func(j *Journal) {
		if entries > 0 && entries <= types.MaxHistoryPageSize {
			j.retention = entries
		}
	}
}

// WithProjectionLedger explicitly binds source commit evidence to a journal.
// It never creates a ledger implicitly.
func WithProjectionLedger(ledger *ProjectionLedger) Option {
	return func(j *Journal) {
		if j == nil || ledger == nil {
			return
		}
		if ledger.source != j.source {
			j.ledgerErr = types.ErrCursorForeign
			return
		}
		j.ledger = ledger
	}
}

type persisted struct {
	Version   string                       `json:"version"`
	Source    types.SourceRef              `json:"source"`
	Sequence  uint64                       `json:"sequence"`
	Floor     uint64                       `json:"floor"`
	Events    []types.NativeExecutionEvent `json:"events"`
	Retention int                          `json:"retention"`
}

type Journal struct {
	mu        sync.RWMutex
	path      string
	source    types.SourceRef
	sequence  uint64
	floor     uint64
	retention int
	events    []types.NativeExecutionEvent
	ledger    *ProjectionLedger
	ledgerErr error
}

func New(source types.SourceRef, options ...Option) (*Journal, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	}
	j := &Journal{source: source, retention: types.MaxHistoryPageSize}
	for _, option := range options {
		if option != nil {
			option(j)
		}
	}
	return j, nil
}

func OpenFile(path string, source types.SourceRef, options ...Option) (*Journal, error) {
	if path == "" {
		return nil, types.ErrSourceUnavailable
	}
	j, err := New(source, options...)
	if err != nil {
		return nil, err
	}
	j.path = filepath.Clean(path)
	raw, err := os.ReadFile(j.path)
	if os.IsNotExist(err) {
		return j, nil
	}
	if err != nil {
		return nil, types.ErrSourceUnavailable
	}
	var stored persisted
	if json.Unmarshal(raw, &stored) != nil || stored.Source != source || stored.Version != cursorVersion || stored.Retention <= 0 {
		return nil, types.ErrSourceUnavailable
	}
	if len(stored.Events) > types.MaxHistoryPageSize || stored.Floor > stored.Sequence {
		return nil, types.ErrSourceUnavailable
	}
	j.sequence, j.floor, j.retention, j.events = stored.Sequence, stored.Floor, stored.Retention, stored.Events
	if err := j.validateLocked(); err != nil {
		return nil, types.ErrSourceUnavailable
	}
	return j, nil
}

func (j *Journal) Append(ctx context.Context, event types.NativeExecutionEvent) (types.NativeHistoryIdentity, error) {
	if err := ctx.Err(); err != nil {
		return types.NativeHistoryIdentity{}, err
	}
	if j == nil {
		return types.NativeHistoryIdentity{}, types.ErrSourceUnavailable
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.ledgerErr != nil {
		return types.NativeHistoryIdentity{}, j.ledgerErr
	}
	previousSequence, previousFloor := j.sequence, j.floor
	previousEvents := append([]types.NativeExecutionEvent(nil), j.events...)
	j.sequence++
	identity := types.NativeHistoryIdentity{Source: j.source, Sequence: j.sequence, EventID: eventID(j.source, j.sequence)}
	if event.IsTerminal() {
		identity.TerminalOutcome = event.SourceState()
	}
	// Retained history is a bounded source fact, not an application payload
	// store. Keep native identity and terminal references while dropping the
	// potentially sensitive or effect-bearing event body before persistence.
	event.Event.Payload = nil
	event.History = &identity
	event.Event.EventID = identity.EventID
	event.NativeVersion = j.source.Namespace
	event.Event.Source = protocolSource(j.source.Kind)
	event.Capabilities.Identity = "source_owned"
	event.Capabilities.Ordering = "source_sequence"
	event.Capabilities.Cursor = true
	event.Capabilities.DurableHistory = j.path != ""
	event.Capabilities.Deduplication = true
	event.Capabilities.GapVerification = true
	snapshot, err := (types.NativeEventPage{Events: []types.NativeExecutionEvent{event}}).Snapshot()
	if err != nil {
		j.sequence--
		return types.NativeHistoryIdentity{}, err
	}
	j.events = append(j.events, snapshot.Events[0])
	if len(j.events) > j.retention {
		j.floor = j.events[0].History.Sequence
		j.events = append([]types.NativeExecutionEvent(nil), j.events[1:]...)
	}
	if err := j.persistLocked(); err != nil {
		j.sequence, j.floor, j.events = previousSequence, previousFloor, previousEvents
		return types.NativeHistoryIdentity{}, err
	}
	if j.ledger != nil {
		if _, err := j.ledger.commitRetained(ctx, snapshot.Events[0]); err != nil {
			j.ledgerErr = fmt.Errorf("%w: %v", types.ErrProjectionRecoveryRequired, err)
			return identity, j.ledgerErr
		}
	}
	return identity, nil
}

// ReconcileProjection repairs the explicitly bound projection ledger from a
// retained source fact. It never creates a store or invokes execution.
func (j *Journal) ReconcileProjection(ctx context.Context, event types.NativeExecutionEvent) (types.ProjectionRecoveryResult, error) {
	if err := ctx.Err(); err != nil {
		return types.ProjectionRecoveryResult{}, err
	}
	if j == nil {
		return types.ProjectionRecoveryResult{}, types.ErrSourceUnavailable
	}
	j.mu.RLock()
	ledger := j.ledger
	ledgerErr := j.ledgerErr
	if ledger == nil {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, types.ErrHistoryUnsupported
	}
	if ledgerErr != nil && !errors.Is(ledgerErr, types.ErrProjectionRecoveryRequired) {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, ledgerErr
	}
	if event.History == nil {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, types.ErrHistoryInvalid
	}
	if event.History.Source != j.source {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, types.ErrCursorForeign
	}
	if event.History.Source.Validate() != nil || event.History.Sequence == 0 {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, types.ErrHistoryInvalid
	}
	if event.History.EventID != eventID(j.source, event.History.Sequence) || event.Event.EventID != event.History.EventID {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, types.ErrHistoryConflict
	}
	var sourceFact *types.NativeExecutionEvent
	for i := range j.events {
		candidate := &j.events[i]
		if candidate.History != nil && candidate.History.EventID == event.History.EventID {
			sourceFact = candidate
			break
		}
	}
	if sourceFact == nil {
		sequence := event.History.Sequence
		switch {
		case sequence <= j.floor:
			j.mu.RUnlock()
			return types.ProjectionRecoveryResult{}, types.ErrCursorExpired
		case sequence > j.sequence:
			j.mu.RUnlock()
			return types.ProjectionRecoveryResult{}, types.ErrHistoryGap
		default:
			j.mu.RUnlock()
			return types.ProjectionRecoveryResult{}, types.ErrHistoryConflict
		}
	}
	left, leftErr := (types.NativeEventPage{Events: []types.NativeExecutionEvent{*sourceFact}}).Snapshot()
	right, rightErr := (types.NativeEventPage{Events: []types.NativeExecutionEvent{event}}).Snapshot()
	if leftErr != nil || rightErr != nil {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, types.ErrHistoryInvalid
	}
	if !reflect.DeepEqual(left.Events[0], right.Events[0]) {
		j.mu.RUnlock()
		return types.ProjectionRecoveryResult{}, types.ErrHistoryConflict
	}
	j.mu.RUnlock()
	result, err := ledger.reconcileSourceEvent(ctx, event)
	if err == nil {
		j.mu.Lock()
		if errors.Is(j.ledgerErr, types.ErrProjectionRecoveryRequired) {
			j.ledgerErr = nil
		}
		j.mu.Unlock()
	}
	return result, err
}

func protocolSource(kind types.HistorySourceKind) types.ProtocolSource {
	switch kind {
	case types.HistorySourceRunner:
		return types.ProtocolSourceRunner
	case types.HistorySourceChild:
		return types.ProtocolSourceComposer
	case types.HistorySourceTeam:
		return types.ProtocolSourceTeams
	case types.HistorySourceWorkflow:
		return types.ProtocolSourceWorkflow
	default:
		return types.ProtocolSource("mailbox")
	}
}

// Source returns the immutable source identity owned by this journal.
func (j *Journal) Source() types.SourceRef {
	if j == nil {
		return types.SourceRef{}
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.source
}

func (j *Journal) Capabilities(ctx context.Context, source types.SourceRef) (types.HistoryCapabilities, error) {
	if err := ctx.Err(); err != nil {
		return types.HistoryCapabilities{}, err
	}
	if j == nil {
		return types.HistoryCapabilities{}, types.ErrSourceUnavailable
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	if source != j.source {
		return types.HistoryCapabilities{}, types.ErrCursorForeign
	}
	floor := encodeCursor(source, j.floor)
	deployment := types.SameHostSingleWriterScope(source.ID, "fixture:deployment-scope.v1")
	caps := types.HistoryCapabilities{Source: source, EventIdentityNamespace: string(source.Kind) + ".event.history.v1", SequenceDefined: true, SequenceScope: "source_instance", RetentionWindow: "last bounded source events", RetentionFloor: &floor, CursorScope: "source_instance", CursorVersion: cursorVersion, Durable: j.path != "", TerminalRecovery: "retained source facts only", Supported: true, Deployment: deployment}
	if j.ledger != nil && j.path != "" && j.ledger.path != "" {
		caps.ProjectionRecovery = true
		caps.ProjectionRecoveryEvidence = "fixture:source-history-projection-transaction.v1"
	} else {
		caps.ProjectionRecoveryReason = "source commit/projection ledger is not durably configured"
	}
	return caps, nil
}

func (j *Journal) ReadAfter(ctx context.Context, source types.SourceRef, cursor *types.OpaqueCursor, limit int) (types.NativeEventPage, error) {
	if err := ctx.Err(); err != nil {
		return types.NativeEventPage{}, err
	}
	if err := types.ValidateHistoryLimit(limit); err != nil {
		return types.NativeEventPage{}, err
	}
	if j == nil {
		return types.NativeEventPage{}, types.ErrSourceUnavailable
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	if source != j.source {
		return types.NativeEventPage{}, types.ErrCursorForeign
	}
	after := j.floor
	if cursor != nil {
		decoded, err := decodeCursor(cursor)
		if err != nil {
			return types.NativeEventPage{}, err
		}
		if decoded.Source != source {
			return types.NativeEventPage{}, types.ErrCursorForeign
		}
		after = decoded.After
	}
	page := types.NativeEventPage{RetentionFloor: ptrCursor(encodeCursor(source, j.floor))}
	if after < j.floor {
		return page, types.ErrCursorExpired
	}
	if after > j.sequence {
		return page, types.ErrHistoryGap
	}
	for _, event := range j.events {
		if event.History == nil || event.History.Sequence <= after {
			continue
		}
		if len(page.Events) == limit {
			break
		}
		page.Events = append(page.Events, event)
		after = event.History.Sequence
	}
	page.NextCursor = ptrCursor(encodeCursor(source, after))
	page.Complete = after == j.sequence
	return page.Snapshot()
}

func (j *Journal) Snapshot(ctx context.Context) (persisted, error) {
	if err := ctx.Err(); err != nil {
		return persisted{}, err
	}
	if j == nil {
		return persisted{}, types.ErrSourceUnavailable
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	return persisted{Version: cursorVersion, Source: j.source, Sequence: j.sequence, Floor: j.floor, Retention: j.retention, Events: append([]types.NativeExecutionEvent(nil), j.events...)}, nil
}

func (j *Journal) persistLocked() error {
	if j.path == "" {
		return nil
	}
	stored := persisted{Version: cursorVersion, Source: j.source, Sequence: j.sequence, Floor: j.floor, Retention: j.retention, Events: j.events}
	raw, err := json.Marshal(stored)
	if err != nil {
		return types.ErrSourceUnavailable
	}
	tmp := j.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return types.ErrSourceUnavailable
	}
	if err := os.Rename(tmp, j.path); err != nil {
		_ = os.Remove(tmp)
		return types.ErrSourceUnavailable
	}
	return nil
}

func (j *Journal) validateLocked() error {
	previous := j.floor
	for _, event := range j.events {
		if event.History == nil || event.History.Source != j.source || event.History.Sequence != previous+1 || event.History.EventID != eventID(j.source, event.History.Sequence) {
			return types.ErrHistoryGap
		}
		previous = event.History.Sequence
	}
	if previous != j.sequence && (len(j.events) != 0 || j.sequence != j.floor) {
		return types.ErrHistoryGap
	}
	return nil
}

type cursor struct {
	Version string          `json:"version"`
	Source  types.SourceRef `json:"source"`
	After   uint64          `json:"after"`
}

func encodeCursor(source types.SourceRef, after uint64) types.OpaqueCursor {
	raw, _ := json.Marshal(cursor{Version: cursorVersion, Source: source, After: after})
	return types.OpaqueCursor{Value: base64.RawURLEncoding.EncodeToString(raw)}
}

func decodeCursor(value *types.OpaqueCursor) (cursor, error) {
	if value == nil {
		return cursor{}, types.ErrCursorMalformed
	}
	if err := value.Validate(); err != nil {
		return cursor{}, err
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(value.Value)
	if err != nil {
		return cursor{}, types.ErrCursorMalformed
	}
	var decoded cursor
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&decoded) != nil || decoded.Version != cursorVersion || decoded.Source.Validate() != nil {
		return cursor{}, types.ErrCursorMalformed
	}
	if encodeCursor(decoded.Source, decoded.After).Value != value.Value {
		return cursor{}, types.ErrCursorMalformed
	}
	return decoded, nil
}

func ptrCursor(value types.OpaqueCursor) *types.OpaqueCursor { return &value }

func eventID(source types.SourceRef, sequence uint64) string {
	h := sha256.Sum256([]byte(string(source.Kind) + "\x00" + source.Namespace + "\x00" + source.ID))
	return "native-history.v1/" + hex.EncodeToString(h[:8]) + "/" + formatSequence(sequence)
}

func formatSequence(sequence uint64) string {
	return base64.RawURLEncoding.EncodeToString([]byte{byte(sequence >> 56), byte(sequence >> 48), byte(sequence >> 40), byte(sequence >> 32), byte(sequence >> 24), byte(sequence >> 16), byte(sequence >> 8), byte(sequence)})
}
