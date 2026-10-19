package mailbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	HistoryVersion          = "mailbox.history.v1"
	HistoryRetentionEntries = 1024
)

// HistoryEntry is a reference-only lifecycle fact. It deliberately has no
// Envelope, Payload, LastError, callback body or host business-commit field.
type HistoryEntry struct {
	Sequence   uint64          `json:"sequence"`
	Transition string          `json:"transition"`
	Time       time.Time       `json:"time"`
	Outcome    HostPlanOutcome `json:"outcome"`
	Reclaimed  bool            `json:"reclaimed,omitempty"`
}

// HistoryJournal is persisted atomically with the existing mailbox snapshot.
// Floor is an after-position immediately preceding the first retained entry.
type HistoryJournal struct {
	Version     string         `json:"version"`
	SourceID    string         `json:"source_id"`
	Sequence    uint64         `json:"sequence"`
	Floor       uint64         `json:"floor"`
	Entries     []HistoryEntry `json:"entries"`
	Unavailable bool           `json:"unavailable,omitempty"`
}

func newHistoryJournal() HistoryJournal {
	return HistoryJournal{Version: HistoryVersion, SourceID: rand.Text()}
}

func (h HistoryJournal) clone() *HistoryJournal {
	h.Entries = append([]HistoryEntry(nil), h.Entries...)
	return &h
}

func (h HistoryJournal) source() types.SourceRef {
	return types.SourceRef{Kind: types.HistorySourceMailbox, Namespace: HistoryVersion, ID: h.SourceID}
}

func (h HistoryJournal) validate() error {
	if h.Version != HistoryVersion || h.source().Validate() != nil || h.Floor > h.Sequence || len(h.Entries) > HistoryRetentionEntries {
		return ErrSnapshotCorrupt
	}
	previous := h.Floor
	for _, entry := range h.Entries {
		if entry.Sequence <= previous || entry.Sequence > h.Sequence || !validHistoryEntry(entry) {
			return ErrSnapshotCorrupt
		}
		previous = entry.Sequence
	}
	return nil
}

func validHistoryEntry(entry HistoryEntry) bool {
	// Validate all strings before persistence. No body or free-form reason enters
	// the journal; oversized legacy identities make history unavailable instead.
	values := []string{entry.Transition, entry.Outcome.MessageID, entry.Outcome.RunID, entry.Outcome.DeliveryID, entry.Outcome.LeaseID, entry.Outcome.DeliveryLeaseID, entry.Outcome.AttemptCausation, entry.Outcome.ParentRunID, entry.Outcome.CausationID, entry.Outcome.AuthorityDigest, entry.Outcome.PlanDigest, entry.Outcome.WriteScope, entry.Outcome.EffectNamespace, entry.Outcome.Occurrence, entry.Outcome.HostCorrelation, entry.Outcome.LifecycleResult, entry.Outcome.LifecycleReason, entry.Outcome.TerminalOutcome}
	for _, v := range values {
		if !utf8.ValidString(v) || len(v) > types.MaxHistoryReferenceBytes {
			return false
		}
	}
	return entry.Outcome.MessageID != "" && !entry.Time.IsZero()
}

func (s *mailboxState) appendHistory(transition string, rec Record, reason string, opts ActionOptions) {
	if s.History.Unavailable {
		return
	}
	s.History.Sequence++
	binding := HostPlanBinding{}
	if rec.Envelope.HostPlanBinding != nil {
		binding = *rec.Envelope.HostPlanBinding
	}
	out := hostPlanOutcome(binding, PublishResult{Record: rec})
	if out.LifecycleReason != "" {
		out.LifecycleReason = CanonicalizeLifecycleReason(out.LifecycleReason, LifecycleReasonHandlerError)
	}
	if out.MessageID == "" {
		out.MessageID = rec.Envelope.MessageID
	}
	if out.RunID == "" {
		out.RunID = rec.Envelope.RunID
	}
	if out.HostCorrelation == "" {
		out.HostCorrelation = rec.Envelope.CorrelationID
	}
	if transition == "admit" {
		out.LifecycleResult = "admitted"
	}
	if reason != "" {
		out.LifecycleReason = CanonicalizeLifecycleReason(reason, LifecycleReasonHandlerError)
	}
	entry := HistoryEntry{Sequence: s.History.Sequence, Transition: transition, Time: rec.UpdatedAt, Outcome: out, Reclaimed: opts.Reclaimed}
	if !validHistoryEntry(entry) {
		s.History.Unavailable = true
		return
	}
	s.History.Entries = append(s.History.Entries, entry)
	if len(s.History.Entries) > HistoryRetentionEntries {
		s.History.Floor = s.History.Entries[0].Sequence
		s.History.Entries = append([]HistoryEntry(nil), s.History.Entries[1:]...)
	}
}

