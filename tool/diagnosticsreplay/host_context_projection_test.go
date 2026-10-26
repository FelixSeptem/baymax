package diagnosticsreplay

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReplayHostContextProjectionFixtureIsDeterministicAndOffline(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "host_context_projection.v1.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	first, err := ReplayHostContextProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("first replay error = %v", err)
	}
	second, err := ReplayHostContextProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("second replay error = %v", err)
	}
	if len(first.Cases) != 16 || !first.Equal(second) {
		t.Fatalf("replay results = %#v / %#v, want deterministic sixteen-case matrix", first, second)
	}
	want := map[string]string{
		"valid":                        "",
		"malformed":                    "host_context.invalid_descriptor",
		"bounds":                       "host_context.limit_exceeded",
		"digest-drift":                 "host_context.digest_mismatch",
		"capability-expansion":         "host_context.capability_expansion",
		"plan-correlation-drift":       "host_context.invalid_plan_correlation",
		"authorization-revoked":        "host_context.resolver_denied",
		"source-reference-loss":        "host_context.source_not_found",
		"restore-rejected":             "host_context.source_not_found",
		"side-effect-before-rejection": "host_context.side_effect_before_rejection",
		"workflow-cross-step-binding":  "host_context.source_binding_mismatch",
		"team-foreign-member-binding":  "host_context.source_binding_mismatch",
		"workflow-stale-binding":       "host_context.source_binding_mismatch",
		"workflow-expired-binding":     "host_context.source_binding_expired",
		"mailbox-unsupported-owner":    "host_context.unsupported_capability",
		"duplicate-restore":            "host_context.duplicate_restore",
	}
	got := make(map[string]string, len(first.Cases))
	for _, item := range first.Cases {
		got[item.CaseID] = item.Reason
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replay classifications = %#v, want %#v", got, want)
	}
}

func TestReplayHostContextProjectionFixtureRejectsBodyAndRunStreamDrift(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want string
	}{
		{name: "fixture body", raw: []byte(`{"version":"host_context_projection_replay.v1","body":"private source body","cases":[]}`), want: "host_context.replay_schema_drift"},
		{name: "reference body", raw: []byte(`{"version":"host_context_projection_replay.v1","cases":[{"case_id":"privacy","descriptor":{"source_references":[{"reference":"opaque://source/body","digest":"d"}]},"run":"rejected","stream":"rejected"}]}`), want: "host_context.privacy_leakage"},
		{name: "Run Stream mismatch", raw: []byte(`{"version":"host_context_projection_replay.v1","cases":[{"case_id":"parity","run":"admitted","stream":"rejected"}]}`), want: "host_context.run_stream_mismatch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReplayHostContextProjectionFixtureJSON(tc.raw)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("ReplayHostContextProjectionFixtureJSON() error = %v, want %q", err, tc.want)
			}
		})
	}
}
