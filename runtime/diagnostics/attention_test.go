package diagnostics

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestAttentionMixedAndOwnerEvidence(t *testing.T) {
	request := AttentionRequest{
		Runs: []RunRecord{
			{RunID: "healthy", Status: "completed"},
			{RunID: "retrying", TerminalRetryable: true},
			{RunID: "capability", ProtocolCapabilityReason: "protocol.capability.missing_required"},
			{RunID: "recovery", RecoveryConflict: true},
			{RunID: "gate", GateDeniedCount: 1},
			{RunID: "effect", Status: "completed"},
		},
		Evidence: []AttentionEvidence{
			{RunID: "effect", Code: "effect-pending", Reference: "effect/ref"},
			{RunID: "source", Code: "source-required", Reference: "source/ref"},
			{RunID: "authorization", Code: "authorization-required", Reference: "authorization/ref"},
		},
	}
	before := fmt.Sprintf("%#v", request)
	got, err := ProjectAttention(request)
	if err != nil {
		t.Fatal(err)
	}
	want := AttentionResult{Total: 6, Items: []AttentionItem{
		{RunID: "authorization", Code: "authorization-required", NextAction: "revalidate_authorization", Owner: "host/authorization", Reference: "authorization/ref"},
		{RunID: "capability", Code: "unsupported", NextAction: "inspect_capability", Owner: "runtime/capability", Reference: "capability"},
		{RunID: "effect", Code: "effect-pending", NextAction: "reconcile_effect", Owner: "host/effect", Reference: "effect/ref"},
		{RunID: "gate", Code: "gate-failed", NextAction: "inspect_gate", Owner: "runtime/gate", Reference: "gate"},
		{RunID: "recovery", Code: "recovery-required", NextAction: "inspect_recovery", Owner: "runtime/recovery", Reference: "recovery"},
		{RunID: "source", Code: "source-required", NextAction: "inspect_source", Owner: "host/source", Reference: "source/ref"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if before != fmt.Sprintf("%#v", request) {
		t.Fatal("input mutated")
	}
	got.Items[0].Reference = "changed"
	if request.Evidence[2].Reference != "authorization/ref" {
		t.Fatal("result aliases input")
	}
}

func TestAttentionFiniteMappingAndUnknown(t *testing.T) {
	cases := []struct {
		name   string
		record RunRecord
		code   string
	}{
		{"unknown-provider-value", RunRecord{ProviderCapabilityOutcome: "unsupported"}, ""},
		{"unknown-protocol-value", RunRecord{ProtocolCapabilityReason: "unsupported_capability"}, ""},
		{"producer-provider", RunRecord{ProviderCapabilityOutcome: "blocked"}, "unsupported"},
		{"producer-protocol", RunRecord{ProtocolCapabilityReason: "protocol.capability.missing_required"}, "unsupported"},
		{"provider-degraded", RunRecord{ProviderCapabilityOutcome: "degraded"}, ""},
		{"protocol-downgraded", RunRecord{ProtocolCapabilityReason: "protocol.capability.optional_downgraded"}, ""},
		{"stream-conflict", RunRecord{StreamRecoveryTerminalConflictRecorded: true}, "recovery-required"},
		{"gate-timeout", RunRecord{GateTimeoutCount: 1}, "gate-failed"},
		{"healthy", RunRecord{Status: "completed"}, ""},
		{"retry", RunRecord{TerminalRetryable: true, TerminalResumable: true}, ""},
		{"unknown", RunRecord{ProtocolCapabilityReason: "future", ReasonCode: "private raw body", ErrorClass: "effect-pending"}, ""},
		{"substring", RunRecord{ProtocolCapabilityReason: "not_unsupported_capability"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.record.RunID = "run"
			got, err := ProjectAttention(AttentionRequest{Runs: []RunRecord{tc.record}})
			if err != nil {
				t.Fatal(err)
			}
			if tc.code == "" {
				if len(got.Items) != 0 {
					t.Fatalf("inferred %#v", got)
				}
				return
			}
			if len(got.Items) != 1 || got.Items[0].Code != tc.code {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

func TestAttentionDedupOrderLimitsAndParity(t *testing.T) {
	evidence := []AttentionEvidence{{RunID: "run", Code: "effect-pending", Reference: "b"}, {RunID: "run", Code: "effect-pending", Reference: "a"}, {RunID: "run", Code: "effect-pending", Reference: "a"}}
	got, err := ProjectAttention(AttentionRequest{Evidence: evidence, Limit: 1})
	if err != nil || got.Total != 2 || !got.Truncated || len(got.Items) != 1 || got.Items[0].Reference != "a" {
		t.Fatalf("%#v %v", got, err)
	}
	evidence[0], evidence[2] = evidence[2], evidence[0]
	again, err := ProjectAttention(AttentionRequest{Evidence: evidence, Limit: 1})
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("order drift %#v %v", again, err)
	}
	records := make([]RunRecord, 200)
	for i := range records {
		records[i] = RunRecord{RunID: fmt.Sprintf("run-%03d", i), RecoveryConflict: true}
	}
	def, err := ProjectAttention(AttentionRequest{Runs: records})
	if err != nil || len(def.Items) != 50 || def.Total != 200 || !def.Truncated {
		t.Fatalf("default %#v %v", def, err)
	}
	max, err := ProjectAttention(AttentionRequest{Runs: records, Limit: 200})
	if err != nil || len(max.Items) != 200 || max.Truncated {
		t.Fatalf("max %#v %v", max, err)
	}
	// Equivalent source evidence has identical read semantics in either execution form.
	run, err := ProjectAttention(AttentionRequest{Runs: []RunRecord{{RunID: "same", RecoveryConflict: true}}})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := ProjectAttention(AttentionRequest{Runs: []RunRecord{{RunID: "same", StreamRecoveryTerminalConflictRecorded: true}}})
	if err != nil || !reflect.DeepEqual(run, stream) {
		t.Fatalf("parity %#v %#v %v", run, stream, err)
	}
}

func TestAttentionInvalidIsAtomic(t *testing.T) {
	cases := []AttentionRequest{
		{Limit: -1}, {Limit: 201},
		{Runs: make([]RunRecord, 201)},
		{Evidence: make([]AttentionEvidence, 201)},
		{Runs: []RunRecord{{RunID: "same"}, {RunID: "same"}}},
		{Runs: []RunRecord{{RunID: ""}}},
		{Runs: []RunRecord{{RunID: strings.Repeat("a", 257)}}},
		{Runs: []RunRecord{{RunID: "secret\nbody"}}},
		{Evidence: []AttentionEvidence{{RunID: "run", Code: "unsupported", Reference: "ref"}}},
		{Evidence: []AttentionEvidence{{RunID: "run", Code: "effect-pending", Reference: ""}}},
		{Evidence: []AttentionEvidence{{RunID: "run", Code: "effect-pending", Reference: "中文"}}},
		{Evidence: []AttentionEvidence{{RunID: "run", Code: "effect-pending", Reference: "secret?token"}}},
	}
	for i, request := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			// A valid first candidate must never escape a later invalid entry.
			request.Runs = append([]RunRecord{{RunID: "valid", GateDeniedCount: 1}}, request.Runs...)
			got, err := ProjectAttention(request)
			if !errors.Is(err, ErrAttentionInput) || !reflect.DeepEqual(got, AttentionResult{}) {
				t.Fatalf("partial %#v err=%v", got, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("raw input leaked")
			}
		})
	}
	maxID := strings.Repeat("a", 256)
	if _, err := ProjectAttention(AttentionRequest{Evidence: []AttentionEvidence{{RunID: maxID, Code: "effect-pending", Reference: maxID}}}); err != nil {
		t.Fatal(err)
	}
}