type historyCursor struct {
	Version string          `json:"version"`
	Source  types.SourceRef `json:"source"`
	After   uint64          `json:"after"`
}

func encodeHistoryCursor(source types.SourceRef, after uint64) *types.OpaqueCursor {
	raw, _ := json.Marshal(historyCursor{Version: HistoryVersion, Source: source, After: after})
	return &types.OpaqueCursor{Value: base64.RawURLEncoding.EncodeToString(raw)}
}

func decodeHistoryCursor(cursor *types.OpaqueCursor) (historyCursor, error) {
	if err := cursor.Validate(); err != nil {
		return historyCursor{}, err
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(cursor.Value)
	if err != nil {
		return historyCursor{}, types.ErrCursorMalformed
	}
	var out historyCursor
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&out) != nil || d.Decode(new(any)) != io.EOF || out.Version != HistoryVersion || out.Source.Validate() != nil {
		return historyCursor{}, types.ErrCursorMalformed
	}
	// Reject omitted/null/duplicate/noncanonical fields instead of interpreting
	// defaults as a valid beginning cursor. The serialized representation is opaque.
	if encodeHistoryCursor(out.Source, out.After).Value != cursor.Value {
		return historyCursor{}, types.ErrCursorMalformed
	}
	return out, nil
}

type historyStore interface {
	historySnapshot(context.Context) (HistoryJournal, bool, error)
}

func (m *Mailbox) history(ctx context.Context) (HistoryJournal, bool, error) {
	if err := ctx.Err(); err != nil {
		return HistoryJournal{}, false, err
	}
	if m == nil || m.store == nil {
		return HistoryJournal{}, false, types.ErrSourceUnavailable
	}
	store, ok := m.store.(historyStore)
	if !ok {
		return HistoryJournal{}, false, types.ErrHistoryUnsupported
	}
	h, durable, err := store.historySnapshot(ctx)
	if err != nil || h.Unavailable {
		return HistoryJournal{}, false, types.ErrSourceUnavailable
	}
	return h, durable, nil
}

func (m *Mailbox) HistorySource(ctx context.Context) (types.SourceRef, error) {
	h, _, err := m.history(ctx)
	if err != nil {
		return types.SourceRef{}, err
	}
	return h.source(), nil
}

func (m *Mailbox) Capabilities(ctx context.Context, source types.SourceRef) (types.HistoryCapabilities, error) {
	if err := source.Validate(); err != nil {
		return types.HistoryCapabilities{}, err
	}
	if source.Kind != types.HistorySourceMailbox {
		return (types.UnsupportedNativeEventHistoryReader{}).Capabilities(ctx, source)
	}
	h, durable, err := m.history(ctx)
	if err != nil {
		return types.HistoryCapabilities{}, err
	}
	if source != h.source() {
		return types.HistoryCapabilities{}, types.ErrCursorForeign
	}
	return types.HistoryCapabilities{Source: source, EventIdentityNamespace: HistoryVersion + "/" + source.ID, SequenceDefined: true, SequenceScope: "source_instance", RetentionWindow: "last 1024 lifecycle events; no time guarantee", RetentionFloor: encodeHistoryCursor(source, h.Floor), CursorScope: "source_instance", CursorVersion: HistoryVersion, Durable: durable, TerminalRecovery: "retained mailbox lifecycle only; no business commit", Supported: true}, nil
}

