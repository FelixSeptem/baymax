package diagnosticsreplay

import (
	"strings"
	"testing"
)

func TestApplicationOutboxReplayIsReferenceOnly(t *testing.T) {
	raw := `{"version":"application_outbox_dispatch.v1","cases":[{"case_id":"accepted","descriptor_digest":"sha256:a","target":"mailbox","dispatch_accepted":true,"effect_committed":false,"outbox_confirmed":false},{"case_id":"drift","descriptor_digest":"sha256:b","target":"scheduler","dispatch_accepted":true,"effect_committed":true,"outbox_confirmed":true,"expected_reason":"dispatch_outbox_conflated"}]}`
	got, err := ReplayApplicationOutboxFixtureJSON([]byte(raw))
	if err != nil || len(got) != 2 || got[0].Reason != "" {
		t.Fatalf("replay: %#v %v", got, err)
	}
	if _, err = ReplayApplicationOutboxFixtureJSON([]byte(strings.Replace(raw, `"target":"mailbox"`, `"target":"unsupported"`, 1))); err == nil {
		t.Fatal("unsupported target accepted")
	}
	if _, err = ReplayApplicationOutboxFixtureJSON([]byte(raw + "{}")); err == nil {
		t.Fatal("trailing input accepted")
	}
}
