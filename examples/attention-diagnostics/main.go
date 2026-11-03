package main

import (
	"fmt"
	"io"
	"os"

	"github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func demonstrate(out io.Writer) error {
	view, err := diagnostics.ProjectAttention(diagnostics.AttentionRequest{
		Runs: []diagnostics.RunRecord{
			{RunID: "healthy", Status: "completed"},
			{RunID: "retrying", TerminalRetryable: true},
			{RunID: "gate", GateDeniedCount: 1},
			{RunID: "unsupported", ProviderCapabilityOutcome: "blocked"},
		},
		Evidence: []diagnostics.AttentionEvidence{
			{RunID: "effect", Code: "effect-pending", Reference: "effect/ref"},
			{RunID: "source", Code: "source-required", Reference: "source/ref"},
		},
		Limit: 2,
	})
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "total=%d shown=%d truncated=%t\n", view.Total, len(view.Items), view.Truncated); err != nil {
		return err
	}
	for _, item := range view.Items {
		if _, err := fmt.Fprintf(out, "%s %s %s %s %s\n", item.RunID, item.Code, item.NextAction, item.Owner, item.Reference); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(out, "healthy_retrying_ignored\neffect_pending_explicit\ntruncation_visible")
	return err
}

func main() {
	if err := demonstrate(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
