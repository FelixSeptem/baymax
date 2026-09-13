package diagnosticsreplay

import (
	"strings"
	"testing"
)

func TestParseCompletionSafePointOwnershipFixture(t *testing.T) {
	raw := []byte(`{"version":"completion_safe_point_ownership.v1","cases":[{"name":"accepted","run_id":"run-1","session_id":"session-1","task_id":"task-1","attempt_id":"attempt-1","mailbox_id":"msg-1","correlation_id":"corr-1","outcome":"accepted","run":{"status":"applied","boundary":"next_decision"},"stream":{"status":"applied","boundary":"next_decision"}}]}`)
	fixture, err := ParseCompletionSafePointOwnershipFixtureJSON(raw)
	if err != nil || len(fixture.Cases) != 1 {
		t.Fatalf("fixture=%#v err=%v", fixture, err)
	}
}

func TestParseCompletionSafePointOwnershipFixtureRejectsRawBodies(t *testing.T) {
	raw := []byte(`{"version":"completion_safe_point_ownership.v1","cases":[{"name":"privacy","run_id":"run-1","session_id":"session-1","task_id":"task-1","attempt_id":"attempt-1","mailbox_id":"msg-1","correlation_id":"corr-1","completion_body":"secret","reasoning":"private","outcome":"accepted","run":{"status":"applied"},"stream":{"status":"applied"}}]}`)
	_, err := ParseCompletionSafePointOwnershipFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeCompletionSafePointPrivacyDrift) {
		t.Fatalf("err=%v", err)
	}
}

func TestParseCompletionSafePointOwnershipFixtureRejectsParityDrift(t *testing.T) {
	raw := []byte(`{"version":"completion_safe_point_ownership.v1","cases":[{"name":"parity","run_id":"run-1","session_id":"session-1","task_id":"task-1","attempt_id":"attempt-1","mailbox_id":"msg-1","correlation_id":"corr-1","outcome":"accepted","run":{"status":"applied"},"stream":{"status":"not_applied"}}]}`)
	_, err := ParseCompletionSafePointOwnershipFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeCompletionSafePointParityDrift) {
		t.Fatalf("err=%v", err)
	}
}

func TestEvaluateCompletionSafePointOwnershipClassifiesDuplicateAndLate(t *testing.T) {
	raw := []byte(`{"version":"completion_safe_point_ownership.v1","cases":[{"name":"dup","run_id":"run-1","session_id":"s","task_id":"t","attempt_id":"a","mailbox_id":"m","correlation_id":"c","outcome":"duplicate","duplicate":true,"run":{"status":"duplicate"},"stream":{"status":"duplicate"}}]}`)
	fixture, err := EvaluateCompletionSafePointOwnershipFixtureJSON(raw)
	if err != nil || fixture.Cases[0].Outcome != "duplicate" { t.Fatalf("fixture=%#v err=%v", fixture, err) }
}
