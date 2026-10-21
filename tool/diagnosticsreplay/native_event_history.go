package diagnosticsreplay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"unicode/utf8"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	NativeHistoryFixtureVersion        = "native_event_history.v1"
	ReasonNativeHistoryIdentityDrift   = "native_event_history.identity_drift"
	ReasonNativeHistoryOrderDrift      = "native_event_history.order_drift"
	ReasonNativeHistoryCursorDrift     = "native_event_history.cursor_drift"
	ReasonNativeHistoryRetentionDrift  = "native_event_history.retention_drift"
	ReasonNativeHistorySideEffect      = "native_event_history.side_effect"
	ReasonNativeHistoryUnsafeFallback  = "native_event_history.unsafe_latest_fallback"
	ReasonNativeHistoryCapabilityDrift = "native_event_history.capability_drift"
	ReasonNativeHistoryTerminalDrift   = "native_event_history.terminal_drift"
)

// NativeHistoryFixture is reference-only evidence, not a runtime store. Cursor
// positions/scopes are recorded source facts; replay never decodes opaque bytes
// or asserts that an unsigned fixture authenticates those facts.
type NativeHistoryFixture struct {
	Version      string                            `json:"version"`
	Capabilities []NativeHistoryCapabilityEvidence `json:"capabilities"`
	Cases        []NativeHistoryCase               `json:"cases"`
}
type NativeHistoryCapabilityEvidence struct {
	Kind          types.HistorySourceKind `json:"kind"`
	Supported     bool                    `json:"supported"`
	Durable       bool                    `json:"durable"`
	AfterCursor   bool                    `json:"after_cursor"`
	SequenceScope string                  `json:"sequence_scope"`
	Reason        string                  `json:"reason,omitempty"`
}
type NativeHistoryCase struct {
	ID               string                       `json:"id"`
	Source           types.SourceRef              `json:"source"`
	Cursor           *types.OpaqueCursor          `json:"cursor,omitempty"`
	ConfirmedCursor  *types.OpaqueCursor          `json:"confirmed_cursor,omitempty"`
	CursorSource     *types.SourceRef             `json:"cursor_source,omitempty"`
	MalformedCursor  bool                         `json:"malformed_cursor,omitempty"`
	After            uint64                       `json:"after"`
	Floor            uint64                       `json:"floor"`
	HighWater        uint64                       `json:"high_water"`
	MissingSequence  bool                         `json:"missing_sequence,omitempty"`
	Unavailable      bool                         `json:"unavailable,omitempty"`
	Events           []types.NativeExecutionEvent `json:"events,omitempty"`
	PageNext         *types.OpaqueCursor          `json:"page_next,omitempty"`
	NextPosition     uint64                       `json:"next_position"`
	Complete         bool                         `json:"complete"`
	Restart          bool                         `json:"restart"`
	LatestFallback   bool                         `json:"latest_fallback"`
	SideEffects      int                          `json:"side_effects"`
	ExpectedError    string                       `json:"expected_error,omitempty"`
	ExpectedTerminal string                       `json:"expected_terminal,omitempty"`
	Configured       bool                         `json:"configured,omitempty"`
}
type NativeHistoryReplayResult struct{ ID, Classification, Digest string }

// ReplayNativeHistoryFixtureJSON checks recorded capability/order/cursor/terminal
// evidence. It has no source reader, observer, recorder or execution dependency.
func ReplayNativeHistoryFixtureJSON(raw []byte) ([]NativeHistoryReplayResult, error) {
	if len(raw) == 0 || len(raw) > 4*1024*1024 || !utf8.Valid(raw) {
		return nil, nativeReplayError(ReasonNativeHistoryCapabilityDrift)
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	var f NativeHistoryFixture
	if d.Decode(&f) != nil || d.Decode(new(any)) != io.EOF || f.Version != NativeHistoryFixtureVersion || len(f.Capabilities) != 5 || len(f.Cases) == 0 || len(f.Cases) > 256 {
		return nil, nativeReplayError(ReasonNativeHistoryCapabilityDrift)
	}
	caps, err := nativeHistoryCapabilities(f.Capabilities)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(f.Cases))
	results := make([]NativeHistoryReplayResult, 0, len(f.Cases))
	for _, c := range f.Cases {
		if c.ID == "" || len(c.ID) > 256 || seen[c.ID] {
			return nil, nativeReplayError(ReasonNativeHistoryIdentityDrift)
		}
		seen[c.ID] = true
		classification, err := replayNativeHistoryCase(c, caps[c.Source.Kind])
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(c)
		if err != nil {
			return nil, nativeReplayError(ReasonNativeHistoryIdentityDrift)
		}
		sum := sha256.Sum256(encoded)
		results = append(results, NativeHistoryReplayResult{ID: c.ID, Classification: classification, Digest: hex.EncodeToString(sum[:])})
	}
	return results, nil
}

