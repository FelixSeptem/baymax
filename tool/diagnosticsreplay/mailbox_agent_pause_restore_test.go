package diagnosticsreplay

import (
	"errors"
	"os"
	"testing"
)

func TestMailboxAgentPauseRestoreReplayFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/mailbox_agent_pause_restore.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReplayMailboxAgentPauseRestoreFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].Reason != "" || got[1].Reason != "" || got[2].Reason != "" || got[3].Reason != "mailbox_handler_safe_point_unsupported" {
		t.Fatalf("replay results = %#v", got)
	}
	if _, err := ReplayMailboxAgentPauseRestoreFixtureJSON([]byte(`{"version":"mailbox_agent_pause_restore.v1","cases":[{"case_id":"drift","pause_reason":"input_required","fresh_process_restore":true,"completed_effects_rerun":true,"duplicate_resume_count":1,"deployment_scope":"same_host_single_writer_sequential_reopen","terminal_outcome":"acked"}]}`)); err == nil {
		t.Fatal("effect replay drift unexpectedly accepted")
	} else {
		var validation *ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("drift error=%v", err)
		}
	}
}
