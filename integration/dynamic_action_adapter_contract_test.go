package integration

import (
	"encoding/json"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

type adapterPendingAction struct {
	ID            string
	SensitiveBody string
	Authorized    bool
}

type pendingActionAdapterFixture struct {
	actions  map[string]adapterPendingAction
	executed []string
}

func (f *pendingActionAdapterFixture) Register(action adapterPendingAction, runID, sessionID, callID string) types.DynamicActionReference {
	token := "application-action:" + action.ID
	if f.actions == nil {
		f.actions = map[string]adapterPendingAction{}
	}
	f.actions[token] = action
	return types.DynamicActionReference{Token: token, Kind: "application.prepare", Resumable: true, RunID: runID, SessionID: sessionID, Iteration: 1, CallID: callID, Source: "application.adapter", Digest: "digest-" + action.ID}
}

func (f *pendingActionAdapterFixture) Confirm(token string) error {
	action, ok := f.actions[token]
	if !ok || !action.Authorized {
		return nil
	}
	f.executed = append(f.executed, action.ID)
	return nil
}

func TestDynamicActionAdapterKeepsBusinessPayloadOutsideCheckpoint(t *testing.T) {
	fixture := &pendingActionAdapterFixture{}
	ref := fixture.Register(adapterPendingAction{ID: "application-42", SensitiveBody: "salary=secret", Authorized: true}, "run-adapter", "session-adapter", "call-prepare")
	checkpoint := types.RunCheckpoint{Version: types.DynamicActionResumeProtocolVersion, CheckpointID: "checkpoint-prepare", RunID: ref.RunID, SessionID: ref.SessionID, State: types.RunStateInputRequired, Mode: "run", Iteration: ref.Iteration, PendingAction: &ref, Digest: ref.Digest}
	if err := checkpoint.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || containsSensitive(string(raw), "salary=secret") {
		t.Fatalf("checkpoint copied adapter business payload: %s", raw)
	}
	if err := fixture.Confirm(ref.Token); err != nil {
		t.Fatal(err)
	}
	if len(fixture.executed) != 1 || fixture.executed[0] != "application-42" {
		t.Fatalf("adapter executor not invoked by adapter: %#v", fixture.executed)
	}
	if err := fixture.Confirm("application-action:missing"); err != nil {
		t.Fatal(err)
	}
	if len(fixture.executed) != 1 {
		t.Fatalf("unknown opaque token mutated executor: %#v", fixture.executed)
	}
}

func containsSensitive(raw, secret string) bool {
	return len(secret) > 0 && len(raw) >= len(secret) && stringContains(raw, secret)
}

func stringContains(raw, needle string) bool {
	for i := 0; i+len(needle) <= len(raw); i++ {
		if raw[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