func (m *Mailbox) ReadAfter(ctx context.Context, source types.SourceRef, cursor *types.OpaqueCursor, limit int) (types.NativeEventPage, error) {
	if err := source.Validate(); err != nil {
		return types.NativeEventPage{}, err
	}
	if err := types.ValidateHistoryLimit(limit); err != nil {
		return types.NativeEventPage{}, err
	}
	var decoded historyCursor
	if cursor != nil {
		var err error
		decoded, err = decodeHistoryCursor(cursor)
		if err != nil {
			return types.NativeEventPage{}, err
		}
		if decoded.Source != source {
			return types.NativeEventPage{}, types.ErrCursorForeign
		}
	}
	if source.Kind != types.HistorySourceMailbox {
		return types.NativeEventPage{}, types.ErrHistoryUnsupported
	}
	h, durable, err := m.history(ctx)
	if err != nil {
		return types.NativeEventPage{}, err
	}
	if source != h.source() {
		return types.NativeEventPage{}, types.ErrCursorForeign
	}
	page := types.NativeEventPage{RetentionFloor: encodeHistoryCursor(source, h.Floor)}
	after := h.Floor
	if cursor != nil {
		after = decoded.After
	}
	if after < h.Floor {
		return page, types.ErrCursorExpired
	}
	if after > h.Sequence {
		return page, types.ErrHistoryGap
	}
	// Validate the entire retained interval, including the tail. A missing event
	// cannot become a plausible Complete page, even if it precedes the cursor.
	previous := h.Floor
	for _, entry := range h.Entries {
		if entry.Sequence != previous+1 {
			return page, types.ErrHistoryGap
		}
		previous = entry.Sequence
	}
	if previous != h.Sequence {
		return page, types.ErrHistoryGap
	}
	for _, entry := range h.Entries {
		if entry.Sequence <= after {
			continue
		}
		if len(page.Events) == limit {
			break
		}
		page.Events = append(page.Events, entry.native(source, durable))
		after = entry.Sequence
	}
	page.NextCursor = encodeHistoryCursor(source, after)
	page.Complete = after == h.Sequence
	return page.Snapshot()
}

func (entry HistoryEntry) native(source types.SourceRef, durable bool) types.NativeExecutionEvent {
	id := fmt.Sprintf("%s/%s/%d", HistoryVersion, source.ID, entry.Sequence)
	// HostPlanOutcome has a fixed, reference-only schema. Convert it without
	// adding application payload or mutable source records to the event.
	raw, _ := json.Marshal(entry.Outcome)
	payload := map[string]any{}
	_ = json.Unmarshal(raw, &payload)
	payload["reclaimed"] = entry.Reclaimed
	return types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "mailbox." + strings.TrimSpace(entry.Transition), NativeVersion: HistoryVersion, History: &types.NativeHistoryIdentity{Source: source, Sequence: entry.Sequence, EventID: id, TerminalOutcome: entry.Outcome.TerminalOutcome}, Event: types.EventEnvelope{EventID: id, RunID: entry.Outcome.RunID, CausationID: entry.Outcome.AttemptCausation, Source: types.ProtocolSource("mailbox"), Time: entry.Time, Payload: payload}, Capabilities: types.NativeExecutionCapabilities{Identity: "source_owned", Ordering: "source_sequence", Cursor: true, DurableHistory: durable, Deduplication: true, GapVerification: true}}
}

func (s *MemoryStore) historySnapshot(_ context.Context) (HistoryJournal, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return *s.state.History.clone(), false, nil
}

func (s *FileStore) historySnapshot(_ context.Context) (HistoryJournal, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingPersist != 0 || s.historyPersistFailed || !s.historyCommitted {
		return HistoryJournal{}, false, types.ErrSourceUnavailable
	}
	return *s.state.History.clone(), true, nil
}