func nativeHistoryCapabilities(items []NativeHistoryCapabilityEvidence) (map[types.HistorySourceKind]NativeHistoryCapabilityEvidence, error) {
	caps := make(map[types.HistorySourceKind]NativeHistoryCapabilityEvidence, len(items))
	for _, c := range items {
		if _, ok := caps[c.Kind]; ok {
			return nil, nativeReplayError(ReasonNativeHistoryCapabilityDrift)
		}
		caps[c.Kind] = c
	}
	for _, k := range []types.HistorySourceKind{types.HistorySourceRunner, types.HistorySourceChild, types.HistorySourceTeam, types.HistorySourceWorkflow, types.HistorySourceMailbox} {
		c, ok := caps[k]
		durable := k == types.HistorySourceMailbox
		if !ok || !c.Supported || c.Durable != durable || c.SequenceScope != "source_instance" || !c.AfterCursor {
			return nil, nativeReplayError(ReasonNativeHistoryCapabilityDrift)
		}
	}
	return caps, nil
}

func replayNativeHistoryCase(c NativeHistoryCase, cap NativeHistoryCapabilityEvidence) (string, error) {
	if c.SideEffects != 0 {
		return "", nativeReplayError(ReasonNativeHistorySideEffect)
	}
	if c.LatestFallback {
		return "", nativeReplayError(ReasonNativeHistoryUnsafeFallback)
	}
	if c.Source.Validate() != nil {
		return "", nativeReplayError(ReasonNativeHistoryIdentityDrift)
	}
	if c.Floor > c.HighWater {
		return "", nativeReplayError(ReasonNativeHistoryRetentionDrift)
	}
	if c.Restart && (c.Cursor == nil || c.ConfirmedCursor == nil || *c.Cursor != *c.ConfirmedCursor) {
		return "", nativeReplayError(ReasonNativeHistoryCursorDrift)
	}
	reason := ""
	switch {
	case c.Cursor != nil && (c.Cursor.Validate() != nil || c.MalformedCursor):
		reason = "cursor_malformed"
	case c.Cursor != nil && c.CursorSource != nil && *c.CursorSource != c.Source:
		reason = "cursor_foreign"
	case !c.Configured && c.ID == "unsupported-source":
		reason = "history_unsupported"
	case c.Unavailable:
		reason = "source_unavailable"
	case c.Cursor != nil && c.After < c.Floor:
		reason = "cursor_expired"
	case c.MissingSequence || c.After > c.HighWater:
		reason = "history_gap"
	}
	if reason != c.ExpectedError {
		return "", nativeReplayError(ReasonNativeHistoryCursorDrift)
	}
	if reason != "" {
		if len(c.Events) != 0 || c.PageNext != nil || c.Complete {
			return "", nativeReplayError(ReasonNativeHistoryUnsafeFallback)
		}
		return reason, nil
	}
	if c.Cursor != nil && (c.CursorSource == nil || c.CursorSource.Validate() != nil) {
		return "", nativeReplayError(ReasonNativeHistoryCursorDrift)
	}
	if len(c.Events) > types.MaxHistoryPageSize {
		return "", nativeReplayError(ReasonNativeHistoryOrderDrift)
	}
	previous := c.After
	if c.Cursor == nil {
		previous = c.Floor
	}
	terminal := ""
	for _, e := range c.Events {
		if e.History == nil || e.History.Source != c.Source || e.History.EventID == "" || len(e.History.EventID) > types.MaxHistoryReferenceBytes || e.Event.Sequence != 0 {
			return "", nativeReplayError(ReasonNativeHistoryIdentityDrift)
		}
		if e.History.Sequence != previous+1 || e.History.Sequence > c.HighWater {
			return "", nativeReplayError(ReasonNativeHistoryOrderDrift)
		}
		previous = e.History.Sequence
		if e.Version != types.NativeExecutionEventVersionV1 || e.NativeVersion != c.Source.Namespace || e.Event.Source != protocolSourceForHistory(c.Source.Kind) {
			return "", nativeReplayError(ReasonNativeHistoryIdentityDrift)
		}
		if c.Source.Kind == types.HistorySourceMailbox && e.History.TerminalOutcome != "" {
			want := map[string]map[string]bool{
				"mailbox.ack":         {"acked": true},
				"mailbox.dead_letter": {"dead_letter": true},
				// TTL expiry can be the final expired state or the triggering
				// transition after the owner moves the record to its DLQ.
				"mailbox.expired": {"expired": true, "dead_letter": true},
			}[e.NativeType]
			if !want[e.History.TerminalOutcome] {
				return "", nativeReplayError(ReasonNativeHistoryTerminalDrift)
			}
			terminal = e.History.TerminalOutcome
		}
		// Offline evidence carries identity facts only. Source journal redaction
		// and full lifecycle coverage are checked in owner integration tests.
		if len(e.Event.Payload) != 0 || !reflect.DeepEqual(e.HostReferences, types.NativeExecutionHostReferences{}) {
			return "", nativeReplayError(ReasonNativeHistoryIdentityDrift)
		}
	}
	if terminal != c.ExpectedTerminal {
		return "", nativeReplayError(ReasonNativeHistoryTerminalDrift)
	}
	if c.PageNext == nil || c.PageNext.Validate() != nil || c.NextPosition != previous || c.Complete != (previous == c.HighWater) {
		return "", nativeReplayError(ReasonNativeHistoryCursorDrift)
	}
	return "", nil
}

func protocolSourceForHistory(kind types.HistorySourceKind) types.ProtocolSource {
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
