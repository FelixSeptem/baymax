package offline

import (
	"context"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
)

func TestOfflineModelAdapterRunAndStreamSemanticEquivalent(t *testing.T) {
	eng := runner.New(OfflineModelAdapter{})
	runRes, err := eng.Run(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	streamRes, err := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if normalize(runRes.FinalAnswer) != normalize(streamRes.FinalAnswer) {
		t.Fatalf("semantic mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
	}
}

func normalize(in string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(in))), " ")
}
