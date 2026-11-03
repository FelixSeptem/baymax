package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAttentionOfflineExample(t *testing.T) {
	var output bytes.Buffer
	if err := demonstrate(&output); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"healthy_retrying_ignored", "effect_pending_explicit", "truncation_visible", "total=4 shown=2 truncated=true", "effect effect-pending reconcile_effect host/effect effect/ref", "gate gate-failed inspect_gate runtime/gate gate"} {
		if !strings.Contains(output.String(), marker) {
			t.Fatalf("missing %q in %s", marker, output.String())
		}
	}
	if strings.Contains(output.String(), "retrying inspect_") || strings.Contains(output.String(), "healthy inspect_") {
		t.Fatal("invented action")
	}
}
