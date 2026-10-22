package types

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"unicode/utf8"
)

const (
	MaxHistoryReferenceBytes = 1024
	MaxHistoryCursorBytes    = 4096
	MaxHistoryPageSize       = 200
)

type HistorySourceKind string

const (
	HistorySourceRunner   HistorySourceKind = "runner"
	HistorySourceChild    HistorySourceKind = "child"
	HistorySourceTeam     HistorySourceKind = "team"
	HistorySourceWorkflow HistorySourceKind = "workflow"
	HistorySourceMailbox  HistorySourceKind = "mailbox"
)

var (
	ErrHistoryUnsupported = errors.New("native event history unsupported")
	ErrCursorMalformed    = errors.New("native event history cursor malformed")
	ErrCursorForeign      = errors.New("native event history cursor foreign")
	ErrCursorExpired      = errors.New("native event history cursor expired")
	ErrHistoryGap         = errors.New("native event history gap")
	ErrSourceUnavailable  = errors.New("native event history source unavailable")
	ErrHistoryInvalid     = errors.New("native event history input invalid")
	ErrHistoryConflict    = errors.New("native event history identity conflict")
)

// SourceRef identifies a source instance; it is not a substitute for a run ID.
type SourceRef struct {
	Kind      HistorySourceKind `json:"kind"`
	Namespace string            `json:"namespace"`
	ID        string            `json:"id"`
}

func (s SourceRef) Validate() error {
	switch s.Kind {
	case HistorySourceRunner, HistorySourceChild, HistorySourceTeam, HistorySourceWorkflow, HistorySourceMailbox:
	default:
		return ErrHistoryInvalid
	}
	for _, value := range []string{s.Namespace, s.ID} {
		if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || len(value) > MaxHistoryReferenceBytes {
			return ErrHistoryInvalid
		}
	}
	return nil
}

// OpaqueCursor must be interpreted only by its source-owned reader.
type OpaqueCursor struct {
	Value string `json:"value"`
}

func (c OpaqueCursor) Validate() error {
	if !utf8.ValidString(c.Value) || strings.TrimSpace(c.Value) == "" || len(c.Value) > MaxHistoryCursorBytes {
		return ErrCursorMalformed
	}
	return nil
}

type HistoryCapabilities struct {
	Source                 SourceRef       `json:"source"`
	EventIdentityNamespace string          `json:"event_identity_namespace"`
	SequenceDefined        bool            `json:"sequence_defined"`
	SequenceScope          string          `json:"sequence_scope"`
	RetentionWindow        string          `json:"retention_window"`
	RetentionFloor         *OpaqueCursor   `json:"retention_floor,omitempty"`
	CursorScope            string          `json:"cursor_scope"`
	CursorVersion          string          `json:"cursor_version"`
	Durable                bool            `json:"durable"`
	TerminalRecovery       string          `json:"terminal_recovery"`
	Supported              bool            `json:"supported"`
	UnsupportedReason      string          `json:"unsupported_reason,omitempty"`
	Deployment             DeploymentScope `json:"deployment,omitempty"`
}

// NativeHistoryIdentity is independent of callback EventID and protocol Sequence.
// Its sequence is ordered only within Source. TerminalOutcome describes the
// source lifecycle; a mailbox acknowledgement is not a runner/business commit.
type NativeHistoryIdentity struct {
	Source          SourceRef `json:"source"`
	Sequence        uint64    `json:"sequence"`
	EventID         string    `json:"event_id"`
	TerminalOutcome string    `json:"terminal_outcome,omitempty"`
}

type NativeEventPage struct {
	Events         []NativeExecutionEvent `json:"events"`
	NextCursor     *OpaqueCursor          `json:"next_cursor,omitempty"`
	RetentionFloor *OpaqueCursor          `json:"retention_floor,omitempty"`
	Complete       bool                   `json:"complete"`
}

func ValidateHistoryLimit(limit int) error {
	if limit <= 0 || limit > MaxHistoryPageSize {
		return ErrHistoryInvalid
	}
	return nil
}

