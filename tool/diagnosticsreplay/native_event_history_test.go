package diagnosticsreplay

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func nativeHistoryFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "integration", "testdata", "diagnostics-replay", "native-event-history", "v1", "conformance.json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestNativeHistoryReplayReferenceFixture(t *testing.T) {
	raw := nativeHistoryFixture(t)
	first, err := ReplayNativeHistoryFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReplayNativeHistoryFixtureJSON(raw)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("replay drift: %v", err)
	}
	if len(first) != 10 {
		t.Fatalf("cases=%d", len(first))
	}
	wantErrors := map[string]string{"malformed-cursor": "cursor_malformed", "foreign-cursor": "cursor_foreign", "expired-cursor": "cursor_expired", "history-gap": "history_gap", "unsupported-source": "history_unsupported", "source-unavailable": "source_unavailable", "no-latest-fallback": "cursor_expired"}
	for _, result := range first {
		if want, ok := wantErrors[result.ID]; ok && result.Classification != want {
			t.Errorf("%s classification=%q want=%q", result.ID, result.Classification, want)
		}
	}
}

func TestNativeHistoryReplayRejectsSemanticDrift(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		mutate       func(*NativeHistoryFixture)
	}{
		{"identity", ReasonNativeHistoryIdentityDrift, func(f *NativeHistoryFixture) { f.Cases[2].Events[0].History.Source.ID = "other" }},
		{"callback-identity", ReasonNativeHistoryIdentityDrift, func(f *NativeHistoryFixture) { f.Cases[2].Events[0].History = nil }},
		{"order", ReasonNativeHistoryOrderDrift, func(f *NativeHistoryFixture) { f.Cases[2].Events[0].History.Sequence = 0 }},
		{"cursor", ReasonNativeHistoryCursorDrift, func(f *NativeHistoryFixture) { f.Cases[1].ConfirmedCursor.Value = "different" }},
		{"retention", ReasonNativeHistoryRetentionDrift, func(f *NativeHistoryFixture) { f.Cases[5].Floor = 10 }},
		{"side-effect", ReasonNativeHistorySideEffect, func(f *NativeHistoryFixture) { f.Cases[0].SideEffects = 1 }},
		{"latest-fallback", ReasonNativeHistoryUnsafeFallback, func(f *NativeHistoryFixture) { f.Cases[9].LatestFallback = true }},
		{"capability", ReasonNativeHistoryCapabilityDrift, func(f *NativeHistoryFixture) { f.Capabilities[0].Supported = true }},
		{"terminal", ReasonNativeHistoryTerminalDrift, func(f *NativeHistoryFixture) { f.Cases[2].Events[0].History.TerminalOutcome = "" }},
		{"invented-terminal", ReasonNativeHistoryTerminalDrift, func(f *NativeHistoryFixture) {
			f.Cases[2].Events[0].History.TerminalOutcome = "business_committed"
			f.Cases[2].ExpectedTerminal = "business_committed"
		}},
		{"terminal-type", ReasonNativeHistoryTerminalDrift, func(f *NativeHistoryFixture) { f.Cases[2].Events[0].NativeType = "mailbox.consume" }},
		{"error-page", ReasonNativeHistoryUnsafeFallback, func(f *NativeHistoryFixture) { f.Cases[5].PageNext = f.Cases[2].PageNext }},
		{"payload", ReasonNativeHistoryIdentityDrift, func(f *NativeHistoryFixture) {
			f.Cases[2].Events[0].Event.Payload = map[string]any{"body": "forbidden"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var fixture NativeHistoryFixture
			if err := json.Unmarshal(nativeHistoryFixture(t), &fixture); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&fixture)
			raw, err := json.Marshal(fixture)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ReplayNativeHistoryFixtureJSON(raw)
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Code != tc.reason {
				t.Fatalf("got %v, want %s", err, tc.reason)
			}
		})
	}
}

func TestNativeHistoryReplayRejectsMalformedFixtures(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte(`{"version":"other"}`), append(nativeHistoryFixture(t), []byte(` {}`)...), []byte(`{"unexpected":true}`)} {
		if _, err := ReplayNativeHistoryFixtureJSON(raw); err == nil {
			t.Fatal("accepted malformed fixture")
		}
	}
}

func TestNativeHistoryReplayAcceptsExpiryTransitionEndingInDLQ(t *testing.T) {
	var fixture NativeHistoryFixture
	if err := json.Unmarshal(nativeHistoryFixture(t), &fixture); err != nil {
		t.Fatal(err)
	}
	terminal := &fixture.Cases[2]
	terminal.Events[0].NativeType = "mailbox.expired"
	terminal.Events[0].History.TerminalOutcome = "dead_letter"
	terminal.ExpectedTerminal = "dead_letter"
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReplayNativeHistoryFixtureJSON(raw); err != nil {
		t.Fatalf("valid TTL-to-DLQ terminal evidence rejected: %v", err)
	}
}