// Snapshot returns detached event payloads and cursor values.
func (p NativeEventPage) Snapshot() (NativeEventPage, error) {
	if len(p.Events) > MaxHistoryPageSize {
		return NativeEventPage{}, ErrHistoryInvalid
	}
	for _, cursor := range []*OpaqueCursor{p.NextCursor, p.RetentionFloor} {
		if cursor != nil {
			if err := cursor.Validate(); err != nil {
				return NativeEventPage{}, err
			}
		}
	}
	for _, event := range p.Events {
		budget := 64 * 1024
		if err := nativeExecutionInspect(reflect.ValueOf(event), 0, &budget); err != nil {
			return NativeEventPage{}, ErrHistoryInvalid
		}
		if h := event.History; h != nil {
			if h.Source.Validate() != nil || h.Sequence == 0 || strings.TrimSpace(h.EventID) == "" || len(h.EventID) > MaxHistoryReferenceBytes || len(h.TerminalOutcome) > MaxHistoryReferenceBytes {
				return NativeEventPage{}, ErrHistoryInvalid
			}
		}
		encoded, err := json.Marshal(event)
		if err != nil || len(encoded) > 64*1024 {
			return NativeEventPage{}, ErrHistoryInvalid
		}
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return NativeEventPage{}, ErrHistoryInvalid
	}
	if len(encoded) > MaxHistoryPageSize*(64*1024) {
		return NativeEventPage{}, ErrHistoryInvalid
	}
	var out NativeEventPage
	d := json.NewDecoder(bytes.NewReader(encoded))
	d.UseNumber()
	if err := d.Decode(&out); err != nil {
		return NativeEventPage{}, ErrHistoryInvalid
	}
	return out, nil
}

// MergeNativeExecutionEvents performs projection-side overlap handling. It
// never contacts a source owner and accepts an overlap only when the complete
// source fact is identical; a reused identity with a changed fact fails closed.
func MergeNativeExecutionEvents(existing, incoming []NativeExecutionEvent) ([]NativeExecutionEvent, error) {
	merged := make([]NativeExecutionEvent, 0, len(existing)+len(incoming))
	byID := make(map[string]int, len(existing)+len(incoming))
	appendFact := func(event NativeExecutionEvent) error {
		if event.History == nil || event.History.Source.Validate() != nil || event.History.Sequence == 0 || strings.TrimSpace(event.History.EventID) == "" {
			return ErrHistoryInvalid
		}
		idx, ok := byID[event.History.EventID]
		if !ok {
			copy, err := (NativeEventPage{Events: []NativeExecutionEvent{event}}).Snapshot()
			if err != nil {
				return err
			}
			byID[event.History.EventID] = len(merged)
			merged = append(merged, copy.Events[0])
			return nil
		}
		candidate, err := (NativeEventPage{Events: []NativeExecutionEvent{event}}).Snapshot()
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(merged[idx], candidate.Events[0]) {
			return ErrHistoryConflict
		}
		return nil
	}
	for _, event := range existing {
		if err := appendFact(event); err != nil {
			return nil, err
		}
	}
	for _, event := range incoming {
		if err := appendFact(event); err != nil {
			return nil, err
		}
	}
	page, err := (NativeEventPage{Events: merged}).Snapshot()
	if err != nil {
		return nil, err
	}
	return page.Events, nil
}

type NativeEventHistoryReader interface {
	Capabilities(ctx context.Context, source SourceRef) (HistoryCapabilities, error)
	ReadAfter(ctx context.Context, source SourceRef, cursor *OpaqueCursor, limit int) (NativeEventPage, error)
}

// UnsupportedNativeEventHistoryReader audits source kinds with no retained journal.
type UnsupportedNativeEventHistoryReader struct{}

func (UnsupportedNativeEventHistoryReader) Capabilities(_ context.Context, source SourceRef) (HistoryCapabilities, error) {
	if err := source.Validate(); err != nil {
		return HistoryCapabilities{}, err
	}
	return HistoryCapabilities{Source: source, EventIdentityNamespace: string(source.Kind) + ".event.unsupported", SequenceDefined: false, SequenceScope: "none", RetentionWindow: "none", CursorScope: "none", CursorVersion: "none", Durable: false, TerminalRecovery: "none", Supported: false, UnsupportedReason: "source journal is not implemented for " + string(source.Kind), Deployment: DefaultDeploymentScope(string(source.Kind), "source journal is not configured")}, nil
}

func (UnsupportedNativeEventHistoryReader) ReadAfter(ctx context.Context, source SourceRef, cursor *OpaqueCursor, limit int) (NativeEventPage, error) {
	if _, err := (UnsupportedNativeEventHistoryReader{}).Capabilities(ctx, source); err != nil {
		return NativeEventPage{}, err
	}
	if err := ValidateHistoryLimit(limit); err != nil {
		return NativeEventPage{}, err
	}
	if cursor != nil {
		if err := cursor.Validate(); err != nil {
			return NativeEventPage{}, err
		}
	}
	return NativeEventPage{}, ErrHistoryUnsupported
}
